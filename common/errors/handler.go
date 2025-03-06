package errors

import (
    "encoding/json"
    "fmt"
    "runtime"
    "strings"
    "time"
)

// Error 定义系统错误结构
type Error struct {
    Code     ErrorCode       // 错误码
    Message  string          // 错误消息
    Metadata map[string]any  // 附加数据
    Cause    error           // 原始错误
    Stack    string          // 堆栈信息
}

// New 创建一个新的系统错误
func New(code ErrorCode, args ...any) *Error {
    var message string
    if len(args) > 0 {
        if msgStr, ok := args[0].(string); ok {
            if len(args) > 1 {
                message = fmt.Sprintf(msgStr, args[1:]...)
            } else {
                message = msgStr
            }
        }
    }

    return &Error{
        Code:    code,
        Message: message,
        Stack:   captureStack(2),
    }
}

// Wrap 包装一个已有错误
func Wrap(err error, code ErrorCode, args ...any) *Error {
    if err == nil {
        return New(code, args...)
    }

    var message string
    if len(args) > 0 {
        if msgStr, ok := args[0].(string); ok {
            if len(args) > 1 {
                message = fmt.Sprintf(msgStr, args[1:]...)
            } else {
                message = msgStr
            }
        }
    }

    // 如果已经是我们的错误类型，只更新消息和代码（如果提供）
    if e, ok := err.(*Error); ok {
        if code != Unknown && e.Code == Unknown {
            e.Code = code
        }
        if message != "" {
            e.Message = message
        }
        return e
    }

    return &Error{
        Code:    code,
        Message: message,
        Cause:   err,
        Stack:   captureStack(2),
    }
}

// Error 实现error接口
func (e *Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// GetCode 从错误中提取错误码
func GetCode(err error) ErrorCode {
    if err == nil {
        return 0
    }

    if e, ok := err.(*Error); ok {
        return e.Code
    }

    return Unknown
}

// Is 检查错误类型
func Is(err error, code ErrorCode) bool {
    return GetCode(err) == code
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
        // 使用更通用的过滤方式
        if !strings.Contains(frame.File, "DFS_v1") && 
           !strings.Contains(frame.File, "BYSJ/Project") {
            if !more {
                break
            }
            continue // 跳过非项目代码
        }
        fmt.Fprintf(&builder, "%s:%d %s\n", frame.File, frame.Line, frame.Function)
        if !more {
            break
        }
    }
    return builder.String()
}

// 错误类型检查函数
func IsNotFound(err error) bool {
    e, ok := err.(*Error)
    return ok && e.Code == ErrorCode(ErrFileNotFound)
}

func IsPermissionDenied(err error) bool {
    e, ok := err.(*Error)
    return ok && e.Code == ErrorCode(ErrPermission)
}

func IsAlreadyExists(err error) bool {
    e, ok := err.(*Error)
    return ok && e.Code == ErrorCode(ErrFileAlreadyExists)
}

// 通用错误码检查函数
func IsErrorCode(err error, code ErrorCode) bool {
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
    switch int(e.Code) / 1000 {
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
        Code     ErrorCode     `json:"code"`
        Message  string        `json:"message"`
        Metadata map[string]any `json:"metadata,omitempty"`
        Cause    string        `json:"cause,omitempty"`
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
    DomainMetadata = "metadata"
    DomainStorage  = "storage"
    DomainNetwork  = "network"
    DomainSecurity = "security"
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