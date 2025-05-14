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

// 構建信息，通過 ldflags 在編譯時注入
var (
	BuildTime string
	GitHash   string
)

// 主函數：使用 fx 框架
// 測試 air 熱重載功能
func main() {
	// 初始化和解析命令行參數，這會將設置與環境變量同步
	config.InitFlags()

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
		// 註冊遊戲流程管理模塊
		gameflow.Module,
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

	// 應用啟動成功後列印構建信息
	logger.Info("Lottery service initialized successfully",
		zap.String("BuildTime", BuildTime),
		zap.String("GitHash", GitHash),
		zap.String("nacos_addr", config.Args.NacosHost+":"+config.Args.NacosPort),
		zap.String("nacos_namespace", config.Args.NacosNamespace),
		zap.String("nacos_group", config.Args.NacosGroup),
		zap.String("nacos_dataid", config.Args.NacosDataId))

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
