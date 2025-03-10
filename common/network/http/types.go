package http

import "net/http"

// RequestOption 表示HTTP请求选项
type RequestOption func(*http.Request)

// WithHeader 添加请求头选项
func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithQueryParam 添加查询参数选项
func WithQueryParam(key, value string) RequestOption {
	return func(req *http.Request) {
		q := req.URL.Query()
		q.Add(key, value)
		req.URL.RawQuery = q.Encode()
	}
}

// WithBasicAuth 添加基本认证选项
func WithBasicAuth(username, password string) RequestOption {
	return func(req *http.Request) {
		req.SetBasicAuth(username, password)
	}
}
