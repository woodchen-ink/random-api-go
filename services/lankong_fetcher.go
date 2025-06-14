package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"random-api-go/models"
	"time"
)

// LankongFetcher 兰空图床获取器
type LankongFetcher struct {
	client *http.Client
}

// NewLankongFetcher 创建兰空图床获取器
func NewLankongFetcher() *LankongFetcher {
	return &LankongFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
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
func (lf *LankongFetcher) FetchURLs(config *models.LankongConfig) ([]string, error) {
	var allURLs []string
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://img.czl.net/api/v1/images"
	}

	for _, albumID := range config.AlbumIDs {
		log.Printf("开始获取相册 %s 的图片", albumID)

		// 获取第一页以确定总页数
		firstPageURL := fmt.Sprintf("%s?album_id=%s&page=1", baseURL, albumID)
		response, err := lf.fetchPage(firstPageURL, config.APIToken)
		if err != nil {
			log.Printf("Failed to fetch first page for album %s: %v", albumID, err)
			continue
		}

		totalPages := response.Data.LastPage
		log.Printf("相册 %s 共有 %d 页", albumID, totalPages)

		// 处理所有页面
		for page := 1; page <= totalPages; page++ {
			reqURL := fmt.Sprintf("%s?album_id=%s&page=%d", baseURL, albumID, page)
			pageResponse, err := lf.fetchPage(reqURL, config.APIToken)
			if err != nil {
				log.Printf("Failed to fetch page %d for album %s: %v", page, albumID, err)
				continue
			}

			for _, item := range pageResponse.Data.Data {
				if item.Links.URL != "" {
					allURLs = append(allURLs, item.Links.URL)
				}
			}

			// 添加小延迟避免请求过快
			if page < totalPages {
				time.Sleep(100 * time.Millisecond)
			}
		}

		log.Printf("完成相册 %s: 收集到 %d 个URL", albumID, len(allURLs))
	}

	return allURLs, nil
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
