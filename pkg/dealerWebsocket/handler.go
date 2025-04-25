package dealerWebsocket

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket 處理器結構體
type WebSocketHandler struct {
	manager     *Manager                         // WebSocket 管理器
	upgrader    websocket.Upgrader               // WebSocket 升級器
	authFunc    func(token string) (uint, error) // 認證函數
	connections int64                            // 連接計數
}

// 創建新的 WebSocket 處理器
func NewWebSocketHandler(manager *Manager, authFunc func(token string) (uint, error)) *WebSocketHandler {
	if manager == nil {
		log.Fatal("Dealer WebSocket Handler: Manager cannot be nil")
	}

	// 配置 WebSocket 升級器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允許所有源的連接
		CheckOrigin: func(r *http.Request) bool {
			// 在生產環境中應該實現更嚴格的檢查
			return true
		},
	}

	return &WebSocketHandler{
		manager:     manager,
		upgrader:    upgrader,
		authFunc:    authFunc,
		connections: 0,
	}
}

// 處理 WebSocket 連接請求的 HTTP 處理器
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升級 HTTP 連接到 WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Dealer WebSocket Handler: Upgrade error: %v", err)
		return
	}

	// 生成客戶端唯一標識
	clientID := uuid.New().String()
	log.Printf("Dealer WebSocket Handler: New connection from %s, assigned ID: %s", conn.RemoteAddr(), clientID)

	// 創建新的客戶端
	client := &Client{
		ID:           clientID,
		Conn:         conn,
		Send:         make(chan []byte, 256), // 緩衝區大小
		manager:      h.manager,
		LastActivity: time.Now(),
		IsAuthed:     false, // 初始未認證
	}

	// 增加連接計數
	atomic.AddInt64(&h.connections, 1)
	log.Printf("Dealer WebSocket Handler: Active connections: %d", atomic.LoadInt64(&h.connections))

	// 向管理器註冊客戶端
	h.manager.register <- client

	// 啟動客戶端讀寫協程
	go client.ReadPump()
	go client.WritePump()
}

// 獲取當前活躍連接數
func (h *WebSocketHandler) GetConnectionCount() int64 {
	return atomic.LoadInt64(&h.connections)
}

// 註冊 WebSocket 路由
func (h *WebSocketHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/dealer/ws", h.HandleWebSocket)
}
