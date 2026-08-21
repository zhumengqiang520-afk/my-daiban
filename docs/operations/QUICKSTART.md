# 生产部署快速指南

本指南对应 `deploy/retail` 中的单机生产试点方案：Caddy 自动 HTTPS、Vikunja 零售扩展、PostgreSQL 16 和本地持久化附件。正式大规模上线可将 PostgreSQL 和附件分别替换为托管数据库与 S3 对象存储。

## 1. 上线前准备

- 一台 Linux 主机，试点建议 4 vCPU、8 GB 内存、100 GB SSD。
- 已安装 Docker Engine 和 Docker Compose v2。
- 域名 A/AAAA 记录已指向主机，80/443 TCP 以及 443 UDP 已放行。
- 中国大陆公网使用时已完成所需备案。
- 企业 GitHub fork 可被系统用户访问，用于履行 AGPL 源码提供义务。

## 2. 首次部署

在仓库根目录执行：

```sh
cd deploy/retail
cp .env.example .env
openssl rand -hex 32
```

将生成值填入 `.env` 的 `SERVICE_SECRET`，同时替换域名、源码地址和数据库密码。`APP_UID` / `APP_GID` 应与部署账号的 `id -u` / `id -g` 一致。

```sh
./scripts/prepare.sh
docker compose build
docker compose up -d
docker compose ps
./scripts/smoke-test.sh
```

首次构建会下载 Go、Node 和前端依赖，所需时间取决于网络。服务启动时会自动执行向前数据库迁移。

## 3. 创建首个管理员

不要把密码写进命令历史；让 CLI 交互读取。

```sh
docker compose run --rm app user create --username admin --email admin@example.com
docker compose run --rm app user set-admin admin --admin
```

登录 `https://<DOMAIN>/`。左侧菜单出现“零售任务”，说明零售模块已启用。

## 4. 初始化业务数据

按以下顺序操作，可避免组织或人员无法被派任：

1. 在“零售任务 → 组织与人员”先创建公司，再建区域，最后建门店；仓库可归公司或区域。
2. 用管理员 CLI 或临时开放注册创建员工账号，再按用户名将其加入门店。
3. 在 Vikunja “项目”中为每家门店建立独立项目，从项目 URL 记下数字 ID。一个项目只能绑定一个零售组织。
4. 创建 5—8 个高频模板，例如开店检查、闭店检查、样床与价签、库存抽盘和客户回访。
5. 从单家试点店手动派发一次，完整走通“开始—清单—凭证—提交—复核—完成”，再开启自动计划。

## 5. 发布更新

```sh
./scripts/backup.sh
git fetch --tags
git checkout <approved-version-tag>
docker compose build --pull
docker compose up -d
./scripts/smoke-test.sh
```

如应用版本需要回退，切换上一个已验收标签后重建。如新迁移与旧应用不兼容，先停止写入，再使用已演练的备份恢复：

```sh
./scripts/restore.sh --confirm /absolute/path/to/retail-backup.zip
```

## 6. 日常操作

```sh
docker compose ps
docker compose logs --since=30m app
./scripts/backup.sh
./scripts/smoke-test.sh
```

建议每日由调度器执行 `backup.sh`，并把 `backups/` 同步到不同主机或对象存储。只有备份文件还不够，每季度必须在隔离环境执行恢复演练。

## 7. 上线后验证

- `/api/v2/health` 返回 `status=OK`。
- `/api/v1/info` 返回 `retail_enabled=true`。
- `/source-code` 跳转到本企业 fork。
- 店员无法读取其他门店的人员、任务、凭证和指标。
- 手机端可完成任务，店长可复核，负荷和运营总览同步更新。

