package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 測試 PingChecker
func TestPingChecker(t *testing.T) {
	checker := &PingChecker{}

	// 名稱應該是 ping
	if checker.Name() != "ping" {
		t.Errorf("Expected name 'ping', got '%s'", checker.Name())
	}

	// 檢查應該始終通過
	err := checker.Check(httptest.NewRequest("GET", "/ping", nil))
	if err != nil {
		t.Errorf("PingChecker.Check() should not return error, got: %v", err)
	}
}

// 測試 ReadinessStateChecker
func TestReadinessStateChecker(t *testing.T) {
	// 創建健康檢查管理器
	manager := New(Config{})

	// 創建檢查器
	checker := &ReadinessStateChecker{manager: manager}

	// 名稱應該是 readiness-state
	if checker.Name() != "readiness-state" {
		t.Errorf("Expected name 'readiness-state', got '%s'", checker.Name())
	}

	// 未就緒狀態下的檢查應該失敗
	err := checker.Check(httptest.NewRequest("GET", "/ready", nil))
	if err == nil {
		t.Error("ReadinessStateChecker.Check() should return error when not ready")
	}

	// 設置為就緒
	manager.SetReady(true)

	// 就緒狀態下的檢查應該通過
	err = checker.Check(httptest.NewRequest("GET", "/ready", nil))
	if err != nil {
		t.Errorf("ReadinessStateChecker.Check() should not return error when ready, got: %v", err)
	}
}

// 測試 CustomChecker
func TestCustomChecker(t *testing.T) {
	// 創建一個始終通過的檢查器
	passChecker := &CustomChecker{
		Name_: "pass-checker",
		CheckFunc: func(r *http.Request) error {
			return nil
		},
	}

	// 檢查名稱
	if passChecker.Name() != "pass-checker" {
		t.Errorf("Expected name 'pass-checker', got '%s'", passChecker.Name())
	}

	// 檢查應該通過
	err := passChecker.Check(httptest.NewRequest("GET", "/check", nil))
	if err != nil {
		t.Errorf("CustomChecker.Check() should not return error for pass checker, got: %v", err)
	}

	// 創建一個始終失敗的檢查器
	failChecker := &CustomChecker{
		Name_: "fail-checker",
		CheckFunc: func(r *http.Request) error {
			return errors.New("預期的錯誤")
		},
	}

	// 檢查應該失敗
	err = failChecker.Check(httptest.NewRequest("GET", "/check", nil))
	if err == nil {
		t.Error("CustomChecker.Check() should return error for fail checker")
	}
}

// 測試 DatabaseChecker
func TestDatabaseChecker(t *testing.T) {
	// 創建一個測試用的 PingContexter
	mockDB := &mockPinger{
		shouldFail: false,
	}

	checker := &DatabaseChecker{
		Name_: "test-db",
		DB:    mockDB,
	}

	// 檢查名稱
	if checker.Name() != "test-db" {
		t.Errorf("Expected name 'test-db', got '%s'", checker.Name())
	}

	// 檢查應該通過
	err := checker.Check(httptest.NewRequest("GET", "/check", nil))
	if err != nil {
		t.Errorf("DatabaseChecker.Check() should not return error when DB is healthy, got: %v", err)
	}

	// 設置數據庫為失敗狀態
	mockDB.shouldFail = true

	// 檢查應該失敗
	err = checker.Check(httptest.NewRequest("GET", "/check", nil))
	if err == nil {
		t.Error("DatabaseChecker.Check() should return error when DB is unhealthy")
	}
}

// 測試 HTTPChecker
func TestHTTPChecker(t *testing.T) {
	// 創建一個測試服務器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// 創建一個檢查器
	checker := &HTTPChecker{
		Name_: "test-http",
		URL:   server.URL + "/health",
		Client: &http.Client{
			Timeout: 1 * time.Second,
		},
	}

	// 檢查名稱
	if checker.Name() != "test-http" {
		t.Errorf("Expected name 'test-http', got '%s'", checker.Name())
	}

	// 檢查應該通過
	err := checker.Check(httptest.NewRequest("GET", "/check", nil))
	if err != nil {
		t.Errorf("HTTPChecker.Check() should not return error for healthy endpoint, got: %v", err)
	}

	// 更新 URL 為失敗的端點
	checker.URL = server.URL + "/fail"

	// 檢查應該失敗
	err = checker.Check(httptest.NewRequest("GET", "/check", nil))
	if err == nil {
		t.Error("HTTPChecker.Check() should return error for unhealthy endpoint")
	}
}

// 測試 RedisChecker
func TestRedisChecker(t *testing.T) {
	// 創建一個通過的檢查器
	passChecker := &RedisChecker{
		Name_: "pass-redis",
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	// 檢查名稱
	if passChecker.Name() != "pass-redis" {
		t.Errorf("Expected name 'pass-redis', got '%s'", passChecker.Name())
	}

	// 檢查應該通過
	err := passChecker.Check(httptest.NewRequest("GET", "/check", nil))
	if err != nil {
		t.Errorf("RedisChecker.Check() should not return error when ping succeeds, got: %v", err)
	}

	// 創建一個失敗的檢查器
	failChecker := &RedisChecker{
		Name_: "fail-redis",
		PingFunc: func(ctx context.Context) error {
			return errors.New("Redis 連接失敗")
		},
	}

	// 檢查應該失敗
	err = failChecker.Check(httptest.NewRequest("GET", "/check", nil))
	if err == nil {
		t.Error("RedisChecker.Check() should return error when ping fails")
	}

	// 測試沒有設置 PingFunc 的情況
	nilChecker := &RedisChecker{
		Name_: "nil-redis",
	}

	// 檢查應該失敗
	err = nilChecker.Check(httptest.NewRequest("GET", "/check", nil))
	if err == nil {
		t.Error("RedisChecker.Check() should return error when PingFunc is nil")
	}
}

// 測試 CompositeChecker
func TestCompositeChecker(t *testing.T) {
	// 創建一個組合檢查器
	checker := &CompositeChecker{
		Name_: "test-composite",
		Checkers: []Checker{
			&PingChecker{},
			&CustomChecker{
				Name_: "always-pass",
				CheckFunc: func(r *http.Request) error {
					return nil
				},
			},
		},
	}

	// 檢查名稱
	if checker.Name() != "test-composite" {
		t.Errorf("Expected name 'test-composite', got '%s'", checker.Name())
	}

	// 檢查應該通過
	err := checker.Check(httptest.NewRequest("GET", "/check", nil))
	if err != nil {
		t.Errorf("CompositeChecker.Check() should not return error when all checks pass, got: %v", err)
	}

	// 添加一個失敗的檢查器
	checker.Checkers = append(checker.Checkers, &CustomChecker{
		Name_: "always-fail",
		CheckFunc: func(r *http.Request) error {
			return errors.New("預期的錯誤")
		},
	})

	// 檢查應該失敗
	err = checker.Check(httptest.NewRequest("GET", "/check", nil))
	if err == nil {
		t.Error("CompositeChecker.Check() should return error when one check fails")
	}
}

// MockPinger 實現 PingContexter 接口
type mockPinger struct {
	shouldFail bool
}

// PingContext 實現 PingContexter 接口的方法
func (m *mockPinger) PingContext(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("數據庫連接失敗")
	}
	return nil
}
