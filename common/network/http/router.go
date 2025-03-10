package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Router 表示HTTP路由器接口
type Router interface {
    http.Handler
    GET(path string, handler Handler)
    POST(path string, handler Handler)
    PUT(path string, handler Handler) 
    DELETE(path string, handler Handler)
    Group(prefix string) RouteGroup
}

// RouteGroup 表示路由组接口
type RouteGroup interface {
    GET(path string, handler Handler)
    POST(path string, handler Handler)
    PUT(path string, handler Handler)
    DELETE(path string, handler Handler)
    Group(prefix string) RouteGroup
}

// routerImpl 实现Router接口
type routerImpl struct {
    router *mux.Router
}

// NewRouter 创建新的路由器
func NewRouter() Router {
    return &routerImpl{
        router: mux.NewRouter(),
    }
}

// ServeHTTP 实现http.Handler接口
func (r *routerImpl) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    r.router.ServeHTTP(w, req)
}

// GET 注册GET请求处理器
func (r *routerImpl) GET(path string, handler Handler) {
    r.router.HandleFunc(path, http.HandlerFunc(handler)).Methods("GET")
}

// POST 注册POST请求处理器
func (r *routerImpl) POST(path string, handler Handler) {
    r.router.HandleFunc(path, http.HandlerFunc(handler)).Methods("POST")
}

// PUT 注册PUT请求处理器
func (r *routerImpl) PUT(path string, handler Handler) {
    r.router.HandleFunc(path, http.HandlerFunc(handler)).Methods("PUT")
}

// DELETE 注册DELETE请求处理器
func (r *routerImpl) DELETE(path string, handler Handler) {
    r.router.HandleFunc(path, http.HandlerFunc(handler)).Methods("DELETE")
}

// Group 创建路由组
func (r *routerImpl) Group(prefix string) RouteGroup {
    return &routeGroupImpl{
        prefix: prefix,
        router: r,
    }
}

// routeGroupImpl 实现RouteGroup接口
type routeGroupImpl struct {
    prefix string
    router *routerImpl
}

// GET 在组内注册GET请求处理器
func (g *routeGroupImpl) GET(path string, handler Handler) {
    g.router.GET(g.prefix+path, handler)
}

// POST 在组内注册POST请求处理器
func (g *routeGroupImpl) POST(path string, handler Handler) {
    g.router.POST(g.prefix+path, handler)
}

// PUT 在组内注册PUT请求处理器
func (g *routeGroupImpl) PUT(path string, handler Handler) {
    g.router.PUT(g.prefix+path, handler)
}

// DELETE 在组内注册DELETE请求处理器
func (g *routeGroupImpl) DELETE(path string, handler Handler) {
    g.router.DELETE(g.prefix+path, handler)
}

// Group 创建子路由组
func (g *routeGroupImpl) Group(prefix string) RouteGroup {
    return &routeGroupImpl{
        prefix: g.prefix + prefix,
        router: g.router,
    }
}
