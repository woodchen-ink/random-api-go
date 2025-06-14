# Random API Go

一个基于Go的随机API服务，支持多种数据源和管理后台。

## 功能特性

- 🎯 支持多种数据源：兰空图床API、手动配置、通用API接口
- 🔐 OAuth2.0 管理后台登录（CZL Connect）
- 💾 SQLite数据库存储
- ⚡ 内存缓存机制
- 🔄 URL替换规则
- 📝 可配置首页内容
- 🎨 现代化Web管理界面

## 环境变量配置

复制 `env.example` 为 `.env` 并配置以下环境变量：

```bash
# 服务器配置
PORT=:5003                    # 服务端口
READ_TIMEOUT=30s             # 读取超时
WRITE_TIMEOUT=30s            # 写入超时
MAX_HEADER_BYTES=1048576     # 最大请求头大小

# 数据存储目录
DATA_DIR=./data              # 数据存储目录

# OAuth2.0 配置 (必需)
OAUTH_CLIENT_ID=your-oauth-client-id        # CZL Connect 客户端ID
OAUTH_CLIENT_SECRET=your-oauth-client-secret # CZL Connect 客户端密钥
```

## 快速开始

1. 克隆项目
```bash
git clone <repository-url>
cd random-api-go
```

2. 配置环境变量
```bash
cp env.example .env
# 编辑 .env 文件，填入正确的 OAuth 配置
```

3. 运行服务
```bash
go run main.go
```

4. 访问服务
- 首页: http://localhost:5003
- 管理后台: http://localhost:5003/admin

## OAuth2.0 配置

本项目使用 CZL Connect 作为 OAuth2.0 提供商：

- 授权端点: https://connect.czl.net/oauth2/authorize
- 令牌端点: https://connect.czl.net/api/oauth2/token
- 用户信息端点: https://connect.czl.net/api/oauth2/userinfo

请在 CZL Connect 中注册应用并获取 `client_id` 和 `client_secret`。

## API 端点

### 公开API
- `GET /` - 首页
- `GET /{endpoint}` - 随机API端点

### 管理API
- `GET /admin/api/oauth-config` - 获取OAuth配置
- `POST /admin/api/oauth-verify` - 验证OAuth授权码
- `GET /admin/api/endpoints` - 列出所有端点
- `POST /admin/api/endpoints/` - 创建端点
- `GET /admin/api/endpoints/{id}` - 获取端点详情
- `PUT /admin/api/endpoints/{id}` - 更新端点
- `DELETE /admin/api/endpoints/{id}` - 删除端点
- `POST /admin/api/data-sources` - 创建数据源
- `GET /admin/api/url-replace-rules` - 列出URL替换规则
- `POST /admin/api/url-replace-rules/` - 创建URL替换规则
- `GET /admin/api/home-config` - 获取首页配置
- `PUT /admin/api/home-config/` - 更新首页配置

## 数据源类型

1. **兰空图床 (lankong)**: 从兰空图床API获取图片
2. **手动配置 (manual)**: 手动配置的URL列表
3. **API GET (api_get)**: 从GET接口获取数据
4. **API POST (api_post)**: 从POST接口获取数据

## 部署

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o random-api-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/random-api-server .
COPY --from=builder /app/web ./web
EXPOSE 5003
CMD ["./random-api-server"]
```

### 环境变量部署

确保在生产环境中正确设置所有必需的环境变量，特别是OAuth配置。

## 许可证

MIT License
