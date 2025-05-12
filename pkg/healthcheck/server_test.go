package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_Basic(t *testing.T) {
	// 創建健康檢查管理器
	logger := setupTestLogger()
	health := New(Config{Logger: logger})

	// 簡單的處理程序
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// 創建 HTTP 伺服器
	server := httptest.NewServer(handler)
	defer server.Close()

	// 設置初始狀態
	health.SetReady(false)
	if health.IsReady() {
		t.Error("Health should not be ready initially")
	}

	// 設置就緒狀態
	health.SetReady(true)
	if !health.IsReady() {
		t.Error("Health should be ready after SetReady(true)")
	}
}

func TestCustomRoutes(t *testing.T) {
	// 創建健康檢查管理器，啟用自定義路由
	logger := setupTestLogger()
	health := New(Config{
		Logger:       logger,
		CustomRoutes: true,
	})

	// 設置為就緒狀態
	health.SetReady(true)

	// 創建測試處理程序
	mux := http.NewServeMux()

	// 自定義健康檢查路徑
	customRoutes := map[string]string{
		"/api/health": "health",
		"/api/live":   "liveness",
		"/api/ready":  "readiness",
	}

	// 安裝自定義健康檢查路由
	health.InstallCustomHandlers(mux, customRoutes)

	// 測試自定義健康端點
	testEndpoint(t, mux, "/api/live", http.StatusOK)
	testEndpoint(t, mux, "/api/ready", http.StatusOK)
	testEndpoint(t, mux, "/api/health", http.StatusOK)

	// 設置為未就緒狀態
	health.SetReady(false)

	// 測試就緒端點應返回錯誤
	testEndpoint(t, mux, "/api/ready", http.StatusInternalServerError)

	// 但存活端點應仍然正常
	testEndpoint(t, mux, "/api/live", http.StatusOK)
}

// 測試輔助函數
func testEndpoint(t *testing.T, handler http.Handler, path string, expectedStatus int) {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != expectedStatus {
		t.Errorf("%s returned status %d, expected %d", path, w.Code, expectedStatus)
	}
}
