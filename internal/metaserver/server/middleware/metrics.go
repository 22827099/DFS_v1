package middleware

import (
    "net/http"
    "time"
    
    nethttp "github.com/22827099/DFS_v1/common/network/http"
    "github.com/22827099/DFS_v1/common/metrics"
)

// Metrics 创建指标收集中间件
func Metrics(metricsCollector metrics.Collector) nethttp.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // 包装ResponseWriter以捕获状态码
            recorder := &responseRecorder{
                ResponseWriter: w,
                statusCode:     http.StatusOK,
            }
            
            // 处理请求
            next.ServeHTTP(recorder, r)
            
            // 记录请求指标
            duration := time.Since(start)
            metricsCollector.RecordHTTPRequest(
                r.Method,
                r.URL.Path,
                recorder.statusCode,
                duration.Milliseconds(),
            )
        })
    }
}