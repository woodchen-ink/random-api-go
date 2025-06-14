package services

import (
	"log"
	"random-api-go/database"
	"random-api-go/models"
	"sync"
	"time"
)

// CacheManager 缓存管理器
type CacheManager struct {
	memoryCache map[string]*CachedEndpoint
	mutex       sync.RWMutex
}

// 注意：CachedEndpoint 类型定义在 endpoint_service.go 中

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	cm := &CacheManager{
		memoryCache: make(map[string]*CachedEndpoint),
	}

	// 启动定期清理过期缓存的协程
	go cm.cleanupExpiredCache()

	return cm
}

// GetFromMemoryCache 从内存缓存获取数据
func (cm *CacheManager) GetFromMemoryCache(key string) ([]string, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	cached, exists := cm.memoryCache[key]
	if !exists || len(cached.URLs) == 0 {
		return nil, false
	}

	return cached.URLs, true
}

// SetMemoryCache 设置内存缓存（duration参数保留以兼容现有接口，但不再使用）
func (cm *CacheManager) SetMemoryCache(key string, urls []string, duration time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.memoryCache[key] = &CachedEndpoint{
		URLs: urls,
	}
}

// InvalidateMemoryCache 清理指定key的内存缓存
func (cm *CacheManager) InvalidateMemoryCache(key string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.memoryCache, key)
}

// GetFromDBCache 从数据库缓存获取URL
func (cm *CacheManager) GetFromDBCache(dataSourceID uint) ([]string, error) {
	var cachedURLs []models.CachedURL
	if err := database.DB.Where("data_source_id = ? AND expires_at > ?", dataSourceID, time.Now()).
		Find(&cachedURLs).Error; err != nil {
		return nil, err
	}

	var urls []string
	for _, cached := range cachedURLs {
		urls = append(urls, cached.FinalURL)
	}

	return urls, nil
}

// SetDBCache 设置数据库缓存
func (cm *CacheManager) SetDBCache(dataSourceID uint, urls []string, duration time.Duration) error {
	// 先删除旧的缓存
	if err := database.DB.Where("data_source_id = ?", dataSourceID).Delete(&models.CachedURL{}).Error; err != nil {
		log.Printf("Failed to delete old cache for data source %d: %v", dataSourceID, err)
	}

	// 插入新的缓存
	expiresAt := time.Now().Add(duration)
	for _, url := range urls {
		cachedURL := models.CachedURL{
			DataSourceID: dataSourceID,
			OriginalURL:  url,
			FinalURL:     url,
			ExpiresAt:    expiresAt,
		}
		if err := database.DB.Create(&cachedURL).Error; err != nil {
			log.Printf("Failed to cache URL: %v", err)
		}
	}

	return nil
}

// UpdateDBCacheIfChanged 只有当数据变化时才更新数据库缓存，并返回是否需要清理内存缓存
func (cm *CacheManager) UpdateDBCacheIfChanged(dataSourceID uint, newURLs []string, duration time.Duration) (bool, error) {
	// 获取现有缓存
	existingURLs, err := cm.GetFromDBCache(dataSourceID)
	if err != nil {
		// 如果获取失败，直接设置新缓存
		return true, cm.SetDBCache(dataSourceID, newURLs, duration)
	}

	// 比较URL列表是否相同
	if cm.urlSlicesEqual(existingURLs, newURLs) {
		// 数据没有变化，只更新过期时间
		expiresAt := time.Now().Add(duration)
		if err := database.DB.Model(&models.CachedURL{}).
			Where("data_source_id = ?", dataSourceID).
			Update("expires_at", expiresAt).Error; err != nil {
			log.Printf("Failed to update cache expiry for data source %d: %v", dataSourceID, err)
		}
		return false, nil
	}

	// 数据有变化，更新缓存
	return true, cm.SetDBCache(dataSourceID, newURLs, duration)
}

// InvalidateMemoryCacheForDataSource 清理与数据源相关的内存缓存
func (cm *CacheManager) InvalidateMemoryCacheForDataSource(dataSourceID uint) error {
	// 获取数据源信息
	var dataSource models.DataSource
	if err := database.DB.Preload("Endpoint").First(&dataSource, dataSourceID).Error; err != nil {
		return err
	}

	// 清理该端点的内存缓存
	cm.InvalidateMemoryCache(dataSource.Endpoint.URL)
	log.Printf("已清理端点 %s 的内存缓存（数据源 %d 数据发生变化）", dataSource.Endpoint.URL, dataSourceID)

	return nil
}

// urlSlicesEqual 比较两个URL切片是否相等
func (cm *CacheManager) urlSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// 创建map来比较
	urlMap := make(map[string]bool)
	for _, url := range a {
		urlMap[url] = true
	}

	for _, url := range b {
		if !urlMap[url] {
			return false
		}
	}

	return true
}

// cleanupExpiredCache 定期清理过期的数据库缓存（内存缓存不再自动过期）
func (cm *CacheManager) cleanupExpiredCache() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		// 内存缓存不再自动过期，只清理数据库中的过期缓存
		if err := database.DB.Where("expires_at < ?", now).Delete(&models.CachedURL{}).Error; err != nil {
			log.Printf("Failed to cleanup expired cache: %v", err)
		}
	}
}
