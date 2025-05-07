package api

import (
	"context"
	"fmt"
	"net/http"

	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/internal/lottery_service/gameflow"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// APIServer 處理 API 請求的 HTTP 服務器
type APIServer struct {
	config      *config.AppConfig
	logger      *zap.Logger
	gameManager *gameflow.GameManager
	repository  *gameflow.CompositeRepository
	server      *http.Server
}

// NewAPIServer 創建新的 API 服務器
func NewAPIServer(
	config *config.AppConfig,
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
	repository *gameflow.CompositeRepository,
) *APIServer {
	// 確保 logger 有適當的 tag
	apiLogger := logger.With(zap.String("component", "api_server"))

	// 創建 API 服務器
	server := &APIServer{
		config:      config,
		logger:      apiLogger,
		gameManager: gameManager,
		repository:  repository,
	}

	return server
}

// Start 啟動 API 服務器
func (s *APIServer) Start(lc fx.Lifecycle) {
	// 使用應用配置中的 API 端口
	port := s.config.Server.Port
	host := s.config.Server.Host
	serverAddr := fmt.Sprintf("%s:%d", host, port)
	s.logger.Info("正在啟動 API 服務器",
		zap.String("address", serverAddr),
		zap.Int("port", port))

	// 建立 ServeMux
	mux := http.NewServeMux()

	// 註冊 API 路由
	s.registerRoutes(mux)

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
				s.logger.Info("API 服務器已啟動",
					zap.String("監聽地址", serverAddr),
					zap.Int("端口", port))

				if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					s.logger.Error("API 服務器運行失敗", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.logger.Info("正在停止 API 服務器")
			return s.server.Shutdown(ctx)
		},
	})
}

// registerRoutes 註冊所有 API 路由
func (s *APIServer) registerRoutes(mux *http.ServeMux) {
	// 健康檢查
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"UP"}`))
	})

	// 根路徑
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Lottery Service API Server is running."))
	})

	s.logger.Info("API 路由註冊完成")
}

// Module 提供 FX 模塊
var Module = fx.Options(
	fx.Provide(NewAPIServer),
	fx.Invoke(func(server *APIServer, lc fx.Lifecycle) {
		server.Start(lc)
	}),
)
