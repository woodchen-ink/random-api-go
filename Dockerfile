# 构建阶段
FROM golang:1.23 AS builder

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

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/random-api .
COPY --from=builder /app/public ./public
# 复制 public 目录到一个临时位置
COPY --from=builder /app/public /tmp/public

# 创建必要的目录
RUN mkdir -p /root/data/logs /root/data/public

EXPOSE 5003

# 使用 tini 作为初始化系统
RUN apk add --no-cache tini
ENTRYPOINT ["/sbin/tini", "--"]

# 创建一个启动脚本
COPY start.sh /start.sh
RUN chmod +x /start.sh

CMD ["/start.sh"]
