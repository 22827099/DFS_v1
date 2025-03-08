package http

// Response 表示标准API响应
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Code    int         `json:"code,omitempty"`
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}) Response {
    return Response{
        Success: true,
        Data:    data,
    }
}

// ErrorResponse 创建错误响应
func ErrorResponse(message string, code int) Response {
    return Response{
        Success: false,
        Error:   message,
        Code:    code,
    }
}