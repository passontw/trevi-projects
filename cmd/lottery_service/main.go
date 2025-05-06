package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/internal/lottery_service/dealerWebsocket"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/internal/lottery_service/grpc"
	"g38_lottery_service/internal/lottery_service/mq"
	"g38_lottery_service/internal/lottery_service/service"
	"g38_lottery_service/pkg/nacosManager"
	redis "g38_lottery_service/pkg/redisManager"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 主函數：使用 fx 框架
// 測試 air 熱重載功能
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
		// 註冊 Redis 模塊
		redis.Module,
		// 註冊 RocketMQ 生產者模塊
		mq.Module,
		// 註冊荷官端 WebSocket 模塊
		dealerWebsocket.Module,
		// 註冊遊戲流程管理模塊
		gameflow.Module,
		// 註冊開獎服務模塊
		service.Module,
		// 註冊 gRPC 服務模塊
		grpc.Module,

		fx.Provide(
			func() *zap.Logger {
				return logger
			},
		),

		// 新增：設置 GameManager 的 onStageChanged 事件，推送 game_events
		fx.Invoke(func(gm *gameflow.GameManager, ds *dealerWebsocket.DealerServer) {
			gm.SetEventHandlers(
				func(gameID string, oldStage, newStage gameflow.GameStage) {
					game := gm.GetCurrentGame()
					if game == nil {
						return
					}
					status := gameflow.BuildGameStatusResponse(game)
					ds.PublishToTopic("game_events", status)
				},
				nil, nil, nil, nil, nil,
			)
		}),
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
