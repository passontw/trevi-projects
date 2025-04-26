package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"g38_lottery_service/internal/config"

	"go.uber.org/fx"
)

// Server 是 WebSocket 和 HTTP API 服務器
type Server struct {
	// WebSocket 管理器
	wsManager *Manager

	// 配置
	config *config.Config
}

// NewServer 創建一個新的服務器
func NewServer(config *config.Config) *Server {
	return &Server{
		wsManager: NewManager(),
		config:    config,
	}
}

// StartServers 啟動 API 和 WebSocket 服務器
func (s *Server) StartServers(lc fx.Lifecycle) {
	// 從配置中讀取端口
	apiPort := s.config.Server.Port
	playerWsPort := s.config.Server.PlayerWSPort

	// 檢查端口是否與現有 API 相同，如相同則跳過 API 服務器啟動
	skipAPIServer := apiPort == 3000 // 假設 3000 是 Gin 使用的端口

	// API 服務器
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello World API Server"))
	})

	apiMux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello World from API!"}`))
	})

	// WebSocket 服務器
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/ws", s.wsManager.ServeWs)

	// 用於健康檢查的端點
	wsMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("WebSocket Server is running"))
	})

	// 為 fx 生命週期掛鉤添加處理程序
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 啟動 WebSocket 管理器
			go s.wsManager.Start(ctx)

			// 啟動 API 服務器 (如果不跳過)
			if !skipAPIServer {
				go func() {
					apiAddr := fmt.Sprintf(":%d", apiPort)
					log.Printf("API Server starting on %s", apiAddr)
					if err := http.ListenAndServe(apiAddr, apiMux); err != nil && err != http.ErrServerClosed {
						log.Printf("API Server error: %v", err)
					}
				}()
			} else {
				log.Printf("跳過啟動 API 服務器，因為端口 %d 已被其他服務使用", apiPort)
			}

			// 啟動 WebSocket 服務器
			go func() {
				wsAddr := fmt.Sprintf(":%d", playerWsPort)
				log.Printf("玩家 WebSocket 服務器正在啟動，監聽端口 %s", wsAddr)
				if err := http.ListenAndServe(wsAddr, wsMux); err != nil && err != http.ErrServerClosed {
					log.Printf("WebSocket Server error: %v", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 關閉 WebSocket 管理器
			s.wsManager.Shutdown()
			log.Println("Servers shutting down")
			return nil
		},
	})
}

// Module 是 fx 模組
var Module = fx.Options(
	fx.Provide(NewServer),
	fx.Invoke(func(server *Server, lc fx.Lifecycle, cfg *config.Config) {
		server.StartServers(lc)
	}),
)
