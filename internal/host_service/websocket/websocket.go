package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"g38_lottery_service/internal/host_service/config"

	"github.com/gorilla/websocket"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WebSocketServer WebSocket 服務器結構
type WebSocketServer struct {
	server   *http.Server
	upgrader websocket.Upgrader
	logger   *zap.Logger
	config   *config.AppConfig
	clients  map[*websocket.Conn]bool
}

// NewWebSocketServer 創建 WebSocket 服務器
func NewWebSocketServer(lc fx.Lifecycle, config *config.AppConfig, logger *zap.Logger) *WebSocketServer {
	// WebSocket 升級器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允許所有來源的連接
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 創建 WebSocket 服務器
	wsServer := &WebSocketServer{
		upgrader: upgrader,
		logger:   logger,
		config:   config,
		clients:  make(map[*websocket.Conn]bool),
	}

	// 創建 HTTP 處理函數
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+config.Websocket.Path, wsServer.handleWebSocket)

	// 創建 HTTP 服務器
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Websocket.Port)
	wsServer.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 註冊生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 啟動 WebSocket 服務器
			go func() {
				logger.Info("啟動 WebSocket 服務器",
					zap.String("addr", addr),
					zap.String("path", config.Websocket.Path))
				if err := wsServer.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("WebSocket 服務器錯誤", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 關閉 WebSocket 服務器
			shutdownTimeout := time.Duration(config.Server.ShutdownTimeout) * time.Second
			shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()

			logger.Info("關閉 WebSocket 服務器")
			return wsServer.server.Shutdown(shutdownCtx)
		},
	})

	return wsServer
}

// handleWebSocket 處理 WebSocket 連接
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升級 HTTP 連接為 WebSocket 連接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("升級 WebSocket 連接失敗", zap.Error(err))
		return
	}
	defer conn.Close()

	// 將連接添加到客戶端映射
	s.clients[conn] = true
	defer delete(s.clients, conn)

	s.logger.Info("WebSocket 客戶端已連接", zap.String("remoteAddr", conn.RemoteAddr().String()))

	// 發送歡迎消息
	welcomeMsg := map[string]interface{}{
		"type":    "welcome",
		"message": "Welcome to Host Service WebSocket Server!",
		"time":    time.Now().Format(time.RFC3339),
	}

	if err := conn.WriteJSON(welcomeMsg); err != nil {
		s.logger.Error("發送歡迎消息失敗", zap.Error(err))
		return
	}

	// 讀取消息循環
	for {
		// 讀取客戶端消息
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket 讀取錯誤", zap.Error(err))
			} else {
				s.logger.Info("WebSocket 客戶端已斷開", zap.String("remoteAddr", conn.RemoteAddr().String()))
			}
			break
		}

		// 記錄收到的消息
		s.logger.Info("收到 WebSocket 消息",
			zap.Int("messageType", messageType),
			zap.String("message", string(message)))

		// 簡單的 echo 回應
		response := map[string]interface{}{
			"type":    "echo",
			"message": string(message),
			"time":    time.Now().Format(time.RFC3339),
		}

		if err := conn.WriteJSON(response); err != nil {
			s.logger.Error("發送 WebSocket 消息失敗", zap.Error(err))
			break
		}
	}
}

// Module WebSocket 服務模組
var Module = fx.Module("websocket",
	fx.Provide(
		NewWebSocketServer,
	),
)
