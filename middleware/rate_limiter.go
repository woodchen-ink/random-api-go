package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// 限流配置：每个 IP 每秒允许 10 个请求，突发容量 20
// 可根据实际需求调整：
// - 严格防护: rate.Limit(5), 10
// - 中等防护: rate.Limit(10), 20
// - 宽松防护: rate.Limit(20), 40
const (
	rateLimitPerSecond = 20
	rateLimitBurst     = 40
)

// IPRateLimiter 基于 IP 的限流器
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter 创建新的 IP 限流器
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}

	// 启动定期清理过期限流器的 goroutine
	go limiter.cleanupExpiredLimiters()

	return limiter
}

// GetLimiter 获取指定 IP 的限流器
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters[ip] = limiter
	}

	return limiter
}

// cleanupExpiredLimiters 定期清理长时间未使用的限流器，防止内存泄漏
func (i *IPRateLimiter) cleanupExpiredLimiters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		i.mu.Lock()
		// 清空所有限流器，下次请求时会重新创建
		// 这样可以释放长时间不活跃的 IP 占用的内存
		i.limiters = make(map[string]*rate.Limiter)
		i.mu.Unlock()
	}
}

var ipLimiter = NewIPRateLimiter(rateLimitPerSecond, rateLimitBurst)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从 context 中获取真实 IP（由 RealIPMiddleware 提供）
		// 使用 middleware.GetRealIP 复用 RealIPMiddleware 的结果
		ip := GetRealIP(r)

		// 获取该 IP 的限流器
		limiter := ipLimiter.GetLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
