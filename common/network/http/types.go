package http

import (
	"net/http"
	"time"
)

// ResponseWriter 是对http.ResponseWriter的简单封装，添加了状态码跟踪
type ResponseWriter interface {
	http.ResponseWriter
	StatusCode() int
}

// Context 表示HTTP请求上下文
type Context struct {
	Request  *http.Request
	Response ResponseWriter
	Params   map[string]string // 用于存储路由参数
}

// HandlerFunc 是处理HTTP请求的函数类型
type HandlerFunc func(*Context)

// Middleware 是HTTP中间件函数类型
type Middleware func(HandlerFunc) HandlerFunc

// ClientOption 定义客户端配置选项
type ClientOption func(*Client)

// ServerOption 定义服务器配置选项
type ServerOption func(*Server)

// 响应类型定义
const (
	ContentTypeJSON  = "application/json"
	ContentTypeXML   = "application/xml"
	ContentTypePlain = "text/plain"
)

// RetryPolicy 定义重试策略
type RetryPolicy struct {
	MaxRetries    int                                       // 最大重试次数
	RetryInterval time.Duration                             // 重试间隔
	MaxBackoff    time.Duration                             // 最大退避时间
	ShouldRetry   func(resp *http.Response, err error) bool // 判断是否应该重试
}

// Logger 定义日志接口
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}
