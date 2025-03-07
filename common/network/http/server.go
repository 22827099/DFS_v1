package http

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
)

// Server 是HTTP服务器封装
type Server struct {
	server     *http.Server
	router     *Router
	logger     Logger
	middleware []Middleware
}

// NewServer 创建新的HTTP服务器
func NewServer(addr string, options ...ServerOption) *Server {
	router := NewRouter()

	server := &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		router:     router,
		logger:     logging.NewLogger(),
		middleware: []Middleware{},
	}

	// 应用选项
	for _, option := range options {
		option(server)
	}

	// 应用中间件
	server.applyMiddleware()

	return server
}

// WithServerLogger 设置服务器日志记录器
func WithServerLogger(logger Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithServerMiddleware 添加服务器中间件
func WithServerMiddleware(middleware ...Middleware) ServerOption {
	return func(s *Server) {
		s.middleware = append(s.middleware, middleware...)
	}
}

// WithReadTimeout 设置服务器读取超时
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.server.ReadTimeout = timeout
	}
}

// WithWriteTimeout 设置服务器写入超时
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.server.WriteTimeout = timeout
	}
}

// applyMiddleware 应用中间件到路由器
func (s *Server) applyMiddleware() {
	// 按顺序应用默认中间件
	s.router.Use(
		RecoveryMiddleware(s.logger), // 首先应用恢复中间件
		LoggerMiddleware(s.logger),   // 然后是日志中间件
	)

	// 应用自定义中间件
	s.router.Use(s.middleware...)
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	s.logger.Info("HTTP服务器启动在 %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// StartTLS 启动HTTPS服务器
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.logger.Info("HTTPS服务器启动在 %s", s.server.Addr)
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// Stop 停止HTTP服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("HTTP服务器正在关闭")
	return s.server.Shutdown(ctx)
}

// Router 返回路由器实例
func (s *Server) Router() *Router {
	return s.router
}

// GET 注册GET路由处理器
func (s *Server) GET(path string, handler HandlerFunc) {
	s.router.GET(path, handler)
}

// POST 注册POST路由处理器
func (s *Server) POST(path string, handler HandlerFunc) {
	s.router.POST(path, handler)
}

// PUT 注册PUT路由处理器
func (s *Server) PUT(path string, handler HandlerFunc) {
	s.router.PUT(path, handler)
}

// DELETE 注册DELETE路由处理器
func (s *Server) DELETE(path string, handler HandlerFunc) {
	s.router.DELETE(path, handler)
}

// Group 创建路由组
func (s *Server) Group(prefix string) *RouterGroup {
	return s.router.Group(prefix)
}

// Use 添加服务器级中间件
func (s *Server) Use(middleware ...Middleware) {
	s.router.Use(middleware...)
}

// Handle 注册自定义方法的路由处理器
func (s *Server) Handle(method, path string, handler HandlerFunc) {
	s.router.Handle(method, path, handler)
}

// ServeHTTP 实现http.Handler接口
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// WithMaxHeaderBytes 设置最大请求头大小
func WithMaxHeaderBytes(size int) ServerOption {
	return func(s *Server) {
		s.server.MaxHeaderBytes = size
	}
}

// WithIdleTimeout 设置空闲连接超时
func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.server.IdleTimeout = timeout
	}
}

// WithTLSConfig 设置TLS配置
func WithTLSConfig(tlsConfig *tls.Config) ServerOption {
	return func(s *Server) {
		s.server.TLSConfig = tlsConfig
	}
}

// WithBaseContext 设置基础上下文生成器
func WithBaseContext(baseCtxFn func(net.Listener) context.Context) ServerOption {
	return func(s *Server) {
		s.server.BaseContext = baseCtxFn
	}
}

// NotFound 设置404处理器
func (s *Server) NotFound(handler HandlerFunc) {
	s.router.NotFound(handler)
}

// Addr 返回服务器地址
func (s *Server) Addr() string {
	return s.server.Addr
}

// SetKeepAlivesEnabled 控制是否启用HTTP长连接
func (s *Server) SetKeepAlivesEnabled(v bool) {
	s.server.SetKeepAlivesEnabled(v)
}
