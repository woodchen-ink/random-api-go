package handler

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

// ServeStatic 处理静态文件请求，实现类似 Nginx try_files 的逻辑
func (s *StaticHandler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// 实现 try_files $uri $uri/ @router 逻辑
	filePath := s.tryFiles(path)

	// 设置正确的 Content-Type 与缓存头（按资源类型分流）
	s.setResponseHeaders(w, r.URL.Path, filePath)

	// 服务文件
	http.ServeFile(w, r, filePath)
}

// tryFiles 实现类似 Nginx try_files 的逻辑
func (s *StaticHandler) tryFiles(requestPath string) string {
	// 清理路径
	if requestPath == "" {
		requestPath = "/"
	}

	// 移除查询参数
	if idx := strings.Index(requestPath, "?"); idx != -1 {
		requestPath = requestPath[:idx]
	}

	// 1. 尝试 $uri - 直接文件路径
	directPath := filepath.Join(s.staticDir, requestPath)
	if s.fileExists(directPath) && !s.isDirectory(directPath) {
		return directPath
	}

	// 2. 尝试 $uri/ - 目录下的 index.html
	if requestPath != "/" {
		dirPath := filepath.Join(s.staticDir, requestPath)
		if s.isDirectory(dirPath) {
			indexPath := filepath.Join(dirPath, "index.html")
			if s.fileExists(indexPath) {
				return indexPath
			}
		}

		// 也尝试添加 .html 扩展名
		htmlPath := directPath + ".html"
		if s.fileExists(htmlPath) {
			return htmlPath
		}
	}

	// 3. @router - 回退到根目录的 index.html (SPA 路由处理)
	fallbackPath := filepath.Join(s.staticDir, "index.html")
	return fallbackPath
}

// fileExists 检查文件是否存在
func (s *StaticHandler) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isDirectory 检查路径是否为目录
func (s *StaticHandler) isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// setResponseHeaders 按资源类型设置 Content-Type / Cache-Control / Vary
// 关键点: Next.js 16 App Router 的 *.txt 是 RSC payload, 必须返回 text/x-component,
// 否则客户端 router 收到非 RSC content-type 会回退到硬导航, 浏览器直接展示纯文本 payload。
func (s *StaticHandler) setResponseHeaders(w http.ResponseWriter, urlPath, filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".txt":
		// Next.js RSC payload (例如 /admin/home.txt)
		w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
		w.Header().Set("Vary", "RSC, Next-Router-State-Tree, Next-Router-Prefetch, Next-Url")
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Vary", "RSC, Next-Router-State-Tree, Next-Router-Prefetch, Next-Url")
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

	// 带 hash 的不可变静态资源 (Next.js: /_next/static/*) 走长期 immutable
	if strings.HasPrefix(urlPath, "/_next/static/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Del("Vary")
	}
}
