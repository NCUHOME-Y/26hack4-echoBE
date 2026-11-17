# 回声后端镜像：多阶段构建，运行层只有一个二进制
# 构建上下文是 backend/ 目录：docker build -t echo-be ./backend
# 数据库迁移不在这个镜像里跑，由 docker-compose.yml 把 backend/migrations 挂给 postgres 自己执行

# ---------- 构建层 ----------
FROM golang:1.27-alpine AS build

WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux

# 构建期下载 Go 模块用的代理。默认值是本机实测可达的地址；
# 到 proxy.golang.org 不通的机器（比如这里）不换它就会在 `go mod download` 上 connection refused。
# 海外构建可以覆盖：docker build --build-arg GOPROXY=https://proxy.golang.org,direct
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

# 依赖单独一层：go.mod / go.sum 不变时，改代码不会重新下载整个依赖树
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/echo .

# ---------- 运行层 ----------
FROM alpine:3.20

# tzdata：zap 打的日志时间戳按本地时区显示，容器默认是 UTC，会让日志时间比北京时间少 8 小时。
# 今日统计不受它影响——日界在 Go 里用 time.FixedZone("+08:00") 算死，不读系统时区。
RUN apk add --no-cache tzdata ca-certificates \
    && addgroup -S echo \
    # 不用 root 跑：容器被攻破时，进程写不了文件系统、也读不到 /etc/shadow
    && adduser -S -G echo -h /nonexistent echo

ENV TZ=Asia/Shanghai \
    GIN_MODE=release \
    PORT=8080

WORKDIR /app
COPY --from=build /out/echo /app/echo

USER echo
EXPOSE 8080

# CMD 留空：端口等配置全部走环境变量，不在镜像里写死启动参数
ENTRYPOINT ["/app/echo"]
