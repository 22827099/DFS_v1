package http

import (
    "context"
    "net"
    "net/http"
    "time"

    "github.com/22827099/DFS_v1/common/logging"
    "github.com/gorilla/mux"
)

// 服务器处理函数类型
type ServerHandler func(w http.ResponseWriter, r *http.Request)

// Server 表示HTTP服务器
type Server struct {
    addr         string
    actualAddr   string
    readTimeout  time.Duration
    writeTimeout time.Duration
    idleTimeout  time.Duration
    router       *mux.Router
    middlewares  []Middleware
    server       *http.Server
    logger       logging.Logger
}

// ServerOption 服务器配置选项
type ServerOption func(*Server)

// NewServer 创建新的HTTP服务器实例
func NewServer(addr string, options ...ServerOption) *Server {
    server := &Server{
        addr:         addr,
        router:       mux.NewRouter(),
        readTimeout:  30 * time.Second,
        writeTimeout: 30 * time.Second,
        idleTimeout:  60 * time.Second,
    }
    
    // 应用所有选项
    for _, option := range options {
        option(server)
    }
    
    return server
}

// Use 添加中间件
func (s *Server) Use(middleware Middleware) {
    s.middlewares = append(s.middlewares, middleware)
    s.router.Use(middleware)
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
    listener, err := net.Listen("tcp", s.addr)
    if err != nil {
        return err
    }
    
    s.actualAddr = listener.Addr().String()
    
    s.server = &http.Server{
        Handler:      s.router,
        ReadTimeout:  s.readTimeout,
        WriteTimeout: s.writeTimeout,
        IdleTimeout:  s.idleTimeout,
    }
    
    if s.logger != nil {
        s.logger.Info("HTTP服务器启动于 %s", s.actualAddr)
    }
    
    return s.server.Serve(listener)
}

// Stop 停止HTTP服务器
func (s *Server) Stop(ctx context.Context) error {
    if s.logger != nil {
        s.logger.Info("正在关闭HTTP服务器")
    }
    
    if s.server != nil {
        return s.server.Shutdown(ctx)
    }
    return nil
}

// GET 注册GET路由
func (s *Server) GET(path string, handler ServerHandler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodGet)
}

// POST 注册POST路由
func (s *Server) POST(path string, handler ServerHandler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodPost)
}

// PUT 注册PUT路由
func (s *Server) PUT(path string, handler ServerHandler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodPut)
}

// DELETE 注册DELETE路由
func (s *Server) DELETE(path string, handler ServerHandler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodDelete)
}

// OPTIONS 注册OPTIONS路由
func (s *Server) OPTIONS(path string, handler ServerHandler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodOptions)
}

// Group 创建路由组
func (s *Server) Group(prefix string) RouteGroup {
    return &routeGroup{
        prefix: prefix,
        server: s,
    }
}

// GetAddr 返回服务器当前监听地址
func (s *Server) GetAddr() string {
    if s.actualAddr != "" {
        return s.actualAddr
    }
    return s.addr
}

// WithLogger 设置服务器日志记录器
func WithLogger(logger logging.Logger) ServerOption {
    return func(s *Server) {
        s.logger = logger
    }
}

// WithServerTimeout 设置服务器的超时设置
func WithServerTimeout(read, write, idle time.Duration) ServerOption {
    return func(s *Server) {
        if read > 0 {
            s.readTimeout = read
        }
        if write > 0 {
            s.writeTimeout = write
        }
        if idle > 0 {
            s.idleTimeout = idle
        }
    }
}

// WithMiddleware 添加中间件
func WithMiddleware(middleware ...Middleware) ServerOption {
    return func(s *Server) {
        for _, m := range middleware {
            s.Use(m)
        }
    }
}

// RouteGroup 表示路由组
type RouteGroup interface {
    GET(path string, handler ServerHandler)
    POST(path string, handler ServerHandler)
    PUT(path string, handler ServerHandler)
    DELETE(path string, handler ServerHandler)
    OPTIONS(path string, handler ServerHandler)
    Group(prefix string) RouteGroup
}

// 路由组实现
type routeGroup struct {
    prefix string
    server *Server
}

// GET 在组内注册GET路由
func (g *routeGroup) GET(path string, handler ServerHandler) {
    g.server.GET(g.prefix+path, handler)
}

// POST 在组内注册POST路由
func (g *routeGroup) POST(path string, handler ServerHandler) {
    g.server.POST(g.prefix+path, handler)
}

// PUT 在组内注册PUT路由
func (g *routeGroup) PUT(path string, handler ServerHandler) {
    g.server.PUT(g.prefix+path, handler)
}

// DELETE 在组内注册DELETE路由
func (g *routeGroup) DELETE(path string, handler ServerHandler) {
    g.server.DELETE(g.prefix+path, handler)
}

// OPTIONS 在组内注册OPTIONS路由
func (g *routeGroup) OPTIONS(path string, handler ServerHandler) {
    g.server.OPTIONS(g.prefix+path, handler)
}

// Group 创建子路由组
func (g *routeGroup) Group(prefix string) RouteGroup {
    return &routeGroup{
        prefix: g.prefix + prefix,
        server: g.server,
    }
}