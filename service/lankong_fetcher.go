package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"random-api-go/model"
	"time"
)

// LankongFetcher 兰空图床获取器
type LankongFetcher struct {
	client      *http.Client
	retryConfig *RetryConfig
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries int           // 最大重试次数
	BaseDelay  time.Duration // 基础延迟
}

// NewLankongFetcher 创建兰空图床获取器
func NewLankongFetcher() *LankongFetcher {
	return &LankongFetcher{
		client: &http.Client{
			Timeout: 60 * time.Second, // 增加超时时间
		},
		retryConfig: &RetryConfig{
			MaxRetries: 7,               // 最多重试7次 (0秒/15秒/15秒/30秒/30秒/60秒/60秒/180秒)
			BaseDelay:  1 * time.Second, // 基础延迟（实际不使用，使用固定延迟序列）
		},
	}
}

// NewLankongFetcherWithConfig 创建带自定义配置的兰空图床获取器
func NewLankongFetcherWithConfig(maxRetries int) *LankongFetcher {
	return &LankongFetcher{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		retryConfig: &RetryConfig{
			MaxRetries: maxRetries,
			BaseDelay:  1 * time.Second,
		},
	}
}

// LankongResponse 兰空图床API响应
type LankongResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
		Data        []struct {
			Links struct {
				URL string `json:"url"`
			} `json:"links"`
		} `json:"data"`
	} `json:"data"`
}

// FetchURLs 从兰空图床获取URL列表
func (lf *LankongFetcher) FetchURLs(config *model.LankongConfig) ([]string, error) {
	var allURLs []string
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://img.czl.net/api/v1/images"
	}

	for _, albumID := range config.AlbumIDs {
		log.Printf("开始获取相册 %s 的图片", albumID)

		// 获取第一页以确定总页数
		firstPageURL := fmt.Sprintf("%s?album_id=%s&page=1", baseURL, albumID)
		response, err := lf.fetchPageWithRetry(firstPageURL, config.APIToken)
		if err != nil {
			log.Printf("Failed to fetch first page for album %s: %v", albumID, err)
			continue
		}

		totalPages := response.Data.LastPage
		log.Printf("相册 %s 共有 %d 页", albumID, totalPages)

		// 处理所有页面
		albumURLs := []string{}
		for page := 1; page <= totalPages; page++ {
			reqURL := fmt.Sprintf("%s?album_id=%s&page=%d", baseURL, albumID, page)
			pageResponse, err := lf.fetchPageWithRetry(reqURL, config.APIToken)
			if err != nil {
				log.Printf("Failed to fetch page %d for album %s: %v", page, albumID, err)
				continue
			}

			for _, item := range pageResponse.Data.Data {
				if item.Links.URL != "" {
					albumURLs = append(albumURLs, item.Links.URL)
				}
			}

			// 进度日志
			if page%10 == 0 || page == totalPages {
				log.Printf("相册 %s: 已处理 %d/%d 页，收集到 %d 个URL", albumID, page, totalPages, len(albumURLs))
			}
		}

		allURLs = append(allURLs, albumURLs...)
		log.Printf("完成相册 %s: 收集到 %d 个URL", albumID, len(albumURLs))
	}

	return allURLs, nil
}

// fetchPageWithRetry 带重试的页面获取
func (lf *LankongFetcher) fetchPageWithRetry(url string, apiToken string) (*LankongResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= lf.retryConfig.MaxRetries; attempt++ {
		response, err := lf.fetchPage(url, apiToken)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// 如果是最后一次尝试，不再重试
		if attempt == lf.retryConfig.MaxRetries {
			break
		}

		// 计算延迟时间
		var delay time.Duration
		if isRateLimitError(err) {
			// 对于429错误，使用固定的延迟序列
			delay = getRateLimitDelay(attempt)
			log.Printf("遇到频率限制 (尝试 %d/%d): %v，等待 %v 后重试", attempt+1, lf.retryConfig.MaxRetries+1, err, delay)
		} else {
			// 其他错误使用较短的延迟
			delay = time.Duration(attempt+1) * time.Second
			log.Printf("请求失败 (尝试 %d/%d): %v，%v 后重试", attempt+1, lf.retryConfig.MaxRetries+1, err, delay)
		}

		time.Sleep(delay)
	}

	return nil, fmt.Errorf("重试 %d 次后仍然失败: %v", lf.retryConfig.MaxRetries, lastErr)
}

// getRateLimitDelay 获取频率限制的延迟时间
// 延迟序列：0秒 / 15秒 / 15秒 / 30秒 / 30秒 / 60秒 / 60秒 / 180秒
func getRateLimitDelay(attempt int) time.Duration {
	delaySequence := []time.Duration{
		0 * time.Second,   // 第1次重试：立即
		15 * time.Second,  // 第2次重试：15秒后
		15 * time.Second,  // 第3次重试：15秒后
		30 * time.Second,  // 第4次重试：30秒后
		30 * time.Second,  // 第5次重试：30秒后
		60 * time.Second,  // 第6次重试：60秒后
		60 * time.Second,  // 第7次重试：60秒后
		180 * time.Second, // 第8次重试：180秒后
	}

	if attempt < len(delaySequence) {
		return delaySequence[attempt]
	}

	// 如果超出序列长度，使用最后一个值
	return delaySequence[len(delaySequence)-1]
}

// isRateLimitError 检查是否是频率限制错误
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return fmt.Sprintf("%v", err) == "rate limit exceeded (429), need to slow down requests" ||
		fmt.Sprintf("%v", errStr) == "API returned status code: 429"
}

// fetchPage 获取兰空图床单页数据
func (lf *LankongFetcher) fetchPage(url string, apiToken string) (*LankongResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := lf.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 特殊处理429错误
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit exceeded (429), need to slow down requests")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lankongResp LankongResponse
	if err := json.Unmarshal(body, &lankongResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !lankongResp.Status {
		return nil, fmt.Errorf("API error: %s", lankongResp.Message)
	}

	return &lankongResp, nil
}
