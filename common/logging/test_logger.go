package logging

import (
    "io"
    "testing"

    "github.com/22827099/DFS_v1/common/types"
)

// TestLogger 实现了基于testing.T的日志记录器
type TestLogger struct {
    t      *testing.T
    level  LogLevel
    output io.Writer
    name   string
}

// NewTestLogger 创建测试日志记录器
func NewTestLogger(t *testing.T) Logger {
    return &TestLogger{
        t:     t,
        level: LevelDebug,
    }
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

// Fatal 记录致命错误日志
func (l *TestLogger) Fatal(format string, args ...interface{}) {
    l.t.Helper()
    l.t.Fatalf("[FATAL] "+format, args...)
}

// DebugWithFields 记录带有字段的调试日志
func (l *TestLogger) DebugWithFields(msg string, fields map[string]interface{}) {
    if l.level <= LevelDebug {
        l.t.Helper()
        l.t.Logf("[DEBUG] %s %v", msg, fields)
    }
}

// InfoWithFields 记录带有字段的信息日志
func (l *TestLogger) InfoWithFields(msg string, fields map[string]interface{}) {
    if l.level <= LevelInfo {
        l.t.Helper()
        l.t.Logf("[INFO] %s %v", msg, fields)
    }
}

// WarnWithFields 记录带有字段的警告日志
func (l *TestLogger) WarnWithFields(msg string, fields map[string]interface{}) {
    if l.level <= LevelWarn {
        l.t.Helper()
        l.t.Logf("[WARN] %s %v", msg, fields)
    }
}

// ErrorWithFields 记录带有字段的错误日志
func (l *TestLogger) ErrorWithFields(msg string, fields map[string]interface{}) {
    if l.level <= LevelError {
        l.t.Helper()
        l.t.Logf("[ERROR] %s %v", msg, fields)
    }
}

// FatalWithFields 记录带有字段的致命错误日志
func (l *TestLogger) FatalWithFields(msg string, fields map[string]interface{}) {
    l.t.Helper()
    l.t.Fatalf("[FATAL] %s %v", msg, fields)
}

// Log 记录指定级别的日志
func (l *TestLogger) Log(level LogLevel, format string, args ...interface{}) {
    // prefix := "[" + LevelToString(level) + "] "
    
    switch level {
    case LevelDebug:
        l.Debug(format, args...)
    case LevelInfo:
        l.Info(format, args...)
    case LevelWarn:
        l.Warn(format, args...)
    case LevelError:
        l.Error(format, args...)
    case LevelFatal:
        l.Fatal(format, args...)
    default:
        l.Info(format, args...)
    }
}

// LogWithFields 记录带有字段的日志
func (l *TestLogger) LogWithFields(level LogLevel, msg string, fields map[string]interface{}) {
    switch level {
    case LevelDebug:
        l.DebugWithFields(msg, fields)
    case LevelInfo:
        l.InfoWithFields(msg, fields)
    case LevelWarn:
        l.WarnWithFields(msg, fields)
    case LevelError:
        l.ErrorWithFields(msg, fields)
    case LevelFatal:
        l.FatalWithFields(msg, fields)
    default:
        l.InfoWithFields(msg, fields)
    }
}

// LogWithNodeID 记录带有节点ID的日志
func (l *TestLogger) LogWithNodeID(nodeID types.NodeID, level LogLevel, format string, args ...interface{}) {
    prefix := "[节点:" + string(nodeID) + "] "
    l.Log(level, prefix+format, args...)
}

// WithContext 创建带有上下文的日志记录器
func (l *TestLogger) WithContext(ctx map[string]interface{}) Logger {
    // 在测试记录器中简单返回自身
    return l
}

// WithNodeID 创建带有节点ID的日志记录器
func (l *TestLogger) WithNodeID(nodeID types.NodeID) Logger {
    return l
}

// WithName 创建带有名称的日志记录器
func (l *TestLogger) WithName(name string) Logger {
    clone := *l
    clone.name = name
    return &clone
}

// SetLevel 设置日志级别
func (l *TestLogger) SetLevel(level LogLevel) {
    l.level = level
}

// SetOutput 设置日志输出
func (l *TestLogger) SetOutput(w io.Writer) {
    l.output = w
}

// Sync 同步日志
func (l *TestLogger) Sync() error {
    return nil
}