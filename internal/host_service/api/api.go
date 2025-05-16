package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"g38_lottery_service/internal/host_service/config"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 回應結構定義
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// HTTPServer HTTP 服務器結構
type HTTPServer struct {
	server *http.Server
	logger *zap.Logger
	config *config.AppConfig
}

// NewHTTPServer 創建 HTTP 服務器
func NewHTTPServer(lc fx.Lifecycle, config *config.AppConfig, logger *zap.Logger) *HTTPServer {
	mux := http.NewServeMux()

	// 設置根路由，返回 Hello World
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Code:    200,
			Message: "Hello World from Host Service!",
			Data:    map[string]interface{}{"service": config.AppName, "version": config.Server.Version},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)

		logger.Info("處理 HTTP 請求",
			zap.String("path", "/"),
			zap.String("method", "GET"),
			zap.String("remoteAddr", r.RemoteAddr))
	})

	// 創建 HTTP 服務器
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	httpServer := &HTTPServer{
		server: server,
		logger: logger,
		config: config,
	}

	// 註冊生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 啟動 HTTP 服務器
			go func() {
				logger.Info("啟動 HTTP 服務器", zap.String("addr", addr))
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP 服務器錯誤", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 關閉 HTTP 服務器
			shutdownTimeout := time.Duration(config.Server.ShutdownTimeout) * time.Second
			shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()

			logger.Info("關閉 HTTP 服務器")
			return server.Shutdown(shutdownCtx)
		},
	})

	return httpServer
}

// Module HTTP 服務模組
var Module = fx.Module("api",
	fx.Provide(
		NewHTTPServer,
	),
)
