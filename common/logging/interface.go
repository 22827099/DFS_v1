package logging

import (
	"io"

	"github.com/22827099/DFS_v1/common/types"
)

// LogLevel 定义日志级别
type LogLevel int

// 日志级别常量
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// Logger 定义统一的日志接口
type Logger interface {
	// 基本日志方法
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})

	// 带结构化字段的日志方法
	DebugWithFields(msg string, fields map[string]interface{})
	InfoWithFields(msg string, fields map[string]interface{})
	WarnWithFields(msg string, fields map[string]interface{})
	ErrorWithFields(msg string, fields map[string]interface{})
	FatalWithFields(msg string, fields map[string]interface{})

	// 通用方法
	Log(level LogLevel, format string, args ...interface{})
	LogWithFields(level LogLevel, msg string, fields map[string]interface{})
	LogWithNodeID(nodeID types.NodeID, level LogLevel, format string, args ...interface{})

	// 配置方法
	WithContext(ctx map[string]interface{}) Logger
	WithNodeID(nodeID types.NodeID) Logger
	WithName(name string) Logger
	SetLevel(level LogLevel)
	SetOutput(w io.Writer)

	// 实用方法
	Sync() error
}

// LoggerFactory 定义日志工厂接口
type LoggerFactory interface {
	CreateLogger(name string, options ...Option) Logger
}

// Formatter 定义日志格式化接口
type Formatter interface {
	Format(entry *LogEntry) string
}

// LogEntry 表示一条日志条目
type LogEntry struct {
	Level     LogLevel
	Message   string
	Fields    map[string]interface{}
	Timestamp int64
	Caller    string
}
