package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// 請求結構
type Request struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// 開始新遊戲請求
type StartNewRoundData struct {
	// 此處為空，因為 START_NEW_ROUND 不需要額外資料
}

// 回應結構
type Response struct {
	Type      string                 `json:"type"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

func main() {
	// 設定中斷訊號處理
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 建立 WebSocket URL
	u := url.URL{Scheme: "ws", Host: "localhost:9000", Path: "/ws"}
	log.Printf("正在連線至 %s", u.String())

	// 連線至 WebSocket 伺服器
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("無法連線: %v", err)
	}
	defer c.Close()

	// 讀取訊息的通道
	done := make(chan struct{})
	responseReceived := make(chan bool)

	// 處理收到的訊息
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("讀取錯誤: %v", err)
				return
			}

			log.Printf("收到原始訊息: %s", string(message))

			// 嘗試解析為回應格式
			var response Response
			if err := json.Unmarshal(message, &response); err != nil {
				log.Printf("解析回應訊息失敗: %v", err)
				continue
			}

			log.Printf("收到回應:")
			log.Printf("  類型: %s", response.Type)
			log.Printf("  成功: %v", response.Success)
			log.Printf("  訊息: %s", response.Message)

			if response.Data != nil {
				log.Printf("  資料:")
				for key, value := range response.Data {
					log.Printf("    %s: %v", key, value)
				}
			}

			log.Printf("  時間戳: %v", time.Unix(response.Timestamp, 0).Format(time.RFC3339))

			// 如果是實際的遊戲回應（而非歡迎訊息），通知已收到回應
			if response.Type == "response" || response.Type == "START_NEW_ROUND" {
				select {
				case responseReceived <- true:
				default:
				}
			}
		}
	}()

	// 等待兩秒後發送 StartNewRoundRequest
	log.Printf("等待兩秒後將發送 StartNewRoundRequest...")
	time.Sleep(2 * time.Second)

	// 構建請求
	request := Request{
		Type: "START_NEW_ROUND",
	}

	// 序列化請求
	requestBytes, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("序列化請求失敗: %v", err)
	}

	// 發送請求
	log.Printf("發送請求: %s", string(requestBytes))
	if err := c.WriteMessage(websocket.TextMessage, requestBytes); err != nil {
		log.Fatalf("發送請求失敗: %v", err)
	}

	// 等待回應或中斷
	for {
		select {
		case <-responseReceived:
			log.Println("已成功接收並處理回應")
			time.Sleep(1 * time.Second) // 給一點時間確保訊息完全處理
			return
		case <-interrupt:
			log.Println("收到中斷信號，正在關閉連線...")

			// 發送關閉訊息
			err := c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("無法發送關閉訊息: %v", err)
			}

			// 等待伺服器關閉連線
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		case <-time.After(30 * time.Second):
			// 30秒超時
			fmt.Println("等待回應超時")
			return
		}
	}
}
