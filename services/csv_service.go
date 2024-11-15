package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"random-api-go/config"
	"random-api-go/models"
	"random-api-go/utils"
	"strings"
	"sync"
)

var (
	CSVPathsCache map[string]map[string]string
	csvCache      = make(map[string]*models.URLSelector)
	Mu            sync.RWMutex
)

// InitializeCSVService 初始化CSV服务
func InitializeCSVService() error {
	// 加载url.json
	if err := LoadCSVPaths(); err != nil {
		return fmt.Errorf("failed to load CSV paths: %v", err)
	}

	// 获取一个CSVPathsCache的副本，避免长时间持有锁
	Mu.RLock()
	pathsCopy := make(map[string]map[string]string)
	for prefix, suffixMap := range CSVPathsCache {
		pathsCopy[prefix] = make(map[string]string)
		for suffix, path := range suffixMap {
			pathsCopy[prefix][suffix] = path
		}
	}
	Mu.RUnlock()

	// 使用副本进行初始化
	for prefix, suffixMap := range pathsCopy {
		for suffix, csvPath := range suffixMap {
			selector, err := GetCSVContent(csvPath)
			if err != nil {
				log.Printf("Warning: Failed to load CSV content for %s/%s: %v", prefix, suffix, err)
				continue
			}

			// 更新URL计数
			endpoint := fmt.Sprintf("%s/%s", prefix, suffix)
			UpdateURLCount(endpoint, csvPath, len(selector.URLs))

			log.Printf("Loaded %d URLs for endpoint: %s/%s", len(selector.URLs), prefix, suffix)
		}
	}

	return nil
}

func LoadCSVPaths() error {
	var data []byte
	var err error

	// 获取环境变量中的基础URL
	baseURL := os.Getenv(config.EnvBaseURL)

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
			Timeout: config.RequestTimeout,
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

	Mu.Lock()
	CSVPathsCache = result
	Mu.Unlock()

	log.Println("CSV paths loaded from url.json")
	return nil
}

func GetCSVContent(path string) (*models.URLSelector, error) {
	log.Printf("开始获取CSV内容: %s", path)

	Mu.RLock()
	selector, exists := csvCache[path]
	Mu.RUnlock()

	if exists {
		log.Printf("从缓存中获取到CSV内容: %s", path)
		return selector, nil
	}

	var fileContent []byte
	var err error

	baseURL := os.Getenv(config.EnvBaseURL)

	if baseURL != "" {
		var fullURL string
		if strings.HasPrefix(baseURL, "http://") || strings.HasPrefix(baseURL, "https://") {
			fullURL = utils.JoinURLPath(baseURL, path)
		} else {
			fullURL = "https://" + utils.JoinURLPath(baseURL, path)
		}

		log.Printf("尝试从URL获取: %s", fullURL)

		client := &http.Client{
			Timeout: config.RequestTimeout,
		}

		resp, err := client.Get(fullURL)
		if err != nil {
			log.Printf("HTTP请求失败: %v", err)
			return nil, fmt.Errorf("HTTP请求失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("HTTP请求返回非200状态码: %d", resp.StatusCode)
			return nil, fmt.Errorf("HTTP请求返回非200状态码: %d", resp.StatusCode)
		}

		fileContent, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("读取响应内容失败: %v", err)
			return nil, fmt.Errorf("读取响应内容失败: %w", err)
		}

		log.Printf("成功读取到CSV内容，长度: %d bytes", len(fileContent))
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
	log.Printf("CSV文件包含 %d 行", len(lines))

	uniqueURLs := make(map[string]bool)
	var fileArray []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !uniqueURLs[trimmed] {
			fileArray = append(fileArray, trimmed)
			uniqueURLs[trimmed] = true
		}
	}

	log.Printf("处理后得到 %d 个唯一URL", len(fileArray))

	selector = models.NewURLSelector(fileArray)

	Mu.Lock()
	csvCache[path] = selector
	Mu.Unlock()

	log.Printf("CSV内容已缓存: %s", path)
	return selector, nil
}
