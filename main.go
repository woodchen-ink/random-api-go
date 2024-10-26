package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"random-api-go/logging"
	"random-api-go/stats"
	"random-api-go/utils"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	port           = ":5003"
	requestTimeout = 10 * time.Second
	envBaseURL     = "BASE_URL"
)

var (
	csvPathsCache map[string]map[string]string
	csvCache      = make(map[string]*URLSelector)
	mu            sync.RWMutex
	rng           *rand.Rand
	statsManager  *stats.StatsManager
)

type URLSelector struct {
	URLs []string
	mu   sync.Mutex
}

func NewURLSelector(urls []string) *URLSelector {
	return &URLSelector{
		URLs: urls,
	}
}

func (us *URLSelector) ShuffleURLs() {
	for i := len(us.URLs) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		us.URLs[i], us.URLs[j] = us.URLs[j], us.URLs[i]
	}
}

func (us *URLSelector) GetRandomURL() string {
	us.mu.Lock()
	defer us.mu.Unlock()

	if len(us.URLs) == 0 {
		return ""
	}
	return us.URLs[rng.Intn(len(us.URLs))]
}

func init() {
	// 确保数据目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}
}

func main() {
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)

	logging.SetupLogging()
	statsManager = stats.NewStatsManager("data/stats.json")

	// 设置优雅关闭
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Server is shutting down...")

		// 关闭统计管理器,确保统计数据被保存
		statsManager.Shutdown()
		log.Println("Stats manager shutdown completed")

		os.Exit(0)
	}()

	if err := loadCSVPaths(); err != nil {
		log.Fatal("Failed to load CSV paths:", err)
	}

	// 设置静态文件服务
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// 设置 API 路由
	http.HandleFunc("/pic/", handleAPIRequest)
	http.HandleFunc("/video/", handleAPIRequest)
	http.HandleFunc("/stats", handleStats)

	log.Printf("Server starting on %s...\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func loadCSVPaths() error {
	var data []byte
	var err error

	// 获取环境变量中的基础URL
	baseURL := os.Getenv(envBaseURL)

	if baseURL != "" {
		// 构建完整的URL
		var fullURL string
		if strings.HasPrefix(baseURL, "http://") || strings.HasPrefix(baseURL, "https://") {
			fullURL = utils.JoinURLPath(baseURL, "url.json")
		} else {
			fullURL = "https://" + utils.JoinURLPath(baseURL, "url.json")
		}

		log.Printf("Attempting to read url.json from: %s", fullURL)

		// 创建HTTP客户端
		client := &http.Client{
			Timeout: requestTimeout,
		}

		resp, err := client.Get(fullURL)
		if err != nil {
			return fmt.Errorf("failed to fetch url.json: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch url.json, status code: %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read url.json response: %w", err)
		}
	} else {
		// 从本地文件读取
		jsonPath := filepath.Join("public", "url.json")
		log.Printf("Attempting to read local file: %s", jsonPath)

		data, err = os.ReadFile(jsonPath)
		if err != nil {
			return fmt.Errorf("failed to read local url.json: %w", err)
		}
	}

	var result map[string]map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to unmarshal url.json: %w", err)
	}

	mu.Lock()
	csvPathsCache = result
	mu.Unlock()

	log.Println("CSV paths loaded from url.json")
	return nil
}

func getCSVContent(path string) (*URLSelector, error) {
	mu.RLock()
	selector, exists := csvCache[path]
	mu.RUnlock()
	if exists {
		return selector, nil
	}

	var fileContent []byte
	var err error

	// 获取环境变量中的基础URL
	baseURL := os.Getenv(envBaseURL)

	if baseURL != "" {
		// 如果设置了基础URL，构建完整的URL
		var fullURL string
		if strings.HasPrefix(baseURL, "http://") || strings.HasPrefix(baseURL, "https://") {
			// 如果baseURL已经包含协议,直接使用
			fullURL = utils.JoinURLPath(baseURL, path)
		} else {
			// 如果没有协议,添加https://
			fullURL = "https://" + utils.JoinURLPath(baseURL, path)
		}

		log.Printf("尝试从URL获取: %s", fullURL)

		// 创建HTTP客户端
		client := &http.Client{
			Timeout: requestTimeout,
		}

		resp, err := client.Get(fullURL)
		if err != nil {
			return nil, fmt.Errorf("HTTP请求失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP请求返回非200状态码: %d", resp.StatusCode)
		}

		fileContent, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应内容失败: %w", err)
		}
	} else {
		// 如果没有设置基础URL，从本地文件读取
		fullPath := filepath.Join("public", path)
		log.Printf("尝试读取本地文件: %s", fullPath)

		fileContent, err = os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("读取CSV内容时出错: %w", err)
		}
	}

	lines := strings.Split(string(fileContent), "\n")
	uniqueURLs := make(map[string]bool)
	var fileArray []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !uniqueURLs[trimmed] {
			fileArray = append(fileArray, trimmed)
			uniqueURLs[trimmed] = true
		}
	}

	selector = NewURLSelector(fileArray)

	mu.Lock()
	csvCache[path] = selector
	mu.Unlock()

	return selector, nil
}

func handleAPIRequest(w http.ResponseWriter, r *http.Request) {
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

	mu.RLock()
	csvPath, ok := csvPathsCache[prefix][suffix]
	mu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	selector, err := getCSVContent(csvPath)
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

// 统计API处理函数
func handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := statsManager.GetStats()
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Error encoding stats", http.StatusInternalServerError)
		log.Printf("Error encoding stats: %v", err)
	}
}
