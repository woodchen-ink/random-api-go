package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	port           = ":5003"
	cacheDuration  = 24 * time.Hour
	requestTimeout = 10 * time.Second
)

var (
	csvPathsCache map[string]map[string]string
	lastFetchTime time.Time
	csvCache      = make(map[string][]string)
	mu            sync.RWMutex
	rng           *rand.Rand
)

func main() {
	// 使用当前时间作为种子初始化随机数生成器
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)

	// 配置日志
	setupLogging()

	// 加载初始的 CSV 路径配置
	if err := loadCSVPaths(); err != nil {
		log.Fatal("Failed to load CSV paths:", err)
	}

	// 提供静态文件
	http.Handle("/", http.FileServer(http.Dir("./public")))

	// 动态请求处理
	http.HandleFunc("/pic/", logRequest(handleDynamicRequest))
	http.HandleFunc("/video/", logRequest(handleDynamicRequest))

	log.Printf("Listening on %s...\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func setupLogging() {
	// 同时输出到标准输出和文件
	logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// 中间件：记录每个请求
func logRequest(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler(w, r)
		duration := time.Since(start)
		log.Printf("Request: %s %s from %s - Duration: %v\n", r.Method, r.URL.Path, r.RemoteAddr, duration)
	}
}

// 加载 CSV 路径配置
func loadCSVPaths() error {
	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
	}
	log.Printf("Current working directory: %s", currentDir)

	// 构建 url.json 的完整路径
	jsonPath := filepath.Join(currentDir, "public", "url.json")
	log.Printf("Attempting to read file: %s", jsonPath)

	// 检查文件是否存在
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			return fmt.Errorf("url.json does not exist at %s", jsonPath)
	}

	data, err := ioutil.ReadFile(jsonPath)
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

func getCSVContent(path string) ([]string, error) {
	mu.RLock()
	content, exists := csvCache[path]
	mu.RUnlock()
	if exists {
			log.Printf("CSV content for %s found in cache\n", path)
			return content, nil
	}

	var fileContent []byte
	var err error

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			// 处理远程 URL
			client := &http.Client{Timeout: requestTimeout}
			resp, err := client.Get(path)
			if err != nil {
					return nil, fmt.Errorf("error fetching CSV content: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
					return nil, fmt.Errorf("failed to fetch CSV content: %s", resp.Status)
			}

			fileContent, err = ioutil.ReadAll(resp.Body)
	} else {
			// 处理本地文件
			fullPath := filepath.Join("public", path) // 注意这里的更改
			log.Printf("Attempting to read file: %s", fullPath)
			fileContent, err = ioutil.ReadFile(fullPath)
	}

	if err != nil {
			return nil, fmt.Errorf("error reading CSV content: %w", err)
	}

	lines := strings.Split(string(fileContent), "\n")
	var fileArray []string
	for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
					fileArray = append(fileArray, trimmed)
			}
	}

	mu.Lock()
	csvCache[path] = fileArray
	mu.Unlock()

	log.Printf("CSV content for %s fetched and cached\n", path)
	return fileArray, nil
}


func handleDynamicRequest(w http.ResponseWriter, r *http.Request) {
	if time.Since(lastFetchTime) > cacheDuration {
		if err := loadCSVPaths(); err != nil {
			http.Error(w, "Failed to load CSV paths", http.StatusInternalServerError)
			log.Println("Error loading CSV paths:", err)
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

	fileArray, err := getCSVContent(csvPath)
	if err != nil {
		http.Error(w, "Failed to fetch CSV content", http.StatusInternalServerError)
		log.Println("Error fetching CSV content:", err)
		return
	}

	randomURL := fileArray[rng.Intn(len(fileArray))]
	log.Printf("Redirecting to %s\n", randomURL)
	http.Redirect(w, r, randomURL, http.StatusFound)
}
