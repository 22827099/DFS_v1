package logging_test

import (
	"bytes"
	"testing"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/22827099/DFS_v1/common/types"
	"github.com/stretchr/testify/assert"
)

// TestZapLoggerCreation 测试ZapLogger的创建
func TestZapLoggerCreation(t *testing.T) {
	// 测试默认配置
	config := logging.NewLogConfig()
	logger := logging.NewZapLogger(config)
	assert.NotNil(t, logger, "使用默认配置创建的ZapLogger不应为nil")

	// 测试空配置
	logger = logging.NewZapLogger(nil)
	assert.NotNil(t, logger, "使用nil配置创建的ZapLogger不应为nil")
}

// TestZapLoggerBasicLogging 测试基本日志方法
func TestZapLoggerBasicLogging(t *testing.T) {
	// 创建带输出缓冲区的日志记录器
	buffer := &bytes.Buffer{}
	config := logging.NewLogConfig()
	config.Output = buffer

	logger := logging.NewZapLogger(config)

	// 测试各种级别的日志
	tests := []struct {
		logFunc   func(string, ...interface{})
		levelName string
		message   string
	}{
		{logger.Debug, "DEBUG", "调试消息"},
		{logger.Info, "INFO", "信息消息"},
		{logger.Warn, "WARN", "警告消息"},
		{logger.Error, "ERROR", "错误消息"},
	}

	for _, tt := range tests {
		buffer.Reset()
		tt.logFunc(tt.message)
		output := buffer.String()
		assert.Contains(t, output, tt.levelName, "日志应包含级别:"+tt.levelName)
		assert.Contains(t, output, tt.message, "日志应包含消息内容")
	}
}

// TestZapLoggerWithFields 测试带字段的日志
func TestZapLoggerWithFields(t *testing.T) {
	buffer := &bytes.Buffer{}
	config := logging.NewLogConfig()
	config.Output = buffer

	logger := logging.NewZapLogger(config)

	fields := map[string]interface{}{
		"user_id": 12345,
		"action":  "login",
	}

	// 测试各级别带字段的日志方法
	tests := []struct {
		logFunc   func(string, map[string]interface{})
		levelName string
		message   string
	}{
		{logger.DebugWithFields, "DEBUG", "调试字段消息"},
		{logger.InfoWithFields, "INFO", "信息字段消息"},
		{logger.WarnWithFields, "WARN", "警告字段消息"},
		{logger.ErrorWithFields, "ERROR", "错误字段消息"},
	}

	for _, tt := range tests {
		buffer.Reset()
		tt.logFunc(tt.message, fields)
		output := buffer.String()

		assert.Contains(t, output, tt.levelName, "日志应包含级别:"+tt.levelName)
		assert.Contains(t, output, tt.message, "日志应包含消息内容")
		assert.Contains(t, output, "12345", "日志应包含字段值")
		assert.Contains(t, output, "login", "日志应包含字段值")
	}
}

// TestZapLoggerWithContext 测试带上下文的日志
func TestZapLoggerWithContext(t *testing.T) {
	buffer := &bytes.Buffer{}
	config := logging.NewLogConfig()
	config.Output = buffer

	logger := logging.NewZapLogger(config)

	context := map[string]interface{}{
		"request_id": "req-123",
		"session":    "sess-456",
	}

	contextLogger := logger.WithContext(context)
	assert.NotNil(t, contextLogger, "上下文日志记录器不应为nil")
	assert.NotEqual(t, logger, contextLogger, "上下文日志记录器应该是新实例")

	buffer.Reset()
	contextLogger.Info("带上下文的消息")
	output := buffer.String()

	assert.Contains(t, output, "带上下文的消息", "日志应包含消息内容")
	assert.Contains(t, output, "req-123", "日志应包含上下文字段")
	assert.Contains(t, output, "sess-456", "日志应包含上下文字段")
}

// TestZapLoggerWithNodeID 测试带节点ID的日志
func TestZapLoggerWithNodeID(t *testing.T) {
	buffer := &bytes.Buffer{}
	config := logging.NewLogConfig()
	config.Output = buffer

	logger := logging.NewZapLogger(config)

	nodeID := types.NodeID("node-123")

	// 测试直接使用节点ID记录日志
	buffer.Reset()
	logger.LogWithNodeID(nodeID, logging.LevelInfo, "节点消息")
	output := buffer.String()
	assert.Contains(t, output, "node-123", "日志应包含节点ID")
	assert.Contains(t, output, "节点消息", "日志应包含消息内容")

	// 测试使用WithNodeID创建新的日志记录器
	nodeLogger := logger.WithNodeID(nodeID)
	assert.NotNil(t, nodeLogger, "节点日志记录器不应为nil")

	buffer.Reset()
	nodeLogger.Info("另一个节点消息")
	output = buffer.String()
	assert.Contains(t, output, "node-123", "日志应包含节点ID")
	assert.Contains(t, output, "另一个节点消息", "日志应包含消息内容")
}

// TestZapLoggerSetLevel 测试设置日志级别
func TestZapLoggerSetLevel(t *testing.T) {
	buffer := &bytes.Buffer{}
	config := logging.NewLogConfig()
	config.Output = buffer

	logger := logging.NewZapLogger(config)

	// 设置为INFO级别
	logger.SetLevel(logging.LevelInfo)

	// Debug消息不应出现
	buffer.Reset()
	logger.Debug("调试消息")
	assert.Empty(t, buffer.String(), "Debug级别的消息不应出现")

	// Info消息应出现
	buffer.Reset()
	logger.Info("信息消息")
	assert.Contains(t, buffer.String(), "信息消息", "Info级别的消息应出现")

	// 还原级别
	logger.SetLevel(logging.LevelDebug)
}
