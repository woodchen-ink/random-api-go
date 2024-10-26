# Random API

介绍,使用方法和更新记录: https://q58.org/t/topic/127

Random API 是一个用 Go 语言编写的简单而强大的随机图片/视频 API 服务。它允许用户通过配置文件轻松管理和提供随机媒体内容。

## 特性

- 动态加载和缓存 CSV 文件内容
- 支持图片和视频随机分发
- 可自定义的 URL 路径配置
- Docker 支持，便于部署和扩展
- 详细的日志记录

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


### 推荐的nginx反代配置

``` 
location ^~ / {
    proxy_pass http://127.0.0.1:5003; 
    proxy_set_header Host $host; 
    proxy_set_header X-Real-IP $remote_addr; 
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; 
    proxy_set_header REMOTE-HOST $remote_addr; 
    proxy_set_header Upgrade $http_upgrade; 
    proxy_set_header Connection "upgrade"; 
    proxy_set_header X-Forwarded-Proto $scheme; 
    proxy_http_version 1.1; 
    add_header X-Cache $upstream_cache_status; 
    add_header Cache-Control no-cache; 
    proxy_ssl_server_name off; 
    add_header Strict-Transport-Security "max-age=31536000"; 
}
```

## 日志

日志文件位于 `/root/data/server.log`。使用 Docker Compose 时，可以通过卷挂载访问日志。

## 贡献

欢迎贡献！请提交 pull request 或创建 issue 来提出建议和报告 bug。

## 许可

[MIT License](LICENSE)
