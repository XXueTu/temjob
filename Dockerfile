# 多阶段构建 Dockerfile
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o temjob main.go

# 最终镜像
FROM alpine:latest

# 安装必要的包
RUN apk --no-cache add ca-certificates curl tzdata && \
    addgroup -g 1000 -S temjob && \
    adduser -u 1000 -S temjob -G temjob

# 设置时区
ENV TZ=Asia/Shanghai

# 创建必要的目录
RUN mkdir -p /app/logs && \
    chown -R temjob:temjob /app

# 切换到非root用户
USER temjob

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/temjob .

# 复制配置文件模板
COPY --from=builder /app/config.yaml ./config.yaml.template

# 复制 Web 资源
COPY --from=builder /app/web/templates ./web/templates
COPY --from=builder /app/web/static ./web/static

# 暴露端口
EXPOSE 8088

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8088/api/v1/stats || exit 1

# 启动命令
CMD ["./temjob", "--config", "config.yaml"]