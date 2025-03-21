package logging

import (
    "context"
)

// contextKey 是用于存储在context中的键类型
type contextKey int

const (
    // loggerKey 用于在context中存储日志记录器
    loggerKey contextKey = iota
    
    // traceIDKey 用于在context中存储跟踪ID
    traceIDKey
    
    // requestIDKey 用于在context中存储请求ID
    requestIDKey
)

// WithLogger 将日志记录器添加到context
func WithLogger(ctx context.Context, logger Logger) context.Context {
    return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger 从context获取日志记录器
func GetLoggerFromContext(ctx context.Context) Logger {
    if logger, ok := ctx.Value(loggerKey).(Logger); ok {
        return logger
    }
    return std
}

// WithTraceID 将跟踪ID添加到context
func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID 从context获取跟踪ID
func GetTraceID(ctx context.Context) string {
    if traceID, ok := ctx.Value(traceIDKey).(string); ok {
        return traceID
    }
    return ""
}

// WithRequestID 将请求ID添加到context
func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID 从context获取请求ID
func GetRequestID(ctx context.Context) string {
    if requestID, ok := ctx.Value(requestIDKey).(string); ok {
        return requestID
    }
    return ""
}

// LoggerFromContext 从context获取日志记录器并添加上下文信息
func LoggerFromContext(ctx context.Context) Logger {
    logger := GetLoggerFromContext(ctx)
    
    fields := make(map[string]interface{})
    
    // 添加跟踪ID
    if traceID := GetTraceID(ctx); traceID != "" {
        fields["trace_id"] = traceID
    }
    
    // 添加请求ID
    if requestID := GetRequestID(ctx); requestID != "" {
        fields["request_id"] = requestID
    }
    
    if len(fields) > 0 {
        return logger.WithContext(fields)
    }
    
    return logger
}