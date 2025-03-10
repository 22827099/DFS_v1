package metrics

import (
    "sync"
    "time"
)

// Collector 定义指标收集器接口
type Collector interface {
    // RecordHTTPRequest 记录HTTP请求指标
    RecordHTTPRequest(method, path string, statusCode int, durationMs int64)
    
    // RecordSystemMetrics 记录系统指标
    RecordSystemMetrics(cpu, memory, disk float64)
    
    // GetHTTPMetrics 获取HTTP指标
    GetHTTPMetrics() []HTTPMetric
    
    // GetSystemMetrics 获取系统指标
    GetSystemMetrics() []SystemMetric
    
    // Reset 重置所有指标
    Reset()
}

// SimpleCollector 是Collector接口的内存实现
type SimpleCollector struct {
    name          string
    httpMetrics   []HTTPMetric
    systemMetrics []SystemMetric
    maxItems      int
    mu            sync.RWMutex
}

// NewCollector 创建一个新的指标收集器
func NewCollector(name string) Collector {
    return &SimpleCollector{
        name:          name,
        httpMetrics:   make([]HTTPMetric, 0, 1000),
        systemMetrics: make([]SystemMetric, 0, 100),
        maxItems:      1000, // 限制存储项数，避免内存无限增长
    }
}

// RecordHTTPRequest 实现Collector接口
func (c *SimpleCollector) RecordHTTPRequest(method, path string, statusCode int, durationMs int64) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 如果达到最大存储限制，移除最早的记录
    if len(c.httpMetrics) >= c.maxItems {
        c.httpMetrics = c.httpMetrics[1:]
    }
    
    c.httpMetrics = append(c.httpMetrics, HTTPMetric{
        Method:     method,
        Path:       path,
        StatusCode: statusCode,
        Duration:   durationMs,
        Timestamp:  time.Now(),
    })
}

// RecordSystemMetrics 实现Collector接口
func (c *SimpleCollector) RecordSystemMetrics(cpu, memory, disk float64) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 如果达到最大存储限制，移除最早的记录
    if len(c.systemMetrics) >= c.maxItems/10 {
        c.systemMetrics = c.systemMetrics[1:]
    }
    
    c.systemMetrics = append(c.systemMetrics, SystemMetric{
        CPUUsage:    cpu,
        MemoryUsage: memory,
        DiskUsage:   disk,
        Timestamp:   time.Now(),
    })
}

// GetHTTPMetrics 实现Collector接口
func (c *SimpleCollector) GetHTTPMetrics() []HTTPMetric {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    // 返回副本以避免并发访问问题
    result := make([]HTTPMetric, len(c.httpMetrics))
    copy(result, c.httpMetrics)
    return result
}

// GetSystemMetrics 实现Collector接口
func (c *SimpleCollector) GetSystemMetrics() []SystemMetric {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    // 返回副本以避免并发访问问题
    result := make([]SystemMetric, len(c.systemMetrics))
    copy(result, c.systemMetrics)
    return result
}

// Reset 实现Collector接口
func (c *SimpleCollector) Reset() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.httpMetrics = make([]HTTPMetric, 0, 1000)
    c.systemMetrics = make([]SystemMetric, 0, 100)
}