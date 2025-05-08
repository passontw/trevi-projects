package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"g38_lottery_service/internal/lottery_service/api"
	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/internal/lottery_service/dealerWebsocket"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/internal/lottery_service/grpc"
	"g38_lottery_service/internal/lottery_service/mq"
	"g38_lottery_service/internal/lottery_service/service"
	"g38_lottery_service/pkg/databaseManager"
	"g38_lottery_service/pkg/nacosManager"
	redis "g38_lottery_service/pkg/redisManager"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WebSocketNotifier 負責將遊戲事件推送到 WebSocket
type WebSocketNotifier struct {
	logger       *zap.Logger
	dealerServer *dealerWebsocket.DealerServer
}

// NewWebSocketNotifier 創建一個新的 WebSocket 通知服務
func NewWebSocketNotifier(logger *zap.Logger, dealerServer *dealerWebsocket.DealerServer) *WebSocketNotifier {
	return &WebSocketNotifier{
		logger:       logger.With(zap.String("component", "websocket_notifier")),
		dealerServer: dealerServer,
	}
}

// OnStageChanged 處理階段變更事件並推送到 WebSocket
func (n *WebSocketNotifier) OnStageChanged(gameID string, oldStage, newStage gameflow.GameStage, game *gameflow.GameData) {
	if game == nil {
		n.logger.Warn("OnStageChanged: 遊戲數據為空")
		return
	}

	// 構建遊戲狀態響應
	status := gameflow.BuildGameStatusResponse(game)

	// 推送到 WebSocket
	n.dealerServer.PublishToTopic("game_events", status)

	n.logger.Info("已推送遊戲狀態到 WebSocket",
		zap.String("topic", "game_events"),
		zap.String("gameID", gameID),
		zap.String("oldStage", string(oldStage)),
		zap.String("newStage", string(newStage)))
}

// 註冊 WebSocketNotifier 模塊
var WebSocketNotifierModule = fx.Module("websocket_notifier",
	fx.Provide(NewWebSocketNotifier),
	fx.Invoke(func(notifier *WebSocketNotifier, gameManager *gameflow.GameManager, lifecycle fx.Lifecycle) {
		lifecycle.Append(fx.Hook{
			OnStart: func(context.Context) error {
				// 註冊通知回調
				gameManager.RegisterWebSocketNotifier(notifier)
				return nil
			},
		})
	}),
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
		// 註冊數據庫模塊
		fx.Provide(func(cfg *config.AppConfig) *databaseManager.MySQLConfig {
			return &databaseManager.MySQLConfig{
				Host:      cfg.Database.Host,
				Port:      cfg.Database.Port,
				User:      cfg.Database.Username,
				Password:  cfg.Database.Password,
				Name:      cfg.Database.DBName,
				Charset:   "utf8mb4",
				ParseTime: true,
				Loc:       "Local",
			}
		}),
		fx.Provide(databaseManager.ProvideMySQLDatabaseManager),
		// 註冊 RocketMQ 生產者模塊
		mq.Module,
		// 註冊荷官端 WebSocket 模塊
		dealerWebsocket.Module,
		// 註冊遊戲流程管理模塊
		gameflow.Module,
		// 註冊 WebSocket 通知模塊 (放在遊戲流程管理模塊之後)
		WebSocketNotifierModule,
		// 註冊開獎服務模塊
		service.Module,
		// 註冊 gRPC 服務模塊
		grpc.Module,
		// 註冊 HTTP API 服務模塊
		api.Module,

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
