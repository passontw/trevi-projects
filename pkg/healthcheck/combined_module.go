package healthcheck

import (
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CombinedModuleConfig 健康檢查與優雅關閉綜合模組配置
type CombinedModuleConfig struct {
	// 健康檢查端口，默認為 8088
	HealthPort int
	// 優雅關閉超時時間
	ShutdownTimeout time.Duration
	// 是否處理系統信號
	HandleSignals bool
}

// DefaultCombinedModuleConfig 返回默認配置
func DefaultCombinedModuleConfig() CombinedModuleConfig {
	return CombinedModuleConfig{
		HealthPort:      8088,
		ShutdownTimeout: 30 * time.Second,
		HandleSignals:   true,
	}
}

// ModuleParams 創建綜合模組所需的參數
type ModuleParams struct {
	Config CombinedModuleConfig
	Logger *zap.Logger
}

// NewCombinedModule 創建新的綜合模組
func NewCombinedModule(params ModuleParams) fx.Option {
	if params.Logger == nil {
		params.Logger = zap.NewNop()
	}

	if params.Config.HealthPort <= 0 {
		params.Config.HealthPort = 8088
	}

	if params.Config.ShutdownTimeout <= 0 {
		params.Config.ShutdownTimeout = 30 * time.Second
	}

	return fx.Options(
		// 提供健康檢查管理器
		fx.Provide(func() *Manager {
			return New(Config{
				Logger:       params.Logger,
				CustomRoutes: false,
			})
		}),

		// 提供優雅關閉管理器
		fx.Provide(func(health *Manager) *GracefulShutdown {
			return NewGracefulShutdown(
				params.Logger,
				health,
				ShutdownConfig{
					ShutdownTimeout: params.Config.ShutdownTimeout,
					HandleSignals:   params.Config.HandleSignals,
				},
			)
		}),

		// 提供關閉指示器
		fx.Provide(func(graceful *GracefulShutdown) *ShutdownIndicator {
			return NewShutdownIndicator(graceful)
		}),

		// 提供健康檢查服務器
		fx.Provide(func(logger *zap.Logger, health *Manager, graceful *GracefulShutdown) *HealthServer {
			server := NewHealthServer(logger, health, graceful)
			return server
		}),

		// 啟動優雅關閉管理器
		fx.Invoke(func(graceful *GracefulShutdown, lc fx.Lifecycle) {
			graceful.Start(lc)
		}),

		// 啟動健康檢查服務器
		fx.Invoke(func(server *HealthServer, lc fx.Lifecycle) {
			server.Start(lc)
		}),

		// 註冊關閉指示器進行健康檢查
		fx.Invoke(func(health *Manager, indicator *ShutdownIndicator) {
			health.AddReadinessCheck(indicator)
		}),
	)
}

// CombinedModule 預設的健康檢查與優雅關閉模組
var CombinedModule = fx.Options(
	fx.Provide(func() *zap.Logger {
		// 如果外部已提供 logger，這個會被覆蓋
		return zap.NewNop()
	}),

	// 提供健康檢查管理器
	fx.Provide(func(logger *zap.Logger) *Manager {
		return New(Config{
			Logger:       logger,
			CustomRoutes: false,
		})
	}),

	// 提供優雅關閉管理器
	fx.Provide(func(logger *zap.Logger, health *Manager) *GracefulShutdown {
		return NewGracefulShutdown(
			logger,
			health,
			ShutdownConfig{
				ShutdownTimeout: 30 * time.Second,
				HandleSignals:   true,
			},
		)
	}),

	// 提供關閉指示器
	fx.Provide(func(graceful *GracefulShutdown) *ShutdownIndicator {
		return NewShutdownIndicator(graceful)
	}),

	// 提供健康檢查服務器
	fx.Provide(func(logger *zap.Logger, health *Manager, graceful *GracefulShutdown) *HealthServer {
		server := NewHealthServer(logger, health, graceful)
		return server
	}),

	// 啟動優雅關閉管理器
	fx.Invoke(func(graceful *GracefulShutdown, lc fx.Lifecycle) {
		graceful.Start(lc)
	}),

	// 啟動健康檢查服務器
	fx.Invoke(func(server *HealthServer, lc fx.Lifecycle) {
		server.Start(lc)
	}),

	// 註冊關閉指示器進行健康檢查
	fx.Invoke(func(health *Manager, indicator *ShutdownIndicator) {
		health.AddReadinessCheck(indicator)
	}),
)

// CombinedHealthModule 已在 module.go 中定義，此處不再重複定義
