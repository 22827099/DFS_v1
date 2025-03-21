package logging

import (
    "io"
    "sync"
)

// 全局日志记录器实例
var (
    std         Logger
    loggers     = make(map[string]Logger)
    loggerMutex sync.RWMutex
)

// LoggerFactory 接口的实现
type defaultLoggerFactory struct {
    defaultOptions []Option
}

// 全局工厂实例
var factory LoggerFactory = &defaultLoggerFactory{
    defaultOptions: []Option{
        WithLevel(LevelInfo),
        WithConsoleFormat(),
        WithCaller(true),
        WithCallerSkip(1),
        WithColors(true),
    },
}

// 初始化标准日志记录器
func init() {
    std = NewLogger()
}

// Debug 输出调试级别日志
func Debug(format string, args ...interface{}) {
    std.Debug(format, args...)
}

// Info 输出信息级别日志
func Info(format string, args ...interface{}) {
    std.Info(format, args...)
}

// Warn 输出警告级别日志
func Warn(format string, args ...interface{}) {
    std.Warn(format, args...)
}

// Error 输出错误级别日志
func Error(format string, args ...interface{}) {
    std.Error(format, args...)
}

// Fatal 输出致命错误日志
func Fatal(format string, args ...interface{}) {
    std.Fatal(format, args...)
}

// Log 输出指定级别日志
func Log(level LogLevel, format string, args ...interface{}) {
    std.Log(level, format, args...)
}

// LogWithFields 输出带有字段的日志
func LogWithFields(level LogLevel, msg string, fields map[string]interface{}) {
    std.LogWithFields(level, msg, fields)
}

// NewLogger 创建新的日志记录器
func NewLogger(options ...Option) Logger {
    // 创建基本配置
    config := NewLogConfig()
    
    // 应用全局默认选项
    for _, option := range factory.(*defaultLoggerFactory).defaultOptions {
        option(config)
    }
    
    // 应用自定义选项
    for _, option := range options {
        option(config)
    }
    
    // 创建日志记录器
    return NewZapLogger(config)
}

// CreateLogger 创建日志记录器
func (f *defaultLoggerFactory) CreateLogger(name string, options ...Option) Logger {
    // 先检查缓存
    loggerMutex.RLock()
    if logger, ok := loggers[name]; ok {
        loggerMutex.RUnlock()
        return logger
    }
    loggerMutex.RUnlock()
    
    // 不存在，创建新的
    allOptions := append([]Option{WithTag("logger", name)}, options...)
    logger := NewLogger(allOptions...)
    
    // 添加到缓存
    loggerMutex.Lock()
    loggers[name] = logger
    loggerMutex.Unlock()
    
    return logger
}

// GetLogger 获取指定名称的日志记录器
func GetLogger(name string) Logger {
    return factory.CreateLogger(name)
}

// SetLoggerFactory 设置全局日志工厂
func SetLoggerFactory(f LoggerFactory) {
    if f != nil {
        factory = f
    }
}

// SetDefaultOptions 设置全局默认选项
func SetDefaultOptions(options ...Option) {
    if factory, ok := factory.(*defaultLoggerFactory); ok {
        factory.defaultOptions = options
    }
}

// SetGlobalLevel 设置全局默认日志级别
func SetGlobalLevel(level LogLevel) {
    if l, ok := std.(*ZapLogger); ok {
        l.SetLevel(level)
    }
    
    // 更新所有已创建的日志记录器
    loggerMutex.RLock()
    defer loggerMutex.RUnlock()
    
    for _, logger := range loggers {
        if l, ok := logger.(*ZapLogger); ok {
            l.SetLevel(level)
        }
    }
}

// SetGlobalOutput 设置全局输出
func SetGlobalOutput(w io.Writer) {
    if l, ok := std.(*ZapLogger); ok {
        l.SetOutput(w)
    }
    
    // 更新所有已创建的日志记录器
    loggerMutex.RLock()
    defer loggerMutex.RUnlock()
    
    for _, logger := range loggers {
        if l, ok := logger.(*ZapLogger); ok {
            l.SetOutput(w)
        }
    }
}

// SyncAll 刷新所有日志缓冲
func SyncAll() {
    if l, ok := std.(*ZapLogger); ok {
        l.Sync()
    }
    
    loggerMutex.RLock()
    defer loggerMutex.RUnlock()
    
    for _, logger := range loggers {
        if l, ok := logger.(*ZapLogger); ok {
            l.Sync()
        }
    }
}