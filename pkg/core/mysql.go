package core

import (
	"context"
	"fmt"
	"os"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/service"
	"g38_lottery_service/pkg/databaseManager"
	"g38_lottery_service/pkg/logger"
	"g38_lottery_service/pkg/nacosManager"
	"g38_lottery_service/pkg/redisManager"
	"g38_lottery_service/pkg/websocketManager"

	"go.uber.org/fx"
)

// DatabaseModule 數據庫模組
var DatabaseModule = fx.Options(
	fx.Provide(
		// 基於 Config 轉換為 MySQLConfig
		fx.Annotate(
			func(cfg *config.Config) *databaseManager.MySQLConfig {
				// 檢查端口是否有效，如果無效則使用默認值
				port := cfg.Database.Port
				if port <= 0 || port > 65535 {
					port = 3306 // 使用默認的MySQL端口
				}

				// 從環境變量獲取配置
				host := os.Getenv("MYSQL_HOST")
				if host == "" {
					host = cfg.Database.Host
				}

				user := os.Getenv("MYSQL_USER")
				if user == "" {
					user = cfg.Database.User
				}

				password := os.Getenv("MYSQL_PASSWORD")
				if password == "" {
					password = cfg.Database.Password
				}

				dbName := os.Getenv("MYSQL_DATABASE")
				if dbName == "" {
					dbName = cfg.Database.Name
				}

				// 記錄最終使用的數據庫配置
				fmt.Printf("最終的MySQL配置: Host=%s, Port=%d, User=%s, DB=%s, 密碼%s\n",
					host, port, user, dbName,
					func(pwd string) string {
						if pwd == "" {
							return "未設置"
						}
						return "已設置"
					}(password))

				return &databaseManager.MySQLConfig{
					Host:      host,
					Port:      port,
					User:      user,
					Password:  password,
					Name:      dbName,
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
		func(cfg *config.Config) *redisManager.RedisConfig {
			return &redisManager.RedisConfig{
				Addr:     cfg.Redis.Addr,
				Username: cfg.Redis.Username,
				Password: cfg.Redis.Password,
				DB:       cfg.Redis.DB,
			}
		},
		// 提供 Redis 客戶端和管理器
		redisManager.ProvideRedisClient,
		redisManager.ProvideRedisManager,
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
