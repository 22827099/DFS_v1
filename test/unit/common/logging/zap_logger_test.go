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

func TestZapLoggerBasic(t *testing.T) {
    // 创建带输出缓冲区的日志记录器
    buffer := &bytes.Buffer{}
    config := logging.NewLogConfig()
    config.Output = buffer
    
    logger := logging.NewZapLogger(config)
    
    // 添加此行明确设置级别为 DEBUG
    logger.SetLevel(logging.LevelDebug)
    
    // 写入各级别日志
    logger.Debug("调试信息")
    logger.Info("普通信息")
    logger.Warn("警告信息")
    logger.Error("错误信息")
    
    // 检查输出
    output := buffer.String()
    assert.Contains(t, output, "调试信息")
    assert.Contains(t, output, "普通信息")
    assert.Contains(t, output, "警告信息")
    assert.Contains(t, output, "错误信息")
}

func TestZapLoggerLevels(t *testing.T) {
    // 测试不同级别的日志过滤
    buffer := &bytes.Buffer{}
    config := logging.NewLogConfig()
    config.Output = buffer
    config.Level = logging.LevelWarn // 只输出警告及以上级别
    
    logger := logging.NewZapLogger(config)
    
    // 写入各级别日志
    logger.Debug("调试信息")
    logger.Info("普通信息")
    logger.Warn("警告信息")
    logger.Error("错误信息")
    
    // 检查输出
    output := buffer.String()
    assert.NotContains(t, output, "调试信息")
    assert.NotContains(t, output, "普通信息")
    assert.Contains(t, output, "警告信息")
    assert.Contains(t, output, "错误信息")
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
