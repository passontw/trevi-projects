package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Server 是一個支持優雅關閉和健康檢查的 HTTP 服務器包裝器
type Server struct {
	server          *http.Server
	healthManager   *Manager
	shutdownTimeout time.Duration
	wg              sync.WaitGroup
	shutdownCh      chan struct{}
	startupDone     chan struct{}
	startupErr      error
	logger          *zap.Logger
}

// ServerConfig 服務器配置選項
type ServerConfig struct {
	// Addr 服務器監聽地址，如 ":8080"
	Addr string

	// Handler HTTP 處理程序
	Handler http.Handler

	// HealthManager 健康檢查管理器
	HealthManager *Manager

	// Logger 日誌記錄器
	Logger *zap.Logger

	// ShutdownTimeout 關閉超時時間
	ShutdownTimeout time.Duration
}

// NewServer 創建一個新的服務器實例
func NewServer(config ServerConfig) *Server {
	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	shutdownTimeout := config.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}

	return &Server{
		server: &http.Server{
			Addr:    config.Addr,
			Handler: config.Handler,
		},
		healthManager:   config.HealthManager,
		shutdownTimeout: shutdownTimeout,
		shutdownCh:      make(chan struct{}),
		startupDone:     make(chan struct{}),
		logger:          logger.With(zap.String("component", "http_server")),
	}
}

// Start 啟動服務器並執行初始化操作
func (s *Server) Start(initFn func(context.Context) error) error {
	// 創建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設置信號處理
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// 啟動 HTTP 服務器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		s.logger.Info("啟動服務器", zap.String("地址", s.server.Addr))
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("服務器錯誤", zap.Error(err))
			cancel() // 通知其他 goroutine 退出
		}
	}()

	// 初始化操作
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(s.startupDone)

		if s.healthManager != nil {
			// 初始化時標記為未就緒
			s.healthManager.SetReady(false)
		}

		if initFn != nil {
			s.logger.Info("執行初始化...")
			s.startupErr = initFn(ctx)
			if s.startupErr != nil {
				s.logger.Error("初始化錯誤", zap.Error(s.startupErr))
				// 初始化失敗時不將服務標記為就緒
			} else {
				s.logger.Info("初始化完成，服務已就緒")
				if s.healthManager != nil {
					s.healthManager.SetReady(true)
				}
			}
		} else {
			// 無初始化函數，直接標記為就緒
			s.logger.Info("無初始化函數，服務已就緒")
			if s.healthManager != nil {
				s.healthManager.SetReady(true)
			}
		}
	}()

	// 等待關閉信號
	select {
	case sig := <-signalCh:
		s.logger.Info("收到信號", zap.String("信號", sig.String()))
		s.shutdown()
	case <-ctx.Done():
		s.logger.Info("上下文已取消")
		s.shutdown()
	case <-s.shutdownCh:
		s.logger.Info("收到關閉請求")
	}

	// 等待所有 goroutine 完成
	s.wg.Wait()
	return s.startupErr
}

// WaitForStartup 等待服務器啟動完成
func (s *Server) WaitForStartup() error {
	<-s.startupDone
	return s.startupErr
}

// Shutdown 請求服務器優雅關閉
func (s *Server) Shutdown() {
	if s.healthManager != nil {
		s.healthManager.SetReady(false) // 立即標記為未就緒
	}
	close(s.shutdownCh)
}

// 內部關閉邏輯
func (s *Server) shutdown() {
	// 標記為未就緒，拒絕新請求
	if s.healthManager != nil {
		s.healthManager.SetReady(false)
	}
	s.logger.Info("服務已標記為未就緒，開始優雅關閉")

	// 創建帶超時的關閉上下文
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// 嘗試優雅關閉
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("優雅關閉失敗", zap.Error(err))
		// 強制關閉
		if err := s.server.Close(); err != nil {
			s.logger.Error("強制關閉失敗", zap.Error(err))
		}
	} else {
		s.logger.Info("服務器已優雅關閉")
	}
}
