package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type StaticHandler struct {
	staticDir string
}

func NewStaticHandler(staticDir string) *StaticHandler {
	return &StaticHandler{
		staticDir: staticDir,
	}
}

// ServeStatic 处理静态文件请求
func (s *StaticHandler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// 获取请求路径
	path := r.URL.Path

	// 如果是根路径，重定向到 index.html
	if path == "/" {
		path = "/index.html"
	}

	// 处理 Next.js 静态导出的路由问题
	filePath := s.resolveFilePath(path)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 如果文件不存在，检查是否是前端路由
		if s.isFrontendRoute(path) {
			// 对于前端路由，返回 index.html
			filePath = filepath.Join(s.staticDir, "index.html")
		} else {
			// 不是前端路由，返回 404
			http.NotFound(w, r)
			return
		}
	}

	// 设置正确的 Content-Type
	s.setContentType(w, filePath)

	// 服务文件
	http.ServeFile(w, r, filePath)
}

// resolveFilePath 解析文件路径，处理 Next.js 静态导出的路由问题
func (s *StaticHandler) resolveFilePath(path string) string {
	// 移除查询参数和锚点
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	if idx := strings.Index(path, "#"); idx != -1 {
		path = path[:idx]
	}

	// 构建初始文件路径
	filePath := filepath.Join(s.staticDir, path)

	// 如果路径以斜杠结尾，尝试查找 index.html
	if strings.HasSuffix(path, "/") {
		indexPath := filepath.Join(filePath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return indexPath
		}
	} else {
		// 如果路径不以斜杠结尾，先检查是否存在对应的文件
		if _, err := os.Stat(filePath); err == nil {
			return filePath
		}

		// 如果文件不存在，尝试查找对应目录下的 index.html
		indexPath := filepath.Join(filePath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return indexPath
		}

		// 尝试添加 .html 扩展名
		htmlPath := filePath + ".html"
		if _, err := os.Stat(htmlPath); err == nil {
			return htmlPath
		}
	}

	return filePath
}

// isFrontendRoute 判断是否是前端路由
func (s *StaticHandler) isFrontendRoute(path string) bool {
	// 前端路由通常以 /admin 开头
	if strings.HasPrefix(path, "/admin") {
		return true
	}

	// 排除 API 路径和静态资源
	if strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/_next/") ||
		strings.HasPrefix(path, "/static/") ||
		strings.Contains(path, ".") {
		return false
	}

	return false
}

// setContentType 设置正确的 Content-Type
func (s *StaticHandler) setContentType(w http.ResponseWriter, filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	}
}
