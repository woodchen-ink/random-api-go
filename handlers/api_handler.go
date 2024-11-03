package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"random-api-go/services"
	"random-api-go/stats"
	"random-api-go/utils"

	"strings"
	"time"
)

var statsManager *stats.StatsManager

// InitializeHandlers 初始化处理器
func InitializeHandlers(sm *stats.StatsManager) error {
	statsManager = sm
	return services.InitializeCSVService()
}

func HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	realIP := utils.GetRealIP(r)
	referer := r.Referer()

	// 修改这部分,获取完整的referer信息
	sourceInfo := "direct"
	if referer != "" {
		if parsedURL, err := url.Parse(referer); err == nil {
			// 包含主机名和路径
			sourceInfo = parsedURL.Host + parsedURL.Path
			// 如果有查询参数,也可以加上
			if parsedURL.RawQuery != "" {
				sourceInfo += "?" + parsedURL.RawQuery
			}
		}
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	pathSegments := strings.Split(path, "/")

	if len(pathSegments) < 2 {
		http.NotFound(w, r)
		return
	}

	prefix := pathSegments[0]
	suffix := pathSegments[1]

	services.Mu.RLock()
	csvPath, ok := services.CSVPathsCache[prefix][suffix]
	services.Mu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	selector, err := services.GetCSVContent(csvPath)
	if err != nil {
		http.Error(w, "Failed to fetch CSV content", http.StatusInternalServerError)
		log.Printf("Error fetching CSV content: %v", err)
		return
	}

	if len(selector.URLs) == 0 {
		http.Error(w, "No content available", http.StatusNotFound)
		return
	}

	randomURL := selector.GetRandomURL()

	// 记录统计
	endpoint := fmt.Sprintf("%s/%s", prefix, suffix)
	statsManager.IncrementCalls(endpoint)

	duration := time.Since(start)

	log.Printf(" %12s | %15s | %-6s | %-50s | %s | %-50s",
		duration,   // 持续时间
		realIP,     // 真实IP
		r.Method,   // HTTP方法
		r.URL.Path, // 请求路径
		sourceInfo, // 来源信息
		randomURL,  // 重定向URL
	)

	http.Redirect(w, r, randomURL, http.StatusFound)
}

func HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := statsManager.GetStats()
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Error encoding stats", http.StatusInternalServerError)
		log.Printf("Error encoding stats: %v", err)
	}
}
