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

	for {
		// 构建请求URL
		reqURL := fmt.Sprintf("%s?album_id=%s&page=%d", BaseURL, albumID, page)

		// 创建请求
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			panic(fmt.Sprintf("Failed to create request: %v", err))
		}

		// 设置请求头
		req.Header.Set("Authorization", apiToken)
		req.Header.Set("Accept", "application/json")

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			panic(fmt.Sprintf("Failed to fetch page %d: %v", page, err))
		}

		// 读取响应
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			panic(fmt.Sprintf("Failed to read response body: %v", err))
		}

		// 解析响应
		var response Response
		if err := json.Unmarshal(body, &response); err != nil {
			panic(fmt.Sprintf("Failed to parse response: %v", err))
		}

		// 提取URLs
		for _, item := range response.Data.Data {
			if item.Links.URL != "" {
				allURLs = append(allURLs, item.Links.URL)
			}
		}

		// 检查是否还有下一页
		if page >= response.Data.LastPage {
			break
		}
		page++

		fmt.Printf("Fetched page %d of %d for album %s\n", page-1, response.Data.LastPage, albumID)
	}

	return allURLs
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
