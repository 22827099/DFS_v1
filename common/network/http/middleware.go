package http

import (
	"net/http"
	"runtime/debug"
	"strings"
	"time"
	"context"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/google/uuid"
)

// Middleware 定义HTTP中间件类型
type Middleware func(http.Handler) http.Handler

// 定义上下文键类型，避免键冲突
type contextKey string

// 定义用于请求ID的上下文键
const requestIDKey contextKey = "request-id"

// WithRequestID 在上下文中存储请求ID
func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID 从上下文中获取请求ID
func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(requestIDKey).(string); ok {
        return id
    }
    return ""
}

// LoggingMiddleware 创建日志中间件
func LoggingMiddleware(logger logging.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 创建响应记录器
			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// 处理请求
			next.ServeHTTP(recorder, r)

			// 记录请求信息
			duration := time.Since(start)
			logger.Info("[%s] %s %s %d %s",
				r.Method, r.URL.Path, r.RemoteAddr,
				recorder.statusCode, duration)
		})
	}
}

// RecoveryMiddleware 创建恢复中间件，防止panic导致服务器崩溃
func RecoveryMiddleware(logger logging.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("HTTP处理器panic: %v\nStack: %s", err, debug.Stack())
					RespondError(w, http.StatusInternalServerError, "服务器内部错误")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware 创建请求ID中间件
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.New().String()
			}

			w.Header().Set("X-Request-ID", id)
			r = r.WithContext(WithRequestID(r.Context(), id))

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware 创建CORS中间件
func CORSMiddleware(allowOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// 检查是否允许该来源
			allowOrigin := "*"
			if len(allowOrigins) > 0 && origin != "" {
				allowed := false
				for _, o := range allowOrigins {
					if o == origin || o == "*" {
						allowed = true
						break
					}
				}

				if allowed {
					allowOrigin = origin
				}
			}

			// 设置CORS头
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// 处理预检请求
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware 创建身份验证中间件
func AuthMiddleware(authFunc func(username, password string) bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 如果是公开路径，则跳过验证
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// 获取验证信息
			username, password, ok := r.BasicAuth()
			if !ok || !authFunc(username, password) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				RespondError(w, http.StatusUnauthorized, "未授权访问")
				return
			}

			// 身份验证通过，继续处理
			next.ServeHTTP(w, r)
		})
	}
}

// 辅助函数

// isPublicPath 检查路径是否为公开路径
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/status",
		"/api/v1/auth/login",
	}

	for _, p := range publicPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// responseRecorder 用于记录响应状态码
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 重写WriteHeader方法以记录状态码
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}


