package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
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
	// 初始化随机数生成器
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)

	// 配置日志
	setupLogging()

	// 加载初始的 CSV 路径配置
	if err := loadCSVPaths(); err != nil {
		log.Fatal("Failed to load CSV paths:", err)
	}

	// 设置路由
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/pic/", logRequest(handleDynamicRequest))
	http.HandleFunc("/video/", logRequest(handleDynamicRequest))

	log.Printf("Listening on %s...\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func setupLogging() {
	logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func getRealIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	ip = r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func logRequest(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler(w, r)
		duration := time.Since(start)

		realIP := getRealIP(r)
		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "" {
			proto = "http"
		}
		host := r.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = r.Host
		}

		log.Printf("Request: %s %s://%s%s from %s - Duration: %v\n",
			r.Method, proto, host, r.URL.Path, realIP, duration)
	}
}

func loadCSVPaths() error {
	jsonPath := filepath.Join("public", "url.json")
	log.Printf("Attempting to read file: %s", jsonPath)

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

	fullPath := filepath.Join("public", path)
	log.Printf("Attempting to read file: %s", fullPath)

	fileContent, err := ioutil.ReadFile(fullPath)
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
	realIP := getRealIP(r)
	log.Printf("Handling request from IP: %s\n", realIP)

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

	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	redirectURL := fmt.Sprintf("%s://%s%s", proto, host, randomURL)

	log.Printf("Redirecting to %s\n", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
