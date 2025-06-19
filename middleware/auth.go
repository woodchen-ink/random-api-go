package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// UserInfo OAuth用户信息结构
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// TokenCache token缓存项
type TokenCache struct {
	UserInfo  *UserInfo
	ExpiresAt time.Time
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	tokenCache sync.Map // map[string]*TokenCache
	cacheTTL   time.Duration
}

// NewAuthMiddleware 创建新的认证中间件
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{
		cacheTTL: 30 * time.Minute, // token缓存30分钟
	}
}

// RequireAuth 认证中间件，验证 OAuth 令牌
func (am *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Authorization header 获取令牌
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// 检查 Bearer 前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			http.Error(w, "Token required", http.StatusUnauthorized)
			return
		}

		// 先检查缓存
		if cached, ok := am.tokenCache.Load(token); ok {
			tokenCache := cached.(*TokenCache)
			// 检查缓存是否过期
			if time.Now().Before(tokenCache.ExpiresAt) {
				// 缓存有效，直接通过
				next(w, r)
				return
			} else {
				// 缓存过期，删除
				am.tokenCache.Delete(token)
			}
		}

		// 缓存未命中或已过期，验证令牌
		userInfo, err := am.getUserInfo(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// 将结果缓存
		am.tokenCache.Store(token, &TokenCache{
			UserInfo:  userInfo,
			ExpiresAt: time.Now().Add(am.cacheTTL),
		})

		// 验证成功，继续处理请求
		next(w, r)
	}
}

// getUserInfo 通过访问令牌获取用户信息
func (am *AuthMiddleware) getUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", "https://connect.czl.net/api/oauth2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: 10 * time.Second, // 添加超时时间
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info, status: %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// InvalidateToken 使token缓存失效（用于登出等场景）
func (am *AuthMiddleware) InvalidateToken(token string) {
	am.tokenCache.Delete(token)
}

// GetCacheStats 获取缓存统计信息（用于监控）
func (am *AuthMiddleware) GetCacheStats() map[string]interface{} {
	count := 0
	am.tokenCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return map[string]interface{}{
		"cached_tokens": count,
		"cache_ttl":     am.cacheTTL.String(),
	}
}
