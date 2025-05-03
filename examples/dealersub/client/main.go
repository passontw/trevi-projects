package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// 客戶端請求結構
type ClientRequest struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// 客戶端訂閱請求資料
type ClientSubscribeData struct {
	Topic string `json:"topic"`
}

// 客戶端發布請求資料
type ClientPublishData struct {
	Topic string      `json:"topic"`
	Data  interface{} `json:"data"`
}

// 服務器回應結構
type ServerResponse struct {
	Type      string                 `json:"type"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// 訂閱客戶端
type SubscribeClient struct {
	conn       *websocket.Conn
	send       chan ClientRequest
	subscribed map[string]bool
	done       chan struct{}
}

// 建立新的訂閱客戶端
func NewSubscribeClient(urlStr string) (*SubscribeClient, error) {
	// 解析 WebSocket URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("無效的 URL: %v", err)
	}

	// 連線至 WebSocket 伺服器
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("無法連線: %v", err)
	}

	client := &SubscribeClient{
		conn:       conn,
		send:       make(chan ClientRequest, 256),
		subscribed: make(map[string]bool),
		done:       make(chan struct{}),
	}

	// 啟動讀寫協程
	go client.readPump()
	go client.writePump()

	return client, nil
}

// 讀取伺服器訊息
func (c *SubscribeClient) readPump() {
	defer func() {
		c.conn.Close()
		close(c.done)
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("讀取錯誤: %v", err)
			}
			break
		}

		// 解析回應
		var response ServerResponse
		if err := json.Unmarshal(message, &response); err != nil {
			log.Printf("解析回應失敗: %v", err)
			continue
		}

		// 顯示收到的訊息
		log.Printf("收到訊息:")
		log.Printf("  類型: %s", response.Type)
		log.Printf("  成功: %v", response.Success)
		log.Printf("  訊息: %s", response.Message)

		// 處理訂閱狀態
		if response.Type == "SUBSCRIBED" && response.Success {
			if topic, ok := response.Data["topic"].(string); ok {
				c.subscribed[topic] = true
				log.Printf("已成功訂閱主題: %s", topic)
			}
		} else if response.Type == "UNSUBSCRIBED" && response.Success {
			if topic, ok := response.Data["topic"].(string); ok {
				delete(c.subscribed, topic)
				log.Printf("已成功取消訂閱主題: %s", topic)
			}
		} else if response.Type == "MESSAGE" {
			// 顯示收到的訊息資料
			if response.Data != nil {
				log.Printf("  訊息資料:")
				if topic, ok := response.Data["topic"].(string); ok {
					log.Printf("    主題: %s", topic)
				}
				if data, ok := response.Data["data"]; ok {
					log.Printf("    內容: %v", data)
				}
			}
		}

		// 顯示詳細資料
		if response.Data != nil && response.Type != "MESSAGE" {
			log.Printf("  詳細資料:")
			for key, value := range response.Data {
				log.Printf("    %s: %v", key, value)
			}
		}

		log.Printf("  時間戳: %v", time.Unix(response.Timestamp, 0).Format(time.RFC3339))
	}
}

// 發送訊息至伺服器
func (c *SubscribeClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case request, ok := <-c.send:
			if !ok {
				// 通道已關閉
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 序列化請求
			requestBytes, err := json.Marshal(request)
			if err != nil {
				log.Printf("序列化請求失敗: %v", err)
				continue
			}

			// 發送請求
			log.Printf("發送請求: %s", string(requestBytes))
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, requestBytes); err != nil {
				log.Printf("發送請求失敗: %v", err)
				return
			}
		case <-ticker.C:
			// 發送 Ping 保持連線
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

// 訂閱主題
func (c *SubscribeClient) Subscribe(topic string) {
	request := ClientRequest{
		Type: "SUBSCRIBE",
		Data: ClientSubscribeData{
			Topic: topic,
		},
	}
	c.send <- request
}

// 取消訂閱主題
func (c *SubscribeClient) Unsubscribe(topic string) {
	request := ClientRequest{
		Type: "UNSUBSCRIBE",
		Data: ClientSubscribeData{
			Topic: topic,
		},
	}
	c.send <- request
}

// 發布訊息到主題
func (c *SubscribeClient) Publish(topic string, data interface{}) {
	request := ClientRequest{
		Type: "PUBLISH",
		Data: ClientPublishData{
			Topic: topic,
			Data:  data,
		},
	}
	c.send <- request
}

// 關閉連線
func (c *SubscribeClient) Close() {
	c.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(time.Second)
	c.conn.Close()
}

func main() {
	// 設定默認參數
	var (
		addr    = "localhost:9000"
		mode    = "subscriber"
		topic   = "game_events"
		message = ""
	)

	// 處理命令行參數
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// 支持 -參數=值 格式
		if strings.HasPrefix(arg, "-mode=") {
			mode = strings.TrimPrefix(arg, "-mode=")
		} else if strings.HasPrefix(arg, "-topic=") {
			topic = strings.TrimPrefix(arg, "-topic=")
		} else if strings.HasPrefix(arg, "-message=") {
			message = strings.TrimPrefix(arg, "-message=")
		} else if strings.HasPrefix(arg, "-addr=") {
			addr = strings.TrimPrefix(arg, "-addr=")
		} else if arg == "-mode" && i+1 < len(os.Args) {
			// 支持 -參數 值 格式
			mode = os.Args[i+1]
			i++
		} else if arg == "-topic" && i+1 < len(os.Args) {
			topic = os.Args[i+1]
			i++
		} else if arg == "-message" && i+1 < len(os.Args) {
			message = os.Args[i+1]
			i++
		} else if arg == "-addr" && i+1 < len(os.Args) {
			addr = os.Args[i+1]
			i++
		}
	}

	// 建立 WebSocket URL
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	log.Printf("正在連線至 %s", u.String())

	// 建立訂閱客戶端
	client, err := NewSubscribeClient(u.String())
	if err != nil {
		log.Fatalf("無法建立客戶端: %v", err)
	}
	defer client.Close()

	// 設定中斷訊號處理
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 根據模式執行不同操作
	switch strings.ToLower(mode) {
	case "subscriber":
		// 訂閱者模式
		log.Printf("以訂閱者模式運行，等待訂閱主題 %s 的訊息", topic)
		time.Sleep(1 * time.Second) // 等待連線建立
		client.Subscribe(topic)
	case "publisher":
		// 發布者模式
		log.Printf("以發布者模式運行，向主題 %s 發布訊息", topic)
		time.Sleep(1 * time.Second) // 等待連線建立

		// 如果沒有指定訊息，使用默認訊息
		msgData := message
		if msgData == "" {
			msgData = fmt.Sprintf("這是一條測試訊息，時間: %s", time.Now().Format(time.RFC3339))
		}

		client.Publish(topic, msgData)

		// 發布完訊息後等待一段時間再退出
		time.Sleep(2 * time.Second)
		return
	case "both":
		// 同時訂閱和發布
		log.Printf("同時以訂閱者和發布者模式運行")
		time.Sleep(1 * time.Second) // 等待連線建立

		// 先訂閱主題
		client.Subscribe(topic)

		// 等待一秒後發布訊息
		time.Sleep(1 * time.Second)

		// 如果沒有指定訊息，使用默認訊息
		msgData := message
		if msgData == "" {
			msgData = fmt.Sprintf("這是一條測試訊息，時間: %s", time.Now().Format(time.RFC3339))
		}

		client.Publish(topic, msgData)
	default:
		log.Printf("未知模式 %s，使用訂閱者模式", mode)
		time.Sleep(1 * time.Second) // 等待連線建立
		client.Subscribe(topic)
	}

	// 等待中斷信號
	for {
		select {
		case <-interrupt:
			log.Println("收到中斷信號，正在關閉連線...")

			// 對所有訂閱的主題取消訂閱
			for topic := range client.subscribed {
				client.Unsubscribe(topic)
			}

			// 關閉客戶端
			client.Close()
			return
		case <-client.done:
			log.Println("連線已關閉")
			return
		}
	}
}
