package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"random-api-go/database"
	"random-api-go/initapp"
	"random-api-go/monitoring"
	"random-api-go/service"
	"random-api-go/stats"
	"random-api-go/utils"
	"strings"
	"sync"
	"time"
)

type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type Handlers struct {
	Stats           *stats.StatsManager
	endpointService *service.EndpointService
	urlStatsCache   map[string]struct {
		TotalURLs int `json:"total_urls"`
	}
	urlStatsCacheTime time.Time
	urlStatsMutex     sync.RWMutex
	cacheDuration     time.Duration
}

func NewHandlers(statsManager *stats.StatsManager) *Handlers {
	return &Handlers{
		Stats:         statsManager,
		cacheDuration: 5 * time.Minute, // 缓存5分钟，减少首次访问等待时间
	}
}

func (h *Handlers) HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 创建一个响应通道，用于传递结果
	type result struct {
		url string
		err error
	}
	resultChan := make(chan result, 1)

	go func() {
		start := time.Now()
		realIP := utils.GetRealIP(r)

		path := strings.TrimPrefix(r.URL.Path, "/")

		// 初始化端点服务
		if h.endpointService == nil {
			h.endpointService = service.GetEndpointService()
		}

		// 使用新的端点服务
		randomURL, err := h.endpointService.GetRandomURL(path)
		if err != nil {
			monitoring.LogRequest(monitoring.RequestLog{
				Time:       time.Now().UnixMilli(),
				Path:       r.URL.Path,
				Method:     r.Method,
				StatusCode: http.StatusNotFound,
				Latency:    float64(time.Since(start).Microseconds()) / 1000,
				IP:         realIP,
				Referer:    r.Referer(),
			})
			resultChan <- result{err: fmt.Errorf("endpoint not found: %v", err)}
			return
		}

		// 成功获取到URL
		h.Stats.IncrementCalls(path)

		duration := time.Since(start)
		monitoring.LogRequest(monitoring.RequestLog{
			Time:       time.Now().UnixMilli(),
			Path:       r.URL.Path,
			Method:     r.Method,
			StatusCode: http.StatusFound,
			Latency:    float64(duration.Microseconds()) / 1000,
			IP:         realIP,
			Referer:    r.Referer(),
		})

		log.Printf(" %-12s | %-15s | %-6s | %-20s | %-20s | %-50s",
			duration,
			realIP,
			r.Method,
			r.URL.Path,
			r.Referer(),
			randomURL,
		)

		resultChan <- result{url: randomURL}
	}()

	// 等待结果或超时
	select {
	case res := <-resultChan:
		if res.err != nil {
			http.Error(w, res.err.Error(), http.StatusNotFound)
			return
		}
		http.Redirect(w, r, res.url, http.StatusFound)
	case <-ctx.Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
	}
}

func (h *Handlers) HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := h.Stats.GetStatsForAPI()

	// 包装数据格式以匹配前端期望
	response := map[string]interface{}{
		"Stats": stats,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding stats", http.StatusInternalServerError)
		log.Printf("Error encoding stats: %v", err)
	}
}

func (h *Handlers) HandlePublicEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 使用端点服务获取端点信息
	if h.endpointService == nil {
		h.endpointService = service.GetEndpointService()
	}

	endpoints, err := h.endpointService.ListEndpoints()
	if err != nil {
		http.Error(w, "Error getting endpoints", http.StatusInternalServerError)
		return
	}

	// 只返回公开信息，不包含数据源配置
	type PublicEndpoint struct {
		ID             uint   `json:"id"`
		Name           string `json:"name"`
		URL            string `json:"url"`
		Description    string `json:"description"`
		IsActive       bool   `json:"is_active"`
		ShowOnHomepage bool   `json:"show_on_homepage"`
		SortOrder      int    `json:"sort_order"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	}

	var publicEndpoints []PublicEndpoint
	for _, endpoint := range endpoints {
		publicEndpoints = append(publicEndpoints, PublicEndpoint{
			ID:             endpoint.ID,
			Name:           endpoint.Name,
			URL:            endpoint.URL,
			Description:    endpoint.Description,
			IsActive:       endpoint.IsActive,
			ShowOnHomepage: endpoint.ShowOnHomepage,
			SortOrder:      endpoint.SortOrder,
			CreatedAt:      endpoint.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      endpoint.UpdatedAt.Format(time.RFC3339),
		})
	}

	response := map[string]interface{}{
		"success": true,
		"data":    publicEndpoints,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleURLStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 检查缓存是否有效
	h.urlStatsMutex.RLock()
	if h.urlStatsCache != nil && time.Since(h.urlStatsCacheTime) < h.cacheDuration {
		// 使用缓存数据
		cache := h.urlStatsCache
		h.urlStatsMutex.RUnlock()

		if err := json.NewEncoder(w).Encode(cache); err != nil {
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		return
	}
	h.urlStatsMutex.RUnlock()

	// 缓存过期或不存在，重新计算
	if h.endpointService == nil {
		h.endpointService = service.GetEndpointService()
	}

	endpoints, err := h.endpointService.ListEndpoints()
	if err != nil {
		http.Error(w, "Error getting endpoint stats", http.StatusInternalServerError)
		return
	}

	// 转换为前端期望的格式
	response := make(map[string]struct {
		TotalURLs int `json:"total_urls"`
	})

	for _, endpoint := range endpoints {
		if endpoint.IsActive {
			totalURLs := 0
			for _, ds := range endpoint.DataSources {
				if ds.IsActive {
					// 尝试获取实际的URL数量
					urls, err := h.endpointService.GetDataSourceURLCount(&ds)
					if err != nil {
						log.Printf("Failed to get URL count for data source %d: %v", ds.ID, err)
						// 如果获取失败，使用估算值
						switch ds.Type {
						case "manual":
							totalURLs += 5 // 手动数据源估算
						case "lankong":
							totalURLs += 50 // 兰空图床估算
						case "api_get", "api_post":
							totalURLs += 1 // API数据源每次返回1个
						}
					} else {
						totalURLs += urls
					}
				}
			}
			response[endpoint.URL] = struct {
				TotalURLs int `json:"total_urls"`
			}{
				TotalURLs: totalURLs,
			}
		}
	}

	// 更新缓存
	h.urlStatsMutex.Lock()
	h.urlStatsCache = response
	h.urlStatsCacheTime = time.Now()
	h.urlStatsMutex.Unlock()

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := monitoring.CollectMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// HandlePublicHomeConfig 处理公开的首页配置请求
func (h *Handlers) HandlePublicHomeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 从数据库获取首页配置
	content := database.GetConfig("homepage_content", "# 欢迎使用随机API服务\n\n这是一个可配置的随机API服务。")

	response := map[string]interface{}{
		"success": true,
		"data":    map[string]string{"content": content},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

// HandleServiceConfig 处理服务配置请求
func (h *Handlers) HandleServiceConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 获取SERVICE_START_TIME环境变量
	serviceStartTime := os.Getenv("SERVICE_START_TIME")

	var data map[string]interface{}
	if serviceStartTime != "" {
		data = map[string]interface{}{
			"service_start_time": serviceStartTime,
		}
	} else {
		data = map[string]interface{}{}
	}

	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

// HandleHealth 处理健康检查请求
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 获取初始化状态
	initStatus := initapp.GetInitStatus()

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"init":      initStatus,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
