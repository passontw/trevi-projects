package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 向客戶端寫入消息的等待時間
	writeWait = 10 * time.Second

	// 讀取下一個 pong 消息的等待時間
	pongWait = 60 * time.Second

	// 發送 ping 消息的頻率
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512
)

// Client 是 WebSocket 連接的中間人
type Client struct {
	// WebSocket 連接
	conn *websocket.Conn

	// 發送消息的緩衝通道
	send chan []byte

	// 管理器
	manager *Manager
}

// NewClient 創建一個新的客戶端
func NewClient(manager *Manager, conn *websocket.Conn) *Client {
	return &Client{
		conn:    conn,
		send:    make(chan []byte, 256),
		manager: manager,
	}
}

// readPump 從 WebSocket 連接中讀取消息
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// 收到消息後，回覆一個 Hello World 消息
		c.manager.broadcast <- []byte("Hello from server: " + string(message))
	}
}

// writePump 將消息寫入 WebSocket 連接
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 管道關閉
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 將隊列中等待的消息添加到當前 WebSocket 消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
