package logging_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/stretchr/testify/assert"
)

// TestNewLogger 测试创建新的日志记录器
func TestNewLogger(t *testing.T) {
	logger := logging.NewLogger()
	assert.NotNil(t, logger, "新建日志记录器不应为nil")

	// 测试带选项的日志记录器
	buffer := &bytes.Buffer{}
	logger = logging.NewLogger(
		logging.WithLevel(logging.LevelDebug),
		logging.WithOutput(buffer),
		logging.WithConsoleFormat(),
	)
	assert.NotNil(t, logger, "带选项的日志记录器不应为nil")

	logger.Debug("测试消息")
	assert.True(t, strings.Contains(buffer.String(), "测试消息"), "日志消息应包含测试内容")
}

// TestGlobalLoggerFunctions 测试全局日志函数
func TestGlobalLoggerFunctions(t *testing.T) {
	// 保存原始输出并在测试后恢复
	buffer := &bytes.Buffer{}
	logging.SetGlobalOutput(buffer)

	tests := []struct {
		logFunc    func(string, ...interface{})
		levelName  string
		formatStr  string
		formatArgs []interface{}
	}{
		{logging.Debug, "DEBUG", "测试调试消息 %d", []interface{}{1}},
		{logging.Info, "INFO", "测试信息消息 %s", []interface{}{"info"}},
		{logging.Warn, "WARN", "测试警告消息", []interface{}{}},
		{logging.Error, "ERROR", "测试错误: %v", []interface{}{"错误"}},
	}

	for _, tt := range tests {
		buffer.Reset()
		tt.logFunc(tt.formatStr, tt.formatArgs...)
		output := buffer.String()
		assert.Contains(t, output, tt.levelName, "日志应包含级别: "+tt.levelName)

		// 检查格式化是否正确
		if len(tt.formatArgs) > 0 {
			expectedMsg := strings.Replace(tt.formatStr, "%d", "1", -1)
			expectedMsg = strings.Replace(expectedMsg, "%s", "info", -1)
			expectedMsg = strings.Replace(expectedMsg, "%v", "错误", -1)
			assert.Contains(t, output, expectedMsg, "日志应包含正确格式化的消息")
		} else {
			assert.Contains(t, output, tt.formatStr, "日志应包含消息内容")
		}
	}
}

// TestGetLogger 测试获取命名日志记录器
func TestGetLogger(t *testing.T) {
	logger1 := logging.GetLogger("test-logger")
	logger2 := logging.GetLogger("test-logger")
	assert.Equal(t, logger1, logger2, "相同名称应返回相同的日志记录器实例")

	logger3 := logging.GetLogger("another-logger")
	assert.NotEqual(t, logger1, logger3, "不同名称应返回不同的日志记录器实例")
}

// TestSetGlobalLevel 测试设置全局日志级别
func TestSetGlobalLevel(t *testing.T) {
	buffer := &bytes.Buffer{}
	logging.SetGlobalOutput(buffer)

	// 将级别设置为INFO
	logging.SetGlobalLevel(logging.LevelInfo)

	// Debug级别信息不应输出
	buffer.Reset()
	logging.Debug("这条消息不应该出现")
	assert.Empty(t, buffer.String(), "Debug级别的消息不应出现")

	// Info级别信息应该输出
	buffer.Reset()
	logging.Info("这条消息应该出现")
	assert.Contains(t, buffer.String(), "这条消息应该出现", "Info级别的消息应该出现")

	// 将级别重置为Debug进行后续测试
	logging.SetGlobalLevel(logging.LevelDebug)
}

// TestLoggingWithFields 测试带字段的日志记录
func TestLoggingWithFields(t *testing.T) {
	buffer := &bytes.Buffer{}
	logging.SetGlobalOutput(buffer)

	fields := map[string]interface{}{
		"user_id": 12345,
		"action":  "login",
	}

	logging.LogWithFields(logging.LevelInfo, "用户操作", fields)
	output := buffer.String()

	assert.Contains(t, output, "用户操作", "日志应包含消息内容")
	assert.Contains(t, output, "12345", "日志应包含字段值")
	assert.Contains(t, output, "login", "日志应包含字段值")
}
