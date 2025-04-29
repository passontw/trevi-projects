package playerWebsocket

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

// PlayerServer 代表玩家端 WebSocket 服務器
type PlayerServer struct {
	config *config.AppConfig
	logger *zap.Logger
	engine *websocket.Engine
	server *http.Server
}

// NewPlayerServer 創建新的玩家端 WebSocket 服務器
func NewPlayerServer(
	config *config.AppConfig,
	logger *zap.Logger,
) *PlayerServer {
	// 創建 WebSocket 引擎
	engine := websocket.NewEngine(logger)

	// 初始化服務器
	server := &PlayerServer{
		config: config,
		logger: logger.With(zap.String("component", "player_websocket")),
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
				"message": "歡迎連接到開獎服務玩家端 (Player WebSocket)",
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
func (s *PlayerServer) registerHandlers() {
	// 處理 ping 訊息
	s.engine.RegisterHandler("ping", func(client *websocket.Client, message websocket.Message) error {
		return client.SendJSON(websocket.Response{
			Type:    "pong",
			Payload: map[string]string{"time": time.Now().Format(time.RFC3339)},
		})
	})

	// 處理玩家訂閱開獎結果
	s.engine.RegisterHandler("subscribe", func(client *websocket.Client, message websocket.Message) error {
		// 從訊息中獲取訂閱的遊戲ID
		gameID, ok := message.Payload["game_id"]
		if !ok {
			return fmt.Errorf("missing game_id in subscribe message")
		}

		// 將遊戲ID設定為客戶端的元數據
		client.SetMetadata("game_id", gameID)

		s.logger.Info("Player subscribed to game",
			zap.String("clientID", client.ID),
			zap.Any("gameID", gameID))

		// 回應訂閱成功
		return client.SendJSON(websocket.Response{
			Type: "subscribe_success",
			Payload: map[string]interface{}{
				"game_id": gameID,
				"message": "Successfully subscribed to game updates",
			},
		})
	})

	// 處理玩家取消訂閱
	s.engine.RegisterHandler("unsubscribe", func(client *websocket.Client, message websocket.Message) error {
		// 從客戶端元數據中移除遊戲ID
		client.SetMetadata("game_id", nil)

		s.logger.Info("Player unsubscribed from game", zap.String("clientID", client.ID))

		// 回應取消訂閱成功
		return client.SendJSON(websocket.Response{
			Type:    "unsubscribe_success",
			Payload: map[string]string{"message": "Successfully unsubscribed from game updates"},
		})
	})
}

// BroadcastLotteryResult 廣播開獎結果給訂閱特定遊戲的玩家
func (s *PlayerServer) BroadcastLotteryResult(gameID string, result interface{}) error {
	return s.engine.BroadcastFilter(
		websocket.Response{
			Type: "lottery_result",
			Payload: map[string]interface{}{
				"game_id": gameID,
				"result":  result,
				"time":    time.Now().Format(time.RFC3339),
			},
		},
		func(client *websocket.Client) bool {
			// 檢查客戶端是否訂閱了該遊戲
			if subscribeGameID, ok := client.GetMetadata("game_id"); ok {
				return subscribeGameID == gameID
			}
			return false
		},
	)
}

// Start 啟動玩家端 WebSocket 服務器
func (s *PlayerServer) Start(lc fx.Lifecycle) {
	// 使用應用配置中的玩家 WebSocket 端口
	serverAddr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.PlayerWsPort)
	s.logger.Info("Starting player WebSocket server",
		zap.String("address", serverAddr),
		zap.Int("port", s.config.Server.PlayerWsPort))

	// 建立 ServeMux
	mux := http.NewServeMux()

	// 註冊 WebSocket 端點
	mux.HandleFunc("GET /ws", s.engine.HandleConnection)

	// 註冊其他端點
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Player WebSocket Server is running. Connect to /ws endpoint."))
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
				s.logger.Info("Player WebSocket server listening", zap.String("address", serverAddr))
				if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					s.logger.Error("Player WebSocket server failed", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.logger.Info("Stopping player WebSocket server")
			return s.server.Shutdown(ctx)
		},
	})
}

// Module 提供 FX 模塊
var Module = fx.Options(
	fx.Provide(NewPlayerServer),
	fx.Invoke(func(server *PlayerServer, lc fx.Lifecycle) {
		server.Start(lc)
	}),
)
