package logging

import (
	"io"
	"testing"
)

// TestLogger 实现了基于testing.T的日志记录器
type TestLogger struct {
	t      *testing.T
	level  LogLevel
	output io.Writer // 添加输出字段，虽然在测试记录器中可能不会直接使用
}

// NewTestLogger 创建一个新的测试日志记录器
func NewTestLogger(t *testing.T) Logger {
	return &TestLogger{
		t:     t,
		level: LevelDebug, // 测试时默认使用最详细的日志级别
	}
}

// SetLevel 设置日志记录器的级别
func (l *TestLogger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput 设置日志输出目标
// 注意：在测试记录器中，这个方法可能不会有实际效果
// 因为测试日志总是输出到testing.T
func (l *TestLogger) SetOutput(w io.Writer) {
	l.output = w
}

// Debug 记录调试级别的日志
func (l *TestLogger) Debug(format string, args ...interface{}) {
	if l.level <= LevelDebug {
		l.t.Helper()
		l.t.Logf("[DEBUG] "+format, args...)
	}
}

// Info 记录信息级别的日志
func (l *TestLogger) Info(format string, args ...interface{}) {
	if l.level <= LevelInfo {
		l.t.Helper()
		l.t.Logf("[INFO] "+format, args...)
	}
}

// Warn 记录警告级别的日志
func (l *TestLogger) Warn(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		l.t.Helper()
		l.t.Logf("[WARN] "+format, args...)
	}
}

// Error 记录错误级别的日志
func (l *TestLogger) Error(format string, args ...interface{}) {
	if l.level <= LevelError {
		l.t.Helper()
		l.t.Logf("[ERROR] "+format, args...)
	}
}

// Fatal 记录致命级别的日志并终止测试
func (l *TestLogger) Fatal(format string, args ...interface{}) {
	l.t.Helper()
	l.t.Fatalf("[FATAL] "+format, args...)
}

// WithContext 创建带有上下文的日志记录器
func (l *TestLogger) WithContext(ctx map[string]interface{}) Logger {
	// 在测试记录器中简单返回自身
	// 更复杂的实现可能会创建新的记录器并保存上下文
	return l
}
