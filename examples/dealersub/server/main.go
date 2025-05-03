package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// 定義 WebSocket 升級器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允許所有來源的連線，生產環境中應限制
	},
}

// 請求結構
type Request struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// 回應結構
type Response struct {
	Type      string                 `json:"type"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// 訂閱資料的結構
type Subscription struct {
	Topic string      `json:"topic"`
	Data  interface{} `json:"data,omitempty"`
}

// 訂閱請求
type SubscribeData struct {
	Topic string `json:"topic"`
}

// 客戶端連線
type Client struct {
	ID            string
	Conn          *websocket.Conn
	Send          chan []byte
	Subscriptions map[string]bool
	mu            sync.Mutex
}

// 服務器管理所有連線和訂閱
type Server struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	topics     map[string]map[string]*Client // 每個主題的訂閱者
	mu         sync.Mutex
}

// 建立新的服務器實例
func NewServer() *Server {
	return &Server{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		topics:     make(map[string]map[string]*Client),
	}
}

// 處理新的 WebSocket 連線
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升級 HTTP 連線到 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("升級連線失敗: %v", err)
		return
	}

	// 為每個客戶端分配唯一 ID
	clientID := uuid.New().String()
	client := &Client{
		ID:            clientID,
		Conn:          conn,
		Send:          make(chan []byte, 256),
		Subscriptions: make(map[string]bool),
	}

	// 註冊新客戶端
	s.register <- client

	// 發送歡迎訊息
	welcomeResp := Response{
		Type:      "WELCOME",
		Success:   true,
		Message:   "歡迎連線到訂閱服務器",
		Data:      map[string]interface{}{"clientId": clientID},
		Timestamp: time.Now().Unix(),
	}

	welcomeMsg, _ := json.Marshal(welcomeResp)
	client.Send <- welcomeMsg

	// 啟動客戶端讀寫協程
	go s.readPump(client)
	go s.writePump(client)
}

// 處理接收客戶端訊息
func (s *Server) readPump(client *Client) {
	defer func() {
		s.unregister <- client
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(512 * 1024) // 設定讀取限制為 512KB
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("讀取錯誤: %v", err)
			}
			break
		}

		// 解析請求
		var request Request
		if err := json.Unmarshal(message, &request); err != nil {
			log.Printf("解析請求失敗: %v", err)
			continue
		}

		// 處理不同類型的請求
		switch request.Type {
		case "SUBSCRIBE":
			s.handleSubscribe(client, message, request)
		case "UNSUBSCRIBE":
			s.handleUnsubscribe(client, message, request)
		case "PUBLISH":
			s.handlePublish(client, message, request)
		default:
			log.Printf("未知請求類型: %s", request.Type)
			response := Response{
				Type:      "ERROR",
				Success:   false,
				Message:   "未知請求類型",
				Data:      map[string]interface{}{"requestType": request.Type},
				Timestamp: time.Now().Unix(),
			}
			responseMsg, _ := json.Marshal(response)
			client.Send <- responseMsg
		}
	}
}

// 處理訂閱請求
func (s *Server) handleSubscribe(client *Client, message []byte, request Request) {
	// 將請求資料解析為訂閱資料
	var subscribeData SubscribeData

	// 嘗試從請求的 Data 欄位獲取訂閱資料
	if data, ok := request.Data.(map[string]interface{}); ok {
		if topic, ok := data["topic"].(string); ok {
			subscribeData.Topic = topic
		}
	} else {
		dataBytes, _ := json.Marshal(request.Data)
		if err := json.Unmarshal(dataBytes, &subscribeData); err != nil {
			log.Printf("解析訂閱資料失敗: %v", err)

			// 發送錯誤回應
			errorResp := Response{
				Type:      "ERROR",
				Success:   false,
				Message:   "無效的訂閱資料",
				Data:      map[string]interface{}{},
				Timestamp: time.Now().Unix(),
			}
			errorMsg, _ := json.Marshal(errorResp)
			client.Send <- errorMsg
			return
		}
	}

	// 檢查主題是否為空
	if subscribeData.Topic == "" {
		errorResp := Response{
			Type:      "ERROR",
			Success:   false,
			Message:   "訂閱主題不能為空",
			Data:      map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		errorMsg, _ := json.Marshal(errorResp)
		client.Send <- errorMsg
		return
	}

	// 訂閱主題
	s.mu.Lock()
	if _, ok := s.topics[subscribeData.Topic]; !ok {
		s.topics[subscribeData.Topic] = make(map[string]*Client)
	}
	s.topics[subscribeData.Topic][client.ID] = client
	s.mu.Unlock()

	// 更新客戶端訂閱列表
	client.mu.Lock()
	client.Subscriptions[subscribeData.Topic] = true
	client.mu.Unlock()

	// 發送訂閱成功回應
	response := Response{
		Type:      "SUBSCRIBED",
		Success:   true,
		Message:   "訂閱成功",
		Data:      map[string]interface{}{"topic": subscribeData.Topic},
		Timestamp: time.Now().Unix(),
	}
	responseMsg, _ := json.Marshal(response)
	client.Send <- responseMsg

	log.Printf("客戶端 %s 訂閱了主題 %s", client.ID, subscribeData.Topic)
}

// 處理取消訂閱請求
func (s *Server) handleUnsubscribe(client *Client, message []byte, request Request) {
	// 將請求資料解析為訂閱資料
	var unsubscribeData SubscribeData

	// 嘗試從請求的 Data 欄位獲取取消訂閱資料
	if data, ok := request.Data.(map[string]interface{}); ok {
		if topic, ok := data["topic"].(string); ok {
			unsubscribeData.Topic = topic
		}
	} else {
		dataBytes, _ := json.Marshal(request.Data)
		if err := json.Unmarshal(dataBytes, &unsubscribeData); err != nil {
			log.Printf("解析取消訂閱資料失敗: %v", err)

			// 發送錯誤回應
			errorResp := Response{
				Type:      "ERROR",
				Success:   false,
				Message:   "無效的取消訂閱資料",
				Data:      map[string]interface{}{},
				Timestamp: time.Now().Unix(),
			}
			errorMsg, _ := json.Marshal(errorResp)
			client.Send <- errorMsg
			return
		}
	}

	// 檢查主題是否為空
	if unsubscribeData.Topic == "" {
		errorResp := Response{
			Type:      "ERROR",
			Success:   false,
			Message:   "取消訂閱主題不能為空",
			Data:      map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		errorMsg, _ := json.Marshal(errorResp)
		client.Send <- errorMsg
		return
	}

	// 檢查客戶端是否訂閱了該主題
	client.mu.Lock()
	if !client.Subscriptions[unsubscribeData.Topic] {
		client.mu.Unlock()
		errorResp := Response{
			Type:      "ERROR",
			Success:   false,
			Message:   "未訂閱該主題",
			Data:      map[string]interface{}{"topic": unsubscribeData.Topic},
			Timestamp: time.Now().Unix(),
		}
		errorMsg, _ := json.Marshal(errorResp)
		client.Send <- errorMsg
		return
	}
	client.mu.Unlock()

	// 取消訂閱主題
	s.mu.Lock()
	if clients, ok := s.topics[unsubscribeData.Topic]; ok {
		delete(clients, client.ID)
		// 如果該主題沒有訂閱者了，刪除主題
		if len(clients) == 0 {
			delete(s.topics, unsubscribeData.Topic)
		}
	}
	s.mu.Unlock()

	// 更新客戶端訂閱列表
	client.mu.Lock()
	delete(client.Subscriptions, unsubscribeData.Topic)
	client.mu.Unlock()

	// 發送取消訂閱成功回應
	response := Response{
		Type:      "UNSUBSCRIBED",
		Success:   true,
		Message:   "取消訂閱成功",
		Data:      map[string]interface{}{"topic": unsubscribeData.Topic},
		Timestamp: time.Now().Unix(),
	}
	responseMsg, _ := json.Marshal(response)
	client.Send <- responseMsg

	log.Printf("客戶端 %s 取消訂閱主題 %s", client.ID, unsubscribeData.Topic)
}

// 處理發布請求
func (s *Server) handlePublish(client *Client, message []byte, request Request) {
	// 解析發布資料
	type PublishData struct {
		Topic string      `json:"topic"`
		Data  interface{} `json:"data"`
	}
	var publishData PublishData

	// 嘗試從請求的 Data 欄位獲取發布資料
	if data, ok := request.Data.(map[string]interface{}); ok {
		if topic, ok := data["topic"].(string); ok {
			publishData.Topic = topic
		}
		if msgData, ok := data["data"]; ok {
			publishData.Data = msgData
		}
	} else {
		dataBytes, _ := json.Marshal(request.Data)
		if err := json.Unmarshal(dataBytes, &publishData); err != nil {
			log.Printf("解析發布資料失敗: %v", err)

			// 發送錯誤回應
			errorResp := Response{
				Type:      "ERROR",
				Success:   false,
				Message:   "無效的發布資料",
				Data:      map[string]interface{}{},
				Timestamp: time.Now().Unix(),
			}
			errorMsg, _ := json.Marshal(errorResp)
			client.Send <- errorMsg
			return
		}
	}

	// 檢查主題是否為空
	if publishData.Topic == "" {
		errorResp := Response{
			Type:      "ERROR",
			Success:   false,
			Message:   "發布主題不能為空",
			Data:      map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		errorMsg, _ := json.Marshal(errorResp)
		client.Send <- errorMsg
		return
	}

	// 發送消息給訂閱該主題的所有客戶端
	s.publishToTopic(publishData.Topic, publishData.Data)

	// 發送發布成功回應
	response := Response{
		Type:      "PUBLISHED",
		Success:   true,
		Message:   "發布成功",
		Data:      map[string]interface{}{"topic": publishData.Topic},
		Timestamp: time.Now().Unix(),
	}
	responseMsg, _ := json.Marshal(response)
	client.Send <- responseMsg

	log.Printf("客戶端 %s 向主題 %s 發布了訊息", client.ID, publishData.Topic)
}

// 向主題發布訊息
func (s *Server) publishToTopic(topic string, data interface{}) {
	// 建立要發送的訊息
	message := Response{
		Type:      "MESSAGE",
		Success:   true,
		Message:   "收到訊息",
		Data:      map[string]interface{}{"topic": topic, "data": data},
		Timestamp: time.Now().Unix(),
	}
	messageBytes, _ := json.Marshal(message)

	// 向所有訂閱該主題的客戶端發送訊息
	s.mu.Lock()
	clients, ok := s.topics[topic]
	s.mu.Unlock()

	if ok {
		for _, client := range clients {
			select {
			case client.Send <- messageBytes:
				// 成功發送
			default:
				// 客戶端緩衝區已滿，關閉連線
				s.unregister <- client
				client.Conn.Close()
			}
		}
	}
}

// 發送訊息至客戶端
func (s *Server) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已關閉
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 在一次寫入中發送所有排隊訊息
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// 運行服務器
func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			// 註冊新客戶端
			s.clients[client.ID] = client
			log.Printf("新客戶端註冊: %s", client.ID)
		case client := <-s.unregister:
			// 取消註冊客戶端
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				close(client.Send)
				log.Printf("客戶端斷開: %s", client.ID)

				// 從所有主題中移除該客戶端
				s.mu.Lock()
				for topic, clients := range s.topics {
					if _, ok := clients[client.ID]; ok {
						delete(clients, client.ID)
						// 如果該主題沒有訂閱者了，刪除主題
						if len(clients) == 0 {
							delete(s.topics, topic)
						}
					}
				}
				s.mu.Unlock()
			}
		case message := <-s.broadcast:
			// 廣播訊息給所有客戶端
			for _, client := range s.clients {
				select {
				case client.Send <- message:
					// 成功發送
				default:
					// 客戶端緩衝區已滿，關閉連線
					close(client.Send)
					delete(s.clients, client.ID)
				}
			}
		}
	}
}

// 每5秒向特定主題發送一條訊息
func (s *Server) sendPeriodicMessages() {
	ticker := time.NewTicker(5 * time.Second)
	topic := "game_events"

	for {
		select {
		case <-ticker.C:
			// 發送定時訊息
			timeStr := time.Now().Format(time.RFC3339)
			message := map[string]interface{}{
				"message":   fmt.Sprintf("伺服器定時訊息 - %s", timeStr),
				"timestamp": timeStr,
			}

			log.Printf("定時向主題 %s 發送訊息", topic)
			s.publishToTopic(topic, message)
		}
	}
}

func main() {
	// 創建新的服務器實例
	server := NewServer()

	// 啟動服務器
	go server.run()

	// 啟動定時發送訊息
	go server.sendPeriodicMessages()

	// 設置 WebSocket 路由
	http.HandleFunc("/ws", server.handleWebSocket)

	// 設置靜態文件服務
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// 設置監聽端口
	port := "9000"
	log.Printf("啟動訂閱服務器，監聽端口: %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
