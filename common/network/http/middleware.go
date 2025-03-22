package http

import (
	"net/http"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Middleware 定义HTTP中间件类型 - 使用别名
type Middleware = mux.MiddlewareFunc

// LoggingMiddleware 创建日志中间件
func LoggingMiddleware(logger logging.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 创建响应记录器以捕获状态码
			recorder := &responseRecorder{
				ResponseWriter: w,
				StatusCode:     http.StatusOK,
			}

			// 处理请求
			next.ServeHTTP(recorder, r)

			// 记录请求详情
			duration := time.Since(start)
			logger.Info("HTTP %s %s %d %s",
				r.Method, r.URL.Path, recorder.StatusCode, duration)
		})
	}
}

// RecoveryMiddleware 创建恢复中间件，防止服务器崩溃
func RecoveryMiddleware(logger logging.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("服务器恢复自panic: %v", err)
					RespondError(w, http.StatusInternalServerError, "服务器内部错误")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware 为每个请求添加唯一ID
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()
			w.Header().Set("X-Request-ID", requestID)

			// 将请求添加到上下文
			ctx := WithRequestID(r.Context(), requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORSMiddleware 处理跨域请求 - 修复了问题
func CORSMiddleware(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// 检查是否是允许的来源
			originAllowed := false
			for _, allowed := range allowedOrigins {
				if allowed == "*" {
					// 如果允许所有来源，直接使用请求的Origin
					if origin != "" {
						w.Header().Set("Access-Control-Allow-Origin", origin)
					} else {
						w.Header().Set("Access-Control-Allow-Origin", "*")
					}
					originAllowed = true
					break
				} else if origin == allowed {
					// 特定来源匹配
					w.Header().Set("Access-Control-Allow-Origin", origin)
					originAllowed = true
					break
				}
			}

			// 如果找不到匹配的来源但配置了来源列表，使用第一个作为默认值
			if !originAllowed && len(allowedOrigins) > 0 && origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigins[0])
			}

			// 设置其他CORS头
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24小时

			// 处理OPTIONS预检请求
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware 创建基本的身份验证中间件
func AuthMiddleware(authFunc func(string, string) bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()

			if !ok || !authFunc(username, password) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				RespondError(w, http.StatusUnauthorized, "认证失败")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseRecorder 是http.ResponseWriter的包装，用于记录状态码
type responseRecorder struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader 覆盖ResponseWriter的WriteHeader方法以记录状态码
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
