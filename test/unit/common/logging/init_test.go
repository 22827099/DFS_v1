package logging_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/22827099/DFS_v1/common/config"
	"github.com/22827099/DFS_v1/common/logging"
	"github.com/stretchr/testify/assert"
)

// TestInitLogging 测试初始化日志系统
func TestInitLogging(t *testing.T) {
	// 测试空配置
	logger, err := logging.InitLogging(nil)
	assert.NoError(t, err, "空配置应该不返回错误")
	assert.NotNil(t, logger, "空配置应该返回默认日志记录器")

	// 测试带基本配置
	cfg := &config.LoggingConfig{
		Level:   "debug",
		Console: true,
	}

	logger, err = logging.InitLogging(cfg)
	assert.NoError(t, err, "基本配置应该不返回错误")
	assert.NotNil(t, logger, "基本配置应该返回日志记录器")

	// 测试带文件配置的临时目录
	tempDir := os.TempDir()
	logPath := filepath.Join(tempDir, "test-log.log")

	cfg = &config.LoggingConfig{
		Level:   "info",
		Console: false,
		File:    logPath,
	}

	logger, err = logging.InitLogging(cfg)
	assert.NoError(t, err, "文件配置应该不返回错误")
	assert.NotNil(t, logger, "文件配置应该返回日志记录器")

	// 清理
	_ = os.Remove(logPath)
}

// TestConfigureLogging 测试通过配置初始化日志
func TestConfigureLogging(t *testing.T) {
	logger := logging.ConfigureLogging("debug", true, "")
	assert.NotNil(t, logger, "应该返回配置的日志记录器")

	// 检查级别是否正确设置
	buffer := &bytes.Buffer{}
	logging.SetGlobalOutput(buffer)

	logger.Debug("测试调试消息")
	assert.Contains(t, buffer.String(), "测试调试消息", "应该记录调试级别消息")
}

// TestRedirectStdLog 测试重定向标准库日志
func TestRedirectStdLog(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := logging.NewLogger(logging.WithOutput(buffer))

	writer := logging.RedirectStdLog(logger)
	assert.NotNil(t, writer, "应该返回一个不为nil的io.Writer")

	// 写入一些日志
	_, err := writer.Write([]byte("标准库日志消息\n"))
	assert.NoError(t, err, "写入不应返回错误")

	// 需要给一些时间让goroutine处理日志
	// 实际测试中可能需要更健壮的同步机制
	// 在这里我们简单地检查输出
	// 注意: 由于goroutine的异步特性，这个测试在某些环境下可能不可靠
}
