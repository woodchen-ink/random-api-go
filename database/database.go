package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"random-api-go/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// 配置缓存
var (
	configCache      = make(map[string]*CachedConfig)
	configCacheMutex sync.RWMutex
)

// CachedConfig 缓存的配置项
type CachedConfig struct {
	Value    string
	CachedAt time.Time
	CacheTTL time.Duration // 缓存生存时间
}

// Initialize 初始化数据库
func Initialize(dataDir string) error {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "data.db")

	// 配置GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), config)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 获取底层的sql.DB来设置连接池参数
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(1) // SQLite建议单连接
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移数据库结构
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("Database initialized successfully at %s", dbPath)
	return nil
}

// autoMigrate 自动迁移数据库结构
func autoMigrate() error {
	return DB.AutoMigrate(
		&model.APIEndpoint{},
		&model.DataSource{},
		&model.URLReplaceRule{},
		&model.Config{},
	)
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetConfig 获取配置值（带缓存）
func GetConfig(key string, defaultValue string) string {
	// 先检查缓存
	configCacheMutex.RLock()
	cached, exists := configCache[key]
	configCacheMutex.RUnlock()

	// 如果缓存存在且未过期，直接返回
	if exists && time.Since(cached.CachedAt) < cached.CacheTTL {
		return cached.Value
	}

	// 从数据库查询
	var config model.Config
	if err := DB.Where("key = ?", key).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 缓存默认值（短时间缓存，避免频繁查询不存在的配置）
			cacheConfig(key, defaultValue, 5*time.Minute)
			return defaultValue
		}
		log.Printf("Failed to get config %s: %v", key, err)
		return defaultValue
	}

	// 缓存查询结果
	var cacheTTL time.Duration
	switch key {
	case "homepage_content":
		cacheTTL = 10 * time.Minute // 首页内容缓存10分钟
	case "oauth_client_id", "oauth_client_secret", "oauth_redirect_uri":
		cacheTTL = 30 * time.Minute // OAuth配置缓存30分钟
	default:
		cacheTTL = 5 * time.Minute // 其他配置缓存5分钟
	}

	cacheConfig(key, config.Value, cacheTTL)
	return config.Value
}

// cacheConfig 缓存配置值
func cacheConfig(key, value string, ttl time.Duration) {
	configCacheMutex.Lock()
	defer configCacheMutex.Unlock()

	configCache[key] = &CachedConfig{
		Value:    value,
		CachedAt: time.Now(),
		CacheTTL: ttl,
	}
}

// SetConfig 设置配置值
func SetConfig(key, value, configType string) error {
	var config model.Config
	err := DB.Where("key = ?", key).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		config = model.Config{
			Key:   key,
			Value: value,
			Type:  configType,
		}
		if err := DB.Create(&config).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// 更新现有配置
		config.Value = value
		config.Type = configType
		if err := DB.Save(&config).Error; err != nil {
			return err
		}
	}

	// 清理缓存
	invalidateConfigCache(key)
	return nil
}

// invalidateConfigCache 清理指定配置的缓存
func invalidateConfigCache(key string) {
	configCacheMutex.Lock()
	defer configCacheMutex.Unlock()
	delete(configCache, key)
	log.Printf("已清理配置 %s 的缓存", key)
}

// ListConfigs 列出所有配置
func ListConfigs() ([]model.Config, error) {
	var configs []model.Config
	err := DB.Order("key").Find(&configs).Error
	return configs, err
}

// DeleteConfig 删除配置
func DeleteConfig(key string) error {
	err := DB.Where("key = ?", key).Delete(&model.Config{}).Error
	if err != nil {
		return err
	}

	// 清理缓存
	invalidateConfigCache(key)
	return nil
}

// GetConfigCacheStats 获取配置缓存统计信息
func GetConfigCacheStats() map[string]interface{} {
	configCacheMutex.RLock()
	defer configCacheMutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_cached"] = len(configCache)

	// 统计各种状态的缓存
	var validCount, expiredCount int
	cacheDetails := make(map[string]interface{})

	for key, cached := range configCache {
		isExpired := time.Since(cached.CachedAt) >= cached.CacheTTL
		if isExpired {
			expiredCount++
		} else {
			validCount++
		}

		cacheDetails[key] = map[string]interface{}{
			"cached_at":    cached.CachedAt.Format("2006-01-02 15:04:05"),
			"ttl_seconds":  int(cached.CacheTTL.Seconds()),
			"expired":      isExpired,
			"value_length": len(cached.Value),
		}
	}

	stats["valid_count"] = validCount
	stats["expired_count"] = expiredCount
	stats["details"] = cacheDetails

	return stats
}
