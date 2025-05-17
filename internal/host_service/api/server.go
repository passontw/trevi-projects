package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"g38_lottery_service/internal/host_service/config"

	"github.com/gorilla/websocket"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Server 整合 HTTP 和 WebSocket 的服務器結構
type Server struct {
	server   *http.Server
	logger   *zap.Logger
	config   *config.AppConfig
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
}

// NewServer 創建整合的服務器
func NewServer(lc fx.Lifecycle, config *config.AppConfig, logger *zap.Logger) *Server {
	// WebSocket 升級器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允許所有來源的連接
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 創建服務器實例
	server := &Server{
		logger:   logger,
		config:   config,
		upgrader: upgrader,
		clients:  make(map[*websocket.Conn]bool),
	}

	// 創建路由器
	mux := http.NewServeMux()

	// 設置根路由 - Hello World 範例
	mux.HandleFunc("GET /", server.handleRoot)

	// 設置 WebSocket 路由
	mux.HandleFunc("GET /ws", server.handleWebSocket)

	// 創建 HTTP 服務器
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	server.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 註冊生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 啟動服務器
			go func() {
				logger.Info("啟動整合服務器",
					zap.String("addr", addr),
					zap.String("httpPath", "/"),
					zap.String("wsPath", "/ws"))
				if err := server.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("服務器錯誤", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 關閉服務器
			shutdownTimeout := time.Duration(config.Server.ShutdownTimeout) * time.Second
			shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()

			logger.Info("關閉服務器")
			return server.server.Shutdown(shutdownCtx)
		},
	})

	return server
}

// handleRoot HTTP GET 請求處理函數
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"code":    200,
		"message": "Hello World from Host Service!",
		"data":    map[string]interface{}{"service": s.config.AppName, "version": s.config.Server.Version},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	s.logger.Info("處理 HTTP 請求",
		zap.String("path", "/"),
		zap.String("method", "GET"),
		zap.String("remoteAddr", r.RemoteAddr))
}

// handleWebSocket WebSocket 連接處理函數
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
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

// BroadcastMessage 向所有連接的客戶端廣播消息
func (s *Server) BroadcastMessage(message interface{}) {
	for client := range s.clients {
		if err := client.WriteJSON(message); err != nil {
			s.logger.Error("廣播消息失敗", zap.Error(err))
			client.Close()
			delete(s.clients, client)
		}
	}
}

// Module HTTP 和 WebSocket 服務模組
var Module = fx.Module("api",
	fx.Provide(
		NewServer,
	),
)
