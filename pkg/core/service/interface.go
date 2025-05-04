package service

import (
	"context"
	"time"
)

// WebsocketManager 是 WebSocket 管理器的介面
type WebsocketManager interface {
	// Broadcast 廣播消息給所有連接的客戶端
	Broadcast(message interface{}) error

	// BroadcastToType 廣播消息給特定類型的客戶端
	BroadcastToType(message interface{}, isDealer bool) error

	// PublishToTopic 發布消息到指定主題
	PublishToTopic(topic string, message interface{}) error

	// GetTopicSubscriberCount 獲取主題訂閱者數量
	GetTopicSubscriberCount(topic string) int

	// Start 啟動管理器
	Start(ctx context.Context)

	// Shutdown 關閉管理器
	Shutdown()
}

// GameService 是遊戲服務的介面
type GameService interface {
	// CreateGame 創建新遊戲
	CreateGame(ctx context.Context) (string, error)

	// EndGame 結束遊戲
	EndGame(ctx context.Context, gameID string) error

	// GetGameInfo 獲取遊戲信息
	GetGameInfo(ctx context.Context, gameID string) (*GameInfo, error)

	// DrawBall 抽取球
	DrawBall(ctx context.Context, gameID string, ballType string) (*BallInfo, error)

	// SelectSide 選擇邊
	SelectSide(ctx context.Context, gameID string, side string) error
}

// GameInfo 遊戲信息
type GameInfo struct {
	GameID       string    `json:"game_id"`
	CurrentStage string    `json:"current_stage"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time,omitempty"`
	IsCancelled  bool      `json:"is_cancelled"`
	CancelReason string    `json:"cancel_reason,omitempty"`
}

// BallInfo 球信息
type BallInfo struct {
	GameID     string    `json:"game_id"`
	BallNumber string    `json:"ball_number"`
	BallType   string    `json:"ball_type"`
	DrawTime   time.Time `json:"draw_time"`
}
