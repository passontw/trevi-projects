//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"g38_lottery_service/pkg/healthcheck"

	"go.uber.org/zap"
)

func main() {
	// 創建日誌記錄器
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("無法創建日誌記錄器: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 創建健康檢查管理器
	health := healthcheck.New(healthcheck.Config{
		Logger: logger,
	})

	// 創建 HTTP 路由
	mux := http.NewServeMux()

	// 註冊自定義路由
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// 安裝健康檢查路由
	health.InstallHandlers(mux)

	// 創建並啟動服務器
	server := healthcheck.NewServer(healthcheck.ServerConfig{
		Addr:          ":8080",
		Handler:       mux,
		HealthManager: health,
		Logger:        logger,
	})

	// 捕獲信號以優雅關閉
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// 啟動伺服器
	go func() {
		if err := server.Start(func(ctx context.Context) error {
			// 模擬初始化過程
			logger.Info("模擬初始化過程...")
			time.Sleep(2 * time.Second)

			// 標記服務為就緒
			logger.Info("設置服務為就緒...")
			health.SetReady(true)

			return nil
		}); err != nil {
			logger.Error("伺服器錯誤", zap.Error(err))
		}
	}()

	logger.Info("等待伺服器啟動...")
	if err := server.WaitForStartup(); err != nil {
		logger.Fatal("啟動失敗", zap.Error(err))
	}

	logger.Info("伺服器已啟動", zap.String("地址", ":8080"))
	logger.Info("健康檢查端點:")
	logger.Info("- 存活檢查: http://localhost:8080/liveness")
	logger.Info("- 就緒檢查: http://localhost:8080/readiness")
	logger.Info("- 健康檢查: http://localhost:8080/healthz")

	// 等待中斷信號
	<-sigCh

	logger.Info("收到關閉信號...")

	// 優雅關閉
	server.Shutdown()

	logger.Info("伺服器已關閉")
}
