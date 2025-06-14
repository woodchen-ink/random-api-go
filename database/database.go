package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"random-api-go/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

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
		&model.CachedURL{},
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

// CleanExpiredCache 清理过期缓存
func CleanExpiredCache() error {
	return DB.Where("expires_at < ?", time.Now()).Delete(&model.CachedURL{}).Error
}

// GetConfig 获取配置值
func GetConfig(key string, defaultValue string) string {
	var config model.Config
	if err := DB.Where("key = ?", key).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaultValue
		}
		log.Printf("Failed to get config %s: %v", key, err)
		return defaultValue
	}
	return config.Value
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
		return DB.Create(&config).Error
	} else if err != nil {
		return err
	}

	// 更新现有配置
	config.Value = value
	config.Type = configType
	return DB.Save(&config).Error
}

// ListConfigs 列出所有配置
func ListConfigs() ([]model.Config, error) {
	var configs []model.Config
	err := DB.Order("key").Find(&configs).Error
	return configs, err
}

// DeleteConfig 删除配置
func DeleteConfig(key string) error {
	return DB.Where("key = ?", key).Delete(&model.Config{}).Error
}
