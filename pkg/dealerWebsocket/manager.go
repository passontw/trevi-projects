package dealerWebsocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// 定義 WebSocket 升級器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允許所有來源的連線 (生產環境應該設定為特定域名)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// TokenValidator 是一個用於驗證 Token 的函數類型
type TokenValidator func(token string) (uint, error)

// Client 代表一個 WebSocket 客戶端連線
type Client struct {
	ID        string
	conn      *websocket.Conn
	manager   *Manager
	send      chan []byte
	userData  map[string]interface{}
	userID    uint
	mu        sync.Mutex
	closed    bool
	isDealer  bool
	topicSubs map[string]bool // 訂閱的主題
}

// Message 結構用於解析接收到的 JSON 訊息
type Message struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// Response 結構用於發送 JSON 回應
type Response struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// MessageHandler 是處理特定類型訊息的介面
type MessageHandler interface {
	Handle(client *Client, message Message) error
}

// Manager 是 WebSocket 連線管理器
type Manager struct {
	clients          map[string]*Client
	dealerClients    map[string]*Client
	playerClients    map[string]*Client
	register         chan *Client
	unregister       chan *Client
	broadcast        chan []byte
	logger           *zap.Logger
	mu               sync.RWMutex
	tokenValidator   TokenValidator
	messageHandler   MessageHandler
	topics           map[string]map[string]*Client // 每個主題的訂閱者
	topicsMu         sync.RWMutex
	shutdownCh       chan struct{}
	shutdownComplete chan struct{}
}

// NewManager 創建新的 WebSocket 管理器
func NewManager(tokenValidator TokenValidator) *Manager {
	return &Manager{
		clients:          make(map[string]*Client),
		dealerClients:    make(map[string]*Client),
		playerClients:    make(map[string]*Client),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		broadcast:        make(chan []byte),
		logger:           zap.L().Named("websocket_manager"),
		tokenValidator:   tokenValidator,
		messageHandler:   nil, // 需要外部設置
		topics:           make(map[string]map[string]*Client),
		shutdownCh:       make(chan struct{}),
		shutdownComplete: make(chan struct{}),
	}
}

// SetMessageHandler 設置訊息處理器
func (m *Manager) SetMessageHandler(handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageHandler = handler
}

// Start 啟動 WebSocket 管理器的主要運行迴圈
func (m *Manager) Start(ctx context.Context) {
	m.logger.Info("Starting WebSocket manager")
	defer close(m.shutdownComplete)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Context done, stopping WebSocket manager")
			return
		case <-m.shutdownCh:
			m.logger.Info("Shutdown signal received, stopping WebSocket manager")
			return
		case client := <-m.register:
			m.registerClient(client)
		case client := <-m.unregister:
			m.unregisterClient(client)
		case message := <-m.broadcast:
			m.broadcastMessage(message)
		}
	}
}

// Shutdown 關閉 WebSocket 管理器
func (m *Manager) Shutdown() {
	m.logger.Info("Shutting down WebSocket manager")
	close(m.shutdownCh)
	<-m.shutdownComplete
	m.logger.Info("WebSocket manager stopped")
}

// HandleConnection 處理新的 WebSocket 連線
func (m *Manager) HandleConnection(w http.ResponseWriter, r *http.Request, isDealer bool) {
	// 從查詢參數或頭部獲取 token
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
	}

	// 驗證 token
	userID, err := m.tokenValidator(token)
	if err != nil {
		m.logger.Warn("Token validation failed", zap.Error(err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 升級連線為 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// 建立客戶端
	clientID := fmt.Sprintf("%d-%s", userID, time.Now().Format(time.RFC3339Nano))
	client := &Client{
		ID:        clientID,
		conn:      conn,
		manager:   m,
		send:      make(chan []byte, 256),
		userData:  make(map[string]interface{}),
		userID:    userID,
		isDealer:  isDealer,
		topicSubs: make(map[string]bool),
	}

	// 註冊客戶端
	m.register <- client

	// 啟動讀寫 goroutines
	go client.readPump()
	go client.writePump()
}

// registerClient 註冊新客戶端
func (m *Manager) registerClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client.ID] = client
	if client.isDealer {
		m.dealerClients[client.ID] = client
		m.logger.Info("Dealer client registered", zap.String("clientID", client.ID), zap.Uint("userID", client.userID))
	} else {
		m.playerClients[client.ID] = client
		m.logger.Info("Player client registered", zap.String("clientID", client.ID), zap.Uint("userID", client.userID))
	}

	// 發送歡迎訊息
	welcomeMsg := Response{
		Type: "connected",
		Payload: map[string]interface{}{
			"message":   "連線成功",
			"client_id": client.ID,
			"user_id":   client.userID,
			"is_dealer": client.isDealer,
			"time":      time.Now().Format(time.RFC3339),
		},
	}

	if err := client.SendJSON(welcomeMsg); err != nil {
		m.logger.Error("Failed to send welcome message", zap.String("clientID", client.ID), zap.Error(err))
	}
}

// unregisterClient 取消註冊客戶端
func (m *Manager) unregisterClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[client.ID]; ok {
		delete(m.clients, client.ID)
		if client.isDealer {
			delete(m.dealerClients, client.ID)
			m.logger.Info("Dealer client unregistered", zap.String("clientID", client.ID))
		} else {
			delete(m.playerClients, client.ID)
			m.logger.Info("Player client unregistered", zap.String("clientID", client.ID))
		}

		close(client.send)
	}

	// 從所有訂閱的主題中移除客戶端
	m.topicsMu.Lock()
	for topic := range client.topicSubs {
		if subscribers, exists := m.topics[topic]; exists {
			delete(subscribers, client.ID)
			if len(subscribers) == 0 {
				delete(m.topics, topic)
			}
		}
	}
	m.topicsMu.Unlock()

	m.logger.Info("Client unsubscribed from all topics", zap.String("clientID", client.ID))
}

// broadcastMessage 廣播訊息給所有客戶端
func (m *Manager) broadcastMessage(message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.clients {
		select {
		case client.send <- message:
		default:
			m.unregister <- client
		}
	}
}

// BroadcastToType 廣播訊息給特定類型的客戶端
func (m *Manager) BroadcastToType(message interface{}, isDealer bool) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var targetClients map[string]*Client
	if isDealer {
		targetClients = m.dealerClients
	} else {
		targetClients = m.playerClients
	}

	for _, client := range targetClients {
		select {
		case client.send <- data:
		default:
			// 如果客戶端的發送通道已滿，放棄此消息並計劃取消註冊該客戶端
			go func(c *Client) {
				m.unregister <- c
			}(client)
		}
	}

	return nil
}

// Subscribe 訂閱主題
func (m *Manager) Subscribe(client *Client, topic string) error {
	if topic == "" {
		return fmt.Errorf("empty topic")
	}

	m.topicsMu.Lock()
	defer m.topicsMu.Unlock()

	// 在主題映射中創建訂閱者列表（如果不存在）
	if _, exists := m.topics[topic]; !exists {
		m.topics[topic] = make(map[string]*Client)
	}

	// 添加客戶端到訂閱者列表
	m.topics[topic][client.ID] = client
	client.topicSubs[topic] = true

	m.logger.Info("Client subscribed to topic",
		zap.String("clientID", client.ID),
		zap.String("topic", topic))

	return nil
}

// Unsubscribe 取消訂閱主題
func (m *Manager) Unsubscribe(client *Client, topic string) error {
	if topic == "" {
		return fmt.Errorf("empty topic")
	}

	m.topicsMu.Lock()
	defer m.topicsMu.Unlock()

	// 檢查主題是否存在
	subscribers, exists := m.topics[topic]
	if !exists {
		return fmt.Errorf("topic not found")
	}

	// 從訂閱者列表中移除客戶端
	delete(subscribers, client.ID)
	delete(client.topicSubs, topic)

	// 如果主題沒有訂閱者，刪除主題
	if len(subscribers) == 0 {
		delete(m.topics, topic)
	}

	m.logger.Info("Client unsubscribed from topic",
		zap.String("clientID", client.ID),
		zap.String("topic", topic))

	return nil
}

// PublishToTopic 向主題發布訊息
func (m *Manager) PublishToTopic(topic string, message interface{}) error {
	if topic == "" {
		return fmt.Errorf("empty topic")
	}

	data, err := json.Marshal(Response{
		Type: "message",
		Payload: map[string]interface{}{
			"topic":     topic,
			"data":      message,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	m.topicsMu.RLock()
	subscribers, exists := m.topics[topic]
	m.topicsMu.RUnlock()

	if !exists || len(subscribers) == 0 {
		m.logger.Info("No subscribers for topic", zap.String("topic", topic))
		return nil
	}

	m.topicsMu.RLock()
	defer m.topicsMu.RUnlock()

	for _, client := range subscribers {
		select {
		case client.send <- data:
		default:
			// 如果客戶端的發送通道已滿，放棄此消息並計劃取消註冊該客戶端
			go func(c *Client) {
				m.unregister <- c
			}(client)
		}
	}

	m.logger.Info("Published message to topic",
		zap.String("topic", topic),
		zap.Int("subscribers", len(subscribers)))

	return nil
}

// readPump 處理從 WebSocket 讀取訊息
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096) // 設置最大訊息大小
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.manager.logger.Error("Unexpected close",
					zap.String("clientID", c.ID),
					zap.Error(err))
			}
			break
		}

		// 解析訊息
		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			c.manager.logger.Error("Failed to parse message",
				zap.String("clientID", c.ID),
				zap.Error(err))
			continue
		}

		// 處理特殊訊息類型
		switch message.Type {
		case "ping":
			// 回應 pong
			if err := c.SendJSON(Response{
				Type:    "pong",
				Payload: map[string]string{"time": time.Now().Format(time.RFC3339)},
			}); err != nil {
				c.manager.logger.Error("Failed to send pong",
					zap.String("clientID", c.ID),
					zap.Error(err))
			}
			continue
		case "subscribe":
			// 處理訂閱請求
			if data, ok := message.Payload["topic"].(string); ok {
				if err := c.manager.Subscribe(c, data); err != nil {
					c.manager.logger.Error("Failed to subscribe",
						zap.String("clientID", c.ID),
						zap.String("topic", data),
						zap.Error(err))
				} else {
					c.SendJSON(Response{
						Type: "subscribed",
						Payload: map[string]interface{}{
							"topic":     data,
							"success":   true,
							"timestamp": time.Now().Format(time.RFC3339),
						},
					})
				}
			}
			continue
		case "unsubscribe":
			// 處理取消訂閱請求
			if data, ok := message.Payload["topic"].(string); ok {
				if err := c.manager.Unsubscribe(c, data); err != nil {
					c.manager.logger.Error("Failed to unsubscribe",
						zap.String("clientID", c.ID),
						zap.String("topic", data),
						zap.Error(err))
				} else {
					c.SendJSON(Response{
						Type: "unsubscribed",
						Payload: map[string]interface{}{
							"topic":     data,
							"success":   true,
							"timestamp": time.Now().Format(time.RFC3339),
						},
					})
				}
			}
			continue
		case "publish":
			// 處理發布請求
			topic, hasTopic := message.Payload["topic"].(string)
			if !hasTopic || topic == "" {
				c.SendJSON(Response{
					Type: "error",
					Payload: map[string]interface{}{
						"message":   "發布請求中缺少主題",
						"timestamp": time.Now().Format(time.RFC3339),
					},
				})
				continue
			}

			data, hasData := message.Payload["data"]
			if !hasData {
				data = nil
			}

			if err := c.manager.PublishToTopic(topic, data); err != nil {
				c.manager.logger.Error("Failed to publish to topic",
					zap.String("clientID", c.ID),
					zap.String("topic", topic),
					zap.Error(err))
			} else {
				c.SendJSON(Response{
					Type: "published",
					Payload: map[string]interface{}{
						"topic":     topic,
						"success":   true,
						"timestamp": time.Now().Format(time.RFC3339),
					},
				})
			}
			continue
		}

		// 處理其他訊息類型
		c.manager.mu.RLock()
		handler := c.manager.messageHandler
		c.manager.mu.RUnlock()

		if handler != nil {
			if err := handler.Handle(c, message); err != nil {
				c.manager.logger.Error("Error handling message",
					zap.String("clientID", c.ID),
					zap.String("type", message.Type),
					zap.Error(err))
			}
		} else {
			c.manager.logger.Warn("No message handler configured",
				zap.String("clientID", c.ID),
				zap.String("type", message.Type))
		}
	}
}

// writePump 處理向 WebSocket 發送訊息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已關閉
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加排隊訊息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendJSON 向客戶端發送 JSON 訊息
func (c *Client) SendJSON(message interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client connection closed")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	c.send <- data
	return nil
}

// GetUserID 獲取用戶ID
func (c *Client) GetUserID() uint {
	return c.userID
}

// IsDealer 檢查客戶端是否為荷官
func (c *Client) IsDealer() bool {
	return c.isDealer
}

// SetUserData 設置用戶數據
func (c *Client) SetUserData(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userData[key] = value
}

// GetUserData 獲取用戶數據
func (c *Client) GetUserData(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	val, ok := c.userData[key]
	return val, ok
}

// Close 關閉客戶端連線
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		c.manager.unregister <- c
	}
}

// GetTopicSubscriberCount 獲取主題訂閱者數量
func (m *Manager) GetTopicSubscriberCount(topic string) int {
	m.topicsMu.RLock()
	defer m.topicsMu.RUnlock()

	if subscribers, exists := m.topics[topic]; exists {
		return len(subscribers)
	}
	return 0
}
