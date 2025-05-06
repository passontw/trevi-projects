package core

import (
	"context"

	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/internal/lottery_service/service"
	"g38_lottery_service/pkg/databaseManager"
	"g38_lottery_service/pkg/dealerWebsocket"
	"g38_lottery_service/pkg/logger"
	"g38_lottery_service/pkg/nacosManager"
	redis "g38_lottery_service/pkg/redisManager"
	rocket "g38_lottery_service/pkg/rocketManager"

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
		// 提供荷官 WebSocket 管理器，使用空的驗證函數
		func() *dealerWebsocket.Manager {
			// 使用一個始終返回成功的驗證函數
			tokenValidator := func(token string) (uint, error) {
				return 1, nil // 假設用戶ID為1
			}
			return dealerWebsocket.NewManager(tokenValidator)
		},
		// 將 *dealerWebsocket.Manager 標記為實現 service.WebsocketManager 介面
		fx.Annotate(
			func(manager *dealerWebsocket.Manager) service.WebsocketManager {
				return manager // 將現有的 Manager 實例作為 WebsocketManager 介面返回
			},
			fx.As(new(service.WebsocketManager)),
		),
		// 提供荷官 WebSocket 處理程序，使用空的驗證函數
		func(manager *dealerWebsocket.Manager, gameService service.GameService) *dealerWebsocket.WebSocketHandler {
			// 使用一個始終返回成功的驗證函數
			tokenValidator := func(token string) (uint, error) {
				return 1, nil // 假設用戶ID為1
			}

			// 創建消息處理器
			messageHandler := dealerWebsocket.NewDealerMessageHandler(gameService)

			// 設置消息處理器
			manager.SetMessageHandler(messageHandler)

			return dealerWebsocket.NewWebSocketHandler(manager, tokenValidator)
		},
		// 提供玩家 WebSocket 管理器，使用空的驗證函數
		fx.Annotate(
			func() *dealerWebsocket.Manager {
				// 使用一個始終返回成功的驗證函數
				tokenValidator := func(token string) (uint, error) {
					return 1, nil // 假設用戶ID為1
				}
				return dealerWebsocket.NewManager(tokenValidator)
			},
			fx.ResultTags(`name:"playerManager"`),
		),
		// 提供玩家 WebSocket 處理程序，使用空的驗證函數
		fx.Annotate(
			func(manager *dealerWebsocket.Manager) *dealerWebsocket.WebSocketHandler {
				// 使用一個始終返回成功的驗證函數
				tokenValidator := func(token string) (uint, error) {
					return 1, nil // 假設用戶ID為1
				}
				return dealerWebsocket.NewWebSocketHandler(manager, tokenValidator)
			},
			fx.ParamTags(`name:"playerManager"`),
			fx.ResultTags(`name:"playerWSHandler"`),
		),
	),
	// 啟動荷官 WebSocket 管理器
	fx.Invoke(
		func(lc fx.Lifecycle, manager *dealerWebsocket.Manager) {
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
	// 啟動玩家 WebSocket 管理器
	fx.Invoke(
		fx.Annotate(
			func(lc fx.Lifecycle, manager *dealerWebsocket.Manager) {
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
			fx.ParamTags(``, `name:"playerManager"`),
		),
	),
)

// LoggerModule 日誌模組
var LoggerModule = fx.Provide(logger.NewLogger)

// RocketMQModule RocketMQ 模組
var RocketMQModule = fx.Options(
	fx.Provide(
		// 提供 RocketMQ 配置
		func(cfg *config.Config) *rocket.RocketConfig {
			return &rocket.RocketConfig{
				NameServers:   cfg.RocketMQ.NameServers,
				AccessKey:     cfg.RocketMQ.AccessKey,
				SecretKey:     cfg.RocketMQ.SecretKey,
				ProducerGroup: cfg.RocketMQ.ProducerGroup,
				ConsumerGroup: cfg.RocketMQ.ConsumerGroup,
			}
		},
		// 提供 RocketMQ 管理器
		rocket.ProvideRocketManager,
	),
)

// 整合的核心模組，包含所有基礎設施
var Module = fx.Options(
	nacosManager.Module,
	DatabaseModule,
	RedisModule,
	WebSocketModule,
	RocketMQModule,
	LoggerModule,
)
