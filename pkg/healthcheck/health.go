package healthcheck

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// Checker 定義健康檢查的接口
type Checker interface {
	// Name 返回檢查器的名稱
	Name() string

	// Check 執行健康檢查，如果健康返回 nil，否則返回錯誤
	Check(r *http.Request) error
}

// CheckType 表示檢查類型：活性檢查或就緒檢查
type CheckType int

const (
	// LivenessCheck 表示活性檢查，確認服務是否運行
	LivenessCheck CheckType = iota

	// ReadinessCheck 表示就緒檢查，確認服務是否可以處理請求
	ReadinessCheck

	// StartupCheck 表示啟動檢查，檢查服務是否正確初始化
	StartupCheck
)

// Manager 健康檢查管理器，管理各種健康檢查
type Manager struct {
	readyState   atomic.Bool
	checkers     map[CheckType][]Checker
	logger       *zap.Logger
	mu           sync.RWMutex
	handlers     map[string]http.HandlerFunc
	customRoutes bool
}

// Config 健康檢查管理器配置
type Config struct {
	// Logger 是用於日誌記錄的 zap logger
	Logger *zap.Logger

	// CustomRoutes 如果為 true，不會自動註冊標準路由
	CustomRoutes bool
}

// New 創建一個新的健康檢查管理器
func New(config Config) *Manager {
	logger := config.Logger
	if logger == nil {
		// 如果沒有提供 logger，使用 noop logger
		logger = zap.NewNop()
	}

	m := &Manager{
		checkers:     make(map[CheckType][]Checker),
		logger:       logger.With(zap.String("component", "health_manager")),
		handlers:     make(map[string]http.HandlerFunc),
		customRoutes: config.CustomRoutes,
	}

	// 設置初始狀態為未就緒
	m.readyState.Store(false)

	// 註冊默認的 ping 檢查
	m.AddLivenessCheck(&PingChecker{})

	// 註冊就緒狀態檢查
	m.AddReadinessCheck(&ReadinessStateChecker{manager: m})

	// 初始化標準處理程序
	m.initHandlers()

	return m
}

// initHandlers 初始化標準的健康檢查處理程序
func (m *Manager) initHandlers() {
	// 存活檢查處理程序
	m.handlers["liveness"] = m.createCheckHandler(LivenessCheck)

	// 就緒檢查處理程序
	m.handlers["readiness"] = m.createCheckHandler(ReadinessCheck)

	// 通用健康檢查處理程序
	m.handlers["health"] = m.createHealthHandler()
}

// createCheckHandler 創建特定類型的健康檢查處理程序
func (m *Manager) createCheckHandler(checkType CheckType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		checkers := append([]Checker{}, m.checkers[checkType]...)
		m.mu.RUnlock()

		for _, checker := range checkers {
			if err := checker.Check(r); err != nil {
				m.logger.Warn("健康檢查失敗",
					zap.String("checker", checker.Name()),
					zap.Error(err))

				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "健康檢查失敗: %s - %v\n", checker.Name(), err)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}
}

// createHealthHandler 創建通用健康檢查處理程序
func (m *Manager) createHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		liveCheckers := append([]Checker{}, m.checkers[LivenessCheck]...)
		readyCheckers := append([]Checker{}, m.checkers[ReadinessCheck]...)
		startupCheckers := append([]Checker{}, m.checkers[StartupCheck]...)
		m.mu.RUnlock()

		allCheckers := append(liveCheckers, readyCheckers...)
		allCheckers = append(allCheckers, startupCheckers...)

		failed := false

		for _, checker := range allCheckers {
			if err := checker.Check(r); err != nil {
				m.logger.Warn("健康檢查失敗",
					zap.String("checker", checker.Name()),
					zap.Error(err))

				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "健康檢查失敗: %s - %v\n", checker.Name(), err)
				failed = true
				break
			}
		}

		if !failed {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "ok")
		}
	}
}

// AddChecker 添加一個特定類型的健康檢查器
func (m *Manager) AddChecker(checkType CheckType, checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug("添加健康檢查",
		zap.String("checker", checker.Name()),
		zap.Int("type", int(checkType)))

	m.checkers[checkType] = append(m.checkers[checkType], checker)
}

// AddLivenessCheck 添加一個活性檢查
func (m *Manager) AddLivenessCheck(checker Checker) {
	m.AddChecker(LivenessCheck, checker)
}

// AddReadinessCheck 添加一個就緒檢查
func (m *Manager) AddReadinessCheck(checker Checker) {
	m.AddChecker(ReadinessCheck, checker)
}

// AddStartupCheck 添加一個啟動檢查
func (m *Manager) AddStartupCheck(checker Checker) {
	m.AddChecker(StartupCheck, checker)
}

// SetReady 設置服務的就緒狀態
func (m *Manager) SetReady(ready bool) {
	oldState := m.readyState.Load()
	m.readyState.Store(ready)

	if oldState != ready {
		if ready {
			m.logger.Info("服務已標記為就緒")
		} else {
			m.logger.Info("服務已標記為未就緒")
		}
	}
}

// IsReady 返回服務的就緒狀態
func (m *Manager) IsReady() bool {
	return m.readyState.Load()
}

// GetHandler 獲取指定路徑的處理程序
func (m *Manager) GetHandler(path string) (http.HandlerFunc, bool) {
	handler, exists := m.handlers[path]
	return handler, exists
}

// Handle 添加自定義處理程序
func (m *Manager) Handle(path string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers[path] = handler
	m.logger.Debug("添加自定義處理程序", zap.String("path", path))
}

// InstallHandlers 在提供的 ServeMux 上安裝健康檢查處理程序
func (m *Manager) InstallHandlers(mux *http.ServeMux) {
	if m.customRoutes {
		// 客戶端需自行註冊路由
		return
	}

	// 安裝標準路徑
	mux.HandleFunc("GET /liveness", m.handlers["liveness"])
	m.logger.Info("安裝健康檢查端點", zap.String("endpoint", "/liveness"))

	mux.HandleFunc("GET /readiness", m.handlers["readiness"])
	m.logger.Info("安裝就緒檢查端點", zap.String("endpoint", "/readiness"))

	// 安裝詳細路徑
	mux.HandleFunc("GET /livez", m.handlers["liveness"])
	m.logger.Info("安裝詳細健康檢查端點", zap.String("endpoint", "/livez"))

	mux.HandleFunc("GET /readyz", m.handlers["readiness"])
	m.logger.Info("安裝詳細就緒檢查端點", zap.String("endpoint", "/readyz"))

	// 安裝通用健康檢查處理程序
	mux.HandleFunc("GET /healthz", m.handlers["health"])
	m.logger.Info("安裝通用健康檢查端點", zap.String("endpoint", "/healthz"))
}

// InstallCustomHandlers 根據自定義路由配置安裝健康檢查處理程序
// 路由參數格式: map[string]string{"路徑": "處理程序名稱"}
func (m *Manager) InstallCustomHandlers(mux *http.ServeMux, routes map[string]string) {
	for path, handlerName := range routes {
		if handler, exists := m.handlers[handlerName]; exists {
			fullPath := fmt.Sprintf("GET %s", path)
			mux.HandleFunc(fullPath, handler)
			m.logger.Info("安裝自定義健康檢查端點",
				zap.String("endpoint", path),
				zap.String("handler", handlerName))
		} else {
			m.logger.Warn("未找到處理程序",
				zap.String("handler", handlerName))
		}
	}
}

// ReadinessStateChecker 檢查服務的就緒狀態
type ReadinessStateChecker struct {
	manager *Manager
}

// Name 返回檢查器的名稱
func (r *ReadinessStateChecker) Name() string {
	return "readiness-state"
}

// Check 檢查服務是否就緒
func (r *ReadinessStateChecker) Check(req *http.Request) error {
	if !r.manager.IsReady() {
		return fmt.Errorf("服務未就緒")
	}
	return nil
}

// PingChecker 是一個簡單的 ping 檢查器
type PingChecker struct{}

// Name 返回檢查器的名稱
func (p *PingChecker) Name() string {
	return "ping"
}

// Check 總是返回成功
func (p *PingChecker) Check(req *http.Request) error {
	return nil
}
