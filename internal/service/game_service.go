package service

import (
	"context"
	"log"

	"g38_lottery_service/game"

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
}

// gameServiceImpl 實現 GameService 接口
type gameServiceImpl struct {
	controller *game.DataFlowController
}

// NewGameService 創建一個新的遊戲服務
func NewGameService(lc fx.Lifecycle, controller *game.DataFlowController) GameService {
	service := &gameServiceImpl{
		controller: controller,
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
