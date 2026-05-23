package middleware

import (
	"net/http"
	"strings"

	"random-api-go/service"
)

// RefererBlockMiddleware 拦截来自被禁用域名的随机端点请求
// 仅作用于"非白名单路径" (与 RandomEndpointBrowserOnlyMiddleware 的判定一致)
// 命中黑名单 → 直接返回 403, 不进入统计中间件
func RefererBlockMiddleware(next http.Handler) http.Handler {
	whitelistPrefixes := []string{
		"/api/",
		"/_next/",
		"/static/",
		"/favicon.ico",
		"/admin",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			next.ServeHTTP(w, r)
			return
		}
		for _, p := range whitelistPrefixes {
			if strings.HasPrefix(path, p) || path == p {
				next.ServeHTTP(w, r)
				return
			}
		}

		svc := service.GetDomainStatsService()
		domain := svc.ExtractDomain(r.Referer())
		if svc.IsBlocked(domain) {
			http.Error(w, "Referer host has been blocked", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
