# Services 架构说明

## 文件结构

### 核心服务
- **endpoint_service.go** - 主要的端点服务，提供API端点的CRUD操作和随机URL获取
- **cache_manager.go** - 缓存管理器，负责内存缓存和数据库缓存的管理
- **preloader.go** - 预加载管理器，负责主动预加载和定时刷新数据

### 数据获取器
- **data_source_fetcher.go** - 数据源获取器，统一管理不同类型数据源的获取逻辑
- **lankong_fetcher.go** - 兰空图床专用获取器，处理兰空图床API的分页获取
- **api_fetcher.go** - API接口获取器，支持GET/POST接口的批量预获取

### 其他
- **url_counter.go** - URL计数器（原有功能）

## 主要改进

### 1. 主动预加载机制
- **保存时预加载**: 创建或更新数据源时，立即在后台预加载数据
- **定时刷新**: 每30分钟检查一次，自动刷新过期或需要更新的数据源
- **智能刷新策略**: 
  - 兰空图床: 每2小时刷新一次
  - API接口: 每1小时刷新一次
  - 手动数据: 不自动刷新

### 2. 优化的缓存策略
- **双层缓存**: 内存缓存(5分钟) + 数据库缓存(可配置)
- **智能更新**: 只有当上游数据变化时才更新数据库缓存
- **自动清理**: 定期清理过期的内存和数据库缓存

### 3. API接口预获取优化
- **批量获取**: GET接口预获取100次，POST接口预获取200次
- **去重处理**: 自动去除重复的URL
- **智能停止**: GET接口如果效率太低会提前停止预获取

### 4. 错误处理和日志
- **详细日志**: 记录每个步骤的执行情况
- **错误恢复**: 单个数据源失败不影响其他数据源
- **进度显示**: 大批量操作时显示进度信息

## 使用方式

### 基本操作
```go
// 获取服务实例
service := GetEndpointService()

// 创建端点（会自动预加载）
endpoint := &model.APIEndpoint{...}
service.CreateEndpoint(endpoint)

// 获取随机URL（优先使用缓存）
url, err := service.GetRandomURL("/api/random")
```

### 手动刷新
```go
// 刷新单个数据源
service.RefreshDataSource(dataSourceID)

// 刷新整个端点
service.RefreshEndpoint(endpointID)
```

### 控制预加载器
```go
preloader := service.GetPreloader()
preloader.Stop()  // 停止自动刷新
preloader.Start() // 重新启动
```

## 性能优化

1. **并发处理**: 多个数据源并行获取数据
2. **请求限制**: 添加延迟避免请求过快
3. **缓存优先**: 优先使用缓存数据，减少API调用
4. **智能刷新**: 根据数据源类型设置不同的刷新策略 