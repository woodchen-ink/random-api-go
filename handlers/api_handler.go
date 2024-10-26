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

	var sourceDomain string
	if referer != "" {
		if parsedURL, err := url.Parse(referer); err == nil {
			sourceDomain = parsedURL.Hostname()
		}
	}
	if sourceDomain == "" {
		sourceDomain = "direct"
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
	log.Printf("请求：%s %s，来自 %s -来源：%s -持续时间: %v - 重定向至: %s",
		r.Method, r.URL.Path, realIP, sourceDomain, duration, randomURL)

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
