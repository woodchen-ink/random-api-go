package router

import (
	"net/http"
	"random-api-go/middleware"
	"strings"
)

type Router struct {
	mux            *http.ServeMux
	staticHandler  StaticHandler
	authMiddleware *middleware.AuthMiddleware
}

// Handler 接口定义处理器需要的方法
type Handler interface {
	Setup(r *Router)
}

// StaticHandler 接口定义静态文件处理器需要的方法
type StaticHandler interface {
	ServeStatic(w http.ResponseWriter, r *http.Request)
}

// AdminHandler 接口定义管理后台处理器需要的方法
type AdminHandler interface {
	// OAuth相关
	GetOAuthConfig(w http.ResponseWriter, r *http.Request)
	VerifyOAuthToken(w http.ResponseWriter, r *http.Request)
	HandleOAuthCallback(w http.ResponseWriter, r *http.Request)

	// 端点管理
	HandleEndpoints(w http.ResponseWriter, r *http.Request)
	HandleEndpointByID(w http.ResponseWriter, r *http.Request)
	HandleEndpointDataSources(w http.ResponseWriter, r *http.Request)
	UpdateEndpointSortOrder(w http.ResponseWriter, r *http.Request)

	// 数据源管理
	CreateDataSource(w http.ResponseWriter, r *http.Request)
	HandleDataSourceByID(w http.ResponseWriter, r *http.Request)
	SyncDataSource(w http.ResponseWriter, r *http.Request)

	// URL替换规则
	ListURLReplaceRules(w http.ResponseWriter, r *http.Request)
	CreateURLReplaceRule(w http.ResponseWriter, r *http.Request)
	HandleURLReplaceRuleByID(w http.ResponseWriter, r *http.Request)

	// 首页配置
	GetHomePageConfig(w http.ResponseWriter, r *http.Request)
	UpdateHomePageConfig(w http.ResponseWriter, r *http.Request)

	// 通用配置管理
	ListConfigs(w http.ResponseWriter, r *http.Request)
	CreateOrUpdateConfig(w http.ResponseWriter, r *http.Request)
	DeleteConfigByKey(w http.ResponseWriter, r *http.Request)
}

func New() *Router {
	return &Router{
		mux:            http.NewServeMux(),
		authMiddleware: middleware.NewAuthMiddleware(),
	}
}

func (r *Router) Setup(h Handler) {
	// 设置API路由
	h.Setup(r)
}

// SetupStaticRoutes 设置静态文件路由
func (r *Router) SetupStaticRoutes(staticHandler StaticHandler) {
	r.staticHandler = staticHandler
}

// SetupAdminRoutes 设置管理后台路由
func (r *Router) SetupAdminRoutes(adminHandler AdminHandler) {
	// OAuth配置API（前端需要获取client_id等信息）- 不需要认证
	r.HandleFunc("/api/admin/oauth-config", adminHandler.GetOAuthConfig)
	// OAuth令牌验证API（保留，以防需要）- 不需要认证
	r.HandleFunc("/api/admin/oauth-verify", adminHandler.VerifyOAuthToken)
	// OAuth回调处理（使用API前缀以便区分前后端）- 不需要认证
	r.HandleFunc("/api/admin/oauth/callback", adminHandler.HandleOAuthCallback)

	// 管理后台API路由 - 需要认证
	r.HandleFunc("/api/admin/endpoints", r.authMiddleware.RequireAuth(adminHandler.HandleEndpoints))

	// 端点排序路由 - 需要认证
	r.HandleFunc("/api/admin/endpoints/sort-order", r.authMiddleware.RequireAuth(adminHandler.UpdateEndpointSortOrder))

	// 数据源路由 - 需要认证
	r.HandleFunc("/api/admin/data-sources", r.authMiddleware.RequireAuth(adminHandler.CreateDataSource))

	// 端点相关路由 - 需要认证
	r.HandleFunc("/api/admin/endpoints/", r.authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/data-sources") {
			adminHandler.HandleEndpointDataSources(w, r)
		} else {
			adminHandler.HandleEndpointByID(w, r)
		}
	}))

	// 数据源操作路由 - 需要认证
	r.HandleFunc("/api/admin/data-sources/", r.authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/sync") {
			adminHandler.SyncDataSource(w, r)
		} else {
			adminHandler.HandleDataSourceByID(w, r)
		}
	}))

	// URL替换规则路由 - 需要认证
	r.HandleFunc("/api/admin/url-replace-rules", r.authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			adminHandler.ListURLReplaceRules(w, r)
		} else if r.Method == http.MethodPost {
			adminHandler.CreateURLReplaceRule(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	r.HandleFunc("/api/admin/url-replace-rules/", r.authMiddleware.RequireAuth(adminHandler.HandleURLReplaceRuleByID))

	// 首页配置路由 - 需要认证
	r.HandleFunc("/api/admin/home-config", r.authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			adminHandler.GetHomePageConfig(w, r)
		} else {
			adminHandler.UpdateHomePageConfig(w, r)
		}
	}))

	// 通用配置管理路由 - 需要认证
	r.HandleFunc("/api/admin/configs", r.authMiddleware.RequireAuth(adminHandler.ListConfigs))
	r.HandleFunc("/api/admin/configs/", r.authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			adminHandler.DeleteConfigByKey(w, r)
		} else {
			adminHandler.CreateOrUpdateConfig(w, r)
		}
	}))
}

func (r *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.mux.HandleFunc(pattern, handler)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 首先检查是否是静态文件请求或前端路由
	if r.staticHandler != nil && r.shouldServeStatic(req.URL.Path) {
		r.staticHandler.ServeStatic(w, req)
		return
	}

	// 否则使用默认的路由处理
	r.mux.ServeHTTP(w, req)
}

// shouldServeStatic 判断是否应该由静态文件处理器处理
func (r *Router) shouldServeStatic(path string) bool {
	// API 路径不由静态文件处理器处理
	if strings.HasPrefix(path, "/api/") {
		return false
	}

	// 根路径由静态文件处理器处理（返回首页）
	if path == "/" {
		return true
	}

	// 前端路由（以 /admin 开头）由静态文件处理器处理
	if strings.HasPrefix(path, "/admin") {
		return true
	}

	// 静态资源文件（包含文件扩展名或特定前缀）
	if strings.HasPrefix(path, "/_next/") ||
		strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/favicon.ico") ||
		r.hasFileExtension(path) {
		return true
	}

	// 其他路径可能是动态API端点，不由静态文件处理器处理
	return false
}

// hasFileExtension 检查路径是否包含文件扩展名
func (r *Router) hasFileExtension(path string) bool {
	// 获取路径的最后一部分
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false
	}

	lastPart := parts[len(parts)-1]

	// 检查是否包含点号且不是隐藏文件
	if strings.Contains(lastPart, ".") && !strings.HasPrefix(lastPart, ".") {
		// 常见的文件扩展名
		commonExts := []string{
			".html", ".css", ".js", ".json", ".png", ".jpg", ".jpeg",
			".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot",
			".txt", ".xml", ".pdf", ".zip", ".mp4", ".mp3",
		}

		for _, ext := range commonExts {
			if strings.HasSuffix(strings.ToLower(lastPart), ext) {
				return true
			}
		}
	}

	return false
}
