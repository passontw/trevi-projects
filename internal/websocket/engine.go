package websocket

import (
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

// Client 代表一個 WebSocket 客戶端連線
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Engine   *Engine
	Send     chan []byte
	mu       sync.Mutex
	closed   bool
	metadata map[string]interface{}
}

// Engine 是 WebSocket 連線管理引擎
type Engine struct {
	Clients       map[string]*Client
	Register      chan *Client
	Unregister    chan *Client
	BroadcastChan chan []byte
	Logger        *zap.Logger
	mu            sync.RWMutex
	handlers      map[string]MessageHandler
	onConnect     func(*Client) // 客戶端連接時的回調函數
}

// Message 結構用於解析接收到的 JSON 訊息
type Message struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// Response 結構用於發送 JSON 回應
type Response struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// MessageHandler 是處理特定類型訊息的函數類型
type MessageHandler func(client *Client, message Message) error

// NewEngine 創建新的 WebSocket 引擎
func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{
		Clients:       make(map[string]*Client),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		BroadcastChan: make(chan []byte),
		Logger:        logger,
		handlers:      make(map[string]MessageHandler),
		onConnect:     nil, // 默認沒有連接回調
	}
}

// Start 啟動 WebSocket 引擎的主要運行迴圈
func (e *Engine) Start() {
	go func() {
		for {
			select {
			case client := <-e.Register:
				e.mu.Lock()
				e.Clients[client.ID] = client
				e.mu.Unlock()
				e.Logger.Info("Client registered", zap.String("id", client.ID))

			case client := <-e.Unregister:
				e.mu.Lock()
				if _, ok := e.Clients[client.ID]; ok {
					delete(e.Clients, client.ID)
					close(client.Send)
				}
				e.mu.Unlock()
				e.Logger.Info("Client unregistered", zap.String("id", client.ID))

			case message := <-e.BroadcastChan:
				e.mu.RLock()
				for id, client := range e.Clients {
					select {
					case client.Send <- message:
					default:
						e.mu.RUnlock()
						e.mu.Lock()
						delete(e.Clients, id)
						close(client.Send)
						e.mu.Unlock()
						e.mu.RLock()
					}
				}
				e.mu.RUnlock()
			}
		}
	}()
}

// HandleConnection 處理新的 WebSocket 連線
func (e *Engine) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		e.Logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}

	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
	client := &Client{
		ID:       clientID,
		Conn:     conn,
		Engine:   e,
		Send:     make(chan []byte, 256),
		metadata: make(map[string]interface{}),
	}

	e.Register <- client

	// 設定讀取訊息的超時時間和 Pong 處理
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 如果有設置連接回調，則調用它
	if e.onConnect != nil {
		e.onConnect(client)
	}

	// 啟動 goroutine 處理讀取和寫入
	go client.writePump()
	go client.readPump()
}

// RegisterHandler 註冊訊息類型的處理函數
func (e *Engine) RegisterHandler(messageType string, handler MessageHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[messageType] = handler
}

// GetHandler 獲取指定訊息類型的處理函數
func (e *Engine) GetHandler(messageType string) (MessageHandler, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	handler, ok := e.handlers[messageType]
	return handler, ok
}

// SetMetadata 為客戶端設定元數據
func (c *Client) SetMetadata(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metadata[key] = value
}

// GetMetadata 獲取客戶端元數據
func (c *Client) GetMetadata(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.metadata[key]
	return value, ok
}

// Close 關閉客戶端連線
func (c *Client) Close() {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		c.Conn.Close()
		c.Engine.Unregister <- c
	}
	c.mu.Unlock()
}

// readPump 負責從 WebSocket 讀取訊息
func (c *Client) readPump() {
	defer c.Close()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Engine.Logger.Error("WebSocket read error",
					zap.String("clientID", c.ID),
					zap.Error(err))
			}
			break
		}

		c.Engine.Logger.Debug("Received message",
			zap.String("clientID", c.ID),
			zap.ByteString("message", message))

		// 處理收到的訊息
		c.handleMessage(message)
	}
}

// writePump 負責向 WebSocket 發送訊息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已關閉
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加等待中的訊息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 處理接收到的訊息
func (c *Client) handleMessage(message []byte) {
	var msg Message
	if err := c.Engine.parseMessage(message, &msg); err != nil {
		c.Engine.Logger.Error("Failed to parse message",
			zap.String("clientID", c.ID),
			zap.Error(err))
		return
	}

	// 查找並調用對應的處理函數
	if handler, ok := c.Engine.GetHandler(msg.Type); ok {
		if err := handler(c, msg); err != nil {
			c.Engine.Logger.Error("Error handling message",
				zap.String("type", msg.Type),
				zap.String("clientID", c.ID),
				zap.Error(err))
		}
	} else {
		c.Engine.Logger.Warn("No handler for message type",
			zap.String("type", msg.Type),
			zap.String("clientID", c.ID))
	}
}

// parseMessage 解析 JSON 訊息
func (e *Engine) parseMessage(data []byte, message interface{}) error {
	if err := json.Unmarshal(data, message); err != nil {
		return fmt.Errorf("parse message error: %w", err)
	}
	return nil
}

// SendJSON 發送 JSON 訊息給客戶端
func (c *Client) SendJSON(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal JSON error: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client connection closed")
	}

	c.Send <- data
	return nil
}

// Broadcast 向所有客戶端廣播訊息
func (e *Engine) Broadcast(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal JSON error: %w", err)
	}

	e.BroadcastChan <- data
	return nil
}

// BroadcastFilter 向符合篩選條件的客戶端廣播訊息
func (e *Engine) BroadcastFilter(message interface{}, filter func(*Client) bool) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal JSON error: %w", err)
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, client := range e.Clients {
		if filter(client) {
			client.Send <- data
		}
	}
	return nil
}

// SetOnConnectHandler 設置客戶端連接時的回調函數
func (e *Engine) SetOnConnectHandler(handler func(*Client)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onConnect = handler
}
