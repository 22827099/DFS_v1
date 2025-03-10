package errors

import (
	"encoding/json"
	"errors" // 导入标准错误库
	"fmt"
)

// Error 定义系统错误结构
type Error struct {
	Code     ErrorCode      // 错误码
	Message  string         // 错误消息
	Metadata map[string]any // 附加数据（可选）
	Cause    error          // 原始错误（可选，用于错误链）
}

// New 创建一个新的系统错误
func New(code ErrorCode, message string, args ...any) *Error {
	var formattedMessage string
	if len(args) > 0 {
		formattedMessage = fmt.Sprintf(message, args...)
	} else {
		formattedMessage = message
	}

	if formattedMessage == "" {
		// 使用默认的错误码描述
		formattedMessage = code.Text()
	}

	return &Error{
		Code:    code,
		Message: formattedMessage,
	}
}

// Newf 使用格式化字符串创建错误
func Newf(code ErrorCode, format string, args ...any) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap 包装一个已有错误
func Wrap(err error, code ErrorCode, message string) *Error {
	if err == nil {
		return New(code, message)
	}

	// 如果已经是我们的错误类型，复用并更新
	var e *Error
	if errors.As(err, &e) {
		// 如果没有指定新消息，保留原始消息
		if message == "" {
			return &Error{
				Code:     code,
				Message:  e.Message,
				Metadata: e.Metadata,
				Cause:    e.Cause,
			}
		}
		return &Error{
			Code:     code,
			Message:  message,
			Metadata: e.Metadata,
			Cause:    e.Cause,
		}
	}

	// 包装普通错误
	if message == "" {
		message = code.Text()
	}

	return &Error{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// Wrapf 使用格式化字符串包装错误
func Wrapf(err error, code ErrorCode, format string, args ...any) *Error {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// Error 实现error接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 实现errors.Unwrap接口，用于错误链支持
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is 支持errors.Is检查错误类型
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithField 为错误添加元数据字段
func (e *Error) WithField(key string, value any) *Error {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// WithFields 添加多个元数据字段
func (e *Error) WithFields(fields map[string]any) *Error {
	if e.Metadata == nil {
		e.Metadata = fields
		return e
	}

	for k, v := range fields {
		e.Metadata[k] = v
	}
	return e
}

// GetCode 从任意错误中提取错误码
func GetCode(err error) ErrorCode {
	if err == nil {
		return 0
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return Unknown
}

// GetMessage 从任意错误中获取错误消息
func GetMessage(err error) string {
	if err == nil {
		return ""
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Message
	}
	return err.Error()
}

// 错误类型检查辅助函数
func IsErrorCode(err error, code ErrorCode) bool {
	return GetCode(err) == code
}

// 检查是否为"未找到"类型错误
func IsNotFound(err error) bool {
	code := GetCode(err)
	return code == NotFound || code == FileNotFound
}

// 检查是否为无效参数错误
func IsInvalidArgument(err error) bool {
	return IsErrorCode(err, InvalidArgument)
}

// 检查是否为权限拒绝错误
func IsPermissionDenied(err error) bool {
	return IsErrorCode(err, PermissionDenied)
}

// 检查是否为已存在类型错误
func IsAlreadyExists(err error) bool {
	code := GetCode(err)
	return code == AlreadyExists || code == FileAlreadyExists
}

// 检查是否为未认证错误
func IsUnauthenticated(err error) bool {
	return IsErrorCode(err, Unauthenticated)
}

// 检查是否为资源耗尽错误
func IsResourceExhausted(err error) bool {
	return IsErrorCode(err, ResourceExhausted)
}

// 检查是否为内部错误
func IsInternal(err error) bool {
	return IsErrorCode(err, Internal)
}

// 实现JSON序列化接口
func (e *Error) MarshalJSON() ([]byte, error) {
	type jsonError struct {
		Code     ErrorCode      `json:"code"`
		Message  string         `json:"message"`
		Metadata map[string]any `json:"metadata,omitempty"`
		Cause    string         `json:"cause,omitempty"`
	}

	je := jsonError{
		Code:     e.Code,
		Message:  e.Message,
		Metadata: e.Metadata,
	}

	if e.Cause != nil {
		je.Cause = e.Cause.Error()
	}

	return json.Marshal(je)
}