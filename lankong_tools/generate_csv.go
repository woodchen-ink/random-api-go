package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	BaseURL = "https://img.czl.net/api/v1/images"
)

// API响应结构体
type Response struct {
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

// 相册映射结构体
type AlbumMapping map[string]string

func main() {
	// 读取API Token
	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		panic("API_TOKEN environment variable is required")
	}

	// 读取相册映射配置
	mappingFile, err := os.ReadFile("lankong_tools/album_mapping.json")
	if err != nil {
		panic(fmt.Sprintf("Failed to read album mapping: %v", err))
	}

	var albumMapping AlbumMapping
	if err := json.Unmarshal(mappingFile, &albumMapping); err != nil {
		panic(fmt.Sprintf("Failed to parse album mapping: %v", err))
	}

	// 创建输出目录
	if err := os.MkdirAll("public", 0755); err != nil {
		panic(fmt.Sprintf("Failed to create output directory: %v", err))
	}

	// 处理每个相册
	for albumID, csvPath := range albumMapping {
		fmt.Printf("Processing album %s -> %s\n", albumID, csvPath)
		urls := fetchAllURLs(albumID, apiToken)

		// 确保目录存在
		dir := filepath.Dir(filepath.Join("public", csvPath))
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create directory for %s: %v", csvPath, err))
		}

		// 写入CSV文件
		if err := writeURLsToFile(urls, filepath.Join("public", csvPath)); err != nil {
			panic(fmt.Sprintf("Failed to write URLs to file %s: %v", csvPath, err))
		}
	}

	fmt.Println("All CSV files generated successfully!")
}

func fetchAllURLs(albumID string, apiToken string) []string {
	var allURLs []string
	page := 1

	client := &http.Client{}

	// 先发送一个请求获取总页数
	firstPageURL := fmt.Sprintf("%s?album_id=%s&page=1", BaseURL, albumID)
	response, err := fetchPage(firstPageURL, apiToken, client)
	if err != nil {
		panic(fmt.Sprintf("Failed to fetch first page: %v", err))
	}

	totalPages := response.Data.LastPage
	fmt.Printf("Album %s has %d pages in total\n", albumID, totalPages)

	for page <= totalPages {
		// 构建请求URL
		reqURL := fmt.Sprintf("%s?album_id=%s&page=%d", BaseURL, albumID, page)

		response, err := fetchPage(reqURL, apiToken, client)
		if err != nil {
			panic(fmt.Sprintf("Failed to fetch page %d: %v", page, err))
		}

		// 提取URLs
		for _, item := range response.Data.Data {
			if item.Links.URL != "" {
				allURLs = append(allURLs, item.Links.URL)
			}
		}

		fmt.Printf("Fetched page %d of %d for album %s (got %d URLs)\n",
			page, totalPages, albumID, len(response.Data.Data))

		page++
	}

	fmt.Printf("Finished album %s: collected %d URLs in total\n", albumID, len(allURLs))
	return allURLs
}

// 添加一个辅助函数来处理单个页面的请求
func fetchPage(url string, apiToken string, client *http.Client) (*Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

func writeURLsToFile(urls []string, filepath string) error {
	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入URLs
	for _, url := range urls {
		if _, err := file.WriteString(url + "\n"); err != nil {
			return err
		}
	}

	return nil
}
