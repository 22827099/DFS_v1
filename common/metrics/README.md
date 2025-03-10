# 监控与指标

此目录提供简单的指标收集和监控功能：
- 支持HTTP请求指标收集
- 支持系统资源使用情况统计
- 内存存储设计，适合轻量级应用

## 使用示例

```go
// 创建指标收集器
collector := metrics.NewCollector("metaserver")

// 记录HTTP请求指标
collector.RecordHTTPRequest("GET", "/api/files", 200, 45)

// 记录系统指标
collector.RecordSystemMetrics(35.6, 42.1, 68.3)

// 获取收集的指标
httpMetrics := collector.GetHTTPMetrics()
systemMetrics := collector.GetSystemMetrics()

// 在中间件中使用
app.Use(middleware.Metrics(collector))
```

## 如何在元数据服务器中使用

在元数据服务器初始化时添加：

```go
// 创建MetadataServer实例
func NewServer(cfg *config.SystemConfig, options ...ServerOption) (*MetadataServer, error) {
    // ...现有代码...
    
    // 初始化指标收集器
    metricsCollector := metrics.NewCollector("metaserver")
    
    server := &MetadataServer{
        config:          cfg,
        logger:          logger,
        httpServer:      httpServer,
        metaCore:        metaCore,
        metricsCollector: metricsCollector,
        // ...其他字段...
    }
    
    // ...现有代码...
    return server, nil
}
```