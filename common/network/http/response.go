package http

import (
	"encoding/json"
	"net/http"
)

// responseWriter 是对http.ResponseWriter的封装
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// 确保responseWriter实现了ResponseWriter接口
var _ ResponseWriter = (*responseWriter)(nil)

// newResponseWriter 创建新的responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // 默认200
	}
}

// WriteHeader 覆盖http.ResponseWriter.WriteHeader以记录状态码
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// StatusCode 返回响应状态码
func (rw *responseWriter) StatusCode() int {
	return rw.statusCode
}

// APIResponse 定义标准API响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int         `json:"code,omitempty"`
}

// WriteJSON 将JSON响应写入ResponseWriter
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// WriteSuccess 写入成功响应
func WriteSuccess(w http.ResponseWriter, data interface{}) error {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	return WriteJSON(w, http.StatusOK, response)
}

// WriteError 写入错误响应
func WriteError(w http.ResponseWriter, statusCode int, errorMsg string, errorCode int) error {
	response := APIResponse{
		Success: false,
		Error:   errorMsg,
		Code:    errorCode,
	}
	return WriteJSON(w, statusCode, response)
}
