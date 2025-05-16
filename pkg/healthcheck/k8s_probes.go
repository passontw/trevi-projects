package healthcheck

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// K8sProbeConfig Kubernetes 探針配置
type K8sProbeConfig struct {
	// 是否啟用 Kubernetes 探針
	Enabled bool

	// Liveness 探針路徑，默認為 /livez
	LivenessPath string

	// Readiness 探針路徑，默認為 /readyz
	ReadinessPath string

	// Startup 探針路徑，默認為 /startupz
	StartupPath string

	// 優雅終止路徑，默認為 /termination
	TerminationPath string
}

// DefaultK8sProbeConfig 返回默認的 Kubernetes 探針配置
func DefaultK8sProbeConfig() K8sProbeConfig {
	return K8sProbeConfig{
		Enabled:         true,
		LivenessPath:    "/livez",
		ReadinessPath:   "/readyz",
		StartupPath:     "/startupz",
		TerminationPath: "/termination",
	}
}

// K8sProbeServer Kubernetes 探針服務器
type K8sProbeServer struct {
	logger       *zap.Logger
	config       K8sProbeConfig
	health       *Manager
	graceful     *GracefulShutdown
	startupReady bool
}

// NewK8sProbeServer 創建新的 Kubernetes 探針服務器
func NewK8sProbeServer(
	logger *zap.Logger,
	health *Manager,
	graceful *GracefulShutdown,
	config K8sProbeConfig,
) *K8sProbeServer {
	if !config.Enabled {
		return nil
	}

	// 確保有合理的默認值
	if config.LivenessPath == "" {
		config.LivenessPath = "/livez"
	}

	if config.ReadinessPath == "" {
		config.ReadinessPath = "/readyz"
	}

	if config.StartupPath == "" {
		config.StartupPath = "/startupz"
	}

	if config.TerminationPath == "" {
		config.TerminationPath = "/termination"
	}

	// 確保 logger 有適當的 tag
	probeLogger := logger.With(zap.String("component", "k8s_probe_server"))

	return &K8sProbeServer{
		logger:       probeLogger,
		config:       config,
		health:       health,
		graceful:     graceful,
		startupReady: false,
	}
}

// RegisterProbes 註冊 Kubernetes 探針到提供的多路複用器
func (k *K8sProbeServer) RegisterProbes(mux *http.ServeMux) {
	if k == nil {
		return
	}

	// 註冊存活探針
	mux.HandleFunc("GET "+k.config.LivenessPath, k.handleLiveness)

	// 註冊就緒探針
	mux.HandleFunc("GET "+k.config.ReadinessPath, k.handleReadiness)

	// 註冊啟動探針
	mux.HandleFunc("GET "+k.config.StartupPath, k.handleStartup)

	// 註冊終止路徑
	mux.HandleFunc("GET "+k.config.TerminationPath, k.handleTermination)

	k.logger.Info("已註冊 Kubernetes 探針",
		zap.String("liveness", k.config.LivenessPath),
		zap.String("readiness", k.config.ReadinessPath),
		zap.String("startup", k.config.StartupPath),
		zap.String("termination", k.config.TerminationPath))
}

// MarkStartupComplete 標記啟動完成
func (k *K8sProbeServer) MarkStartupComplete() {
	if k == nil {
		return
	}

	k.startupReady = true
	k.logger.Info("標記啟動探針為就緒")
}

// handleLiveness 處理存活探針請求
func (k *K8sProbeServer) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// 檢查是否在關閉過程中
	if k.graceful != nil && k.graceful.IsShuttingDown() {
		k.logger.Debug("存活探針檢查失敗：服務正在關閉")
		http.Error(w, "服務正在關閉", http.StatusServiceUnavailable)
		return
	}

	// 執行所有存活檢查
	k.health.handlers["liveness"](w, r)
}

// handleReadiness 處理就緒探針請求
func (k *K8sProbeServer) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// 檢查是否在關閉過程中
	if k.graceful != nil && k.graceful.IsShuttingDown() {
		k.logger.Debug("就緒探針檢查失敗：服務正在關閉")
		http.Error(w, "服務正在關閉", http.StatusServiceUnavailable)
		return
	}

	// 執行所有就緒檢查
	k.health.handlers["readiness"](w, r)
}

// handleStartup 處理啟動探針請求
func (k *K8sProbeServer) handleStartup(w http.ResponseWriter, r *http.Request) {
	if !k.startupReady {
		k.logger.Debug("啟動探針檢查失敗：服務尚未完成啟動")
		http.Error(w, "服務尚未完成啟動", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleTermination 處理終止請求
func (k *K8sProbeServer) handleTermination(w http.ResponseWriter, r *http.Request) {
	// 檢查請求是否來自同一台主機或可信來源
	// 在生產環境應該增加適當的認證檢查
	clientIP := r.RemoteAddr

	k.logger.Info("收到終止請求",
		zap.String("clientIP", clientIP))

	// 回應 200 OK
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("開始終止服務"))

	// 非同步觸發關閉
	go func() {
		// 給 HTTP 響應一些時間完成
		time.Sleep(100 * time.Millisecond)

		if k.graceful != nil {
			k.logger.Info("開始優雅關閉過程")
			k.graceful.Shutdown(context.Background())
		}
	}()
}
