package healthcheck

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ShutdownConfig 優雅關閉配置
type ShutdownConfig struct {
	// 優雅關閉的超時時間
	ShutdownTimeout time.Duration
	// 是否自動設置信號處理
	HandleSignals bool
	// 自定義關閉前的回調函數
	BeforeShutdown func(ctx context.Context)
	// 自定義關閉後的回調函數
	AfterShutdown func(ctx context.Context)
}

// GracefulShutdown 管理應用程序優雅關閉
type GracefulShutdown struct {
	logger          *zap.Logger
	shutdownTimeout time.Duration
	healthManager   *Manager
	beforeShutdown  func(ctx context.Context)
	afterShutdown   func(ctx context.Context)
	shutdownCh      chan struct{}
	shutdownOnce    sync.Once
	isShuttingDown  bool
	lock            sync.RWMutex
}

// NewGracefulShutdown 創建新的優雅關閉管理器
func NewGracefulShutdown(
	logger *zap.Logger,
	healthManager *Manager,
	config ShutdownConfig,
) *GracefulShutdown {
	// 確保 logger 有適當的 tag
	shutdownLogger := logger.With(zap.String("component", "graceful_shutdown"))

	// 如果沒有指定超時，使用默認值30秒
	shutdownTimeout := config.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 30 * time.Second
	}

	return &GracefulShutdown{
		logger:          shutdownLogger,
		shutdownTimeout: shutdownTimeout,
		healthManager:   healthManager,
		beforeShutdown:  config.BeforeShutdown,
		afterShutdown:   config.AfterShutdown,
		shutdownCh:      make(chan struct{}),
		isShuttingDown:  false,
	}
}

// Start 註冊生命週期鉤子
func (g *GracefulShutdown) Start(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			g.logger.Info("優雅關閉管理器已啟動",
				zap.Duration("shutdownTimeout", g.shutdownTimeout))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return g.handleShutdown(ctx)
		},
	})
}

// WaitForShutdown 阻塞等待關閉信號
func (g *GracefulShutdown) WaitForShutdown(ctx context.Context) {
	<-g.shutdownCh
}

// IsShuttingDown 檢查服務是否正在關閉
func (g *GracefulShutdown) IsShuttingDown() bool {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.isShuttingDown
}

// Shutdown 發起優雅關閉
func (g *GracefulShutdown) Shutdown(ctx context.Context) error {
	g.shutdownOnce.Do(func() {
		close(g.shutdownCh)
	})
	return nil
}

// SetupSignalHandling 設置信號處理
func (g *GracefulShutdown) SetupSignalHandling(cancel context.CancelFunc) {
	g.logger.Info("註冊系統信號處理器")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		g.logger.Info("收到系統信號，準備關閉服務", zap.String("signal", sig.String()))

		// 設置正在關閉標誌
		g.lock.Lock()
		g.isShuttingDown = true
		g.lock.Unlock()

		// 標記服務為未就緒
		if g.healthManager != nil {
			g.healthManager.SetReady(false)
			g.logger.Info("服務已標記為未就緒，拒絕新請求")
		}

		// 取消上下文
		cancel()
	}()
}

// handleShutdown 處理關閉流程
func (g *GracefulShutdown) handleShutdown(ctx context.Context) error {
	g.logger.Info("開始執行優雅關閉流程")

	// 設置關閉標誌
	g.lock.Lock()
	g.isShuttingDown = true
	g.lock.Unlock()

	// 標記服務為未就緒
	if g.healthManager != nil {
		g.healthManager.SetReady(false)
		g.logger.Info("服務已標記為未就緒，拒絕新請求")
	}

	// 創建帶超時的上下文
	shutdownCtx, cancel := context.WithTimeout(context.Background(), g.shutdownTimeout)
	defer cancel()

	// 執行關閉前回調
	if g.beforeShutdown != nil {
		g.logger.Info("執行關閉前回調")
		g.beforeShutdown(shutdownCtx)
	}

	// 等待超時或正常關閉
	g.logger.Info("等待服務關閉（最多等待）", zap.Duration("timeout", g.shutdownTimeout))

	// 發送關閉信號
	g.shutdownOnce.Do(func() {
		close(g.shutdownCh)
	})

	// 創建一個計時器，用於在超時情況下記錄
	timer := time.NewTimer(g.shutdownTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		g.logger.Warn("服務關閉等待超時")
	case <-shutdownCtx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			g.logger.Warn("服務關閉上下文超時")
		}
	}

	// 執行關閉後回調
	if g.afterShutdown != nil {
		g.logger.Info("執行關閉後回調")
		g.afterShutdown(shutdownCtx)
	}

	g.logger.Info("優雅關閉完成")
	return nil
}

// ShutdownIndicator 關閉指示器，可用於健康檢查
type ShutdownIndicator struct {
	gracefulShutdown *GracefulShutdown
}

// NewShutdownIndicator 創建新的關閉指示器
func NewShutdownIndicator(g *GracefulShutdown) *ShutdownIndicator {
	return &ShutdownIndicator{
		gracefulShutdown: g,
	}
}

// Name 返回檢查器的名稱
func (s *ShutdownIndicator) Name() string {
	return "shutdown-indicator"
}

// Check 檢查服務是否正在關閉
func (s *ShutdownIndicator) Check(req *http.Request) error {
	if s.gracefulShutdown.IsShuttingDown() {
		return context.Canceled
	}
	return nil
}
