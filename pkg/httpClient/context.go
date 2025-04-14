package httpClient

import (
	"context"

	"github.com/google/uuid"
)

// traceIDKey 是追蹤ID在上下文中的鍵名稱
type traceIDKey struct{}

// WithTraceID 將追蹤ID添加到上下文中
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// GetTraceID 從上下文中獲取追蹤ID
func GetTraceID(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey{}).(string)
	return traceID, ok && traceID != ""
}

// EnsureTraceID 確保上下文中有追蹤ID，如果沒有則創建一個新的
func EnsureTraceID(ctx context.Context) (context.Context, string) {
	if traceID, ok := GetTraceID(ctx); ok {
		return ctx, traceID
	}

	traceID := uuid.New().String()
	return WithTraceID(ctx, traceID), traceID
}

// WithHTTPHeaders 從HTTP請求的頭部中提取追蹤ID並添加到上下文中
func WithHTTPHeaders(ctx context.Context, headers map[string]string) context.Context {
	if traceID, ok := headers[HeaderTraceID]; ok && traceID != "" {
		return WithTraceID(ctx, traceID)
	}
	return ctx
}
