package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

// ContextHandler 定义基于Context的处理函数类型
type ContextHandler func(c *Context)

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

		// 调用原始处理函数
		handler(ctx)
	}
}
