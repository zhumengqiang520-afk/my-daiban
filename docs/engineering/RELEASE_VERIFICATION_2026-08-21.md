# MVP 发布验证报告（2026-08-21）

验证分支：`feat/retail-foundation`

验证基线：Vikunja `v2.5.0`，零售扩展提交至 `04a8803`。

## 通过项

| 范围 | 命令/方式 | 结果 |
|---|---|---|
| 后端核心回归 | `go test ./pkg/models ./pkg/routes/api/v2 ./pkg/routes/api/shared ./pkg/webtests` | 通过 |
| Go 定向静态检查 | `golangci-lint run ./pkg/models ./pkg/routes/api/v2 ./pkg/routes/api/shared` | 0 issues |
| 发布二进制 | `mage build:build` | 通过，生成 `vikunja` |
| 前端全量单测 | `vitest run --dir ./src` | 54 个文件、1096 项全部通过 |
| 零售接口单测 | `retail.test.ts` | 4 项通过，包括分页转换、顶层数组、派发和人员 PATCH |
| 新增前端静态检查 | 零售页面/服务/导航/路由 ESLint，零售页面 Stylelint | 通过 |
| 前端生产构建 | `pnpm build` | 通过，包含 PWA service worker |
| Compose 配置 | `docker compose ... config --quiet` | 通过 |
| 部署脚本 | `sh -n deploy/retail/scripts/*.sh` | 通过 |
| 首次准备脚本 | 在临时目录使用非默认假密钥运行 `prepare.sh` | 通过 |
| 差异质量 | `git diff --check` | 通过 |

## 自动化覆盖要点

- 组织上级继承、同店允许、跨店拒绝、临时借调到期。
- 任务组织隔离、责任人/复核人资格、任务状态不可绕过。
- 必填清单、附件归属、多次提交、驳回原因、重提和完成同步。
- 模板版本不可变、派发预览、幂等生成、清单快照。
- 日/周/月自动派发，包括短月月末夹取和恢复锚点。
- 日容量、已分配分钟数、超载阈值和 Asia/Shanghai 营业日。
- 责任人/店长/上级管理员逾期升级及投递幂等。
- 完成率、按时率、驳回率、状态/类别分布和逾期队列。

## 已知基线问题

- 仓库全量 `pnpm typecheck` 仍会因为 Vikunja 基线中大量已存的 Vue/TypeScript 类型错误退出 2。本次新增零售页面、服务和配置在报告中没有类型错误，且生产构建通过。这些基线错误应单独建立技术债 Issue，不应在零售功能 PR 中大范围修改上游页面。
- 仓库全量 `golangci-lint` 存在一个与本次无关的历史 G115：`pkg/webtests/huma_avatar_upload_test.go:48`。本次后端范围的定向 lint 为 0。
- `go build ./...` 会扫描到没有 `main` 入口的示例插件；本仓库的正式构建入口 `mage build:build` 已通过。
- Sass `@import` 和 PostCSS 插件在测试/构建时有上游弃用警告，未影响产物生成。

## 必须在真实部署环境执行的门禁

当前验证机的 Docker daemon 不可用，且没有企业域名、云主机、生产密钥或真实门店账号。因此以下项目不能由本地代码验证代替，且未通过前不应宣布正式生产上线：

1. 构建并启动 Compose 容器，验证 PostgreSQL 全量迁移和 HTTPS 证书。
2. 运行 `smoke-test.sh`，以及一次备份—隔离恢复—业务核对。
3. 在预发布按 [UAT_AND_GO_LIVE.md](../operations/UAT_AND_GO_LIVE.md) 完成真实角色、手机浏览器和跨门店越权验收。
4. 记录正式镜像摘要、数据库备份、告警接收人和业务/技术/运维签字。

