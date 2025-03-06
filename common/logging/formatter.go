package logging

import (
	"fmt"
	"time"
)

// LogEntry 表示一条日志条目
type LogEntry struct {
	Time    time.Time
	Level   LogLevel
	Message string
}

// Formatter 日志格式化接口
type Formatter interface {
	Format(entry *LogEntry) string
}

// DefaultFormatter 默认格式化器
type DefaultFormatter struct {
	TimeFormat string
	Colors     bool
}

// 颜色常量
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
)

// NewDefaultFormatter 创建默认格式化器
func NewDefaultFormatter() Formatter {
	return &DefaultFormatter{
		TimeFormat: "2006-01-02 15:04:05",
		Colors:     true,
	}
}

// Format 格式化日志条目
func (f *DefaultFormatter) Format(entry *LogEntry) string {
	levelName := levelNames[entry.Level]
	timeStr := entry.Time.Format(f.TimeFormat)

	if !f.Colors {
		return fmt.Sprintf("[%s] [%s] %s", timeStr, levelName, entry.Message)
	}

	// 为不同级别使用不同颜色
	var colorCode string
	switch entry.Level {
	case DEBUG:
		colorCode = colorBlue
	case INFO:
		colorCode = colorGreen
	case WARN:
		colorCode = colorYellow
	case ERROR, FATAL:
		colorCode = colorRed
	}

	return fmt.Sprintf("[%s] %s[%s]%s %s", timeStr, colorCode, levelName, colorReset, entry.Message)
}

// JSONFormatter JSON格式化器
type JSONFormatter struct{}

// NewJSONFormatter 创建JSON格式化器
func NewJSONFormatter() Formatter {
	return &JSONFormatter{}
}

// Format 以JSON格式输出日志
func (f *JSONFormatter) Format(entry *LogEntry) string {
	return fmt.Sprintf(`{"time":"%s","level":"%s","message":"%s"}`,
		entry.Time.Format(time.RFC3339),
		levelNames[entry.Level],
		entry.Message,
	)
}
