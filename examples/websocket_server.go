package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:5500", "服務器地址")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允許所有來源的跨域請求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 表示客戶端連接的結構
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	subscribe SubscribeRequest
}

// 訂閱請求的結構
type SubscribeRequest struct {
	Type    string `json:"type"`
	Payload struct {
		Method string `json:"method"`
		Data   struct {
			EventTypes []string `json:"eventTypes"`
		} `json:"data"`
	} `json:"payload"`
}

// 消息的通用結構
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// 處理 WebSocket 連接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// 創建一個新的客戶端
	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	// 發送歡迎消息
	welcomeMsg := Message{
		Type: "hello",
		Payload: map[string]interface{}{
			"message": "Welcome to gRPC WebSocket Demo Server",
			"time":    time.Now().Format(time.RFC3339),
		},
	}

	welcomeBytes, _ := json.Marshal(welcomeMsg)
	client.send <- welcomeBytes

	// 啟動讀寫 goroutines
	go client.readPump()
	go client.writePump()
}

// 處理從客戶端讀取的消息
func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("錯誤: %v", err)
			}
			break
		}

		// 處理消息
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("解析消息錯誤: %v", err)
			continue
		}

		// 處理不同類型的消息
		switch msg.Type {
		case "ping":
			// 回應 ping
			pongMsg := Message{Type: "pong"}
			pongBytes, _ := json.Marshal(pongMsg)
			c.send <- pongBytes

		case "subscribe":
			// 處理訂閱請求
			var subReq SubscribeRequest
			if err := json.Unmarshal(message, &subReq); err != nil {
				log.Printf("解析訂閱請求錯誤: %v", err)
				continue
			}
			c.subscribe = subReq

			// 如果是訂閱 GameEvents，啟動模擬事件發送
			if subReq.Payload.Method == "SubscribeGameEvents" {
				go c.simulateEvents()
			}

			// 確認訂閱成功
			confirmMsg := Message{
				Type: "subscribed",
				Payload: map[string]interface{}{
					"method": subReq.Payload.Method,
					"time":   time.Now().Format(time.RFC3339),
				},
			}
			confirmBytes, _ := json.Marshal(confirmMsg)
			c.send <- confirmBytes
		}
	}
}

// 向客戶端寫入消息
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

// 模擬事件發送
func (c *Client) simulateEvents() {
	// 模擬發送 10 個事件，間隔 1 秒
	eventTypes := []string{
		"GAME_EVENT_TYPE_STAGE_CHANGED",
		"GAME_EVENT_TYPE_BALL_DRAWN",
		"GAME_EVENT_TYPE_GAME_CREATED",
	}

	for i := 0; i < 10; i++ {
		// 如果連接已關閉，停止發送
		if c.conn == nil {
			return
		}

		// 隨機選擇一個事件類型
		eventType := eventTypes[i%len(eventTypes)]

		// 如果客戶端指定了事件類型且當前事件不在列表中，跳過
		if len(c.subscribe.Payload.Data.EventTypes) > 0 {
			found := false
			for _, t := range c.subscribe.Payload.Data.EventTypes {
				if t == eventType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 根據事件類型構建不同的事件數據
		var eventData map[string]interface{}
		switch eventType {
		case "GAME_EVENT_TYPE_STAGE_CHANGED":
			eventData = map[string]interface{}{
				"eventType": eventType,
				"timestamp": time.Now().Format(time.RFC3339),
				"gameId":    "test-game-1",
				"stageChanged": map[string]interface{}{
					"oldStage": "GAME_STAGE_DRAWING_START",
					"newStage": "GAME_STAGE_DRAWING_CLOSE",
				},
			}
		case "GAME_EVENT_TYPE_BALL_DRAWN":
			eventData = map[string]interface{}{
				"eventType": eventType,
				"timestamp": time.Now().Format(time.RFC3339),
				"gameId":    "test-game-1",
				"ballDrawn": map[string]interface{}{
					"ball": map[string]interface{}{
						"number":    10 + i,
						"type":      "BALL_TYPE_REGULAR",
						"isLast":    i == 9,
						"timestamp": time.Now().Format(time.RFC3339),
					},
				},
			}
		case "GAME_EVENT_TYPE_GAME_CREATED":
			eventData = map[string]interface{}{
				"eventType": eventType,
				"timestamp": time.Now().Format(time.RFC3339),
				"gameId":    "test-game-1",
				"gameCreated": map[string]interface{}{
					"initialState": map[string]interface{}{
						"id":             "test-game-1",
						"stage":          "GAME_STAGE_NEW_ROUND",
						"startTime":      time.Now().Format(time.RFC3339),
						"hasJackpot":     true,
						"extraBallCount": 1,
					},
				},
			}
		}

		// 添加一個服務器標記
		eventData["_serverNote"] = "來自真實服務器的事件"

		// 將事件數據發送給客戶端
		eventBytes, _ := json.Marshal(eventData)
		c.send <- eventBytes

		// 等待 1 秒
		time.Sleep(1 * time.Second)
	}

	// 發送訂閱完成通知
	completeMsg := Message{
		Type: "subscription_complete",
		Payload: map[string]interface{}{
			"message": "All events have been sent",
			"time":    time.Now().Format(time.RFC3339),
		},
	}
	completeBytes, _ := json.Marshal(completeMsg)
	c.send <- completeBytes
}

func main() {
	flag.Parse()

	// 設置路由
	http.HandleFunc("/subscribe", handleWebSocket)

	// 提供靜態文件服務
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	// 啟動服務器
	fmt.Printf("WebSocket 服務器啟動於 %s\n", *addr)
	fmt.Println("請使用瀏覽器打開 http://localhost:9100/examples/grpc_demo.html 進行測試")
	log.Fatal(http.ListenAndServe(*addr, nil))
}
