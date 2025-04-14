package httpClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// HTTPClient 定義HTTP客戶端接口
type HTTPClient interface {
	// Get 發送GET請求
	Get(ctx context.Context, serviceName, path string, queryParams map[string]string, headers map[string]string) (*StandardResponse, error)

	// Post 發送POST請求，支持JSON格式的請求體
	Post(ctx context.Context, serviceName, path string, body interface{}, headers map[string]string) (*StandardResponse, error)

	// Put 發送PUT請求，支持JSON格式的請求體
	Put(ctx context.Context, serviceName, path string, body interface{}, headers map[string]string) (*StandardResponse, error)

	// Delete 發送DELETE請求
	Delete(ctx context.Context, serviceName, path string, headers map[string]string) (*StandardResponse, error)

	// Request 發送自定義請求
	Request(ctx context.Context, req *StandardRequest) (*StandardResponse, error)

	// SetServiceEndpoint 設置或更新服務端點
	SetServiceEndpoint(endpoint ServiceEndpoint)

	// GetServiceEndpoint 獲取服務端點
	GetServiceEndpoint(serviceName string) (ServiceEndpoint, bool)
}

// Client 實現HTTPClient接口
type Client struct {
	client           *http.Client
	serviceEndpoints map[string]ServiceEndpoint
	serviceName      string // 當前服務名稱
}

// ClientOption 定義客戶端配置選項
type ClientOption func(*Client)

// WithTimeout 設置請求超時時間
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// WithServiceName 設置當前服務名稱
func WithServiceName(name string) ClientOption {
	return func(c *Client) {
		c.serviceName = name
	}
}

// WithServiceEndpoints 設置服務端點
func WithServiceEndpoints(endpoints []ServiceEndpoint) ClientOption {
	return func(c *Client) {
		for _, endpoint := range endpoints {
			c.serviceEndpoints[endpoint.Name] = endpoint
		}
	}
}

// NewClient 創建新的HTTP客戶端
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		serviceEndpoints: make(map[string]ServiceEndpoint),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// getTraceIDFromContext 從上下文中獲取追蹤ID，如果不存在則創建新的
func getTraceIDFromContext(ctx context.Context) string {
	traceID, ok := GetTraceID(ctx)
	if !ok || traceID == "" {
		// 如果上下文中沒有追蹤ID，則創建一個新的
		traceID = uuid.New().String()
	}
	return traceID
}

// Get 實現GET請求
func (c *Client) Get(ctx context.Context, serviceName, path string, queryParams map[string]string, headers map[string]string) (*StandardResponse, error) {
	req := &StandardRequest{
		TraceID:     getTraceIDFromContext(ctx),
		RequestTime: time.Now(),
		ServiceName: serviceName,
		Method:      http.MethodGet,
		Path:        path,
		Headers:     headers,
		QueryParams: queryParams,
	}
	return c.Request(ctx, req)
}

// Post 實現POST請求
func (c *Client) Post(ctx context.Context, serviceName, path string, body interface{}, headers map[string]string) (*StandardResponse, error) {
	req := &StandardRequest{
		TraceID:     getTraceIDFromContext(ctx),
		RequestTime: time.Now(),
		ServiceName: serviceName,
		Method:      http.MethodPost,
		Path:        path,
		Headers:     headers,
	}

	if body != nil {
		bodyMap, err := convertToMap(body)
		if err != nil {
			return nil, err
		}
		req.Body = bodyMap
	}

	return c.Request(ctx, req)
}

// Put 實現PUT請求
func (c *Client) Put(ctx context.Context, serviceName, path string, body interface{}, headers map[string]string) (*StandardResponse, error) {
	req := &StandardRequest{
		TraceID:     getTraceIDFromContext(ctx),
		RequestTime: time.Now(),
		ServiceName: serviceName,
		Method:      http.MethodPut,
		Path:        path,
		Headers:     headers,
	}

	if body != nil {
		bodyMap, err := convertToMap(body)
		if err != nil {
			return nil, err
		}
		req.Body = bodyMap
	}

	return c.Request(ctx, req)
}

// Delete 實現DELETE請求
func (c *Client) Delete(ctx context.Context, serviceName, path string, headers map[string]string) (*StandardResponse, error) {
	req := &StandardRequest{
		TraceID:     getTraceIDFromContext(ctx),
		RequestTime: time.Now(),
		ServiceName: serviceName,
		Method:      http.MethodDelete,
		Path:        path,
		Headers:     headers,
	}
	return c.Request(ctx, req)
}

// Request 實現自定義請求
func (c *Client) Request(ctx context.Context, req *StandardRequest) (*StandardResponse, error) {
	endpoint, ok := c.GetServiceEndpoint(req.ServiceName)
	if !ok {
		return nil, fmt.Errorf("service endpoint not found for service: %s", req.ServiceName)
	}

	// 構建URL
	reqURL, err := url.Parse(endpoint.BaseURL)
	if err != nil {
		return nil, err
	}

	// 移除開頭的斜杠
	reqURL.Path = fmt.Sprintf("%s/%s", strings.TrimSuffix(reqURL.Path, "/"), strings.TrimPrefix(req.Path, "/"))

	// 添加查詢參數
	if len(req.QueryParams) > 0 {
		q := reqURL.Query()
		for k, v := range req.QueryParams {
			q.Add(k, v)
		}
		reqURL.RawQuery = q.Encode()
	}

	var reqBody io.Reader = nil
	// 處理請求體
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// 創建HTTP請求
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL.String(), reqBody)
	if err != nil {
		return nil, err
	}

	// 設置默認頭部
	httpReq.Header.Set(HeaderContentType, ContentTypeJSON)
	httpReq.Header.Set(HeaderAccept, ContentTypeJSON)
	httpReq.Header.Set(HeaderTraceID, req.TraceID)

	// 添加自定義頭部
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 發送請求
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 讀取響應體
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析響應
	var standardResp StandardResponse
	if err := json.Unmarshal(respBody, &standardResp); err != nil {
		// 如果不是標準響應格式，則創建一個標準響應
		standardResp = StandardResponse{
			TraceID:      req.TraceID,
			StatusCode:   resp.StatusCode,
			Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
			ResponseTime: time.Now(),
			ServiceName:  req.ServiceName,
		}

		// 嘗試將響應體解析為任意JSON
		var rawData interface{}
		if jsonErr := json.Unmarshal(respBody, &rawData); jsonErr == nil {
			standardResp.Data = rawData
		} else {
			// 如果不是JSON，則保存原始響應
			standardResp.Data = string(respBody)
		}
	}

	return &standardResp, nil
}

// SetServiceEndpoint 設置或更新服務端點
func (c *Client) SetServiceEndpoint(endpoint ServiceEndpoint) {
	c.serviceEndpoints[endpoint.Name] = endpoint
}

// GetServiceEndpoint 獲取服務端點
func (c *Client) GetServiceEndpoint(serviceName string) (ServiceEndpoint, bool) {
	endpoint, ok := c.serviceEndpoints[serviceName]
	return endpoint, ok
}

// convertToMap 將對象轉換為map
func convertToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
