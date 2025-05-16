package healthcheck

import (
	"context"
	"database/sql"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module 提供一個基本的健康檢查管理器
var Module = fx.Provide(func(logger *zap.Logger) *Manager {
	return New(Config{
		Logger: logger,
	})
})

// CustomModule 返回一個 fx 模組，提供一個支持自定義路由的健康檢查管理器
func CustomModule() fx.Option {
	return fx.Provide(func(logger *zap.Logger) *Manager {
		return New(Config{
			Logger:       logger,
			CustomRoutes: true,
		})
	})
}

// WithDBChecks 添加數據庫健康檢查
// 注意：使用者需要自行實現檢查邏輯
func WithDBChecks(name string, db PingContexter) fx.Option {
	return fx.Invoke(func(health *Manager) {
		health.AddReadinessCheck(&DatabaseChecker{
			Name_: name,
			DB:    db,
		})
	})
}

// WithSQLDBChecks 添加 SQL 數據庫健康檢查的便捷方法
func WithSQLDBChecks(name string, db *sql.DB) fx.Option {
	return fx.Invoke(func(health *Manager) {
		health.AddReadinessCheck(SQLDBChecker(name, db))
	})
}

// WithRedisChecks 添加 Redis 健康檢查
// pingFunc 參數是一個函數，執行 Redis 的 Ping 操作並返回錯誤（如果有）
func WithRedisChecks(name string, pingFunc func(ctx context.Context) error) fx.Option {
	return fx.Invoke(func(health *Manager) {
		health.AddReadinessCheck(&RedisChecker{
			Name_:    name,
			PingFunc: pingFunc,
		})
	})
}

// CombinedHealthModule 提供完整的健康檢查服務，包括健康檢查管理器和健康檢查服務器
var CombinedHealthModule = fx.Options(
	Module,
	HealthServerModule,
)
