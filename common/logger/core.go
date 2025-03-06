package logger

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Logger struct {
	mu       sync.Mutex
	level    string
	format   string
	output   string
	filePath string
}

// 新建日志实例
func NewLogger(config Config) *Logger {
	return &Logger{
		level:    config.Level,
		format:   config.Format,
		output:   config.Output,
		filePath: config.FilePath,
	}
}

// 写入日志
func (l *Logger) log(level, message string, fields map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level == l.level {
		logData := map[string]interface{}{
			"level":     level,
			"message":   message,
			"timestamp": time.Now().Format(time.RFC3339Nano),
			"file":      getCallerInfo(),
			"traceID":   "trace-id-placeholder", // 可选的分布式追踪ID
		}

		// 添加其他上下文信息
		for k, v := range fields {
			logData[k] = v
		}

		if l.format == "json" {
			jsonLog, _ := json.Marshal(logData)
			fmt.Println(string(jsonLog)) // 输出JSON格式
		} else {
			fmt.Println(logData) // 输出Text格式
		}
	}
}

// 获取调用者信息（文件名 + 行号）
func getCallerInfo() string {
	_, file, line, _ := runtime.Caller(2)
	return fmt.Sprintf("%s:%d", file, line)
}
