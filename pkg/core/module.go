package core

import (
	"fmt"
	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/pkg/databaseManager"
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
			func(cfg *config.AppConfig) *databaseManager.MySQLConfig {
				return &databaseManager.MySQLConfig{
					Host:      cfg.Database.Host,
					Port:      cfg.Database.Port,
					User:      cfg.Database.Username,
					Password:  cfg.Database.Password,
					Name:      cfg.Database.DBName,
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
		func(cfg *config.AppConfig) *redis.RedisConfig {
			addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
			return &redis.RedisConfig{
				Addr:     addr,
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

// LoggerModule 日誌模組
var LoggerModule = fx.Provide(logger.NewLogger)

// RocketMQModule RocketMQ 模組
var RocketMQModule = fx.Options(
	fx.Provide(
		// 提供 RocketMQ 配置
		func(cfg *config.AppConfig) *rocket.RocketConfig {
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
	RocketMQModule,
	LoggerModule,
)
