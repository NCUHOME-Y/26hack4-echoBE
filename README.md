# 回声（Echo）后端

解决"选择困难"的小 web 应用后端：提问者发布一道题和几个选项，回答者随机抽题作答、评论区讨论，到期由后端自动结算占比；票数焦灼的题进入争议海螺区。

## 技术栈

| 项 | 值 |
|---|---|
| 语言 / 运行时 | Go 1.27（模块名 `echo`） |
| Web 框架 | Gin v1.12 |
| ORM | GORM v1.31（`TranslateError` 开启） |
| 数据库 | PostgreSQL（开发实例 18.6） |
| 鉴权 | JWT HS256，有效期 24 小时 |
| 口令哈希 | bcrypt（`golang.org/x/crypto`） |
| 日志 | zap（结构化）＋ 标准库 `log` |
| 迁移工具 | golang-migrate（`migrate` CLI） |

## 快速开始

### 0. 需要的工具

- Go 1.27+、PostgreSQL（迁移里用了 `GENERATED ALWAYS AS IDENTITY`，需要 PG 10 以上；开发实例是 18.6）
- golang-migrate 命令行：`go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
  确认：`migrate -version`

### 1. 配置

```bash
cd backend
cp .env.example .env      # .env 已被 .gitignore 挡住，不会进仓库
```

至少改这三项：`DB_USER`、`DB_PASSWORD`、`JWT_SECRET`（留空则服务直接拒绝启动）。各项含义见下面"环境变量"。

### 2. 建库（需要超级用户，只需做一次）

应用账号 `echo_app` 默认没有建库权限（`rolcreatedb = false`），所以这一步要用超级用户：

```bash
psql -h 127.0.0.1 -p 5432 -U postgres -c 'CREATE DATABASE echodb OWNER echo_app;'
```

数据库已经存在就跳过。

### 3. 迁移：用 migrate 自带的命令

在仓库根目录执行：

```bash
export PGPASSWORD='你的 DB_PASSWORD'
migrate -path backend/migrations \
  -database "postgres://echo_app@127.0.0.1:5432/echodb?sslmode=disable" up
```

- 查当前版本：把结尾的 `up` 换成 `version`，正常输出 `6`
- 回滚一条：`down 1`；回滚全部：`down -all`
- 某条迁移中途失败会把 `schema_migrations.dirty` 置为 `true`，此后 `up` 会拒绝执行。先确认那个迁移是做了一半还是完全没做，再强制对齐版本号：`force 6`

迁移文件在 `backend/migrations/`，命名 `0000NN_名称.up.sql` / `.down.sql`，`NN` 是 migrate 记录的版本号。**已经执行过的迁移不要改，要变更就新增一对。**

三个会踩的坑（实测过）：

| 现象 | 原因 | 做法 |
|---|---|---|
| `password authentication failed for user "你的系统用户名"` | migrate 不读 `DB_USER`，用户名不在 URL 里就退回操作系统登录名 | URL 里写全 `postgres://echo_app@host:port/db` |
| `unrecognized configuration parameter "password_env"` | 当前驱动版本不支持 `password_env` 这个 URL 参数 | 用 `export PGPASSWORD=...` |
| 口令写进 URL 后能用 | 但它会出现在 `ps` 和 shell history 里 | URL 不带口令，口令只走环境变量 |

### 4. 灌开发数据

seed 在迁移链之外，`migrate down` 不会清它，所以任何时候想让库回到基线，就重跑这一句：

```bash
psql -h 127.0.0.1 -p 5432 -U echo_app -d echodb -f backend/seed/seed.dev.sql
```

### 5. 启动

```bash
cd backend && go run .
```

监听 `:8080`。探活：

```bash
curl http://127.0.0.1:8080/api/ping
```

## 测试账号

`seed.dev.sql` 里四个账号共用同一个演示口令，可直接登录（bcrypt 真哈希，不是占位值）：

| 账号 | 昵称 | 密码 | exp | 等级 |
|---|---|---|---|---|
| `alice01` | Alice | `Echo12345678` | 85 | 1 |
| `bob01` | Bob | `Echo12345678` | 85 | 1 |
| `carol01` | Carol | `Echo12345678` | 20 | 0 |
| `dave01` | Dave | `Echo12345678` | 0 | 0 |

等级 = `exp / 50` 向下取整，0 级起步。**这套 seed 的口令是公开在仓库里的演示值，不要灌到任何公网可达的库。**

## 环境变量

| 键 | 现在生效吗 | 说明 |
|---|---|---|
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` `DB_SSLMODE` | 生效 | 数据库连接 |
| `JWT_SECRET` | 生效 | HS256 签名密钥，留空启动即失败 |
| `GIN_MODE` | 生效（gin 自己读，代码无需配合） | 生产设 `release`，关掉 debug 输出和路由表打印 |
| `PORT` | 生效 | 监听端口，留空按 8080 |
| `JWT_EXPIRE_HOURS` | 生效 | token 有效期小时数，留空或填非正整数都回落 24。改这个值等于改契约，要先通知前端 |
| `CORS_ALLOW_ORIGIN` | 生效 | 逗号分隔白名单：命中才把对方的源原样回写并加 `Vary: Origin`，名单外不发 CORS 头。留空或写成 `*` 都表示放开（本机开发不用配），生产应显式配前端地址 |


开发环境和生产环境的区别只体现在注入的变量值上：本机用 `.env`，容器和服务器用环境变量注入，**代码只认一套读取路径**。
`DB_SSLMODE` 本机开发用 `disable`，生产走容器或远端库应改成 `require`，口令由部署环境注入。

## 后台任务

启动后每 30 秒扫一轮到期问题：写入 `settled_at`、按票数最高者填 `final_option_id`（**平票或无人投票则留空，由提问者用公布接口补定**），同时做争议判定——总票数 > 30 且最高两个选项占比差 ≤ 15 个百分点即打 `contested_at`。`status` 只有 `open` / `closed` 两种值，是否争议看 `contested_at` 是否为空。

## 目录结构

```
backend/
  main.go          依赖装配与启动
  router/          路由与全局中间件（CORS、访问日志、panic 兜底、404/405 统一返回）
  middleware/      JWT 鉴权、跨域、请求日志
  handler/         各接口的入参校验与响应
  service/         业务规则（投票占比、结算、争议判定、今日统计、软删）
  repository/      GORM 查询与事务
  model/           实体、请求结构、哨兵错误、经验值常量
  migrations/      golang-migrate 的 up/down 对
  seed/            seed.dev.sql 开发数据
```
