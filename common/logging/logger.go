package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// Logger 日志接口
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	SetLevel(level LogLevel)
	SetOutput(w io.Writer)
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	level     LogLevel
	output    io.Writer
	mu        sync.Mutex
	formatter Formatter
}

// NewLogger 创建新的日志记录器
func NewLogger() Logger {
	return &DefaultLogger{
		level:     INFO,
		output:    os.Stdout,
		formatter: NewDefaultFormatter(),
	}
}

// SetFormatter 设置日志格式化器
func (l *DefaultLogger) SetFormatter(formatter Formatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.formatter = formatter
}

// SetLevel 设置日志级别
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput 设置输出
func (l *DefaultLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// log 内部日志方法
func (l *DefaultLogger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	entry := &LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: fmt.Sprintf(format, args...),
	}

	formatted := l.formatter.Format(entry)
	fmt.Fprintln(l.output, formatted)

	// 如果是致命错误，程序退出
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 记录调试日志
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 记录信息日志
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 记录警告日志
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 记录错误日志
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal 记录致命错误日志
func (l *DefaultLogger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// 全局日志实例
var std = NewLogger()

// 提供全局日志方法
func Debug(format string, args ...interface{}) {
	std.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	std.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	std.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	std.Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	std.Fatal(format, args...)
}
