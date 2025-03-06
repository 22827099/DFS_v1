package errors

import (
	"fmt"
)

// 错误类型定义
type Error struct {
	Code     int            // 错误码
	Message  string         // 格式化后的消息
	Metadata map[string]any // 附加数据
	Cause    error          // 原始错误
}

// 创建新错误（自动应用当前语言）
func New(code int, args ...any) *Error {
	lang := "zh" // 从配置中获取当前语言，默认中文
	msgTemplate, ok := messages[code][lang]
	if !ok {
		msgTemplate = "未知错误"
	}

	msg := fmt.Sprintf(msgTemplate, args...)
	return &Error{
		Code:     code,
		Message:  msg,
		Metadata: make(map[string]any),
	}
}

// 错误包装
func Wrap(err error, code int, args ...any) *Error {
	e, ok := err.(*Error)
	if !ok {
		return New(code, args...).WithCause(err)
	}
	return e
}

// 判断错误类型
func IsNotFound(err error) bool {
	e, ok := err.(*Error)
	return ok && e.Code == ErrFileNotFound
}

// 为错误添加附加字段
func (e *Error) WithField(key string, value any) *Error {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// 为错误添加原始错误
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// 获取错误的HTTP状态码
func (e *Error) HTTPStatus() int {
	// 示例映射规则：
	switch e.Code / 1000 {
	case 1:
		return 400 // 客户端错误
	case 2:
		return 500 // 服务端错误
	default:
		return 500
	}
}

// 实现 error 接口
func (e *Error) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Message)
}
