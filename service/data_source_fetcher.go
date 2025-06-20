package service

import (
	"encoding/json"
	"fmt"
	"log"
	"random-api-go/database"
	"random-api-go/model"
	"strconv"
	"strings"
	"time"
)

// DataSourceFetcher 数据源获取器
type DataSourceFetcher struct {
	cacheManager   *CacheManager
	lankongFetcher *LankongFetcher
	apiFetcher     *APIFetcher
	s3Fetcher      *S3Fetcher
}

// NewDataSourceFetcher 创建数据源获取器
func NewDataSourceFetcher(cacheManager *CacheManager) *DataSourceFetcher {
	// 从配置中获取兰空图床最大重试次数
	maxRetries := getIntConfig("lankong_max_retries", 7)

	var lankongFetcher *LankongFetcher
	if maxRetries > 0 {
		// 使用自定义配置
		lankongFetcher = NewLankongFetcherWithConfig(maxRetries)
		log.Printf("兰空图床获取器配置: 最大重试%d次", maxRetries)
	} else {
		// 使用默认配置
		lankongFetcher = NewLankongFetcher()
		log.Printf("兰空图床获取器使用默认配置")
	}

	return &DataSourceFetcher{
		cacheManager:   cacheManager,
		lankongFetcher: lankongFetcher,
		apiFetcher:     NewAPIFetcher(),
		s3Fetcher:      NewS3Fetcher(),
	}
}

// getIntConfig 获取整数配置，如果不存在或无效则返回默认值
func getIntConfig(key string, defaultValue int) int {
	configStr := database.GetConfig(key, "")
	if configStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(configStr)
	if err != nil {
		log.Printf("配置 %s 的值 '%s' 不是有效整数，使用默认值 %d", key, configStr, defaultValue)
		return defaultValue
	}

	return value
}

// FetchURLs 从数据源获取URL列表
func (dsf *DataSourceFetcher) FetchURLs(dataSource *model.DataSource) ([]string, error) {
	return dsf.FetchURLsWithOptions(dataSource, false)
}

// FetchURLsWithOptions 从数据源获取URL列表，支持跳过缓存选项
func (dsf *DataSourceFetcher) FetchURLsWithOptions(dataSource *model.DataSource, skipCache bool) ([]string, error) {
	// API类型的数据源直接实时请求，不使用缓存
	if dataSource.Type == "api_get" || dataSource.Type == "api_post" {
		return dsf.fetchAPIURLs(dataSource)
	}

	// 构建内存缓存的key（使用数据源ID）
	cacheKey := fmt.Sprintf("datasource_%d", dataSource.ID)

	// 如果不跳过缓存，先检查内存缓存
	if !skipCache {
		if cachedURLs, exists := dsf.cacheManager.GetFromMemoryCache(cacheKey); exists && len(cachedURLs) > 0 {
			return cachedURLs, nil
		}
	}

	var urls []string
	var err error

	log.Printf("开始从数据源获取URL (类型: %s, ID: %d)", dataSource.Type, dataSource.ID)

	switch dataSource.Type {
	case "lankong":
		urls, err = dsf.fetchLankongURLs(dataSource)
	case "manual":
		urls, err = dsf.fetchManualURLs(dataSource)
	case "endpoint":
		urls, err = dsf.fetchEndpointURLs(dataSource)
	case "s3":
		urls, err = dsf.fetchS3URLs(dataSource)
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", dataSource.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch URLs from %s data source: %w", dataSource.Type, err)
	}

	if len(urls) == 0 {
		log.Printf("警告: 数据源 %d 没有获取到任何URL", dataSource.ID)
		return urls, nil
	}

	// 缓存结果到内存
	dsf.cacheManager.SetMemoryCache(cacheKey, urls)
	log.Printf("数据源 %d 已缓存 %d 个URL到内存", dataSource.ID, len(urls))

	// 更新最后同步时间
	now := time.Now()
	dataSource.LastSync = &now
	if err := dsf.updateDataSourceSyncTime(dataSource); err != nil {
		log.Printf("Failed to update sync time for data source %d: %v", dataSource.ID, err)
	}

	return urls, nil
}

// fetchLankongURLs 获取兰空图床URL
func (dsf *DataSourceFetcher) fetchLankongURLs(dataSource *model.DataSource) ([]string, error) {
	var config model.LankongConfig
	if err := json.Unmarshal([]byte(dataSource.Config), &config); err != nil {
		return nil, fmt.Errorf("invalid lankong config: %w", err)
	}

	return dsf.lankongFetcher.FetchURLs(&config)
}

// fetchManualURLs 获取手动配置的URL
func (dsf *DataSourceFetcher) fetchManualURLs(dataSource *model.DataSource) ([]string, error) {
	// 手动配置可能是JSON格式或者纯文本格式
	config := strings.TrimSpace(dataSource.Config)

	// 尝试解析为JSON格式
	var manualConfig model.ManualConfig
	if err := json.Unmarshal([]byte(config), &manualConfig); err == nil {
		return manualConfig.URLs, nil
	}

	// 如果不是JSON，按行分割处理
	lines := strings.Split(config, "\n")
	var urls []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") { // 忽略空行和注释
			urls = append(urls, line)
		}
	}

	return urls, nil
}

// fetchAPIURLs 获取API接口URL (实时请求，不缓存)
func (dsf *DataSourceFetcher) fetchAPIURLs(dataSource *model.DataSource) ([]string, error) {
	var config model.APIConfig
	if err := json.Unmarshal([]byte(dataSource.Config), &config); err != nil {
		return nil, fmt.Errorf("invalid API config: %w", err)
	}

	// 对于API类型的数据源，直接进行实时请求，不使用预存储的数据
	return dsf.apiFetcher.FetchSingleURL(&config)
}

// fetchEndpointURLs 获取端点URL (直接返回端点URL列表)
func (dsf *DataSourceFetcher) fetchEndpointURLs(dataSource *model.DataSource) ([]string, error) {
	var config model.EndpointConfig
	if err := json.Unmarshal([]byte(dataSource.Config), &config); err != nil {
		return nil, fmt.Errorf("invalid endpoint config: %w", err)
	}

	if len(config.EndpointIDs) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}

	// 这里我们需要导入database包来查询端点信息
	// 为了避免循环依赖，我们返回一个特殊的URL格式，让服务层处理
	var urls []string
	for _, endpointID := range config.EndpointIDs {
		// 使用特殊格式标记这是一个端点引用
		urls = append(urls, fmt.Sprintf("endpoint://%d", endpointID))
	}

	return urls, nil
}

// fetchS3URLs 获取S3存储桶URL
func (dsf *DataSourceFetcher) fetchS3URLs(dataSource *model.DataSource) ([]string, error) {
	var config model.S3Config
	if err := json.Unmarshal([]byte(dataSource.Config), &config); err != nil {
		return nil, fmt.Errorf("invalid S3 config: %w", err)
	}

	return dsf.s3Fetcher.FetchURLs(&config)
}

// updateDataSourceSyncTime 更新数据源的同步时间
func (dsf *DataSourceFetcher) updateDataSourceSyncTime(dataSource *model.DataSource) error {
	if err := database.DB.Model(dataSource).Update("last_sync", dataSource.LastSync).Error; err != nil {
		return fmt.Errorf("failed to update sync time for data source %d: %w", dataSource.ID, err)
	}
	log.Printf("已更新数据源 %d 的同步时间", dataSource.ID)
	return nil
}

// PreloadDataSource 预加载数据源（在保存时调用）
func (dsf *DataSourceFetcher) PreloadDataSource(dataSource *model.DataSource) error {
	log.Printf("开始预加载数据源 (类型: %s, ID: %d)", dataSource.Type, dataSource.ID)

	_, err := dsf.FetchURLs(dataSource)
	if err != nil {
		return fmt.Errorf("failed to preload data source %d: %w", dataSource.ID, err)
	}

	log.Printf("数据源 %d 预加载完成", dataSource.ID)
	return nil
}

// RefreshDataSource 强制刷新数据源（跳过缓存）
func (dsf *DataSourceFetcher) RefreshDataSource(dataSource *model.DataSource) error {
	log.Printf("开始强制刷新数据源 (类型: %s, ID: %d)", dataSource.Type, dataSource.ID)

	_, err := dsf.FetchURLsWithOptions(dataSource, true) // 跳过缓存
	if err != nil {
		return fmt.Errorf("failed to refresh data source %d: %w", dataSource.ID, err)
	}

	log.Printf("数据源 %d 强制刷新完成", dataSource.ID)
	return nil
}
