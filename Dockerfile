# 前端构建阶段
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# 复制前端依赖文件
COPY web/package*.json ./

# 安装前端依赖
RUN npm ci --only=production

# 复制前端源代码
COPY web/ ./

# 构建前端静态文件
RUN npm run build

# 后端构建阶段
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制后端源代码
COPY . .

# 构建后端应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o random-api .

# 运行阶段
FROM alpine:latest

# 安装必要的工具
RUN apk --no-cache add ca-certificates tzdata tini

WORKDIR /root/

# 从后端构建阶段复制二进制文件
COPY --from=backend-builder /app/random-api .

# 从前端构建阶段复制静态文件
COPY --from=frontend-builder /app/web/out ./web/out

# 创建必要的目录
RUN mkdir -p /root/data/logs

# 暴露端口
EXPOSE 5003

# 使用 tini 作为初始化系统
ENTRYPOINT ["/sbin/tini", "--"]

# 启动应用
CMD ["./random-api"]
