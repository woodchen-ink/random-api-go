package middleware

import (
	"context"
	"net/http"

	"github.com/woodchen-ink/go-web-utils/iputil"
)

// contextKey 用于在 context 中存储真实 IP
type contextKey string

const (
	// RealIPKey 用于在 context 中存储真实 IP 的 key
	RealIPKey contextKey = "real_ip"
)

// RealIPMiddleware 中间件用于检测用户真实 IP
// 使用 go-web-utils 的 iputil 包来获取真实 IP
// 支持从多个 header 中提取 IP,包括 X-Forwarded-For, X-Real-IP 等
func RealIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 使用 iputil.GetClientIP 获取真实 IP
		realIP := iputil.GetClientIP(r)

		// 将真实 IP 存储到 context 中,方便后续处理器使用
		ctx := context.WithValue(r.Context(), RealIPKey, realIP)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetRealIP 从 request context 中获取真实 IP
func GetRealIP(r *http.Request) string {
	if ip, ok := r.Context().Value(RealIPKey).(string); ok {
		return ip
	}
	// 如果 context 中没有,直接使用 iputil 获取
	return iputil.GetClientIP(r)
}
