package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"g38_lottery_service/game"
	"g38_lottery_service/internal/model"
	"g38_lottery_service/pkg/databaseManager"

	"github.com/google/uuid"
	"go.uber.org/fx"
)

// GameService 定義遊戲服務介面
type GameService interface {
	// 獲取遊戲當前狀態
	GetGameStatus() *game.GameStatusResponse
	// 獲取當前遊戲狀態
	GetCurrentState() game.GameState
	// 更改遊戲狀態
	ChangeState(state game.GameState) error
	// 設置JP觸發號碼
	SetJPTriggerNumbers(numbers []int) error
	// 驗證兩顆球的有效性
	VerifyTwoBalls(ball1, ball2 int) bool
	// 抽取一顆球
	DrawBall() (*game.DrawResult, error)
	// 抽取額外球
	DrawExtraBall() (*game.DrawResult, error)
	// 獲取已抽出的球
	GetDrawnBalls() []game.DrawResult
	// 獲取額外球
	GetExtraBalls() []game.DrawResult
	// 創建新遊戲（處理 GAME_START 命令）
	CreateGame() (*model.Game, error)
}

// gameServiceImpl 實現 GameService 接口
type gameServiceImpl struct {
	controller *game.DataFlowController
	db         databaseManager.DatabaseManager
}

// NewGameService 創建一個新的遊戲服務
func NewGameService(lc fx.Lifecycle, controller *game.DataFlowController, db databaseManager.DatabaseManager) GameService {
	service := &gameServiceImpl{
		controller: controller,
		db:         db,
	}

	// 設置生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("遊戲服務已初始化，當前狀態:", string(controller.GetCurrentState()))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("關閉遊戲服務...")
			return nil
		},
	})

	return service
}

// GetGameStatus 獲取遊戲當前狀態
func (s *gameServiceImpl) GetGameStatus() *game.GameStatusResponse {
	return s.controller.GetGameStatus()
}

// GetCurrentState 獲取當前遊戲狀態
func (s *gameServiceImpl) GetCurrentState() game.GameState {
	return s.controller.GetCurrentState()
}

// ChangeState 更改遊戲狀態
func (s *gameServiceImpl) ChangeState(state game.GameState) error {
	return s.controller.ChangeState(state)
}

// SetJPTriggerNumbers 設置JP觸發號碼
func (s *gameServiceImpl) SetJPTriggerNumbers(numbers []int) error {
	s.controller.SetJPTriggerNumbers(numbers)
	return nil
}

// VerifyTwoBalls 驗證兩顆球的有效性
func (s *gameServiceImpl) VerifyTwoBalls(ball1, ball2 int) bool {
	return s.controller.VerifyTwoBalls(ball1, ball2)
}

// DrawBall 抽取一顆球
func (s *gameServiceImpl) DrawBall() (*game.DrawResult, error) {
	return s.controller.DrawBall()
}

// DrawExtraBall 抽取額外球
func (s *gameServiceImpl) DrawExtraBall() (*game.DrawResult, error) {
	return s.controller.DrawExtraBall()
}

// GetDrawnBalls 獲取已抽出的球
func (s *gameServiceImpl) GetDrawnBalls() []game.DrawResult {
	return s.controller.GetDrawnBalls()
}

// GetExtraBalls 獲取額外球
func (s *gameServiceImpl) GetExtraBalls() []game.DrawResult {
	return s.controller.GetExtraBalls()
}

// CreateGame 創建新遊戲（處理 GAME_START 命令）
func (s *gameServiceImpl) CreateGame() (*model.Game, error) {
	gameID := uuid.NewString()
	game := &model.Game{
		ID:         gameID,
		State:      model.GameStateReady,
		StartTime:  time.Now(),
		HasJackpot: false,
	}

	// 寫入資料庫
	db := s.db.GetDB()
	txdb := db.Begin()
	if txdb.Error != nil {
		return nil, fmt.Errorf("開始事務失敗: %w", txdb.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			txdb.Rollback()
		}
	}()

	// 插入記錄到資料庫
	sql := `INSERT INTO games (id, state, start_time, has_jackpot) VALUES (?, ?, ?, ?)`
	err := txdb.Exec(sql, game.ID, game.State, game.StartTime, game.HasJackpot).Error
	if err != nil {
		txdb.Rollback()
		return nil, fmt.Errorf("創建遊戲記錄失敗: %w", err)
	}

	if err := txdb.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事務失敗: %w", err)
	}

	// 更新控制器的遊戲ID
	s.controller.SetCurrentGameID(game.ID)

	// 嘗試將狀態更改為準備狀態
	_ = s.ChangeState("READY")

	log.Printf("成功創建新遊戲 ID: %s, 開始時間: %v", game.ID, game.StartTime)

	return game, nil
}
