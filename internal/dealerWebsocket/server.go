package dealerWebsocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/websocket"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DealerServer 代表荷官端 WebSocket 服務器
type DealerServer struct {
	config *config.AppConfig
	logger *zap.Logger
	engine *websocket.Engine
	server *http.Server
}

// NewDealerServer 創建新的荷官端 WebSocket 服務器
func NewDealerServer(
	config *config.AppConfig,
	logger *zap.Logger,
) *DealerServer {
	// 創建 WebSocket 引擎
	engine := websocket.NewEngine(logger)

	// 初始化服務器
	server := &DealerServer{
		config: config,
		logger: logger.With(zap.String("component", "dealer_websocket")),
		engine: engine,
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
