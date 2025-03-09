package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Logger 定义日志记录器接口
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	SetLevel(level LogLevel)
	SetOutput(w io.Writer)
	WithContext(ctx map[string]interface{}) Logger
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	level     LogLevel
	output    io.Writer
	mu        sync.Mutex
	formatter Formatter
}

// WithContext 创建带有上下文的日志记录器
func (l *DefaultLogger) WithContext(ctx map[string]interface{}) Logger {
	// 创建新的日志记录器，复制原始记录器的设置
	newLogger := &DefaultLogger{
		level:     l.level,
		output:    l.output,
		formatter: l.formatter,
	}

	// 如果原始格式化器支持上下文，则设置上下文
	if contextFormatter, ok := newLogger.formatter.(ContextFormatter); ok {
		contextFormatter = contextFormatter.WithContext(ctx)
		newLogger.formatter = contextFormatter
	}

	return newLogger
}

// ContextFormatter 扩展基本的Formatter，支持上下文信息
type ContextFormatter interface {
	Formatter
	WithContext(ctx map[string]interface{}) ContextFormatter
}

// WithContext 创建带有上下文的格式化器
func (f *DefaultFormatter) WithContext(ctx map[string]interface{}) ContextFormatter {
	newFormatter := &DefaultFormatter{
		context: make(map[string]interface{}, len(f.context)+len(ctx)),
	}

	// 复制现有上下文
	for k, v := range f.context {
		newFormatter.context[k] = v
	}

	// 添加新上下文
	for k, v := range ctx {
		newFormatter.context[k] = v
	}

	return newFormatter
}

// NewLogger 创建新的日志记录器
func NewLogger() Logger {
	return &DefaultLogger{
		level:     INFO,
		output:    os.Stdout,
		formatter: NewDefaultFormatter(),
	}
}

// defaultLogger 提供基本的日志实现
type defaultLogger struct {
	name   string
	logger *log.Logger
	level  LogLevel
}
// GetLogger 返回指定名称的日志记录器，如果不存在则创建新的
func GetLogger(name string) Logger {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	if logger, exists := loggers[name]; exists {
		return logger
	}

	// 创建新的日志记录器
	logger := &defaultLogger{
		name:   name,
		logger: log.New(os.Stdout, fmt.Sprintf("[%s] ", name), log.LstdFlags),
		level:  INFO,
	}
	loggers[name] = logger
	return logger
}

// Debug 实现Logger接口的Debug方法
func (l *defaultLogger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		l.logger.Printf("DEBUG: "+format, args...)
	}
}

// Info 实现Logger接口的Info方法
func (l *defaultLogger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		l.logger.Printf("INFO: "+format, args...)
	}
}

// Warn 实现Logger接口的Warn方法
func (l *defaultLogger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		l.logger.Printf("WARN: "+format, args...)
	}
}

// Error 实现Logger接口的Error方法
func (l *defaultLogger) Error(format string, args ...interface{}) {
	if l.level <= ERROR {
		l.logger.Printf("ERROR: "+format, args...)
	}
}

// Fatal 实现Logger接口的Fatal方法
func (l *defaultLogger) Fatal(format string, args ...interface{}) {
	if l.level <= FATAL {
		l.logger.Printf("FATAL: "+format, args...)
		os.Exit(1)
	}
}

// SetLevel 实现Logger接口的SetLevel方法
func (l *defaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput 实现Logger接口的SetOutput方法
func (l *defaultLogger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
}

// WithContext 实现Logger接口的WithContext方法
func (l *defaultLogger) WithContext(ctx map[string] interface{}) Logger {
	// 简单实现，实际可能需要更复杂的处理
	newLogger := &defaultLogger{
		name:   l.name,
		logger: l.logger,
		level:  l.level,
	}
	return newLogger
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

// SetOutput 设置日志输出
func (l *DefaultLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// 全局日志实例
var DefaultLoggerInstance = &DefaultLogger{
	level:     INFO,
	output:    os.Stdout,
	formatter: NewDefaultFormatter(),
}

// 全局日志变量声明
var (
	loggerMutex sync.Mutex
	loggers     = make(map[string]Logger)
)

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
