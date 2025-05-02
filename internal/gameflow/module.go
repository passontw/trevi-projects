package gameflow

import (
	"context"

	redis "g38_lottery_service/pkg/redisManager"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ProvideRedisRepository 提供 Redis 實現的遊戲儲存庫
func ProvideRedisRepository(redisManager redis.RedisManager, logger *zap.Logger) *RedisRepository {
	return NewRedisRepository(redisManager, logger)
}

// ProvideGameRepository 提供遊戲儲存庫實例
func ProvideGameRepository(redisRepo *RedisRepository) GameRepository {
	// 這裡使用組合儲存庫，但目前我們只實現了 Redis 部分
	// 將來可以添加 TiDB 或其他持久化儲存的實現
	return redisRepo
}

// ProvideGameManager 提供遊戲流程管理器
func ProvideGameManager(repo GameRepository, logger *zap.Logger) (*GameManager, error) {
	// 創建管理器
	manager := NewGameManager(repo, logger)
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
	fx.Provide(ProvideGameRepository),

	// 提供遊戲流程管理器
	fx.Provide(ProvideGameManager),

	// 啟動遊戲流程管理器
	fx.Invoke(StartGameManager),
)
