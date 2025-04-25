package core

import (
	"context"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/service"
	"g38_lottery_service/pkg/databaseManager"
	"g38_lottery_service/pkg/logger"
	"g38_lottery_service/pkg/nacosManager"
	redis "g38_lottery_service/pkg/redisManager"
	"g38_lottery_service/pkg/websocketManager"

	"go.uber.org/fx"
)

// DatabaseModule 數據庫模組
var DatabaseModule = fx.Options(
	fx.Provide(
		// 基於 Config 轉換為 MySQLConfig
		fx.Annotate(
			func(cfg *config.Config) *databaseManager.MySQLConfig {
				return &databaseManager.MySQLConfig{
					Host:      cfg.Database.Host,
					Port:      cfg.Database.Port,
					User:      cfg.Database.User,
					Password:  cfg.Database.Password,
					Name:      cfg.Database.Name,
					Charset:   "utf8mb4",
					ParseTime: true,
					Loc:       "Local",
				}
			},
			fx.ResultTags(`name:"mysqlConfig"`),
		),
		// 提供 DatabaseManager 實例
		fx.Annotate(
			func(lc fx.Lifecycle, config *databaseManager.MySQLConfig) (databaseManager.DatabaseManager, error) {
				return databaseManager.ProvideMySQLDatabaseManager(lc, config)
			},
			fx.ParamTags(``, `name:"mysqlConfig"`),
		),
	),
)

// RedisModule Redis 模組
var RedisModule = fx.Options(
	fx.Provide(
		// 提供 Redis 配置
		func(cfg *config.Config) *redis.RedisConfig {
			return &redis.RedisConfig{
				Addr:     cfg.Redis.Addr,
				Username: cfg.Redis.Username,
				Password: cfg.Redis.Password,
				DB:       cfg.Redis.DB,
			}
		},
		// 提供 Redis 客戶端和管理器
		redis.ProvideRedisClient,
		redis.ProvideRedisManager,
	),
)

// WebSocketModule WebSocket 模組
var WebSocketModule = fx.Options(
	fx.Provide(
		// 提供 WebSocket 管理器
		func(authService service.AuthService) *websocketManager.Manager {
			return websocketManager.NewManager(authService.ValidateToken)
		},
		// 提供 WebSocket 處理程序
		websocketManager.NewWebSocketHandler,
	),
	// 啟動 WebSocket 管理器
	fx.Invoke(
		func(lc fx.Lifecycle, manager *websocketManager.Manager) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go manager.Start(ctx)
					return nil
				},
				OnStop: func(ctx context.Context) error {
					manager.Shutdown()
					return nil
				},
			})
		},
	),
)

// LoggerModule 日誌模組
var LoggerModule = fx.Provide(logger.NewLogger)

// 整合的核心模組，包含所有基礎設施
var Module = fx.Options(
	nacosManager.Module,
	DatabaseModule,
	RedisModule,
	WebSocketModule,
	LoggerModule,
)
