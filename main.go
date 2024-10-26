package main

import (
	"encoding/json"
	"fmt"
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
	cacheDuration  = 24 * time.Hour
	requestTimeout = 10 * time.Second
	noRepeatCount  = 3 // 在这个次数内不重复选择
)

var (
	csvPathsCache map[string]map[string]string
	lastFetchTime time.Time
	csvCache      = make(map[string]*URLSelector)
	mu            sync.RWMutex
	rng           *rand.Rand
	statsManager  *stats.StatsManager
)

type URLSelector struct {
	URLs         []string
	CurrentIndex int
	RecentUsed   map[string]int
	mu           sync.Mutex
}

func NewURLSelector(urls []string) *URLSelector {
	return &URLSelector{
		URLs:         urls,
		CurrentIndex: 0,
		RecentUsed:   make(map[string]int),
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

	if us.CurrentIndex == 0 {
		us.ShuffleURLs()
	}

	for i := 0; i < len(us.URLs); i++ {
		url := us.URLs[us.CurrentIndex]
		us.CurrentIndex = (us.CurrentIndex + 1) % len(us.URLs)

		if us.RecentUsed[url] < noRepeatCount {
			us.RecentUsed[url]++
			// 如果某个URL使用次数达到上限，从RecentUsed中移除
			if us.RecentUsed[url] == noRepeatCount {
				delete(us.RecentUsed, url)
			}
			return url
		}
	}

	// 如果所有URL都被最近使用过，重置RecentUsed并返回第一个URL
	us.RecentUsed = make(map[string]int)
	return us.URLs[0]
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
	jsonPath := filepath.Join("public", "url.json")
	log.Printf("Attempting to read file: %s", jsonPath)

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read url.json: %w", err)
	}

	var result map[string]map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to unmarshal url.json: %w", err)
	}

	mu.Lock()
	csvPathsCache = result
	lastFetchTime = time.Now()
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

	fullPath := filepath.Join("public", path)
	log.Printf("尝试读取文件: %s", fullPath)

	fileContent, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取 CSV 内容时出错: %w", err)
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

	// 获取来源域名
	var sourceDomain string
	if referer != "" {
		if parsedURL, err := url.Parse(referer); err == nil {
			sourceDomain = parsedURL.Hostname()
		}
	}
	if sourceDomain == "" {
		sourceDomain = "direct"
	}

	if time.Since(lastFetchTime) > cacheDuration {
		if err := loadCSVPaths(); err != nil {
			http.Error(w, "无法加载 CSV 路径", http.StatusInternalServerError)
			log.Printf("加载 CSV 路径时出错: %v", err)
			return
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

	mu.RLock()
	csvPath, ok := csvPathsCache[prefix][suffix]
	mu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	selector, err := getCSVContent(csvPath)
	if err != nil {
		http.Error(w, "无法获取 CSV 内容", http.StatusInternalServerError)
		log.Printf("获取 CSV 内容时出错: %v", err)
		return
	}

	if len(selector.URLs) == 0 {
		http.Error(w, "无可用内容", http.StatusNotFound)
		return
	}

	randomURL := selector.GetRandomURL()

	// 记录统计
	endpoint := fmt.Sprintf("%s/%s", prefix, suffix)
	statsManager.IncrementCalls(endpoint)

	duration := time.Since(start)
	log.Printf("请求：%s %s，来自 %s -来源：%s -持续时间：%v -重定向至：%s",
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
