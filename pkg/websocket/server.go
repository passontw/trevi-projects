package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go.uber.org/fx"
)

// Server 是 WebSocket 和 HTTP API 服務器
type Server struct {
	// WebSocket 管理器
	wsManager *Manager

	// API 端口
	apiPort int

	// WebSocket 端口
	wsPort int
}

// NewServer 創建一個新的服務器
func NewServer() *Server {
	return &Server{
		wsManager: NewManager(),
		apiPort:   3000,
		wsPort:    3001,
	}
}

// StartServers 啟動 API 和 WebSocket 服務器
func (s *Server) StartServers(lc fx.Lifecycle) {
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

			// 啟動 API 服務器
			go func() {
				apiAddr := fmt.Sprintf(":%d", s.apiPort)
				log.Printf("API Server starting on %s", apiAddr)
				if err := http.ListenAndServe(apiAddr, apiMux); err != nil && err != http.ErrServerClosed {
					log.Printf("API Server error: %v", err)
				}
			}()

			// 啟動 WebSocket 服務器
			go func() {
				wsAddr := fmt.Sprintf(":%d", s.wsPort)
				log.Printf("WebSocket Server starting on %s", wsAddr)
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
	fx.Invoke(func(server *Server, lc fx.Lifecycle) {
		server.StartServers(lc)
	}),
)
