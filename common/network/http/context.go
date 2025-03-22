package http

import (
    "context"
)

type contextKey int

const (
    requestIDKey contextKey = iota
)

// WithRequestID 在上下文中设置请求ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID 从上下文获取请求ID
func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(requestIDKey).(string); ok {
        return id
    }
    return ""
}