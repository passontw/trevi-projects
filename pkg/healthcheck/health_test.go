package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func setupTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestHealth_SetReady(t *testing.T) {
	logger := setupTestLogger()
	h := New(Config{Logger: logger})

	if h.IsReady() {
		t.Error("Health should not be ready by default")
	}

	h.SetReady(true)

	if !h.IsReady() {
		t.Error("Health should be ready after SetReady(true)")
	}
}

func TestHealth_AddCheckers(t *testing.T) {
	logger := setupTestLogger()
	h := New(Config{Logger: logger})

	// 添加一個自定義檢查器
	customChecker := &CustomChecker{
		Name_: "test-checker",
		CheckFunc: func(r *http.Request) error {
			return nil
		},
	}

	h.AddReadinessCheck(customChecker)

	// 設置服務為就緒狀態
	h.SetReady(true)

	// 由於我們使用了私有字段，無法直接檢查，但可以通過行為測試
	// 在這裡測試就緒檢查
	req := httptest.NewRequest("GET", "/readiness", nil)
	w := httptest.NewRecorder()

	// 獲取處理程序
	handler, exists := h.GetHandler("readiness")
	if !exists {
		t.Fatal("Readiness handler should exist")
	}

	// 執行處理程序
	handler(w, req)

	// 檢查結果 - 應該是就緒的，因為所有檢查都通過
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, w.Code)
	}
}

func TestHealth_Handlers(t *testing.T) {
	logger := setupTestLogger()
	h := New(Config{Logger: logger})
	mux := http.NewServeMux()
	h.InstallHandlers(mux)

	// 測試未就緒時的就緒端點
	req := httptest.NewRequest("GET", "/readiness", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// 應該返回 500，因為我們還未設置為就緒
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d but got %d", http.StatusInternalServerError, w.Code)
	}

	// 測試就緒後的就緒端點
	h.SetReady(true)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, w.Code)
	}

	// 測試存活端點
	req = httptest.NewRequest("GET", "/liveness", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, w.Code)
	}
}

func TestHealth_CustomRoutes(t *testing.T) {
	logger := setupTestLogger()
	h := New(Config{
		Logger:       logger,
		CustomRoutes: true,
	})
	mux := http.NewServeMux()

	// 定義自定義路由
	customRoutes := map[string]string{
		"/my/health": "health",
		"/my/ping":   "liveness",
		"/my/ready":  "readiness",
	}

	// 安裝自定義路由
	h.InstallCustomHandlers(mux, customRoutes)

	// 測試自定義存活端點
	req := httptest.NewRequest("GET", "/my/ping", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, w.Code)
	}

	// 測試自定義就緒端點
	req = httptest.NewRequest("GET", "/my/ready", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// 尚未就緒，應返回錯誤
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d but got %d", http.StatusInternalServerError, w.Code)
	}

	// 設置為就緒
	h.SetReady(true)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// 現在應該就緒了
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, w.Code)
	}
}
