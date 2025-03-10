package metrics

import (
    "time"
)

// HTTPMetric 表示HTTP请求指标数据
type HTTPMetric struct {
    Method     string        // HTTP方法(GET, POST等)
    Path       string        // 请求路径
    StatusCode int           // HTTP状态码
    Duration   int64         // 请求处理时间(毫秒)
    Timestamp  time.Time     // 请求时间
}

// SystemMetric 表示系统指标数据
type SystemMetric struct {
    CPUUsage    float64   // CPU使用率 (0-100%)
    MemoryUsage float64   // 内存使用率 (0-100%)
    DiskUsage   float64   // 磁盘使用率 (0-100%)
    Timestamp   time.Time // 收集时间
}

