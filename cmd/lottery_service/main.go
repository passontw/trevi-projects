package main

import (
	"context"
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

	"git.trevi.cc/server/go_gamecommon/cache"
	"git.trevi.cc/server/go_gamecommon/db"
	"git.trevi.cc/server/go_gamecommon/log"
	"git.trevi.cc/server/go_gamecommon/msgqueue"
	"git.trevi.cc/server/go_gamecommon/nacosmgr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 構建信息，通過 ldflags 在編譯時注入
var (
	BuildTime string
	GitHash   string
)

var (
	appConfig       *config.AppConfig
	redisConfig     *cache.RedisConfig
	dbConfigs       []db.DBConfig
	dnsresolver     *msgqueue.DnsResolver
	rocketmqconfigs *msgqueue.RocketMQConfig
)

// 載入配置
func loadConfig() {
	// 使用 go_gamecommon 中的 nacosmgr 創建 Nacos 客戶端
	nacosServer := config.GetNacosServer()

	nacosclient := nacosmgr.NewNacosClient(
		config.Args.LogDir,
		nacosServer,
		config.Args.NacosNamespace,
		config.Args.NacosUsername,
		config.Args.NacosPassword,
	)

	// 載入主配置
	configContent, err := nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosDataId)
	if err != nil {
		log.Error("無法獲取配置", zap.Error(err))
		panic(err)
	}

	// 解析主配置
	appConfig, err = config.ParseConfig([]byte(configContent))
	if err != nil {
		log.Error("解析配置失敗", zap.Error(err))
		panic(err)
	}

	// 載入 Redis 配置
	configContent, err = nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosRedisDataId)
	if err != nil {
		log.Error("獲取 Redis 配置失敗", zap.Error(err))
		panic(err)
	}
	redisConfig, err = cache.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		log.Error("解析 Redis 配置失敗", zap.Error(err))
		panic(err)
	}

	// 載入數據庫配置
	configContent, err = nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosTidbDataId)
	if err != nil {
		log.Error("獲取數據庫配置失敗", zap.Error(err))
		panic(err)
	}
	dbConfigs, err = db.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		log.Error("解析數據庫配置失敗", zap.Error(err))
		panic(err)
	}

	// 載入 RocketMQ 配置
	configContent, err = nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosRocketMQDataId)
	if err != nil {
		log.Error("獲取 RocketMQ 配置失敗", zap.Error(err))
		panic(err)
	}
	rocketmqconfigs, err = msgqueue.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		log.Error("解析 RocketMQ 配置失敗", zap.Error(err))
		panic(err)
	}
	dnsresolver = msgqueue.NewDnsResolver(rocketmqconfigs.Namesrvs)
}

// 主函數
func main() {
	// 初始化 flag 參數
	config.InitFlags()

	// 獲取 Nacos 地址診斷資訊
	host, port, isHttps := config.GetNacosHostAndPort()
	protocol := "HTTP"
	if isHttps {
		protocol = "HTTPS"
	}
	nacosServer := config.GetNacosServer()

	// 初始化日誌
	log.Init(log.Options{
		LogDir:     config.Args.LogDir,
		MaxSize:    100,
		MaxAge:     0,
		MaxBackups: 0,
		Compress:   true,
		TimeFormat: config.Args.LogFormat,
		Filename:   "lottery_service.log",
	})
	defer log.DeInit()
	defer func() {
		if err := recover(); err != nil {
			log.Error("panic", zap.Any("error", err))
		}
	}()

	log.Info("初始化彩票服務",
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

	// 載入配置
	loadConfig()

	// 初始化 Redis
	err := cache.RedisInit(*redisConfig, 3*time.Second)
	if err != nil {
		log.Error("初始化 Redis 失敗", zap.Error(err))
		panic(err)
	}
	defer cache.RedisClose()

	// 構建 MySQL 配置
	var mysqlConfig *databaseManager.MySQLConfig
	for _, dbConfig := range dbConfigs {
		if dbConfig.Name == "g38_lottery_service" || dbConfig.Name == "lottery" {
			// 找到彩票服務的數據庫配置
			mysqlConfig = &databaseManager.MySQLConfig{
				Host:      dbConfig.Host,
				Port:      dbConfig.Port,
				User:      dbConfig.Username,
				Password:  dbConfig.Password,
				Name:      dbConfig.Name,
				Charset:   "utf8mb4",
				ParseTime: true,
				Loc:       "Local",
			}
			break
		}
	}

	if mysqlConfig == nil {
		log.Error("未找到彩票服務的數據庫配置")
		panic("未找到彩票服務的數據庫配置")
	}

	// 構建應用程序
	app := fx.New(
		// 提供配置
		fx.Supply(appConfig),
		fx.Supply(dnsresolver),

		// 提供數據庫配置
		fx.Supply(mysqlConfig),
		fx.Provide(databaseManager.ProvideMySQLDatabaseManager),

		// 註冊模塊
		mq.Module,
		gameflow.Module,
		service.Module,
		grpc.Module,
		api.Module,

		// 提供日誌
		fx.Provide(
			func() *zap.Logger {
				return log.GetLogger() // 使用全局日誌
			},
		),
	)

	// 啟動應用
	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		log.Fatal("啟動應用失敗", zap.Error(err))
	}

	// 應用啟動成功後列印構建信息
	log.Info("彩票服務初始化成功",
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
