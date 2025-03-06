package logger

import "runtime/debug"

// 错误堆栈捕获
func CaptureStackTrace() string {
	return string(debug.Stack())
}
