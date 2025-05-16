package main

import (
	"context"
	"strings"
	"time"

	"g38_lottery_service/internal/lottery_service/api"
	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/internal/lottery_service/grpc"
	"g38_lottery_service/internal/lottery_service/mq"
	"g38_lottery_service/internal/lottery_service/service"
	"g38_lottery_service/pkg/healthcheck"

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
	appConfig            *config.AppConfig
	lotteryServiceConfig *config.LotteryServiceNacosConfig
	redisConfig          *cache.RedisConfig
	dbConfigs            []db.DBConfig
	dnsresolver          *msgqueue.DnsResolver
	rocketmqconfigs      *msgqueue.RocketMQConfig
)

// 載入配置
func loadConfig() {
	// 使用 go_gamecommon 中的 nacosmgr 創建 Nacos 客戶端
	nacosServer := config.GetNacosServer()

	// 檢查 Nacos 服務器連接
	if err := config.CheckNacosConnection(); err != nil {
		log.Error("Nacos 連接檢查失敗", zap.Error(err))
		log.Error("請確認 Nacos 服務器地址是否正確，服務是否可用")
		log.Error("如使用 Docker 部署，請確認網絡設置允許容器間通信")

		// 如果在開發環境，延遲 3 秒再嘗試，最多重試 3 次
		maxRetries := 3
		if config.Args.ServerMode == "dev" {
			for retryCount := 1; retryCount <= maxRetries; retryCount++ {
				retryDelay := time.Duration(retryCount) * 3 * time.Second
				log.Info("處於開發模式，等待", zap.Duration("delay", retryDelay), zap.Int("retry", retryCount), zap.Int("maxRetries", maxRetries))
				time.Sleep(retryDelay)

				// 重新檢查連接
				if err := config.CheckNacosConnection(); err != nil {
					log.Error("重試後仍無法連接到 Nacos 服務器", zap.Error(err), zap.Int("retryCount", retryCount))
					if retryCount == maxRetries {
						log.Error("已達最大重試次數，請檢查以下事項:")
						log.Error("1. Nacos 服務器是否已啟動")
						log.Error("2. 環境變數 NACOS_ADDR 是否正確設置")
						log.Error("3. .env 文件是否存在且包含正確配置")
						log.Error("4. 命令行參數是否正確")
						log.Error("5. 網絡是否通暢")
						panic(err)
					}
				} else {
					log.Info("重試成功，繼續初始化")
					break
				}
			}
		} else {
			panic(err)
		}
	}

	// 創建 Nacos 客戶端
	nacosclient := nacosmgr.NewNacosClient(
		config.Args.LogDir,
		nacosServer,
		config.Args.NacosNamespace,
		config.Args.NacosUsername,
		config.Args.NacosPassword,
	)

	// 載入主配置
	log.Info("正在從 Nacos 獲取主配置",
		zap.String("group", config.Args.NacosGroup),
		zap.String("dataId", config.Args.NacosDataId))

	configContent, err := nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosDataId)
	if err != nil {
		log.Error("無法獲取配置", zap.Error(err))

		// 增加更詳細的錯誤信息
		log.Error("配置獲取失敗，請檢查以下內容：",
			zap.String("dataId", config.Args.NacosDataId),
			zap.String("group", config.Args.NacosGroup),
			zap.String("namespace", config.Args.NacosNamespace),
			zap.String("server", nacosServer))

		// 檢查是否存在認證問題
		if strings.Contains(err.Error(), "invalid user") ||
			strings.Contains(err.Error(), "no auth") {
			log.Error("可能是認證問題，請檢查用戶名和密碼",
				zap.String("username", config.Args.NacosUsername))
			log.Error("您可以在 .env 文件或環境變量中設置 NACOS_USERNAME 和 NACOS_PASSWORD")
		}

		// 如果在開發環境中，提供更多調試信息
		if config.Args.ServerMode == "dev" {
			log.Error("開發環境調試提示:")
			log.Error("1. 請確認 Nacos 服務器正在運行")
			log.Error("2. 請檢查 .env 文件中的 NACOS_ADDR 設置是否正確")
			log.Error("3. 使用瀏覽器訪問 Nacos 控制台驗證配置是否存在")
			log.Error("4. 如果使用 Docker，請確保網絡設置正確")
		}

		panic(err)
	}

	// 解析主配置
	appConfig, err = config.ParseConfig([]byte(configContent))
	if err != nil {
		log.Error("解析配置失敗", zap.Error(err))
		log.Error("配置內容可能格式不正確", zap.String("content_snippet", configContent[:min(100, len(configContent))]))
		panic(err)
	}

	// 載入彩票服務配置
	log.Info("正在從 Nacos 獲取彩票服務配置", zap.String("dataId", "lotterysvr.xml"))
	lotteryContent, err := nacosclient.GetConfig(config.Args.NacosGroup, "lotterysvr.xml")
	if err != nil {
		log.Error("獲取彩票服務配置失敗", zap.Error(err))
		log.Error("將使用默認配置值")
	} else {
		// 解析彩票服務配置
		lotteryServiceConfig, err = config.ParseLotteryServiceConfig([]byte(lotteryContent))
		if err != nil {
			log.Error("解析彩票服務配置失敗", zap.Error(err))
			log.Error("將使用默認配置值")
		} else {
			// 將彩票服務配置應用到主配置
			err = lotteryServiceConfig.ApplyToAppConfig(appConfig)
			if err != nil {
				log.Error("應用彩票服務配置失敗", zap.Error(err))
			}
		}
	}

	// 載入 Redis 配置
	log.Info("正在從 Nacos 獲取 Redis 配置", zap.String("dataId", config.Args.NacosRedisDataId))
	configContent, err = nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosRedisDataId)
	if err != nil {
		log.Error("獲取 Redis 配置失敗", zap.Error(err))
		log.Error("請確認 Redis 配置文件存在於 Nacos 中",
			zap.String("dataId", config.Args.NacosRedisDataId),
			zap.String("group", config.Args.NacosGroup))
		panic(err)
	}
	redisConfig, err = cache.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		log.Error("解析 Redis 配置失敗", zap.Error(err))
		log.Error("Redis 配置 XML 可能格式不正確",
			zap.String("content_snippet", configContent[:min(100, len(configContent))]))
		panic(err)
	}

	// 載入數據庫配置
	log.Info("正在從 Nacos 獲取數據庫配置", zap.String("dataId", config.Args.NacosTidbDataId))
	configContent, err = nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosTidbDataId)
	if err != nil {
		log.Error("獲取數據庫配置失敗", zap.Error(err))
		log.Error("請確認數據庫配置文件存在於 Nacos 中",
			zap.String("dataId", config.Args.NacosTidbDataId),
			zap.String("group", config.Args.NacosGroup))
		panic(err)
	}
	dbConfigs, err = db.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		log.Error("解析數據庫配置失敗", zap.Error(err))
		log.Error("數據庫配置 XML 可能格式不正確",
			zap.String("content_snippet", configContent[:min(100, len(configContent))]))
		panic(err)
	}

	// 載入 RocketMQ 配置
	log.Info("正在從 Nacos 獲取 RocketMQ 配置", zap.String("dataId", config.Args.NacosRocketMQDataId))
	configContent, err = nacosclient.GetConfig(config.Args.NacosGroup, config.Args.NacosRocketMQDataId)
	if err != nil {
		log.Error("獲取 RocketMQ 配置失敗", zap.Error(err))
		log.Error("請確認 RocketMQ 配置文件存在於 Nacos 中",
			zap.String("dataId", config.Args.NacosRocketMQDataId),
			zap.String("group", config.Args.NacosGroup))
		panic(err)
	}
	rocketmqconfigs, err = msgqueue.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		log.Error("解析 RocketMQ 配置失敗", zap.Error(err))
		log.Error("RocketMQ 配置 XML 可能格式不正確",
			zap.String("content_snippet", configContent[:min(100, len(configContent))]))
		panic(err)
	}
	dnsresolver = msgqueue.NewDnsResolver(rocketmqconfigs.Namesrvs)
}

// min 返回兩個整數中較小的一個
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	// 尋找並初始化對應的數據庫
	var dbMgr *db.DBMgr

	for _, cfg := range dbConfigs {
		if cfg.Name == "g38_lottery_service" || cfg.Name == "lottery" {
			dbMgr, err = db.NewDBMgr(cfg)
			if err != nil {
				log.Error("初始化數據庫失敗", zap.Error(err))
				panic(err)
			}
			log.Info("數據庫連線成功",
				zap.String("name", cfg.Name),
				zap.String("type", cfg.Type),
				zap.String("host", cfg.Host),
				zap.Int("port", cfg.Port))
			break
		}
	}

	if dbMgr == nil {
		log.Error("未找到彩票服務的數據庫配置")
		panic("未找到彩票服務的數據庫配置")
	}
	defer dbMgr.Close()

	// 創建健康檢查模組配置
	healthModuleConfig := healthcheck.CombinedModuleConfig{
		// 健康檢查端口
		HealthPort: 8088,
		// 優雅關閉超時（秒）
		ShutdownTimeout: 30 * time.Second,
		// 是否處理系統信號
		HandleSignals: true,
	}

	// 使用 fx 啟動應用
	app := fx.New(
		// 提供配置
		fx.Provide(func() *config.AppConfig {
			return appConfig
		}),
		fx.Provide(func() *zap.Logger {
			return log.GetLogger()
		}),
		fx.Provide(func() *db.DBMgr {
			return dbMgr
		}),
		fx.Provide(func() *msgqueue.DnsResolver {
			return dnsresolver
		}),
		// 註冊業務邏輯模塊
		service.Module,
		gameflow.Module,
		// 註冊 gRPC 模塊
		grpc.Module,
		// 註冊 API 模塊
		api.Module,
		// 註冊 MQ 模塊
		mq.Module,
		// 註冊健康檢查模塊（使用新的綜合模組）
		healthcheck.NewCombinedModule(healthcheck.ModuleParams{
			Config: healthModuleConfig,
			Logger: log.GetLogger(),
		}),
	)

	// 啟動應用
	if err := app.Start(context.Background()); err != nil {
		log.Error("應用啟動失敗", zap.Error(err))
		return
	}

	log.Info("彩票服務初始化成功",
		zap.String("BuildTime", BuildTime),
		zap.String("GitHash", GitHash))

	// 等待應用停止
	<-app.Done()
	log.Info("應用已完全關閉")
}
