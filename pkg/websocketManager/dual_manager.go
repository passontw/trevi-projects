package websocketManager

import (
	"context"
	"log"
	"sync"
)

// 客戶端類型
type ClientType string

const (
	ClientTypeDealer ClientType = "DEALER" // 荷官客戶端
	ClientTypePlayer ClientType = "PLAYER" // 遊戲客戶端
)

// DualWebSocketService 管理玩家端和荷官端的 WebSocket 服務
type DualWebSocketService struct {
	dealerManager *Manager           // 荷官端 WebSocket 管理器
	playerManager *Manager           // 遊戲端 WebSocket 管理器
	ctx           context.Context    // 上下文
	cancel        context.CancelFunc // 取消函數
	mutex         sync.RWMutex       // 讀寫鎖
	isRunning     bool               // 服務是否運行中
}

// NewDualWebSocketService 創建新的雙端口 WebSocket 服務
func NewDualWebSocketService(
	dealerAuthFunc func(token string) (uint, error),
	playerAuthFunc func(token string) (uint, error),
) *DualWebSocketService {
	// 創建帶有取消功能的上下文
	ctx, cancel := context.WithCancel(context.Background())

	return &DualWebSocketService{
		dealerManager: NewManager(dealerAuthFunc),
		playerManager: NewManager(playerAuthFunc),
		ctx:           ctx,
		cancel:        cancel,
		mutex:         sync.RWMutex{},
		isRunning:     false,
	}
}

// Start 啟動雙端口 WebSocket 服務
func (s *DualWebSocketService) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		log.Println("Dual WebSocket Service: 服務已在運行中")
		return
	}

	// 啟動荷官端管理器
	go s.dealerManager.Start(s.ctx)
	log.Println("Dual WebSocket Service: 荷官端 WebSocket 管理器已啟動")

	// 啟動遊戲端管理器
	go s.playerManager.Start(s.ctx)
	log.Println("Dual WebSocket Service: 遊戲端 WebSocket 管理器已啟動")

	s.isRunning = true
	log.Println("Dual WebSocket Service: 雙端口服務已啟動")
}

// Stop 停止雙端口 WebSocket 服務
func (s *DualWebSocketService) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		log.Println("Dual WebSocket Service: 服務未運行")
		return
	}

	// 調用上下文取消函數，兩個管理器會自動關閉
	s.cancel()
	log.Println("Dual WebSocket Service: 已發送關閉信號")

	// 創建新的上下文以供將來使用
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.isRunning = false
	log.Println("Dual WebSocket Service: 雙端口服務已停止")
}

// GetDealerManager 獲取荷官端 WebSocket 管理器
func (s *DualWebSocketService) GetDealerManager() *Manager {
	return s.dealerManager
}

// GetPlayerManager 獲取遊戲端 WebSocket 管理器
func (s *DualWebSocketService) GetPlayerManager() *Manager {
	return s.playerManager
}

// BroadcastToAllDealers 向所有荷官端廣播消息
func (s *DualWebSocketService) BroadcastToAllDealers(message interface{}) error {
	return s.dealerManager.BroadcastToAll(message)
}

// BroadcastToAllPlayers 向所有遊戲端廣播消息
func (s *DualWebSocketService) BroadcastToAllPlayers(message interface{}) error {
	return s.playerManager.BroadcastToAll(message)
}

// BroadcastToAll 向所有客戶端（荷官和遊戲端）廣播消息
func (s *DualWebSocketService) BroadcastToAll(message interface{}) error {
	// 先向荷官端廣播
	err1 := s.dealerManager.BroadcastToAll(message)

	// 再向遊戲端廣播
	err2 := s.playerManager.BroadcastToAll(message)

	// 如果有任何一個廣播出錯，返回錯誤
	if err1 != nil {
		return err1
	}

	return err2
}

// IsRunning 檢查服務是否運行中
func (s *DualWebSocketService) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isRunning
}
