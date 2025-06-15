package initapp

import (
	"log"
	"random-api-go/database"
	"random-api-go/model"
	"random-api-go/service"
	"sync"
	"time"
)

// InitData 初始化应用数据，预加载所有需要的数据到内存中
func InitData() error {
	log.Println("开始初始化应用数据...")
	start := time.Now()

	// 1. 初始化端点服务（这会启动预加载器）
	endpointService := service.GetEndpointService()
	log.Println("✓ 端点服务已初始化")

	// 2. 暂停预加载器的定期刷新，避免与初始化冲突
	preloader := endpointService.GetPreloader()
	preloader.PausePeriodicRefresh()
	log.Println("✓ 已暂停预加载器定期刷新")

	// 3. 获取所有活跃的端点和数据源
	endpoints, err := endpointService.ListEndpoints()
	if err != nil {
		log.Printf("获取端点列表失败: %v", err)
		return err
	}

	// 4. 统计需要预加载的数据源
	var activeDataSources []model.DataSource
	var totalDataSources, disabledDataSources, apiDataSources int

	for _, endpoint := range endpoints {
		if !endpoint.IsActive {
			continue
		}
		for _, ds := range endpoint.DataSources {
			totalDataSources++

			if !ds.IsActive {
				disabledDataSources++
				log.Printf("跳过禁用的数据源: %s (ID: %d)", ds.Name, ds.ID)
				continue
			}

			if ds.Type == "api_get" || ds.Type == "api_post" {
				apiDataSources++
				log.Printf("跳过API类型数据源: %s (ID: %d, 类型: %s) - 使用实时请求", ds.Name, ds.ID, ds.Type)
				continue
			}

			// 需要预加载的数据源
			activeDataSources = append(activeDataSources, ds)
		}
	}

	log.Printf("发现 %d 个端点，总共 %d 个数据源", len(endpoints), totalDataSources)
	log.Printf("其中: 禁用 %d 个，API类型 %d 个，需要预加载 %d 个", disabledDataSources, apiDataSources, len(activeDataSources))

	if len(activeDataSources) == 0 {
		log.Println("✓ 没有需要预加载的数据源")
		// 恢复预加载器定期刷新
		preloader.ResumePeriodicRefresh()
		log.Printf("应用数据初始化完成，耗时: %v", time.Since(start))
		return nil
	}

	// 5. 并发预加载所有数据源
	var wg sync.WaitGroup
	var successCount, failCount int
	var mutex sync.Mutex

	// 限制并发数，避免过多的并发请求
	semaphore := make(chan struct{}, 5) // 最多5个并发

	for _, ds := range activeDataSources {
		wg.Add(1)
		go func(dataSource model.DataSource) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Printf("预加载数据源: %s (ID: %d, 类型: %s)", dataSource.Name, dataSource.ID, dataSource.Type)

			// 使用预加载器预加载数据源
			if err := preloader.RefreshDataSource(dataSource.ID); err != nil {
				log.Printf("预加载数据源 %d 失败: %v", dataSource.ID, err)
				mutex.Lock()
				failCount++
				mutex.Unlock()
			} else {
				log.Printf("预加载数据源 %d 成功", dataSource.ID)
				mutex.Lock()
				successCount++
				mutex.Unlock()
			}
		}(ds)
	}

	// 等待所有预加载完成
	wg.Wait()

	log.Printf("✓ 数据源预加载完成: 成功 %d 个，失败 %d 个", successCount, failCount)

	// 6. 预热URL统计缓存
	log.Println("预热URL统计缓存...")
	if err := preloadURLStats(endpointService, endpoints); err != nil {
		log.Printf("预热URL统计缓存失败: %v", err)
	} else {
		log.Println("✓ URL统计缓存预热完成")
	}

	// 7. 预加载配置
	log.Println("预加载系统配置...")
	preloadConfigs()
	log.Println("✓ 系统配置预加载完成")

	// 8. 恢复预加载器定期刷新
	preloader.ResumePeriodicRefresh()
	log.Println("✓ 已恢复预加载器定期刷新")

	duration := time.Since(start)
	log.Printf("🎉 应用数据初始化完成，总耗时: %v", duration)

	return nil
}

// preloadURLStats 预热URL统计缓存
func preloadURLStats(endpointService *service.EndpointService, endpoints []*model.APIEndpoint) error {
	urlStats := make(map[string]struct {
		TotalURLs int `json:"total_urls"`
	})

	for _, endpoint := range endpoints {
		if !endpoint.IsActive {
			continue
		}

		totalURLs := 0
		for _, ds := range endpoint.DataSources {
			if ds.IsActive {
				// 使用优化后的URL计数方法
				count, err := endpointService.GetDataSourceURLCount(&ds)
				if err != nil {
					log.Printf("获取数据源 %d URL数量失败: %v", ds.ID, err)
					// 使用估算值
					switch ds.Type {
					case "manual":
						totalURLs += 5
					case "lankong":
						totalURLs += 100
					case "api_get", "api_post":
						totalURLs += 1
					default:
						totalURLs += 1
					}
				} else {
					totalURLs += count
				}
			}
		}

		urlStats[endpoint.URL] = struct {
			TotalURLs int `json:"total_urls"`
		}{
			TotalURLs: totalURLs,
		}
	}

	log.Printf("预热了 %d 个端点的URL统计", len(urlStats))
	return nil
}

// preloadConfigs 预加载系统配置
func preloadConfigs() {
	// 预加载常用配置，触发数据库查询并缓存结果
	configs := []string{
		// 基础配置
		"homepage_content",

		// OAuth配置
		"oauth_client_id",
		"oauth_client_secret",
		"oauth_redirect_uri",

		// 系统配置
		"site_title",
		"site_description",
		"admin_email",
		"max_file_size",
		"allowed_file_types",

		// API配置
		"rate_limit_enabled",
		"rate_limit_requests",
		"rate_limit_window",
		"cors_enabled",
		"cors_origins",

		// 兰空图床配置
		"lankong_max_retries",

		// 缓存配置
		"cache_enabled",
		"cache_ttl",
		"max_cache_size",

		// 日志配置
		"log_level",
		"log_file_enabled",
		"log_retention_days",
	}

	var loadedCount int
	for _, key := range configs {
		value := database.GetConfig(key, "")
		if value != "" {
			log.Printf("预加载配置 %s: %d 字符", key, len(value))
			loadedCount++
		}
	}

	log.Printf("预加载了 %d 个配置项", loadedCount)
}

// GetInitStatus 获取初始化状态（可用于健康检查）
func GetInitStatus() map[string]interface{} {
	endpointService := service.GetEndpointService()

	// 获取缓存统计
	cacheStats := endpointService.GetCacheManager().GetCacheStats()

	// 获取端点数量
	endpoints, _ := endpointService.ListEndpoints()
	activeEndpoints := 0
	for _, ep := range endpoints {
		if ep.IsActive {
			activeEndpoints++
		}
	}

	// 获取配置缓存统计
	configCacheStats := database.GetConfigCacheStats()

	return map[string]interface{}{
		"data_cache": map[string]interface{}{
			"items":   len(cacheStats),
			"details": cacheStats,
		},
		"config_cache": configCacheStats,
		"endpoints": map[string]interface{}{
			"total":  len(endpoints),
			"active": activeEndpoints,
		},
	}
}
