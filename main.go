package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/dealerWebsocket"
	"g38_lottery_service/internal/playerWebsocket"
	"g38_lottery_service/internal/service"
	"g38_lottery_service/pkg/nacosManager"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 主函數：使用 fx 框架
func main() {
	// 建立 logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	app := fx.New(
		// 註冊 Nacos 模塊
		nacosManager.Module,
		// 註冊配置模塊
		config.Module,
		// 註冊荷官端 WebSocket 模塊
		dealerWebsocket.Module,
		// 註冊玩家端 WebSocket 模塊
		playerWebsocket.Module,
		// 註冊開獎服務模塊
		service.Module,

		fx.Provide(
			func() *zap.Logger {
				return logger
			},
		),
	)

	// 啟動應用
	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	// 等待系統信號
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// 關閉應用
	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Stop(stopCtx); err != nil {
		logger.Fatal("Failed to stop application gracefully", zap.Error(err))
	}

	logger.Info("Application stopped gracefully")
}
