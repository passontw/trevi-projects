// 遊戲服務實現
package services

import (
	"context"
	"g38_lottery_service/game"
	"g38_lottery_service/internal/logger"
	"g38_lottery_service/pkg/websocketManager"

	"gorm.io/gorm"
)

// GameService 遊戲服務介面
type GameService interface {
	Initialize() error
	Shutdown(ctx context.Context) error
	GetGameController() *game.DataFlowController
}

// SimpleGameService 簡易遊戲服務實現
type SimpleGameService struct {
	wsService      *websocketManager.DualWebSocketService
	db             *gorm.DB
	logger         *logger.SimpleLogger
	gameController *game.DataFlowController
}

// NewGameService 創建一個新的遊戲服務
func NewGameService(wsService *websocketManager.DualWebSocketService, db *gorm.DB) GameService {
	return &SimpleGameService{
		wsService:      wsService,
		db:             db,
		logger:         logger.NewSimpleLogger("[遊戲服務] "),
		gameController: game.NewDataFlowController(),
	}
}

// Initialize 初始化遊戲服務
func (s *SimpleGameService) Initialize() error {
	s.logger.Info("初始化遊戲服務")

	// 啟動WebSocket服務
	s.wsService.Start()

	// 將遊戲狀態從初始化改為待機狀態
	currentState := s.gameController.GetCurrentState()
	s.logger.Info("當前遊戲狀態：%s", currentState)

	// 檢查當前狀態，如果是初始狀態，則更改為準備狀態
	if currentState == game.StateInitial {
		if err := s.gameController.ChangeState(game.StateReady); err != nil {
			s.logger.Error("更改遊戲狀態失敗: %v", err)
			return err
		}
		s.logger.Info("遊戲狀態已更改為: %s", game.StateReady)
	}

	return nil
}

// Shutdown 關閉遊戲服務
func (s *SimpleGameService) Shutdown(ctx context.Context) error {
	s.logger.Info("關閉遊戲服務")

	// 嘗試先將遊戲狀態改回初始狀態
	currentState := s.gameController.GetCurrentState()
	s.logger.Info("關閉時遊戲狀態: %s", currentState)

	// 不強制改變狀態，只記錄當前狀態
	s.logger.Info("遊戲狀態保持為: %s", currentState)

	// 停止WebSocket服務
	s.wsService.Stop()

	return nil
}

// GetGameController 獲取遊戲控制器
func (s *SimpleGameService) GetGameController() *game.DataFlowController {
	return s.gameController
}
