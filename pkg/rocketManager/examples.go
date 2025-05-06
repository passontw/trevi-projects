package rocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// 以下範例展示如何在遊戲流程管理中使用RocketManager

// 玩家通信服務範例
type PlayerCommunicationService struct {
	rocketManager RocketManager
	logger        Logger
}

// Logger 介面
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// 建立新的玩家通信服務
func NewPlayerCommunicationService(rocketManager RocketManager, logger Logger) *PlayerCommunicationService {
	return &PlayerCommunicationService{
		rocketManager: rocketManager,
		logger:        logger,
	}
}

// 初始化服務，設置訂閱
func (s *PlayerCommunicationService) Initialize(ctx context.Context) error {
	// 訂閱全局遊戲狀態更新
	if err := s.rocketManager.Subscribe(TopicGameStatus, "", s.handleGameStatusMessage); err != nil {
		return fmt.Errorf("failed to subscribe to game status topic: %w", err)
	}

	// 訂閱抽球結果
	if err := s.rocketManager.Subscribe(TopicGameDraw, TagBallDrawn, s.handleBallDrawnMessage); err != nil {
		return fmt.Errorf("failed to subscribe to ball drawn topic: %w", err)
	}

	// 啟動RocketMQ消費者
	go func() {
		if err := s.rocketManager.Start(ctx); err != nil {
			s.logger.Error("Failed to start RocketMQ consumer", "error", err)
		}
	}()

	return nil
}

// 處理遊戲狀態消息
func (s *PlayerCommunicationService) handleGameStatusMessage(ctx context.Context, msg *primitive.MessageExt) error {
	var message Message
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		return fmt.Errorf("failed to unmarshal game status message: %w", err)
	}

	s.logger.Info("Received game status message",
		"gameID", message.GameID,
		"stage", message.Stage,
		"event", message.Event)

	// 根據不同的事件類型處理消息
	switch message.Event {
	case TagStageChange:
		return s.handleStageChange(message)
	case TagGameCreated:
		return s.handleGameCreated(message)
	case TagGameCancelled, TagGameFinished:
		return s.handleGameEnded(message)
	default:
		s.logger.Debug("Unhandled game status event", "event", message.Event)
		return nil
	}
}

// 處理階段變更
func (s *PlayerCommunicationService) handleStageChange(message Message) error {
	s.logger.Info("Game stage changed",
		"gameID", message.GameID,
		"newStage", message.Stage)

	// 這裡可以添加針對不同階段的處理邏輯
	return nil
}

// 處理遊戲創建
func (s *PlayerCommunicationService) handleGameCreated(message Message) error {
	s.logger.Info("New game created", "gameID", message.GameID)

	// 可以在這裡處理訂閱特定遊戲ID的邏輯
	return s.rocketManager.SubscribeGame(message.GameID, s.handleGameSpecificMessage)
}

// 處理遊戲結束
func (s *PlayerCommunicationService) handleGameEnded(message Message) error {
	s.logger.Info("Game ended",
		"gameID", message.GameID,
		"event", message.Event)

	// 處理遊戲結束邏輯
	return nil
}

// 處理抽球消息
func (s *PlayerCommunicationService) handleBallDrawnMessage(ctx context.Context, msg *primitive.MessageExt) error {
	var message Message
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		return fmt.Errorf("failed to unmarshal ball drawn message: %w", err)
	}

	s.logger.Info("Received ball drawn message",
		"gameID", message.GameID,
		"event", message.Event)

	// 從消息中提取球信息
	var ballInfo struct {
		Number int  `json:"number"`
		IsLast bool `json:"isLast"`
	}

	data, err := json.Marshal(message.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal ball info data: %w", err)
	}

	if err := json.Unmarshal(data, &ballInfo); err != nil {
		return fmt.Errorf("failed to unmarshal ball info: %w", err)
	}

	s.logger.Info("Ball drawn",
		"gameID", message.GameID,
		"ballNumber", ballInfo.Number,
		"isLast", ballInfo.IsLast)

	return nil
}

// 處理特定遊戲消息
func (s *PlayerCommunicationService) handleGameSpecificMessage(ctx context.Context, msg *primitive.MessageExt) error {
	var message Message
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		return fmt.Errorf("failed to unmarshal game specific message: %w", err)
	}

	s.logger.Debug("Received game specific message",
		"gameID", message.GameID,
		"event", message.Event,
		"tag", msg.GetTags())

	return nil
}

// ==============================================
// 遊戲流程管理示例：如何發送消息到遊戲端
// ==============================================

// 示例：從遊戲流程管理中發送階段變更通知
func ExampleSendStageChangeFromGameManager(rocketManager RocketManager, gameID, stage string) error {
	ctx := context.Background()

	// 階段變更數據
	stageData := map[string]interface{}{
		"previous_stage": "StageDrawingStart",
		"current_stage":  stage,
		"timestamp":      time.Now().Unix(),
	}

	// 發送階段變更通知
	return rocketManager.SendGameStatus(ctx, gameID, stage, TagStageChange, stageData)
}

// 示例：廣播抽球結果
func ExampleBroadcastBallDrawn(rocketManager RocketManager, gameID string, ballNumber int, isLast bool) error {
	ctx := context.Background()

	// 抽球數據
	ballInfo := map[string]interface{}{
		"number": ballNumber,
		"isLast": isLast,
		"time":   time.Now().Format(time.RFC3339),
	}

	// 廣播抽球結果
	return rocketManager.BroadcastBallDrawn(ctx, gameID, ballInfo)
}

// 示例：通知遊戲創建
func ExampleNotifyGameCreated(rocketManager RocketManager, gameID string, gameInfo map[string]interface{}) error {
	ctx := context.Background()

	// 發送遊戲創建通知
	return rocketManager.SendGameStatus(ctx, gameID, "StageNewRound", TagGameCreated, gameInfo)
}

// 示例：通知遊戲取消
func ExampleNotifyGameCancelled(rocketManager RocketManager, gameID, reason string) error {
	ctx := context.Background()

	// 取消數據
	cancelData := map[string]interface{}{
		"reason":    reason,
		"timestamp": time.Now().Unix(),
	}

	// 發送遊戲取消通知
	return rocketManager.SendGameStatus(ctx, gameID, "StageCancelled", TagGameCancelled, cancelData)
}

// 示例：在實際應用程序中使用
func ExampleUsage() {
	// 假設我們已經有了RocketManager實例
	var rocketManager RocketManager
	// 假設我們已經有了logger實例
	var logger Logger

	// 創建玩家通信服務
	playerComm := NewPlayerCommunicationService(rocketManager, logger)

	// 初始化服務
	ctx := context.Background()
	if err := playerComm.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize player communication service: %v", err)
	}

	// 模擬遊戲流程

	// 1. 創建新遊戲
	gameID := "game123"
	gameInfo := map[string]interface{}{
		"id":        gameID,
		"timestamp": time.Now().Unix(),
	}
	if err := ExampleNotifyGameCreated(rocketManager, gameID, gameInfo); err != nil {
		log.Printf("Failed to notify game creation: %v", err)
	}

	// 2. 發送階段變更
	if err := ExampleSendStageChangeFromGameManager(rocketManager, gameID, "StageDrawingStart"); err != nil {
		log.Printf("Failed to send stage change: %v", err)
	}

	// 3. 發送抽球結果
	for i := 1; i <= 5; i++ {
		isLast := i == 5
		if err := ExampleBroadcastBallDrawn(rocketManager, gameID, i, isLast); err != nil {
			log.Printf("Failed to broadcast ball drawn: %v", err)
		}
		time.Sleep(time.Second)
	}

	// 4. 結束遊戲
	if err := ExampleSendStageChangeFromGameManager(rocketManager, gameID, "StageGameOver"); err != nil {
		log.Printf("Failed to send game over stage: %v", err)
	}
}
