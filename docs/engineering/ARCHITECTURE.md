# 技术架构与二开设计

## 1. 总体方案

以 Vikunja 稳定发行版为单体应用核心，保持其 Go API、Vue/TypeScript Web、任务模型、身份认证、附件和提醒能力。零售能力以独立领域包、独立前端模块和新增数据表实现。

```text
手机/电脑浏览器
       |
    HTTPS
       |
 Caddy / Nginx
       |
 Vikunja + Retail Extension
   |        |          |
PostgreSQL  对象存储   通知适配器
             |        |-- 站内
             |        |-- 邮件
             |        `-- 企业微信（增强版）
             `-- 任务照片/附件
```

首版保持模块化单体，不引入微服务。只有当通知、报表或文件处理实际形成独立扩容需求时才拆分。

## 2. 上游复用与新增边界

### 复用上游

- 用户、登录和基础团队共享。
- 项目、任务、负责人、标签、任务关系。
- 到期时间、提醒、重复能力。
- 评论、附件、通知和 API 基础设施。
- 列表、看板、日历等通用视图。

### 零售扩展

- `retail_org_units`：公司、区域、门店、仓库；每个单元绑定一个上游 Vikunja Team。
- 上游 `team_members`：作为组织成员与管理员权限的单一真实来源，区域/公司权限向下继承。
- `retail_memberships`：补充岗位、直属负责人、主归属、临时借调及有效期；管理权实时同步到 `team_members.admin`，不重复保存基础团队权限。
- `retail_staff_capacity`：每日可分配容量和变更原因。
- `retail_task_profiles`：任务对应的门店、类别、主负责人、复核人、预计工时、来源和策略。
- `retail_task_templates` / `retail_template_versions`：模板及不可变版本。
- `retail_template_dispatches`：模板版本、目标组织、计划时间与幂等键对应的生成记录。
- `retail_template_schedules`：日/周/月重复计划、下次执行时间、月末锚点及停用状态。
- `retail_checklist_items`：验收清单及完成状态。
- `retail_submissions` / `retail_submission_files`：每次完成提交及凭证。
- `retail_reviews`：通过、驳回和意见。
- `retail_task_transitions`：独立于通用审计的零售任务业务状态流转历史。
- `retail_notification_deliveries`：责任人、店长和上级管理员的发送幂等记录；MVP 使用 0/30/120 分钟默认升级阈值。
- 任务提交和复核历史保存在对应业务表；通用审计复用获得许可的 Vikunja 审计能力。

表名仅为设计名，实施前根据上游迁移和命名规范调整。

## 3. 关键约束

- 所有零售任务必须对应一个有效组织单元。
- 主负责人必须在任务组织单元具有有效成员关系或有效借调关系。
- 任务状态不能仅依赖前端显示；服务端执行状态机校验。
- 提交、复核和驳回在数据库事务内写入状态与审计事件。
- 重复任务使用“模板版本 + 计划发生时间”唯一键确保幂等。
- 通知投递使用“事件 + 渠道 + 接收人”唯一键防止重复。
- 所有查询先约束授权组织范围，再应用用户筛选条件。

## 4. 核心状态机

```text
draft -> assigned -> in_progress -> pending_review -> completed
                          ^              |
                          |              v
                          `---------- rejected

draft / assigned / in_progress -> cancelled
```

状态转换由领域服务统一执行，禁止控制器或前端直接修改状态字段。

## 5. API 草案

沿用上游 `/api/v2`，新增零售命名空间：

| 方法与路径 | 用途 |
|---|---|
| `GET /api/v2/retail/org-units` | 获取有权限的组织树 |
| `POST /api/v2/retail/org-units` | 创建组织单元 |
| `GET/POST /api/v2/retail/memberships` | 查询或新增人员归属 |
| `GET/PUT/PATCH/DELETE /api/v2/retail/memberships/{id}` | 管理岗位、主管、借调和作用域权限 |
| `GET/POST /api/v2/retail/task-profiles` | 查询或绑定零售任务属性 |
| `GET/PUT/PATCH/DELETE /api/v2/retail/task-profiles/{id}` | 管理任务组织、类别、人员、工时和凭证策略 |
| `GET /api/v2/retail/staff/workload` | 查询日期范围内的人员负荷 |
| `PUT /api/v2/retail/staff/{id}/capacity` | 调整人员容量 |
| `POST /api/v2/retail/templates` | 创建任务模板 |
| `POST /api/v2/retail/templates/{id}/dispatch-preview` | 预览批量派发 |
| `POST /api/v2/retail/templates/{id}/dispatch` | 幂等派发 |
| `GET /api/v2/retail/staff/workload` | 按组织和日期查询人员容量与已分配分钟数 |
| `PUT /api/v2/retail/staff/{id}/capacity` | 设置个人单日容量覆盖 |
| `GET/POST /api/v2/retail/template-schedules` | 查询或新建自动派发计划 |
| `GET/PUT/PATCH/DELETE /api/v2/retail/template-schedules/{id}` | 管理日/周/月派发计划 |
| `POST /api/v2/retail/tasks/{id}/start` | 开始任务 |
| `GET /api/v2/retail/tasks/{id}/workflow` | 读取清单、提交、复核及流转历史 |
| `PUT /api/v2/retail/checklist-items/{id}/completion` | 勾选或取消清单项 |
| `POST /api/v2/retail/tasks/{id}/submissions` | 提交完成凭证 |
| `POST /api/v2/retail/tasks/{id}/reviews` | 通过或驳回 |
| `POST /api/v2/retail/tasks/{id}/cancel` | 由组织管理员取消任务 |
| `GET /api/v2/retail/dashboard/operations` | 运营统计 |
| `POST /api/v2/retail/exports` | 创建异步导出 |

API 必须输出稳定错误码，不允许前端依赖中文错误消息判断逻辑。

已实现组织、成员、任务扩展属性、清单定义和上述状态流转接口；当 `retail.enabled=false` 时零售路由不注册。通用 `PUT/PATCH` 无法直接篡改业务状态。

## 6. 权限实现

采用 RBAC + 组织作用域：

- 权限描述“能做什么”，例如 `task.review`。
- 作用域描述“能对哪里做”，例如门店 A 或区域华东。
- 数据对象携带组织单元，查询时计算用户有效作用域。
- 系统管理员和业务数据查看权限分离。

每个接口至少测试：无登录、无权限、同店权限、跨店拒绝、上级区域权限、账号停用六类情况。

## 7. 文件与图片

- 开发环境可以使用本地卷；测试和生产优先使用兼容 S3 的对象存储。
- 数据库只保存对象键、哈希、MIME、大小、上传者和关联提交，不保存公开 URL。
- 下载使用短期签名 URL 或受鉴权代理。
- 限制单文件和单任务总大小；图片生成缩略图。
- 删除任务时默认软删除引用，按照保留策略异步清理对象。

## 8. 报表策略

首版从事务库按受控日期范围聚合，并为常用查询建立复合索引。数据达到以下任一阈值后再引入汇总表或独立分析库：

- 单表任务超过 300 万。
- 常用运营看板 P95 连续一周超过 2 秒。
- 跨年报表影响在线事务。

指标计算函数应有固定样例测试，防止页面与导出使用不同口径。

## 9. 配置与密钥

- 配置按开发、测试、预发布、生产分离。
- `.env.example` 只包含键名和安全示例，不包含真实凭证。
- 生产密码、邮件密钥、对象存储密钥和企业微信密钥由云密钥服务或受控部署环境注入。
- 日志不得打印密码、令牌、完整手机号、文件签名 URL。

## 10. 上游同步

1. `upstream` 指向官方 Vikunja 仓库，`origin` 指向企业 fork。
2. 每月或高危安全公告后创建 `chore/upstream-sync-YYYY-MM`。
3. 先在同步分支合并上游稳定标签，解决冲突并更新变更记录。
4. 执行数据库迁移演练、单元测试、端到端测试和备份恢复冒烟。
5. 在预发布至少运行两个工作日，再进入生产发布。
