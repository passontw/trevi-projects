package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// Command 定義一個通用的命令結構
type Command struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
}

// Response 定義服務器回應結構
type Response struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Type    string                 `json:"type,omitempty"`
}

// GameStartCommand 創建一個開始遊戲的命令
func GameStartCommand() Command {
	return Command{
		Type: "GAME_START",
	}
}

// HeartbeatCommand 創建一個心跳命令
func HeartbeatCommand(clientID string) Command {
	return Command{
		Type: "HEARTBEAT",
		Data: map[string]interface{}{
			"clientId": clientID,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
}

// testHTTPConnection 測試HTTP連接到給定的URL
func testHTTPConnection(urlStr string) {
	// 從WebSocket URL轉換為HTTP URL
	httpURL := urlStr
	if len(urlStr) > 3 && urlStr[0:3] == "ws:" {
		httpURL = "http:" + urlStr[3:]
	} else if len(urlStr) > 4 && urlStr[0:4] == "wss:" {
		httpURL = "https:" + urlStr[4:]
	}

	log.Printf("正在測試HTTP連接: %s\n", httpURL)

	// 創建請求
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", httpURL, nil)
	if err != nil {
		log.Printf("創建HTTP請求失敗: %v\n", err)
		return
	}

	// 發送請求
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP連接測試失敗: %v\n", err)
		log.Println("這可能表明目標服務器未運行或網絡連接問題")
		return
	}
	defer resp.Body.Close()

	log.Printf("HTTP連接成功，狀態碼: %d\n", resp.StatusCode)
	log.Println("伺服器正在運行，但可能不支援WebSocket或需要特殊的頭部")
}

func main() {
	// 定義命令行參數
	wsURL := flag.String("url", "ws://localhost:3002/ws", "WebSocket服務器URL")
	clientID := flag.String("client", "dealer-001", "客戶端ID")
	debug := flag.Bool("debug", false, "是否啟用調試模式")
	httpTest := flag.Bool("http-test", false, "在連接失敗時測試HTTP連接")
	flag.Parse()

	// 設置日誌
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("正在連接到服務器: %s\n", *wsURL)

	// 創建自定義的HTTP請求頭
	header := http.Header{}
	header.Add("User-Agent", "Lottery-Dealer-Client")
	header.Add("X-Client-ID", *clientID)

	// 設置websocket連接選項
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second // 增加握手超時時間

	if *debug {
		dialer.EnableCompression = false
		log.Println("調試模式已啟用")
	}

	// 連接到WebSocket服務器
	log.Println("正在嘗試連接...")
	c, resp, err := dialer.Dial(*wsURL, header)

	if err != nil {
		if resp != nil {
			log.Printf("連接失敗: %v, HTTP狀態碼: %d", err, resp.StatusCode)
			log.Printf("服務器回應頭: %v", resp.Header)
		} else {
			log.Printf("連接失敗: %v, 無HTTP回應", err)
			log.Println("請確認服務器是否已啟動，或檢查防火牆設置")
		}

		// 如果啟用HTTP測試且WebSocket連接失敗，嘗試進行HTTP連接測試
		if *httpTest {
			testHTTPConnection(*wsURL)
		}

		log.Println("\n您可以嘗試以下解決方案:")
		log.Println("1. 確認服務器地址和端口是否正確")
		log.Println("2. 檢查服務器是否已啟動並支援WebSocket")
		log.Println("3. 嘗試使用-url參數指定其他服務器地址")
		log.Println("4. 檢查網絡連接和防火牆設置")
		log.Println("5. 使用-http-test參數進行HTTP連接測試")
		log.Println("6. 檢查服務器是否需要額外的認證或頭部信息")

		// 提供可能的服務器地址建議
		fmt.Println("\n可能的WebSocket地址:")
		fmt.Println("- ws://localhost:3002 (默認)")
		fmt.Println("- ws://localhost:3002/ws")
		fmt.Println("- ws://localhost:8080/ws")
		fmt.Println("- ws://localhost:8080/socket")
		fmt.Println("- ws://localhost:8080/lottery/ws")
		fmt.Println("- ws://127.0.0.1:3002")

		os.Exit(1)
	}

	defer c.Close()
	log.Println("已成功連接到服務器")

	// 創建一個上下文，可以在接收到中斷信號時取消
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 創建一個通道來處理中斷信號
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 啟動一個 goroutine 來讀取服務器的消息
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("讀取消息失敗: %v", err)
				cancel()
				return
			}

			// 嘗試解析回應
			var response Response
			if err := json.Unmarshal(message, &response); err != nil {
				// 如果不是標準格式，僅顯示原始消息
				log.Printf("收到: %s", message)
			} else {
				// 解析成功，根據回應類型處理
				if response.Success {
					log.Printf("收到成功回應: %s", message)

					// 特別處理 GAME_START 回應
					if response.Type == "GAME_START_RESPONSE" || (response.Success && len(message) < 50) {
						log.Println("遊戲已成功創建！系統已在TiDB中建立新遊戲記錄。")
						log.Println("初始階段 has_jackpot 欄位設置為 false")
					}
				} else {
					log.Printf("收到錯誤回應: %s", message)
					if response.Message != "" {
						log.Printf("錯誤信息: %s", response.Message)
					}
				}
			}
		}
	}()

	// 啟動一個 goroutine 發送心跳
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				cmd := HeartbeatCommand(*clientID)
				err := sendCommand(c, cmd)
				if err != nil {
					log.Printf("發送心跳失敗: %v", err)
				}
			}
		}
	}()

	// 設置一個計時器，在連接5秒後發送GAME_START命令
	gameStartTimer := time.NewTimer(5 * time.Second)
	defer gameStartTimer.Stop()

	// 主循環
	for {
		select {
		case <-ctx.Done():
			return
		case <-gameStartTimer.C:
			cmd := GameStartCommand()
			log.Printf("發送開始遊戲命令: %+v", cmd)
			log.Println("此命令將在TiDB的games表中創建新遊戲記錄，初始階段has_jackpot=false")

			err := sendCommand(c, cmd)
			if err != nil {
				log.Printf("發送命令失敗: %v", err)
			} else {
				log.Println("命令已發送，等待回應 {\"success\": true}...")
			}
		case <-interrupt:
			log.Println("收到中斷信號，關閉連接...")
			// 發送關閉消息
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("發送關閉消息失敗: %v", err)
			}
			// 等待服務器關閉連接
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
			}
			return
		}
	}
}

// sendCommand 將命令轉換為JSON並發送到WebSocket連接
func sendCommand(c *websocket.Conn, cmd Command) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, data)
}
