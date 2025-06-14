package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"random-api-go/models"
	"strings"
	"time"
)

// APIFetcher API接口获取器
type APIFetcher struct {
	client *http.Client
}

// NewAPIFetcher 创建API接口获取器
func NewAPIFetcher() *APIFetcher {
	return &APIFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchURLs 从API接口获取URL列表
func (af *APIFetcher) FetchURLs(config *models.APIConfig) ([]string, error) {
	var allURLs []string

	// 对于GET/POST接口，我们预获取多次以获得不同的URL
	maxFetches := 200
	if config.Method == "GET" {
		maxFetches = 100 // GET接口可能返回相同结果，减少请求次数
	}

	log.Printf("开始从 %s 接口预获取 %d 次URL", config.Method, maxFetches)

	urlSet := make(map[string]bool) // 用于去重

	for i := 0; i < maxFetches; i++ {
		urls, err := af.fetchSingleRequest(config)
		if err != nil {
			log.Printf("第 %d 次请求失败: %v", i+1, err)
			continue
		}

		// 添加到集合中（自动去重）
		for _, url := range urls {
			if url != "" && !urlSet[url] {
				urlSet[url] = true
				allURLs = append(allURLs, url)
			}
		}

		// 如果是GET接口且连续几次都没有新URL，提前结束
		if config.Method == "GET" && i > 10 && len(allURLs) > 0 {
			// 检查最近10次是否有新增URL
			if i%10 == 0 {
				currentCount := len(allURLs)
				// 如果URL数量没有显著增长，可能接口返回固定结果
				if currentCount < i/5 { // 如果平均每5次请求才有1个新URL，可能效率太低
					log.Printf("GET接口效率较低，在第 %d 次请求后停止预获取", i+1)
					break
				}
			}
		}

		// 添加小延迟避免请求过快
		if i < maxFetches-1 {
			time.Sleep(50 * time.Millisecond)
		}

		// 每50次请求输出一次进度
		if (i+1)%50 == 0 {
			log.Printf("已完成 %d/%d 次请求，获得 %d 个唯一URL", i+1, maxFetches, len(allURLs))
		}
	}

	log.Printf("完成API预获取: 总共获得 %d 个唯一URL", len(allURLs))
	return allURLs, nil
}

// FetchSingleURL 实时获取单个URL (用于GET/POST实时请求)
func (af *APIFetcher) FetchSingleURL(config *models.APIConfig) ([]string, error) {
	log.Printf("实时请求 %s 接口: %s", config.Method, config.URL)
	return af.fetchSingleRequest(config)
}

// fetchSingleRequest 执行单次API请求
func (af *APIFetcher) fetchSingleRequest(config *models.APIConfig) ([]string, error) {
	var req *http.Request
	var err error

	if config.Method == "POST" {
		var body io.Reader
		if config.Body != "" {
			body = strings.NewReader(config.Body)
		}
		req, err = http.NewRequest("POST", config.URL, body)
		if config.Body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
	} else {
		req, err = http.NewRequest("GET", config.URL, nil)
	}

	if err != nil {
		return nil, err
	}

	// 设置请求头
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := af.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return af.extractURLsFromJSON(data, config.URLField)
}

// extractURLsFromJSON 从JSON数据中提取URL
func (af *APIFetcher) extractURLsFromJSON(data interface{}, fieldPath string) ([]string, error) {
	var urls []string

	// 分割字段路径
	fields := strings.Split(fieldPath, ".")

	// 递归提取URL
	af.extractURLsRecursive(data, fields, 0, &urls)

	return urls, nil
}

// extractURLsRecursive 递归提取URL
func (af *APIFetcher) extractURLsRecursive(data interface{}, fields []string, depth int, urls *[]string) {
	if depth >= len(fields) {
		// 到达目标字段，提取URL
		if url, ok := data.(string); ok && url != "" {
			*urls = append(*urls, url)
		}
		return
	}

	currentField := fields[depth]

	switch v := data.(type) {
	case map[string]interface{}:
		if value, exists := v[currentField]; exists {
			af.extractURLsRecursive(value, fields, depth+1, urls)
		}
	case []interface{}:
		for _, item := range v {
			af.extractURLsRecursive(item, fields, depth, urls)
		}
	}
}
