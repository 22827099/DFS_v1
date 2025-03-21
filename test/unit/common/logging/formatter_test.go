package logging_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/stretchr/testify/assert"
)

// TestTextFormatter 测试文本格式化器
func TestTextFormatter(t *testing.T) {
	formatter := logging.NewTextFormatter()
	assert.NotNil(t, formatter, "文本格式化器不应为nil")

	// 创建日志条目
	entry := &logging.LogEntry{
		Level:     logging.LevelInfo,
		Message:   "这是一条测试消息",
		Timestamp: time.Now().UnixNano(),
		Fields: map[string]interface{}{
			"user_id": 12345,
			"action":  "login",
		},
		Caller: "main.go:123",
	}

	// 格式化并验证
	output := formatter.Format(entry)

	assert.Contains(t, output, "INFO", "格式化输出应包含日志级别")
	assert.Contains(t, output, "这是一条测试消息", "格式化输出应包含消息内容")
	assert.Contains(t, output, "user_id=12345", "格式化输出应包含字段")
	assert.Contains(t, output, "action=login", "格式化输出应包含字段")
	assert.Contains(t, output, "main.go:123", "格式化输出应包含调用者信息")
}

// TestJSONFormatter 测试JSON格式化器
func TestJSONFormatter(t *testing.T) {
	formatter := logging.NewJSONFormatter()
	assert.NotNil(t, formatter, "JSON格式化器不应为nil")

	// 创建日志条目
	entry := &logging.LogEntry{
		Level:     logging.LevelError,
		Message:   "发生错误",
		Timestamp: time.Now().UnixNano(),
		Fields: map[string]interface{}{
			"error_code": 500,
			"method":     "GET",
			"path":       "/api/users",
		},
		Caller: "handler.go:45",
	}

	// 格式化并验证
	output := formatter.Format(entry)

	// 检查是否为有效JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err, "输出应为有效的JSON")

	// 验证字段
	assert.Equal(t, "ERROR", parsed["level"], "JSON应包含正确的级别")
	assert.Equal(t, "发生错误", parsed["message"], "JSON应包含正确的消息")
	assert.Equal(t, float64(500), parsed["error_code"], "JSON应包含错误代码")
	assert.Equal(t, "GET", parsed["method"], "JSON应包含HTTP方法")
	assert.Equal(t, "/api/users", parsed["path"], "JSON应包含路径")
	assert.Equal(t, "handler.go:45", parsed["caller"], "JSON应包含调用者信息")
	assert.NotEmpty(t, parsed["time"], "JSON应包含时间戳")
}

// TestLevelToString 测试日志级别到字符串的转换
func TestLevelToString(t *testing.T) {
	tests := []struct {
		level    logging.LogLevel
		expected string
	}{
		{logging.LevelDebug, "DEBUG"},
		{logging.LevelInfo, "INFO"},
		{logging.LevelWarn, "WARN"},
		{logging.LevelError, "ERROR"},
		{logging.LevelFatal, "FATAL"},
		{logging.LogLevel(99), "UNKNOWN"}, // 无效级别
	}

	for _, tt := range tests {
		result := logging.LevelToString(tt.level)
		assert.Equal(t, tt.expected, result, "日志级别字符串表示不匹配")
	}
}

// TestStringToLevel 测试字符串到日志级别的转换
func TestStringToLevel(t *testing.T) {
	tests := []struct {
		str      string
		expected logging.LogLevel
	}{
		{"debug", logging.LevelDebug},
		{"DEBUG", logging.LevelDebug},
		{"info", logging.LevelInfo},
		{"INFO", logging.LevelInfo},
		{"warn", logging.LevelWarn},
		{"WARN", logging.LevelWarn},
		{"warning", logging.LevelWarn},
		{"WARNING", logging.LevelWarn},
		{"error", logging.LevelError},
		{"ERROR", logging.LevelError},
		{"fatal", logging.LevelFatal},
		{"FATAL", logging.LevelFatal},
		{"invalid", logging.LevelInfo}, // 默认为INFO
		{"", logging.LevelInfo},        // 默认为INFO
	}

	for _, tt := range tests {
		result := logging.StringToLevel(tt.str)
		assert.Equal(t, tt.expected, result, "字符串到日志级别的转换不匹配: "+tt.str)
	}
}
