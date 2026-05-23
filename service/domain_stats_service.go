package service

import (
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"random-api-go/database"
	"random-api-go/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DomainStatsService 域名统计服务
// 内存中按 (domain, path) 二维聚合, 定期 flush 到数据库
type DomainStatsService struct {
	mu          sync.RWMutex
	memoryStats map[domainPathKey]*DomainStatsBatch

	blockMu      sync.RWMutex
	blockedHosts map[string]struct{}
}

type domainPathKey struct {
	Domain string
	Path   string
}

// DomainStatsBatch 单个 (domain, path) 的内存批量数据
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
			memoryStats:  make(map[domainPathKey]*DomainStatsBatch),
			blockedHosts: make(map[string]struct{}),
		}
		if err := domainStatsService.reloadBlockedDomains(); err != nil {
			log.Printf("Failed to load blocked domains: %v", err)
		}
		go domainStatsService.startPeriodicSave()
	})
	return domainStatsService
}

// ExtractDomain 从 Referer 中提取主机名; 中间件复用本逻辑保证统计与黑名单口径一致
func (s *DomainStatsService) ExtractDomain(referer string) string {
	if referer == "" {
		return "direct"
	}
	parsedURL, err := url.Parse(referer)
	if err != nil {
		return "unknown"
	}
	domain := parsedURL.Hostname()
	if domain == "" {
		return "unknown"
	}
	return strings.ToLower(domain)
}

// IsBlocked 判断指定域名是否被黑名单禁用
func (s *DomainStatsService) IsBlocked(domain string) bool {
	if domain == "" {
		return false
	}
	s.blockMu.RLock()
	defer s.blockMu.RUnlock()
	_, ok := s.blockedHosts[strings.ToLower(domain)]
	return ok
}

// reloadBlockedDomains 从数据库重新加载黑名单到内存
func (s *DomainStatsService) reloadBlockedDomains() error {
	var rows []model.BlockedDomain
	if err := database.DB.Find(&rows).Error; err != nil {
		return err
	}
	next := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		next[strings.ToLower(r.Domain)] = struct{}{}
	}
	s.blockMu.Lock()
	s.blockedHosts = next
	s.blockMu.Unlock()
	return nil
}

// SetBlocked 设置/取消禁用指定域名, 同步更新内存缓存
// direct / unknown 是占位伪域名, 禁用会误伤无 referer 的真实浏览器访问, 在此层硬挡
func (s *DomainStatsService) SetBlocked(domain, reason string, blocked bool) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" || domain == "direct" || domain == "unknown" {
		return nil
	}
	if blocked {
		entry := model.BlockedDomain{Domain: domain, Reason: reason}
		err := database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "domain"}},
			DoUpdates: clause.AssignmentColumns([]string{"reason", "updated_at"}),
		}).Create(&entry).Error
		if err != nil {
			return err
		}
	} else {
		if err := database.DB.Unscoped().Where("domain = ?", domain).Delete(&model.BlockedDomain{}).Error; err != nil {
			return err
		}
	}
	return s.reloadBlockedDomains()
}

// ListBlockedDomains 返回当前所有被禁用的域名
func (s *DomainStatsService) ListBlockedDomains() ([]model.BlockedDomain, error) {
	var rows []model.BlockedDomain
	err := database.DB.Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// RecordRequest 记录一次请求 (忽略静态文件与管理后台)
func (s *DomainStatsService) RecordRequest(path, referer string) {
	if s.shouldIgnorePath(path) {
		return
	}
	domain := s.ExtractDomain(referer)
	key := domainPathKey{Domain: domain, Path: path}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	if batch, ok := s.memoryStats[key]; ok {
		batch.Count++
		batch.LastSeen = now
	} else {
		s.memoryStats[key] = &DomainStatsBatch{Count: 1, LastSeen: now}
	}
}

// shouldIgnorePath 判断是否应该忽略此路径的统计
func (s *DomainStatsService) shouldIgnorePath(path string) bool {
	if strings.HasPrefix(path, "/api") { // 含 /api/admin
		return true
	}
	if strings.HasPrefix(path, "/_next/") ||
		strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/favicon.ico") ||
		s.hasFileExtension(path) {
		return true
	}
	if strings.HasPrefix(path, "/admin") {
		return true
	}
	if path == "" || path == "/" {
		return true
	}
	return false
}

// hasFileExtension 检查路径是否包含常见的静态文件扩展名
func (s *DomainStatsService) hasFileExtension(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false
	}
	lastPart := parts[len(parts)-1]
	if !strings.Contains(lastPart, ".") || strings.HasPrefix(lastPart, ".") {
		return false
	}
	exts := []string{".html", ".css", ".js", ".json", ".png", ".jpg", ".jpeg",
		".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot",
		".txt", ".xml", ".pdf", ".zip", ".mp4", ".mp3", ".webp"}
	low := strings.ToLower(lastPart)
	for _, ext := range exts {
		if strings.HasSuffix(low, ext) {
			return true
		}
	}
	return false
}

// startPeriodicSave 启动定期保存与清理任务
func (s *DomainStatsService) startPeriodicSave() {
	flushTicker := time.NewTicker(1 * time.Minute)
	defer flushTicker.Stop()
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			if err := s.flushToDatabase(); err != nil {
				log.Printf("Failed to flush domain stats to database: %v", err)
			}
		case <-cleanupTicker.C:
			if err := s.CleanupOldStats(); err != nil {
				log.Printf("Failed to cleanup old domain stats: %v", err)
			}
		}
	}
}

// flushToDatabase 将内存中的 (domain, path) 统计批量 upsert 到数据库
func (s *DomainStatsService) flushToDatabase() error {
	s.mu.Lock()
	current := s.memoryStats
	s.memoryStats = make(map[domainPathKey]*DomainStatsBatch)
	s.mu.Unlock()

	if len(current) == 0 {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)

	return database.DB.Transaction(func(tx *gorm.DB) error {
		for key, batch := range current {
			// 累计表 upsert: count += batch.Count, last_seen 取较新值
			total := model.DomainStats{
				Domain:   key.Domain,
				Path:     key.Path,
				Count:    batch.Count,
				LastSeen: batch.LastSeen,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "domain"}, {Name: "path"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"count":      gorm.Expr("count + ?", batch.Count),
					"last_seen":  batch.LastSeen,
					"updated_at": time.Now(),
				}),
			}).Create(&total).Error; err != nil {
				return err
			}

			// 每日表 upsert
			daily := model.DailyDomainStats{
				Domain: key.Domain,
				Path:   key.Path,
				Date:   today,
				Count:  batch.Count,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "date"}, {Name: "domain"}, {Name: "path"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"count":      gorm.Expr("count + ?", batch.Count),
					"updated_at": time.Now(),
				}),
			}).Create(&daily).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// memoryAggregateByDomain 把内存数据按 domain 维度聚合
func (s *DomainStatsService) memoryAggregateByDomain() map[string]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]uint64)
	for key, batch := range s.memoryStats {
		out[key.Domain] += batch.Count
	}
	return out
}

// memoryAggregateByDomainPath 内存数据按 (domain, path) 输出, 供详情接口合并使用
func (s *DomainStatsService) memoryAggregateByDomainPath(domain string) map[string]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]uint64)
	for key, batch := range s.memoryStats {
		if key.Domain == domain {
			out[key.Path] += batch.Count
		}
	}
	return out
}

// sortAndLimit 按 count 降序 / 同分按 domain 升序排序并截断 top
func sortAndLimit(results []model.DomainStatsResult, top int) []model.DomainStatsResult {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Count != results[j].Count {
			return results[i].Count > results[j].Count
		}
		return results[i].Domain < results[j].Domain
	})
	if top > 0 && len(results) > top {
		results = results[:top]
	}
	return results
}

func (s *DomainStatsService) attachBlockedFlag(results []model.DomainStatsResult) []model.DomainStatsResult {
	s.blockMu.RLock()
	defer s.blockMu.RUnlock()
	for i := range results {
		_, ok := s.blockedHosts[strings.ToLower(results[i].Domain)]
		results[i].IsBlocked = ok
	}
	return results
}

// getDomainRankWithinDays 查询最近 N 天 (含今天) 按 domain 聚合的访问排行
func (s *DomainStatsService) getDomainRankWithinDays(days int) ([]model.DomainStatsResult, error) {
	end := time.Now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(days - 1))

	var rows []model.DomainStatsResult
	err := database.DB.Model(&model.DailyDomainStats{}).
		Select("domain, SUM(count) as count").
		Where("date >= ? AND date <= ?", start, end).
		Group("domain").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 合并内存数据 (今日尚未 flush 部分)
	counts := make(map[string]uint64)
	for _, r := range rows {
		counts[r.Domain] += r.Count
	}
	for d, c := range s.memoryAggregateByDomain() {
		counts[d] += c
	}

	out := make([]model.DomainStatsResult, 0, len(counts))
	for d, c := range counts {
		out = append(out, model.DomainStatsResult{Domain: d, Count: c})
	}
	return s.attachBlockedFlag(sortAndLimit(out, 30)), nil
}

// GetTop24HourDomains 24 小时内访问最多的域名
func (s *DomainStatsService) GetTop24HourDomains() ([]model.DomainStatsResult, error) {
	return s.getDomainRankWithinDays(1)
}

// GetTop7DayDomains 最近 7 天访问最多的域名
func (s *DomainStatsService) GetTop7DayDomains() ([]model.DomainStatsResult, error) {
	return s.getDomainRankWithinDays(7)
}

// GetTop30DayDomains 最近 30 天访问最多的域名
func (s *DomainStatsService) GetTop30DayDomains() ([]model.DomainStatsResult, error) {
	return s.getDomainRankWithinDays(30)
}

// GetTopTotalDomains 历史总访问最多的域名
func (s *DomainStatsService) GetTopTotalDomains() ([]model.DomainStatsResult, error) {
	var rows []model.DomainStatsResult
	err := database.DB.Model(&model.DomainStats{}).
		Select("domain, SUM(count) as count").
		Group("domain").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]uint64)
	for _, r := range rows {
		counts[r.Domain] += r.Count
	}
	for d, c := range s.memoryAggregateByDomain() {
		counts[d] += c
	}

	out := make([]model.DomainStatsResult, 0, len(counts))
	for d, c := range counts {
		out = append(out, model.DomainStatsResult{Domain: d, Count: c})
	}
	return s.attachBlockedFlag(sortAndLimit(out, 30)), nil
}

// GetDomainPathStats 返回指定 domain 下的 path 维度排行
// rangeKey: "24h" | "7d" | "30d" | "total"
func (s *DomainStatsService) GetDomainPathStats(domain, rangeKey string) ([]model.DomainPathStatsResult, error) {
	domain = strings.ToLower(domain)
	pathCounts := make(map[string]uint64)

	switch rangeKey {
	case "total":
		var rows []struct {
			Path  string
			Count uint64
		}
		if err := database.DB.Model(&model.DomainStats{}).
			Select("path, SUM(count) as count").
			Where("domain = ?", domain).
			Group("path").
			Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			pathCounts[r.Path] += r.Count
		}
	default:
		days := 1
		switch rangeKey {
		case "7d":
			days = 7
		case "30d":
			days = 30
		}
		end := time.Now().Truncate(24 * time.Hour)
		start := end.AddDate(0, 0, -(days - 1))
		var rows []struct {
			Path  string
			Count uint64
		}
		if err := database.DB.Model(&model.DailyDomainStats{}).
			Select("path, SUM(count) as count").
			Where("domain = ? AND date >= ? AND date <= ?", domain, start, end).
			Group("path").
			Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			pathCounts[r.Path] += r.Count
		}
	}

	for p, c := range s.memoryAggregateByDomainPath(domain) {
		pathCounts[p] += c
	}

	out := make([]model.DomainPathStatsResult, 0, len(pathCounts))
	for p, c := range pathCounts {
		out = append(out, model.DomainPathStatsResult{Path: p, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// CleanupOldStats 清理 90 天前的每日统计数据
func (s *DomainStatsService) CleanupOldStats() error {
	cutoff := time.Now().AddDate(0, 0, -90).Truncate(24 * time.Hour)
	return database.DB.Where("date < ?", cutoff).Delete(&model.DailyDomainStats{}).Error
}

// GetDailyTotalSeries 返回最近 N 天的每日总访问量序列 (按日期升序)
// 缺数据的日子补 0, 方便前端直接画线
func (s *DomainStatsService) GetDailyTotalSeries(days int) ([]model.DomainDailyPoint, error) {
	if days <= 0 {
		days = 30
	}
	end := time.Now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(days - 1))

	var rows []struct {
		Date  time.Time
		Count uint64
	}
	if err := database.DB.Model(&model.DailyDomainStats{}).
		Select("date, SUM(count) as count").
		Where("date >= ? AND date <= ?", start, end).
		Group("date").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	bucket := make(map[string]uint64, days)
	for _, r := range rows {
		bucket[r.Date.Format("2006-01-02")] += r.Count
	}
	// 把内存中今日尚未 flush 的部分加到今天
	var memToday uint64
	for _, b := range s.snapshotMemoryCounts() {
		memToday += b
	}
	todayKey := end.Format("2006-01-02")
	bucket[todayKey] += memToday

	out := make([]model.DomainDailyPoint, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		out = append(out, model.DomainDailyPoint{Date: d, Count: bucket[d]})
	}
	return out, nil
}

// snapshotMemoryCounts 返回内存批次的 count 快照, 不暴露内部 map 结构
func (s *DomainStatsService) snapshotMemoryCounts() []uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]uint64, 0, len(s.memoryStats))
	for _, b := range s.memoryStats {
		out = append(out, b.Count)
	}
	return out
}
