package healthcheck

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// PingContexter 定義了數據庫和其他服務的 PingContext 接口
type PingContexter interface {
	PingContext(ctx context.Context) error
}

// DatabaseChecker 檢查數據庫連接健康狀況
type DatabaseChecker struct {
	Name_   string
	DB      PingContexter
	Timeout time.Duration
}

// Name 返回檢查器的名稱
func (d *DatabaseChecker) Name() string {
	if d.Name_ != "" {
		return d.Name_
	}
	return "database-checker"
}

// Check 執行數據庫健康檢查
func (d *DatabaseChecker) Check(r *http.Request) error {
	if d.DB == nil {
		return fmt.Errorf("數據庫未配置")
	}

	timeout := d.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	return d.DB.PingContext(ctx)
}

// RedisChecker 檢查 Redis 連接健康狀況
type RedisChecker struct {
	Name_    string
	PingFunc func(ctx context.Context) error
	Timeout  time.Duration
}

// Name 返回檢查器的名稱
func (rc *RedisChecker) Name() string {
	if rc.Name_ != "" {
		return rc.Name_
	}
	return "redis-checker"
}

// Check 執行 Redis 健康檢查
func (rc *RedisChecker) Check(r *http.Request) error {
	if rc.PingFunc == nil {
		return fmt.Errorf("Redis Ping 函數未配置")
	}

	timeout := rc.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// 使用提供的函數執行 ping
	return rc.PingFunc(ctx)
}

// HTTPChecker 檢查外部 HTTP 服務健康狀況
type HTTPChecker struct {
	Name_          string
	URL            string
	Client         *http.Client
	Timeout        time.Duration
	ExpectedStatus int
}

// Name 返回檢查器的名稱
func (h *HTTPChecker) Name() string {
	if h.Name_ != "" {
		return h.Name_
	}
	return fmt.Sprintf("http-checker-%s", h.URL)
}

// Check 執行 HTTP 服務健康檢查
func (h *HTTPChecker) Check(r *http.Request) error {
	if h.URL == "" {
		return fmt.Errorf("URL 未配置")
	}

	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}

	timeout := h.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", h.URL, nil)
	if err != nil {
		return fmt.Errorf("創建請求失敗: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("連接到 %s 失敗: %w", h.URL, err)
	}
	defer resp.Body.Close()

	expectedStatus := h.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("非預期的狀態碼: %d，預期: %d", resp.StatusCode, expectedStatus)
	}

	return nil
}

// CompositeChecker 組合多個檢查器的結果
type CompositeChecker struct {
	Name_    string
	Checkers []Checker
}

// Name 返回檢查器的名稱
func (c *CompositeChecker) Name() string {
	if c.Name_ != "" {
		return c.Name_
	}
	return "composite-checker"
}

// Check 執行所有組合的健康檢查
func (c *CompositeChecker) Check(r *http.Request) error {
	for _, checker := range c.Checkers {
		if err := checker.Check(r); err != nil {
			return fmt.Errorf("%s: %w", checker.Name(), err)
		}
	}
	return nil
}

// CustomChecker 自定義健康檢查器，使用提供的函數執行檢查
type CustomChecker struct {
	Name_     string
	CheckFunc func(r *http.Request) error
}

// Name 返回檢查器的名稱
func (c *CustomChecker) Name() string {
	if c.Name_ != "" {
		return c.Name_
	}
	return "custom-checker"
}

// Check 使用自定義函數執行健康檢查
func (c *CustomChecker) Check(r *http.Request) error {
	if c.CheckFunc == nil {
		return fmt.Errorf("檢查函數未配置")
	}
	return c.CheckFunc(r)
}

// SQLDBChecker 是 DatabaseChecker 的便捷包裝，專門用於 SQL 數據庫
func SQLDBChecker(name string, db *sql.DB) *DatabaseChecker {
	return &DatabaseChecker{
		Name_: name,
		DB:    db,
	}
}
