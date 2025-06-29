package service

import (
	"fmt"
	"math/rand"
	"random-api-go/database"
	"random-api-go/model"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EndpointService API端点服务
type EndpointService struct {
	cacheManager      *CacheManager
	dataSourceFetcher *DataSourceFetcher
	preloader         *Preloader
}

var endpointService *EndpointService
var once sync.Once

// 支持的数据源类型列表
var supportedDataSourceTypes = []string{
	"lankong",
	"manual",
	"api_get",
	"api_post",
	"endpoint",
	"s3",
}

// validateDataSourceType 验证数据源类型
func validateDataSourceType(dataSourceType string) error {
	for _, supportedType := range supportedDataSourceTypes {
		if dataSourceType == supportedType {
			return nil
		}
	}
	return fmt.Errorf("unsupported data source type: %s, supported types: %v", dataSourceType, supportedDataSourceTypes)
}

// GetEndpointService 获取端点服务单例
func GetEndpointService() *EndpointService {
	once.Do(func() {
		// 创建组件
		cacheManager := NewCacheManager()
		dataSourceFetcher := NewDataSourceFetcher(cacheManager)
		preloader := NewPreloader(dataSourceFetcher, cacheManager)

		endpointService = &EndpointService{
			cacheManager:      cacheManager,
			dataSourceFetcher: dataSourceFetcher,
			preloader:         preloader,
		}

		// 启动预加载器
		preloader.Start()
	})
	return endpointService
}

// CreateEndpoint 创建API端点
func (s *EndpointService) CreateEndpoint(endpoint *model.APIEndpoint) error {
	if err := database.DB.Create(endpoint).Error; err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	// 清理缓存
	s.cacheManager.InvalidateMemoryCache(endpoint.URL)

	// 预加载数据源
	s.preloader.PreloadEndpointOnSave(endpoint)

	return nil
}

// GetEndpoint 获取API端点
func (s *EndpointService) GetEndpoint(id uint) (*model.APIEndpoint, error) {
	var endpoint model.APIEndpoint
	if err := database.DB.Preload("DataSources").Preload("URLReplaceRules").First(&endpoint, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}
	return &endpoint, nil
}

// GetEndpointByURL 根据URL获取端点
func (s *EndpointService) GetEndpointByURL(url string) (*model.APIEndpoint, error) {
	var endpoint model.APIEndpoint
	if err := database.DB.Preload("DataSources").Preload("URLReplaceRules").
		Where("url = ? AND is_active = ?", url, true).First(&endpoint).Error; err != nil {
		return nil, fmt.Errorf("failed to get endpoint by URL: %w", err)
	}
	return &endpoint, nil
}

// ListEndpoints 列出所有端点
func (s *EndpointService) ListEndpoints() ([]*model.APIEndpoint, error) {
	var endpoints []*model.APIEndpoint
	if err := database.DB.Preload("DataSources").Preload("URLReplaceRules").
		Order("sort_order ASC, created_at DESC").Find(&endpoints).Error; err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}
	return endpoints, nil
}

// UpdateEndpoint 更新API端点
func (s *EndpointService) UpdateEndpoint(endpoint *model.APIEndpoint) error {
	// 只更新指定字段，避免覆盖 created_at 和 sort_order 等字段
	updates := map[string]interface{}{
		"name":             endpoint.Name,
		"url":              endpoint.URL,
		"description":      endpoint.Description,
		"is_active":        endpoint.IsActive,
		"show_on_homepage": endpoint.ShowOnHomepage,
		"updated_at":       time.Now(),
	}

	// 如果 sort_order 不为 0，则更新它（0 表示未设置）
	if endpoint.SortOrder != 0 {
		updates["sort_order"] = endpoint.SortOrder
	}

	if err := database.DB.Model(&model.APIEndpoint{}).Where("id = ?", endpoint.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	// 清理缓存
	s.cacheManager.InvalidateMemoryCache(endpoint.URL)

	// 预加载数据源
	s.preloader.PreloadEndpointOnSave(endpoint)

	return nil
}

// DeleteEndpoint 删除API端点
func (s *EndpointService) DeleteEndpoint(id uint) error {
	// 先获取URL用于清理缓存
	endpoint, err := s.GetEndpoint(id)
	if err != nil {
		return fmt.Errorf("failed to get endpoint for deletion: %w", err)
	}

	// 删除相关的数据源和URL替换规则
	if err := database.DB.Select("DataSources", "URLReplaceRules").Delete(&model.APIEndpoint{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete endpoint: %w", err)
	}

	// 清理缓存
	s.cacheManager.InvalidateMemoryCache(endpoint.URL)

	return nil
}

// GetRandomURL 获取随机URL
func (s *EndpointService) GetRandomURL(url string) (string, error) {
	// 获取端点信息
	endpoint, err := s.GetEndpointByURL(url)
	if err != nil {
		return "", fmt.Errorf("endpoint not found: %w", err)
	}

	// 检查是否包含API类型或端点类型的数据源
	hasRealtimeDataSource := false
	for _, dataSource := range endpoint.DataSources {
		if dataSource.IsActive && (dataSource.Type == "api_get" || dataSource.Type == "api_post" || dataSource.Type == "endpoint") {
			hasRealtimeDataSource = true
			break
		}
	}

	// 如果包含实时数据源，不使用内存缓存，直接实时获取
	if hasRealtimeDataSource {
		return s.getRandomURLRealtime(endpoint)
	}

	// 非实时数据源，使用缓存模式但也先选择数据源
	return s.getRandomURLWithCache(endpoint)
}

// getRandomURLRealtime 实时获取随机URL（用于包含API数据源的端点）
func (s *EndpointService) getRandomURLRealtime(endpoint *model.APIEndpoint) (string, error) {
	// 收集所有激活的数据源
	var activeDataSources []model.DataSource
	for _, dataSource := range endpoint.DataSources {
		if dataSource.IsActive {
			activeDataSources = append(activeDataSources, dataSource)
		}
	}

	if len(activeDataSources) == 0 {
		return "", fmt.Errorf("no active data sources for endpoint: %s", endpoint.URL)
	}

	// 先随机选择一个数据源
	selectedDataSource := activeDataSources[rand.Intn(len(activeDataSources))]

	// 只从选中的数据源获取URL
	urls, err := s.dataSourceFetcher.FetchURLs(&selectedDataSource)
	if err != nil {
		return "", fmt.Errorf("failed to get URLs from selected data source %d: %w", selectedDataSource.ID, err)
	}

	if len(urls) == 0 {
		return "", fmt.Errorf("no URLs available from selected data source %d", selectedDataSource.ID)
	}

	// 从选中数据源的URL中随机选择一个
	randomURL := urls[rand.Intn(len(urls))]

	// 如果是端点类型的URL，需要递归调用
	if strings.HasPrefix(randomURL, "endpoint://") {
		endpointIDStr := strings.TrimPrefix(randomURL, "endpoint://")
		endpointID, err := strconv.ParseUint(endpointIDStr, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid endpoint ID in URL: %s", randomURL)
		}

		// 获取目标端点信息
		targetEndpoint, err := s.GetEndpoint(uint(endpointID))
		if err != nil {
			return "", fmt.Errorf("target endpoint not found: %w", err)
		}

		// 递归调用获取目标端点的随机URL
		targetURL, err := s.GetRandomURL(targetEndpoint.URL)
		if err != nil {
			return "", err
		}

		// 对从目标端点获取的URL应用当前端点的替换规则
		return s.applyURLReplaceRules(targetURL, endpoint.URL), nil
	}

	return s.applyURLReplaceRules(randomURL, endpoint.URL), nil
}

// getRandomURLWithCache 使用缓存模式获取随机URL（先选择数据源）
func (s *EndpointService) getRandomURLWithCache(endpoint *model.APIEndpoint) (string, error) {
	// 收集所有激活的数据源
	var activeDataSources []model.DataSource
	for _, dataSource := range endpoint.DataSources {
		if dataSource.IsActive {
			activeDataSources = append(activeDataSources, dataSource)
		}
	}

	if len(activeDataSources) == 0 {
		return "", fmt.Errorf("no active data sources for endpoint: %s", endpoint.URL)
	}

	// 先随机选择一个数据源
	selectedDataSource := activeDataSources[rand.Intn(len(activeDataSources))]

	// 从选中的数据源获取URL（会使用缓存）
	urls, err := s.dataSourceFetcher.FetchURLs(&selectedDataSource)
	if err != nil {
		return "", fmt.Errorf("failed to get URLs from selected data source %d: %w", selectedDataSource.ID, err)
	}

	if len(urls) == 0 {
		return "", fmt.Errorf("no URLs available from selected data source %d", selectedDataSource.ID)
	}

	// 从选中数据源的URL中随机选择一个
	randomURL := urls[rand.Intn(len(urls))]

	// 如果是端点类型的URL，需要递归调用
	if strings.HasPrefix(randomURL, "endpoint://") {
		endpointIDStr := strings.TrimPrefix(randomURL, "endpoint://")
		endpointID, err := strconv.ParseUint(endpointIDStr, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid endpoint ID in URL: %s", randomURL)
		}

		// 获取目标端点信息
		targetEndpoint, err := s.GetEndpoint(uint(endpointID))
		if err != nil {
			return "", fmt.Errorf("target endpoint not found: %w", err)
		}

		// 递归调用获取目标端点的随机URL
		targetURL, err := s.GetRandomURL(targetEndpoint.URL)
		if err != nil {
			return "", err
		}

		// 对从目标端点获取的URL应用当前端点的替换规则
		return s.applyURLReplaceRules(targetURL, endpoint.URL), nil
	}

	return s.applyURLReplaceRules(randomURL, endpoint.URL), nil
}

// applyURLReplaceRules 应用URL替换规则
func (s *EndpointService) applyURLReplaceRules(url, endpointURL string) string {
	// 获取端点的替换规则
	endpoint, err := s.GetEndpointByURL(endpointURL)
	if err != nil {
		return url
	}

	result := url
	for _, rule := range endpoint.URLReplaceRules {
		if rule.IsActive {
			result = strings.ReplaceAll(result, rule.FromURL, rule.ToURL)
		}
	}

	return result
}

// CreateDataSource 创建数据源
func (s *EndpointService) CreateDataSource(dataSource *model.DataSource) error {
	// 验证数据源类型
	if err := validateDataSourceType(dataSource.Type); err != nil {
		return err
	}

	if err := database.DB.Create(dataSource).Error; err != nil {
		return fmt.Errorf("failed to create data source: %w", err)
	}

	// 获取关联的端点URL用于清理缓存
	if endpoint, err := s.GetEndpoint(dataSource.EndpointID); err == nil {
		s.cacheManager.InvalidateMemoryCache(endpoint.URL)
	}

	// 预加载数据源
	s.preloader.PreloadDataSourceOnSave(dataSource)

	return nil
}

// UpdateDataSource 更新数据源
func (s *EndpointService) UpdateDataSource(dataSource *model.DataSource) error {
	// 验证数据源类型
	if err := validateDataSourceType(dataSource.Type); err != nil {
		return err
	}

	if err := database.DB.Save(dataSource).Error; err != nil {
		return fmt.Errorf("failed to update data source: %w", err)
	}

	// 获取关联的端点URL用于清理缓存
	if endpoint, err := s.GetEndpoint(dataSource.EndpointID); err == nil {
		s.cacheManager.InvalidateMemoryCache(endpoint.URL)
	}

	// 预加载数据源
	s.preloader.PreloadDataSourceOnSave(dataSource)

	return nil
}

// DeleteDataSource 删除数据源
func (s *EndpointService) DeleteDataSource(id uint) error {
	// 先获取数据源信息
	var dataSource model.DataSource
	if err := database.DB.First(&dataSource, id).Error; err != nil {
		return fmt.Errorf("failed to get data source: %w", err)
	}

	// 删除数据源
	if err := database.DB.Delete(&dataSource).Error; err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}

	// 获取关联的端点URL用于清理缓存
	if endpoint, err := s.GetEndpoint(dataSource.EndpointID); err == nil {
		s.cacheManager.InvalidateMemoryCache(endpoint.URL)
	}

	return nil
}

// RefreshDataSource 手动刷新数据源
func (s *EndpointService) RefreshDataSource(dataSourceID uint) error {
	return s.preloader.RefreshDataSource(dataSourceID)
}

// RefreshEndpoint 手动刷新端点
func (s *EndpointService) RefreshEndpoint(endpointID uint) error {
	return s.preloader.RefreshEndpoint(endpointID)
}

// GetPreloader 获取预加载器（用于外部控制）
func (s *EndpointService) GetPreloader() *Preloader {
	return s.preloader
}

// GetCacheManager 获取缓存管理器（用于外部控制）
func (s *EndpointService) GetCacheManager() *CacheManager {
	return s.cacheManager
}

// GetDataSourceURLCount 获取数据源的URL数量
func (s *EndpointService) GetDataSourceURLCount(dataSource *model.DataSource) (int, error) {
	// 对于API类型和端点类型的数据源，返回1（因为每次都是实时请求）
	if dataSource.Type == "api_get" || dataSource.Type == "api_post" || dataSource.Type == "endpoint" {
		return 1, nil
	}

	// 优先从内存缓存获取URL数量，避免触发耗时的网络请求
	cacheKey := fmt.Sprintf("datasource_%d", dataSource.ID)
	if cachedURLs, exists := s.cacheManager.GetFromMemoryCache(cacheKey); exists {
		return len(cachedURLs), nil
	}

	// 如果缓存中没有数据，返回估算值，避免在统计时触发耗时操作
	switch dataSource.Type {
	case "manual":
		// 手动数据源可以快速解析配置获取数量
		urls, err := s.dataSourceFetcher.FetchURLs(dataSource)
		if err != nil {
			return 0, nil // 返回0而不是错误，避免影响整体统计
		}
		return len(urls), nil
	case "lankong":
		// 兰空图床如果没有缓存，返回估算值
		return 100, nil // 估算值
	default:
		return 0, nil
	}
}
