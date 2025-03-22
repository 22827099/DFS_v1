package http

import (
	"encoding/json"
	"net/http"
)

// StandardResponse 标准响应结构
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo 错误信息结构
type ErrorInfo struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}) StandardResponse {
	return StandardResponse{
		Success: true,
		Data:    data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(message string, code ...string) StandardResponse {
	errCode := ""
	if len(code) > 0 {
		errCode = code[0]
	}

	return StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    errCode,
			Message: message,
		},
	}
}

// RespondJSON 发送JSON响应
func RespondJSON(w http.ResponseWriter, status int, data interface{}) error {
	// 如果不是StandardResponse，则包装为成功响应
	response, ok := data.(StandardResponse)
	if !ok {
		response = SuccessResponse(data)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(response)
}

// RespondError 发送错误响应
func RespondError(w http.ResponseWriter, status int, message string, code ...string) error {
	errResponse := ErrorResponse(message, code...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(errResponse)
}

// RespondText 发送文本响应
func RespondText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(text))
}

// RespondNoContent 发送无内容响应
func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
