package service

import (
	"fmt"
	"log"
	"sync"
)

// CacheManager 缓存管理器
type CacheManager struct {
	memoryCache map[string]*CachedItem
	mutex       sync.RWMutex
}

// CachedItem 缓存项（永久缓存，只在数据变动时刷新）
type CachedItem struct {
	URLs []string
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	cm := &CacheManager{
		memoryCache: make(map[string]*CachedItem),
	}

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

// SetMemoryCache 设置内存缓存
func (cm *CacheManager) SetMemoryCache(key string, urls []string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.memoryCache[key] = &CachedItem{
		URLs: urls,
	}
}

// InvalidateMemoryCache 清理指定key的内存缓存
func (cm *CacheManager) InvalidateMemoryCache(key string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.memoryCache, key)
}

// InvalidateMemoryCacheForDataSource 清理与数据源相关的内存缓存
func (cm *CacheManager) InvalidateMemoryCacheForDataSource(dataSourceID uint) {
	cacheKey := fmt.Sprintf("datasource_%d", dataSourceID)
	cm.InvalidateMemoryCache(cacheKey)
	log.Printf("已清理数据源 %d 的内存缓存", dataSourceID)
}

// InvalidateMemoryCacheForEndpoint 清理与端点相关的内存缓存
func (cm *CacheManager) InvalidateMemoryCacheForEndpoint(endpointURL string) {
	cm.InvalidateMemoryCache(endpointURL)
	log.Printf("已清理端点 %s 的内存缓存", endpointURL)
}

// GetCacheStats 获取缓存统计信息
func (cm *CacheManager) GetCacheStats() map[string]int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[string]int)
	for key, cached := range cm.memoryCache {
		stats[key] = len(cached.URLs)
	}

	return stats
}
