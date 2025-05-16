// module.go：定義 fx 框架使用的模塊和依賴注入

package config

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"git.trevi.cc/server/go_gamecommon/db"
	"git.trevi.cc/server/go_gamecommon/msgqueue"
	"git.trevi.cc/server/go_gamecommon/nacosmgr"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ProvideAppConfig 從 Nacos 獲取應用配置
func ProvideAppConfig(logger *zap.Logger) (*AppConfig, error) {
	// 獲取 Nacos 地址診斷資訊
	nacosHost, nacosPort, isHttps := GetNacosHostAndPort()
	protocol := "HTTP"
	if isHttps {
		protocol = "HTTPS"
	}
	nacosServer := GetNacosServer()

	logger.Info("從 Nacos 獲取配置",
		zap.String("nacos_addr", Args.NacosAddr),
		zap.String("nacos_server", nacosServer),
		zap.String("nacos_host", nacosHost),
		zap.String("nacos_port", nacosPort),
		zap.String("nacos_protocol", protocol),
		zap.String("nacos_namespace", Args.NacosNamespace),
		zap.String("nacos_group", Args.NacosGroup),
		zap.String("nacos_dataid", "lotterysvr.xml"))

	// 檢查 Nacos 服務器連接
	if err := CheckNacosConnection(); err != nil {
		logger.Error("Nacos 連接檢查失敗", zap.Error(err))
		logger.Error("請確認 Nacos 服務器地址是否正確，服務是否可用")
		logger.Error("如使用 Docker 部署，請確認網絡設置允許容器間通信")

		// 如果在開發環境，延遲 3 秒再嘗試，最多重試 3 次
		maxRetries := 3
		if Args.ServerMode == "dev" {
			for retryCount := 1; retryCount <= maxRetries; retryCount++ {
				retryDelay := time.Duration(retryCount) * 3 * time.Second
				logger.Info("處於開發模式，等待", zap.Duration("delay", retryDelay), zap.Int("retry", retryCount), zap.Int("maxRetries", maxRetries))
				time.Sleep(retryDelay)

				// 重新檢查連接
				if err := CheckNacosConnection(); err != nil {
					logger.Error("重試後仍無法連接到 Nacos 服務器", zap.Error(err), zap.Int("retryCount", retryCount))
					if retryCount == maxRetries {
						logger.Error("已達最大重試次數，請檢查以下事項:")
						logger.Error("1. Nacos 服務器是否已啟動")
						logger.Error("2. 環境變數 NACOS_ADDR 是否正確設置")
						logger.Error("3. .env 文件是否存在且包含正確配置")
						logger.Error("4. 命令行參數是否正確")
						logger.Error("5. 網絡是否通暢")
						return nil, err
					}
				} else {
					logger.Info("重試成功，繼續初始化")
					break
				}
			}
		} else {
			return nil, err
		}
	}

	// 獲取 Nacos 服務器地址
	nacosAddr := nacosHost + ":" + nacosPort // 使用 "host:port" 格式而不是完整的 URL

	// 創建 Nacos 客戶端
	nacosClient := nacosmgr.NewNacosClient(
		Args.LogDir,
		nacosAddr,
		Args.NacosNamespace,
		Args.NacosUsername,
		Args.NacosPassword,
	)

	// 載入主配置
	logger.Info("正在從 Nacos 獲取主配置",
		zap.String("group", Args.NacosGroup),
		zap.String("dataId", "lotterysvr.xml"))

	configContent, err := nacosClient.GetConfig(Args.NacosGroup, "lotterysvr.xml")
	if err != nil {
		logger.Error("無法獲取配置", zap.Error(err))

		// 增加更詳細的錯誤信息
		logger.Error("配置獲取失敗，請檢查以下內容：",
			zap.String("dataId", "lotterysvr.xml"),
			zap.String("group", Args.NacosGroup),
			zap.String("namespace", Args.NacosNamespace),
			zap.String("server", nacosServer))

		return nil, err
	}

	// 解析主配置
	appConfig, err := ParseConfig([]byte(configContent))
	if err != nil {
		logger.Error("解析配置失敗", zap.Error(err))
		logger.Error("配置內容可能格式不正確", zap.String("content_snippet", configContent[:min(100, len(configContent))]))
		return nil, err
	}

	// 設置服務信息
	appConfig.Server.ServiceName = Args.ServiceName

	// 如果命令行或環境變量指定了端口，則覆蓋配置文件中的端口
	if Args.ServicePort != "" {
		if port, err := strconv.Atoi(Args.ServicePort); err == nil {
			appConfig.Server.Port = port
		}
	}

	// 如果命令行或環境變量指定了WebSocket端口，則覆蓋配置文件中的端口
	if Args.WebsocketPort != "" {
		if port, err := strconv.Atoi(Args.WebsocketPort); err == nil {
			appConfig.Websocket.Port = port
		}
	}

	return appConfig, nil
}

// ProvideNacosClient 提供 Nacos 客戶端
func ProvideNacosClient(logger *zap.Logger) (*nacosmgr.NacosClient, error) {
	// 獲取 Nacos 服務器地址
	host, port, _ := GetNacosHostAndPort()
	nacosAddr := host + ":" + port // 使用 "host:port" 格式而不是完整的 URL

	// 創建 Nacos 客戶端
	nacosClient := nacosmgr.NewNacosClient(
		Args.LogDir,
		nacosAddr,
		Args.NacosNamespace,
		Args.NacosUsername,
		Args.NacosPassword,
	)

	return nacosClient, nil
}

// ProvideDatabaseManager 提供數據庫管理器
func ProvideDatabaseManager(appConfig *AppConfig, logger *zap.Logger) (*db.DBMgr, error) {
	// 獲取 Nacos 服務器地址
	host, port, _ := GetNacosHostAndPort()
	nacosAddr := host + ":" + port // 使用 "host:port" 格式而不是完整的 URL

	// 從 Nacos 獲取數據庫配置
	nacosClient := nacosmgr.NewNacosClient(
		Args.LogDir,
		nacosAddr,
		Args.NacosNamespace,
		Args.NacosUsername,
		Args.NacosPassword,
	)

	logger.Info("正在從 Nacos 獲取數據庫配置", zap.String("dataId", Args.NacosTidbDataId))
	configContent, err := nacosClient.GetConfig(Args.NacosGroup, Args.NacosTidbDataId)
	if err != nil {
		logger.Error("獲取數據庫配置失敗", zap.Error(err))
		logger.Error("請確認數據庫配置文件存在於 Nacos 中",
			zap.String("dataId", Args.NacosTidbDataId),
			zap.String("group", Args.NacosGroup))
		return nil, err
	}

	// 解析數據庫配置
	dbConfigs, err := db.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		logger.Error("解析數據庫配置失敗", zap.Error(err))
		logger.Error("數據庫配置 XML 可能格式不正確",
			zap.String("content_snippet", configContent[:min(100, len(configContent))]))
		return nil, err
	}

	// 尋找並初始化對應的數據庫
	var dbMgr *db.DBMgr
	for _, cfg := range dbConfigs {
		if cfg.Name == Args.ServiceName || cfg.Name == "host" || cfg.Name == "g38_host_service" {
			dbMgr, err = db.NewDBMgr(cfg)
			if err != nil {
				logger.Error("初始化數據庫失敗", zap.Error(err))
				return nil, err
			}
			logger.Info("數據庫連線成功",
				zap.String("name", cfg.Name),
				zap.String("type", cfg.Type),
				zap.String("host", cfg.Host),
				zap.Int("port", cfg.Port))
			break
		}
	}

	if dbMgr == nil {
		logger.Warn("未找到主機服務的數據庫配置，將使用通用配置或跳過數據庫初始化")
		// 可以返回nil或者使用默認配置創建一個DBMgr
	}

	return dbMgr, nil
}

// ProvideRocketMQResolver 提供 RocketMQ 解析器
func ProvideRocketMQResolver(appConfig *AppConfig, logger *zap.Logger) (*msgqueue.DnsResolver, error) {
	// 獲取 Nacos 服務器地址
	host, port, _ := GetNacosHostAndPort()
	nacosAddr := host + ":" + port // 使用 "host:port" 格式而不是完整的 URL

	// 從 Nacos 獲取 RocketMQ 配置
	nacosClient := nacosmgr.NewNacosClient(
		Args.LogDir,
		nacosAddr,
		Args.NacosNamespace,
		Args.NacosUsername,
		Args.NacosPassword,
	)

	logger.Info("正在從 Nacos 獲取 RocketMQ 配置", zap.String("dataId", Args.NacosRocketMQDataId))
	configContent, err := nacosClient.GetConfig(Args.NacosGroup, Args.NacosRocketMQDataId)
	if err != nil {
		logger.Error("獲取 RocketMQ 配置失敗", zap.Error(err))
		logger.Error("請確認 RocketMQ 配置文件存在於 Nacos 中",
			zap.String("dataId", Args.NacosRocketMQDataId),
			zap.String("group", Args.NacosGroup))
		return nil, err
	}

	// 解析 RocketMQ 配置
	rocketmqconfigs, err := msgqueue.LoadConfigFromXML([]byte(configContent))
	if err != nil {
		logger.Error("解析 RocketMQ 配置失敗", zap.Error(err))
		logger.Error("RocketMQ 配置 XML 可能格式不正確",
			zap.String("content_snippet", configContent[:min(100, len(configContent))]))
		return nil, err
	}

	// 創建 DNS 解析器
	dnsresolver := msgqueue.NewDnsResolver(rocketmqconfigs.Namesrvs)
	return dnsresolver, nil
}

// RegisterServiceToNacos 向 Nacos 註冊服務
func RegisterServiceToNacos(lc fx.Lifecycle, appConfig *AppConfig, nacosClient *nacosmgr.NacosClient, logger *zap.Logger) error {
	if !appConfig.Server.RegisterService {
		logger.Info("服務註冊已禁用，跳過向 Nacos 註冊服務")
		return nil
	}

	// 在應用啟動時註冊服務
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("向 Nacos 註冊服務")
			registered, err := RegisterServiceInstance(appConfig, nacosClient)
			if err != nil {
				logger.Error("服務註冊失敗", zap.Error(err))
				return err
			}
			logger.Info("服務註冊狀態", zap.Bool("registered", registered))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 在應用關閉時可以考慮註銷服務
			return nil
		},
	})

	return nil
}

// RegisterServiceInstance 向 Nacos 註冊服務實例
func RegisterServiceInstance(config *AppConfig, nacosClient *nacosmgr.NacosClient) (bool, error) {
	// 獲取服務 IP
	serviceIP := config.Server.ServiceIP
	if serviceIP == "" {
		// 如果配置中沒有指定 IP，可以嘗試自動獲取
		serviceIP = "127.0.0.1" // 這裡應該有更好的方法來獲取當前主機的 IP
	}

	// 獲取服務端口
	servicePort := config.Server.ServicePort
	if servicePort <= 0 {
		servicePort = config.Server.Port
	}

	// 檢查服務名稱
	serviceName := config.Server.ServiceName
	if serviceName == "" {
		serviceName = "g38_host_service"
	}

	// 使用 Nacos 客戶端的 GetNamingClient() 和 RegisterInstance 方法註冊服務
	namingClient := nacosClient.GetNamingClient()
	if namingClient == nil {
		return false, fmt.Errorf("無法獲取 Nacos 命名服務客戶端")
	}

	return namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          serviceIP,
		Port:        uint64(servicePort),
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata: map[string]string{
			"version": config.Server.Version,
			"id":      config.Server.ServiceID,
		},
	})
}

// ConfigModule 配置模塊，提供應用配置
var ConfigModule = fx.Options(
	fx.Provide(ProvideAppConfig),
)

// ServiceModule 服務模塊，處理服務註冊
var ServiceModule = fx.Options(
	fx.Provide(ProvideNacosClient),
	fx.Invoke(RegisterServiceToNacos),
)

// DatabaseModule 數據庫模塊
var DatabaseModule = fx.Options(
	fx.Provide(ProvideDatabaseManager),
)

// Module 所有配置相關模塊的组合
var Module = fx.Options(
	ConfigModule,
	ServiceModule,
	DatabaseModule,
	fx.Provide(ProvideRocketMQResolver),
)
