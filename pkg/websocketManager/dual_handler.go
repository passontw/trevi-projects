package websocketManager

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DualWebSocketHandler 提供荷官端和遊戲端 WebSocket 連接處理
type DualWebSocketHandler struct {
	service *DualWebSocketService
}

// NewDualWebSocketHandler 創建新的雙端口 WebSocket 處理器
func NewDualWebSocketHandler(service *DualWebSocketService) *DualWebSocketHandler {
	return &DualWebSocketHandler{
		service: service,
	}
}

// HandleDealerConnection 處理荷官端 WebSocket 連接請求
func (h *DualWebSocketHandler) HandleDealerConnection(c *gin.Context) {
	// 設置客戶端類型為荷官
	clientType := ClientTypeDealer

	// 升級 HTTP 連接為 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Dual WebSocket Handler: Failed to upgrade dealer connection: %v\n", err)
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
		manager:      h.service.GetDealerManager(),
		LastActivity: time.Now(),
		IsAuthed:     false, // 荷官端可配置為不需要驗證
		closeChan:    make(chan struct{}),
	}

	// 註冊客戶端
	h.service.GetDealerManager().register <- client

	// 啟動訊息讀取和寫入
	go client.ReadPump()
	go client.WritePump()

	log.Printf("Dual WebSocket Handler: New %s connection established for client %s\n", clientType, clientID)
}

// HandlePlayerConnection 處理遊戲端 WebSocket 連接請求
func (h *DualWebSocketHandler) HandlePlayerConnection(c *gin.Context) {
	// 設置客戶端類型為遊戲端
	clientType := ClientTypePlayer

	// 取得令牌（如果需要）
	token := c.Query("token")

	// 升級 HTTP 連接為 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Dual WebSocket Handler: Failed to upgrade player connection: %v\n", err)
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
		manager:      h.service.GetPlayerManager(),
		LastActivity: time.Now(),
		IsAuthed:     false, // 初始設置為未驗證
		closeChan:    make(chan struct{}),
	}

	// 註冊客戶端
	h.service.GetPlayerManager().register <- client

	// 啟動訊息讀取和寫入
	go client.ReadPump()
	go client.WritePump()

	log.Printf("Dual WebSocket Handler: New %s connection established for client %s\n", clientType, clientID)

	// 如果提供了令牌，嘗試進行身份驗證
	if token != "" {
		go func() {
			// 給客戶端一些時間完成初始化
			time.Sleep(500 * time.Millisecond)

			// 進行身份驗證
			err := h.service.GetPlayerManager().AuthenticateClient(client, token)
			if err != nil {
				log.Printf("Dual WebSocket Handler: Authentication failed for client %s: %v\n", clientID, err)

				// 發送驗證失敗消息
				errorMsg := Message{
					Type:    "error",
					Content: map[string]interface{}{"code": "AUTH_FAILED", "message": "驗證失敗: " + err.Error()},
				}

				if msgBytes, err := json.Marshal(errorMsg); err == nil {
					select {
					case client.Send <- msgBytes:
						// 錯誤消息已送入通道
					default:
						log.Printf("Dual WebSocket Handler: Send buffer full for client %s\n", clientID)
					}
				}
			} else {
				log.Printf("Dual WebSocket Handler: Client %s authenticated successfully\n", clientID)
			}
		}()
	}
}

// HandleDealerAuthRequest 處理荷官端 WebSocket 認證請求
func (h *DualWebSocketHandler) HandleDealerAuthRequest(c *gin.Context) {
	token := c.GetHeader("Authorization")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "缺少認證令牌",
		})
		return
	}

	// 驗證令牌
	userID, err := h.service.GetDealerManager().auth(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "荷官認證失敗: " + err.Error(),
		})
		return
	}

	// 返回 WebSocket 連接 URL 和用戶信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "荷官認證成功",
		"data": gin.H{
			"wsURL":  "/dealer/ws",
			"userID": userID,
			"token":  token,
		},
	})
}

// HandlePlayerAuthRequest 處理遊戲端 WebSocket 認證請求
func (h *DualWebSocketHandler) HandlePlayerAuthRequest(c *gin.Context) {
	token := c.GetHeader("Authorization")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "缺少認證令牌",
		})
		return
	}

	// 驗證令牌
	userID, err := h.service.GetPlayerManager().auth(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "玩家認證失敗: " + err.Error(),
		})
		return
	}

	// 返回 WebSocket 連接 URL 和用戶信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "玩家認證成功",
		"data": gin.H{
			"wsURL":  "/player/ws",
			"userID": userID,
			"token":  token,
		},
	})
}
