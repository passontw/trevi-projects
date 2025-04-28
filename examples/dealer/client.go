package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// 定義 WebSocket 連接常量
const (
	// 允許客戶端不發送 pong 的最大時間
	pongWait = 60 * time.Second

	// 發送 ping 的頻率，必須小於 pongWait
	pingPeriod = (pongWait * 9) / 10

	// 寫入超時時間
	writeWait = 10 * time.Second
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

// WebSocketError 定義WebSocket錯誤類型
type WebSocketError struct {
	Code        int
	Description string
	Detail      string
	Suggestion  string
}

// WebSocketErrorCodes 定義常見的WebSocket錯誤碼及說明
var WebSocketErrorCodes = map[int]WebSocketError{
	1000: {1000, "正常關閉", "連接正常關閉", "這是正常行為，無需擔心"},
	1001: {1001, "離開", "客戶端或服務器已離開", "檢查服務器狀態或重新連接"},
	1002: {1002, "協議錯誤", "端點因協議錯誤而終止連接", "檢查客戶端和服務器的WebSocket協議實現"},
	1003: {1003, "不支持的數據", "端點收到了無法處理的數據類型", "檢查發送的數據格式是否符合服務器要求"},
	1004: {1004, "保留", "保留的狀態碼", "這是保留的狀態碼，不應由應用程序使用"},
	1005: {1005, "無狀態碼", "連接關閉但未提供狀態碼", "檢查服務器日誌獲取更多信息"},
	1006: {1006, "異常關閉", "連接異常關閉，可能是網絡中斷或服務器崩潰", "檢查網絡連接或服務器狀態，嘗試重新連接"},
	1007: {1007, "無效數據", "收到的消息包含不一致的數據", "檢查發送的數據格式是否正確"},
	1008: {1008, "策略違規", "收到的消息違反服務器策略", "檢查發送的消息是否符合服務器的安全策略"},
	1009: {1009, "消息過大", "收到的消息太大，無法處理", "減小發送消息的大小"},
	1010: {1010, "需要擴展", "客戶端請求的擴展未得到服務器支持", "檢查WebSocket配置，移除不必要的擴展"},
	1011: {1011, "意外情況", "服務器遇到意外情況導致無法完成請求", "檢查服務器日誌查找錯誤原因"},
	1012: {1012, "服務重啟", "服務器正在重啟", "稍後重試連接"},
	1013: {1013, "暫時性問題", "服務器暫時性問題，如過載", "稍後重試連接"},
	1014: {1014, "保留", "保留的狀態碼", "這是保留的狀態碼，不應由應用程序使用"},
	1015: {1015, "TLS握手失敗", "TLS握手失敗", "檢查SSL/TLS配置或證書是否有效"},
}

// 追蹤命令發送狀態的全局變數
var (
	bettingClosedSent = make(map[string]bool) // 使用 gameID 作為 key
	commandLock       sync.Mutex              // 用於保護上面的 map
)

// 檢查並標記命令是否已發送
func markCommandSent(gameID string, commandType string) bool {
	commandLock.Lock()
	defer commandLock.Unlock()

	key := gameID + ":" + commandType
	if bettingClosedSent[key] {
		return true // 已發送過
	}

	bettingClosedSent[key] = true
	return false // 尚未發送過
}

// 客戶端結構體
type ClientState struct {
	conn          *websocket.Conn
	isConnected   atomic.Bool
	messageBuffer []Command
	connMutex     sync.Mutex
	sendMutex     sync.Mutex
	config        *ClientConfig
}

// 客戶端配置
type ClientConfig struct {
	wsURL           string
	clientID        string
	debug           bool
	httpTest        bool
	maxRetries      int
	permanentRetry  bool
	retryDelay      time.Duration
	timeout         time.Duration
	keepalive       bool
	autoReconnect   bool
	reconnectDelay  time.Duration
	heartbeatTicker *time.Ticker
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
	if resp.StatusCode == 101 {
		log.Println("服務器返回101狀態碼，表示協議切換成功，WebSocket應該可以正常工作")
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("伺服器返回成功狀態碼，但不是101，這可能意味著端點存在但不支援WebSocket")
	} else if resp.StatusCode == 404 {
		log.Println("伺服器返回404，這表示WebSocket路徑可能不正確")
	} else {
		log.Println("伺服器正在運行，但可能不支援WebSocket或需要特殊的頭部")
	}
}

// analyzeCloseError 分析WebSocket關閉錯誤並提供診斷信息
func analyzeCloseError(err error) {
	if err == nil {
		return
	}

	// 檢查是否為CloseError類型
	if closeErr, ok := err.(*websocket.CloseError); ok {
		code := closeErr.Code

		log.Printf("WebSocket連接關閉，錯誤碼: %d\n", code)

		if errInfo, exists := WebSocketErrorCodes[code]; exists {
			log.Printf("錯誤類型: %s\n", errInfo.Description)
			log.Printf("詳細說明: %s\n", errInfo.Detail)
			log.Printf("建議解決方案: %s\n", errInfo.Suggestion)
		} else {
			log.Printf("未知的錯誤碼: %d\n", code)
		}

		return
	}

	// 檢查其他常見錯誤
	errMsg := err.Error()

	if strings.Contains(errMsg, "unexpected EOF") {
		log.Println("連接意外終止，可能是服務器崩潰或網絡問題")
		log.Println("建議: 檢查服務器日誌，確認服務器狀態，或稍後重試")
	} else if strings.Contains(errMsg, "connection reset by peer") {
		log.Println("連接被對方重置，服務器可能已關閉連接")
		log.Println("建議: 確認服務器是否仍在運行，或檢查服務器日誌")
	} else if strings.Contains(errMsg, "i/o timeout") {
		log.Println("操作超時，可能是網絡延遲或服務器無響應")
		log.Println("建議: 檢查網絡連接，增加超時時間，或確認服務器負載")
	} else if strings.Contains(errMsg, "use of closed network connection") {
		log.Println("使用已關閉的網絡連接，可能是客戶端代碼邏輯問題")
		log.Println("建議: 檢查代碼中是否有提前關閉連接的操作")
	} else {
		log.Printf("未分類的WebSocket錯誤: %v\n", err)
		log.Println("建議: 查看詳細錯誤信息，檢查服務器日誌，並考慮重新連接")
	}
}

// NewClientState 創建一個新的客戶端狀態
func NewClientState(config *ClientConfig) *ClientState {
	return &ClientState{
		conn:          nil,
		messageBuffer: make([]Command, 0),
		connMutex:     sync.Mutex{},
		sendMutex:     sync.Mutex{},
		config:        config,
	}
}

// Connect 連接到WebSocket服務器
func (c *ClientState) Connect(ctx context.Context) error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	if c.conn != nil {
		return nil // 已經連接
	}

	// 創建自定義的HTTP請求頭
	header := http.Header{}
	header.Add("User-Agent", "Lottery-Dealer-Client")
	header.Add("X-Client-ID", c.config.clientID)

	// 設置websocket連接選項
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = c.config.timeout

	if c.config.debug {
		dialer.EnableCompression = false
		log.Println("調試模式已啟用")
	}

	// 嘗試連接
	log.Println("正在嘗試連接...")
	conn, resp, err := dialer.Dial(c.config.wsURL, header)

	if err != nil {
		if resp != nil {
			log.Printf("連接失敗: %v, HTTP狀態碼: %d", err, resp.StatusCode)
			log.Printf("服務器回應頭: %v", resp.Header)
		} else {
			log.Printf("連接失敗: %v, 無HTTP回應", err)
		}

		if c.config.httpTest {
			testHTTPConnection(c.config.wsURL)
		}

		return err
	}

	log.Println("已成功連接到服務器")
	c.conn = conn
	c.isConnected.Store(true)

	// 設置讀取限制
	c.conn.SetReadLimit(512 * 1024) // 512KB

	// 設置初始讀取截止時間
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// 設置 pong 處理程序，在收到 ping 時更新最後活動時間
	c.conn.SetPongHandler(func(string) error {
		// 收到 pong 後更新讀取截止時間
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if c.config.debug {
			log.Println("收到服務器的 ping，已回應 pong")
		}
		return nil
	})

	return nil
}

// Disconnect 斷開與WebSocket服務器的連接
func (c *ClientState) Disconnect() {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	if c.conn == nil {
		c.isConnected.Store(false)
		return // 已經斷開連接
	}

	// 先停止心跳
	if c.config.heartbeatTicker != nil {
		c.config.heartbeatTicker.Stop()
		c.config.heartbeatTicker = nil
	}

	// 嘗試發送關閉消息，最多重試3次
	var closeErr error
	for i := 0; i < 3; i++ {
		// 設置寫入超時
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))

		// 發送關閉消息
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "客戶端主動關閉連接")
		closeErr = c.conn.WriteMessage(websocket.CloseMessage, closeMessage)
		if closeErr == nil {
			break
		}

		// 記錄錯誤但不中斷，因為我們仍然會嘗試關閉連接
		log.Printf("發送關閉消息嘗試 %d 失敗: %v", i+1, closeErr)

		// 如果錯誤是由於連接已關閉，不再重試
		if strings.Contains(closeErr.Error(), "use of closed network connection") ||
			strings.Contains(closeErr.Error(), "broken pipe") {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if closeErr != nil && !strings.Contains(closeErr.Error(), "use of closed network connection") {
		log.Printf("發送關閉消息失敗: %v", closeErr)
	}

	// 設置關閉讀取超時，確保任何讀取操作都能立即返回
	c.conn.SetReadDeadline(time.Now())

	// 關閉連接
	err := c.conn.Close()
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		log.Printf("關閉連接失敗: %v", err)
	}

	c.conn = nil
	c.isConnected.Store(false)
	log.Println("已斷開與服務器的連接")
}

// reconnect 重新連接到WebSocket服務器
func (c *ClientState) reconnect(ctx context.Context) error {
	// 先標記連接為已斷開
	c.isConnected.Store(false)

	// 先確保優雅地斷開現有連接
	c.Disconnect()

	log.Println("準備重新連接到服務器...")

	// 嘗試重新連接
	maxRetries := c.config.maxRetries
	if c.config.permanentRetry {
		maxRetries = -1 // 無限重試
	}

	retryCount := 0
	var lastError error

	// 設置指數退避重試
	backoff := c.config.reconnectDelay
	maxBackoff := 60 * time.Second // 最大退避時間為60秒

	for {
		if maxRetries >= 0 && retryCount >= maxRetries {
			log.Printf("重連失敗: 達到最大重試次數 %d", maxRetries)
			c.isConnected.Store(false)
			return fmt.Errorf("達到最大重試次數 %d: %w", maxRetries, lastError)
		}

		log.Printf("嘗試重新連接 (嘗試 %d)...", retryCount+1)
		err := c.Connect(ctx)
		if err == nil {
			log.Printf("重新連接成功 (嘗試 %d)", retryCount+1)

			// 確保標記為已連接
			c.isConnected.Store(true)

			// 重新啟動心跳
			if c.config.keepalive {
				log.Println("重新啟動心跳...")
				c.StartHeartbeat(ctx)
			}

			// 重新發送緩衝區中的消息
			if len(c.messageBuffer) > 0 {
				log.Printf("重新發送 %d 個緩衝消息", len(c.messageBuffer))
				messagesCopy := make([]Command, len(c.messageBuffer))
				copy(messagesCopy, c.messageBuffer)

				// 清空緩衝區
				c.messageBuffer = make([]Command, 0)

				// 發送緩衝的消息
				for _, cmd := range messagesCopy {
					if err := c.SendCommand(cmd); err != nil {
						log.Printf("重新發送消息失敗: %v", err)
						// 不中斷，繼續發送其他消息
					} else {
						log.Printf("成功重新發送消息: %v", cmd.Type)
					}
				}
			}

			return nil
		}

		lastError = err
		retryCount++

		// 計算指數退避時間，但不超過最大退避時間
		if retryCount > 1 {
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		log.Printf("重新連接失敗: %v，將在 %v 後重試", err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// 繼續重試
		}
	}
}

// SendCommand 發送命令到WebSocket服務器
func (c *ClientState) SendCommand(cmd Command) error {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	// 檢查連接狀態
	if !c.isConnected.Load() {
		// 將消息加入緩衝區，等待重連後發送
		c.messageBuffer = append(c.messageBuffer, cmd)
		return fmt.Errorf("未連接到服務器，消息已加入緩衝區")
	}

	// 序列化命令
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("序列化命令失敗: %w", err)
	}

	// 發送消息
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	if c.conn == nil {
		// 將消息加入緩衝區，等待重連後發送
		c.messageBuffer = append(c.messageBuffer, cmd)
		return fmt.Errorf("連接為空，消息已加入緩衝區")
	}

	// 設置寫入超時
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))

	// 嘗試發送消息，最多重試3次
	var sendErr error
	for i := 0; i < 3; i++ {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			sendErr = err
			// 休眠短暫時間後重試
			time.Sleep(100 * time.Millisecond)
			continue
		}
		// 發送成功，清除錯誤並跳出循環
		sendErr = nil
		break
	}

	// 如果在多次嘗試後仍然失敗
	if sendErr != nil {
		// 將消息加入緩衝區，等待重連後發送
		c.messageBuffer = append(c.messageBuffer, cmd)

		// 檢查是否為超時或連接關閉錯誤
		if strings.Contains(sendErr.Error(), "i/o timeout") ||
			strings.Contains(sendErr.Error(), "use of closed network connection") ||
			strings.Contains(sendErr.Error(), "broken pipe") {
			// 標記連接為斷開，觸發重連
			c.isConnected.Store(false)
		}

		return fmt.Errorf("發送消息失敗: %w", sendErr)
	}

	return nil
}

// StartHeartbeat 啟動心跳檢測
func (c *ClientState) StartHeartbeat(ctx context.Context) {
	if !c.config.keepalive {
		return // 不需要心跳
	}

	if c.config.heartbeatTicker != nil {
		c.config.heartbeatTicker.Stop()
	}

	// 使用 pingPeriod 常量代替硬編碼的 15 秒
	c.config.heartbeatTicker = time.NewTicker(pingPeriod)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("心跳協程崩潰: %v", r)
			}
			if c.config.heartbeatTicker != nil {
				c.config.heartbeatTicker.Stop()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.config.heartbeatTicker.C:
				// 發送應用層心跳消息
				cmd := HeartbeatCommand(c.config.clientID)
				err := c.SendCommand(cmd)
				if err != nil && c.config.debug {
					log.Printf("發送心跳失敗: %v", err)
				}

				// 發送 WebSocket 協議層的 Ping
				c.connMutex.Lock()
				if c.conn != nil {
					// 設置寫入超時
					c.conn.SetWriteDeadline(time.Now().Add(writeWait))
					// 發送 ping 消息
					if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						if c.config.debug {
							log.Printf("發送 ping 消息失敗: %v", err)
						}
						// 不要中斷心跳循環，只記錄錯誤
					}
				}
				c.connMutex.Unlock()
			}
		}
	}()
}

// ReadMessages 啟動消息讀取循環
func (c *ClientState) ReadMessages(ctx context.Context, msgChan chan<- []byte, errChan chan<- error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("讀取消息協程崩潰: %v", r)
				errChan <- fmt.Errorf("讀取消息協程崩潰: %v", r)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if !c.isConnected.Load() {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				c.connMutex.Lock()
				conn := c.conn
				c.connMutex.Unlock()

				if conn == nil {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				_, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err,
						websocket.CloseNormalClosure,
						websocket.CloseGoingAway) {
						analyzeCloseError(err)
						errChan <- err

						// 如果需要自動重連
						if c.config.autoReconnect {
							log.Println("連接關閉，嘗試自動重連...")
							go func() {
								reconnectErr := c.reconnect(ctx)
								if reconnectErr != nil {
									log.Printf("自動重連失敗: %v", reconnectErr)
									errChan <- reconnectErr
								}
							}()
						} else {
							c.isConnected.Store(false)
						}
					} else {
						log.Printf("讀取消息失敗: %v", err)
						errChan <- err
					}

					time.Sleep(c.config.reconnectDelay)
					continue
				}

				// 成功讀取消息後，更新讀取截止時間
				_ = conn.SetReadDeadline(time.Now().Add(pongWait))

				// 檢查是否為心跳消息，如果是則靜默處理
				if isHeartbeatMessage(message) {
					if c.config.debug {
						log.Printf("收到心跳消息: %s", message)
					}
					continue
				}

				// 處理可能包含多個JSON消息的情況
				messages := bytes.Split(message, []byte{'\n'})
				for _, msg := range messages {
					// 跳過空消息
					if len(bytes.TrimSpace(msg)) == 0 {
						continue
					}
					// 發送至消息通道進行處理
					msgChan <- msg
				}
			}
		}
	}()
}

// isHeartbeatMessage 檢查消息是否為心跳消息
func isHeartbeatMessage(message []byte) bool {
	// 快速檢查是否包含HEARTBEAT關鍵字
	if !strings.Contains(string(message), "HEARTBEAT") {
		return false
	}

	// 更詳細的檢查
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		// 如果解析失敗，不是有效的JSON，不是心跳消息
		return false
	}

	// 檢查消息類型
	msgTypeStr, ok := msg["type"].(string)
	if !ok {
		return false
	}

	// 檢查類型是否為heartbeat
	return strings.ToUpper(msgTypeStr) == "HEARTBEAT"
}

func main() {
	// 定義命令行參數
	wsURL := flag.String("url", "ws://localhost:3002/ws", "WebSocket服務器URL")
	clientID := flag.String("client", "dealer-001", "客戶端ID")
	debug := flag.Bool("debug", false, "是否啟用調試模式")
	httpTest := flag.Bool("http-test", false, "在連接失敗時測試HTTP連接")
	maxRetries := flag.Int("retries", 3, "連接失敗時的最大重試次數")
	permanentRetry := flag.Bool("permanent-retry", false, "是否永久重試連接")
	retryDelay := flag.Duration("retry-delay", 3*time.Second, "重試之間的延遲時間")
	timeout := flag.Duration("timeout", 10*time.Second, "連接超時時間")
	keepalive := flag.Bool("keepalive", true, "是否啟用保活機制")
	autoReconnect := flag.Bool("auto-reconnect", true, "是否自動重連")
	reconnectDelay := flag.Duration("reconnect-delay", 2*time.Second, "重連延遲時間")
	flag.Parse()

	// 設置日誌
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("正在連接到服務器: %s\n", *wsURL)

	// 創建一個上下文，可以在接收到中斷信號時取消
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 創建一個通道來處理中斷信號
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 創建客戶端配置
	config := &ClientConfig{
		wsURL:          *wsURL,
		clientID:       *clientID,
		debug:          *debug,
		httpTest:       *httpTest,
		maxRetries:     *maxRetries,
		permanentRetry: *permanentRetry,
		retryDelay:     *retryDelay,
		timeout:        *timeout,
		keepalive:      *keepalive,
		autoReconnect:  *autoReconnect,
		reconnectDelay: *reconnectDelay,
	}

	// 創建客戶端狀態
	client := NewClientState(config)

	// 嘗試連接
	err := client.Connect(ctx)
	if err != nil {
		log.Println("\n您可以嘗試以下解決方案:")
		log.Println("1. 確認服務器地址和端口是否正確")
		log.Println("2. 檢查服務器是否已啟動並支援WebSocket")
		log.Println("3. 嘗試使用-url參數指定其他服務器地址")
		log.Println("4. 檢查網絡連接和防火牆設置")
		log.Println("5. 使用-http-test參數進行HTTP連接測試")
		log.Println("6. 檢查服務器是否需要額外的認證或頭部信息")
		log.Println("7. 增加-timeout參數值以延長連接超時時間")
		log.Println("8. 使用-permanent-retry參數啟用永久重試")
		log.Println("9. 檢查服務器日誌了解更多信息")

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

	// 啟動心跳
	client.StartHeartbeat(ctx)

	// 創建消息通道和錯誤通道
	msgChan := make(chan []byte, 10)
	errChan := make(chan error, 10)

	// 啟動消息讀取循環
	client.ReadMessages(ctx, msgChan, errChan)

	// 設置一個計時器，在連接5秒後發送GAME_START命令
	gameStartTimer := time.NewTimer(5 * time.Second)
	defer gameStartTimer.Stop()

	// 設置一個定時器，每隔30秒打印命令發送狀態
	statusTimer := time.NewTicker(30 * time.Second)
	defer statusTimer.Stop()

	// 主循環
	for {
		select {
		case <-ctx.Done():
			client.Disconnect()
			return
		case <-statusTimer.C:
			printCommandStatus()
		case msg := <-msgChan:
			// 嘗試解析回應
			var rawJSON map[string]interface{}
			if err := json.Unmarshal(msg, &rawJSON); err != nil {
				log.Printf("收到非JSON消息: %s", msg)
				continue
			}

			// 檢查消息類型
			if msgType, ok := rawJSON["type"].(string); ok {
				// 心跳檢測
				if strings.ToUpper(msgType) == "HEARTBEAT" {
					if client.config.debug {
						log.Printf("收到心跳回應: %s", msg)
					}
					continue
				}

				// 系統通知
				if msgType == "system_notice" {
					if data, ok := rawJSON["data"].(map[string]interface{}); ok {
						if message, ok := data["message"].(string); ok {
							log.Printf("系統通知: %s", message)
						} else {
							log.Printf("收到系統通知: %s", msg)
						}
					} else {
						log.Printf("收到系統通知: %s", msg)
					}
					continue
				}

				// 遊戲事件: LUCKY_NUMBERS_SET, GAME_CREATED 等
				if msgType == "LUCKY_NUMBERS_SET" {
					log.Printf("遊戲事件 (%s): %s", msgType, prettyJSON(msg))

					// 獲取遊戲數據
					gameData, ok := rawJSON["data"].(map[string]interface{})
					if !ok {
						log.Printf("無法獲取遊戲數據")
						continue
					}

					gameInfo, ok := gameData["game"].(map[string]interface{})
					if !ok {
						log.Printf("無法獲取遊戲信息")
						continue
					}

					// 注意: 僅當遊戲狀態為 SHOW_LUCKYNUMS 時才發送 BETTING_STARTED 事件
					// 避免重複發送導致服務器錯誤
					if gameState, ok := gameInfo["state"].(string); ok && gameState == "SHOW_LUCKYNUMS" {
						// 創建一個發送BETTING_STARTED事件的計時器，1秒後觸發
						go func() {
							time.Sleep(1 * time.Second)

							// 創建BETTING_STARTED事件
							bettingStartedCmd := Command{
								Type: "BETTING_STARTED",
								Data: map[string]interface{}{
									"game": map[string]interface{}{
										"id":         gameInfo["id"],
										"state":      "betting",
										"startTime":  time.Now().Format(time.RFC3339),
										"hasJackpot": gameInfo["hasJackpot"],
									},
									"betting": map[string]interface{}{
										"playerCount": 0,
										"totalAmount": 0,
									},
								},
								Timestamp: time.Now().Format(time.RFC3339Nano),
							}

							// 發送命令
							err := client.SendCommand(bettingStartedCmd)
							if err != nil {
								log.Printf("發送BETTING_STARTED事件失敗: %v", err)
							} else {
								log.Printf("已發送BETTING_STARTED事件，遊戲ID: %v", gameInfo["id"])
							}
						}()
					} else {
						log.Printf("遊戲狀態不是SHOW_LUCKYNUMS，跳過發送BETTING_STARTED事件")
					}

					continue
				} else if msgType == "GAME_CREATED" {
					log.Printf("遊戲事件 (%s): %s", msgType, prettyJSON(msg))
					continue
				} else if msgType == "BETTING_STARTED" {
					log.Printf("收到類型為 'BETTING_STARTED' 的消息: %s", string(msg))

					// 獲取遊戲ID和狀態信息
					if rawJSON["data"] != nil {
						gameData, ok := rawJSON["data"].(map[string]interface{})
						if !ok {
							log.Printf("無法獲取遊戲數據")
							continue
						}

						gameInfo, ok := gameData["game"].(map[string]interface{})
						if !ok {
							log.Printf("無法獲取遊戲信息")
							continue
						}

						gameID := gameInfo["id"]
						gameState := gameInfo["state"]
						hasJackpot := gameInfo["hasJackpot"]

						gameIDStr := fmt.Sprintf("%v", gameID)
						log.Printf("遊戲ID: %v, 遊戲狀態: %v, 是否有彩池: %v",
							gameID, gameState, hasJackpot)

						// 設置5秒後發送BETTING_CLOSED命令
						go func(gameID interface{}, gameState interface{}, hasJackpot interface{}, gameIDStr string) {
							time.Sleep(5 * time.Second)

							// 檢查該遊戲的BETTING_CLOSED命令是否已發送
							if markCommandSent(gameIDStr, "BETTING_CLOSED") {
								log.Printf("BETTING_CLOSED命令已經為遊戲ID %s 發送過，跳過重複發送", gameIDStr)
								return
							}

							// 創建BETTING_CLOSED命令
							bettingClosedCmd := Command{
								Type: "BETTING_CLOSED",
								Data: map[string]interface{}{
									"game": map[string]interface{}{
										"id":         gameID,
										"state":      gameState,
										"hasJackpot": hasJackpot,
									},
									"betting_summary": map[string]interface{}{
										"player_count": 15,
										"total_amount": 1500.00,
										"timestamp":    time.Now().Format(time.RFC3339),
									},
								},
								Timestamp: time.Now().Format(time.RFC3339),
							}

							log.Printf("5秒計時結束，準備發送BETTING_CLOSED命令")

							// 重試機制，確保命令發送成功
							maxRetries := 3
							for i := 0; i < maxRetries; i++ {
								err := client.SendCommand(bettingClosedCmd)
								if err != nil {
									log.Printf("發送BETTING_CLOSED失敗 (嘗試 %d/%d): %v", i+1, maxRetries, err)
									time.Sleep(1 * time.Second)
								} else {
									log.Printf("已成功發送BETTING_CLOSED命令，遊戲ID: %v", gameID)
									break
								}
							}
						}(gameID, gameState, hasJackpot, gameIDStr)
					}
				} else if msgType == "DRAW_RESULT_RESPONSE" {
					log.Printf("收到類型為 'DRAW_RESULT_RESPONSE' 的消息: %s", string(msg))

					// 獲取遊戲ID和相關信息
					if data, ok := rawJSON["data"].(map[string]interface{}); ok {
						gameID := data["game_id"]
						gameState := data["state"]
						hasJackpot := data["hasJackpot"]

						// 設置1秒後發送SHOW_LUCKY_NUMBERS命令
						go func(gameID interface{}, gameState interface{}, hasJackpot interface{}) {
							time.Sleep(1 * time.Second)

							// 建立顯示中獎號碼命令
							showLuckyNumbersCmd := Command{
								Type: "SHOW_LUCKY_NUMBERS",
								Data: map[string]interface{}{
									"game": map[string]interface{}{
										"id":         gameID,
										"state":      "SHOWING_LUCKY_NUMBERS",
										"hasJackpot": hasJackpot,
									},
									"timestamp": time.Now().Format(time.RFC3339),
								},
								Timestamp: time.Now().Format(time.RFC3339),
							}

							log.Printf("1秒計時結束，準備發送SHOW_LUCKY_NUMBERS命令")

							// 重試機制，確保命令發送成功
							maxRetries := 3
							for i := 0; i < maxRetries; i++ {
								err := client.SendCommand(showLuckyNumbersCmd)
								if err != nil {
									log.Printf("發送SHOW_LUCKY_NUMBERS失敗 (嘗試 %d/%d): %v", i+1, maxRetries, err)
									time.Sleep(1 * time.Second)
								} else {
									log.Printf("已成功發送SHOW_LUCKY_NUMBERS命令，遊戲ID: %v", gameID)
									break
								}
							}
						}(gameID, gameState, hasJackpot)
					}
				} else if msgType == "GAME_STATE_CHANGED" {
					log.Printf("收到類型為 'GAME_STATE_CHANGED' 的消息: %s", string(msg))
				} else if msgType == "ERROR" {
					log.Printf("收到類型為 'ERROR' 的消息: %s", string(msg))
				} else {
					// 檢查是否為GAME_STATE_CHANGED消息
					var stateChangedMsg map[string]interface{}
					if err := json.Unmarshal(msg, &stateChangedMsg); err == nil {
						if msgType, ok := stateChangedMsg["type"].(string); ok && msgType == "GAME_STATE_CHANGED" {
							log.Printf("收到遊戲狀態變更消息: %s", string(msg))
							continue
						}
					}

					log.Printf("收到錯誤回應: %s", msg)
					// 嘗試解析錯誤信息
					var errResponse map[string]interface{}
					if err := json.Unmarshal(msg, &errResponse); err == nil {
						if message, ok := errResponse["message"].(string); ok && message != "" {
							log.Printf("錯誤信息: %s", message)
							log.Println("建議檢查命令格式或服務器處理邏輯")
						}
					}
				}
			}

			// 檢查是否為標準Response結構
			var response Response
			if err := json.Unmarshal(msg, &response); err == nil && (response.Type != "" || response.Success) {
				if response.Success {
					if response.Type == "GAME_START_RESPONSE" || response.Type == "game_start_response" {
						log.Printf("遊戲創建成功: %s", msg)
						log.Println("遊戲已成功創建！系統已在TiDB中建立新遊戲記錄。")
						log.Println("初始階段 has_jackpot 欄位設置為 false")
					} else if response.Type == "BETTING_STARTED_RESPONSE" {
						log.Printf("收到類型為 'BETTING_STARTED_RESPONSE' 的消息: %s", string(msg))

						// 獲取遊戲ID和狀態信息
						if response.Data != nil {
							gameID := response.Data["game_id"]
							gameState := response.Data["state"]
							hasJackpot := response.Data["hasJackpot"]

							// 檢查是否已經發送過 BETTING_CLOSED 命令
							gameIDStr := ""
							if gid, ok := gameID.(string); ok {
								gameIDStr = gid
							} else if gameID != nil {
								gameIDStr = fmt.Sprintf("%v", gameID)
							}

							if gameIDStr != "" && markCommandSent(gameIDStr, "BETTING_CLOSED") {
								log.Printf("已經發送過 BETTING_CLOSED 命令，遊戲ID: %v, 跳過此次發送", gameIDStr)
								continue
							}

							// 設置5秒後發送BETTING_CLOSED命令
							go func(gameID interface{}, gameState interface{}, hasJackpot interface{}) {
								time.Sleep(5 * time.Second)

								// 創建BETTING_CLOSED命令
								bettingClosedCmd := Command{
									Type: "BETTING_CLOSED",
									Data: map[string]interface{}{
										"game": map[string]interface{}{
											"id":         gameID,
											"state":      gameState,
											"hasJackpot": hasJackpot,
										},
										"betting_summary": map[string]interface{}{
											"player_count": 15,
											"total_amount": 1500.00,
											"timestamp":    time.Now().Format(time.RFC3339),
										},
									},
									Timestamp: time.Now().Format(time.RFC3339),
								}

								log.Printf("5秒計時結束，準備發送BETTING_CLOSED命令")

								// 重試機制，確保命令發送成功
								maxRetries := 3
								for i := 0; i < maxRetries; i++ {
									err := client.SendCommand(bettingClosedCmd)
									if err != nil {
										log.Printf("發送BETTING_CLOSED失敗 (嘗試 %d/%d): %v", i+1, maxRetries, err)
										time.Sleep(1 * time.Second)
									} else {
										log.Printf("已成功發送BETTING_CLOSED命令，遊戲ID: %v", gameID)
										break
									}
								}
							}(gameID, gameState, hasJackpot)
						}
					} else if response.Type == "BETTING_CLOSED_RESPONSE" {
						log.Printf("收到類型為 'BETTING_CLOSED_RESPONSE' 的消息: %s", string(msg))

						// 獲取遊戲ID和狀態信息
						if response.Data != nil {
							gameID := response.Data["game_id"]
							gameState := response.Data["state"]
							hasJackpot := response.Data["hasJackpot"]

							// 設置2秒後發送DRAW_BALL命令
							go func(gameID interface{}, gameState interface{}, hasJackpot interface{}) {
								time.Sleep(2 * time.Second)

								// 建立抽球命令
								drawBallCmd := Command{
									Type: "DRAW_BALL",
									Data: map[string]interface{}{
										"game": map[string]interface{}{
											"id":         gameID,
											"state":      gameState,
											"hasJackpot": hasJackpot,
										},
									},
									Timestamp: time.Now().Format(time.RFC3339),
								}

								log.Printf("2秒計時結束，準備發送DRAW_BALL命令")

								// 發送命令
								err := client.SendCommand(drawBallCmd)
								if err != nil {
									log.Printf("發送DRAW_BALL命令失敗: %v", err)
								} else {
									log.Printf("已成功發送DRAW_BALL命令，遊戲ID: %v", gameID)
								}
							}(gameID, gameState, hasJackpot)
						}
					} else if response.Type == "DRAW_BALL_RESPONSE" {
						log.Printf("收到類型為 'DRAW_BALL_RESPONSE' 的消息: %s", string(msg))

						// 獲取遊戲ID和相關數據
						if response.Data != nil {
							gameID := response.Data["game_id"]
							drawnNumbers := response.Data["drawnNumbers"]
							remainingCount := response.Data["remainingCount"]

							log.Printf("遊戲ID: %v, 已抽出的號碼: %v, 剩餘號碼數: %v",
								gameID, drawnNumbers, remainingCount)

							// 模擬不斷抽球的過程
							if remainingCount != nil {
								if count, ok := remainingCount.(float64); ok && count > 0 {
									// 設置適當延遲後再次發送抽球命令
									go func(gameID interface{}, drawnNums interface{}) {
										time.Sleep(800 * time.Millisecond)

										// 建立下一次抽球命令
										nextDrawBallCmd := Command{
											Type: "DRAW_BALL",
											Data: map[string]interface{}{
												"game_id":      gameID,
												"drawnNumbers": drawnNums,
											},
											Timestamp: time.Now().Format(time.RFC3339),
										}

										// 發送命令
										err := client.SendCommand(nextDrawBallCmd)
										if err != nil {
											log.Printf("發送下一次DRAW_BALL命令失敗: %v", err)
										} else {
											log.Printf("已發送下一次DRAW_BALL命令，遊戲ID: %v", gameID)
										}
									}(gameID, drawnNumbers)
								} else if count == 0 {
									// 當所有球都抽完後，發送DRAW_RESULT命令
									go func(gameID interface{}) {
										time.Sleep(1 * time.Second)

										// 建立抽獎結果命令
										drawResultCmd := Command{
											Type: "DRAW_RESULT",
											Data: map[string]interface{}{
												"game_id":      gameID,
												"drawnNumbers": drawnNumbers,
											},
											Timestamp: time.Now().Format(time.RFC3339),
										}

										// 發送命令
										err := client.SendCommand(drawResultCmd)
										if err != nil {
											log.Printf("發送DRAW_RESULT命令失敗: %v", err)
										} else {
											log.Printf("已發送DRAW_RESULT命令，遊戲ID: %v", gameID)
										}
									}(gameID)
								}
							}
						}
					} else if response.Type == "GAME_STATE_CHANGED" {
						log.Printf("收到類型為 'GAME_STATE_CHANGED' 的消息: %s", string(msg))
					} else if response.Type == "ERROR" {
						log.Printf("收到類型為 'ERROR' 的消息: %s", string(msg))
					} else {
						// 處理其他類型的消息
						log.Printf("收到類型為 '%s' 的消息: %s", response.Type, string(msg))
					}
				} else {
					log.Printf("收到錯誤回應: %s", msg)
					if response.Message != "" {
						log.Printf("錯誤信息: %s", response.Message)
						log.Println("建議檢查命令格式或服務器處理邏輯")
					}
				}
				continue
			}

			// 其他未識別的消息
			log.Printf("收到未識別格式的消息: %s", msg)
		case err := <-errChan:
			log.Printf("發生錯誤: %v", err)
			// 客戶端會自動處理重連，這裡不需要額外操作
		case <-gameStartTimer.C:
			cmd := GameStartCommand()
			log.Printf("發送開始遊戲命令: %+v", cmd)
			log.Println("此命令將在TiDB的games表中創建新遊戲記錄，初始階段has_jackpot=false")

			err := client.SendCommand(cmd)
			if err != nil {
				log.Printf("發送命令失敗: %v", err)
				// 不需要手動重連，客戶端會自動處理
			} else {
				log.Println("命令已發送，等待回應 {\"success\": true}...")
			}
		case <-interrupt:
			log.Println("收到中斷信號，關閉連接...")
			cancel()
			client.Disconnect()
			return
		}
	}
}

// 添加一個工具函數用於美化JSON輸出
func prettyJSON(data []byte) string {
	var out bytes.Buffer
	err := json.Indent(&out, data, "", "  ")
	if err != nil {
		return string(data)
	}
	return out.String()
}

// 定期打印命令發送狀態
func printCommandStatus() {
	commandLock.Lock()
	defer commandLock.Unlock()

	log.Println("=== 命令發送狀態 ===")
	if len(bettingClosedSent) == 0 {
		log.Println("尚未發送任何被標記的命令")
		return
	}

	for key, sent := range bettingClosedSent {
		log.Printf("命令 %s 發送狀態: %v", key, sent)
	}
	log.Println("===================")
}
