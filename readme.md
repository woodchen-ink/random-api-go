# Random API

Random API 是一个用 Go 语言编写的简单而强大的随机图片/视频 API 服务。它允许用户通过配置文件轻松管理和提供随机媒体内容。

## 特性

- 动态加载和缓存 CSV 文件内容
- 支持图片和视频随机分发
- 可自定义的 URL 路径配置
- Docker 支持，便于部署和扩展
- 详细的日志记录

## 快速开始

### 使用 Docker Compose

1. 克隆仓库：
   ```
   git clone https://github.com/yourusername/random-api.git
   cd random-api
   ```

2. 创建并编辑 `public/url.json` 文件来配置你的 URL 路径。

3. 启动服务：
   ```
   docker-compose up -d
   ```

4. 访问 `http://localhost:5003` 来使用 API。

### 手动运行

1. 确保你已安装 Go 1.21 或更高版本。

2. 克隆仓库并进入项目目录。

3. 运行以下命令：
   ```
   go mod download
   go run main.go
   ```

4. 服务将在 `http://localhost:5003` 上运行。

## 配置

### url.json

在 `public/url.json` 文件中配置你的 URL 路径和对应的 CSV 文件：

```json
{
  "pic": {
    "example": "https://example.com/pics.csv"
  },
  "video": {
    "example": "https://example.com/videos.csv"
  }
}
```

### CSV 文件

CSV 文件应包含每行一个 URL。例如：

```
https://example.com/image1.jpg
https://example.com/image2.jpg
https://example.com/image3.jpg
```

## API 使用

访问 `/pic/example` 或 `/video/example` 将重定向到相应 CSV 文件中的随机 URL。

## 日志

日志文件位于 `logs/server.log`。使用 Docker Compose 时，可以通过卷挂载访问日志。

## 贡献

欢迎贡献！请提交 pull request 或创建 issue 来提出建议和报告 bug。

## 许可

[MIT License](LICENSE)