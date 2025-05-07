package gameflow

import (
	"context"

	mq "g38_lottery_service/internal/lottery_service/mq"
	"g38_lottery_service/pkg/databaseManager"
	redis "g38_lottery_service/pkg/redisManager"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ProvideRedisRepository 提供 Redis 實現的遊戲儲存庫
func ProvideRedisRepository(redisManager redis.RedisManager, logger *zap.Logger) *RedisRepository {
	return NewRedisRepository(redisManager, logger)
}

// ProvideTiDBRepository 提供 TiDB 實現的持久化儲存庫
func ProvideTiDBRepository(dbManager databaseManager.DatabaseManager, logger *zap.Logger) *TiDBRepository {
	return NewTiDBRepository(dbManager, logger)
}

// ProvideCompositeRepository 提供組合儲存庫
func ProvideCompositeRepository(redisRepo *RedisRepository, tidbRepo *TiDBRepository) *CompositeRepository {
	return NewCompositeRepository(redisRepo, tidbRepo)
}

// ProvideGameRepository 提供遊戲儲存庫實例
func ProvideGameRepository(compositeRepo *CompositeRepository) GameRepository {
	return compositeRepo
}

// ProvideGameManager 提供遊戲流程管理器
func ProvideGameManager(repo GameRepository, logger *zap.Logger, mqProducer *mq.MessageProducer) (*GameManager, error) {
	// 創建管理器
	manager := NewGameManager(repo, logger, mqProducer)
	return manager, nil
}

// StartGameManager 啟動遊戲流程管理器
func StartGameManager(lc fx.Lifecycle, manager *GameManager, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("初始化遊戲流程管理器")
			return manager.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("關閉遊戲流程管理器")
			return nil
		},
	})
}

// Module 提供 FX 模塊
var Module = fx.Options(
	// 提供遊戲儲存庫
	fx.Provide(ProvideRedisRepository),
	fx.Provide(ProvideTiDBRepository),
	fx.Provide(ProvideCompositeRepository),
	fx.Provide(ProvideGameRepository),

	// 提供遊戲流程管理器
	fx.Provide(ProvideGameManager),

	// 啟動遊戲流程管理器
	fx.Invoke(StartGameManager),
)
