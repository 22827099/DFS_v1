package http // 假设这是您的包名

import (
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
)

// Context 封装HTTP请求和响应
type Context struct {
    Request  *http.Request
    Response http.ResponseWriter
    Params   map[string]string
    // 可以添加更多字段，如：
    // Logger   *logging.Logger
    // UserID   string
    // Session  *sessions.Session
}

// JSON 发送JSON响应
func (c *Context) JSON(statusCode int, data interface{}) error {
    c.Response.Header().Set("Content-Type", "application/json")
    c.Response.WriteHeader(statusCode)
    return json.NewEncoder(c.Response).Encode(data)
}

// Text 发送文本响应
func (c *Context) Text(statusCode int, text string) {
    c.Response.Header().Set("Content-Type", "text/plain")
    c.Response.WriteHeader(statusCode)
    c.Response.Write([]byte(text))
}

// Error 发送错误响应
func (c *Context) Error(statusCode int, message string) error {
    return c.JSON(statusCode, map[string]string{"error": message})
}

// GetParam 获取URL参数
func (c *Context) GetParam(key string) string {
    return c.Params[key]
}

// GetQuery 获取查询参数
func (c *Context) GetQuery(key string) string {
    return c.Request.URL.Query().Get(key)
}

// BindJSON 将请求体绑定到结构体
func (c *Context) BindJSON(obj interface{}) error {
    return json.NewDecoder(c.Request.Body).Decode(obj)
}

type ContextHandler func(c *Context)

// Handler 是HTTP处理函数类型
type Handler func(w http.ResponseWriter, r *http.Request)

// Adapt 将ContextHandler转换为标准Handler
func Adapt(handler ContextHandler) Handler {
    return func(w http.ResponseWriter, r *http.Request) {
        // 从请求中获取路由参数
        vars := mux.Vars(r)

        // 创建Context对象
        ctx := &Context{
            Request:  r,
            Response: w,
            Params:   vars,
        }

        // 调用处理函数
        handler(ctx)
    }
}