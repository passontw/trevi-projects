package dealerWebsocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 心跳間隔設置為15秒
	heartbeatInterval = 15 * time.Second
	// 連接超時設置
	readTimeout  = 60 * time.Second
	writeTimeout = 10 * time.Second
	// 非活躍連接超時（5分鐘）
	inactivityTimeout = 5 * time.Minute
	// 發送通道緩衝大小
	writeWait = 10 * time.Second

	// 允許客戶端不發送ping的最大時間
	pongWait = 60 * time.Second

	// 發送ping的頻率，必須小於pongWait
	pingPeriod = (pongWait * 9) / 10

	// 消息最大大小
	maxMessageSize = 512 * 1024 // 512KB
)

var (
	// 換行符用於分隔消息
	newline = []byte{'\n'}
)

// 心跳消息結構
type HeartbeatMessage struct {
	Type      string `json:"type"`      // 消息類型
	Timestamp int64  `json:"timestamp"` // 時間戳（毫秒）
}

// 客戶端結構體，代表一個 WebSocket 連接
type Client struct {
	ID                   string          // 客戶端唯一標識
	UserID               uint            // 用戶 ID
	Conn                 *websocket.Conn // WebSocket 連接
	Send                 chan []byte     // 發送訊息的通道
	manager              *Manager        // 所屬的管理器
	LastActivity         time.Time       // 最後活動時間
	IsAuthed             bool            // 是否已認證
	closeChan            chan struct{}   // 關閉通道
	heartbeatTicker      *time.Ticker    // 心跳定時器
	connMutex            sync.Mutex      // 連接鎖，防止並發讀寫
	closeReason          string          // 關閉原因
	rooms                []string        // 客戶端所在的房間列表
	games                []string        // 客戶端所在的遊戲列表
	heartbeatErrorLogged bool
}

// 客戶端狀態
type ClientState int

const (
	StateDisconnected ClientState = iota
	StateConnecting
	StateConnected
	StateFailed
)

// MessageHandler 定義消息處理器介面
type MessageHandler interface {
	// 處理消息
	HandleMessage(client *Client, messageType string, data interface{})
	// 處理連接
	HandleConnect(client *Client)
	// 處理斷開連接
	HandleDisconnect(client *Client)
}

// Message 消息結構
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// WebSocket 管理器結構體
type Manager struct {
	clients         map[*Client]bool
	authClients     map[uint][]*Client
	userClients     map[uint]map[string]*Client // 用戶ID到客戶端ID的映射
	roomClients     map[string]map[*Client]bool // 房間ID到客戶端的映射
	gameClients     map[string]map[*Client]bool // 遊戲ID到客戶端的映射
	directBroadcast map[uint]chan []byte
	messageHandler  MessageHandler
	register        chan *Client
	unregister      chan *Client
	broadcast       chan []byte
	shutdown        chan struct{}
	auth            func(token string) (uint, error)
	mutex           sync.RWMutex
	wsHandler       *WebSocketHandler // 引用 WebSocketHandler 以更新連接計數
}

// 創建新的 WebSocket 管理器
func NewManager(authFunc func(token string) (uint, error)) *Manager {
	return &Manager{
		clients:         make(map[*Client]bool),
		authClients:     make(map[uint][]*Client),
		userClients:     make(map[uint]map[string]*Client),
		roomClients:     make(map[string]map[*Client]bool), // 初始化房間客戶端映射
		gameClients:     make(map[string]map[*Client]bool), // 初始化遊戲客戶端映射
		directBroadcast: make(map[uint]chan []byte),
		messageHandler:  nil,
		register:        make(chan *Client, 10),
		unregister:      make(chan *Client, 10),
		broadcast:       make(chan []byte, 100),
		shutdown:        make(chan struct{}),
		auth:            authFunc,
		mutex:           sync.RWMutex{},
		wsHandler:       nil, // 初始為 nil，稍後在 WebSocketHandler 建立時設置
	}
}

// 設置 WebSocketHandler 的引用
func (manager *Manager) SetWebSocketHandler(wsHandler *WebSocketHandler) {
	manager.wsHandler = wsHandler
}

// 啟動 WebSocket 管理器
func (manager *Manager) Start(ctx context.Context) {
	log.Println("Dealer WebSocket Manager: Starting...")

	// 創建獨立的上下文確保完整的生命週期
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
		log.Println("Dealer WebSocket Manager: Using fallback background context")
	}

	// 恢復panic以防止整個服務崩潰
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Dealer WebSocket Manager: Recovered from panic: %v", r)
		}
	}()

	// 非活躍連接檢查定時器
	inactivityTicker := time.NewTicker(1 * time.Minute)
	defer inactivityTicker.Stop()

	log.Println("Dealer WebSocket Manager: Running main event loop")
	running := true

	for running {
		select {
		case <-ctx.Done():
			// 只在有活躍連接時才輸出關閉訊息
			clientCount := len(manager.clients)
			if clientCount > 0 {
				log.Println("Dealer WebSocket Manager: Context cancelled, shutting down...")
			}
			manager.cleanupAllConnections()
			running = false

		case client, ok := <-manager.register:
			if !ok {
				log.Println("Dealer WebSocket Manager: Register channel closed")
				continue
			}

			manager.mutex.Lock()
			manager.clients[client] = true
			manager.mutex.Unlock()
			log.Printf("Dealer WebSocket Manager: Client %s registered\n", client.ID)

		case client, ok := <-manager.unregister:
			if !ok {
				log.Println("Dealer WebSocket Manager: Unregister channel closed")
				continue
			}

			manager.removeClient(client)

		case message, ok := <-manager.broadcast:
			if !ok {
				log.Println("Dealer WebSocket Manager: Broadcast channel closed")
				continue
			}

			manager.broadcastMessage(message)

		case <-inactivityTicker.C:
			manager.cleanupInactiveConnections()
		}
	}

	// 只在有活躍連接時才輸出關閉訊息
	if len(manager.clients) > 0 {
		log.Println("Dealer WebSocket Manager: Event loop terminated")
	}
}

// 清理所有連接
func (manager *Manager) cleanupAllConnections() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	clientCount := len(manager.clients)
	// 只在有連接需要清理時才輸出日誌
	if clientCount > 0 {
		log.Printf("Dealer WebSocket Manager: Cleaning up all %d connections", clientCount)
	}

	for client := range manager.clients {
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
	manager.clients = make(map[*Client]bool)
	manager.authClients = make(map[uint][]*Client)
	manager.userClients = make(map[uint]map[string]*Client)
	manager.directBroadcast = make(map[uint]chan []byte)

	// 重置連接計數
	if manager.wsHandler != nil {
		atomic.StoreInt64(&manager.wsHandler.connections, 0)
		// 只有在之前有連接時才輸出
		if clientCount > 0 {
			log.Printf("Dealer WebSocket Handler: Reset active connections to 0")
		}
	}
}

// 移除指定客戶端
func (manager *Manager) removeClient(client *Client) {
	if client == nil {
		return
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	// 檢查客戶端是否存在
	if _, exists := manager.clients[client]; !exists {
		return // 客戶端已經被移除
	}

	// 刪除客戶端
	delete(manager.clients, client)

	// 降低活躍連接計數
	if manager.wsHandler != nil {
		currentConnections := atomic.AddInt64(&manager.wsHandler.connections, -1)
		// 只在連接數較少或者每10個連接減少時輸出日誌
		if currentConnections < 10 || currentConnections%10 == 0 {
			log.Printf("Dealer WebSocket Handler: Decreased active connections to %d", currentConnections)
		}
	}

	// 刪除認證客戶端
	if client.UserID > 0 {
		// 從用戶ID映射中刪除
		if clients, exists := manager.authClients[client.UserID]; exists {
			newClients := make([]*Client, 0, len(clients)-1)
			for _, c := range clients {
				if c != client {
					newClients = append(newClients, c)
				}
			}

			if len(newClients) > 0 {
				manager.authClients[client.UserID] = newClients
			} else {
				delete(manager.authClients, client.UserID)
			}
		}

		// 從用戶ID和客戶端ID映射中刪除
		if clientsMap, exists := manager.userClients[client.UserID]; exists {
			if _, exists := clientsMap[client.ID]; exists {
				delete(clientsMap, client.ID)
			}

			if len(clientsMap) == 0 {
				delete(manager.userClients, client.UserID)
			}
		}
	}

	// 停止心跳協程和關閉通道
	if client.heartbeatTicker != nil {
		client.heartbeatTicker.Stop()
	}
	if client.closeChan != nil {
		close(client.closeChan)
	}

	// 關閉發送通道
	close(client.Send)

	// 只有在連接非正常關閉時才輸出日誌
	if client.closeReason != "" && client.closeReason != "normal closure" {
		log.Printf("Dealer WebSocket Manager: Client %s removed: %s", client.ID, client.closeReason)
	}
}

// 廣播消息到所有客戶端
func (manager *Manager) broadcastMessage(message []byte) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	failedClients := make([]*Client, 0)

	for client := range manager.clients {
		select {
		case client.Send <- message:
			// 消息已送入通道 - 不記錄日誌
		default:
			// 發送通道已滿或已關閉，記錄待移除的客戶端
			failedClients = append(failedClients, client)
		}
	}

	// 如果有發送失敗的客戶端，解鎖後移除它們
	if len(failedClients) > 0 {
		manager.mutex.RUnlock()
		manager.mutex.Lock()

		// 只記錄總體數量而不是每個客戶端
		if len(failedClients) > 0 {
			log.Printf("Dealer WebSocket Manager: Removing %d clients due to full send buffer", len(failedClients))
		}

		for _, client := range failedClients {
			// 設置關閉原因
			client.closeReason = "broadcast buffer full"

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
				// 從 authClients 映射中移除
				clients := manager.authClients[client.UserID]
				for i, c := range clients {
					if c == client {
						// 從切片中移除元素
						clients = append(clients[:i], clients[i+1:]...)
						break
					}
				}

				// 更新或刪除映射
				if len(clients) == 0 {
					delete(manager.authClients, client.UserID)
				} else {
					manager.authClients[client.UserID] = clients
				}

				// 從 userClients 映射中移除
				if clientsMap, exists := manager.userClients[client.UserID]; exists {
					delete(clientsMap, client.ID)
					if len(clientsMap) == 0 {
						delete(manager.userClients, client.UserID)
					}
				}
			}

			// 關閉發送通道
			close(client.Send)

			// 從客戶端列表中刪除
			delete(manager.clients, client)

			// 減少連接計數
			if manager.wsHandler != nil {
				atomic.AddInt64(&manager.wsHandler.connections, -1)
			}
		}

		// 只在有較多客戶端被移除時記錄連接數
		if len(failedClients) >= 5 && manager.wsHandler != nil {
			log.Printf("Dealer WebSocket Handler: Active connections after broadcast cleanup: %d",
				atomic.LoadInt64(&manager.wsHandler.connections))
		}

		manager.mutex.Unlock()
		manager.mutex.RLock()
	}
}

// 清理非活躍連接
func (manager *Manager) cleanupInactiveConnections() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	now := time.Now()
	inactiveCount := 0

	for client := range manager.clients {
		// 檢查最後活動時間
		if now.Sub(client.LastActivity) > inactivityTimeout {
			inactiveCount++

			// 關閉非活躍客戶端
			client.closeReason = "inactivity timeout"
			delete(manager.clients, client)

			// 降低活躍連接計數
			if manager.wsHandler != nil {
				atomic.AddInt64(&manager.wsHandler.connections, -1)
			}

			// 停止心跳
			if client.heartbeatTicker != nil {
				client.heartbeatTicker.Stop()
			}
			if client.closeChan != nil {
				close(client.closeChan)
			}

			// 關閉發送通道
			close(client.Send)

			// 移除從認證客戶端映射中
			if client.UserID > 0 {
				// 從用戶ID映射中刪除
				if clients, exists := manager.authClients[client.UserID]; exists {
					newClients := make([]*Client, 0, len(clients)-1)
					for _, c := range clients {
						if c != client {
							newClients = append(newClients, c)
						}
					}

					if len(newClients) > 0 {
						manager.authClients[client.UserID] = newClients
					} else {
						delete(manager.authClients, client.UserID)
					}
				}

				// 從用戶ID和客戶端ID映射中刪除
				if clientsMap, exists := manager.userClients[client.UserID]; exists {
					if _, exists := clientsMap[client.ID]; exists {
						delete(clientsMap, client.ID)
					}

					if len(clientsMap) == 0 {
						delete(manager.userClients, client.UserID)
					}
				}
			}
		}
	}

	// 只有在實際清理了連接時才輸出日誌
	if inactiveCount > 0 {
		log.Printf("Dealer WebSocket Manager: Cleaned up %d inactive connections", inactiveCount)
		if manager.wsHandler != nil {
			log.Printf("Dealer WebSocket Handler: Active connections after cleanup: %d", atomic.LoadInt64(&manager.wsHandler.connections))
		}
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
		manager.userClients[userID] = make(map[string]*Client)
	}
	manager.userClients[userID][client.ID] = client

	// 添加到 authClients 映射
	manager.authClients[userID] = append(manager.authClients[userID], client)

	log.Printf("Dealer WebSocket Manager: Client %s authenticated for user %d\n", client.ID, userID)
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

	failedClients := 0 // 記錄失敗客戶端數量而不是每個都輸出日誌
	totalClients := 0  // 總客戶端數量

	// 發送訊息給用戶的所有客戶端
	for _, client := range clientMap {
		if client.IsAuthed {
			totalClients++
			select {
			case client.Send <- msgBytes:
				// 訊息已送入通道 - 不記錄日誌
			default:
				// 發送通道已滿或已關閉，移除客戶端
				failedClients++
				client.closeReason = "send buffer full"

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
				delete(manager.userClients[userID], client.ID)
				if len(manager.userClients[userID]) == 0 {
					delete(manager.userClients, userID)
				}

				// 移除客戶端
				close(client.Send)
				delete(manager.clients, client)

				// 減少連接計數
				if manager.wsHandler != nil {
					atomic.AddInt64(&manager.wsHandler.connections, -1)
				}
			}
		}
	}

	// 只在有失敗的客戶端時輸出一條匯總日誌
	if failedClients > 0 {
		log.Printf("Dealer WebSocket Manager: Failed to send to %d/%d clients for user %d",
			failedClients, totalClients, userID)
	}

	return nil
}

// ReadPump 從websocket連接中讀取資料
func (c *Client) ReadPump() {
	defer func() {
		c.manager.unregister <- c
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { _ = c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// 如果設置了處理連接的消息處理器，則調用它
	if c.manager.messageHandler != nil {
		c.manager.messageHandler.HandleConnect(c)
	}

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket連接錯誤: %v", err)
			}
			// 如果設置了處理斷開連接的消息處理器，則調用它
			if c.manager.messageHandler != nil {
				c.manager.messageHandler.HandleDisconnect(c)
			}
			break
		}

		// 嘗試將消息解析為標準命令格式
		var cmd struct {
			Type      string          `json:"type"`
			Data      json.RawMessage `json:"data"`
<<<<<<< Updated upstream
			Timestamp int64           `json:"timestamp"`
=======
			Timestamp string          `json:"timestamp"`
>>>>>>> Stashed changes
		}

		if err := json.Unmarshal(message, &cmd); err == nil && cmd.Type != "" {
			// 消息是標準命令格式

			// 特別處理心跳消息
			if cmd.Type == "heartbeat" {
				// 回應心跳消息
				heartbeatResponse := []byte(`{"type":"heartbeat_response","data":{},"timestamp":` + strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10) + `}`)

				// 不阻塞地發送心跳響應
				select {
				case c.Send <- heartbeatResponse:
					// 心跳響應已發送
				default:
					// 發送通道已滿，這種情況下只記錄一次
					if !c.heartbeatErrorLogged {
						log.Printf("客戶端 %s 發送緩衝區已滿，無法發送心跳響應", c.ID)
						c.heartbeatErrorLogged = true
					}
				}
				continue
			}

			// 特別處理基準測試消息
			if cmd.Type == "benchmark" {
				// 回應基準測試消息
				benchmarkResponse := []byte(`{"type":"benchmark_response","data":{},"timestamp":` + strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10) + `}`)

				// 不阻塞地發送基準測試響應
				select {
				case c.Send <- benchmarkResponse:
					// 基準測試響應已發送
				default:
					// 發送通道已滿，記錄錯誤
					log.Printf("客戶端 %s 發送緩衝區已滿，無法發送基準測試響應", c.ID)
				}
				continue
			}

			// 處理業務命令消息
			log.Printf("處理業務命令: %s", cmd.Type)

			// 如果設置了消息處理器，使用它處理消息
			if c.manager.messageHandler != nil {
				var data interface{}
				if len(cmd.Data) > 0 {
					if err := json.Unmarshal(cmd.Data, &data); err != nil {
						log.Printf("解析命令數據失敗: %v", err)
						// 如果解析失敗，傳遞原始數據
						data = cmd.Data
					}
				}

				c.manager.messageHandler.HandleMessage(c, cmd.Type, data)
			} else {
				log.Printf("未設置消息處理器，無法處理消息: %s", cmd.Type)
			}

			continue
		}

		// 如果不是標準命令格式，則處理為普通消息
		log.Printf("收到普通消息（非標準命令格式）: %s", message)

		msg := Message{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("解析消息失敗: %v", err)
			continue
		}

		// 如果是心跳消息，則不記錄
		if msg.Type == "heartbeat" {
			// 回應心跳消息
			response := Message{
				Type: "heartbeat_response",
				Data: map[string]interface{}{},
			}
			responseJSON, _ := json.Marshal(response)

			// 不阻塞地發送心跳響應
			select {
			case c.Send <- responseJSON:
				// 心跳響應已發送
			default:
				// 發送通道已滿，這種情況下只記錄一次
				if !c.heartbeatErrorLogged {
					log.Printf("客戶端 %s 發送緩衝區已滿，無法發送心跳響應", c.ID)
					c.heartbeatErrorLogged = true
				}
			}
		} else if msg.Type == "benchmark" {
			// 回應基準測試消息
			response := Message{
				Type: "benchmark_response",
				Data: map[string]interface{}{},
			}
			responseJSON, _ := json.Marshal(response)

			// 不阻塞地發送基準測試響應
			select {
			case c.Send <- responseJSON:
				// 基準測試響應已發送
			default:
				// 發送通道已滿，記錄錯誤
				log.Printf("客戶端 %s 發送緩衝區已滿，無法發送基準測試響應", c.ID)
			}
		} else {
			// 記錄重要消息
			log.Printf("收到消息: %v", msg)
<<<<<<< Updated upstream
=======

			// 處理特殊消息類型
			// 對於BETTING_STARTED消息，確保它能被正確處理
			if msg.Type == "BETTING_STARTED" && c.manager.messageHandler != nil {
				log.Printf("檢測到BETTING_STARTED消息，轉發給消息處理器")
				// 調用消息處理器處理BETTING_STARTED消息
				c.manager.messageHandler.HandleMessage(c, msg.Type, msg.Data)
			}
>>>>>>> Stashed changes
		}
	}
}

// 客戶端寫入訊息
func (client *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道已關閉
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				// 只在連接已關閉以外的錯誤情況下記錄日誌
				if !strings.Contains(err.Error(), "closed") &&
					!strings.Contains(err.Error(), "broken pipe") {
					log.Printf("Dealer WebSocket Error (NextWriter): %v", err)
				}
				return
			}
			w.Write(message)

			// 添加隊列中的所有消息
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				// 只在連接已關閉以外的錯誤情況下記錄日誌
				if !strings.Contains(err.Error(), "closed") &&
					!strings.Contains(err.Error(), "broken pipe") {
					log.Printf("Dealer WebSocket Error (Close Writer): %v", err)
				}
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// 只在連接已關閉以外的錯誤情況下記錄日誌
				if !strings.Contains(err.Error(), "closed") &&
					!strings.Contains(err.Error(), "broken pipe") {
					log.Printf("Dealer WebSocket Error (Ping): %v", err)
				}
				return
			}
		case <-client.closeChan:
			return
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
	// 不再輸出每個客戶端的心跳啟動日誌

	// 創建本地副本
	localCloseChan := client.closeChan
	heartbeatTicker := client.heartbeatTicker

	// 啟動心跳協程
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Dealer WebSocket Manager: Heartbeat routine for client %s recovered from panic: %v", client.ID, r)
			}
			// 不再輸出心跳協程退出的日誌
		}()

		for {
			select {
			case <-localCloseChan:
				// 不再輸出關閉信號的日誌
				return
			case <-heartbeatTicker.C:
				// 使用本地副本確保即使 client 被修改也能正確工作
				if err := client.sendPing(); err != nil {
					// 只在異常錯誤時輸出日誌
					if !strings.Contains(err.Error(), "connection is nil") &&
						!strings.Contains(err.Error(), "use of closed network connection") {
						log.Printf("Dealer WebSocket Manager: Client %s ping failed: %v", client.ID, err)
					}
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

	// 嘗試3次發送Ping消息
	var pingErr error
	for i := 0; i < 3; i++ {
		// 發送Ping訊息
		pingErr = client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeTimeout))
		if pingErr == nil {
			break
		}
		// 只在第三次嘗試失敗時記錄日誌
		if i == 2 {
			log.Printf("Dealer WebSocket Manager: Client %s all ping attempts failed: %v", client.ID, pingErr)
		}
		time.Sleep(100 * time.Millisecond) // 短暫延遲後重試
	}

	if pingErr != nil {
		// 直接調用 attemptReconnect (已在 goroutine 中)
		client.connMutex.Unlock() // 先解鎖以避免死鎖
		client.attemptReconnect() // 直接調用而不是啟動新的 goroutine
		client.connMutex.Lock()   // 重新鎖定
		return pingErr
	}

	// 發送應用層心跳訊息
	heartbeat := HeartbeatMessage{
		Type:      "heartbeat",
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}
	heartbeatBytes, _ := json.Marshal(heartbeat)

	select {
	case client.Send <- heartbeatBytes:
		// 心跳已送入通道，不輸出日誌
	default:
		// 僅在通道已滿時輸出警告
		log.Printf("Dealer WebSocket Manager: Client %s send channel full, cannot send heartbeat", client.ID)
	}

	return nil
}

// 嘗試重連
func (client *Client) attemptReconnect() {
	// 設置關閉原因
	client.closeReason = "connection terminated"

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
	// 檢查是否有活躍連接，只有在有連接時才記錄關閉訊息
	manager.mutex.RLock()
	clientCount := len(manager.clients)
	manager.mutex.RUnlock()

	if clientCount > 0 {
		log.Println("Dealer WebSocket Manager: Shutdown initiated, closing all connections...")
	}

	// 通知所有客戶端關閉
	manager.mutex.Lock()

	for client := range manager.clients {
		// 只有在有大量連接時才詳細記錄每個客戶端的關閉
		if clientCount < 10 {
			log.Printf("Dealer WebSocket Manager: Closing connection for client %s", client.ID)
		}

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
	manager.clients = make(map[*Client]bool)
	manager.authClients = make(map[uint][]*Client)
	manager.userClients = make(map[uint]map[string]*Client)
	manager.mutex.Unlock()

	// 重置連接計數
	if manager.wsHandler != nil {
		atomic.StoreInt64(&manager.wsHandler.connections, 0)
		if clientCount > 0 {
			log.Printf("Dealer WebSocket Handler: Reset active connections to 0")
		}
	}

	// 關閉管理器的shutdown通道
	close(manager.shutdown)

	// 只有在實際關閉了連接時才輸出
	if clientCount > 0 {
		log.Printf("Dealer WebSocket Manager: Successfully closed %d connections", clientCount)
	}
}

// 廣播消息到特定房間
func (manager *Manager) broadcastToRoom(roomID string, message []byte) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	failedCount := 0
	totalClients := 0

	// 檢查房間是否存在
	clients, exists := manager.roomClients[roomID]
	if !exists {
		// 房間不存在，沒有客戶端可廣播
		return
	}

	// 廣播消息到房間中的所有客戶端
	for client := range clients {
		totalClients++
		select {
		case client.Send <- message:
			// 消息發送成功 - 不記錄日誌
		default:
			// 客戶端發送通道已滿，跳過並計數
			failedCount++
		}
	}

	// 只在有發送失敗時記錄日誌
	if failedCount > 0 {
		log.Printf("Dealer WebSocket Manager: Failed to send to %d/%d clients in room %s",
			failedCount, totalClients, roomID)
	}
}

// 廣播消息到特定遊戲
func (manager *Manager) broadcastToGame(gameID string, message []byte) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	failedCount := 0
	totalClients := 0

	// 檢查遊戲是否存在
	clients, exists := manager.gameClients[gameID]
	if !exists {
		// 遊戲不存在，沒有客戶端可廣播
		return
	}

	// 廣播消息到遊戲中的所有客戶端
	for client := range clients {
		totalClients++
		select {
		case client.Send <- message:
			// 消息發送成功 - 不記錄日誌
		default:
			// 客戶端發送通道已滿，跳過並計數
			failedCount++
		}
	}

	// 只在有發送失敗時記錄日誌
	if failedCount > 0 {
		log.Printf("Dealer WebSocket Manager: Failed to send to %d/%d clients in game %s",
			failedCount, totalClients, gameID)
	}
}

// AddClientToRoom 將客戶端添加到指定房間
func (m *Manager) AddClientToRoom(client *Client, roomID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 檢查此客戶端是否已在該房間中
	for _, id := range client.rooms {
		if id == roomID {
			return // 客戶端已在房間中，不需再添加
		}
	}

	// 若房間不存在，則創建它
	if _, exists := m.roomClients[roomID]; !exists {
		m.roomClients[roomID] = make(map[*Client]bool)
	}

	// 將客戶端加入到房間
	m.roomClients[roomID][client] = true
	client.rooms = append(client.rooms, roomID)

	log.Printf("Dealer WebSocket Manager: 客戶端 %s 已加入房間 %s", client.ID, roomID)
}

// RemoveClientFromRoom 將客戶端從指定房間移除
func (m *Manager) RemoveClientFromRoom(client *Client, roomID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 檢查房間是否存在
	clients, exists := m.roomClients[roomID]
	if !exists {
		return // 房間不存在，不需處理
	}

	// 從房間中移除客戶端
	delete(clients, client)

	// 若房間為空，則移除該房間
	if len(clients) == 0 {
		delete(m.roomClients, roomID)
	}

	// 從客戶端的房間列表中移除
	for i, id := range client.rooms {
		if id == roomID {
			client.rooms = append(client.rooms[:i], client.rooms[i+1:]...)
			break
		}
	}

	log.Printf("Dealer WebSocket Manager: 客戶端 %s 已離開房間 %s", client.ID, roomID)
}

// AddClientToGame 將客戶端添加到指定遊戲
func (m *Manager) AddClientToGame(client *Client, gameID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 檢查此客戶端是否已在該遊戲中
	for _, id := range client.games {
		if id == gameID {
			return // 客戶端已在遊戲中，不需再添加
		}
	}

	// 若遊戲不存在，則創建它
	if _, exists := m.gameClients[gameID]; !exists {
		m.gameClients[gameID] = make(map[*Client]bool)
	}

	// 將客戶端加入到遊戲
	m.gameClients[gameID][client] = true
	client.games = append(client.games, gameID)

	log.Printf("Dealer WebSocket Manager: 客戶端 %s 已加入遊戲 %s", client.ID, gameID)
}

// RemoveClientFromGame 將客戶端從指定遊戲移除
func (m *Manager) RemoveClientFromGame(client *Client, gameID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 檢查遊戲是否存在
	clients, exists := m.gameClients[gameID]
	if !exists {
		return // 遊戲不存在，不需處理
	}

	// 從遊戲中移除客戶端
	delete(clients, client)

	// 若遊戲為空，則移除該遊戲
	if len(clients) == 0 {
		delete(m.gameClients, gameID)
	}

	// 從客戶端的遊戲列表中移除
	for i, id := range client.games {
		if id == gameID {
			client.games = append(client.games[:i], client.games[i+1:]...)
			break
		}
	}

	log.Printf("Dealer WebSocket Manager: 客戶端 %s 已離開遊戲 %s", client.ID, gameID)
}

// BroadcastToRoom 向特定房間內所有客戶端廣播消息
func (m *Manager) BroadcastToRoom(roomID string, message []byte) {
	m.mutex.RLock()
	clients, exists := m.roomClients[roomID]
	m.mutex.RUnlock()

	if !exists {
		// 只有當房間存在但沒有客戶端時才記錄
		log.Printf("Dealer WebSocket Manager: 嘗試廣播到不存在的房間: %s", roomID)
		return
	}

	failCount := 0
	for client := range clients {
		if client.Send == nil || len(client.Send) >= cap(client.Send) {
			failCount++
			continue
		}

		select {
		case client.Send <- message:
			// 成功發送，不需處理
		default:
			// 發送緩衝區已滿，移除客戶端
			failCount++
			m.removeClientFromRooms(client)
			m.removeClientFromGames(client)
			m.removeClient(client)
		}
	}

	// 只有當有失敗時才記錄
	if failCount > 0 {
		log.Printf("Dealer WebSocket Manager: 向房間 %s 廣播時，有 %d 個客戶端發送失敗", roomID, failCount)
	}
}

// BroadcastToGame 向特定遊戲內所有客戶端廣播消息
func (m *Manager) BroadcastToGame(gameID string, message []byte) {
	m.mutex.RLock()
	clients, exists := m.gameClients[gameID]
	m.mutex.RUnlock()

	if !exists {
		// 只有當遊戲存在但沒有客戶端時才記錄
		log.Printf("Dealer WebSocket Manager: 嘗試廣播到不存在的遊戲: %s", gameID)
		return
	}

	failCount := 0
	for client := range clients {
		if client.Send == nil || len(client.Send) >= cap(client.Send) {
			failCount++
			continue
		}

		select {
		case client.Send <- message:
			// 成功發送，不需處理
		default:
			// 發送緩衝區已滿，移除客戶端
			failCount++
			m.removeClientFromRooms(client)
			m.removeClientFromGames(client)
			m.removeClient(client)
		}
	}

	// 只有當有失敗時才記錄
	if failCount > 0 {
		log.Printf("Dealer WebSocket Manager: 向遊戲 %s 廣播時，有 %d 個客戶端發送失敗", gameID, failCount)
	}
}

// removeClientFromRooms 從所有房間中移除客戶端
func (m *Manager) removeClientFromRooms(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, roomID := range client.rooms {
		if clients, exists := m.roomClients[roomID]; exists {
			delete(clients, client)
			if len(clients) == 0 {
				delete(m.roomClients, roomID)
			}
		}
	}
	client.rooms = []string{}
}

// removeClientFromGames 從所有遊戲中移除客戶端
func (m *Manager) removeClientFromGames(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, gameID := range client.games {
		if clients, exists := m.gameClients[gameID]; exists {
			delete(clients, client)
			if len(clients) == 0 {
				delete(m.gameClients, gameID)
			}
		}
	}
	client.games = []string{}
}

// SetMessageHandler 設置消息處理器
func (manager *Manager) SetMessageHandler(handler MessageHandler) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.messageHandler = handler
	log.Printf("Dealer WebSocket Manager: Message handler has been set")
}
