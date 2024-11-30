package middleware

import (
	"net/http"
	"random-api-go/monitoring"
	"random-api-go/utils"
	"time"

	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(rate.Limit(1000), 100)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建自定义的ResponseWriter来捕获状态码
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 处理请求
		next.ServeHTTP(rw, r)

		// 记录请求数据
		duration := time.Since(start)
		monitoring.LogRequest(monitoring.RequestLog{
			Time:       time.Now().Unix(),
			Path:       r.URL.Path,
			Method:     r.Method,
			StatusCode: rw.statusCode,
			Latency:    float64(duration.Microseconds()) / 1000,
			IP:         utils.GetRealIP(r),
			Referer:    r.Referer(),
		})
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
