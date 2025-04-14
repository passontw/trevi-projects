package websocketManager

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket 連接升級器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// 允許所有來源的跨域請求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	// 設置60秒的握手超時
	HandshakeTimeout: 60 * time.Second,
}

// WebSocketHandler 提供 WebSocket 連接處理
type WebSocketHandler struct {
	manager *Manager
}

// NewWebSocketHandler 創建新的 WebSocket 處理程序
func NewWebSocketHandler(manager *Manager) *WebSocketHandler {
	return &WebSocketHandler{
		manager: manager,
	}
}

// HandleConnection 處理 WebSocket 連接請求
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	// 升級 HTTP 連接為 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket Handler: Failed to upgrade connection: %v\n", err)
		return
	}

	// 配置連接
	conn.SetReadLimit(4 * 1024 * 1024) // 4MB
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	// 生成客戶端 ID
	clientID := uuid.New().String()

	// 創建新的客戶端
	client := &Client{
		ID:           clientID,
		Conn:         conn,
		Send:         make(chan []byte, 256), // 緩沖區設置為 256 條訊息
		manager:      h.manager,
		LastActivity: time.Now(),
		IsAuthed:     false,
		closeChan:    make(chan struct{}),
	}

	// 註冊客戶端
	h.manager.register <- client

	// 啟動訊息讀取和寫入
	go client.ReadPump()
	go client.WritePump()

	log.Printf("WebSocket Handler: New connection established for client %s\n", clientID)
}

// 在 Core 模組中提供 WebSocketManager
func ProvideWebSocketManager(auth func(token string) (uint, error)) *Manager {
	return NewManager(auth)
}

// 處理WebSocket認證請求
func (h *WebSocketHandler) HandleAuthRequest(c *gin.Context) {
	token := c.GetHeader("Authorization")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "缺少認證令牌",
		})
		return
	}

	// 驗證令牌，這裡只是演示，實際應用中應該使用更複雜的驗證機制
	userID, err := h.manager.auth(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "認證失敗: " + err.Error(),
		})
		return
	}

	// 返回WebSocket連接URL和用戶信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "認證成功",
		"data": gin.H{
			"wsURL":  "/ws",
			"userID": userID,
			"token":  token,
		},
	})
}
