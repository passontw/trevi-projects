package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 從環境變數獲取整數值，如果未設置或無效則返回默認值
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// HealthServerConfig 健康檢查服務器配置
type HealthServerConfig struct {
	// 服務端口
	Port int
	// 是否啟用 Kubernetes 探針
	EnableK8sProbes bool
	// Kubernetes 探針配置
	K8sProbeConfig K8sProbeConfig
}

// DefaultHealthServerConfig 返回默認的健康檢查服務器配置
func DefaultHealthServerConfig() HealthServerConfig {
	return HealthServerConfig{
		Port:            getEnvAsInt("API_HEALTH_PORT", 8088),
		EnableK8sProbes: true,
		K8sProbeConfig:  DefaultK8sProbeConfig(),
	}
}

// HealthServer 處理健康檢查請求的 HTTP 服務器
type HealthServer struct {
	logger         *zap.Logger
	server         *http.Server
	health         *Manager
	graceful       *GracefulShutdown
	k8sProbeServer *K8sProbeServer
	port           int
	shutdownCh     chan struct{}
	customRoutes   bool
	config         HealthServerConfig
}

// NewHealthServer 創建新的健康檢查服務器
func NewHealthServer(
	logger *zap.Logger,
	health *Manager,
	graceful *GracefulShutdown,
) *HealthServer {
	return NewConfiguredHealthServer(
		logger,
		health,
		graceful,
		DefaultHealthServerConfig(),
	)
}

// NewConfiguredHealthServer 創建新的健康檢查服務器，使用自定義配置
func NewConfiguredHealthServer(
	logger *zap.Logger,
	health *Manager,
	graceful *GracefulShutdown,
	config HealthServerConfig,
) *HealthServer {
	// 確保 logger 有適當的 tag
	healthLogger := logger.With(zap.String("component", "health_server"))

	// 創建健康檢查服務器
	server := &HealthServer{
		logger:       healthLogger,
		health:       health,
		graceful:     graceful,
		shutdownCh:   make(chan struct{}),
		customRoutes: false,
		config:       config,
	}

	// 如果啟用了 Kubernetes 探針，創建對應的服務器
	if config.EnableK8sProbes {
		server.k8sProbeServer = NewK8sProbeServer(
			logger,
			health,
			graceful,
			config.K8sProbeConfig,
		)
	}

	return server
}

// Start 啟動健康檢查服務器
func (s *HealthServer) Start(lc fx.Lifecycle) {
	// 使用環境變數或配置中的端口
	s.port = s.config.Port
	if s.port <= 0 {
		s.port = getEnvAsInt("API_HEALTH_PORT", 8088)
	}

	// 記錄實際使用的端口
	s.logger.Info("使用的健康檢查端口配置",
		zap.Int("環境變數API_HEALTH_PORT", getEnvAsInt("API_HEALTH_PORT", 0)),
		zap.Int("配置端口", s.config.Port),
		zap.Int("最終使用Port", s.port))

	serverAddr := fmt.Sprintf(":%d", s.port)
	s.logger.Info("正在啟動健康檢查服務器",
		zap.String("address", serverAddr),
		zap.Int("port", s.port))

	// 創建 HTTP 伺服器的 Mux，使用 Go 1.22 新的路由功能
	mux := http.NewServeMux()

	// 安裝健康檢查路由
	mux.HandleFunc("GET /healthz", s.health.handlers["health"])

	// 如果沒有啟用 Kubernetes 探針，才註冊這些路由，避免衝突
	if s.k8sProbeServer == nil {
		mux.HandleFunc("GET /livez", s.health.handlers["liveness"])
		mux.HandleFunc("GET /readyz", s.health.handlers["readiness"])
	}

	// 向後兼容的路由
	mux.HandleFunc("GET /liveness", s.health.handlers["liveness"])
	mux.HandleFunc("GET /readiness", s.health.handlers["readiness"])

	// 安裝 Kubernetes 探針（如果啟用）
	if s.k8sProbeServer != nil {
		s.k8sProbeServer.RegisterProbes(mux)
	}

	// 添加根路徑處理
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "彩票服務健康檢查伺服器正在運行\n")
		fmt.Fprintf(w, "可用端點：\n")
		fmt.Fprintf(w, "- /healthz：詳細健康狀態檢查\n")
		fmt.Fprintf(w, "- /livez：存活檢查\n")
		fmt.Fprintf(w, "- /readyz：就緒檢查\n")
		if s.k8sProbeServer != nil {
			fmt.Fprintf(w, "- %s：啟動檢查\n", s.k8sProbeServer.config.StartupPath)
			fmt.Fprintf(w, "- %s：終止服務\n", s.k8sProbeServer.config.TerminationPath)
		}
	})

	// 建立 HTTP 服務器
	s.server = &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	// 生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 啟動 HTTP 服務器
			go func() {
				s.logger.Info("健康檢查服務器已啟動",
					zap.String("監聽地址", serverAddr),
					zap.Int("端口", s.port))

				if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					s.logger.Error("健康檢查服務器運行失敗", zap.Error(err))
				}
			}()

			// 標記服務為就緒（如果有 K8s 探針）
			if s.k8sProbeServer != nil {
				// 設置一個計時器，延遲標記為就緒
				go func() {
					// 等待 2 秒鐘以讓服務有時間啟動
					time.Sleep(2 * time.Second)
					s.k8sProbeServer.MarkStartupComplete()
				}()
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.logger.Info("正在停止健康檢查服務器")

			// 設置關閉超時時間
			timeout := 15 * time.Second

			// 創建帶超時的上下文
			shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// 嘗試優雅關閉
			if err := s.server.Shutdown(shutdownCtx); err != nil {
				s.logger.Error("健康檢查服務器優雅關閉失敗", zap.Error(err))
				return err
			}

			s.logger.Info("健康檢查服務器已優雅關閉")
			return nil
		},
	})
}

// MarkStartupComplete 標記啟動完成
func (s *HealthServer) MarkStartupComplete() {
	if s.k8sProbeServer != nil {
		s.k8sProbeServer.MarkStartupComplete()
	}
}

// HealthServerModule 提供健康檢查服務器作為 fx 模組
var HealthServerModule = fx.Options(
	fx.Provide(NewHealthServer),
	fx.Invoke(func(server *HealthServer, lc fx.Lifecycle) {
		server.Start(lc)
	}),
)
