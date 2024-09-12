# 构建阶段
FROM golang:1.21 AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum 文件（如果存在）
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o random-api .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/random-api .
COPY --from=builder /app/public ./public

# 创建日志目录并设置权限
RUN mkdir -p /var/log/random-api && chmod 755 /var/log/random-api

EXPOSE 5003

# 使用 tini 作为初始化系统
RUN apk add --no-cache tini
ENTRYPOINT ["/sbin/tini", "--"]

# 运行应用
CMD ["./random-api"]
