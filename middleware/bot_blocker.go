package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/woodchen-ink/go-web-utils/uautil"
)

// SmartBotBlockerMiddleware 智能机器人阻止中间件
// 根据访问路径应用不同的机器人策略:
// - 静态资源和健康检查: 允许所有访问(包括机器人)
// - 管理后台 API: 阻止所有机器人(包括合法爬虫)
// - 公开 API: 允许合法爬虫,阻止恶意机器人
func SmartBotBlockerMiddleware(next http.Handler) http.Handler {
	// 使用 uautil.BlockBotMiddleware 创建两个不同策略的中间件
	// allowLegitimate=true: 允许合法爬虫,只阻止恶意机器人
	allowLegitimateMiddleware := uautil.BlockBotMiddleware(true, "Forbidden: Bad Bot Detected")
	// allowLegitimate=false: 阻止所有机器人
	blockAllBotsMiddleware := uautil.BlockBotMiddleware(false, "Forbidden: Bot Access Not Allowed")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 1. 白名单路径: 不检查机器人,直接放行
		// 包括: 健康检查、指标监控、静态资源等
		whitelistPaths := []string{
			"/api/health",
			"/api/metrics",
			"/_next/",
			"/static/",
			"/favicon.ico",
		}
		for _, whitePath := range whitelistPaths {
			if strings.HasPrefix(path, whitePath) || path == whitePath {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 2. 管理后台路径: 阻止所有机器人(包括合法爬虫)
		// 包括: /api/admin/* 所有管理接口
		if strings.HasPrefix(path, "/api/admin/") {
			// 记录访问尝试
			userAgent := r.Header.Get("User-Agent")
			if uautil.IsBot(r, true) { // 检查是否是机器人
				realIP := GetRealIP(r)
				log.Printf("Blocked bot access to admin API from IP: %s, User-Agent: %s, Path: %s", realIP, userAgent, path)
			}
			blockAllBotsMiddleware(next).ServeHTTP(w, r)
			return
		}

		// 3. 公开 API 路径: 允许合法爬虫(如 GoogleBot, BingBot),阻止恶意机器人
		// 包括: 随机 API、统计接口、公开端点等
		if strings.HasPrefix(path, "/api/") || path == "/" {
			allowLegitimateMiddleware(next).ServeHTTP(w, r)
			return
		}

		// 4. 前端页面路径: 允许合法爬虫,有利于 SEO
		// 包括: /admin 管理后台前端页面
		if strings.HasPrefix(path, "/admin") {
			allowLegitimateMiddleware(next).ServeHTTP(w, r)
			return
		}

		// 5. 其他路径: 默认允许合法爬虫
		allowLegitimateMiddleware(next).ServeHTTP(w, r)
	})
}

// StrictBotBlockerMiddleware 严格的机器人阻止中间件
// 阻止所有机器人(包括合法爬虫),适用于需要严格保护的接口
func StrictBotBlockerMiddleware(next http.Handler) http.Handler {
	return uautil.BlockBotMiddleware(false, "Forbidden: Bot Access Not Allowed")(next)
}

// LenientBotBlockerMiddleware 宽松的机器人阻止中间件
// 允许合法爬虫,只阻止恶意机器人,适用于公开内容
func LenientBotBlockerMiddleware(next http.Handler) http.Handler {
	return uautil.BlockBotMiddleware(true, "Forbidden: Bad Bot Detected")(next)
}

// CustomBotBlockerMiddleware 自定义机器人阻止中间件
// 允许指定白名单路径和是否允许合法爬虫
func CustomBotBlockerMiddleware(whitelistPaths []string, allowLegitimate bool, customMessage string) func(http.Handler) http.Handler {
	if customMessage == "" {
		customMessage = "Forbidden: Bot Access Not Allowed"
	}

	botBlocker := uautil.BlockBotMiddleware(allowLegitimate, customMessage)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// 检查是否在白名单中
			for _, whitePath := range whitelistPaths {
				if strings.HasPrefix(path, whitePath) || path == whitePath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 应用机器人检测
			botBlocker(next).ServeHTTP(w, r)
		})
	}
}
