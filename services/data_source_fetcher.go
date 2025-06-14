package services

import (
	"encoding/json"
	"fmt"
	"log"
	"random-api-go/models"
	"strings"
	"time"
)

// DataSourceFetcher 数据源获取器
type DataSourceFetcher struct {
	cacheManager   *CacheManager
	lankongFetcher *LankongFetcher
	apiFetcher     *APIFetcher
}

// NewDataSourceFetcher 创建数据源获取器
func NewDataSourceFetcher(cacheManager *CacheManager) *DataSourceFetcher {
	return &DataSourceFetcher{
		cacheManager:   cacheManager,
		lankongFetcher: NewLankongFetcher(),
		apiFetcher:     NewAPIFetcher(),
	}
}

// FetchURLs 从数据源获取URL列表
func (dsf *DataSourceFetcher) FetchURLs(dataSource *models.DataSource) ([]string, error) {
	// API类型的数据源直接实时请求，不使用缓存
	if dataSource.Type == "api_get" || dataSource.Type == "api_post" {
		log.Printf("实时请求API数据源 (类型: %s, ID: %d)", dataSource.Type, dataSource.ID)
		return dsf.fetchAPIURLs(dataSource)
	}

	// 其他类型的数据源先检查数据库缓存
	if cachedURLs, err := dsf.cacheManager.GetFromDBCache(dataSource.ID); err == nil && len(cachedURLs) > 0 {
		log.Printf("从数据库缓存获取到 %d 个URL (数据源ID: %d)", len(cachedURLs), dataSource.ID)
		return cachedURLs, nil
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

	// 缓存结果到数据库
	cacheDuration := time.Duration(dataSource.CacheDuration) * time.Second
	changed, err := dsf.cacheManager.UpdateDBCacheIfChanged(dataSource.ID, urls, cacheDuration)
	if err != nil {
		log.Printf("Failed to cache URLs for data source %d: %v", dataSource.ID, err)
	} else if changed {
		log.Printf("数据源 %d 的数据已更新，缓存了 %d 个URL", dataSource.ID, len(urls))
		// 数据发生变化，清理相关的内存缓存
		if err := dsf.cacheManager.InvalidateMemoryCacheForDataSource(dataSource.ID); err != nil {
			log.Printf("Failed to invalidate memory cache for data source %d: %v", dataSource.ID, err)
		}
	} else {
		log.Printf("数据源 %d 的数据未变化，仅更新了过期时间", dataSource.ID)
	}

	// 更新最后同步时间
	now := time.Now()
	dataSource.LastSync = &now
	if err := dsf.updateDataSourceSyncTime(dataSource); err != nil {
		log.Printf("Failed to update sync time for data source %d: %v", dataSource.ID, err)
	}

	return urls, nil
}

// fetchLankongURLs 获取兰空图床URL
func (dsf *DataSourceFetcher) fetchLankongURLs(dataSource *models.DataSource) ([]string, error) {
	var config models.LankongConfig
	if err := json.Unmarshal([]byte(dataSource.Config), &config); err != nil {
		return nil, fmt.Errorf("invalid lankong config: %w", err)
	}

	return dsf.lankongFetcher.FetchURLs(&config)
}

// fetchManualURLs 获取手动配置的URL
func (dsf *DataSourceFetcher) fetchManualURLs(dataSource *models.DataSource) ([]string, error) {
	// 手动配置可能是JSON格式或者纯文本格式
	config := strings.TrimSpace(dataSource.Config)

	// 尝试解析为JSON格式
	var manualConfig models.ManualConfig
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
func (dsf *DataSourceFetcher) fetchAPIURLs(dataSource *models.DataSource) ([]string, error) {
	var config models.APIConfig
	if err := json.Unmarshal([]byte(dataSource.Config), &config); err != nil {
		return nil, fmt.Errorf("invalid API config: %w", err)
	}

	// 对于API类型的数据源，直接进行实时请求，不使用预存储的数据
	return dsf.apiFetcher.FetchSingleURL(&config)
}

// fetchEndpointURLs 获取端点URL (直接返回端点URL列表)
func (dsf *DataSourceFetcher) fetchEndpointURLs(dataSource *models.DataSource) ([]string, error) {
	var config models.EndpointConfig
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

// updateDataSourceSyncTime 更新数据源的同步时间
func (dsf *DataSourceFetcher) updateDataSourceSyncTime(dataSource *models.DataSource) error {
	// 这里需要导入database包来更新数据库
	// 为了避免循环依赖，我们通过回调或者接口来处理
	// 暂时先记录日志，具体实现在主服务中处理
	log.Printf("需要更新数据源 %d 的同步时间", dataSource.ID)
	return nil
}

// PreloadDataSource 预加载数据源（在保存时调用）
func (dsf *DataSourceFetcher) PreloadDataSource(dataSource *models.DataSource) error {
	log.Printf("开始预加载数据源 (类型: %s, ID: %d)", dataSource.Type, dataSource.ID)

	_, err := dsf.FetchURLs(dataSource)
	if err != nil {
		return fmt.Errorf("failed to preload data source %d: %w", dataSource.ID, err)
	}

	log.Printf("数据源 %d 预加载完成", dataSource.ID)
	return nil
}
