package dealerWebsocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/gameflow"
	"g38_lottery_service/internal/websocket"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DealerServer 代表荷官端 WebSocket 服務器
type DealerServer struct {
	config      *config.AppConfig
	logger      *zap.Logger
	engine      *websocket.Engine
	server      *http.Server
	gameManager *gameflow.GameManager
	topics      map[string]map[string]*websocket.Client // 每個主題的訂閱者
	mu          sync.Mutex
}

// NewDealerServer 創建新的荷官端 WebSocket 服務器
func NewDealerServer(
	config *config.AppConfig,
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
) *DealerServer {
	// 創建 WebSocket 引擎
	engine := websocket.NewEngine(logger)

	// 初始化服務器
	server := &DealerServer{
		config:      config,
		logger:      logger.With(zap.String("component", "dealer_websocket")),
		engine:      engine,
		gameManager: gameManager,
		topics:      make(map[string]map[string]*websocket.Client), // 初始化主題映射
	}

	// 註冊訊息處理函數
	server.registerHandlers()

	// 設置連接回調處理函數
	server.engine.SetOnConnectHandler(func(client *websocket.Client) {
		// 發送 HelloResponse 到客戶端
		err := client.SendJSON(websocket.Response{
			Type: "hello",
			Payload: map[string]interface{}{
				"message": "歡迎連接到開獎服務荷官端 (Dealer WebSocket)",
			},
		})

		if err != nil {
			server.logger.Error("Failed to send hello message to client",
				zap.String("clientID", client.ID),
				zap.Error(err))
		} else {
			server.logger.Info("Sent hello message to client",
				zap.String("clientID", client.ID))
		}
	})

	// 啟動定期發送 game_events 消息
	// go server.sendPeriodicGameEvents()

	return server
}

// 註冊訊息處理函數
func (s *DealerServer) registerHandlers() {
	// 處理 ping 訊息
	s.engine.RegisterHandler("ping", func(client *websocket.Client, message websocket.Message) error {
		return client.SendJSON(websocket.Response{
			Type:    "pong",
			Payload: map[string]string{"time": time.Now().Format(time.RFC3339)},
		})
	})

	// 處理荷官開獎訊息
	s.engine.RegisterHandler("draw_lottery", func(client *websocket.Client, message websocket.Message) error {
		// 從訊息中獲取開獎數據
		result, ok := message.Payload["result"]
		if !ok {
			return fmt.Errorf("missing result in draw_lottery message")
		}

		// 獲取遊戲ID，如果沒有提供則使用默認值
		gameID, ok := message.Payload["game_id"].(string)
		if !ok {
			gameID = "default_game" // 默認遊戲ID
			s.logger.Warn("No game_id provided in draw_lottery, using default",
				zap.String("clientID", client.ID))
		}

		// 記錄開獎結果
		s.logger.Info("Dealer drew lottery",
			zap.String("clientID", client.ID),
			zap.String("gameID", gameID),
			zap.Any("result", result))

		// 廣播開獎結果給所有連接的荷官
		return s.engine.Broadcast(websocket.Response{
			Type: "lottery_result",
			Payload: map[string]interface{}{
				"game_id": gameID,
				"result":  result,
				"time":    time.Now().Format(time.RFC3339),
			},
		})
	})

	// 處理開始新局請求
	s.engine.RegisterHandler("START_NEW_ROUND", func(client *websocket.Client, message websocket.Message) error {
		s.logger.Info("收到開始新局請求",
			zap.String("clientID", client.ID),
			zap.Any("payload", message.Payload))

		// 創建上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 檢查當前階段是否為準備階段
		currentStage := s.gameManager.GetCurrentStage()
		if currentStage != gameflow.StagePreparation && currentStage != gameflow.StageGameOver {
			errorMessage := fmt.Sprintf("無法開始新局，當前階段不是準備階段或遊戲結束階段。當前階段: %s", string(currentStage))
			s.logger.Warn(errorMessage, zap.String("clientID", client.ID))

			// 返回錯誤回應
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "START_NEW_ROUND",
					"message": errorMessage,
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		// 創建新遊戲
		gameID, err := s.gameManager.CreateNewGame(ctx)
		if err != nil {
			s.logger.Error("創建新遊戲失敗",
				zap.String("clientID", client.ID),
				zap.Error(err))

			// 返回錯誤回應
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "START_NEW_ROUND",
					"message": fmt.Sprintf("創建新遊戲失敗: %v", err),
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		// 獲取當前遊戲數據
		game := s.gameManager.GetCurrentGame()
		if game == nil {
			errorMessage := "獲取新創建的遊戲失敗"
			s.logger.Error(errorMessage, zap.String("clientID", client.ID))

			// 返回錯誤回應
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "START_NEW_ROUND",
					"message": errorMessage,
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		// 推進到新局階段
		err = s.gameManager.AdvanceStage(ctx, true)
		if err != nil {
			s.logger.Error("推進到新局階段失敗",
				zap.String("clientID", client.ID),
				zap.Error(err))

			// 返回錯誤回應
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "START_NEW_ROUND",
					"message": fmt.Sprintf("推進到新局階段失敗: %v", err),
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		// 重新獲取更新後的遊戲數據
		game = s.gameManager.GetCurrentGame()

		// 返回成功回應
		s.logger.Info("成功開始新局",
			zap.String("clientID", client.ID),
			zap.String("gameID", gameID),
			zap.String("stage", string(game.CurrentStage)))

		return client.SendJSON(websocket.Response{
			Type: "response",
			Payload: map[string]interface{}{
				"success":   true,
				"type":      "START_NEW_ROUND",
				"game_id":   gameID,
				"stage":     string(game.CurrentStage),
				"timestamp": game.StartTime.Format(time.RFC3339),
				"time":      time.Now().Format(time.RFC3339),
			},
		})
	})

	// 處理客戶端斷開連接
	s.engine.RegisterHandler("__disconnect", func(client *websocket.Client, message websocket.Message) error {
		// 從所有訂閱主題中移除客戶端
		s.mu.Lock()
		defer s.mu.Unlock()

		for topic, clients := range s.topics {
			delete(clients, client.ID)
			// 如果主題沒有訂閱者，則刪除該主題
			if len(clients) == 0 {
				delete(s.topics, topic)
			}
		}

		s.logger.Info("客戶端斷開連接，已從所有訂閱主題中移除",
			zap.String("clientID", client.ID))
		return nil
	})

	// 處理訂閱主題請求
	s.engine.RegisterHandler("SUBSCRIBE", func(client *websocket.Client, message websocket.Message) error {
		// 從訊息中獲取訂閱主題
		topicData, ok := message.Payload["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("missing or invalid data in SUBSCRIBE message")
		}

		topic, ok := topicData["topic"].(string)
		if !ok || topic == "" {
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "SUBSCRIBED",
					"message": "訂閱主題不能為空",
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		// 添加客戶端到主題訂閱者列表
		s.mu.Lock()
		if _, exists := s.topics[topic]; !exists {
			s.topics[topic] = make(map[string]*websocket.Client)
		}
		s.topics[topic][client.ID] = client
		s.mu.Unlock()

		s.logger.Info("客戶端訂閱主題",
			zap.String("clientID", client.ID),
			zap.String("topic", topic))

		// 發送訂閱成功回應
		return client.SendJSON(websocket.Response{
			Type: "response",
			Payload: map[string]interface{}{
				"success": true,
				"type":    "SUBSCRIBED",
				"topic":   topic,
				"message": "訂閱成功",
				"time":    time.Now().Format(time.RFC3339),
			},
		})
	})

	// 處理取消訂閱主題請求
	s.engine.RegisterHandler("UNSUBSCRIBE", func(client *websocket.Client, message websocket.Message) error {
		// 從訊息中獲取要取消訂閱的主題
		topicData, ok := message.Payload["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("missing or invalid data in UNSUBSCRIBE message")
		}

		topic, ok := topicData["topic"].(string)
		if !ok || topic == "" {
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "UNSUBSCRIBED",
					"message": "取消訂閱主題不能為空",
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		// 檢查客戶端是否訂閱了該主題
		s.mu.Lock()
		defer s.mu.Unlock()

		clients, topicExists := s.topics[topic]
		if !topicExists || clients[client.ID] == nil {
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "UNSUBSCRIBED",
					"message": "未訂閱該主題",
					"topic":   topic,
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		// 從主題訂閱者列表中移除客戶端
		delete(clients, client.ID)
		if len(clients) == 0 {
			delete(s.topics, topic)
		}

		s.logger.Info("客戶端取消訂閱主題",
			zap.String("clientID", client.ID),
			zap.String("topic", topic))

		// 發送取消訂閱成功回應
		return client.SendJSON(websocket.Response{
			Type: "response",
			Payload: map[string]interface{}{
				"success": true,
				"type":    "UNSUBSCRIBED",
				"topic":   topic,
				"message": "取消訂閱成功",
				"time":    time.Now().Format(time.RFC3339),
			},
		})
	})

	// 處理發布訊息請求
	s.engine.RegisterHandler("PUBLISH", func(client *websocket.Client, message websocket.Message) error {
		// 從訊息中獲取要發布的主題和數據
		publishData, ok := message.Payload["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("missing or invalid data in PUBLISH message")
		}

		topic, ok := publishData["topic"].(string)
		if !ok || topic == "" {
			return client.SendJSON(websocket.Response{
				Type: "response",
				Payload: map[string]interface{}{
					"success": false,
					"type":    "PUBLISHED",
					"message": "發布主題不能為空",
					"time":    time.Now().Format(time.RFC3339),
				},
			})
		}

		data, ok := publishData["data"]
		if !ok {
			data = "" // 如果沒有提供數據，使用空字串
		}

		// 向主題發布訊息
		s.publishToTopic(topic, data)

		s.logger.Info("客戶端發布訊息至主題",
			zap.String("clientID", client.ID),
			zap.String("topic", topic))

		// 發送發布成功回應
		return client.SendJSON(websocket.Response{
			Type: "response",
			Payload: map[string]interface{}{
				"success": true,
				"type":    "PUBLISHED",
				"topic":   topic,
				"message": "發布成功",
				"time":    time.Now().Format(time.RFC3339),
			},
		})
	})
}

// 向主題發布訊息
func (s *DealerServer) publishToTopic(topic string, data interface{}) {
	// 建立要發送的訊息
	message := websocket.Response{
		Type: "MESSAGE",
		Payload: map[string]interface{}{
			"topic":     topic,
			"data":      data,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	// 向所有訂閱該主題的客戶端發送訊息
	s.mu.Lock()
	clients, ok := s.topics[topic]
	s.mu.Unlock()

	if ok && len(clients) > 0 {
		for _, client := range clients {
			err := client.SendJSON(message)
			if err != nil {
				s.logger.Error("向客戶端發送主題訊息失敗",
					zap.String("clientID", client.ID),
					zap.String("topic", topic),
					zap.Error(err))
			}
		}
		s.logger.Info("已向主題訂閱者發送訊息",
			zap.String("topic", topic),
			zap.Int("subscribers", len(clients)))
	} else {
		s.logger.Info("主題沒有訂閱者，訊息未發送",
			zap.String("topic", topic))
	}
}

// 每5秒向 game_events 主題發送一條訊息
// func (s *DealerServer) sendPeriodicGameEvents() {
// 	ticker := time.NewTicker(5 * time.Second)
// 	topic := "game_events"

// 	for {
// 		select {
// 		case <-ticker.C:
// 			// 獲取當前遊戲數據
// 			game := s.gameManager.GetCurrentGame()
// 			currentTime := time.Now()
// 			timeStr := currentTime.Format(time.RFC3339)

// 			// 構建遊戲事件訊息
// 			var eventData map[string]interface{}

// 			if game != nil {
// 				eventData = map[string]interface{}{
// 					"message":       fmt.Sprintf("遊戲事件訊息 - %s", timeStr),
// 					"timestamp":     timeStr,
// 					"game_id":       game.GameID,
// 					"current_stage": string(game.CurrentStage),
// 					"start_time":    game.StartTime.Format(time.RFC3339),
// 				}
// 			} else {
// 				eventData = map[string]interface{}{
// 					"message":   fmt.Sprintf("尚無活動遊戲 - %s", timeStr),
// 					"timestamp": timeStr,
// 					"status":    "no_active_game",
// 				}
// 			}

// 			s.logger.Info("定時向 game_events 主題發送訊息")
// 			s.publishToTopic(topic, eventData)
// 		}
// 	}
// }

// RegisterExternalHandler 註冊外部處理函數
func (s *DealerServer) RegisterExternalHandler(messageType string, handler websocket.MessageHandler) {
	s.engine.RegisterHandler(messageType, handler)
	s.logger.Info("Registered external handler for message type", zap.String("type", messageType))
}

// Start 啟動荷官端 WebSocket 服務器
func (s *DealerServer) Start(lc fx.Lifecycle) {
	// 使用應用配置中的荷官 WebSocket 端口
	serverAddr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.DealerWsPort)
	s.logger.Info("Starting dealer WebSocket server",
		zap.String("address", serverAddr),
		zap.Int("port", s.config.Server.DealerWsPort))

	// 建立 ServeMux
	mux := http.NewServeMux()

	// 註冊 WebSocket 端點
	mux.HandleFunc("GET /ws", s.engine.HandleConnection)

	// 註冊其他端點
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Dealer WebSocket Server is running. Connect to /ws endpoint."))
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"UP"}`))
	})

	// 建立 HTTP 服務器
	s.server = &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	// 啟動 WebSocket 引擎
	s.engine.Start()

	// 生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 啟動 HTTP 服務器
			go func() {
				s.logger.Info("Dealer WebSocket server listening", zap.String("address", serverAddr))
				if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					s.logger.Error("Dealer WebSocket server failed", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.logger.Info("Stopping dealer WebSocket server")
			return s.server.Shutdown(ctx)
		},
	})
}

// Module 提供 FX 模塊
var Module = fx.Options(
	fx.Provide(NewDealerServer),
	fx.Invoke(func(server *DealerServer, lc fx.Lifecycle) {
		server.Start(lc)
	}),
)

// 添加獲取引擎方法
func (s *DealerServer) GetEngine() *websocket.Engine {
	return s.engine
}

// BroadcastMessage 廣播消息到所有連接的客戶端
func (s *DealerServer) BroadcastMessage(message interface{}) error {
	return s.engine.Broadcast(websocket.Response{
		Type:    "event",
		Payload: message,
	})
}

// PublishToTopic 對外暴露向主題發布訊息的方法
func (s *DealerServer) PublishToTopic(topic string, data interface{}) {
	s.publishToTopic(topic, data)
}

// GetTopicSubscribers 獲取特定主題的訂閱者數量
func (s *DealerServer) GetTopicSubscribers(topic string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.topics[topic]; ok {
		return len(clients)
	}
	return 0
}
