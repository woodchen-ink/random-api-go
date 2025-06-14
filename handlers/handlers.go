package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"random-api-go/monitoring"
	"random-api-go/router"
	"random-api-go/services"
	"random-api-go/stats"
	"random-api-go/utils"
	"strings"
	"time"
)

type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type Handlers struct {
	Stats           *stats.StatsManager
	endpointService *services.EndpointService
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
			h.endpointService = services.GetEndpointService()
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

func (h *Handlers) HandleURLStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 使用新的端点服务获取统计信息
	if h.endpointService == nil {
		h.endpointService = services.GetEndpointService()
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

func (h *Handlers) Setup(r *router.Router) {
	// 通用路由处理 - 匹配所有路径
	r.HandleFunc("/", h.HandleAPIRequest)

	// API 统计和监控
	r.HandleFunc("/api/stats", h.HandleStats)
	r.HandleFunc("/api/urlstats", h.HandleURLStats)
	r.HandleFunc("/api/metrics", h.HandleMetrics)
}
