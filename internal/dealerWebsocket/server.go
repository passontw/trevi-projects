package dealerWebsocket

import (
	"context"
	"fmt"
	"net/http"
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
}

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
