package service

import "sync"

type URLStats struct {
	TotalURLs int `json:"total_urls"`
}

var (
	URLCounts = make(map[string]struct {
		stats   URLStats
		csvPath string // 内部使用，不会输出到JSON
	})
	urlMu sync.RWMutex
)

// 更新URL计数
func UpdateURLCount(endpoint string, csvPath string, count int) {
	urlMu.Lock()
	defer urlMu.Unlock()
	URLCounts[endpoint] = struct {
		stats   URLStats
		csvPath string
	}{
		stats: URLStats{
			TotalURLs: count,
		},
		csvPath: csvPath,
	}
}

// 获取URL计数
func GetURLCounts() map[string]URLStats {
	urlMu.RLock()
	defer urlMu.RUnlock()

	// 返回一个只包含统计信息的副本
	counts := make(map[string]URLStats)
	for k, v := range URLCounts {
		counts[k] = v.stats
	}
	return counts
}
