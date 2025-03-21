package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/22827099/DFS_v1/common/config"
)

// InitLogging 初始化日志系统
func InitLogging(cfg *config.LoggingConfig) (Logger, error) {
	if cfg == nil {
		return std, nil
	}

	options := []Option{}

	// 设置日志级别
	level := StringToLevel(cfg.Level)
	options = append(options, WithLevel(level))

	// 设置控制台输出
	if cfg.Console {
		options = append(options, WithConsoleFormat(), WithColors(true))
	} else {
		options = append(options, WithJSONFormat())
	}

	// 设置文件输出
	if cfg.File != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("无法创建日志目录: %w", err)
		}

		options = append(options,
			WithFilePath(cfg.File),
			WithMaxSize(100),      // 默认单个文件最大100MB
			WithMaxBackups(5),     // 默认最多保留5个备份
			WithMaxAge(30),        // 默认最多保留30天
			WithCompression(true), // 默认压缩旧日志
		)
	}

	// 添加调用者信息
	options = append(options, WithCaller(true))

	// 创建日志记录器
	logger := NewLogger(options...)

	// 设置为全局默认日志记录器
	std = logger

	return logger, nil
}

// ConfigureLogging 通过配置初始化日志系统
func ConfigureLogging(level string, console bool, file string) Logger {
	cfg := &config.LoggingConfig{
		Level:   level,
		Console: console,
		File:    file,
	}

	logger, err := InitLogging(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志系统失败: %v\n", err)
		// 回退到标准日志
		return std
	}

	return logger
}

// RedirectStdLog 重定向标准库日志到我们的日志系统
func RedirectStdLog(logger Logger) io.Writer {
	// 创建一个管道，用于重定向
	r, w := io.Pipe()

	// 启动一个协程来处理写入的日志
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				logger.Error("读取标准日志失败: %v", err)
				return
			}

			if n > 0 {
				logger.Info("%s", string(buf[:n]))
			}
		}
	}()

	return w
}
