package logging

import (
    "io"

    "github.com/22827099/DFS_v1/common/types"
)

// Option 定义日志配置选项
type Option func(*LogConfig)

// LogConfig 定义日志配置
type LogConfig struct {
    // 基本配置
    Name      string
    Level     LogLevel
    Output    io.Writer
    Formatter Formatter
    
    // 高级配置
    UseJSON    bool
    TimeFormat string
    Colors     bool
    AddCaller  bool
    CallerSkip int
    
    // 文件配置
    FilePath       string
    MaxSize        int  // MB
    MaxBackups     int
    MaxAge         int  // 天
    Compress       bool
    LocalTime      bool
    
    // 默认标签
    DefaultTags map[string]interface{}
    NodeID      types.NodeID
}

// NewLogConfig 创建默认日志配置
func NewLogConfig() *LogConfig {
    return &LogConfig{
        Level:        LevelInfo,
        TimeFormat:   "2006-01-02 15:04:05.000",
        Colors:       true,
        AddCaller:    true,
        CallerSkip:   1,
        MaxSize:      100,
        MaxBackups:   5,
        MaxAge:       30,
        Compress:     true,
        LocalTime:    true,
        DefaultTags:  make(map[string]interface{}),
    }
}

// 以下是各种配置选项

// WithLevel 设置日志级别
func WithLevel(level LogLevel) Option {
    return func(cfg *LogConfig) {
        cfg.Level = level
    }
}

// WithOutput 设置输出位置
func WithOutput(w io.Writer) Option {
    return func(cfg *LogConfig) {
        cfg.Output = w
    }
}

// WithFormatter 设置格式化器
func WithFormatter(formatter Formatter) Option {
    return func(cfg *LogConfig) {
        cfg.Formatter = formatter
    }
}

// WithJSONFormat 启用JSON格式
func WithJSONFormat() Option {
    return func(cfg *LogConfig) {
        cfg.UseJSON = true
    }
}

// WithConsoleFormat 启用控制台格式
func WithConsoleFormat() Option {
    return func(cfg *LogConfig) {
        cfg.UseJSON = false
    }
}

// WithTimeFormat 设置时间格式
func WithTimeFormat(format string) Option {
    return func(cfg *LogConfig) {
        cfg.TimeFormat = format
    }
}

// WithColors 设置是否使用彩色输出
func WithColors(enabled bool) Option {
    return func(cfg *LogConfig) {
        cfg.Colors = enabled
    }
}

// WithCaller 设置是否添加调用者信息
func WithCaller(enabled bool) Option {
    return func(cfg *LogConfig) {
        cfg.AddCaller = enabled
    }
}

// WithCallerSkip 设置调用栈跳过层数
func WithCallerSkip(skip int) Option {
    return func(cfg *LogConfig) {
        cfg.CallerSkip = skip
    }
}

// WithFilePath 设置日志文件路径
func WithFilePath(path string) Option {
    return func(cfg *LogConfig) {
        cfg.FilePath = path
    }
}

// WithMaxSize 设置单个日志文件最大大小(MB)
func WithMaxSize(size int) Option {
    return func(cfg *LogConfig) {
        cfg.MaxSize = size
    }
}

// WithMaxBackups 设置最大备份文件数
func WithMaxBackups(count int) Option {
    return func(cfg *LogConfig) {
        cfg.MaxBackups = count
    }
}

// WithMaxAge 设置日志文件最大保留天数
func WithMaxAge(days int) Option {
    return func(cfg *LogConfig) {
        cfg.MaxAge = days
    }
}

// WithCompression 设置是否压缩旧日志
func WithCompression(compress bool) Option {
    return func(cfg *LogConfig) {
        cfg.Compress = compress
    }
}

// WithTag 添加固定标签
func WithTag(key string, value interface{}) Option {
    return func(cfg *LogConfig) {
        cfg.DefaultTags[key] = value
    }
}

// WithNodeID 设置节点ID
func WithNodeID(nodeID types.NodeID) Option {
    return func(cfg *LogConfig) {
        cfg.NodeID = nodeID
        cfg.DefaultTags["node_id"] = string(nodeID)
    }
}