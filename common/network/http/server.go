package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
)

// Handler 是HTTP处理函数类型
type Handler func(w http.ResponseWriter, r *http.Request)

// HandlerAdapter 将Handler适配为http.HandlerFunc
func HandlerAdapter(h Handler) http.HandlerFunc {
    return http.HandlerFunc(h)
}

// Server 表示HTTP服务器
type Server struct {
	addr         string
	handler      http.Handler
	router       Router
	logger       logging.Logger
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	middlewares  []Middleware
	server       *http.Server
}

// NewServer 创建新的HTTP服务器实例
func NewServer(addr string, options ...ServerOption) *Server {
    server := &Server{
        addr:         addr,
        router:       NewRouter(),
        readTimeout:  time.Second * 30,
        writeTimeout: time.Second * 30,
        idleTimeout:  time.Second * 60,
        middlewares:  []Middleware{},
    }
    
    server.handler = server.router
    
    // 应用所有选项
    for _, option := range options {
        option(server)
    }
    
    return server
}

// Use 添加中间件到服务器
func (s *Server) Use(middleware Middleware) {
	s.middlewares = append(s.middlewares, middleware)
}

// Router 返回服务器的路由器
func (s *Server) Router() Router {
	return s.router
}

// SetLogger 设置日志记录器
func (s *Server) SetLogger(logger logging.Logger) {
	s.logger = logger
}

// SetReadTimeout 设置读取超时
func (s *Server) SetReadTimeout(timeout time.Duration) {
	s.readTimeout = timeout
}

// SetWriteTimeout 设置写入超时
func (s *Server) SetWriteTimeout(timeout time.Duration) {
	s.writeTimeout = timeout
}

// SetIdleTimeout 设置空闲超时
func (s *Server) SetIdleTimeout(timeout time.Duration) {
	s.idleTimeout = timeout
}

// // buildHandler 构建处理器链
// func (s *Server) buildHandler() http.Handler {
//     // 从路由器开始
//     var handler http.Handler = s.router
    
//     // 应用所有中间件
//     for i := len(s.middlewares) - 1; i >= 0; i-- {
//         handler = s.middlewares[i](handler)
//     }
    
//     return handler
// }

// Start 启动HTTP服务器
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      s.handler,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
		IdleTimeout:  s.idleTimeout,
	}

	if s.logger != nil {
		s.logger.Info("HTTP服务器启动于 %s", s.addr)
	}

	return s.server.ListenAndServe()
}

// StartTLS 启动HTTPS服务器
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      s.handler,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
		IdleTimeout:  s.idleTimeout,
	}

	if s.logger != nil {
		s.logger.Info("HTTPS服务器启动于 %s", s.addr)
	}

	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// Stop 停止HTTP服务器
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Shutdown 优雅地关闭HTTP服务器
func (s *Server) Shutdown(ctx context.Context) error {
    // 委托给标准库Server的Shutdown方法
    return s.server.Shutdown(ctx)
}

// GET 注册GET路由
func (s *Server) GET(path string, handler Handler) {
    s.router.GET(path, handler)  // 使用Router接口定义的GET方法
}

// POST 注册POST路由
func (s *Server) POST(path string, handler Handler) {
    s.router.POST(path, handler)  // 使用Router接口定义的POST方法
}

// PUT 注册PUT路由
func (s *Server) PUT(path string, handler Handler) {
    s.router.PUT(path, handler)  // 使用Router接口定义的PUT方法
}

// DELETE 注册DELETE路由
func (s *Server) DELETE(path string, handler Handler) {
    s.router.DELETE(path, handler)  // 使用Router接口定义的DELETE方法
}

// Group 创建路由组
func (s *Server) Group(prefix string) RouteGroup {
    return s.router.Group(prefix)  // 返回RouteGroup接口
}
// RespondJSON 发送JSON响应
func RespondJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// RespondError 发送错误响应
func RespondError(w http.ResponseWriter, status int, message string) error {
	return RespondJSON(w, status, map[string]string{"error": message})
}

// ServerOption 服务器配置选项
type ServerOption func(*Server)

// WithServerLogger 设置服务器日志记录器
func WithServerLogger(logger logging.Logger) ServerOption {
    return func(s *Server) {
        s.SetLogger(logger)
    }
}

// WithReadTimeout 设置读取超时
func WithReadTimeout(timeout time.Duration) ServerOption {
    return func(s *Server) {
        s.SetReadTimeout(timeout)
    }
}

// WithWriteTimeout 设置写入超时
func WithWriteTimeout(timeout time.Duration) ServerOption {
    return func(s *Server) {
        s.SetWriteTimeout(timeout)
    }
}

// WithIdleTimeout 设置空闲超时
func WithIdleTimeout(timeout time.Duration) ServerOption {
    return func(s *Server) {
        s.SetIdleTimeout(timeout)
    }
}

// WithMiddleware 添加中间件
func WithMiddleware(middleware Middleware) ServerOption {
    return func(s *Server) {
        s.Use(middleware)
    }
}

// WithCORS 添加CORS中间件
func WithCORS(allowOrigins []string) ServerOption {
    return func(s *Server) {
        s.Use(CORSMiddleware(allowOrigins))
    }
}

// WithRecovery 添加恢复中间件
func WithRecovery(logger logging.Logger) ServerOption {
    return func(s *Server) {
        s.Use(RecoveryMiddleware(logger))
    }
}

// WithLogging 添加日志中间件
func WithLogging(logger logging.Logger) ServerOption {
    return func(s *Server) {
        s.Use(LoggingMiddleware(logger))
    }
}

// WithAuth 添加认证中间件
func WithAuth(authFunc func(username, password string) bool) ServerOption {
    return func(s *Server) {
        s.Use(AuthMiddleware(authFunc))
    }
}