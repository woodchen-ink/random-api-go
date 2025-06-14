package handlers

import (
	"fmt"
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

	// 添加调试日志
	fmt.Printf("DEBUG: 请求路径: %s\n", path)

	// 实现 try_files $uri $uri/ @router 逻辑
	filePath := s.tryFiles(path)

	fmt.Printf("DEBUG: 最终文件路径: %s\n", filePath)

	// 设置正确的 Content-Type
	s.setContentType(w, filePath)

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
		fmt.Printf("DEBUG: 找到直接文件: %s\n", directPath)
		return directPath
	}

	// 2. 尝试 $uri/ - 目录下的 index.html
	if requestPath != "/" {
		dirPath := filepath.Join(s.staticDir, requestPath)
		if s.isDirectory(dirPath) {
			indexPath := filepath.Join(dirPath, "index.html")
			if s.fileExists(indexPath) {
				fmt.Printf("DEBUG: 找到目录下的 index.html: %s\n", indexPath)
				return indexPath
			}
		}

		// 也尝试添加 .html 扩展名
		htmlPath := directPath + ".html"
		if s.fileExists(htmlPath) {
			fmt.Printf("DEBUG: 找到 HTML 文件: %s\n", htmlPath)
			return htmlPath
		}
	}

	// 3. @router - 回退到根目录的 index.html (SPA 路由处理)
	fallbackPath := filepath.Join(s.staticDir, "index.html")
	fmt.Printf("DEBUG: 回退到根 index.html: %s\n", fallbackPath)
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
