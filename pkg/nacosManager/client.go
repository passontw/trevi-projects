package nacosManager

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"go.uber.org/fx"
)

// NacosConfig 存儲 Nacos 連接配置
type NacosConfig struct {
	IpAddr      string
	Port        uint64
	NamespaceId string
	Group       string
	DataId      string
	LogDir      string
	CacheDir    string
	Username    string
	Password    string
}

// NacosClient 提供對 Nacos 服務的訪問
type NacosClient interface {
	// 配置操作
	GetConfig(dataId, group string) (string, error)
	ListenConfig(dataId, group string, onConfigChange func(string)) error

	// 服務發現操作
	RegisterInstance(param vo.RegisterInstanceParam) (bool, error)
	DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error)
	GetService(param vo.GetServiceParam) (model.Service, error)
	SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error)

	// 獲取原始客戶端
	GetConfigClient() config_client.IConfigClient
	GetNamingClient() naming_client.INamingClient
}

// nacosClientImpl 是 NacosClient 介面的實作
type nacosClientImpl struct {
	configClient config_client.IConfigClient
	namingClient naming_client.INamingClient
	config       *NacosConfig
}

// 實現 NacosClient 介面

func (n *nacosClientImpl) GetConfig(dataId, group string) (string, error) {
	return n.configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
}

func (n *nacosClientImpl) ListenConfig(dataId, group string, onConfigChange func(string)) error {
	return n.configClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			// Call the simple callback with just the data
			onConfigChange(data)
		},
	})
}

func (n *nacosClientImpl) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	return n.namingClient.RegisterInstance(param)
}

func (n *nacosClientImpl) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	return n.namingClient.DeregisterInstance(param)
}

func (n *nacosClientImpl) GetService(param vo.GetServiceParam) (model.Service, error) {
	return n.namingClient.GetService(param)
}

func (n *nacosClientImpl) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	return n.namingClient.SelectInstances(param)
}

func (n *nacosClientImpl) GetConfigClient() config_client.IConfigClient {
	return n.configClient
}

func (n *nacosClientImpl) GetNamingClient() naming_client.INamingClient {
	return n.namingClient
}

// NewNacosClient 創建一個新的 Nacos 客戶端
func NewNacosClient(config *NacosConfig) (NacosClient, error) {
	// 創建 ServerConfig
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: config.IpAddr,
			Port:   config.Port,
		},
	}

	// 創建 ClientConfig
	clientConfig := constant.ClientConfig{
		NamespaceId:         config.NamespaceId,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              config.LogDir,
		CacheDir:            config.CacheDir,
		LogLevel:            "error",
		Username:            config.Username,
		Password:            config.Password,
	}

	// 創建配置客戶端
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create config client error: %w", err)
	}

	// 創建命名客戶端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create naming client error: %w", err)
	}

	return &nacosClientImpl{
		configClient: configClient,
		namingClient: namingClient,
		config:       config,
	}, nil
}

// ProvideNacosConfig 提供 Nacos 配置，用於 fx
func ProvideNacosConfig() *NacosConfig {
	// 嘗試加載 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// 從環境變量或其他配置源獲取
	config := &NacosConfig{
		IpAddr:      getEnv("NACOS_HOST", "127.0.0.1"),
		Port:        uint64(getEnvAsInt("NACOS_PORT", 8488)),
		NamespaceId: getEnv("NACOS_NAMESPACE", "public"),
		Group:       getEnv("NACOS_GROUP", "DEFAULT_GROUP"),
		DataId:      getEnv("NACOS_DATAID", "application"),
		LogDir:      "/tmp/nacos/log",
		CacheDir:    "/tmp/nacos/cache",
		Username:    getEnv("NACOS_USERNAME", "username"),
		Password:    getEnv("NACOS_PASSWORD", "password"),
	}

	log.Printf("Nacos Config: Host=%s, Port=%d, Namespace=%s, Group=%s, DataId=%s",
		config.IpAddr, config.Port, config.NamespaceId, config.Group, config.DataId)

	return config
}

func ProvideNacosClient(lc fx.Lifecycle, config *NacosConfig) (NacosClient, error) {
	log.Printf("開始創建 Nacos 客戶端: %s:%d", config.IpAddr, config.Port)

	client, err := NewNacosClient(config)
	if err != nil {
		log.Printf("創建 Nacos 客戶端失敗: %v", err)
		return nil, err
	}

	// 嘗試立即獲取配置以驗證連接
	_, err = client.GetConfig(config.DataId, config.Group)
	if err != nil {
		log.Printf("測試 Nacos 連接失敗 (獲取配置): %v", err)
		// 不返回錯誤，讓應用程序繼續啟動
	} else {
		log.Printf("成功連接到 Nacos 服務器並獲取配置")
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Nacos client connected successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Closing Nacos client connections...")
			// Nacos SDK 沒有明確的關閉方法，所以這裡不需要執行任何操作
			return nil
		},
	})

	return client, nil
}

func ProvideNacosClientPtr(client NacosClient) *NacosClient {
	return &client
}

// Module 創建 fx 模組，包含所有 Nacos 相關組件
var Module = fx.Module("nacos",
	fx.Provide(
		ProvideNacosConfig,
		ProvideNacosClient,
		ProvideNacosClientPtr,
	),
)
