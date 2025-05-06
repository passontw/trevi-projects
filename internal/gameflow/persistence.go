package gameflow

import (
	"context"
)

// GameRepository 定義遊戲數據儲存庫介面
type GameRepository interface {
	// SaveGame 保存遊戲數據
	SaveGame(ctx context.Context, game *GameData) error

	// GetCurrentGame 獲取當前正在進行的遊戲
	GetCurrentGame(ctx context.Context) (*GameData, error)

	// DeleteCurrentGame 刪除當前遊戲(通常在遊戲結束後)
	DeleteCurrentGame(ctx context.Context) error

	// SaveGameHistory 保存遊戲歷史記錄(通常在遊戲結束後保存到永久儲存)
	SaveGameHistory(ctx context.Context, game *GameData) error

	// GetLuckyBalls 獲取當前的幸運號碼球
	GetLuckyBalls(ctx context.Context) ([]Ball, error)

	// SaveLuckyBalls 保存新的幸運號碼球
	SaveLuckyBalls(ctx context.Context, balls []Ball) error

	// GetRecentGameHistories 獲取最近的遊戲歷史記錄
	GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error)
}

// CacheRepository 緩存儲存庫介面，主要用於Redis實現
type CacheRepository interface {
	// 基礎方法
	SaveGame(ctx context.Context, game *GameData) error
	GetCurrentGame(ctx context.Context) (*GameData, error)
	DeleteCurrentGame(ctx context.Context) error
	GetLuckyBalls(ctx context.Context) ([]Ball, error)
	SaveLuckyBalls(ctx context.Context, balls []Ball) error

	// 緩存特有方法
	SaveStageTimeout(ctx context.Context, gameID string, stage GameStage, timeout int64) error
	GetStageTimeout(ctx context.Context, gameID string, stage GameStage) (int64, error)
	DeleteStageTimeout(ctx context.Context, gameID string, stage GameStage) error
}

// PersistentRepository 持久化儲存庫介面，主要用於TiDB實現
type PersistentRepository interface {
	// 基礎方法
	SaveGameHistory(ctx context.Context, game *GameData) error
	GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error)

	// 持久化特有方法
	GetGameByID(ctx context.Context, gameID string) (*GameData, error)
	GetGamesByDateRange(ctx context.Context, startDate, endDate string) ([]*GameData, error)
	GetTotalGamesCount(ctx context.Context) (int64, error)
	GetCancelledGamesCount(ctx context.Context) (int64, error)
}

// CompositeRepository 組合儲存庫，同時使用緩存和持久化儲存
type CompositeRepository struct {
	cache      CacheRepository
	persistent PersistentRepository
}

// NewCompositeRepository 創建新的組合儲存庫
func NewCompositeRepository(cache CacheRepository, persistent PersistentRepository) *CompositeRepository {
	return &CompositeRepository{
		cache:      cache,
		persistent: persistent,
	}
}

// SaveGame 保存遊戲數據到緩存
func (r *CompositeRepository) SaveGame(ctx context.Context, game *GameData) error {
	return r.cache.SaveGame(ctx, game)
}

// GetCurrentGame 從緩存獲取當前遊戲
func (r *CompositeRepository) GetCurrentGame(ctx context.Context) (*GameData, error) {
	return r.cache.GetCurrentGame(ctx)
}

// DeleteCurrentGame 從緩存刪除當前遊戲
func (r *CompositeRepository) DeleteCurrentGame(ctx context.Context) error {
	return r.cache.DeleteCurrentGame(ctx)
}

// SaveGameHistory 保存遊戲歷史到持久儲存
func (r *CompositeRepository) SaveGameHistory(ctx context.Context, game *GameData) error {
	return r.persistent.SaveGameHistory(ctx, game)
}

// GetLuckyBalls 從緩存獲取幸運號碼球
func (r *CompositeRepository) GetLuckyBalls(ctx context.Context) ([]Ball, error) {
	return r.cache.GetLuckyBalls(ctx)
}

// SaveLuckyBalls 保存幸運號碼球到緩存
func (r *CompositeRepository) SaveLuckyBalls(ctx context.Context, balls []Ball) error {
	return r.cache.SaveLuckyBalls(ctx, balls)
}

// GetRecentGameHistories 從持久儲存獲取最近的遊戲歷史
func (r *CompositeRepository) GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error) {
	return r.persistent.GetRecentGameHistories(ctx, limit)
}

// GetGameByID 從持久儲存獲取特定ID的遊戲
func (r *CompositeRepository) GetGameByID(ctx context.Context, gameID string) (*GameData, error) {
	return r.persistent.GetGameByID(ctx, gameID)
}

// SaveStageTimeout 保存階段超時時間到緩存
func (r *CompositeRepository) SaveStageTimeout(ctx context.Context, gameID string, stage GameStage, timeout int64) error {
	return r.cache.SaveStageTimeout(ctx, gameID, stage, timeout)
}

// GetStageTimeout 從緩存獲取階段超時時間
func (r *CompositeRepository) GetStageTimeout(ctx context.Context, gameID string, stage GameStage) (int64, error) {
	return r.cache.GetStageTimeout(ctx, gameID, stage)
}

// DeleteStageTimeout 從緩存刪除階段超時時間
func (r *CompositeRepository) DeleteStageTimeout(ctx context.Context, gameID string, stage GameStage) error {
	return r.cache.DeleteStageTimeout(ctx, gameID, stage)
}
