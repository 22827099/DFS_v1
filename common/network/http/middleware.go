package http

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// LoggerMiddleware 创建日志中间件
func LoggerMiddleware(logger Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			start := time.Now()

			// 处理请求
			next(c)

			// 记录请求信息
			duration := time.Since(start)
			logger.Info("[%s] %s %s %d %v",
				c.Request.Method,
				c.Request.URL.Path,
				c.Request.RemoteAddr,
				c.Response.StatusCode(),
				duration,
			)
		}
	}
}

// RecoveryMiddleware 创建恢复中间件，防止panic导致程序崩溃
func RecoveryMiddleware(logger Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			defer func() {
				if r := recover(); r != nil {
					// 记录堆栈信息
					stack := string(debug.Stack())
					logger.Error("HTTP处理器panic: %v\n%s", r, stack)

					// 确保返回500响应
					http.Error(c.Response, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next(c)
		}
	}
}

// CORSMiddleware 创建跨域资源共享中间件
func CORSMiddleware(allowedOrigins []string, allowedMethods []string) Middleware {
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginsMap[origin] = true
	}

	allowedMethodsStr := strings.Join(allowedMethods, ", ")
	allowAllOrigins := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			origin := c.Request.Header.Get("Origin")

			// 检查是否允许该来源
			if allowAllOrigins || allowedOriginsMap[origin] {
				c.Response.Header().Set("Access-Control-Allow-Origin", origin)
				c.Response.Header().Set("Access-Control-Allow-Methods", allowedMethodsStr)
				c.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Response.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// 如果是预检请求，直接返回
			if c.Request.Method == http.MethodOptions {
				c.Response.WriteHeader(http.StatusOK)
				return
			}

			next(c)
		}
	}
}

// TimeoutMiddleware 创建超时中间件
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			ctx := c.Request.Context()

			// 创建带超时的上下文
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// 使用新的上下文创建请求
			c.Request = c.Request.WithContext(ctx)

			// 处理请求
			done := make(chan struct{})
			go func() {
				next(c)
				close(done)
			}()

			// 等待请求处理完成或超时
			select {
			case <-done:
				// 请求正常完成
				return
			case <-ctx.Done():
				// 请求超时
				c.Response.WriteHeader(http.StatusRequestTimeout)
				fmt.Fprintf(c.Response, "Request timed out after %v", timeout)
				return
			}
		}
	}
}

// AuthMiddleware 创建认证中间件
func AuthMiddleware(authFunc func(*http.Request) bool) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			if !authFunc(c.Request) {
				c.Response.Header().Set("WWW-Authenticate", "Bearer")
				c.Response.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(c.Response, "Unauthorized")
				return
			}

			next(c)
		}
	}
}

// 辅助函数：连接字符串数组
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}
