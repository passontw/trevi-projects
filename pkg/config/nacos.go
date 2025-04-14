package config

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

// NacosConfig Nacos配置
type NacosConfig struct {
	Host        string
	Port        uint64
	Namespace   string
	Group       string
	Username    string
	Password    string
	DataID      string
	ServiceName string
	EnableNacos bool
}

// LoadNacosConfig 從環境變量加載Nacos配置
func LoadNacosConfig() NacosConfig {
	portStr := os.Getenv("NACOS_PORT")
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		port = 8848 // 默認端口
	}

	enableNacos := os.Getenv("ENABLE_NACOS") == "true"

	return NacosConfig{
		Host:        os.Getenv("NACOS_HOST"),
		Port:        port,
		Namespace:   os.Getenv("NACOS_NAMESPACE"),
		Group:       os.Getenv("NACOS_GROUP"),
		Username:    os.Getenv("NACOS_USERNAME"),
		Password:    os.Getenv("NACOS_PASSWORD"),
		DataID:      os.Getenv("NACOS_DATAID"),
		ServiceName: os.Getenv("NACOS_SERVICE_NAME"),
		EnableNacos: enableNacos,
	}
}

// NacosClient Nacos客戶端
type NacosClient struct {
	ConfigClient config_client.IConfigClient
	NamingClient naming_client.INamingClient
	Config       NacosConfig
}

// NewNacosClient 創建新的Nacos客戶端
func NewNacosClient(cfg NacosConfig) (*NacosClient, error) {
	if !cfg.EnableNacos {
		log.Println("Nacos is disabled, using local configuration")
		return &NacosClient{
			Config: cfg,
		}, nil
	}

	// 創建ServerConfig
	sc := []constant.ServerConfig{
		{
			IpAddr: cfg.Host,
			Port:   cfg.Port,
		},
	}

	// 創建ClientConfig
	cc := constant.ClientConfig{
		NamespaceId:         cfg.Namespace,
		Username:            cfg.Username,
		Password:            cfg.Password,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "debug",
	}

	// 創建配置客戶端
	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return nil, err
	}

	// 創建命名服務客戶端
	namingClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return nil, err
	}

	return &NacosClient{
		ConfigClient: configClient,
		NamingClient: namingClient,
		Config:       cfg,
	}, nil
}

// GetServiceConfig 從Nacos獲取服務配置
func (c *NacosClient) GetServiceConfig() (*ServiceConfig, error) {
	if !c.Config.EnableNacos {
		// 如果禁用Nacos，使用本地配置
		return &ServiceConfig{
			Server: ServerConfig{
				Port:        "8080",
				Environment: "development",
			},
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "postgres",
				Name:     "shoppingcart",
			},
			Redis: RedisConfig{
				Host:     "localhost",
				Port:     "6379",
				Password: "",
				DB:       0,
			},
		}, nil
	}

	// 從Nacos獲取配置
	content, err := c.ConfigClient.GetConfig(vo.ConfigParam{
		DataId: c.Config.DataID,
		Group:  c.Config.Group,
	})
	if err != nil {
		return nil, err
	}

	// 解析配置
	var config ServiceConfig
	if err := json.Unmarshal([]byte(content), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// RegisterService 向Nacos註冊服務
func (c *NacosClient) RegisterService(ip string, port uint64) (bool, error) {
	if !c.Config.EnableNacos {
		log.Println("Nacos is disabled, skipping service registration")
		return true, nil
	}

	return c.NamingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: c.Config.ServiceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		GroupName:   c.Config.Group,
	})
}

// DeregisterService 從Nacos註銷服務
func (c *NacosClient) DeregisterService(ip string, port uint64) (bool, error) {
	if !c.Config.EnableNacos {
		log.Println("Nacos is disabled, skipping service deregistration")
		return true, nil
	}

	return c.NamingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: c.Config.ServiceName,
		Ephemeral:   true,
		GroupName:   c.Config.Group,
	})
}
