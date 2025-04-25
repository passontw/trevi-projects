package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允許所有來源的連接，生產環境應該限制
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Manager 管理 WebSocket 連接
type Manager struct {
	// 註冊的客戶端
	clients map[*Client]bool

	// 廣播消息通道
	broadcast chan []byte

	// 註冊請求
	register chan *Client

	// 取消註冊請求
	unregister chan *Client

	// 互斥鎖，保護資源
	mutex sync.Mutex
}

// NewManager 創建一個新的管理器
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Start 啟動管理器
func (m *Manager) Start(ctx context.Context) {
	log.Println("WebSocket Manager started")

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，關閉所有連接
			m.mutex.Lock()
			for client := range m.clients {
				close(client.send)
				delete(m.clients, client)
			}
			m.mutex.Unlock()
			return

		case client := <-m.register:
			// 註冊新客戶端
			m.mutex.Lock()
			m.clients[client] = true
			m.mutex.Unlock()
			// 發送歡迎消息
			client.send <- []byte("Hello World! Welcome to WebSocket Server")
			log.Println("New client connected")

		case client := <-m.unregister:
			// 取消註冊客戶端
			m.mutex.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.send)
				log.Println("Client disconnected")
			}
			m.mutex.Unlock()

		case message := <-m.broadcast:
			// 廣播消息給所有客戶端
			m.mutex.Lock()
			for client := range m.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(m.clients, client)
				}
			}
			m.mutex.Unlock()
		}
	}
}

// Shutdown 關閉管理器
func (m *Manager) Shutdown() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for client := range m.clients {
		close(client.send)
	}

	log.Println("WebSocket Manager shutting down")
}

// ServeWs 處理 WebSocket 請求
func (m *Manager) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(m, conn)
	m.register <- client

	// 啟動客戶端的讀寫協程
	go client.writePump()
	go client.readPump()
}

// BroadcastMessage 廣播消息給所有客戶端
func (m *Manager) BroadcastMessage(message []byte) {
	m.broadcast <- message
}
