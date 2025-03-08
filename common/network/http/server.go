package http

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
)

// Handler 是HTTP处理函数类型
type Handler func(w http.ResponseWriter, r *http.Request)

// Server 是HTTP服务器的简单封装
type Server struct {
    server *http.Server
    router *mux.Router
    logger *log.Logger
}

// NewServer 创建一个新的HTTP服务器
func NewServer(addr string) *Server {
    router := mux.NewRouter()
    return &Server{
        server: &http.Server{
            Addr:         addr,
            Handler:      router,
            ReadTimeout:  15 * time.Second,
            WriteTimeout: 15 * time.Second,
            IdleTimeout:  60 * time.Second,
        },
        router: router,
        logger: log.Default(),
    }
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
    s.logger.Printf("HTTP服务器启动于 %s\n", s.server.Addr)
    return s.server.ListenAndServe()
}

// Stop 停止HTTP服务器
func (s *Server) Stop(ctx context.Context) error {
    s.logger.Println("HTTP服务器正在关闭")
    return s.server.Shutdown(ctx)
}

// GET 注册GET路由
func (s *Server) GET(path string, handler Handler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodGet)
}

// POST 注册POST路由
func (s *Server) POST(path string, handler Handler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodPost)
}

// PUT 注册PUT路由
func (s *Server) PUT(path string, handler Handler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodPut)
}

// DELETE 注册DELETE路由
func (s *Server) DELETE(path string, handler Handler) {
    s.router.HandleFunc(path, handler).Methods(http.MethodDelete)
}

// Group 创建路由组
func (s *Server) Group(prefix string) *mux.Router {
    return s.router.PathPrefix(prefix).Subrouter()
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