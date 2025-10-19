package middleware

import (
	"net/http"

	"golang.org/x/time/rate"
)

// 限流配置：每秒允许 50 个请求，突发容量 100
// 可根据实际需求调整：
// - 严格防护: rate.NewLimiter(10, 20)
// - 中等防护: rate.NewLimiter(50, 100)
// - 宽松防护: rate.NewLimiter(100, 200)
var limiter = rate.NewLimiter(50, 100)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
