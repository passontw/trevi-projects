package gameflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redis "g38_lottery_service/pkg/redisManager"

	"go.uber.org/zap"
)

const (
	// Redis 鍵前綴
	redisKeyPrefix          = "g38:lottery:gameflow:"
	redisCurrentGameKeyTpl  = redisKeyPrefix + "room:%s:current"      // 當前遊戲，格式 room:{roomID}:current
	redisLuckyBallsKeyTpl   = redisKeyPrefix + "room:%s:lucky_balls"  // 幸運號碼球，格式 room:{roomID}:lucky_balls
	redisGameHistoryKeyTpl  = redisKeyPrefix + "history:%s"           // 遊戲歷史記錄，格式 history:gameID
	redisStageTimeoutKeyTpl = redisKeyPrefix + "timeout:%s:%s"        // 階段超時時間，格式 timeout:gameID:stage
	redisHistoryListKeyTpl  = redisKeyPrefix + "room:%s:history_list" // 房間遊戲歷史ID列表，格式 room:{roomID}:history_list
)

// RedisRepository 使用 Redis 實現的遊戲數據存儲庫
type RedisRepository struct {
	redisClient redis.RedisManager
	logger      *zap.Logger
	defaultRoom string // 默認房間 ID
}

// NewRedisRepository 創建新的 Redis 遊戲數據存儲庫
func NewRedisRepository(redisClient redis.RedisManager, logger *zap.Logger) *RedisRepository {
	return &RedisRepository{
		redisClient: redisClient,
		logger:      logger.With(zap.String("component", "redis_repository")),
		defaultRoom: "SG01", // 預設使用 SG01 房間
	}
}

// getCurrentGameKey 獲取當前遊戲的 Redis 鍵
func (r *RedisRepository) getCurrentGameKey(roomID string) string {
	if roomID == "" {
		roomID = r.defaultRoom
	}
	return fmt.Sprintf(redisCurrentGameKeyTpl, roomID)
}

// getLuckyBallsKey 獲取幸運號碼球的 Redis 鍵
func (r *RedisRepository) getLuckyBallsKey(roomID string) string {
	if roomID == "" {
		roomID = r.defaultRoom
	}
	return fmt.Sprintf(redisLuckyBallsKeyTpl, roomID)
}

// getHistoryListKey 獲取歷史列表的 Redis 鍵
func (r *RedisRepository) getHistoryListKey(roomID string) string {
	if roomID == "" {
		roomID = r.defaultRoom
	}
	return fmt.Sprintf(redisHistoryListKeyTpl, roomID)
}

// SaveGame 保存遊戲數據到 Redis
func (r *RedisRepository) SaveGame(ctx context.Context, game *GameData) error {
	// 從遊戲ID中提取房間ID
	roomID := GetRoomIDFromGameID(game.GameID)

	// 序列化遊戲數據
	gameJSON, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("序列化遊戲數據失敗: %w", err)
	}

	// 保存到 Redis，過期時間設為 24 小時
	key := r.getCurrentGameKey(roomID)
	err = r.redisClient.Set(ctx, key, gameJSON, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("保存遊戲數據到 Redis 失敗: %w", err)
	}

	r.logger.Debug("已保存遊戲數據到 Redis",
		zap.String("roomID", roomID),
		zap.String("gameID", game.GameID),
		zap.String("stage", string(game.CurrentStage)))

	return nil
}

// GetCurrentGame 從 Redis 獲取當前遊戲數據
func (r *RedisRepository) GetCurrentGame(ctx context.Context) (*GameData, error) {
	return r.GetCurrentGameByRoom(ctx, r.defaultRoom)
}

// GetCurrentGameByRoom 從 Redis 獲取特定房間的當前遊戲數據
func (r *RedisRepository) GetCurrentGameByRoom(ctx context.Context, roomID string) (*GameData, error) {
	r.logger.Debug("嘗試從 Redis 獲取當前遊戲", zap.String("roomID", roomID))

	// 獲取當前遊戲數據
	key := r.getCurrentGameKey(roomID)
	gameJSON, err := r.redisClient.Get(ctx, key)
	if err != nil {
		// 檢查是否是「不存在」錯誤
		if redis.IsKeyNotExist(err) {
			r.logger.Info("Redis 中沒有當前遊戲", zap.String("roomID", roomID))
			return nil, nil
		}
		r.logger.Error("無法從 Redis 獲取當前遊戲", zap.Error(err), zap.String("roomID", roomID))
		return nil, fmt.Errorf("無法獲取當前遊戲: %w", err)
	}

	// 解碼遊戲數據
	var gameData GameData
	if err := json.Unmarshal([]byte(gameJSON), &gameData); err != nil {
		r.logger.Error("無法解碼遊戲數據", zap.Error(err))
		return nil, fmt.Errorf("無法解碼遊戲數據: %w", err)
	}

	r.logger.Debug("成功從 Redis 獲取遊戲數據",
		zap.String("roomID", roomID),
		zap.String("gameID", gameData.GameID),
		zap.String("stage", string(gameData.CurrentStage)))

	return &gameData, nil
}

// DeleteCurrentGame 從 Redis 刪除當前遊戲
func (r *RedisRepository) DeleteCurrentGame(ctx context.Context) error {
	return r.DeleteCurrentGameByRoom(ctx, r.defaultRoom)
}

// DeleteCurrentGameByRoom 從 Redis 刪除特定房間的當前遊戲
func (r *RedisRepository) DeleteCurrentGameByRoom(ctx context.Context, roomID string) error {
	key := r.getCurrentGameKey(roomID)
	err := r.redisClient.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("從 Redis 刪除當前遊戲失敗: %w", err)
	}

	r.logger.Debug("已從 Redis 刪除當前遊戲", zap.String("roomID", roomID))
	return nil
}

// SaveGameHistory 保存遊戲歷史記錄
func (r *RedisRepository) SaveGameHistory(ctx context.Context, game *GameData) error {
	// 從遊戲ID中提取房間ID
	roomID := GetRoomIDFromGameID(game.GameID)

	// 序列化遊戲數據
	gameJSON, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("序列化遊戲歷史記錄失敗: %w", err)
	}

	// 保存到 Redis，過期時間設為 30 天
	historyKey := fmt.Sprintf(redisGameHistoryKeyTpl, game.GameID)
	err = r.redisClient.Set(ctx, historyKey, gameJSON, 30*24*time.Hour)
	if err != nil {
		return fmt.Errorf("保存遊戲歷史記錄到 Redis 失敗: %w", err)
	}

	// 同時添加到對應房間的歷史記錄列表
	historyListKey := r.getHistoryListKey(roomID)
	err = r.redisClient.LPush(ctx, historyListKey, game.GameID)
	if err != nil {
		r.logger.Warn("添加遊戲ID到歷史列表失敗",
			zap.Error(err),
			zap.String("roomID", roomID),
			zap.String("gameID", game.GameID))
		// 這不是致命錯誤，可以繼續執行
	}

	r.logger.Debug("已保存遊戲歷史記錄",
		zap.String("roomID", roomID),
		zap.String("gameID", game.GameID))

	return nil
}

// GetLuckyBalls 獲取當前的幸運號碼球
func (r *RedisRepository) GetLuckyBalls(ctx context.Context) ([]Ball, error) {
	return r.GetLuckyBallsByRoom(ctx, r.defaultRoom)
}

// GetLuckyBallsByRoom 獲取特定房間的幸運號碼球
func (r *RedisRepository) GetLuckyBallsByRoom(ctx context.Context, roomID string) ([]Ball, error) {
	r.logger.Debug("嘗試從 Redis 獲取幸運號碼球", zap.String("roomID", roomID))

	// 從 Redis 獲取幸運號碼球
	key := r.getLuckyBallsKey(roomID)
	ballsJSON, err := r.redisClient.Get(ctx, key)

	// 如果鍵不存在，返回空數組而不是錯誤
	if redis.IsKeyNotExist(err) {
		r.logger.Debug("Redis 中未找到幸運號碼球，返回空數組", zap.String("roomID", roomID))
		return []Ball{}, nil
	}

	if err != nil {
		r.logger.Error("從 Redis 獲取幸運號碼球失敗", zap.Error(err), zap.String("roomID", roomID))
		return nil, fmt.Errorf("從 Redis 獲取幸運號碼球失敗: %w", err)
	}

	// 反序列化數據
	var balls []Ball
	err = json.Unmarshal([]byte(ballsJSON), &balls)
	if err != nil {
		r.logger.Error("反序列化幸運號碼球數據失敗", zap.Error(err))
		return nil, fmt.Errorf("反序列化幸運號碼球數據失敗: %w", err)
	}

	r.logger.Debug("已從 Redis 獲取幸運號碼球",
		zap.Int("數量", len(balls)),
		zap.String("roomID", roomID))
	return balls, nil
}

// SaveLuckyBalls 保存新的幸運號碼球
func (r *RedisRepository) SaveLuckyBalls(ctx context.Context, balls []Ball) error {
	// 預設使用默認房間
	return r.SaveLuckyBallsToRoom(ctx, r.defaultRoom, balls)
}

// SaveLuckyBallsToRoom 保存新的幸運號碼球到特定房間
func (r *RedisRepository) SaveLuckyBallsToRoom(ctx context.Context, roomID string, balls []Ball) error {
	key := r.getLuckyBallsKey(roomID)

	// 先刪除現有的幸運號碼球資料
	err := r.redisClient.Delete(ctx, key)
	if err != nil {
		r.logger.Warn("刪除現有幸運號碼球資料失敗",
			zap.Error(err),
			zap.String("key", key),
			zap.String("roomID", roomID))
		// 繼續執行，不要因為刪除失敗而停止
	} else {
		r.logger.Debug("已刪除現有幸運號碼球資料",
			zap.String("key", key),
			zap.String("roomID", roomID))
	}

	// 序列化球數據
	ballsJSON, err := json.Marshal(balls)
	if err != nil {
		return fmt.Errorf("序列化幸運號碼球數據失敗: %w", err)
	}

	// 保存到 Redis，設為永久保存
	err = r.redisClient.Set(ctx, key, ballsJSON, 0)
	if err != nil {
		return fmt.Errorf("保存幸運號碼球到 Redis 失敗: %w", err)
	}

	r.logger.Info("已保存幸運號碼球到 Redis",
		zap.Int("數量", len(balls)),
		zap.String("key", key),
		zap.String("roomID", roomID))

	// 驗證寫入是否成功
	verifyJSON, err := r.redisClient.Get(ctx, key)
	if err != nil {
		r.logger.Warn("無法驗證幸運號碼球寫入", zap.Error(err))
	} else if verifyJSON != string(ballsJSON) {
		r.logger.Warn("幸運號碼球寫入驗證失敗，資料不匹配")
	} else {
		r.logger.Debug("幸運號碼球寫入驗證成功")
	}

	return nil
}

// GetRecentGameHistories 獲取最近的遊戲歷史記錄
func (r *RedisRepository) GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error) {
	return r.GetRecentGameHistoriesByRoom(ctx, r.defaultRoom, limit)
}

// GetRecentGameHistoriesByRoom 獲取特定房間最近的遊戲歷史記錄
func (r *RedisRepository) GetRecentGameHistoriesByRoom(ctx context.Context, roomID string, limit int) ([]*GameData, error) {
	// 從 Redis 獲取最近的遊戲ID列表
	historyListKey := r.getHistoryListKey(roomID)
	gameIDs, err := r.redisClient.LRange(ctx, historyListKey, 0, int64(limit-1))
	if err != nil {
		return nil, fmt.Errorf("獲取遊戲歷史ID列表失敗: %w", err)
	}

	var games []*GameData
	for _, gameID := range gameIDs {
		historyKey := fmt.Sprintf(redisGameHistoryKeyTpl, gameID)
		gameJSON, err := r.redisClient.Get(ctx, historyKey)

		// 如果某個遊戲記錄不存在，則跳過
		if redis.IsKeyNotExist(err) {
			continue
		}

		if err != nil {
			r.logger.Warn("獲取特定遊戲歷史記錄失敗",
				zap.String("gameID", gameID),
				zap.Error(err))
			continue
		}

		// 反序列化遊戲數據
		var game GameData
		if err := json.Unmarshal([]byte(gameJSON), &game); err != nil {
			r.logger.Warn("反序列化遊戲歷史記錄失敗",
				zap.String("gameID", gameID),
				zap.Error(err))
			continue
		}

		games = append(games, &game)
	}

	r.logger.Debug("已獲取最近遊戲歷史記錄",
		zap.String("roomID", roomID),
		zap.Int("請求數量", limit),
		zap.Int("實際獲取", len(games)))

	return games, nil
}

// 實現 CacheRepository 的擴展方法

// SaveStageTimeout 保存階段超時時間
func (r *RedisRepository) SaveStageTimeout(ctx context.Context, gameID string, stage GameStage, timeout int64) error {
	key := fmt.Sprintf(redisStageTimeoutKeyTpl, gameID, string(stage))
	err := r.redisClient.Set(ctx, key, timeout, time.Duration(timeout)*time.Millisecond)
	if err != nil {
		return fmt.Errorf("保存階段超時時間失敗: %w", err)
	}
	return nil
}

// GetStageTimeout 獲取階段超時時間
func (r *RedisRepository) GetStageTimeout(ctx context.Context, gameID string, stage GameStage) (int64, error) {
	key := fmt.Sprintf(redisStageTimeoutKeyTpl, gameID, string(stage))
	timeoutStr, err := r.redisClient.Get(ctx, key)

	if redis.IsKeyNotExist(err) {
		return 0, nil // 不存在則返回0
	}

	if err != nil {
		return 0, fmt.Errorf("獲取階段超時時間失敗: %w", err)
	}

	var timeout int64
	if err := json.Unmarshal([]byte(timeoutStr), &timeout); err != nil {
		return 0, fmt.Errorf("解析階段超時時間失敗: %w", err)
	}

	return timeout, nil
}

// DeleteStageTimeout 刪除階段超時時間
func (r *RedisRepository) DeleteStageTimeout(ctx context.Context, gameID string, stage GameStage) error {
	key := fmt.Sprintf(redisStageTimeoutKeyTpl, gameID, string(stage))
	err := r.redisClient.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("刪除階段超時時間失敗: %w", err)
	}
	return nil
}

// 實現 PersistentRepository 的方法 (Redis 模擬)

// GetGameByID 從歷史記錄中獲取特定ID的遊戲
func (r *RedisRepository) GetGameByID(ctx context.Context, gameID string) (*GameData, error) {
	historyKey := fmt.Sprintf(redisGameHistoryKeyTpl, gameID)
	gameJSON, err := r.redisClient.Get(ctx, historyKey)

	if redis.IsKeyNotExist(err) {
		return nil, ErrGameNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("獲取特定遊戲記錄失敗: %w", err)
	}

	var game GameData
	if err := json.Unmarshal([]byte(gameJSON), &game); err != nil {
		return nil, fmt.Errorf("反序列化遊戲記錄失敗: %w", err)
	}

	return &game, nil
}

// GetGamesByDateRange 從歷史記錄中獲取特定日期範圍的遊戲
func (r *RedisRepository) GetGamesByDateRange(ctx context.Context, startDate, endDate string) ([]*GameData, error) {
	// 在 Redis 中實現日期範圍查詢比較複雜
	// 這裡僅返回最近的幾個遊戲作為模擬
	return r.GetRecentGameHistories(ctx, 10)
}

// GetTotalGamesCount 獲取總遊戲數量
func (r *RedisRepository) GetTotalGamesCount(ctx context.Context) (int64, error) {
	return r.GetTotalGamesCountByRoom(ctx, r.defaultRoom)
}

// GetTotalGamesCountByRoom 獲取特定房間總遊戲數量
func (r *RedisRepository) GetTotalGamesCountByRoom(ctx context.Context, roomID string) (int64, error) {
	// 從 Redis 獲取歷史遊戲ID列表的長度
	historyListKey := r.getHistoryListKey(roomID)
	count, err := r.redisClient.LRange(ctx, historyListKey, 0, -1)
	if err != nil {
		return 0, fmt.Errorf("獲取遊戲歷史總數失敗: %w", err)
	}

	return int64(len(count)), nil
}

// GetCancelledGamesCount 獲取取消的遊戲數量
func (r *RedisRepository) GetCancelledGamesCount(ctx context.Context) (int64, error) {
	// 這個功能需要遍歷所有遊戲記錄，在 Redis 中不太實用
	// 實際應用中應該使用數據庫來實現統計功能

	// 這裡只是模擬返回一個固定值
	return 0, nil
}
