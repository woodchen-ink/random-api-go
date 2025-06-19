package middleware

import (
	"net/http"
	"random-api-go/monitoring"
	"random-api-go/service"
	"random-api-go/utils"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(rate.Limit(1000), 100)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

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

		// 对于管理后台API请求，跳过域名统计以提升性能
		if !strings.HasPrefix(r.URL.Path, "/api/admin/") {
			// 记录域名统计
			domainStatsService := service.GetDomainStatsService()
			domainStatsService.RecordRequest(r.URL.Path, r.Referer())
		}
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
