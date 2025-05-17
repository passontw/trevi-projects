package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"g38_lottery_service/internal/host_service/api"
	"g38_lottery_service/internal/host_service/config"
	"g38_lottery_service/pkg/healthcheck"

	"git.trevi.cc/server/go_gamecommon/log"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 構建信息，通過 ldflags 在編譯時注入
var (
	BuildTime string
	GitHash   string
)

func main() {
	// 初始化 flag 參數
	config.InitFlags()

	// 初始化日誌
	log.Init(log.Options{
		LogDir:     config.Args.LogDir,
		MaxSize:    100,
		MaxAge:     0,
		MaxBackups: 0,
		Compress:   true,
		TimeFormat: config.Args.LogFormat,
		Filename:   "host_service.log",
	})
	defer log.DeInit()
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic", zap.Any("error", err))
		}
	}()

	// 獲取 Nacos 地址診斷資訊
	host, port, isHttps := config.GetNacosHostAndPort()
	protocol := "HTTP"
	if isHttps {
		protocol = "HTTPS"
	}
	nacosServer := config.GetNacosServer()

	log.Info("初始化主機服務",
		zap.String("BuildTime", BuildTime),
		zap.String("GitHash", GitHash),
		zap.String("nacos_addr", config.Args.NacosAddr),
		zap.String("nacos_server", nacosServer),
		zap.String("nacos_host", host),
		zap.String("nacos_port", port),
		zap.String("nacos_protocol", protocol),
		zap.String("nacos_namespace", config.Args.NacosNamespace),
		zap.String("nacos_group", config.Args.NacosGroup),
		zap.String("nacos_dataid", config.Args.NacosDataId))

	// 構建應用程序
	app := fx.New(
		// 提供日誌
		fx.Provide(
			func() *zap.Logger {
				return log.GetLogger()
			},
		),

		// 註冊模塊
		config.Module, // 配置模塊
		api.Module,    // 整合 HTTP 和 WebSocket 模塊

		// 註冊健康檢查模塊
		healthcheck.NewCombinedModule(healthcheck.ModuleParams{
			Config: healthcheck.CombinedModuleConfig{
				HealthPort:      8088,
				ShutdownTimeout: 30 * time.Second,
				HandleSignals:   true,
			},
			Logger: log.GetLogger(),
		}),
	)

	// 啟動應用
	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		log.Fatal("啟動應用失敗", zap.Error(err))
	}

	// 應用啟動成功後列印構建信息
	log.Info("主機服務初始化成功",
		zap.String("BuildTime", BuildTime),
		zap.String("GitHash", GitHash))

	// 等待系統信號
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Info("服務即將關閉...")

	// 關閉應用
	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Stop(stopCtx); err != nil {
		log.Fatal("優雅關閉應用失敗", zap.Error(err))
	}

	log.Info("應用已優雅關閉")
}
