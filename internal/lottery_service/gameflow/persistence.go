package gameflow

import (
	"context"
)

// GameRepository 遊戲數據存儲庫接口
type GameRepository interface {
	// 基本遊戲管理功能
	SaveGame(ctx context.Context, game *GameData) error
	GetCurrentGame(ctx context.Context) (*GameData, error)
	DeleteCurrentGame(ctx context.Context) error

	// 多房間版本的基本功能
	GetCurrentGameByRoom(ctx context.Context, roomID string) (*GameData, error)
	DeleteCurrentGameByRoom(ctx context.Context, roomID string) error

	// 幸運號碼球管理
	GetLuckyBalls(ctx context.Context) ([]Ball, error)
	SaveLuckyBalls(ctx context.Context, balls []Ball) error

	// 多房間版本的幸運號碼球管理
	GetLuckyBallsByRoom(ctx context.Context, roomID string) ([]Ball, error)
	SaveLuckyBallsToRoom(ctx context.Context, roomID string, balls []Ball) error

	// 遊戲歷史記錄管理
	SaveGameHistory(ctx context.Context, game *GameData) error
	GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error)

	// 多房間版本的遊戲歷史記錄管理
	GetRecentGameHistoriesByRoom(ctx context.Context, roomID string, limit int) ([]*GameData, error)

	// 統計功能
	GetTotalGamesCount(ctx context.Context) (int64, error)
	GetCancelledGamesCount(ctx context.Context) (int64, error)

	// 多房間版本的統計功能
	GetTotalGamesCountByRoom(ctx context.Context, roomID string) (int64, error)
}

// PersistentRepository 永久存儲庫接口
type PersistentRepository interface {
	// 根據ID獲取遊戲數據
	GetGameByID(ctx context.Context, gameID string) (*GameData, error)

	// 獲取指定日期範圍的遊戲數據
	GetGamesByDateRange(ctx context.Context, startDate, endDate string) ([]*GameData, error)

	// 新增方法以支持 CompositeRepository
	SaveGameHistory(ctx context.Context, game *GameData) error
	GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error)
	GetRecentGameHistoriesByRoom(ctx context.Context, roomID string, limit int) ([]*GameData, error)
	GetTotalGamesCount(ctx context.Context) (int64, error)
	GetTotalGamesCountByRoom(ctx context.Context, roomID string) (int64, error)
	GetCancelledGamesCount(ctx context.Context) (int64, error)
}

// CacheRepository 快取存儲庫接口
type CacheRepository interface {
	// 存儲和獲取階段超時時間
	SaveStageTimeout(ctx context.Context, gameID string, stage GameStage, timeout int64) error
	GetStageTimeout(ctx context.Context, gameID string, stage GameStage) (int64, error)
	DeleteStageTimeout(ctx context.Context, gameID string, stage GameStage) error

	// 新增方法以支持 CompositeRepository
	SaveGame(ctx context.Context, game *GameData) error
	GetCurrentGame(ctx context.Context) (*GameData, error)
	GetCurrentGameByRoom(ctx context.Context, roomID string) (*GameData, error)
	DeleteCurrentGame(ctx context.Context) error
	DeleteCurrentGameByRoom(ctx context.Context, roomID string) error
	GetLuckyBalls(ctx context.Context) ([]Ball, error)
	GetLuckyBallsByRoom(ctx context.Context, roomID string) ([]Ball, error)
	SaveLuckyBalls(ctx context.Context, balls []Ball) error
	SaveLuckyBallsToRoom(ctx context.Context, roomID string, balls []Ball) error
}

// CompositeRepository 組合儲存庫，同時使用緩存和持久化儲存
type CompositeRepository struct {
	cache      CacheRepository
	persistent PersistentRepository
}

// NewCompositeRepository 創建一個組合儲存庫
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

// GetCurrentGameByRoom 從緩存獲取特定房間的當前遊戲
func (r *CompositeRepository) GetCurrentGameByRoom(ctx context.Context, roomID string) (*GameData, error) {
	return r.cache.GetCurrentGameByRoom(ctx, roomID)
}

// DeleteCurrentGame 從緩存刪除當前遊戲
func (r *CompositeRepository) DeleteCurrentGame(ctx context.Context) error {
	return r.cache.DeleteCurrentGame(ctx)
}

// DeleteCurrentGameByRoom 從緩存刪除特定房間的當前遊戲
func (r *CompositeRepository) DeleteCurrentGameByRoom(ctx context.Context, roomID string) error {
	return r.cache.DeleteCurrentGameByRoom(ctx, roomID)
}

// SaveGameHistory 保存遊戲歷史到永久儲存
func (r *CompositeRepository) SaveGameHistory(ctx context.Context, game *GameData) error {
	return r.persistent.SaveGameHistory(ctx, game)
}

// GetLuckyBalls 從緩存獲取幸運號碼球
func (r *CompositeRepository) GetLuckyBalls(ctx context.Context) ([]Ball, error) {
	return r.cache.GetLuckyBalls(ctx)
}

// GetLuckyBallsByRoom 從緩存獲取特定房間的幸運號碼球
func (r *CompositeRepository) GetLuckyBallsByRoom(ctx context.Context, roomID string) ([]Ball, error) {
	return r.cache.GetLuckyBallsByRoom(ctx, roomID)
}

// SaveLuckyBalls 保存幸運號碼球到緩存
func (r *CompositeRepository) SaveLuckyBalls(ctx context.Context, balls []Ball) error {
	return r.cache.SaveLuckyBalls(ctx, balls)
}

// SaveLuckyBallsToRoom 保存幸運號碼球到特定房間
func (r *CompositeRepository) SaveLuckyBallsToRoom(ctx context.Context, roomID string, balls []Ball) error {
	return r.cache.SaveLuckyBallsToRoom(ctx, roomID, balls)
}

// GetRecentGameHistories 從永久儲存獲取遊戲歷史記錄
func (r *CompositeRepository) GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error) {
	return r.persistent.GetRecentGameHistories(ctx, limit)
}

// GetRecentGameHistoriesByRoom 從永久儲存獲取特定房間的遊戲歷史記錄
func (r *CompositeRepository) GetRecentGameHistoriesByRoom(ctx context.Context, roomID string, limit int) ([]*GameData, error) {
	return r.persistent.GetRecentGameHistoriesByRoom(ctx, roomID, limit)
}

// GetGameByID 從永久儲存獲取特定ID的遊戲
func (r *CompositeRepository) GetGameByID(ctx context.Context, gameID string) (*GameData, error) {
	return r.persistent.GetGameByID(ctx, gameID)
}

// GetGamesByDateRange 從永久儲存獲取特定日期範圍的遊戲
func (r *CompositeRepository) GetGamesByDateRange(ctx context.Context, startDate, endDate string) ([]*GameData, error) {
	return r.persistent.GetGamesByDateRange(ctx, startDate, endDate)
}

// GetTotalGamesCount 獲取遊戲總數
func (r *CompositeRepository) GetTotalGamesCount(ctx context.Context) (int64, error) {
	return r.persistent.GetTotalGamesCount(ctx)
}

// GetTotalGamesCountByRoom 獲取特定房間的遊戲總數
func (r *CompositeRepository) GetTotalGamesCountByRoom(ctx context.Context, roomID string) (int64, error) {
	return r.persistent.GetTotalGamesCountByRoom(ctx, roomID)
}

// GetCancelledGamesCount 獲取取消的遊戲數
func (r *CompositeRepository) GetCancelledGamesCount(ctx context.Context) (int64, error) {
	return r.persistent.GetCancelledGamesCount(ctx)
}

// SaveStageTimeout 保存階段超時時間
func (r *CompositeRepository) SaveStageTimeout(ctx context.Context, gameID string, stage GameStage, timeout int64) error {
	return r.cache.SaveStageTimeout(ctx, gameID, stage, timeout)
}

// GetStageTimeout 獲取階段超時時間
func (r *CompositeRepository) GetStageTimeout(ctx context.Context, gameID string, stage GameStage) (int64, error) {
	return r.cache.GetStageTimeout(ctx, gameID, stage)
}

// DeleteStageTimeout 刪除階段超時時間
func (r *CompositeRepository) DeleteStageTimeout(ctx context.Context, gameID string, stage GameStage) error {
	return r.cache.DeleteStageTimeout(ctx, gameID, stage)
}
