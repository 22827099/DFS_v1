package logging_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/22827099/DFS_v1/common/logging"
	"github.com/stretchr/testify/assert"
)

// TestWithLogger 测试将日志记录器添加到上下文
func TestWithLogger(t *testing.T) {
	logger := logging.NewLogger()
	ctx := context.Background()

	// 添加日志记录器到上下文
	newCtx := logging.WithLogger(ctx, logger)
	assert.NotEqual(t, ctx, newCtx, "添加日志记录器后上下文应该改变")

	// 尝试获取日志记录器
	retrievedLogger := logging.GetLoggerFromContext(newCtx)
	assert.Equal(t, logger, retrievedLogger, "应该获取到原始的日志记录器")

	// 从空上下文获取日志记录器应返回默认日志记录器
	emptyLogger := logging.GetLoggerFromContext(context.Background())
	assert.NotNil(t, emptyLogger, "从空上下文获取的日志记录器不应为nil")
}

// TestWithTraceID 测试将跟踪ID添加到上下文
func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "trace-123456"

	// 添加跟踪ID到上下文
	newCtx := logging.WithTraceID(ctx, traceID)
	assert.NotEqual(t, ctx, newCtx, "添加跟踪ID后上下文应该改变")

	// 尝试获取跟踪ID
	retrievedTraceID := logging.GetTraceID(newCtx)
	assert.Equal(t, traceID, retrievedTraceID, "应该获取到原始的跟踪ID")

	// 从空上下文获取跟踪ID应返回空字符串
	emptyTraceID := logging.GetTraceID(context.Background())
	assert.Equal(t, "", emptyTraceID, "从空上下文获取跟踪ID应返回空字符串")
}

// TestWithRequestID 测试将请求ID添加到上下文
func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "req-789012"

	// 添加请求ID到上下文
	newCtx := logging.WithRequestID(ctx, requestID)
	assert.NotEqual(t, ctx, newCtx, "添加请求ID后上下文应该改变")

	// 尝试获取请求ID
	retrievedRequestID := logging.GetRequestID(newCtx)
	assert.Equal(t, requestID, retrievedRequestID, "应该获取到原始的请求ID")

	// 从空上下文获取请求ID应返回空字符串
	emptyRequestID := logging.GetRequestID(context.Background())
	assert.Equal(t, "", emptyRequestID, "从空上下文获取请求ID应返回空字符串")
}

// TestLoggerFromContext 测试从上下文获取日志记录器并添加上下文信息
func TestLoggerFromContext(t *testing.T) {
	// 创建带缓冲区的日志记录器
	buffer := &bytes.Buffer{}
	logger := logging.NewLogger(logging.WithOutput(buffer))

	// 创建包含日志记录器、跟踪ID和请求ID的上下文
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logger)
	ctx = logging.WithTraceID(ctx, "trace-123")
	ctx = logging.WithRequestID(ctx, "req-456")

	// 从上下文获取日志记录器
	contextLogger := logging.LoggerFromContext(ctx)
	assert.NotNil(t, contextLogger, "上下文日志记录器不应为nil")

	// 记录日志并验证是否包含上下文信息
	buffer.Reset()
	contextLogger.Info("上下文消息")
	output := buffer.String()

	assert.Contains(t, output, "上下文消息", "日志应包含消息内容")
	assert.Contains(t, output, "trace-123", "日志应包含跟踪ID")
	assert.Contains(t, output, "req-456", "日志应包含请求ID")
}

// TestLoggerFromEmptyContext 测试从空上下文获取日志记录器
func TestLoggerFromEmptyContext(t *testing.T) {
	// 从空上下文获取日志记录器应返回默认日志记录器
	emptyCtx := context.Background()
	logger := logging.LoggerFromContext(emptyCtx)
	assert.NotNil(t, logger, "从空上下文获取的日志记录器不应为nil")
}
