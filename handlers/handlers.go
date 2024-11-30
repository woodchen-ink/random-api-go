package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	Stats *stats.StatsManager
}

func (h *Handlers) HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		start := time.Now()
		realIP := utils.GetRealIP(r)

		// 获取并处理 referer
		sourceInfo := "direct"
		if referer := r.Referer(); referer != "" {
			if parsedURL, err := url.Parse(referer); err == nil {
				sourceInfo = parsedURL.Host + parsedURL.Path
				if parsedURL.RawQuery != "" {
					sourceInfo += "?" + parsedURL.RawQuery
				}
			}
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		pathSegments := strings.Split(path, "/")

		if len(pathSegments) < 2 {
			monitoring.LogRequest(monitoring.RequestLog{
				Time:       time.Now().Unix(),
				Path:       r.URL.Path,
				Method:     r.Method,
				StatusCode: http.StatusNotFound,
				Latency:    float64(time.Since(start).Microseconds()) / 1000,
				IP:         realIP,
				Referer:    sourceInfo,
			})
			http.NotFound(w, r)
			return
		}

		prefix := pathSegments[0]
		suffix := pathSegments[1]

		services.Mu.RLock()
		csvPath, ok := services.CSVPathsCache[prefix][suffix]
		services.Mu.RUnlock()

		if !ok {
			monitoring.LogRequest(monitoring.RequestLog{
				Time:       time.Now().Unix(),
				Path:       r.URL.Path,
				Method:     r.Method,
				StatusCode: http.StatusNotFound,
				Latency:    float64(time.Since(start).Microseconds()) / 1000,
				IP:         realIP,
				Referer:    sourceInfo,
			})
			http.NotFound(w, r)
			return
		}

		selector, err := services.GetCSVContent(csvPath)
		if err != nil {
			log.Printf("Error fetching CSV content: %v", err)
			monitoring.LogRequest(monitoring.RequestLog{
				Time:       time.Now().Unix(),
				Path:       r.URL.Path,
				Method:     r.Method,
				StatusCode: http.StatusInternalServerError,
				Latency:    float64(time.Since(start).Microseconds()) / 1000,
				IP:         realIP,
				Referer:    sourceInfo,
			})
			http.Error(w, "Failed to fetch content", http.StatusInternalServerError)
			return
		}

		if len(selector.URLs) == 0 {
			monitoring.LogRequest(monitoring.RequestLog{
				Time:       time.Now().Unix(),
				Path:       r.URL.Path,
				Method:     r.Method,
				StatusCode: http.StatusNotFound,
				Latency:    float64(time.Since(start).Microseconds()) / 1000,
				IP:         realIP,
				Referer:    sourceInfo,
			})
			http.Error(w, "No content available", http.StatusNotFound)
			return
		}

		randomURL := selector.GetRandomURL()
		endpoint := fmt.Sprintf("%s/%s", prefix, suffix)
		h.Stats.IncrementCalls(endpoint)

		duration := time.Since(start)
		monitoring.LogRequest(monitoring.RequestLog{
			Time:       time.Now().Unix(),
			Path:       r.URL.Path,
			Method:     r.Method,
			StatusCode: http.StatusFound,
			Latency:    float64(duration.Microseconds()) / 1000,
			IP:         realIP,
			Referer:    sourceInfo,
		})

		log.Printf(" %-12s | %-15s | %-6s | %-20s | %-20s | %-50s",
			duration,
			realIP,
			r.Method,
			r.URL.Path,
			sourceInfo,
			randomURL,
		)

		done <- struct{}{}
	}()

	select {
	case <-done:
		// 请求成功完成
	case <-ctx.Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
	}
}

func (h *Handlers) HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := h.Stats.GetStats()
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Error encoding stats", http.StatusInternalServerError)
		log.Printf("Error encoding stats: %v", err)
	}
}

func (h *Handlers) HandleURLStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := services.GetURLCounts()

	// 转换为前端期望的格式
	response := make(map[string]struct {
		TotalURLs int `json:"total_urls"`
	})

	for endpoint, stat := range stats {
		response[endpoint] = struct {
			TotalURLs int `json:"total_urls"`
		}{
			TotalURLs: stat.TotalURLs,
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
	// 动态路由处理
	r.HandleFunc("/pic/", h.HandleAPIRequest)
	r.HandleFunc("/video/", h.HandleAPIRequest)

	// API 统计和监控
	r.HandleFunc("/stats", h.HandleStats)
	r.HandleFunc("/urlstats", h.HandleURLStats)
	r.HandleFunc("/metrics", h.HandleMetrics)
}
