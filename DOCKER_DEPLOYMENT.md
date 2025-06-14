# Docker 部署说明

## 概述

本项目现在使用单一Docker镜像部署，包含前端（Next.js）和后端（Go）。前端被构建为静态文件并由后端服务器提供服务。

## 架构变更

### 之前的架构
- 前端：独立的Next.js开发服务器
- 后端：Go API服务器
- 部署：需要分别处理前后端

### 现在的架构
- 前端：构建为静态文件（Next.js export）
- 后端：Go服务器同时提供API和静态文件服务
- 部署：单一Docker镜像包含完整应用

## 构建流程

### 多阶段Docker构建

1. **前端构建阶段**
   ```dockerfile
   FROM node:20-alpine AS frontend-builder
   # 安装依赖并构建前端静态文件
   RUN npm run build
   ```

2. **后端构建阶段**
   ```dockerfile
   FROM golang:1.23-alpine AS backend-builder
   # 构建Go二进制文件
   RUN go build -o random-api .
   ```

3. **运行阶段**
   ```dockerfile
   FROM alpine:latest
   # 复制后端二进制文件和前端静态文件
   COPY --from=backend-builder /app/random-api .
   COPY --from=frontend-builder /app/web/out ./web/out
   ```

## 路由处理

### 静态文件优先级
后端路由器现在按以下优先级处理请求：

1. **API路径** (`/api/*`) → 后端API处理器
2. **静态文件** (包含文件扩展名) → 静态文件服务
3. **前端路由** (`/`, `/admin/*`) → 返回index.html
4. **动态API端点** (其他路径) → 后端API处理器

### 路由判断逻辑
```go
func (r *Router) shouldServeStatic(path string) bool {
    // API路径不由静态文件处理
    if strings.HasPrefix(path, "/api/") {
        return false
    }
    
    // 根路径和前端路由
    if path == "/" || strings.HasPrefix(path, "/admin") {
        return true
    }
    
    // 静态资源文件
    if r.hasFileExtension(path) {
        return true
    }
    
    return false
}
```

## 部署配置

### GitHub Actions
- 自动构建多架构镜像 (amd64/arm64)
- 推送到Docker Hub
- 自动部署到服务器

### Docker Compose
```yaml
services:
  random-api-go:
    container_name: random-api-go
    image: woodchen/random-api-go:latest
    ports:
      - "5003:5003"
    volumes:
      - ./data:/root/data
    environment:
      - TZ=Asia/Shanghai
      - BASE_URL=https://random-api.czl.net
    restart: unless-stopped
```

## 访问地址

部署完成后，可以通过以下地址访问：

- **前端首页**: `http://localhost:5003/`
- **管理后台**: `http://localhost:5003/admin`
- **API统计**: `http://localhost:5003/api/stats`
- **动态API端点**: `http://localhost:5003/{endpoint-name}`

## 开发环境

### 本地开发
在开发环境中，前端仍然可以使用开发服务器：

```bash
# 启动后端
go run main.go

# 启动前端（另一个终端）
cd web
npm run dev
```

前端的`next.config.ts`会在开发环境中自动代理API请求到后端。

### 生产构建测试
```bash
# 构建前端
cd web
npm run build

# 启动后端（会自动服务静态文件）
cd ..
go run main.go
```

## 注意事项

1. **前端路由**: 所有前端路由都会返回`index.html`，由前端路由器处理
2. **API端点冲突**: 确保动态API端点名称不与静态文件路径冲突
3. **缓存**: 静态文件会被适当缓存，API响应不会被缓存
4. **错误处理**: 404错误会根据路径类型返回相应的错误页面 