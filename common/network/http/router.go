package http

import (
	"fmt"
	"net/http"
	"regexp"
)

// Router 是一个简单的路由器
type Router struct {
	routes     []routeEntry
	middleware []Middleware
	notFound   HandlerFunc
}

// routeEntry 表示一个路由项
type routeEntry struct {
	method      string
	pathRegexp  *regexp.Regexp
	pathPattern string
	handler     HandlerFunc
}

// NewRouter 创建一个新的路由器
func NewRouter() *Router {
	return &Router{
		routes:     make([]routeEntry, 0),
		middleware: make([]Middleware, 0),
		notFound: func(c *Context) {
			c.Response.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(c.Response, "Not Found: %s %s", c.Request.Method, c.Request.URL.Path)
		},
	}
}

// Use 添加中间件
func (r *Router) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

// NotFound 设置404处理器
func (r *Router) NotFound(handler HandlerFunc) {
	r.notFound = handler
}

// Handle 注册路由处理函数
func (r *Router) Handle(method, path string, handler HandlerFunc) {
	// 将路径模式转换为正则表达式
	regexpPattern := "^" + path + "$"

	// 替换路径参数为正则捕获组
	paramRegex := regexp.MustCompile(`{([a-zA-Z0-9_]+)}`)
	regexpPattern = paramRegex.ReplaceAllString(regexpPattern, `(?P<$1>[^/]+)`)

	// 编译正则表达式
	pathRegexp := regexp.MustCompile(regexpPattern)

	// 添加路由
	r.routes = append(r.routes, routeEntry{
		method:      method,
		pathRegexp:  pathRegexp,
		pathPattern: path,
		handler:     handler,
	})
}

// GET 注册GET路由
func (r *Router) GET(path string, handler HandlerFunc) {
	r.Handle(http.MethodGet, path, handler)
}

// POST 注册POST路由
func (r *Router) POST(path string, handler HandlerFunc) {
	r.Handle(http.MethodPost, path, handler)
}

// PUT 注册PUT路由
func (r *Router) PUT(path string, handler HandlerFunc) {
	r.Handle(http.MethodPut, path, handler)
}

// DELETE 注册DELETE路由
func (r *Router) DELETE(path string, handler HandlerFunc) {
	r.Handle(http.MethodDelete, path, handler)
}

// ServeHTTP 实现http.Handler接口
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 创建响应写入器
	rw := newResponseWriter(w)

	// 创建上下文
	ctx := &Context{
		Request:  req,
		Response: rw,
		Params:   make(map[string]string),
	}

	// 查找匹配的路由
	var handler HandlerFunc
	for _, route := range r.routes {
		if route.method == req.Method {
			matches := route.pathRegexp.FindStringSubmatch(req.URL.Path)
			if matches != nil {
				// 提取路径参数
				for i, name := range route.pathRegexp.SubexpNames() {
					if i > 0 && i < len(matches) && name != "" {
						ctx.Params[name] = matches[i]
					}
				}
				handler = route.handler
				break
			}
		}
	}

	// 如果没有匹配的路由，使用notFound处理器
	if handler == nil {
		handler = r.notFound
	}

	// 应用中间件（按添加顺序的逆序）
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	// 执行处理器
	handler(ctx)
}

// Group 创建路由组
func (r *Router) Group(prefix string) *RouterGroup {
	return &RouterGroup{
		router: r,
		prefix: prefix,
	}
}

// RouterGroup 路由组
type RouterGroup struct {
	router *Router
	prefix string
}

// Use 为路由组添加中间件
func (g *RouterGroup) Use(middleware ...Middleware) {
	// 注意：这里的中间件会应用于所有路由，不只是组内的路由
	// 要实现组级别的中间件需要更复杂的实现
	g.router.middleware = append(g.router.middleware, middleware...)
}

// GET 在组内注册GET路由
func (g *RouterGroup) GET(path string, handler HandlerFunc) {
	g.router.GET(g.prefix+path, handler)
}

// POST 在组内注册POST路由
func (g *RouterGroup) POST(path string, handler HandlerFunc) {
	g.router.POST(g.prefix+path, handler)
}

// PUT 在组内注册PUT路由
func (g *RouterGroup) PUT(path string, handler HandlerFunc) {
	g.router.PUT(g.prefix+path, handler)
}

// DELETE 在组内注册DELETE路由
func (g *RouterGroup) DELETE(path string, handler HandlerFunc) {
	g.router.DELETE(g.prefix+path, handler)
}
