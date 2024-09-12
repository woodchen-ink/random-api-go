package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)

	setupLogging()

	if err := loadCSVPaths(); err != nil {
		log.Fatal("Failed to load CSV paths:", err)
	}

	// 设置静态文件服务
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// 设置 API 路由
	http.HandleFunc("/pic/", handleAPIRequest)
	http.HandleFunc("/video/", handleAPIRequest)

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
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
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

	return fileArray, nil
}

func handleAPIRequest(w http.ResponseWriter, r *http.Request) {
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

	if len(fileArray) == 0 {
		http.Error(w, "No content available", http.StatusNotFound)
		return
	}

	randomURL := fileArray[rng.Intn(len(fileArray))]

	log.Printf("Redirecting to %s\n", randomURL)
	http.Redirect(w, r, randomURL, http.StatusFound)
}
