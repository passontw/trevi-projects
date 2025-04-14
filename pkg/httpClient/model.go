package httpClient

import (
	"time"
)

const (
	// HTTP狀態碼
	StatusOK                  = 200
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusInternalServerError = 500

	// HTTP頭部常量
	HeaderTraceID       = "X-Trace-ID"
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderAuthorization = "Authorization"

	// 內容類型
	ContentTypeJSON    = "application/json"
	ContentTypeFormURL = "application/x-www-form-urlencoded"
)

// StandardRequest 定義標準請求結構
type StandardRequest struct {
	TraceID     string                 `json:"traceId,omitempty"`
	RequestTime time.Time              `json:"requestTime,omitempty"`
	ServiceName string                 `json:"serviceName,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Path        string                 `json:"path,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	QueryParams map[string]string      `json:"queryParams,omitempty"`
	Body        map[string]interface{} `json:"body,omitempty"`
}

// StandardResponse 定義標準響應結構
type StandardResponse struct {
	TraceID      string                 `json:"traceId,omitempty"`
	StatusCode   int                    `json:"statusCode"`
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	ErrorCode    string                 `json:"errorCode,omitempty"`
	Data         interface{}            `json:"data,omitempty"`
	ResponseTime time.Time              `json:"responseTime,omitempty"`
	ServiceName  string                 `json:"serviceName,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ServiceEndpoint 定義微服務端點
type ServiceEndpoint struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}
