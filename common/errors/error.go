package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// 错误类型定义
type Error struct {
	Code     int            // 错误码
	Message  string         // 格式化后的消息
	Metadata map[string]any // 附加数据
	Cause    error          // 原始错误
	Stack    string         // 堆栈跟踪信息
}

// 捕获当前堆栈信息
func captureStack(skip int) string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var builder strings.Builder
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "github.com/22827099/DFS_v1") {
			continue // 跳过非项目代码
		}
		fmt.Fprintf(&builder, "%s:%d %s\n", frame.File, frame.Line, frame.Function)
		if !more {
			break
		}
	}
	return builder.String()
}

// 创建新错误
func New(code int, args ...any) *Error {
	var message string
	if len(args) > 0 {
		message = fmt.Sprintf(args[0].(string), args[1:]...)
	}
	return &Error{
		Code:    code,
		Message: message,
		Stack:   captureStack(2),
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

// 错误类型检查函数
func IsNotFound(err error) bool {
    e, ok := err.(*Error)
    return ok && e.Code == ErrFileNotFound
}

func IsPermissionDenied(err error) bool {
    e, ok := err.(*Error)
    return ok && e.Code == ErrPermission
}

func IsAlreadyExists(err error) bool {
    e, ok := err.(*Error)
    return ok && e.Code == ErrFileAlreadyExists
}

// 通用错误码检查函数
func IsErrorCode(err error, code int) bool {
    e, ok := err.(*Error)
    return ok && e.Code == code
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

// 生成JSON格式的错误表示
func (e *Error) MarshalJSON() ([]byte, error) {
    type jsonError struct {
        Code     int               `json:"code"`
        Message  string            `json:"message"`
        Metadata map[string]any    `json:"metadata,omitempty"`
        Cause    string            `json:"cause,omitempty"`
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

// 错误域
const (
    DomainMetadata  = "metadata"
    DomainStorage   = "storage"
    DomainNetwork   = "network"
    DomainSecurity  = "security"
)

// 为错误添加域
func (e *Error) WithDomain(domain string) *Error {
    e.WithField("domain", domain)
    return e
}

// 错误处理策略
type ErrorPolicy interface {
    // 判断错误是否可以重试
    IsRetryable(err error) bool
    
    // 获取建议的重试等待时间
    RetryDelay(err error, attempt int) time.Duration
    
    // 执行降级策略
    Fallback(err error) (any, error)
}

// 实现 error 接口
func (e *Error) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Message)
}
