package websocketManager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 心跳間隔設置為15秒，避開30秒超時
	heartbeatInterval = 15 * time.Second
	// 連接超時設置
	readTimeout  = 30 * time.Second
	writeTimeout = 10 * time.Second
	// 非活躍連接超時（5分鐘）
	inactivityTimeout = 5 * time.Minute
)

// 客戶端結構體，代表一個 WebSocket 連接
type Client struct {
	ID              string          // 客戶端唯一標識
	UserID          uint            // 用戶 ID
	Conn            *websocket.Conn // WebSocket 連接
	Send            chan []byte     // 發送訊息的通道
	manager         *Manager        // 所屬的管理器
	LastActivity    time.Time       // 最後活動時間
	IsAuthed        bool            // 是否已認證
	closeChan       chan struct{}   // 關閉通道
	heartbeatTicker *time.Ticker    // 心跳定時器
	connMutex       sync.Mutex      // 連接鎖，防止並發讀寫
}

// 客戶端狀態
type ClientState int

const (
	StateDisconnected ClientState = iota
	StateConnecting
	StateConnected
	StateFailed
)

// 訊息結構體
type Message struct {
	Type      string      `json:"type"`                 // 訊息類型
	Content   interface{} `json:"content,omitempty"`    // 訊息內容
	From      string      `json:"from,omitempty"`       // 發送方
	To        string      `json:"to,omitempty"`         // 接收方
	Timestamp int64       `json:"timestamp,omitempty"`  // 時間戳
	MessageID string      `json:"message_id,omitempty"` // 消息ID
}

// 心跳訊息
type HeartbeatMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
}

// WebSocket 管理器結構體
type Manager struct {
	clients     map[string]*Client               // 所有已連接的客戶端
	userClients map[uint]map[string]bool         // 用戶ID對應的客戶端map
	broadcast   chan []byte                      // 廣播訊息的通道
	register    chan *Client                     // 註冊客戶端的通道
	unregister  chan *Client                     // 註銷客戶端的通道
	shutdown    chan struct{}                    // 關閉的通道
	auth        func(token string) (uint, error) // 認證函數
	mutex       sync.RWMutex                     // 讀寫鎖
}

// 創建新的 WebSocket 管理器
func NewManager(authFunc func(token string) (uint, error)) *Manager {
	return &Manager{
		clients:     make(map[string]*Client),
		userClients: make(map[uint]map[string]bool),
		broadcast:   make(chan []byte, 100), // 增大緩衝區
		register:    make(chan *Client, 10),
		unregister:  make(chan *Client, 10),
		shutdown:    make(chan struct{}),
		auth:        authFunc,
		mutex:       sync.RWMutex{},
	}
}

// 啟動 WebSocket 管理器
func (manager *Manager) Start(ctx context.Context) {
	log.Println("WebSocket Manager: Starting...")

	// 創建獨立的上下文確保完整的生命週期
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
		log.Println("WebSocket Manager: Using fallback background context")
	}

	// 恢復panic以防止整個服務崩潰
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WebSocket Manager: Recovered from panic: %v", r)
		}
	}()

	// 非活躍連接檢查定時器
	inactivityTicker := time.NewTicker(1 * time.Minute)
	defer inactivityTicker.Stop()

	log.Println("WebSocket Manager: Running main event loop")
	running := true

	for running {
		select {
		case <-ctx.Done():
			log.Println("WebSocket Manager: Context cancelled, shutting down...")
			manager.cleanupAllConnections()
			running = false

		case client, ok := <-manager.register:
			if !ok {
				log.Println("WebSocket Manager: Register channel closed")
				continue
			}

			manager.mutex.Lock()
			manager.clients[client.ID] = client
			manager.mutex.Unlock()
			log.Printf("WebSocket Manager: Client %s registered\n", client.ID)

		case client, ok := <-manager.unregister:
			if !ok {
				log.Println("WebSocket Manager: Unregister channel closed")
				continue
			}

			manager.removeClient(client)

		case message, ok := <-manager.broadcast:
			if !ok {
				log.Println("WebSocket Manager: Broadcast channel closed")
				continue
			}

			manager.broadcastMessage(message)

		case <-inactivityTicker.C:
			manager.cleanupInactiveConnections()
		}
	}

	log.Println("WebSocket Manager: Event loop terminated")
}

// 清理所有連接
func (manager *Manager) cleanupAllConnections() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	log.Printf("WebSocket Manager: Cleaning up all %d connections", len(manager.clients))

	for _, client := range manager.clients {
		if client.heartbeatTicker != nil {
			client.heartbeatTicker.Stop()
		}

		if client.closeChan != nil {
			close(client.closeChan)
		}

		client.Conn.Close()
		close(client.Send)
	}

	// 清空映射
	manager.clients = make(map[string]*Client)
	manager.userClients = make(map[uint]map[string]bool)
}

// 移除指定客戶端
func (manager *Manager) removeClient(client *Client) {
	if _, ok := manager.clients[client.ID]; !ok {
		return
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	// 從用戶-客戶端映射中移除
	if client.IsAuthed {
		delete(manager.userClients[client.UserID], client.ID)
		// 如果用戶沒有其他客戶端連接，則刪除用戶映射
		if len(manager.userClients[client.UserID]) == 0 {
			delete(manager.userClients, client.UserID)
		}
	}

	// 停止心跳
	if client.heartbeatTicker != nil {
		client.heartbeatTicker.Stop()
	}

	// 關閉信號通道
	if client.closeChan != nil {
		close(client.closeChan)
	}

	// 關閉連接
	client.Conn.Close()

	// 關閉發送通道
	close(client.Send)

	// 從客戶端列表中刪除
	delete(manager.clients, client.ID)

	log.Printf("WebSocket Manager: Client %s unregistered\n", client.ID)
}

// 廣播消息到所有客戶端
func (manager *Manager) broadcastMessage(message []byte) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	failedClients := make([]string, 0)

	for id, client := range manager.clients {
		select {
		case client.Send <- message:
			// 消息已送入通道
		default:
			// 發送通道已滿或已關閉，記錄待移除的客戶端
			failedClients = append(failedClients, id)
		}
	}

	// 如果有發送失敗的客戶端，解鎖後移除它們
	if len(failedClients) > 0 {
		manager.mutex.RUnlock()
		manager.mutex.Lock()

		for _, id := range failedClients {
			if client, exists := manager.clients[id]; exists {
				log.Printf("WebSocket Manager: Removing client %s due to full send buffer", id)

				// 停止心跳
				if client.heartbeatTicker != nil {
					client.heartbeatTicker.Stop()
				}

				// 關閉信號通道
				if client.closeChan != nil {
					close(client.closeChan)
				}

				// 關閉連接
				client.Conn.Close()

				// 從用戶映射中移除
				if client.IsAuthed {
					delete(manager.userClients[client.UserID], id)
					if len(manager.userClients[client.UserID]) == 0 {
						delete(manager.userClients, client.UserID)
					}
				}

				// 關閉發送通道
				close(client.Send)

				// 從客戶端列表中刪除
				delete(manager.clients, id)
			}
		}

		manager.mutex.Unlock()
		manager.mutex.RLock()
	}
}

// 清理非活躍連接
func (manager *Manager) cleanupInactiveConnections() {
	threshold := time.Now().Add(-inactivityTimeout)

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	inactiveCount := 0

	for id, client := range manager.clients {
		if client.LastActivity.Before(threshold) {
			inactiveCount++
			log.Printf("WebSocket Manager: Client %s inactive for too long, closing connection", id)

			// 停止心跳
			if client.heartbeatTicker != nil {
				client.heartbeatTicker.Stop()
			}

			// 關閉信號通道
			if client.closeChan != nil {
				close(client.closeChan)
			}

			// 關閉連接
			client.Conn.Close()

			// 從用戶映射中移除
			if client.IsAuthed {
				delete(manager.userClients[client.UserID], id)
				if len(manager.userClients[client.UserID]) == 0 {
					delete(manager.userClients, client.UserID)
				}
			}

			// 關閉發送通道
			close(client.Send)

			// 從客戶端列表中刪除
			delete(manager.clients, id)
		}
	}

	if inactiveCount > 0 {
		log.Printf("WebSocket Manager: Removed %d inactive connections", inactiveCount)
	}
}

// 驗證客戶端
func (manager *Manager) AuthenticateClient(client *Client, token string) error {
	userID, err := manager.auth(token)
	if err != nil {
		return err
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	client.UserID = userID
	client.IsAuthed = true

	// 將客戶端添加到用戶-客戶端映射
	if _, exists := manager.userClients[userID]; !exists {
		manager.userClients[userID] = make(map[string]bool)
	}
	manager.userClients[userID][client.ID] = true

	log.Printf("WebSocket Manager: Client %s authenticated for user %d\n", client.ID, userID)
	return nil
}

// 廣播訊息給所有已認證的客戶端
func (manager *Manager) BroadcastToAll(message interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	manager.broadcast <- msgBytes
	return nil
}

// 向指定用戶發送訊息
func (manager *Manager) SendToUser(userID uint, message interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	// 檢查用戶是否有連接的客戶端
	clientMap, exists := manager.userClients[userID]
	if !exists || len(clientMap) == 0 {
		return fmt.Errorf("no connected clients for user %d", userID)
	}

	// 發送訊息給用戶的所有客戶端
	for clientID := range clientMap {
		if client, ok := manager.clients[clientID]; ok && client.IsAuthed {
			select {
			case client.Send <- msgBytes:
				// 訊息已送入通道
			default:
				// 發送通道已滿或已關閉，移除客戶端
				client.connMutex.Lock()
				if client.heartbeatTicker != nil {
					client.heartbeatTicker.Stop()
				}
				if client.closeChan != nil {
					close(client.closeChan)
				}
				client.Conn.Close()
				client.connMutex.Unlock()

				// 移除用戶客戶端映射
				delete(manager.userClients[userID], clientID)
				if len(manager.userClients[userID]) == 0 {
					delete(manager.userClients, userID)
				}

				// 移除客戶端
				close(client.Send)
				delete(manager.clients, clientID)
				log.Printf("WebSocket Manager: Client %s removed (failed to send to user)\n", clientID)
			}
		}
	}

	return nil
}

// 客戶端讀取訊息
func (client *Client) ReadPump() {
	// 初始化關閉通道
	if client.closeChan == nil {
		client.closeChan = make(chan struct{})
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("WebSocket Manager: Client %s ReadPump recovered from panic: %v\n", client.ID, r)
		}
		log.Printf("WebSocket Manager: Client %s ReadPump exiting\n", client.ID)
		client.manager.unregister <- client
		client.Conn.Close()
	}()

	// 設置讀取參數
	client.Conn.SetReadLimit(4096) // 4KB
	client.Conn.SetReadDeadline(time.Now().Add(readTimeout))

	// 設置Pong處理器，更新最後活動時間
	client.Conn.SetPongHandler(func(string) error {
		client.connMutex.Lock()
		defer client.connMutex.Unlock()

		client.Conn.SetReadDeadline(time.Now().Add(readTimeout))
		client.LastActivity = time.Now()
		return nil
	})

	// 啟動心跳
	client.StartHeartbeat()

	for {
		select {
		case <-client.closeChan:
			log.Printf("WebSocket Manager: Client %s received close signal\n", client.ID)
			return
		default:
			messageType, message, err := client.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket Manager: Client %s unexpected close: %v\n", client.ID, err)
				} else {
					log.Printf("WebSocket Manager: Client %s read error: %v\n", client.ID, err)
				}
				return
			}

			// 過濾非文本消息
			if messageType != websocket.TextMessage {
				log.Printf("WebSocket Manager: Client %s received non-text message type: %d\n", client.ID, messageType)
				continue
			}

			client.connMutex.Lock()
			client.LastActivity = time.Now()
			client.Conn.SetReadDeadline(time.Now().Add(readTimeout))
			client.connMutex.Unlock()

			// 處理接收到的訊息
			var msg Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("WebSocket Manager: Error unmarshaling message from client %s: %v\n", client.ID, err)
				log.Printf("WebSocket Manager: Raw message: %s\n", string(message))
				continue
			}

			// 處理心跳訊息
			if msg.Type == "heartbeat" {
				// 客戶端發送的心跳，直接回應
				heartbeatResponse := HeartbeatMessage{
					Type:      "heartbeat",
					Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
				}
				responseBytes, _ := json.Marshal(heartbeatResponse)

				select {
				case client.Send <- responseBytes:
					// 心跳已送入通道
				default:
					log.Printf("WebSocket Manager: Client %s send channel full for heartbeat\n", client.ID)
				}
				continue
			}

			// 處理基準測試訊息
			if msg.Type == "benchmark" {
				// 確保我們返回一個與發送格式相同的消息
				// 直接使用收到的消息，僅更新時間戳
				benchMsg := msg
				// 使用服務端當前時間，以便客戶端計算延遲
				benchMsg.Timestamp = time.Now().UnixNano()

				// 直接使用相同的消息結構，確保格式一致
				responseBytes, err := json.Marshal(benchMsg)
				if err != nil {
					log.Printf("WebSocket Manager: Error marshaling benchmark response: %v\n", err)
					continue
				}

				// 發送回客戶端
				select {
				case client.Send <- responseBytes:
					// 成功發送
					log.Printf("WebSocket Manager: Processed benchmark message from client %s\n", client.ID)
				default:
					log.Printf("WebSocket Manager: Failed to send benchmark response, buffer full for client %s\n", client.ID)
				}
				continue
			}

			// 處理其他訊息...
			log.Printf("WebSocket Manager: Received message from client %s: %s\n", client.ID, message)
		}
	}
}

// 客戶端寫入訊息
func (client *Client) WritePump() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WebSocket Manager: Client %s WritePump recovered from panic: %v\n", client.ID, r)
		}
		log.Printf("WebSocket Manager: Client %s WritePump exiting\n", client.ID)
		client.Conn.Close()
	}()

	// 確保 closeChan 已初始化
	if client.closeChan == nil {
		client.closeChan = make(chan struct{})
	}

	for {
		select {
		case <-client.closeChan:
			log.Printf("WebSocket Manager: Client %s writer received close signal\n", client.ID)
			return
		case message, ok := <-client.Send:
			if !ok {
				// 通道已關閉
				log.Printf("WebSocket Manager: Client %s send channel closed\n", client.ID)
				client.connMutex.Lock()
				err := client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				client.connMutex.Unlock()
				if err != nil {
					log.Printf("WebSocket Manager: Client %s error sending close message: %v\n", client.ID, err)
				}
				return
			}

			client.connMutex.Lock()
			// 檢查連接是否有效
			if client.Conn == nil {
				client.connMutex.Unlock()
				log.Printf("WebSocket Manager: Client %s connection is nil\n", client.ID)
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				client.connMutex.Unlock()
				log.Printf("WebSocket Manager: Client %s error getting writer: %v\n", client.ID, err)
				return
			}

			_, err = w.Write(message)
			if err != nil {
				client.connMutex.Unlock()
				log.Printf("WebSocket Manager: Client %s error writing message: %v\n", client.ID, err)
				return
			}

			// 將佇列中的其他訊息也一起發送，但限制數量
			n := len(client.Send)
			maxMessages := 10 // 每次最多處理10條消息
			if n > maxMessages {
				n = maxMessages
			}

			for i := 0; i < n; i++ {
				nextMsg, ok := <-client.Send
				if !ok {
					break
				}
				w.Write([]byte{'\n'})
				w.Write(nextMsg)
			}

			if err := w.Close(); err != nil {
				client.connMutex.Unlock()
				log.Printf("WebSocket Manager: Client %s error closing writer: %v\n", client.ID, err)
				return
			}
			client.connMutex.Unlock()

			// 更新最後活動時間
			client.connMutex.Lock()
			client.LastActivity = time.Now()
			client.connMutex.Unlock()
		}
	}
}

// 開始心跳
func (client *Client) StartHeartbeat() {
	client.connMutex.Lock()
	defer client.connMutex.Unlock()

	// 停止舊的心跳計時器（如果存在）
	if client.heartbeatTicker != nil {
		client.heartbeatTicker.Stop()
		client.heartbeatTicker = nil
	}

	// 創建新的心跳計時器
	client.heartbeatTicker = time.NewTicker(heartbeatInterval)
	log.Printf("WebSocket Manager: Started heartbeat for client %s\n", client.ID)

	// 創建本地副本
	localCloseChan := client.closeChan
	heartbeatTicker := client.heartbeatTicker

	// 啟動心跳協程
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("WebSocket Manager: Heartbeat routine for client %s recovered from panic: %v\n", client.ID, r)
			}
			log.Printf("WebSocket Manager: Heartbeat routine for client %s exiting\n", client.ID)
		}()

		for {
			select {
			case <-localCloseChan:
				log.Printf("WebSocket Manager: Client %s heartbeat received close signal\n", client.ID)
				return
			case <-heartbeatTicker.C:
				// 使用本地副本確保即使 client 被修改也能正確工作
				if err := client.sendPing(); err != nil {
					log.Printf("WebSocket Manager: Client %s ping failed: %v\n", client.ID, err)
					return
				}
			}
		}
	}()
}

// 發送 Ping 消息
func (client *Client) sendPing() error {
	client.connMutex.Lock()
	defer client.connMutex.Unlock()

	// 檢查連接是否有效
	if client.Conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 發送Ping訊息
	if err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeTimeout)); err != nil {
		log.Printf("WebSocket Manager: Client %s ping error: %v\n", client.ID, err)

		// 直接調用 attemptReconnect (已在 goroutine 中)
		client.connMutex.Unlock() // 先解鎖以避免死鎖
		client.attemptReconnect() // 直接調用而不是啟動新的 goroutine
		client.connMutex.Lock()   // 重新鎖定
		return err
	}

	// 發送應用層心跳訊息
	heartbeat := HeartbeatMessage{
		Type:      "heartbeat",
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}
	heartbeatBytes, _ := json.Marshal(heartbeat)

	select {
	case client.Send <- heartbeatBytes:
		// 心跳已送入通道
	default:
		// 發送通道已滿，可能需要處理
		log.Printf("WebSocket Manager: Client %s send channel full, cannot send heartbeat\n", client.ID)
	}

	return nil
}

// 嘗試重連
func (client *Client) attemptReconnect() {
	// 不再嘗試重連，直接關閉連接
	log.Printf("WebSocket Manager: Client %s connection terminated, waiting for client to reconnect\n", client.ID)

	// 清理客戶端資源
	client.connMutex.Lock()
	// 關閉舊連接前先停止心跳
	if client.heartbeatTicker != nil {
		client.heartbeatTicker.Stop()
		client.heartbeatTicker = nil
	}

	// 關閉舊連接
	if client.Conn != nil {
		client.Conn.Close()
	}
	client.connMutex.Unlock()

	// 更新客戶端在管理器中的狀態
	if client.manager != nil {
		client.manager.mutex.Lock()
		// 設置客戶端狀態為已斷開
		client.LastActivity = time.Now() // 更新最後活動時間，防止被立即清理
		client.manager.mutex.Unlock()

		// 通知管理器註銷客戶端
		client.manager.unregister <- client
	}
}

// 關閉連接
func (manager *Manager) Shutdown() {
	log.Println("WebSocket Manager: Shutdown initiated, closing all connections...")

	// 通知所有客戶端關閉
	manager.mutex.Lock()
	clientCount := len(manager.clients)

	for id, client := range manager.clients {
		log.Printf("WebSocket Manager: Closing connection for client %s", id)

		if client.heartbeatTicker != nil {
			client.heartbeatTicker.Stop()
		}

		if client.closeChan != nil {
			close(client.closeChan)
		}

		// 向客戶端發送關閉消息
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server shutting down")
		_ = client.Conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))

		client.Conn.Close()
		close(client.Send)
	}

	// 清空客戶端映射
	manager.clients = make(map[string]*Client)
	manager.userClients = make(map[uint]map[string]bool)
	manager.mutex.Unlock()

	// 關閉管理器的shutdown通道
	close(manager.shutdown)

	log.Printf("WebSocket Manager: Successfully closed %d connections", clientCount)
}
