package middleware

import (
	"net/http"
	"sync"
	"time"
	"strings"
	"net"
	"fmt"

	"github.com/22827099/DFS_v1/common/errors"
	nethttp "github.com/22827099/DFS_v1/common/network/http"
	"github.com/22827099/DFS_v1/internal/metaserver/server/api"
)

// RateLimit 创建速率限制中间件
func RateLimit(limit int, window time.Duration) nethttp.Middleware {
    var mu sync.Mutex
    requests := make(map[string][]time.Time)

    // 定期清理过期的请求记录
    go func() {
        for {
            time.Sleep(window)
            mu.Lock()
            now := time.Now()
            for ip, times := range requests {
                var active []time.Time
                for _, t := range times {
                    if now.Sub(t) < window {
                        active = append(active, t)
                    }
                }
                if len(active) == 0 {
                    delete(requests, ip)
                } else {
                    requests[ip] = active
                }
            }
            mu.Unlock()
        }
    }()

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 获取客户端IP地址
            ip := getClientIP(r)
            
            mu.Lock()
            // 清理该IP的过期请求
            now := time.Now()
            active := make([]time.Time, 0)
            
            if times, found := requests[ip]; found {
                for _, t := range times {
                    if now.Sub(t) < window {
                        active = append(active, t)
                    }
                }
            }
            
            // 检查是否超过速率限制
            if len(active) >= limit {
                mu.Unlock()
                w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
                api.RespondError(w, r, http.StatusTooManyRequests, 
                    errors.New(errors.RateLimitExceeded, "请求频率超过限制，请稍后再试"))
                return
            }
            
            // 记录新的请求时间
            requests[ip] = append(active, now)
            mu.Unlock()
            
            // 继续处理请求
            next.ServeHTTP(w, r)
        })
    }
}

// getClientIP 从请求中提取客户端IP地址
func getClientIP(r *http.Request) string {
    // 尝试从X-Forwarded-For头获取
    ip := r.Header.Get("X-Forwarded-For")
    if ip != "" {
        parts := strings.Split(ip, ",")
        return strings.TrimSpace(parts[0])
    }
    
    // 尝试从X-Real-IP头获取
    ip = r.Header.Get("X-Real-IP")
    if ip != "" {
        return ip
    }
    
    // 从RemoteAddr获取
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return ip
}