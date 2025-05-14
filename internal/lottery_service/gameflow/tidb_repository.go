package gameflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"g38_lottery_service/pkg/databaseManager"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 資料庫模型結構定義
type GameModel struct {
	ID                    uint       `gorm:"primaryKey;autoIncrement"`
	GameID                string     `gorm:"column:game_id;type:varchar(100);uniqueIndex;not null"`
	RoomID                string     `gorm:"column:room_id;type:varchar(20);not null"`
	State                 string     `gorm:"column:state;type:varchar(30);not null"`
	StartTime             time.Time  `gorm:"column:start_time;type:timestamp;not null"`
	EndTime               *time.Time `gorm:"column:end_time;type:timestamp;null"`
	HasJackpot            bool       `gorm:"column:has_jackpot;type:boolean;not null;default:false"`
	JackpotAmount         float64    `gorm:"column:jackpot_amount;type:decimal(18,2);default:0"`
	ExtraBallCount        int        `gorm:"column:extra_ball_count;type:int;not null;default:3"`
	CurrentStateStartTime time.Time  `gorm:"column:current_state_start_time;type:timestamp;not null"`
	StageExpireTime       *time.Time `gorm:"column:stage_expire_time;type:timestamp;null"`
	MaxTimeout            int        `gorm:"column:max_timeout;type:int;not null;default:60"`
	TotalCards            int        `gorm:"column:total_cards;type:int;default:0"`
	TotalPlayers          int        `gorm:"column:total_players;type:int;default:0"`
	TotalBetAmount        float64    `gorm:"column:total_bet_amount;type:decimal(18,2);default:0"`
	TotalWinAmount        float64    `gorm:"column:total_win_amount;type:decimal(18,2);default:0"`
	Cancelled             bool       `gorm:"column:cancelled;type:boolean;not null;default:false"`
	CancelTime            *time.Time `gorm:"column:cancel_time;type:timestamp;null"`
	CancelledBy           string     `gorm:"column:cancelled_by;type:varchar(50);null"`
	CancelReason          string     `gorm:"column:cancel_reason;type:varchar(255);null"`
	GameSnapshot          string     `gorm:"column:game_snapshot;type:json;null"`

	// GORM 默認字段
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamp;not null"`
	UpdatedAt time.Time  `gorm:"column:updated_at;type:timestamp;not null"`
	DeletedAt *time.Time `gorm:"column:deleted_at;type:timestamp;null"`
}

// 設置表名
func (GameModel) TableName() string {
	return "games"
}

// Jackpot遊戲模型
type JackpotGameModel struct {
	ID        uint       `gorm:"primaryKey;autoIncrement"`
	GameID    string     `gorm:"column:game_id;type:varchar(100);not null"`
	JackpotID string     `gorm:"column:jackpot_id;type:varchar(50);uniqueIndex;not null"`
	StartTime *time.Time `gorm:"column:start_time;type:timestamp;null"`
	EndTime   *time.Time `gorm:"column:end_time;type:timestamp;null"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time  `gorm:"column:updated_at;type:timestamp;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

// 設置表名
func (JackpotGameModel) TableName() string {
	return "jackpot_games"
}

// 球資料庫模型
type DrawnBallModel struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	GameID     string    `gorm:"column:game_id;type:varchar(100);not null;index"`
	Number     int       `gorm:"column:number;type:int;not null"`
	Sequence   int       `gorm:"column:sequence;type:int;not null"`
	BallType   string    `gorm:"column:ball_type;type:enum('REGULAR','EXTRA','JACKPOT','LUCKY');not null"`
	DrawnTime  time.Time `gorm:"column:drawn_time;type:timestamp;not null"`
	Side       string    `gorm:"column:side;type:varchar(10);null"`
	IsLastBall bool      `gorm:"column:is_last_ball;type:boolean;not null;default:false"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp;not null;default:CURRENT_TIMESTAMP"`
}

// 設置表名
func (DrawnBallModel) TableName() string {
	return "drawn_balls"
}

// 幸運號碼資料庫模型
type LuckyBallsModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	GameID    string    `gorm:"column:game_id;type:varchar(100);null;index"`
	DrawDate  time.Time `gorm:"column:draw_date;type:timestamp;not null"`
	Number1   int       `gorm:"column:number1;type:int;not null"`
	Number2   int       `gorm:"column:number2;type:int;not null"`
	Number3   int       `gorm:"column:number3;type:int;not null"`
	Number4   int       `gorm:"column:number4;type:int;not null"`
	Number5   int       `gorm:"column:number5;type:int;not null"`
	Number6   int       `gorm:"column:number6;type:int;not null"`
	Number7   int       `gorm:"column:number7;type:int;not null"`
	Active    bool      `gorm:"column:active;type:boolean;not null;default:true;index"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;not null;default:CURRENT_TIMESTAMP"`
}

// 設置表名
func (LuckyBallsModel) TableName() string {
	return "lucky_balls"
}

// 遊戲階段記錄模型
type GameStateLogModel struct {
	ID              uint       `gorm:"primaryKey;autoIncrement"`
	GameID          string     `gorm:"column:game_id;type:varchar(100);not null;index"`
	State           string     `gorm:"column:state;type:varchar(30);not null"`
	StartTime       time.Time  `gorm:"column:start_time;type:timestamp;not null"`
	EndTime         *time.Time `gorm:"column:end_time;type:timestamp;null"`
	DurationSeconds int        `gorm:"column:duration_seconds;type:int;null"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamp;not null;default:CURRENT_TIMESTAMP"`
}

// 設置表名
func (GameStateLogModel) TableName() string {
	return "game_state_logs"
}

// TiDBRepository 使用 TiDB/MySQL 實現的遊戲數據持久化儲存庫
type TiDBRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewTiDBRepository 創建新的 TiDB 遊戲數據存儲庫
func NewTiDBRepository(dbManager databaseManager.DatabaseManager, logger *zap.Logger) *TiDBRepository {
	// 設置自定義 logger
	repoLogger := logger.With(zap.String("component", "tidb_repository"))

	// 獲取 GORM DB 實例
	db := dbManager.GetDB()

	// 自動遷移表結構
	err := db.AutoMigrate(
		&GameModel{},
		&DrawnBallModel{},
		&LuckyBallsModel{},
		&GameStateLogModel{},
		&JackpotGameModel{},
	)

	if err != nil {
		repoLogger.Error("自動遷移表結構失敗", zap.Error(err))
	} else {
		repoLogger.Info("自動遷移表結構完成")
	}

	return &TiDBRepository{
		db:     db,
		logger: repoLogger,
	}
}

// SaveGameHistory 保存遊戲歷史記錄到 TiDB
func (r *TiDBRepository) SaveGameHistory(ctx context.Context, game *GameData) error {
	// 開始事務
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("開始事務失敗: %w", tx.Error)
	}

	// 在函數結束時檢查是否需要回滾
	defer func() {
		if recovered := recover(); recovered != nil {
			tx.Rollback()
			r.logger.Error("保存遊戲歷史記錄時發生嚴重錯誤", zap.Any("panic", recovered))
		}
	}()

	// 檢查是否已存在此遊戲記錄
	var existingGame GameModel
	result := tx.Where("game_id = ?", game.GameID).First(&existingGame)

	// 1. 儲存遊戲基本資訊
	gameModel := convertGameDataToGameModel(game)

	// 序列化遊戲快照
	gameSnapshot, err := json.Marshal(game)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("序列化遊戲快照失敗: %w", err)
	}
	gameModel.GameSnapshot = string(gameSnapshot)

	// 根據是否已存在記錄決定創建或更新
	if result.Error == nil { // 記錄已存在，執行更新
		r.logger.Info("更新現有遊戲記錄",
			zap.String("gameID", game.GameID),
			zap.String("stage", string(game.CurrentStage)))

		if err := tx.Model(&existingGame).Updates(gameModel).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("更新遊戲基本資訊失敗: %w", err)
		}
	} else { // 記錄不存在，執行創建
		r.logger.Info("創建新遊戲記錄",
			zap.String("gameID", game.GameID),
			zap.String("stage", string(game.CurrentStage)))

		if err := tx.Create(&gameModel).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("保存遊戲基本資訊失敗: %w", err)
		}
	}

	// 2. 先刪除現有球記錄，然後重新儲存已抽出的球
	if err := tx.Where("game_id = ?", game.GameID).Delete(&DrawnBallModel{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("刪除舊球記錄失敗: %w", err)
	}

	// 重新保存所有球
	if err := saveDrawnBalls(tx, game); err != nil {
		tx.Rollback()
		return fmt.Errorf("保存已抽出的球失敗: %w", err)
	}

	// 3. 儲存最新的幸運號碼球 (如果有)
	if game.Jackpot != nil && len(game.Jackpot.LuckyBalls) == 7 {
		r.logger.Info("保存幸運號碼球到lucky_balls表",
			zap.String("gameID", game.GameID),
			zap.Int("luckyBallsCount", len(game.Jackpot.LuckyBalls)))

		if err := saveLuckyBalls(tx, game); err != nil {
			tx.Rollback()
			return fmt.Errorf("保存幸運號碼球失敗: %w", err)
		}
	}

	// 4. 儲存遊戲階段記錄
	if err := saveGameStateLog(tx, game); err != nil {
		tx.Rollback()
		return fmt.Errorf("保存遊戲階段記錄失敗: %w", err)
	}

	// 5. 儲存Jackpot遊戲數據（如果有）
	if game.HasJackpot && game.Jackpot != nil {
		r.logger.Info("準備保存Jackpot遊戲數據",
			zap.String("gameID", game.GameID),
			zap.String("jackpotID", game.Jackpot.ID),
			zap.Bool("endTimeSet", !game.Jackpot.EndTime.IsZero()))

		// 僅保存與資料庫表對應的欄位，JackpotGame中的其他欄位（如Amount、Active、WinnerUserID等）只在內存中使用
		jpModel := JackpotGameModel{
			GameID:    game.GameID,
			JackpotID: game.Jackpot.ID,
		}

		// 設置開始時間
		if !game.Jackpot.StartTime.IsZero() {
			startTime := game.Jackpot.StartTime
			jpModel.StartTime = &startTime
		}

		// 設置結束時間
		if !game.Jackpot.EndTime.IsZero() {
			endTime := game.Jackpot.EndTime
			jpModel.EndTime = &endTime
		}

		if err := tx.Create(&jpModel).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("保存jackpot遊戲記錄失敗: %w", err)
		}
	}

	// 提交事務
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事務失敗: %w", err)
	}

	r.logger.Info("已成功保存遊戲歷史記錄到數據庫",
		zap.String("gameID", game.GameID),
		zap.String("stage", string(game.CurrentStage)),
		zap.Bool("cancelled", game.IsCancelled),
		zap.Bool("hasJackpot", game.HasJackpot),
		zap.Int("luckyBallsCount", func() int {
			if game.Jackpot != nil {
				return len(game.Jackpot.LuckyBalls)
			}
			return 0
		}()))

	return nil
}

// GetRecentGameHistories 從 TiDB 獲取最近的遊戲歷史記錄
func (r *TiDBRepository) GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error) {
	var gameModels []GameModel

	// 查詢最近的遊戲記錄，按開始時間降序排序
	if err := r.db.WithContext(ctx).Order("start_time DESC").Limit(limit).Find(&gameModels).Error; err != nil {
		return nil, fmt.Errorf("查詢最近遊戲記錄失敗: %w", err)
	}

	// 轉換為遊戲數據對象
	var games []*GameData
	for _, model := range gameModels {
		// 如果有遊戲快照，直接反序列化
		if model.GameSnapshot != "" {
			var game GameData
			if err := json.Unmarshal([]byte(model.GameSnapshot), &game); err != nil {
				r.logger.Warn("反序列化遊戲快照失敗",
					zap.String("gameID", model.GameID),
					zap.Error(err))
				continue
			}
			games = append(games, &game)
		} else {
			// 如果沒有快照，需要從數據庫重建遊戲數據
			game, err := r.rebuildGameDataFromDB(ctx, model.GameID)
			if err != nil {
				r.logger.Warn("重建遊戲數據失敗",
					zap.String("gameID", model.GameID),
					zap.Error(err))
				continue
			}
			games = append(games, game)
		}
	}

	r.logger.Debug("已獲取最近遊戲歷史記錄",
		zap.Int("請求數量", limit),
		zap.Int("實際獲取", len(games)))

	return games, nil
}

// GetGameByID 從 TiDB 獲取特定ID的遊戲
func (r *TiDBRepository) GetGameByID(ctx context.Context, gameID string) (*GameData, error) {
	var model GameModel

	// 查詢特定ID的遊戲
	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("查詢遊戲記錄失敗: %w", err)
	}

	// 如果有遊戲快照，直接反序列化
	if model.GameSnapshot != "" {
		var game GameData
		if err := json.Unmarshal([]byte(model.GameSnapshot), &game); err != nil {
			return nil, fmt.Errorf("反序列化遊戲快照失敗: %w", err)
		}
		return &game, nil
	}

	// 如果沒有快照，從數據庫重建遊戲數據
	return r.rebuildGameDataFromDB(ctx, gameID)
}

// GetGamesByDateRange 從 TiDB 獲取特定日期範圍的遊戲
func (r *TiDBRepository) GetGamesByDateRange(ctx context.Context, startDate, endDate string) ([]*GameData, error) {
	var gameModels []GameModel

	// 查詢日期範圍內的遊戲記錄
	if err := r.db.WithContext(ctx).
		Where("start_time >= ? AND start_time <= ?", startDate, endDate).
		Order("start_time DESC").
		Find(&gameModels).Error; err != nil {
		return nil, fmt.Errorf("查詢日期範圍遊戲記錄失敗: %w", err)
	}

	// 轉換為遊戲數據對象
	var games []*GameData
	for _, model := range gameModels {
		// 如果有遊戲快照，直接反序列化
		if model.GameSnapshot != "" {
			var game GameData
			if err := json.Unmarshal([]byte(model.GameSnapshot), &game); err != nil {
				r.logger.Warn("反序列化遊戲快照失敗",
					zap.String("gameID", model.GameID),
					zap.Error(err))
				continue
			}
			games = append(games, &game)
		} else {
			// 如果沒有快照，需要從數據庫重建遊戲數據
			game, err := r.rebuildGameDataFromDB(ctx, model.GameID)
			if err != nil {
				r.logger.Warn("重建遊戲數據失敗",
					zap.String("gameID", model.GameID),
					zap.Error(err))
				continue
			}
			games = append(games, game)
		}
	}

	return games, nil
}

// GetTotalGamesCount 從 TiDB 獲取總遊戲數量
func (r *TiDBRepository) GetTotalGamesCount(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&GameModel{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("統計遊戲總數失敗: %w", err)
	}
	return count, nil
}

// GetCancelledGamesCount 從 TiDB 獲取取消的遊戲數量
func (r *TiDBRepository) GetCancelledGamesCount(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&GameModel{}).Where("cancelled = ?", true).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("統計取消遊戲數失敗: %w", err)
	}
	return count, nil
}

// GetRecentGameHistoriesByRoom 從 TiDB 獲取特定房間最近的遊戲歷史記錄
func (r *TiDBRepository) GetRecentGameHistoriesByRoom(ctx context.Context, roomID string, limit int) ([]*GameData, error) {
	var gameModels []GameModel

	// 查詢最近的遊戲記錄，按開始時間降序排序
	// 從遊戲ID中提取房間ID，格式假設為 "room_{roomID}_game_{uuid}"
	gameIDLike := fmt.Sprintf("room_%s_game_%%", roomID)

	if err := r.db.WithContext(ctx).
		Where("game_id LIKE ?", gameIDLike).
		Order("start_time DESC").
		Limit(limit).
		Find(&gameModels).Error; err != nil {
		return nil, fmt.Errorf("查詢房間最近遊戲記錄失敗: %w", err)
	}

	// 轉換為遊戲數據對象
	var games []*GameData
	for _, model := range gameModels {
		// 如果有遊戲快照，直接反序列化
		if model.GameSnapshot != "" {
			var game GameData
			if err := json.Unmarshal([]byte(model.GameSnapshot), &game); err != nil {
				r.logger.Warn("反序列化遊戲快照失敗",
					zap.String("gameID", model.GameID),
					zap.Error(err))
				continue
			}
			games = append(games, &game)
		} else {
			// 如果沒有快照，需要從數據庫重建遊戲數據
			game, err := r.rebuildGameDataFromDB(ctx, model.GameID)
			if err != nil {
				r.logger.Warn("重建遊戲數據失敗",
					zap.String("gameID", model.GameID),
					zap.Error(err))
				continue
			}
			games = append(games, game)
		}
	}

	r.logger.Debug("已獲取房間最近遊戲歷史記錄",
		zap.String("roomID", roomID),
		zap.Int("請求數量", limit),
		zap.Int("實際獲取", len(games)))

	return games, nil
}

// GetTotalGamesCountByRoom 從 TiDB 獲取特定房間的總遊戲數量
func (r *TiDBRepository) GetTotalGamesCountByRoom(ctx context.Context, roomID string) (int64, error) {
	var count int64
	gameIDLike := fmt.Sprintf("room_%s_game_%%", roomID)

	if err := r.db.WithContext(ctx).
		Model(&GameModel{}).
		Where("game_id LIKE ?", gameIDLike).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("統計房間遊戲總數失敗: %w", err)
	}

	return count, nil
}

// 輔助函數：從數據庫重建遊戲數據
func (r *TiDBRepository) rebuildGameDataFromDB(ctx context.Context, gameID string) (*GameData, error) {
	// 獲取遊戲基本信息
	var gameModel GameModel
	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).First(&gameModel).Error; err != nil {
		return nil, fmt.Errorf("獲取遊戲基本信息失敗: %w", err)
	}

	// 獲取已抽出的球
	var drawnBalls []DrawnBallModel
	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Order("sequence ASC").Find(&drawnBalls).Error; err != nil {
		return nil, fmt.Errorf("獲取已抽出的球失敗: %w", err)
	}

	// 構建遊戲數據
	game := &GameData{
		GameID:         gameModel.GameID,
		RoomID:         gameModel.RoomID,
		CurrentStage:   GameStage(gameModel.State),
		StartTime:      gameModel.StartTime,
		HasJackpot:     gameModel.HasJackpot,
		IsCancelled:    gameModel.Cancelled,
		CancelReason:   gameModel.CancelReason,
		LastUpdateTime: gameModel.UpdatedAt,
		RegularBalls:   make([]Ball, 0),
		ExtraBalls:     make([]Ball, 0),
	}

	// 設置結束時間
	if gameModel.EndTime != nil {
		game.EndTime = *gameModel.EndTime
	}

	// 設置取消時間
	if gameModel.CancelTime != nil {
		game.CancelTime = *gameModel.CancelTime
	}

	// 設置階段過期時間
	if gameModel.StageExpireTime != nil {
		game.StageExpireTime = *gameModel.StageExpireTime
	}

	// 創建 Jackpot 結構體 (如果需要)
	var luckyBalls []Ball

	// 處理已抽出的球
	for _, ball := range drawnBalls {
		gameBall := Ball{
			Number:    ball.Number,
			Timestamp: ball.DrawnTime,
			IsLast:    ball.IsLastBall,
		}

		// 根據球類型添加到相應的數組
		switch ball.BallType {
		case "REGULAR":
			gameBall.Type = BallTypeRegular
			game.RegularBalls = append(game.RegularBalls, gameBall)
		case "EXTRA":
			gameBall.Type = BallTypeExtra
			game.ExtraBalls = append(game.ExtraBalls, gameBall)
		case "JACKPOT":
			gameBall.Type = BallTypeJackpot
			// 如果遊戲有JP，但 Jackpot 為 nil，返回錯誤
			if game.HasJackpot && game.Jackpot == nil {
				return nil, fmt.Errorf("遊戲有 Jackpot 標記，但 Jackpot 結構未初始化")
			}
			// 將JP球加入到Jackpot的DrawnBalls中
			if game.Jackpot != nil {
				game.Jackpot.DrawnBalls = append(game.Jackpot.DrawnBalls, gameBall)
			}
		case "LUCKY":
			gameBall.Type = BallTypeLucky
			// 收集幸運號碼球，稍後加入到 Jackpot
			luckyBalls = append(luckyBalls, gameBall)
		}

		// 設置額外球選邊
		if ball.BallType == "EXTRA" && ball.Side != "" {
			game.SelectedSide = ExtraBallSide(ball.Side)
		}
	}

	// 如果有幸運號碼球，確保 Jackpot 存在
	if len(luckyBalls) > 0 {
		if game.Jackpot == nil {
			return nil, fmt.Errorf("發現幸運號碼球，但 Jackpot 結構未初始化")
		}
		game.Jackpot.LuckyBalls = luckyBalls
	}

	return game, nil
}

// 輔助函數：將 GameData 轉換為 GameModel
func convertGameDataToGameModel(game *GameData) GameModel {
	model := GameModel{
		GameID:                game.GameID,
		RoomID:                game.RoomID,
		State:                 string(game.CurrentStage),
		StartTime:             game.StartTime,
		HasJackpot:            game.HasJackpot,
		ExtraBallCount:        len(game.ExtraBalls),
		CurrentStateStartTime: game.LastUpdateTime,
		MaxTimeout:            60, // 默認值，可根據實際情況調整
		Cancelled:             game.IsCancelled,
		CancelReason:          game.CancelReason,
		CreatedAt:             game.StartTime,
		UpdatedAt:             game.LastUpdateTime,
	}

	// 設置非必需欄位
	if !game.EndTime.IsZero() {
		model.EndTime = &game.EndTime
	}

	if !game.CancelTime.IsZero() {
		model.CancelTime = &game.CancelTime
	}

	// 設置階段過期時間
	if !game.StageExpireTime.IsZero() {
		model.StageExpireTime = &game.StageExpireTime
	}

	// 統計數據預設為0，實際應用中可根據需要從其他地方獲取
	model.TotalCards = 0
	model.TotalPlayers = 0
	model.TotalBetAmount = 0
	model.TotalWinAmount = 0

	return model
}

// 輔助函數：保存已抽出的球
func saveDrawnBalls(tx *gorm.DB, game *GameData) error {
	// 保存常規球
	for i, ball := range game.RegularBalls {
		model := DrawnBallModel{
			GameID:     game.GameID,
			Number:     ball.Number,
			Sequence:   i + 1,
			BallType:   "REGULAR",
			DrawnTime:  ball.Timestamp,
			IsLastBall: ball.IsLast,
			CreatedAt:  time.Now(),
		}
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
	}

	// 保存額外球
	for i, ball := range game.ExtraBalls {
		model := DrawnBallModel{
			GameID:     game.GameID,
			Number:     ball.Number,
			Sequence:   i + 1,
			BallType:   "EXTRA",
			DrawnTime:  ball.Timestamp,
			Side:       string(game.SelectedSide),
			IsLastBall: ball.IsLast,
			CreatedAt:  time.Now(),
		}
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
	}

	// 保存JP球
	if game.Jackpot != nil {
		// 保存已抽出的 JP 球
		for i, ball := range game.Jackpot.DrawnBalls {
			model := DrawnBallModel{
				GameID:     game.GameID,
				Number:     ball.Number,
				Sequence:   i + 1,
				BallType:   "JACKPOT",
				DrawnTime:  ball.Timestamp,
				IsLastBall: ball.IsLast,
				CreatedAt:  time.Now(),
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		}

		// 保存幸運號碼球
		for i, ball := range game.Jackpot.LuckyBalls {
			model := DrawnBallModel{
				GameID:     game.GameID,
				Number:     ball.Number,
				Sequence:   i + 1,
				BallType:   "LUCKY",
				DrawnTime:  ball.Timestamp,
				IsLastBall: ball.IsLast,
				CreatedAt:  time.Now(),
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// 輔助函數：保存幸運號碼球
func saveLuckyBalls(tx *gorm.DB, game *GameData) error {
	logger := zap.L().With(zap.String("component", "tidb_repository.saveLuckyBalls"))

	// 檢查 Jackpot 是否存在
	if game.Jackpot == nil || len(game.Jackpot.LuckyBalls) < 7 {
		errMsg := fmt.Sprintf("幸運號碼球數量不足，需要7顆，目前有 %d 顆",
			func() int {
				if game.Jackpot != nil {
					return len(game.Jackpot.LuckyBalls)
				}
				return 0
			}())
		logger.Error(errMsg, zap.String("gameID", game.GameID))
		return fmt.Errorf(errMsg)
	}

	// 打印所有幸運號碼球，方便排查
	var luckyNumbers []int
	for _, ball := range game.Jackpot.LuckyBalls {
		luckyNumbers = append(luckyNumbers, ball.Number)
	}
	logger.Info("準備保存幸運號碼球到TiDB",
		zap.String("gameID", game.GameID),
		zap.Ints("luckyNumbers", luckyNumbers),
		zap.Bool("isLastBall", game.Jackpot.LuckyBalls[len(game.Jackpot.LuckyBalls)-1].IsLast))

	// 1. 判斷是否已存在此遊戲的幸運號碼球記錄
	var existingRecord LuckyBallsModel
	result := tx.Where("game_id = ?", game.GameID).First(&existingRecord)

	if result.Error == nil {
		// 已有記錄，先刪除
		logger.Info("此遊戲已有幸運號碼記錄，準備更新",
			zap.String("gameID", game.GameID),
			zap.Uint("recordID", existingRecord.ID))

		if err := tx.Where("game_id = ?", game.GameID).Delete(&LuckyBallsModel{}).Error; err != nil {
			logger.Error("刪除舊的幸運號碼記錄失敗",
				zap.String("gameID", game.GameID),
				zap.Error(err))
			return fmt.Errorf("刪除舊的幸運號碼記錄失敗: %w", err)
		}
	} else {
		logger.Info("此遊戲沒有現有幸運號碼記錄，將創建新記錄",
			zap.String("gameID", game.GameID))
	}

	// 2. 將現有的活躍幸運號碼設為非活躍
	if err := tx.Model(&LuckyBallsModel{}).Where("active = ?", true).Update("active", false).Error; err != nil {
		logger.Error("更新舊的活躍幸運號碼狀態失敗",
			zap.Error(err))
		return fmt.Errorf("更新舊的活躍幸運號碼狀態失敗: %w", err)
	}
	logger.Info("已將現有活躍幸運號碼設為非活躍")

	// 3. 保存新的幸運號碼
	model := LuckyBallsModel{
		GameID:    game.GameID,
		DrawDate:  time.Now(),
		Number1:   game.Jackpot.LuckyBalls[0].Number,
		Number2:   game.Jackpot.LuckyBalls[1].Number,
		Number3:   game.Jackpot.LuckyBalls[2].Number,
		Number4:   game.Jackpot.LuckyBalls[3].Number,
		Number5:   game.Jackpot.LuckyBalls[4].Number,
		Number6:   game.Jackpot.LuckyBalls[5].Number,
		Number7:   game.Jackpot.LuckyBalls[6].Number,
		Active:    true,
		CreatedAt: time.Now(),
	}

	// 創建新記錄
	if err := tx.Create(&model).Error; err != nil {
		logger.Error("保存新的幸運號碼記錄失敗",
			zap.String("gameID", game.GameID),
			zap.Error(err))
		return fmt.Errorf("保存新的幸運號碼記錄失敗: %w", err)
	}
	logger.Info("成功保存新的幸運號碼記錄到TiDB",
		zap.String("gameID", game.GameID),
		zap.Ints("luckyNumbers", luckyNumbers))

	return nil
}

// 輔助函數：保存遊戲階段記錄
func saveGameStateLog(tx *gorm.DB, game *GameData) error {
	// 計算階段持續時間
	var durationSeconds int = 0
	endTime := game.EndTime
	if endTime.IsZero() {
		endTime = time.Now()
	}
	durationSeconds = int(endTime.Sub(game.StartTime).Seconds())

	// 保存階段記錄
	model := GameStateLogModel{
		GameID:          game.GameID,
		State:           string(game.CurrentStage),
		StartTime:       game.StartTime,
		EndTime:         &endTime,
		DurationSeconds: durationSeconds,
		CreatedAt:       time.Now(),
	}

	return tx.Create(&model).Error
}
