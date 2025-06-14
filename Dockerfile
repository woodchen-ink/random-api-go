# 前端构建阶段
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# 复制前端依赖文件（优先缓存依赖层）
COPY web/package*.json ./

# 安装前端依赖（包括开发依赖，构建需要）
RUN npm ci --prefer-offline --no-audit --progress=false

# 复制前端配置文件
COPY web/next.config.ts ./
COPY web/tsconfig.json ./
COPY web/postcss.config.mjs ./
COPY web/components.json ./
COPY web/eslint.config.mjs ./
COPY web/next-env.d.ts ./

# 复制前端源代码目录
COPY web/app ./app
COPY web/components ./components
COPY web/lib ./lib
COPY web/types ./types
COPY web/public ./public

# 构建前端静态文件
RUN npm run build

# 后端构建阶段
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go.mod 和 go.sum 文件（优先缓存依赖层）
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download && go mod verify

# 复制后端源代码
COPY . .

# 构建后端应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o random-api .

# 运行阶段
FROM alpine:latest

# 安装必要的工具
RUN apk --no-cache add ca-certificates tzdata tini && \
    adduser -D -s /bin/sh appuser

WORKDIR /app

# 从后端构建阶段复制二进制文件
COPY --from=backend-builder /app/random-api .

# 从前端构建阶段复制静态文件
COPY --from=frontend-builder /app/web/out ./web/out

# 创建必要的目录并设置权限
RUN mkdir -p /app/data/logs && \
    chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 5003

# 使用 tini 作为初始化系统
ENTRYPOINT ["/sbin/tini", "--"]

# 启动应用
CMD ["./random-api"]
