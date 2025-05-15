package config

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"git.trevi.cc/server/go_gamecommon/nacosmgr"

	"github.com/joho/godotenv"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"go.uber.org/fx"
)

// ===== 配置結構定義 =====

// AppConfig 應用程式配置結構
type AppConfig struct {
	AppName  string         `json:"appName"`
	Debug    bool           `json:"debug"`
	Server   ServerConfig   `json:"server"`
	Redis    RedisConfig    `json:"redis"`
	Database DatabaseConfig `json:"database"`
	JWT      JWTConfig      `json:"jwt"`
	Nacos    NacosConfig    `json:"nacos"`
	RocketMQ RocketMQConfig `json:"rocketmq"`
}

// ServerConfig 服務器配置
type ServerConfig struct {
	Host                   string `json:"host"`
	Port                   int    `json:"port"`
	Version                string `json:"version"`
	ServiceName            string `json:"serviceName"`
	ServiceID              string `json:"serviceId"`
	ServiceIP              string `json:"serviceIp"`
	ServicePort            int    `json:"servicePort"`
	DealerWsPort           int    `json:"dealerWsPort"`   // 荷官 WebSocket 端口
	GrpcPort               int    `json:"grpcPort"`       // gRPC 服務端口
	GrpcMaxConnectionAge   int    `json:"grpcMaxConnAge"` // gRPC 最大連接存活時間（秒）
	RegisterService        bool   `json:"registerService"`
	ShutdownTimeoutSeconds int    `json:"shutdownTimeout"` // 優雅關閉超時秒數
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string                 `json:"host"`
	Port     int                    `json:"port"`
	Username string                 `json:"username"`
	Password string                 `json:"password"`
	DB       int                    `json:"db"`
	Extra    map[string]interface{} `json:"-"` // 額外的配置項，不會被 JSON 序列化
}

// DatabaseConfig 數據庫配置
type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret      string        `json:"secret"`
	AdminSecret string        `json:"adminSecret"`
	ExpiresIn   time.Duration `json:"expiresIn"`
}

// NacosConfig Nacos 配置
type NacosConfig struct {
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        uint64 `json:"port"`
	Namespace   string `json:"namespace"`
	Group       string `json:"group"`
	DataId      string `json:"dataId"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	ServiceIP   string `json:"serviceIp"`
	ServicePort int    `json:"servicePort"`
}

// RocketMQConfig RocketMQ 配置
type RocketMQConfig struct {
	Enabled       bool     `json:"enabled"` // 是否啟用 RocketMQ
	NameServers   []string `json:"nameServers"`
	AccessKey     string   `json:"accessKey"`
	SecretKey     string   `json:"secretKey"`
	ProducerGroup string   `json:"producerGroup"`
	ConsumerGroup string   `json:"consumerGroup"`
}

// RocketMQXMLConfig 是 RocketMQ XML 配置的結構
type RocketMQXMLConfig struct {
	XMLName  xml.Name `xml:"config"`
	RocketMQ struct {
		NameSrv struct {
			Host string `xml:"host,attr"`
			Port string `xml:"port,attr"`
		} `xml:"namesrv"`
	} `xml:"rocketmq"`
}

// RedisXMLConfig 是 Redis XML 配置的結構
type RedisXMLConfig struct {
	XMLName xml.Name `xml:"config"`
	Redis   struct {
		Host      string `xml:"host,attr"`
		Port      string `xml:"port,attr"`
		Username  string `xml:"username,attr"`
		Password  string `xml:"password,attr"`
		DB        string `xml:"db,attr"`
		IsCluster string `xml:"is_cluster,attr"`
		Nodes     struct {
			NodeList []string `xml:"node"`
		} `xml:"nodes"`
	} `xml:"redis"`
}

// TiDBXMLConfig 是 TiDB XML 配置的結構
type TiDBXMLConfig struct {
	XMLName xml.Name `xml:"config"`
	DBs     []struct {
		Name     string `xml:"name,attr"`
		Type     string `xml:"type"`
		Host     string `xml:"host"`
		Port     string `xml:"port"`
		DBName   string `xml:"name"`
		User     string `xml:"user"`
		Password string `xml:"password"`
	} `xml:"db"`
}

// ===== 環境變量工具函數 =====

// getEnv 從環境變量獲取字符串值，如果不存在則返回默認值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 從環境變量獲取整數值，如果不存在或無法解析則返回默認值
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool 從環境變量獲取布爾值，如果不存在或無法解析則返回默認值
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsDuration 從環境變量獲取時間間隔，格式如 "24h"
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// ===== Nacos 相關函數 =====

// uploadConfigToNacos 上傳配置到 Nacos
func uploadConfigToNacos(client *nacosmgr.NacosClient, dataId, group, content string) (bool, error) {
	// 獲取原始的 Nacos client 以使用 PublishConfig 功能
	configClient := client.GetConfigClient()
	if configClient == nil {
		return false, fmt.Errorf("無法獲取 Nacos 配置客戶端")
	}

	// 上傳配置
	success, err := configClient.PublishConfig(vo.ConfigParam{
		DataId:  dataId,
		Group:   group,
		Content: content,
	})

	if err != nil {
		return false, err
	}

	return success, nil
}

// DetectAndParseConfig 檢測並解析配置格式 (XML 或 JSON)
func DetectAndParseConfig(data string) (map[string]interface{}, error) {
	var result map[string]interface{}

	// 嘗試解析為 JSON
	if err := json.Unmarshal([]byte(data), &result); err == nil {
		return result, nil
	}

	// 嘗試解析為 XML
	var xmlMap map[string]interface{}
	if err := xml.Unmarshal([]byte(data), &xmlMap); err == nil {
		return xmlMap, nil
	}

	// 都失敗了，返回錯誤
	return nil, fmt.Errorf("無法解析配置: 不支持的格式")
}

// LoadConfig 加載配置，優先使用 .env 中的 Nacos 設定取回配置
// 當無法從 Nacos 獲取有效配置時，返回錯誤
func LoadConfig(nacosClient *nacosmgr.NacosClient) (*AppConfig, error) {
	// 嘗試加載 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 找不到 .env 文件: %v", err)
	}

	// 從環境變量獲取 Nacos 配置
	nacosEnabled := getEnvAsBool("ENABLE_NACOS", false)

	// 初始化默認配置
	config := createDefaultConfig()

	// 如果啟用了 Nacos，從 Nacos 獲取配置
	if nacosEnabled && nacosClient != nil {
		log.Println("Nacos 配置已啟用，正在從 Nacos 獲取配置...")

		// 從 Nacos 獲取主配置
		content, err := nacosClient.GetConfig(config.Nacos.Group, config.Nacos.DataId)
		if err != nil {
			log.Printf("從 Nacos 獲取配置失敗: %v", err)
			return nil, fmt.Errorf("無法從 Nacos 獲取配置: %w", err)
		}

		// 檢測配置格式並解析
		configMap, err := DetectAndParseConfig(content)
		if err != nil {
			log.Printf("檢測或解析配置格式失敗: %v", err)
			return nil, fmt.Errorf("無法解析從 Nacos 獲取的配置: %w", err)
		}

		// 將配置映射到 AppConfig 結構
		if err := mapJsonMapToAppConfig(configMap, config); err != nil {
			log.Printf("映射配置到應用配置失敗: %v", err)
			return nil, fmt.Errorf("映射配置到應用配置失敗: %w", err)
		}

		log.Println("成功從 Nacos 獲取配置並合併")

		// 獲取 Redis XML 配置
		redisDataId := getEnv("NACOS_REDIS_DATAID", "redisconfig.xml")
		log.Printf("將使用 Redis 配置 DataId: %s", redisDataId)

		if redisDataId != "" {
			log.Printf("嘗試從 Nacos 獲取 Redis XML 配置 (DataId: %s)...", redisDataId)
			redisXmlContent, err := nacosClient.GetConfig(config.Nacos.Group, redisDataId)
			if err != nil {
				log.Printf("警告: 無法從 Nacos 獲取 Redis XML 配置: %v", err)
			} else if redisXmlContent != "" {
				log.Printf("已從 Nacos 獲取 Redis XML 配置，長度: %d 字節", len(redisXmlContent))
				// 解析並應用 Redis XML 配置
				if err := parseRedisXmlConfig(redisXmlContent, config); err != nil {
					log.Printf("警告: 解析 Redis XML 配置失敗: %v", err)
				} else {
					log.Printf("成功應用 Redis XML 配置")
				}
			} else {
				log.Printf("警告: 從 Nacos 獲取的 Redis XML 配置為空")
			}
		}

		// 獲取 TiDB XML 配置
		tidbDataId := getEnv("NACOS_TIDB_DATAID", "dbconfig.xml")
		serviceName := getEnv("SERVICE_NAME", "g38_lottery_service")

		log.Printf("將使用 TiDB 配置 DataId: %s, 服務名稱: %s", tidbDataId, serviceName)

		if tidbDataId != "" {
			log.Printf("嘗試從 Nacos 獲取 TiDB XML 配置 (DataId: %s, 服務名稱: %s)...", tidbDataId, serviceName)
			tidbXmlContent, err := nacosClient.GetConfig(config.Nacos.Group, tidbDataId)
			if err != nil {
				log.Printf("警告: 無法從 Nacos 獲取 TiDB XML 配置: %v", err)
			} else if tidbXmlContent != "" {
				log.Printf("已從 Nacos 獲取 TiDB XML 配置，長度: %d 字節", len(tidbXmlContent))
				log.Printf("TiDB XML 配置內容摘要: %s", tidbXmlContent[:min(100, len(tidbXmlContent))])

				// 解析並應用 TiDB XML 配置
				if err := parseTiDBXmlConfig(tidbXmlContent, config, serviceName); err != nil {
					log.Printf("警告: 解析 TiDB XML 配置失敗: %v", err)
				} else {
					log.Printf("成功應用 TiDB XML 配置")
				}
			} else {
				log.Printf("警告: 從 Nacos 獲取的 TiDB XML 配置為空")
			}
		}

		// 獲取 RocketMQ XML 配置
		rocketMQDataId := getEnv("NACOS_ROCKETMQ_DATAID", "rocketmq.xml")
		log.Printf("將使用 RocketMQ 配置 DataId: %s", rocketMQDataId)

		if rocketMQDataId != "" {
			log.Printf("嘗試從 Nacos 獲取 RocketMQ XML 配置 (DataId: %s)...", rocketMQDataId)
			rocketMQXmlContent, err := nacosClient.GetConfig(config.Nacos.Group, rocketMQDataId)
			if err != nil {
				log.Printf("警告: 無法從 Nacos 獲取 RocketMQ XML 配置: %v", err)
			} else if rocketMQXmlContent != "" {
				log.Printf("已從 Nacos 獲取 RocketMQ XML 配置，長度: %d 字節", len(rocketMQXmlContent))
				log.Printf("RocketMQ XML 配置內容摘要: %s", rocketMQXmlContent[:min(100, len(rocketMQXmlContent))])

				// 解析並應用 RocketMQ XML 配置
				if err := parseRocketMQXmlConfig(rocketMQXmlContent, config); err != nil {
					log.Printf("警告: 解析 RocketMQ XML 配置失敗: %v", err)
				} else {
					log.Printf("成功應用 RocketMQ XML 配置")
				}
			} else {
				log.Printf("警告: 從 Nacos 獲取的 RocketMQ XML 配置為空")
			}
		}

		// 保留 Nacos 連接信息
		preserveNacosConnectionInfo(config)
	} else {
		// Nacos 未啟用或客戶端無效
		log.Println("Nacos 未啟用或客戶端無效，無法獲取配置")
		return nil, fmt.Errorf("Nacos 未啟用或客戶端無效，無法獲取配置")
	}

	// 打印最終配置
	log.Printf("最終數據庫配置: Host=%s, Port=%d, Username=%s, DBName=%s",
		config.Database.Host, config.Database.Port, config.Database.Username, config.Database.DBName)

	// 創建 Redis 額外配置映射
	config.Redis.Extra = make(map[string]interface{})

	return config, nil
}

// min 返回兩個整數中較小的一個
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// createDefaultConfig 創建預設配置
func createDefaultConfig() *AppConfig {
	nacosEnabled := getEnvAsBool("ENABLE_NACOS", true)

	// 獲取 Nacos 主機和端口
	var nacosHost string
	var nacosPort uint64

	// 使用 NacosAddr 進行配置
	if Args.parsed {
		// 如果已經解析過命令行參數，則使用 Args
		host, port, _ := GetNacosHostAndPort()
		nacosHost = host
		if p, err := strconv.ParseUint(port, 10, 64); err == nil {
			nacosPort = p
		} else {
			nacosPort = 8848 // 默認端口
		}
	} else {
		// 直接從環境變量獲取
		nacosHost = getEnv("NACOS_HOST", "127.0.0.1")
		nacosPort = uint64(getEnvAsInt("NACOS_PORT", 8848))
	}

	return &AppConfig{
		AppName: getEnv("APP_NAME", "g38_lottery_service"),
		Debug:   getEnvAsBool("DEBUG", true),
		Server: ServerConfig{
			Host:                   getEnv("SERVER_HOST", "0.0.0.0"),
			Port:                   getEnvAsInt("SERVER_PORT", 8000),
			Version:                getEnv("SERVER_VERSION", "v1"),
			ServiceName:            getEnv("SERVICE_NAME", "lottery-service"),
			ServiceID:              getEnv("SERVICE_ID", "lottery-service-1"),
			ServiceIP:              getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort:            getEnvAsInt("SERVICE_PORT", 8080),
			DealerWsPort:           getEnvAsInt("DEALER_WS_PORT", 9000),   // 默認荷官 WebSocket 端口
			GrpcPort:               getEnvAsInt("GRPC_PORT", 9100),        // 默認 gRPC 端口
			GrpcMaxConnectionAge:   getEnvAsInt("GRPC_MAX_CONN_AGE", 600), // gRPC 最大連接存活時間（秒），默認 10 分鐘
			RegisterService:        getEnvAsBool("REGISTER_SERVICE", true),
			ShutdownTimeoutSeconds: getEnvAsInt("SHUTDOWN_TIMEOUT", 30), // 優雅關閉默認 30 秒
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Username: getEnv("REDIS_USERNAME", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Database: DatabaseConfig{
			Driver:   getEnv("DB_DRIVER", "mysql"),
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("DB_PORT", 4000),
			Username: getEnv("DB_USERNAME", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "lottery"),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "your-secret-key"),
			AdminSecret: getEnv("ADMIN_JWT_SECRET", "your-admin-secret-key"),
			ExpiresIn:   getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour),
		},
		Nacos: NacosConfig{
			Enabled:     nacosEnabled,
			Host:        nacosHost,
			Port:        nacosPort,
			Namespace:   getEnv("NACOS_NAMESPACE", "public"),
			Group:       getEnv("NACOS_GROUP", "DEFAULT_GROUP"),
			DataId:      getEnv("NACOS_DATAID", "lotterysvr.xml"),
			Username:    getEnv("NACOS_USERNAME", "nacos"),
			Password:    getEnv("NACOS_PASSWORD", "nacos"),
			ServiceIP:   getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort: getEnvAsInt("SERVICE_PORT", 8080),
		},
		RocketMQ: RocketMQConfig{
			Enabled:       getEnvAsBool("ROCKETMQ_ENABLED", false),
			NameServers:   []string{getEnv("ROCKETMQ_NAME_SERVERS", "localhost:9876")},
			AccessKey:     getEnv("ROCKETMQ_ACCESS_KEY", ""),
			SecretKey:     getEnv("ROCKETMQ_SECRET_KEY", ""),
			ProducerGroup: getEnv("ROCKETMQ_PRODUCER_GROUP", "default"),
			ConsumerGroup: getEnv("ROCKETMQ_CONSUMER_GROUP", "default"),
		},
	}
}

// restoreWebSocketPorts 恢復 WebSocket 端口和 gRPC 端口
func restoreWebSocketPorts(config *AppConfig, dealerWsPort int, grpcPort int, apiPort int) {
	// 檢查荷官WebSocket端口是否需要恢復
	if config.Server.DealerWsPort == 0 {
		log.Printf("Nacos 中未配置有效的 DEALER_WS_PORT，恢復為原值: %d", dealerWsPort)
		config.Server.DealerWsPort = dealerWsPort
	} else if config.Server.DealerWsPort != dealerWsPort {
		log.Printf("已從 Nacos 更新荷官 WebSocket 端口: %d -> %d", dealerWsPort, config.Server.DealerWsPort)
	}

	// 檢查 gRPC 端口是否需要恢復
	if config.Server.GrpcPort == 0 {
		log.Printf("Nacos 中未配置有效的 GRPC_PORT，恢復為原值: %d", grpcPort)
		config.Server.GrpcPort = grpcPort
	} else if config.Server.GrpcPort != grpcPort {
		log.Printf("已從 Nacos 更新 gRPC 端口: %d -> %d", grpcPort, config.Server.GrpcPort)
	}

	// 檢查 API 端口是否需要恢復
	if config.Server.Port == 0 {
		log.Printf("Nacos 中未配置有效的 API_PORT，恢復為原值: %d", apiPort)
		config.Server.Port = apiPort
	} else if config.Server.Port != apiPort {
		log.Printf("已從 Nacos 更新 API 端口: %d -> %d", apiPort, config.Server.Port)
	}
}

// preserveNacosConnectionInfo 保留 Nacos 連接信息
// 在從 Nacos 獲取配置後調用，確保不會覆蓋當前的連接信息
func preserveNacosConnectionInfo(config *AppConfig) {
	// 保存原始的WebSocket端口和gRPC端口設置
	dealerWsPort := config.Server.DealerWsPort
	grpcPort := config.Server.GrpcPort
	apiPort := config.Server.Port

	log.Printf("保存端口值以進行還原: DealerWsPort=%d, GrpcPort=%d, ApiPort=%d",
		dealerWsPort, grpcPort, apiPort)

	// 獲取 Nacos 主機和端口
	var nacosHost string
	var nacosPort uint64

	// 使用 NacosAddr 進行配置
	if Args.parsed {
		// 如果已經解析過命令行參數，則使用 Args
		host, port, _ := GetNacosHostAndPort()
		nacosHost = host
		if p, err := strconv.ParseUint(port, 10, 64); err == nil {
			nacosPort = p
		} else {
			nacosPort = 8848 // 默認端口
		}
	} else {
		// 直接從環境變量獲取
		nacosHost = getEnv("NACOS_HOST", "127.0.0.1")
		nacosPort = uint64(getEnvAsInt("NACOS_PORT", 8848))
	}

	// 保留當前的 Nacos 連接信息
	config.Nacos.Host = nacosHost
	config.Nacos.Port = nacosPort
	config.Nacos.Namespace = getEnv("NACOS_NAMESPACE", "public")
	config.Nacos.Group = getEnv("NACOS_GROUP", "DEFAULT_GROUP")
	config.Nacos.DataId = getEnv("NACOS_DATAID", "lotterysvr.xml")
	config.Nacos.Username = getEnv("NACOS_USERNAME", "nacos")
	config.Nacos.Password = getEnv("NACOS_PASSWORD", "nacos")
	config.Nacos.ServiceIP = getEnv("SERVICE_IP", "127.0.0.1")
	config.Nacos.ServicePort = getEnvAsInt("SERVICE_PORT", 8080)

	// 恢復端口設置
	restoreWebSocketPorts(config, dealerWsPort, grpcPort, apiPort)

	log.Printf("Nacos 初始配置設置完成，最終設置: DealerWsPort=%d, GrpcPort=%d, ApiPort=%d",
		config.Server.DealerWsPort, config.Server.GrpcPort, config.Server.Port)
}

// mapJsonToAppConfig 將 JSON 映射到 AppConfig 結構
func mapJsonToAppConfig(jsonMap map[string]interface{}, config *AppConfig) error {
	// 定義配置映射關係
	configMappings := map[string]func(interface{}){
		"APP_NAME": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.AppName = s
			}
		},
		"DEBUG": func(v interface{}) {
			if b, ok := v.(bool); ok {
				config.Debug = b
			} else if s, ok := v.(string); ok {
				config.Debug = strings.ToLower(s) == "true"
			}
		},
		"API_PORT": func(v interface{}) {
			log.Printf("Nacos 配置中的 API_PORT 原始值: %v (類型: %T)", v, v)

			if port, ok := parseIntValue(v); ok {
				config.Server.Port = port
				log.Printf("成功設置 API_PORT 為: %d", port)
			} else {
				log.Printf("無法解析 API_PORT 值: %v", v)
			}
		},
		"DEALER_WS_PORT": func(v interface{}) {
			log.Printf("Nacos 配置中的 DEALER_WS_PORT 原始值: %v (類型: %T)", v, v)

			if port, ok := parseIntValue(v); ok {
				config.Server.DealerWsPort = port
				log.Printf("成功設置 DEALER_WS_PORT 為: %d", port)
			} else {
				log.Printf("無法解析 DEALER_WS_PORT 值: %v", v)
			}
		},
		"GRPC_PORT": func(v interface{}) {
			log.Printf("Nacos 配置中的 GRPC_PORT 原始值: %v (類型: %T)", v, v)

			if port, ok := parseIntValue(v); ok {
				config.Server.GrpcPort = port
				log.Printf("成功設置 GRPC_PORT 為: %d", port)
			} else {
				log.Printf("無法解析 GRPC_PORT 值: %v", v)
			}
		},
		"VERSION": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Server.Version = s
			}
		},
		"REDIS_HOST": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Redis.Host = s
			}
		},
		"REDIS_PORT": func(v interface{}) {
			if port, ok := parseIntValue(v); ok {
				config.Redis.Port = port
			}
		},
		"REDIS_USERNAME": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Redis.Username = s
			}
		},
		"REDIS_PASSWORD": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Redis.Password = s
			}
		},
		"REDIS_DB": func(v interface{}) {
			if db, ok := parseIntValue(v); ok {
				config.Redis.DB = db
			}
		},
		"DB_HOST": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.Host = s
			}
		},
		"DB_PORT": func(v interface{}) {
			if port, ok := parseIntValue(v); ok {
				config.Database.Port = port
			}
		},
		"DB_NAME": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.DBName = s
			}
		},
		"DB_USER": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.Username = s
			}
		},
		"DB_PASSWORD": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.Password = s
			}
		},
		// RocketMQ 配置映射
		"ROCKETMQ_ENABLED": func(v interface{}) {
			log.Printf("解析 ROCKETMQ_ENABLED: %v (類型: %T)", v, v)

			if b, ok := v.(bool); ok {
				config.RocketMQ.Enabled = b
				log.Printf("ROCKETMQ_ENABLED 設定: %v", b)
			} else if n, ok := v.(json.Number); ok {
				if i, err := n.Int64(); err == nil {
					config.RocketMQ.Enabled = i != 0
					log.Printf("ROCKETMQ_ENABLED 設定: %v", config.RocketMQ.Enabled)
				}
			} else if s, ok := v.(string); ok {
				log.Printf("ROCKETMQ_ENABLED 設定: %v", strings.ToLower(s) == "true")
			}
		},
		"ROCKETMQ_NAME_SERVERS": func(v interface{}) {
			// 處理字符串格式的 NameServers
			if s, ok := v.(string); ok && s != "" {
				servers := strings.Split(s, ",")
				for i, server := range servers {
					servers[i] = strings.TrimSpace(server)
				}
				config.RocketMQ.NameServers = servers
				log.Printf("從字符串設置 RocketMQ NameServers: %v", servers)
				return
			}

			// 處理數組格式的 NameServers
			if arr, ok := v.([]interface{}); ok {
				servers := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						servers = append(servers, s)
					}
				}
				config.RocketMQ.NameServers = servers
				log.Printf("從數組設置 RocketMQ NameServers: %v", servers)
				return
			}

			// 嘗試處理可能的 JSON 字符串數組
			if s, ok := v.(string); ok && strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
				var servers []string
				if err := json.Unmarshal([]byte(s), &servers); err == nil {
					config.RocketMQ.NameServers = servers
					log.Printf("從 JSON 字符串數組設置 RocketMQ NameServers: %v", servers)
					return
				}
			}

			log.Printf("無法解析 ROCKETMQ_NAME_SERVERS: %v (類型: %T)", v, v)
		},
		"ROCKETMQ_PRODUCER_GROUP": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.RocketMQ.ProducerGroup = s
			}
		},
		"ROCKETMQ_CONSUMER_GROUP": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.RocketMQ.ConsumerGroup = s
			}
		},
	}

	// 應用映射
	for key, value := range jsonMap {
		if mapFunc, exists := configMappings[key]; exists {
			mapFunc(value)
		}
	}

	return nil
}

// parseIntValue 解析整數值，支持字符串或數字
func parseIntValue(v interface{}) (int, bool) {
	log.Printf("嘗試解析整數值: %v (類型: %T)", v, v)

	switch val := v.(type) {
	case int:
		log.Printf("直接使用整數值: %d", val)
		return val, true
	case float64:
		log.Printf("將浮點數轉換為整數: %f -> %d", val, int(val))
		return int(val), true
	case string:
		// 嘗試直接轉換字符串
		if i, err := strconv.Atoi(val); err == nil {
			log.Printf("成功將字符串轉換為整數: %s -> %d", val, i)
			return i, true
		}

		// 嘗試解析浮點數然後轉為整數
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			i := int(f)
			log.Printf("通過浮點數將字符串轉換為整數: %s -> %f -> %d", val, f, i)
			return i, true
		}

		log.Printf("無法將字符串轉換為整數: %s", val)
	case json.Number:
		// 處理 json.Number 類型
		if i, err := val.Int64(); err == nil {
			log.Printf("成功將 json.Number 轉換為整數: %s -> %d", val.String(), i)
			return int(i), true
		}

		// 嘗試解析為浮點數
		if f, err := val.Float64(); err == nil {
			i := int(f)
			log.Printf("通過浮點數將 json.Number 轉換為整數: %s -> %f -> %d", val.String(), f, i)
			return i, true
		}

		log.Printf("無法將 json.Number 轉換為整數: %s", val.String())
	default:
		// 嘗試將其他類型轉換為字符串再解析
		str := fmt.Sprintf("%v", v)
		if i, err := strconv.Atoi(str); err == nil {
			log.Printf("成功將其他類型轉換為整數: %v -> %s -> %d", v, str, i)
			return i, true
		}

		log.Printf("無法將其他類型轉換為整數: %v", v)
	}

	return 0, false
}

// parseRedisXmlConfig 解析 Redis XML 配置並更新應用配置
func parseRedisXmlConfig(xmlContent string, cfg *AppConfig) error {
	// 防止空 XML 內容
	if xmlContent == "" {
		return fmt.Errorf("空的 Redis XML 配置內容")
	}

	log.Printf("開始解析 Redis XML 配置，長度: %d 字節", len(xmlContent))
	log.Printf("XML 內容: %s", xmlContent)

	// 解析 XML 配置
	var redisConfig RedisXMLConfig
	if err := xml.Unmarshal([]byte(xmlContent), &redisConfig); err != nil {
		log.Printf("解析 Redis XML 配置錯誤: %v", err)
		return fmt.Errorf("解析 Redis XML 配置錯誤: %w", err)
	}

	// 配置已成功解析，現在更新主配置
	redis := redisConfig.Redis

	log.Printf("成功解析 XML 配置: Host=%s, Port=%s, IsCluster=%s, 節點數=%d",
		redis.Host, redis.Port, redis.IsCluster, len(redis.Nodes.NodeList))

	// 記錄所有節點
	for i, node := range redis.Nodes.NodeList {
		log.Printf("  節點 %d: %s", i+1, node)
	}

	// 如果有 Username 屬性，為避免認證問題，我們明確將其設置為空
	// 這是因為某些 Redis 版本在使用非空但無效的用戶名時會引發認證問題
	cfg.Redis.Username = ""

	// 初始化 Extra 映射
	if cfg.Redis.Extra == nil {
		cfg.Redis.Extra = make(map[string]interface{})
		log.Printf("初始化 Redis.Extra 映射")
	}

	// 讀取集群設定
	cfg.Redis.Extra["is_cluster"] = redis.IsCluster
	log.Printf("設置 is_cluster = %s", redis.IsCluster)

	// 如果是單節點模式
	if redis.IsCluster == "false" {
		log.Printf("解析到 Redis 單節點配置: host=%s, port=%s", redis.Host, redis.Port)

		// 更新主機和端口
		if redis.Host != "" {
			cfg.Redis.Host = redis.Host
		}

		if redis.Port != "" {
			port, err := strconv.Atoi(redis.Port)
			if err != nil {
				log.Printf("警告: 無法解析端口 '%s', 將使用默認端口", redis.Port)
			} else {
				cfg.Redis.Port = port
			}
		}

		// 更新密碼和數據庫
		if redis.Password != "" {
			cfg.Redis.Password = redis.Password
		}

		if redis.DB != "" {
			db, err := strconv.Atoi(redis.DB)
			if err != nil {
				log.Printf("警告: 無法解析數據庫編號 '%s', 將使用默認值", redis.DB)
			} else {
				cfg.Redis.DB = db
			}
		}

		log.Printf("已更新 Redis 單節點配置: host=%s, port=%d, database=%d",
			cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)
	} else if redis.IsCluster == "true" {
		// 集群模式
		if len(redis.Nodes.NodeList) == 0 {
			log.Printf("錯誤: Redis 集群配置中未找到節點")
			return fmt.Errorf("Redis 集群配置中未找到節點")
		}

		log.Printf("解析到 Redis 集群配置，節點數量: %d", len(redis.Nodes.NodeList))

		// 保存所有節點信息 - 確保它是字符串切片
		// 檢查節點列表是否為 nil
		if redis.Nodes.NodeList == nil {
			log.Printf("警告: 節點列表為 nil，將創建空切片")
			cfg.Redis.Extra["nodes"] = []string{}
		} else {
			// 複製節點列表
			nodeList := make([]string, len(redis.Nodes.NodeList))
			for i, node := range redis.Nodes.NodeList {
				nodeList[i] = node
				log.Printf("  添加節點 %d: %s", i+1, node)
			}
			cfg.Redis.Extra["nodes"] = nodeList
			log.Printf("已保存 %d 個節點到 Extra[nodes]", len(nodeList))
		}

		// 使用第一個節點的地址作為主連接點
		// 格式應該是 host:port
		firstNode := redis.Nodes.NodeList[0]
		parts := strings.Split(firstNode, ":")

		if len(parts) != 2 {
			log.Printf("錯誤: Redis 節點地址格式無效: %s", firstNode)
			return fmt.Errorf("Redis 節點地址格式無效: %s", firstNode)
		}

		cfg.Redis.Host = parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("錯誤: 無法解析 Redis 節點端口: %s", parts[1])
			return fmt.Errorf("無法解析 Redis 節點端口: %s", parts[1])
		}
		cfg.Redis.Port = port

		// 更新密碼和數據庫
		if redis.Password != "" {
			cfg.Redis.Password = redis.Password
		}

		if redis.DB != "" {
			db, err := strconv.Atoi(redis.DB)
			if err != nil {
				log.Printf("警告: 無法解析數據庫編號 '%s', 將使用默認值", redis.DB)
			} else {
				cfg.Redis.DB = db
			}
		}

		log.Printf("已更新 Redis 集群配置: 主節點=%s:%d, 數據庫=%d",
			cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)
	} else {
		log.Printf("錯誤: 未知的 Redis 模式: %s", redis.IsCluster)
		return fmt.Errorf("未知的 Redis 模式: %s", redis.IsCluster)
	}

	return nil
}

// parseTiDBXmlConfig 解析 TiDB XML 配置並更新應用配置
func parseTiDBXmlConfig(xmlContent string, cfg *AppConfig, serviceName string) error {
	// 防止空 XML 內容
	if xmlContent == "" {
		return fmt.Errorf("空的 TiDB XML 配置內容")
	}

	log.Printf("開始解析 TiDB XML 配置，長度: %d 字節", len(xmlContent))

	// 解析 XML 配置
	var tidbConfig TiDBXMLConfig
	if err := xml.Unmarshal([]byte(xmlContent), &tidbConfig); err != nil {
		log.Printf("解析 TiDB XML 配置錯誤: %v", err)
		return fmt.Errorf("解析 TiDB XML 配置錯誤: %w", err)
	}

	// 尋找匹配的服務數據庫配置
	var foundDB bool
	for _, db := range tidbConfig.DBs {
		// 檢查服務名稱是否匹配，處理特殊情況
		if db.Name == serviceName ||
			(serviceName == "g38_lottery_service" && db.Name == "g38_loterry_service") ||
			(serviceName == "g38_loterry_service" && db.Name == "g38_lottery_service") {
			log.Printf("找到服務 %s 對應的數據庫配置 (XML 中名稱: %s)", serviceName, db.Name)

			// 更新數據庫配置
			cfg.Database.Host = db.Host

			// 解析端口
			if db.Port != "" {
				port, err := strconv.Atoi(db.Port)
				if err != nil {
					log.Printf("警告: 無法解析 TiDB 端口 '%s', 將使用默認端口", db.Port)
				} else {
					cfg.Database.Port = port
				}
			}

			cfg.Database.DBName = db.DBName
			cfg.Database.Username = db.User
			cfg.Database.Password = db.Password

			log.Printf("已更新 TiDB 配置: Host=%s, Port=%d, DBName=%s, User=%s",
				cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.Username)

			foundDB = true
			break
		}
	}

	if !foundDB {
		log.Printf("警告: 在 TiDB XML 配置中未找到服務 %s 的配置", serviceName)

		// 如果沒有找到匹配的服務，輸出所有可用的服務名稱以便調試
		var availableServices []string
		for _, db := range tidbConfig.DBs {
			availableServices = append(availableServices, db.Name)
		}
		log.Printf("可用的服務名稱: %v", availableServices)
	}

	return nil
}

// parseRocketMQXmlConfig 解析 RocketMQ XML 配置
func parseRocketMQXmlConfig(xmlContent string, cfg *AppConfig) error {
	var rocketConfig RocketMQXMLConfig
	if err := xml.Unmarshal([]byte(xmlContent), &rocketConfig); err != nil {
		return fmt.Errorf("解析 RocketMQ XML 配置失敗: %w", err)
	}

	// 日誌輸出解析結果
	log.Printf("從 XML 解析 RocketMQ 配置: Host=%s, Port=%s",
		rocketConfig.RocketMQ.NameSrv.Host,
		rocketConfig.RocketMQ.NameSrv.Port)

	// 只有當 host 和 port 都有值時才設置 NameServers
	if rocketConfig.RocketMQ.NameSrv.Host != "" {
		host := rocketConfig.RocketMQ.NameSrv.Host
		port := rocketConfig.RocketMQ.NameSrv.Port
		if port == "" {
			port = "9876" // 使用默認端口
		}

		// 確保配置已初始化
		if cfg.RocketMQ.NameServers == nil {
			cfg.RocketMQ.NameServers = make([]string, 0)
		}

		// 設置 NameServers
		nameServer := fmt.Sprintf("%s:%s", host, port)
		cfg.RocketMQ.NameServers = []string{nameServer}
		cfg.RocketMQ.Enabled = true

		log.Printf("成功設置 RocketMQ NameServer: %s", nameServer)
	} else {
		log.Println("RocketMQ XML 配置中未找到 NameServer 主機資訊")
	}

	return nil
}

// mapJsonMapToAppConfig 將 map[string]interface{} 格式的配置映射到 AppConfig 結構
// 這是為了支持XML和JSON兩種格式的配置解析
func mapJsonMapToAppConfig(configMap map[string]interface{}, config *AppConfig) error {
	// 保存原始的WebSocket端口設置，以便後續還原
	dealerWsPort := config.Server.DealerWsPort
	grpcPort := config.Server.GrpcPort
	apiPort := config.Server.Port

	// 打印關鍵配置項
	if apiPortValue, ok := configMap["API_PORT"]; ok {
		log.Printf("從配置獲取的 API_PORT: %v (類型: %T)", apiPortValue, apiPortValue)
	} else {
		log.Printf("配置中未找到 API_PORT 設定")
	}

	if dealerPort, ok := configMap["DEALER_WS_PORT"]; ok {
		log.Printf("從配置獲取的 DEALER_WS_PORT: %v (類型: %T)", dealerPort, dealerPort)
	} else {
		log.Printf("配置中未找到 DEALER_WS_PORT 設定")
	}

	if grpcPortValue, ok := configMap["GRPC_PORT"]; ok {
		log.Printf("從配置獲取的 GRPC_PORT: %v (類型: %T)", grpcPortValue, grpcPortValue)
	} else {
		log.Printf("配置中未找到 GRPC_PORT 設定")
	}

	// 將配置映射到 AppConfig 結構
	if err := mapJsonToAppConfig(configMap, config); err != nil {
		return fmt.Errorf("映射配置到應用配置失敗: %w", err)
	}

	// 恢復 WebSocket 端口設置（如果配置中未提供或無效）
	restoreWebSocketPorts(config, dealerWsPort, grpcPort, apiPort)

	return nil
}

// ParseConfig 從 XML 字節數據解析應用配置
func ParseConfig(data []byte) (*AppConfig, error) {
	// 這裡我們簡單地將 XML 解析為 AppConfig
	// 實際實現可能需要根據您特定的 XML 格式進行調整

	// 首先嘗試檢測是 XML 還是 JSON 格式
	configMap, err := detectAndParseConfig(string(data))
	if err != nil {
		return nil, fmt.Errorf("解析配置失敗: %w", err)
	}

	// 創建默認配置
	config := createDefaultConfig()

	// 將解析的配置映射到 AppConfig 結構
	if err := mapJsonMapToAppConfig(configMap, config); err != nil {
		return nil, fmt.Errorf("映射配置到應用配置失敗: %w", err)
	}

	return config, nil
}

// detectAndParseConfig 檢測並解析配置格式 (XML 或 JSON)
func detectAndParseConfig(data string) (map[string]interface{}, error) {
	var result map[string]interface{}

	// 嘗試解析為 JSON
	if err := json.Unmarshal([]byte(data), &result); err == nil {
		return result, nil
	}

	// 嘗試解析為 XML
	var xmlMap map[string]interface{}
	if err := xml.Unmarshal([]byte(data), &xmlMap); err == nil {
		return xmlMap, nil
	}

	// 都失敗了，返回錯誤
	return nil, fmt.Errorf("無法解析配置: 不支持的格式")
}

// ===== 依賴注入相關函數 =====

// ProvideAppConfig 提供應用配置，用於 fx 依賴注入
func ProvideAppConfig(nacosClient *nacosmgr.NacosClient) (*AppConfig, error) {
	return LoadConfig(nacosClient)
}

// RegisterService 在 Nacos 註冊服務
func RegisterService(config *AppConfig, nacosClient *nacosmgr.NacosClient) error {
	if !config.Server.RegisterService || !config.Nacos.Enabled || nacosClient == nil {
		log.Println("服務註冊未啟用或 Nacos 客戶端不可用")
		return nil
	}

	// 註冊服務實例
	success, err := registerServiceInstance(config, nacosClient)
	if err != nil {
		return fmt.Errorf("註冊服務實例失敗: %w", err)
	}

	if success {
		log.Printf("成功將服務 %s (ID: %s) 註冊到 Nacos 服務列表",
			config.Server.ServiceName, config.Server.ServiceID)
	} else {
		log.Printf("註冊服務到 Nacos 失敗")
	}

	return nil
}

// registerServiceInstance 註冊服務實例
func registerServiceInstance(config *AppConfig, nacosClient *nacosmgr.NacosClient) (bool, error) {
	namingClient := nacosClient.GetNamingClient()
	if namingClient == nil {
		return false, fmt.Errorf("無法獲取 Nacos 命名服務客戶端")
	}

	return namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          config.Server.ServiceIP,
		Port:        uint64(config.Server.ServicePort),
		ServiceName: config.Server.ServiceName,
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

// Module 創建 fx 模組，包含所有配置相關組件
var Module = fx.Module("config",
	fx.Provide(
		ProvideAppConfig,
	),
	fx.Invoke(
		RegisterService,
	),
)
