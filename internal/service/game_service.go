package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"g38_lottery_service/game"
	"g38_lottery_service/internal/model"
	"g38_lottery_service/pkg/databaseManager"

	"go.uber.org/fx"
)

// WebsocketManager 是一個抽象的 WebSocket 管理器接口
type WebsocketManager interface {
	BroadcastToAll(message interface{}) error
}

// GameService 定義遊戲服務介面
type GameService interface {
	// 獲取遊戲當前狀態
	GetGameStatus() *game.GameStatusResponse
	// 獲取當前遊戲狀態
	GetCurrentState() game.GameState
	// 更改遊戲狀態
	ChangeState(state game.GameState) error
	// 開始投注
	StartBetting() error
	// 關閉投注
	CloseBetting() error
	// 開始抽球
	StartDrawing() error
	// 開始額外球投注
	StartExtraBetting() error
	// 結束額外球投注
	FinishExtraBetting() error
	// 開始選擇額外球
	ChooseExtraBall() error
	// 進入結算階段
	StartResult() error
	// 進入JP待機階段
	StartJPStandby() error
	// 進入JP投注階段
	StartJPBetting() error
	// 進入JP抽球階段
	StartJPDrawing() error
	// 結束JP抽球並進入JP結果階段
	StopJPDrawing() error
	// 進入JP開獎階段
	StartJPShowBalls() error
	// 進入遊戲完成階段
	StartCompleted() error
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
	// 生成幸運號碼
	SetLuckyNumbers() []int
	// 獲取幸運號碼
	GetLuckyNumbers() []int
}

// gameServiceImpl 實現 GameService 接口
type gameServiceImpl struct {
	controller *game.DataFlowController
	db         databaseManager.DatabaseManager
	wsManager  WebsocketManager
}

// NewGameService 創建一個新的遊戲服務
func NewGameService(lc fx.Lifecycle, controller *game.DataFlowController, db databaseManager.DatabaseManager, wsManager WebsocketManager) GameService {
	service := &gameServiceImpl{
		controller: controller,
		db:         db,
		wsManager:  wsManager,
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

// StartBetting 開始投注
func (s *gameServiceImpl) StartBetting() error {
	return s.controller.StartBetting()
}

// CloseBetting 關閉投注
func (s *gameServiceImpl) CloseBetting() error {
	return s.controller.CloseBetting()
}

// StartDrawing 開始抽球
func (s *gameServiceImpl) StartDrawing() error {
	return s.controller.StartDrawing()
}

// StartExtraBetting 開始額外球投注
func (s *gameServiceImpl) StartExtraBetting() error {
	return s.controller.StartExtraBetting()
}

// FinishExtraBetting 結束額外球投注
func (s *gameServiceImpl) FinishExtraBetting() error {
	return s.controller.FinishExtraBetting()
}

// ChooseExtraBall 開始選擇額外球階段
func (s *gameServiceImpl) ChooseExtraBall() error {
	return s.controller.ChooseExtraBall()
}

// StartResult 進入結算階段
func (s *gameServiceImpl) StartResult() error {
	return s.controller.StartResult()
}

// StartJPStandby 進入JP待機階段
func (s *gameServiceImpl) StartJPStandby() error {
	return s.controller.StartJPStandby()
}

// StartJPBetting 進入JP投注階段
func (s *gameServiceImpl) StartJPBetting() error {
	return s.controller.StartJPBetting()
}

// StartJPDrawing 進入JP抽球階段
func (s *gameServiceImpl) StartJPDrawing() error {
	return s.controller.StartJPDrawing()
}

// StopJPDrawing 結束JP抽球並進入JP結果階段
func (s *gameServiceImpl) StopJPDrawing() error {
	return s.controller.StopJPDrawing()
}

// StartJPShowBalls 進入JP開獎階段
func (s *gameServiceImpl) StartJPShowBalls() error {
	return s.controller.StartJPShowBalls()
}

// StartCompleted 進入遊戲完成階段
func (s *gameServiceImpl) StartCompleted() error {
	return s.controller.StartCompleted()
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

// CreateGame 創建新遊戲
func (s *gameServiceImpl) CreateGame() (*model.Game, error) {
	// 重置控制器狀態
	s.controller.ResetGame()

	// 創建新遊戲記錄
	newGame := model.NewGame()

	// 在這裡設置遊戲ID
	s.controller.SetCurrentGameID(newGame.ID)

	// 保存遊戲記錄到數據庫
	// TODO: 需要實現遊戲存儲功能
	// 暫時跳過資料庫存儲
	log.Printf("創建新遊戲: %s", newGame.ID)

	// 生成幸運號碼
	luckyNumbers := s.SetLuckyNumbers()
	log.Printf("已為遊戲 %s 生成幸運號碼: %v", newGame.ID, luckyNumbers)

	// 確保遊戲記錄中包含幸運號碼
	luckyNumbersJSON, _ := json.Marshal(luckyNumbers)
	luckyNumbersJSONStr := string(luckyNumbersJSON)
	newGame.LuckyNumbersJSON = &luckyNumbersJSONStr

	// 更改遊戲狀態為顯示幸運號碼
	if err := s.ChangeState(game.StateShowLuckyNums); err != nil {
		log.Printf("更改遊戲狀態失敗: %v", err)
		return newGame, err
	}

	log.Printf("遊戲狀態已更改為: %s", game.StateShowLuckyNums)

	return newGame, nil
}

// SetLuckyNumbers 生成幸運號碼並通知荷官端
func (s *gameServiceImpl) SetLuckyNumbers() []int {
	// 生成幸運號碼
	luckyNumbers := s.controller.SetLuckyNumbers()

	// 構建幸運號碼設置通知
	status := s.controller.GetGameStatus()

	// 創建通知結構 - 嚴格遵照 EventLuckyNumbersSet 格式
	notification := map[string]interface{}{
		"type": "LUCKY_NUMBERS_SET",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":             status.Game.ID,
				"state":          status.Game.State,
				"startTime":      status.Game.StartTime,
				"endTime":        status.Game.EndTime,
				"hasJackpot":     status.Game.HasJackpot,
				"extraBallCount": status.Game.ExtraBallCount,
			},
			"luckyNumbers": luckyNumbers,
			"drawnBalls":   []interface{}{},
			"extraBalls":   []interface{}{},
			"jackpot": map[string]interface{}{
				"active":     false,
				"gameId":     nil,
				"amount":     0,
				"startTime":  nil,
				"endTime":    nil,
				"drawnBalls": []interface{}{},
				"winner":     nil,
			},
			"topPlayers":     []interface{}{},
			"totalWinAmount": 0,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 詳細記錄通知內容以便調試
	notificationJSON, _ := json.Marshal(notification)
	log.Printf("發送幸運號碼通知: %s", string(notificationJSON))

	// 發送通知給所有荷官端
	if err := s.wsManager.BroadcastToAll(notification); err != nil {
		log.Printf("發送幸運號碼通知失敗: %v", err)
	} else {
		log.Printf("成功向所有荷官端發送幸運號碼通知")
	}

	return luckyNumbers
}

// GetLuckyNumbers 獲取幸運號碼
func (s *gameServiceImpl) GetLuckyNumbers() []int {
	return s.controller.GetLuckyNumbers()
}
