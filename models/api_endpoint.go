package models

import (
	"time"

	"gorm.io/gorm"
)

// APIEndpoint API端点模型
type APIEndpoint struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Name           string         `json:"name" gorm:"uniqueIndex;not null"`
	URL            string         `json:"url" gorm:"uniqueIndex;not null"`
	Description    string         `json:"description"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	ShowOnHomepage bool           `json:"show_on_homepage" gorm:"default:true"`
	SortOrder      int            `json:"sort_order" gorm:"default:0;index"` // 排序字段，数值越小越靠前
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	DataSources     []DataSource     `json:"data_sources,omitempty" gorm:"foreignKey:EndpointID"`
	URLReplaceRules []URLReplaceRule `json:"url_replace_rules,omitempty" gorm:"foreignKey:EndpointID"`
}

// DataSource 数据源模型
type DataSource struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	EndpointID    uint           `json:"endpoint_id" gorm:"not null;index"`
	Name          string         `json:"name" gorm:"not null"`
	Type          string         `json:"type" gorm:"not null;check:type IN ('lankong', 'manual', 'api_get', 'api_post', 'endpoint')"`
	Config        string         `json:"config" gorm:"not null"`
	CacheDuration int            `json:"cache_duration" gorm:"default:3600"` // 缓存时长（秒）
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	LastSync      *time.Time     `json:"last_sync,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Endpoint   APIEndpoint `json:"-" gorm:"foreignKey:EndpointID"`
	CachedURLs []CachedURL `json:"-" gorm:"foreignKey:DataSourceID"`
}

// URLReplaceRule URL替换规则模型
type URLReplaceRule struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	EndpointID *uint          `json:"endpoint_id" gorm:"index"` // 可以为空，表示全局规则
	Name       string         `json:"name" gorm:"not null"`
	FromURL    string         `json:"from_url" gorm:"not null"`
	ToURL      string         `json:"to_url" gorm:"not null"`
	IsActive   bool           `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Endpoint *APIEndpoint `json:"endpoint,omitempty" gorm:"foreignKey:EndpointID"`
}

// CachedURL 缓存URL模型
type CachedURL struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	DataSourceID uint      `json:"data_source_id" gorm:"not null;index"`
	OriginalURL  string    `json:"original_url" gorm:"not null"`
	FinalURL     string    `json:"final_url" gorm:"not null"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`

	// 关联
	DataSource DataSource `json:"-" gorm:"foreignKey:DataSourceID"`
}

// Config 通用配置表
type Config struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"` // 配置键，如 "homepage_content"
	Value     string    `json:"value" gorm:"type:text"`          // 配置值
	Type      string    `json:"type" gorm:"default:'string'"`    // 配置类型：string, json, number, boolean
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DataSourceConfig 数据源配置结构体
type DataSourceConfig struct {
	// 兰空图床配置
	LankongConfig *LankongConfig `json:"lankong_config,omitempty"`

	// 手动数据配置
	ManualConfig *ManualConfig `json:"manual_config,omitempty"`

	// API配置
	APIConfig *APIConfig `json:"api_config,omitempty"`

	// 端点配置
	EndpointConfig *EndpointConfig `json:"endpoint_config,omitempty"`
}

type LankongConfig struct {
	APIToken string   `json:"api_token"`
	AlbumIDs []string `json:"album_ids"`
	BaseURL  string   `json:"base_url"`
}

type ManualConfig struct {
	URLs []string `json:"urls"`
}

type APIConfig struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"` // GET, POST
	Headers  map[string]string `json:"headers,omitempty"`
	Body     string            `json:"body,omitempty"`
	URLField string            `json:"url_field"` // JSON字段路径，如 "data.url" 或 "urls[0]"
}

type EndpointConfig struct {
	EndpointIDs []uint `json:"endpoint_ids"` // 选中的端点ID列表
}
