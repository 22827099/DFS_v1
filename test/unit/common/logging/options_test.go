package logging_test

import (
	"bytes"
	"testing"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/common/types"
	"github.com/stretchr/testify/assert"
)

// TestNewLogConfig 测试创建默认配置
func TestNewLogConfig(t *testing.T) {
	config := logging.NewLogConfig()

	assert.Equal(t, logging.LevelInfo, config.Level, "默认级别应为INFO")
	assert.Equal(t, "2006-01-02 15:04:05.000", config.TimeFormat, "默认时间格式不匹配")
	assert.True(t, config.Colors, "默认应启用颜色")
	assert.True(t, config.AddCaller, "默认应添加调用者信息")
	assert.Equal(t, 1, config.CallerSkip, "默认调用栈跳过应为1")
	assert.Equal(t, 100, config.MaxSize, "默认文件大小限制应为100MB")
	assert.Equal(t, 5, config.MaxBackups, "默认备份数应为5")
	assert.Equal(t, 30, config.MaxAge, "默认保留天数应为30")
	assert.True(t, config.Compress, "默认应压缩旧日志")
	assert.NotNil(t, config.DefaultTags, "默认标签映射不应为nil")
}

// TestOptionApplication 测试应用选项
func TestOptionApplication(t *testing.T) {
	config := logging.NewLogConfig()

	buffer := &bytes.Buffer{}
	nodeID := types.NodeID("test-node")

	// 应用各种选项
	options := []logging.Option{
		logging.WithLevel(logging.LevelDebug),
		logging.WithOutput(buffer),
		logging.WithJSONFormat(),
		logging.WithTimeFormat("2006/01/02"),
		logging.WithColors(false),
		logging.WithCaller(false),
		logging.WithCallerSkip(2),
		logging.WithFilePath("/tmp/test.log"),
		logging.WithMaxSize(200),
		logging.WithMaxBackups(10),
		logging.WithMaxAge(60),
		logging.WithCompression(false),
		logging.WithTag("app", "test-app"),
		logging.WithNodeID(nodeID),
	}

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	// 验证选项是否已正确应用
	assert.Equal(t, logging.LevelDebug, config.Level)
	assert.Equal(t, buffer, config.Output)
	assert.True(t, config.UseJSON)
	assert.Equal(t, "2006/01/02", config.TimeFormat)
	assert.False(t, config.Colors)
	assert.False(t, config.AddCaller)
	assert.Equal(t, 2, config.CallerSkip)
	assert.Equal(t, "/tmp/test.log", config.FilePath)
	assert.Equal(t, 200, config.MaxSize)
	assert.Equal(t, 10, config.MaxBackups)
	assert.Equal(t, 60, config.MaxAge)
	assert.False(t, config.Compress)
	assert.Equal(t, "test-app", config.DefaultTags["app"])
	assert.Equal(t, nodeID, config.NodeID)
	assert.Equal(t, string(nodeID), config.DefaultTags["node_id"])
}

// TestOptionsCombination 测试组合选项创建日志记录器
func TestOptionsCombination(t *testing.T) {
	buffer := &bytes.Buffer{}

	// 创建带多个选项的日志记录器
	logger := logging.NewLogger(
		logging.WithLevel(logging.LevelDebug),
		logging.WithOutput(buffer),
		logging.WithConsoleFormat(),
		logging.WithTag("app", "test-app"),
	)

	assert.NotNil(t, logger, "带选项的日志记录器不应为nil")

	// 记录日志并验证
	logger.Debug("测试选项组合")
	output := buffer.String()

	assert.Contains(t, output, "DEBUG", "日志应包含DEBUG级别")
	assert.Contains(t, output, "测试选项组合", "日志应包含消息内容")
	assert.Contains(t, output, "test-app", "日志应包含标签值")
}
