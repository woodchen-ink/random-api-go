package middleware

import (
	"net/http"
	"strings"

	"github.com/woodchen-ink/go-web-utils/uautil"
)

// RandomEndpointBrowserOnlyMiddleware 随机端点浏览器限制中间件
// 对随机端点(动态路径)仅允许浏览器访问,阻止机器人和脚本
// 其他路径(API、管理后台、静态资源等)不受此限制
func RandomEndpointBrowserOnlyMiddleware(next http.Handler) http.Handler {
	// 创建浏览器限制中间件
	browserOnly := uautil.BrowserOnlyMiddleware("仅限浏览器访问此端点")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 白名单路径: 不检查浏览器,直接放行
		// 包括: API接口、健康检查、指标监控、静态资源、管理后台等
		whitelistPrefixes := []string{
			"/api/",        // 所有API接口(包括管理后台API和公开API)
			"/_next/",      // Next.js静态资源
			"/static/",     // 静态文件
			"/favicon.ico", // 网站图标
			"/admin",       // 管理后台前端页面
		}

		// 检查是否在白名单中
		for _, prefix := range whitelistPrefixes {
			if strings.HasPrefix(path, prefix) || path == prefix {
				// 白名单路径,直接放行
				next.ServeHTTP(w, r)
				return
			}
		}

		// 根路径也直接放行(可能是前端首页)
		if path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// 其他所有路径都被视为随机端点,仅允许浏览器访问
		// 这些是动态配置的端点路径,如 /img, /video, /wallpaper 等
		browserOnly(next).ServeHTTP(w, r)
	})
}
