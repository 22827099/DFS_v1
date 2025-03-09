package http

import (
	"encoding/json"
	"net/http"
)

// Context 请求上下文，简化处理器的参数传递和响应写入
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Params   map[string]string
}

// NewContext 创建新的请求上下文
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Request:  r,
		Response: w,
		Params:   make(map[string]string),
	}
}

// Param 获取URL参数值
func (c *Context) Param(name string) string {
	return c.Params[name]
}

// QueryParam 获取查询参数值
func (c *Context) QueryParam(name string) string {
	return c.Request.URL.Query().Get(name)
}

// BindJSON 解析请求体为JSON
func (c *Context) BindJSON(v interface{}) error {
	return json.NewDecoder(c.Request.Body).Decode(v)
}
