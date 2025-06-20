package service

import (
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"random-api-go/database"
	"random-api-go/model"

	"gorm.io/gorm"
)

// DomainStatsService 域名统计服务
type DomainStatsService struct {
	mu          sync.RWMutex
	memoryStats map[string]*DomainStatsBatch // 内存中的批量统计
}

// DomainStatsBatch 批量统计数据
type DomainStatsBatch struct {
	Count    uint64
	LastSeen time.Time
}

var (
	domainStatsService *DomainStatsService
	domainStatsOnce    sync.Once
)

// GetDomainStatsService 获取域名统计服务实例
func GetDomainStatsService() *DomainStatsService {
	domainStatsOnce.Do(func() {
		domainStatsService = &DomainStatsService{
			memoryStats: make(map[string]*DomainStatsBatch),
		}
		// 启动定期保存任务
		go domainStatsService.startPeriodicSave()
	})
	return domainStatsService
}

// extractDomain 从Referer中提取域名
func (s *DomainStatsService) extractDomain(referer string) string {
	if referer == "" {
		return "direct" // 直接访问
	}

	parsedURL, err := url.Parse(referer)
	if err != nil {
		return "unknown"
	}

	domain := parsedURL.Hostname()
	if domain == "" {
		return "unknown"
	}

	return domain
}

// RecordRequest 记录请求（忽略静态文件和管理后台）
func (s *DomainStatsService) RecordRequest(path, referer string) {
	// 忽略的路径模式
	if s.shouldIgnorePath(path) {
		return
	}

	domain := s.extractDomain(referer)

	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if batch, exists := s.memoryStats[domain]; exists {
		batch.Count++
		batch.LastSeen = now
	} else {
		s.memoryStats[domain] = &DomainStatsBatch{
			Count:    1,
			LastSeen: now,
		}
	}
}

// shouldIgnorePath 判断是否应该忽略此路径的统计
func (s *DomainStatsService) shouldIgnorePath(path string) bool {
	// 忽略管理后台API
	if strings.HasPrefix(path, "/api/admin/") {
		return true
	}

	// 忽略系统API
	if strings.HasPrefix(path, "/api") {
		return true
	}

	// 忽略静态文件路径
	if strings.HasPrefix(path, "/_next/") ||
		strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/favicon.ico") ||
		s.hasFileExtension(path) {
		return true
	}

	// 忽略前端路由
	if strings.HasPrefix(path, "/admin") {
		return true
	}

	// 忽略根路径（通常是前端首页）
	if path == "/" {
		return true
	}

	// 其他路径都统计（包括所有API端点）
	return false
}

// hasFileExtension 检查路径是否包含文件扩展名
func (s *DomainStatsService) hasFileExtension(path string) bool {
	// 获取路径的最后一部分
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false
	}

	lastPart := parts[len(parts)-1]

	// 检查是否包含点号且不是隐藏文件
	if strings.Contains(lastPart, ".") && !strings.HasPrefix(lastPart, ".") {
		// 常见的文件扩展名
		commonExts := []string{
			".html", ".css", ".js", ".json", ".png", ".jpg", ".jpeg",
			".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot",
			".txt", ".xml", ".pdf", ".zip", ".mp4", ".mp3", ".webp",
		}

		for _, ext := range commonExts {
			if strings.HasSuffix(strings.ToLower(lastPart), ext) {
				return true
			}
		}
	}

	return false
}

// startPeriodicSave 启动定期保存任务（每1分钟保存一次）
func (s *DomainStatsService) startPeriodicSave() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// 定期清理任务（每天执行一次）
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.flushToDatabase(); err != nil {
				log.Printf("Failed to flush domain stats to database: %v", err)
			}
		case <-cleanupTicker.C:
			if err := s.CleanupOldStats(); err != nil {
				log.Printf("Failed to cleanup old domain stats: %v", err)
			} else {
				log.Println("Domain stats cleanup completed successfully")
			}
		}
	}
}

// flushToDatabase 将内存中的统计数据保存到数据库
func (s *DomainStatsService) flushToDatabase() error {
	s.mu.Lock()
	currentStats := s.memoryStats
	s.memoryStats = make(map[string]*DomainStatsBatch) // 重置内存统计
	s.mu.Unlock()

	if len(currentStats) == 0 {
		return nil
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		today := time.Now().Truncate(24 * time.Hour)

		for domain, batch := range currentStats {
			// 更新总统计
			var domainStats model.DomainStats
			err := tx.Where("domain = ?", domain).First(&domainStats).Error
			if err == gorm.ErrRecordNotFound {
				// 创建新记录
				domainStats = model.DomainStats{
					Domain:   domain,
					Count:    batch.Count,
					LastSeen: batch.LastSeen,
				}
				if err := tx.Create(&domainStats).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				// 更新现有记录
				domainStats.Count += batch.Count
				domainStats.LastSeen = batch.LastSeen
				if err := tx.Save(&domainStats).Error; err != nil {
					return err
				}
			}

			// 更新每日统计
			var dailyStats model.DailyDomainStats
			err = tx.Where("domain = ? AND date = ?", domain, today).First(&dailyStats).Error
			if err == gorm.ErrRecordNotFound {
				// 创建新记录
				dailyStats = model.DailyDomainStats{
					Domain: domain,
					Date:   today,
					Count:  batch.Count,
				}
				if err := tx.Create(&dailyStats).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				// 更新现有记录
				dailyStats.Count += batch.Count
				if err := tx.Save(&dailyStats).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// GetTop24HourDomains 获取24小时内访问最多的前20个域名
func (s *DomainStatsService) GetTop24HourDomains() ([]model.DomainStatsResult, error) {
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	today := time.Now().Truncate(24 * time.Hour)

	// 从数据库获取数据
	var dbResults []model.DomainStatsResult
	err := database.DB.Model(&model.DailyDomainStats{}).
		Select("domain, SUM(count) as count").
		Where("date >= ? AND date <= ?", yesterday, today).
		Group("domain").
		Scan(&dbResults).Error
	if err != nil {
		return nil, err
	}

	// 合并内存数据
	s.mu.RLock()
	memoryStats := make(map[string]uint64)
	for domain, batch := range s.memoryStats {
		memoryStats[domain] = batch.Count
	}
	s.mu.RUnlock()

	// 创建域名计数映射
	domainCounts := make(map[string]uint64)

	// 添加数据库数据
	for _, result := range dbResults {
		domainCounts[result.Domain] += result.Count
	}

	// 添加内存数据
	for domain, count := range memoryStats {
		domainCounts[domain] += count
	}

	// 转换为结果切片并排序
	var results []model.DomainStatsResult
	for domain, count := range domainCounts {
		results = append(results, model.DomainStatsResult{
			Domain: domain,
			Count:  count,
		})
	}

	// 按访问次数降序排序，次数相同时按域名首字母排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Count < results[j].Count ||
				(results[i].Count == results[j].Count && results[i].Domain > results[j].Domain) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// 限制返回前20个
	if len(results) > 30 {
		results = results[:30]
	}

	return results, nil
}

// GetTopTotalDomains 获取总访问最多的前20个域名
func (s *DomainStatsService) GetTopTotalDomains() ([]model.DomainStatsResult, error) {
	// 从数据库获取数据
	var dbResults []model.DomainStatsResult
	err := database.DB.Model(&model.DomainStats{}).
		Select("domain, count").
		Scan(&dbResults).Error
	if err != nil {
		return nil, err
	}

	// 合并内存数据
	s.mu.RLock()
	memoryStats := make(map[string]uint64)
	for domain, batch := range s.memoryStats {
		memoryStats[domain] = batch.Count
	}
	s.mu.RUnlock()

	// 创建域名计数映射
	domainCounts := make(map[string]uint64)

	// 添加数据库数据
	for _, result := range dbResults {
		domainCounts[result.Domain] += result.Count
	}

	// 添加内存数据
	for domain, count := range memoryStats {
		domainCounts[domain] += count
	}

	// 转换为结果切片并排序
	var results []model.DomainStatsResult
	for domain, count := range domainCounts {
		results = append(results, model.DomainStatsResult{
			Domain: domain,
			Count:  count,
		})
	}

	// 按访问次数降序排序，次数相同时按域名首字母排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Count < results[j].Count ||
				(results[i].Count == results[j].Count && results[i].Domain > results[j].Domain) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// 限制返回前20个
	if len(results) > 30 {
		results = results[:30]
	}

	return results, nil
}

// CleanupOldStats 清理旧的统计数据（保留最近30天的每日统计）
func (s *DomainStatsService) CleanupOldStats() error {
	cutoffDate := time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	return database.DB.Where("date < ?", cutoffDate).Delete(&model.DailyDomainStats{}).Error
}
