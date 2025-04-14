package httpClient

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// TraceMiddleware 是一個HTTP中間件，用於確保每個請求都有一個追蹤ID
// 如果請求頭中已經包含追蹤ID，則使用該ID
// 否則創建一個新的追蹤ID，並將其添加到請求頭和響應頭中
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 從請求頭中獲取追蹤ID
		traceID := r.Header.Get(HeaderTraceID)

		// 如果請求頭中沒有追蹤ID，則創建一個新的
		if traceID == "" {
			traceID = uuid.New().String()
			r.Header.Set(HeaderTraceID, traceID)
		}

		// 將追蹤ID添加到上下文中
		ctx := WithTraceID(r.Context(), traceID)

		// 創建一個自定義響應寫入器，用於設置響應頭
		rw := &responseWriter{ResponseWriter: w, traceID: traceID}

		// 使用新的上下文和響應寫入器調用下一個處理程序
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}

// responseWriter 是一個自定義的ResponseWriter，用於在響應頭中添加追蹤ID
type responseWriter struct {
	http.ResponseWriter
	traceID     string
	wroteHeader bool
}

// WriteHeader 重寫WriteHeader方法，在寫入頭部之前添加追蹤ID
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.wroteHeader {
		// 設置追蹤ID頭部
		rw.ResponseWriter.Header().Set(HeaderTraceID, rw.traceID)
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write 重寫Write方法，確保在寫入之前已經設置了頭部
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// TraceIDFromRequest 從HTTP請求中獲取追蹤ID
func TraceIDFromRequest(r *http.Request) string {
	return r.Header.Get(HeaderTraceID)
}

// ContextWithTraceIDFromRequest 從HTTP請求中獲取追蹤ID並添加到上下文中
func ContextWithTraceIDFromRequest(ctx context.Context, r *http.Request) context.Context {
	traceID := TraceIDFromRequest(r)
	if traceID != "" {
		return WithTraceID(ctx, traceID)
	}
	return ctx
}
