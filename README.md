# 回声（Echo）后端

解决"选择困难"的小 web 应用后端：提问者发布一道题和几个选项，回答者随机抽题作答、评论区讨论，到期由后端自动结算占比；票数焦灼的题进入争议海螺区。

技术栈：Go 1.27 + Gin + GORM + PostgreSQL，鉴权用 JWT（HS256，24 小时），口令用 bcrypt，日志用 zap，表结构用 golang-migrate 管理。

## 先选一条路

| | A 路：本机直接跑 | B 路：用容器跑 |
|---|---|---|
| 要装什么 | Go、PostgreSQL、migrate 命令行 | 只要 Docker Desktop |
| 要不要手工建库、灌数据 | 要（三条命令） | 不用，容器自己做 |
| 适合谁 | 想改代码、看日志、调试 | 想先把它跑起来看看 |

---

## A 路：本机直接跑

### A1. 装三样东西

| 装什么 | 怎么装 | 装好后敲这句确认 |
|---|---|---|
| Go 1.27+ | 官网下载装包，Windows 一路下一步 | `go version` 出现 `go version go1.27.x windows/amd64` |
| PostgreSQL | 官方安装包，装的过程中会让你给超级用户 `postgres` 设一个密码（**记住它**，A3 要用） | `psql --version`；迁移用了 `GENERATED ALWAYS AS IDENTITY`，需要 PG 10 以上 |
| migrate 命令行 | `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest` | `migrate -version`（有版本输出即可，显示 `dev` 也正常） |

Windows 上建议用 **Git Bash** 敲下面所有命令（`export`、`curl` 这些在 PowerShell 里写法不同）。

### A2. 配 .env

```bash
cd backend
cp .env.example .env
```

打开 `.env`，至少改这三项：

| 键 | 怎么填 |
|---|---|
| `DB_USER` | 应用账号名，默认 `echo_app`（A3 会教你建它） |
| `DB_PASSWORD` | A3 里给 `echo_app` 设的那个密码，两处必须一致 |
| `JWT_SECRET` | 随便一串长字符，例如 `openssl rand -hex 32` 的输出。留空同样拒绝启动 |

`.env` 已经被 `.gitignore` 挡住，不会进仓库，也不进镜像。

### A3. 建应用账号和库（只做一次，需要超级用户）

装完 PostgreSQL 只有 `postgres` 这一个超级用户，**`echo_app` 不会自动存在**，得先建账号再建库。两条命令都要输 `postgres` 的密码：

```bash
psql -h 127.0.0.1 -p 5432 -U postgres -c "CREATE ROLE echo_app LOGIN PASSWORD '你自己定的密码';"
psql -h 127.0.0.1 -p 5432 -U postgres -c 'CREATE DATABASE echodb OWNER echo_app;'
```

成功时一行是 `CREATE ROLE`，一行是 `CREATE DATABASE`。第一条里设的那个密码，就是 A2 中 `DB_PASSWORD` 要填的值——两处必须一致，否则后面会一直报 `password authentication failed`。

> - 已经建过就会报 `role "echo_app" already exists` 或 `database "echodb" already exists`，看到这两条跳过即可，不会破坏数据。想改密码用 `ALTER ROLE echo_app LOGIN PASSWORD '新密码';`。
> - **这两条必须用 `-U postgres` 跑**（应用账号没有建角色和建库的权限，用它会报 `permission denied to create role` / `permission denied to create database`，那是权限不够，不是数据库坏了）。

### A4. 建表（跑迁移）

回到**仓库根目录**执行：

```bash
export PGPASSWORD='你在 A2 填的 DB_PASSWORD'
migrate -path backend/migrations \
  -database "postgres://echo_app@127.0.0.1:5432/echodb?sslmode=disable" up
```

没有报错就是成功了——`migrate` 干活时不输出任何东西，如果已经是最新版本会打印一行 `no change`，那也是正常的。确认一下：

```bash
migrate -path backend/migrations \
  -database "postgres://echo_app@127.0.0.1:5432/echodb?sslmode=disable" version
```

应该打印 `6`。

常用变体：`down 1` 回滚一条，`down -all` 回滚全部，`goto 3` 跳到某个版本。

> **已经执行过的迁移文件不要改**，要变更结构就新增一对 `000007_xxx.up.sql` / `.down.sql`。

### A5. 灌演示数据

```bash
psql -h 127.0.0.1 -p 5432 -U echo_app -d echodb -f backend/seed/seed.dev.sql
```

会刷一屏 `INSERT 0 1` 之类，最后能看到 `UPDATE 4`。这份数据里有 5 道题、11 张票、2 条评论和 4 个能登录的账号。

> seed 不在迁移链里，`migrate down` 不会清它；想回到干净状态，重跑这一句就行（它会先把题目、投票、评论清空再灌）。

### A6. 启动

```bash
cd backend && go run .
```

看到一屏 `[GIN-debug] ...` 的路由表就说明起来了，最后停在 `Listening and serving HTTP on :8080`。**这个窗口要一直开着**，按 `Ctrl+C` 是正常停止。

嫌 debug 输出多就在启动前加一句 `export GIN_MODE=release`。

### A7. 验一下通了（三条命令）

再开一个终端窗口：

```bash
curl http://127.0.0.1:8080/api/ping
```

看到 `{"code":200,"msg":"success","data":"pong 后端基建运行正常"}` 就是通了。

```bash
curl -X POST http://127.0.0.1:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account":"alice01","password":"Echo12345678"}'
```

看到 `"code":200` 并且 `data.token` 有一长串字符，说明登录和数据库都通了。

带着这个 token 再取一次自己的资料（把 `粘贴你的token` 换成上面那串）：

```bash
curl http://127.0.0.1:8080/api/users/me -H "Authorization: Bearer 粘贴你的token"
```

看到 `nickname`、`exp`、`level`、`level_progress` 这些字段就全对了。

---

## B 路：用容器跑

### B1. 装 Docker Desktop，并配镜像加速（这步别跳）

装好 Docker Desktop 并启动（左下角显示 **Engine running** 才算起来）。

然后 **Settings → Docker Engine**，把编辑框里的内容换成下面这段，点 Apply & restart：

```json
{
  "builder": {
    "gc": {
      "defaultKeepStorage": "20GB",
      "enabled": true
    }
  },
  "experimental": false,
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://docker.xuanyuan.me"
  ]
}
```

> 为什么要这一步：构建镜像要先从 Docker Hub 下载基础镜像，国内网络经常连不上，不配加速会在第一行 `FROM` 就失败。上面两个地址是实测可达的，第一个不行会自动试第二个。

### B2. 一条命令起

在**仓库根目录**执行：

```bash
docker compose --env-file backend/.env up --build
```

第一次会编译并打包镜像（一两分钟）。它自己就把库建好、表建好、演示数据灌好，**不需要 A3、A4、A5**。

### B3. 看到什么算成功

```bash
docker compose --env-file backend/.env ps
```

三行状态应该是：

```
26hack4-echobe-app-1    running   Up ...
26hack4-echobe-db-1     running   Up ... (healthy)
26hack4-echobe-dbinit-1 exited    Exited (0) ...
```

`dbinit` 是干完活就退出的，**退出码 0 是正常状态**，不是出错。

### B4. 验一下通了

和 A7 那三条命令一模一样，直接照着敲即可（端口同样是 8080）。

### B5. 停止、重置、改了代码之后

```bash
docker compose --env-file backend/.env down                  # 停，数据留着，下次起是秒开
docker compose --env-file backend/.env down -v               # 停，并把数据库整个清空
docker compose --env-file backend/.env up -d --build         # 改了代码后重新构建并起
docker compose --env-file backend/.env logs -f app           # 看后端日志（Ctrl+C 退出跟踪）
```

停止请用 `down` 或 `stop`（发"请你收工"的信号，会等在途请求做完），不要用 `kill`（直接拔电）。

### B6. 想连容器里的数据库看看

```bash
docker compose --env-file backend/.env exec db psql -U echo_app -d echodb
```

进去之后 `\dt` 看表、`\d questions` 看字段、`SELECT count(*) FROM votes;` 数行数，`\q` 退出。

> 容器里的库对外映射在 **5433**（不是 5432），因为你本机那个 PostgreSQL 已经占了 5432，两个库互不干扰。用图形化工具连的时候填：主机 `127.0.0.1`、端口 `5433`、库 `echodb`、用户 `echo_app`。

---

## 测试账号

`seed.dev.sql` 里四个账号共用同一个演示口令，可以直接登录（是真实 bcrypt 哈希，不是占位值）：

| 账号 | 昵称 | 密码 | seed 刚灌完的 exp |
|---|---|---|---|
| `alice01` | Alice | `Echo12345678` | 85 |
| `bob01` | Bob | `Echo12345678` | 85 |
| `carol01` | Carol | `Echo12345678` | 20 |
| `dave01` | Dave | `Echo12345678` | 0 |

等级 = `exp / 50` 向下取整，0 级起步。发题 +10、答一题 +5、到期结算给提问者 +40、答对 +25、答错 +5、平票没定成时补偿 +15，**所以这几个数字会随你操作和后台结算一直变大，不是固定的**。

这套 seed 的口令是公开在仓库里的演示值，**不要灌到任何公网可达的库**。

## 环境变量

| 键 | 生效吗 | 说明 |
|---|---|---|
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` `DB_SSLMODE` | 生效 | 数据库连接 |
| `JWT_SECRET` | 生效 | 签名密钥，留空启动即失败 |
| `PORT` | 生效 | 监听端口，留空按 8080 |
| `JWT_EXPIRE_HOURS` | 生效 | token 有效期小时数，留空或填非正整数都回落 24。改它等于改契约，要先通知前端 |
| `CORS_ALLOW_ORIGIN` | 生效 | 逗号分隔的前端地址白名单。留空或写 `*` 都表示放开，本机开发不用配；生产要显式配 |
| `GIN_MODE` | 生效 | 设 `release` 关掉调试输出，代码不用改 |

开发环境和生产环境的区别只体现在注入的变量值上：本机用 `.env`，容器和服务器用环境变量注入，**代码只认一套读取路径**，没有 `.env.dev` / `.env.prod` 两套文件。`DB_SSLMODE` 本机用 `disable`，连远端库应改成 `require`。

## 跑不起来怎么查

| 看到的报错 | 是什么意思 | 怎么办 |
|---|---|---|
| `缺少 JWT_SECRET，请检查 .env` | 密钥没填，服务故意不起 | 回 A2 填上 |
| `DB_PASSWORD 在 .env 里是空的` 或 `缺少 DB_HOST 或 DB_NAME` | `.env` 没配或没改名 | 确认 `backend/.env` 存在且填了值 |
| `数据库初始化失败 ... connection refused` | PostgreSQL 没在跑，或端口写错 | 启动数据库服务；确认 `DB_PORT=5432` |
| `permission denied to create role` 或 `... to create database` | A3 那两条用应用账号跑了，权限不够 | 改成 `-U postgres` 重跑 A3 |
| `password authentication failed for user "你的系统用户名"` | `migrate` 的连接串里没写用户名，退回用了系统账号 | 连接串写全 `postgres://echo_app@127.0.0.1:5432/echodb?...` |
| `unrecognized configuration parameter "password_env"` | 这个驱动版本不认 `password_env` 参数 | 用 `export PGPASSWORD=...`，别写进连接串 |
| `Dirty database version 6. Fix and force version.` | 某次迁移跑到一半失败，被标记为脏 | 先确认那条迁移做了一半还是完全没做，再 `migrate ... force 6` |
| `relation "questions" does not exist` | 还没跑迁移 | 执行 A4 |
| `listen tcp :8080: bind: Only one usage...` | 8080 被别的进程占了（常见是另一个后端实例或容器还在跑） | 关掉另一个，或改 `.env` 里的 `PORT` |
| 登录返回 401 `账号或密码错误` | 没灌 seed，或者用的是旧版 seed 的占位密码 | 执行 A5 重灌，再用 `alice01 / Echo12345678` |
| `curl` 完全没有响应 | 服务没起或已经退出 | 看启动那个窗口的报错；容器方式跑 `docker compose logs app` |
| Docker：`FROM golang:1.27-alpine` 卡住或超时 | 拉不到基础镜像 | 按 B1 配 `registry-mirrors` |
| Docker：`go mod download` 报 `connection refused` | 构建容器里连不上 Go 代理 | `docker compose build --build-arg GOPROXY=https://proxy.golang.org,direct`（国内默认值已经能用了） |
| Docker：db 容器起不来，日志里有 `unused mount/volume` | 数据卷挂在了旧路径 | 保持 compose 里 `dbdata:/var/lib/postgresql`（Postgres 18 的新规矩，别改回 `.../data`） |

想确认后端有没有出错，看日志里的 `level` 字段：`error` 才是服务端故障，`warn` 多半是请求参数不对，`info` 是正常访问记录。
