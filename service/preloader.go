package service

import (
	"fmt"
	"log"
	"random-api-go/database"
	"random-api-go/model"
	"sync"
	"time"
)

// Preloader 预加载管理器
type Preloader struct {
	dataSourceFetcher *DataSourceFetcher
	cacheManager      *CacheManager
	running           bool
	paused            bool
	stopChan          chan struct{}
	mutex             sync.RWMutex
}

// NewPreloader 创建预加载管理器
func NewPreloader(dataSourceFetcher *DataSourceFetcher, cacheManager *CacheManager) *Preloader {
	return &Preloader{
		dataSourceFetcher: dataSourceFetcher,
		cacheManager:      cacheManager,
		stopChan:          make(chan struct{}),
	}
}

// Start 启动预加载器
func (p *Preloader) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.running {
		return
	}

	p.running = true
	go p.runPeriodicRefresh()
	log.Println("预加载器已启动")
}

// Stop 停止预加载器
func (p *Preloader) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.running {
		return
	}

	p.running = false
	close(p.stopChan)
	log.Println("预加载器已停止")
}

// PausePeriodicRefresh 暂停定期刷新
func (p *Preloader) PausePeriodicRefresh() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.paused = true
	log.Println("预加载器定期刷新已暂停")
}

// ResumePeriodicRefresh 恢复定期刷新
func (p *Preloader) ResumePeriodicRefresh() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.paused = false
	log.Println("预加载器定期刷新已恢复")
}

// PreloadDataSourceOnSave 在保存数据源时预加载数据
func (p *Preloader) PreloadDataSourceOnSave(dataSource *model.DataSource) {
	// 检查数据源是否处于活跃状态
	if !dataSource.IsActive {
		log.Printf("数据源 %d 已禁用，跳过预加载", dataSource.ID)
		return
	}

	// API类型的数据源不需要预加载，使用实时请求
	if dataSource.Type == "api_get" || dataSource.Type == "api_post" {
		log.Printf("API数据源 %d (%s) 使用实时请求，跳过预加载", dataSource.ID, dataSource.Type)
		return
	}

	// 异步预加载，避免阻塞保存操作
	go func() {
		log.Printf("开始预加载数据源 %d (%s)", dataSource.ID, dataSource.Type)

		if err := p.dataSourceFetcher.PreloadDataSource(dataSource); err != nil {
			log.Printf("预加载数据源 %d 失败: %v", dataSource.ID, err)
		} else {
			log.Printf("数据源 %d 预加载成功", dataSource.ID)
		}
	}()
}

// PreloadEndpointOnSave 在保存端点时预加载所有相关数据源
func (p *Preloader) PreloadEndpointOnSave(endpoint *model.APIEndpoint) {
	// 异步预加载，避免阻塞保存操作
	go func() {
		log.Printf("开始预加载端点 %d 的所有数据源", endpoint.ID)

		var wg sync.WaitGroup
		for _, dataSource := range endpoint.DataSources {
			if !dataSource.IsActive {
				continue
			}

			// API类型和端点类型的数据源跳过预加载
			if dataSource.Type == "api_get" || dataSource.Type == "api_post" || dataSource.Type == "endpoint" {
				log.Printf("实时数据源 %d (%s) 使用实时请求，跳过预加载", dataSource.ID, dataSource.Type)
				continue
			}

			wg.Add(1)
			go func(ds model.DataSource) {
				defer wg.Done()

				if err := p.dataSourceFetcher.PreloadDataSource(&ds); err != nil {
					log.Printf("预加载数据源 %d 失败: %v", ds.ID, err)
				}
			}(dataSource)
		}

		wg.Wait()
		log.Printf("端点 %d 的所有数据源预加载完成", endpoint.ID)

		// 预加载完成后，清理该端点的内存缓存，强制下次访问时重新构建
		p.cacheManager.InvalidateMemoryCache(endpoint.URL)
	}()
}

// RefreshDataSource 手动刷新指定数据源
func (p *Preloader) RefreshDataSource(dataSourceID uint) error {
	var dataSource model.DataSource
	if err := database.DB.First(&dataSource, dataSourceID).Error; err != nil {
		return err
	}

	// 检查数据源是否处于活跃状态
	if !dataSource.IsActive {
		log.Printf("数据源 %d 已禁用，跳过刷新", dataSourceID)
		return nil
	}

	// API类型的数据源不需要预加载，使用实时请求
	if dataSource.Type == "api_get" || dataSource.Type == "api_post" {
		log.Printf("API数据源 %d (%s) 使用实时请求，跳过刷新", dataSource.ID, dataSource.Type)
		return nil
	}

	log.Printf("手动刷新数据源 %d", dataSourceID)
	return p.dataSourceFetcher.RefreshDataSource(&dataSource)
}

// RefreshEndpoint 手动刷新指定端点的所有数据源
func (p *Preloader) RefreshEndpoint(endpointID uint) error {
	var endpoint model.APIEndpoint
	if err := database.DB.Preload("DataSources").First(&endpoint, endpointID).Error; err != nil {
		return err
	}

	log.Printf("手动刷新端点 %d 的所有数据源", endpointID)

	var wg sync.WaitGroup
	var lastErr error

	for _, dataSource := range endpoint.DataSources {
		if !dataSource.IsActive {
			continue
		}

		// API类型和端点类型的数据源跳过刷新
		if dataSource.Type == "api_get" || dataSource.Type == "api_post" || dataSource.Type == "endpoint" {
			log.Printf("实时数据源 %d (%s) 使用实时请求，跳过刷新", dataSource.ID, dataSource.Type)
			continue
		}

		wg.Add(1)
		go func(ds model.DataSource) {
			defer wg.Done()

			if err := p.dataSourceFetcher.RefreshDataSource(&ds); err != nil {
				log.Printf("刷新数据源 %d 失败: %v", ds.ID, err)
				lastErr = err
			}
		}(dataSource)
	}

	wg.Wait()

	// 刷新完成后，清理该端点的内存缓存
	p.cacheManager.InvalidateMemoryCache(endpoint.URL)

	return lastErr
}

// runPeriodicRefresh 运行定期刷新任务
func (p *Preloader) runPeriodicRefresh() {
	// 定期刷新间隔：每30分钟检查一次
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// 启动时立即执行一次检查
	p.checkAndRefreshExpiredData()

	for {
		select {
		case <-ticker.C:
			p.checkAndRefreshExpiredData()
		case <-p.stopChan:
			return
		}
	}
}

// checkAndRefreshExpiredData 检查并刷新过期数据
func (p *Preloader) checkAndRefreshExpiredData() {
	// 检查是否暂停
	p.mutex.RLock()
	isPaused := p.paused
	p.mutex.RUnlock()

	if isPaused {
		log.Println("预加载器定期刷新已暂停，跳过此次检查")
		return
	}

	log.Println("开始检查过期数据...")

	// 获取所有活跃的数据源
	var dataSources []model.DataSource
	if err := database.DB.Where("is_active = ?", true).Find(&dataSources).Error; err != nil {
		log.Printf("获取数据源列表失败: %v", err)
		return
	}

	var refreshCount int
	var wg sync.WaitGroup

	for _, dataSource := range dataSources {
		// API类型和端点类型的数据源跳过定期刷新
		if dataSource.Type == "api_get" || dataSource.Type == "api_post" || dataSource.Type == "endpoint" {
			continue
		}

		// 检查内存缓存是否存在
		cacheKey := fmt.Sprintf("datasource_%d", dataSource.ID)
		cachedURLs, exists := p.cacheManager.GetFromMemoryCache(cacheKey)
		if !exists || len(cachedURLs) == 0 {
			// 没有缓存数据，需要刷新
			refreshCount++
			wg.Add(1)
			go func(ds model.DataSource) {
				defer wg.Done()
				p.refreshDataSourceAsync(&ds)
			}(dataSource)
			continue
		}

		// 检查是否需要定期刷新（兰空图床需要定期刷新）
		if p.shouldPeriodicRefresh(&dataSource) {
			refreshCount++
			wg.Add(1)
			go func(ds model.DataSource) {
				defer wg.Done()
				p.refreshDataSourceAsync(&ds)
			}(dataSource)
		}
	}

	if refreshCount > 0 {
		log.Printf("正在刷新 %d 个数据源...", refreshCount)
		wg.Wait()
		log.Printf("数据源刷新完成")
	} else {
		log.Println("所有数据源都是最新的，无需刷新")
	}
}

// shouldPeriodicRefresh 判断是否需要定期刷新
func (p *Preloader) shouldPeriodicRefresh(dataSource *model.DataSource) bool {
	// 手动数据、API数据和端点数据不需要定期刷新
	if dataSource.Type == "manual" || dataSource.Type == "api_get" || dataSource.Type == "api_post" || dataSource.Type == "endpoint" {
		return false
	}

	// 如果没有最后同步时间，需要刷新
	if dataSource.LastSync == nil {
		return true
	}

	// 根据数据源类型设置不同的刷新间隔
	var refreshInterval time.Duration
	switch dataSource.Type {
	case "lankong":
		refreshInterval = 24 * time.Hour // 兰空图床每24小时刷新一次
	case "s3":
		refreshInterval = 24 * time.Hour // S3存储每24小时刷新一次
	default:
		return false
	}

	return time.Since(*dataSource.LastSync) > refreshInterval
}

// refreshDataSourceAsync 异步刷新数据源
func (p *Preloader) refreshDataSourceAsync(dataSource *model.DataSource) {
	if err := p.dataSourceFetcher.PreloadDataSource(dataSource); err != nil {
		log.Printf("定期刷新数据源 %d 失败: %v", dataSource.ID, err)
	} else {
		log.Printf("数据源 %d 定期刷新成功", dataSource.ID)

		// 更新数据库中的同步时间
		now := time.Now()
		if err := database.DB.Model(dataSource).Update("last_sync", now).Error; err != nil {
			log.Printf("更新数据源 %d 同步时间失败: %v", dataSource.ID, err)
		}
	}
}
