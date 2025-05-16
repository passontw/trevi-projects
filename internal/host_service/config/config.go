package config

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.trevi.cc/server/go_gamecommon/cache"
	"git.trevi.cc/server/go_gamecommon/db"
	"git.trevi.cc/server/go_gamecommon/msgqueue"
	"git.trevi.cc/server/go_gamecommon/nacosmgr"
	"github.com/joho/godotenv"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ===== 配置結構定義 =====

// AppConfig 應用程式配置結構
type AppConfig struct {
	AppName   string          `json:"appName"`
	Debug     bool            `json:"debug"`
	Server    ServerConfig    `json:"server"`
	Websocket WebsocketConfig `json:"websocket"`
	Redis     RedisConfig     `json:"redis"`
	Database  DatabaseConfig  `json:"database"`
	Nacos     NacosConfig     `json:"nacos"`
	RocketMQ  RocketMQConfig  `json:"rocketmq"`
}

// ServerConfig 服務器配置
type ServerConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Version         string `json:"version"`
	ServiceName     string `json:"serviceName"`
	ServiceID       string `json:"serviceId"`
	ServiceIP       string `json:"serviceIp"`
	ServicePort     int    `json:"servicePort"`
	RegisterService bool   `json:"registerService"`
	ShutdownTimeout int    `json:"shutdownTimeout"` // 優雅關閉超時秒數
}

// WebsocketConfig WebSocket 配置
type WebsocketConfig struct {
	Port int    `json:"port"`
	Path string `json:"path"`
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

// CommandLineArgs 存儲從命令行解析的參數
type CommandLineArgs struct {
	// Nacos 相關配置
	NacosAddr           string
	NacosNamespace      string
	NacosGroup          string
	NacosUsername       string
	NacosPassword       string
	NacosDataId         string
	NacosRedisDataId    string
	NacosTidbDataId     string
	NacosRocketMQDataId string
	EnableNacos         bool

	// 服務設定
	ServiceName   string
	ServicePort   string
	WebsocketPort string
	ServerMode    string
	LogLevel      string
	LogDir        string
	LogFormat     string

	// 是否已處理命令行參數
	parsed bool
}

// 全局變量，用於存儲解析後的命令行參數
var Args CommandLineArgs

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

// formatBool 將布爾值轉換為字符串
func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// min 返回兩個整數中較小的一個
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ===== 命令行參數處理函數 =====

// InitFlags 初始化並解析命令行參數
func InitFlags() {
	// 如果已經解析過命令行參數，則直接返回
	if Args.parsed {
		return
	}

	// 嘗試先載入 .env 文件
	log.Println("嘗試載入 .env 文件...")
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 找不到或無法載入 .env 文件: %v", err)
		log.Println("將使用命令行參數和環境變數")
	} else {
		log.Println("成功載入 .env 文件")
	}

	// Nacos 相關配置
	flag.StringVar(&Args.NacosAddr, "nacos_addr", getEnv("NACOS_ADDR", "http://127.0.0.1:8848"), "Nacos server address")
	flag.StringVar(&Args.NacosNamespace, "nacos_namespace", getEnv("NACOS_NAMESPACE", "public"), "Nacos namespace")
	flag.StringVar(&Args.NacosGroup, "nacos_group", getEnv("NACOS_GROUP", "DEFAULT_GROUP"), "Nacos group")
	flag.StringVar(&Args.NacosUsername, "nacos_username", getEnv("NACOS_USERNAME", "nacos"), "Nacos username")
	flag.StringVar(&Args.NacosPassword, "nacos_password", getEnv("NACOS_PASSWORD", "nacos"), "Nacos password")
	flag.StringVar(&Args.NacosDataId, "nacos_dataid", getEnv("NACOS_DATAID", "lotterysvr.xml"), "Nacos data ID")
	flag.StringVar(&Args.NacosRedisDataId, "nacos_redis_dataid", getEnv("NACOS_REDIS_DATAID", "redisconfig.xml"), "Nacos Redis config data ID")
	flag.StringVar(&Args.NacosTidbDataId, "nacos_tidb_dataid", getEnv("NACOS_TIDB_DATAID", "dbconfig.xml"), "Nacos TiDB config data ID")
	flag.StringVar(&Args.NacosRocketMQDataId, "nacos_rocketmq_dataid", getEnv("NACOS_ROCKETMQ_DATAID", "rocketmq.xml"), "Nacos RocketMQ config data ID")

	// 服務設定
	flag.StringVar(&Args.ServiceName, "service_name", getEnv("SERVICE_NAME", "g38_host_service"), "Service name")
	flag.StringVar(&Args.ServicePort, "service_port", getEnv("SERVICE_PORT", "8081"), "Service port")
	flag.StringVar(&Args.WebsocketPort, "websocket_port", getEnv("WEBSOCKET_PORT", "9001"), "WebSocket port")
	flag.StringVar(&Args.ServerMode, "server_mode", getEnv("SERVER_MODE", "dev"), "Server mode (dev, prod)")
	flag.StringVar(&Args.LogLevel, "log_level", getEnv("LOG_LEVEL", "debug"), "Log level")
	flag.StringVar(&Args.LogDir, "log_dir", getEnv("LOG_DIR", "logs"), "Log directory")
	flag.StringVar(&Args.LogFormat, "log_format", getEnv("LOG_FORMAT", "hour"), "Log time format: hour or day")

	// 布爾型參數
	enableNacos := getEnvAsBool("ENABLE_NACOS", true)
	flag.BoolVar(&Args.EnableNacos, "enable_nacos", enableNacos, "Enable Nacos configuration")

	// 解析命令行參數
	flag.Parse()

	// 將解析後的命令行參數設置到環境變量中
	setEnvironmentVariables()

	// 標記為已解析
	Args.parsed = true

	// 輸出解析後的參數（僅在開發模式下）
	if Args.ServerMode == "dev" {
		log.Println("已解析的命令行參數:")
		log.Printf("  Nacos 地址: %s", Args.NacosAddr)
		log.Printf("  Nacos 命名空間: %s", Args.NacosNamespace)
		log.Printf("  Nacos 組: %s", Args.NacosGroup)
		log.Printf("  Nacos 用戶名: %s", Args.NacosUsername)
		log.Printf("  Nacos 數據ID: %s", Args.NacosDataId)
		log.Printf("  Nacos Redis 數據ID: %s", Args.NacosRedisDataId)
		log.Printf("  Nacos TiDB 數據ID: %s", Args.NacosTidbDataId)
		log.Printf("  Nacos RocketMQ 數據ID: %s", Args.NacosRocketMQDataId)
		log.Printf("  是否啟用 Nacos: %v", Args.EnableNacos)
		log.Printf("  服務名稱: %s", Args.ServiceName)
		log.Printf("  服務端口: %s", Args.ServicePort)
		log.Printf("  WebSocket 端口: %s", Args.WebsocketPort)
		log.Printf("  服務器模式: %s", Args.ServerMode)
		log.Printf("  日誌級別: %s", Args.LogLevel)
		log.Printf("  日誌目錄: %s", Args.LogDir)
		log.Printf("  日誌時間格式: %s", Args.LogFormat)
	}
}

// setEnvironmentVariables 將解析後的命令行參數設置到環境變量中
func setEnvironmentVariables() {
	// Nacos 相關配置
	os.Setenv("NACOS_ADDR", Args.NacosAddr)
	os.Setenv("NACOS_NAMESPACE", Args.NacosNamespace)
	os.Setenv("NACOS_GROUP", Args.NacosGroup)
	os.Setenv("NACOS_USERNAME", Args.NacosUsername)
	os.Setenv("NACOS_PASSWORD", Args.NacosPassword)
	os.Setenv("NACOS_DATAID", Args.NacosDataId)
	os.Setenv("NACOS_REDIS_DATAID", Args.NacosRedisDataId)
	os.Setenv("NACOS_TIDB_DATAID", Args.NacosTidbDataId)
	os.Setenv("NACOS_ROCKETMQ_DATAID", Args.NacosRocketMQDataId)
	os.Setenv("ENABLE_NACOS", formatBool(Args.EnableNacos))

	// 服務設定
	os.Setenv("SERVICE_NAME", Args.ServiceName)
	os.Setenv("SERVICE_PORT", Args.ServicePort)
	os.Setenv("WEBSOCKET_PORT", Args.WebsocketPort)
	os.Setenv("SERVER_MODE", Args.ServerMode)
	os.Setenv("LOG_LEVEL", Args.LogLevel)
	os.Setenv("LOG_DIR", Args.LogDir)
	os.Setenv("LOG_FORMAT", Args.LogFormat)
}

// GetNacosHostAndPort 從 NacosAddr 中提取主機、端口和協議
// 返回: 主機, 端口, 是否使用HTTPS
func GetNacosHostAndPort() (string, string, bool) {
	addr := Args.NacosAddr
	isHttps := strings.HasPrefix(addr, "https://")

	// 去除協議前綴
	if strings.HasPrefix(addr, "http://") {
		addr = addr[7:]
	} else if isHttps {
		addr = addr[8:]
	}

	// 分離主機和端口
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// 如果沒有指定端口，則使用默認端口
		host = addr
		port = "8848"
	}

	return host, port, isHttps
}

// GetNacosServer 返回完整的 Nacos 服務器地址
func GetNacosServer() string {
	return Args.NacosAddr
}

// CheckNacosConnection 檢查 Nacos 服務器連通性
func CheckNacosConnection() error {
	// 構建 ping URL
	host, port, isHttps := GetNacosHostAndPort()
	protocol := "http"
	if isHttps {
		protocol = "https"
	}
	pingURL := fmt.Sprintf("%s://%s:%s/nacos/v1/ns/operator/metrics", protocol, host, port)

	// 創建 HTTP 客戶端並設置超時
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 發送 GET 請求
	resp, err := client.Get(pingURL)
	if err != nil {
		return fmt.Errorf("無法連接到 Nacos 服務器 %s: %w", pingURL, err)
	}
	defer resp.Body.Close()

	// 檢查響應狀態碼
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Nacos 服務器連接檢查失敗，狀態碼: %d", resp.StatusCode)
	}

	return nil
}

// ConnectionModule 創建 fx 模組，用於處理連接相關組件
var ConnectionModule = fx.Module("connections",
	fx.Provide(
		// 提供配置和連接
		NewAppConfig,
		NewRedisClient,
		NewDatabaseManager,
		NewRocketMQResolver,
	),
	fx.Invoke(
		// 輸出連接成功訊息
		LogConnectionStatus,
	),
)

// NewAppConfig 創建應用配置
func NewAppConfig(logger *zap.Logger) (*AppConfig, error) {
	// 獲取 Nacos 服務器
	nacosServer := GetNacosServer()

	// 創建 Nacos 客戶端
	nacosclient := nacosmgr.NewNacosClient(
		Args.LogDir,
		nacosServer,
		Args.NacosNamespace,
		Args.NacosUsername,
		Args.NacosPassword,
	)

	// 載入主配置
	logger.Info("正在從 Nacos 獲取主配置",
		zap.String("group", Args.NacosGroup),
		zap.String("dataId", Args.NacosDataId))

	configContent, err := nacosclient.GetConfig(Args.NacosGroup, Args.NacosDataId)
	if err != nil {
		logger.Error("無法獲取配置", zap.Error(err))
		return nil, err
	}

	// 檢測配置格式並解析
	configMap, err := detectAndParseConfig(configContent)
	if err != nil {
		logger.Error("檢測或解析配置格式失敗", zap.Error(err))
		return nil, err
	}

	// 創建默認配置
	config := &AppConfig{
		AppName: getEnv("APP_NAME", "g38_host_service"),
		Debug:   getEnvAsBool("DEBUG", true),
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvAsInt("SERVER_PORT", 8081),
			Version:         getEnv("SERVER_VERSION", "v1"),
			ServiceName:     getEnv("SERVICE_NAME", "host-service"),
			ServiceID:       getEnv("SERVICE_ID", "host-service-1"),
			ServiceIP:       getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort:     getEnvAsInt("SERVICE_PORT", 8081),
			RegisterService: getEnvAsBool("REGISTER_SERVICE", true),
			ShutdownTimeout: getEnvAsInt("SHUTDOWN_TIMEOUT", 30),
		},
		Websocket: WebsocketConfig{
			Port: getEnvAsInt("WEBSOCKET_PORT", 9001),
			Path: getEnv("WEBSOCKET_PATH", "/ws"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Username: getEnv("REDIS_USERNAME", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			Extra:    make(map[string]interface{}),
		},
		Database: DatabaseConfig{
			Driver:   getEnv("DB_DRIVER", "mysql"),
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("DB_PORT", 4000),
			Username: getEnv("DB_USERNAME", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "host"),
		},
		Nacos: NacosConfig{
			Enabled:     Args.EnableNacos,
			Host:        getEnv("NACOS_HOST", "127.0.0.1"),
			Port:        uint64(getEnvAsInt("NACOS_PORT", 8848)),
			Namespace:   Args.NacosNamespace,
			Group:       Args.NacosGroup,
			DataId:      Args.NacosDataId,
			Username:    Args.NacosUsername,
			Password:    Args.NacosPassword,
			ServiceIP:   getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort: getEnvAsInt("SERVICE_PORT", 8081),
		},
		RocketMQ: RocketMQConfig{
			Enabled:       getEnvAsBool("ROCKETMQ_ENABLED", false),
			NameServers:   []string{getEnv("ROCKETMQ_NAME_SERVERS", "localhost:9876")},
			AccessKey:     getEnv("ROCKETMQ_ACCESS_KEY", ""),
			SecretKey:     getEnv("ROCKETMQ_SECRET_KEY", ""),
			ProducerGroup: getEnv("ROCKETMQ_PRODUCER_GROUP", "host-producer"),
			ConsumerGroup: getEnv("ROCKETMQ_CONSUMER_GROUP", "host-consumer"),
		},
	}

	// 使用從 Nacos 獲取的配置更新默認配置
	logger.Info("使用從 Nacos 獲取的配置更新默認配置")
	for key, value := range configMap {
		logger.Debug("配置項", zap.String("key", key), zap.Any("value", value))
	}

	// 應用配置映射
	applyConfigMapping(configMap, config, logger)

	// 載入 Redis 配置
	logger.Info("正在從 Nacos 獲取 Redis 配置", zap.String("dataId", Args.NacosRedisDataId))
	redisContent, err := nacosclient.GetConfig(Args.NacosGroup, Args.NacosRedisDataId)
	if err != nil {
		logger.Warn("獲取 Redis 配置失敗", zap.Error(err))
	} else {
		// 解析 Redis XML 配置
		if err := parseRedisXmlConfig(redisContent, config); err != nil {
			logger.Warn("解析 Redis 配置失敗", zap.Error(err))
		} else {
			logger.Info("已成功解析 Redis 配置")
		}
	}

	// 載入數據庫配置
	logger.Info("正在從 Nacos 獲取數據庫配置", zap.String("dataId", Args.NacosTidbDataId))
	dbContent, err := nacosclient.GetConfig(Args.NacosGroup, Args.NacosTidbDataId)
	if err != nil {
		logger.Warn("獲取數據庫配置失敗", zap.Error(err))
	} else {
		// 解析 TiDB XML 配置
		if err := parseTiDBXmlConfig(dbContent, config, Args.ServiceName); err != nil {
			logger.Warn("解析數據庫配置失敗", zap.Error(err))
		} else {
			logger.Info("已成功解析數據庫配置")
		}
	}

	// 載入 RocketMQ 配置
	logger.Info("正在從 Nacos 獲取 RocketMQ 配置", zap.String("dataId", Args.NacosRocketMQDataId))
	rocketMQContent, err := nacosclient.GetConfig(Args.NacosGroup, Args.NacosRocketMQDataId)
	if err != nil {
		logger.Warn("獲取 RocketMQ 配置失敗", zap.Error(err))
	} else {
		// 解析 RocketMQ XML 配置
		if err := parseRocketMQXmlConfig(rocketMQContent, config); err != nil {
			logger.Warn("解析 RocketMQ 配置失敗", zap.Error(err))
		} else {
			logger.Info("已成功解析 RocketMQ 配置")
		}
	}

	return config, nil
}

// applyConfigMapping 將配置映射應用到 AppConfig 對象
func applyConfigMapping(configMap map[string]interface{}, config *AppConfig, logger *zap.Logger) {
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
			if port, ok := parseIntValue(v); ok {
				config.Server.Port = port
				logger.Debug("設置 API_PORT", zap.Int("value", port))
			}
		},
		"WEBSOCKET_PORT": func(v interface{}) {
			if port, ok := parseIntValue(v); ok {
				config.Websocket.Port = port
				logger.Debug("設置 WEBSOCKET_PORT", zap.Int("value", port))
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
		"DB_USERNAME": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.Username = s
			}
		},
		"DB_PASSWORD": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.Password = s
			}
		},
		"ROCKETMQ_ENABLED": func(v interface{}) {
			if b, ok := v.(bool); ok {
				config.RocketMQ.Enabled = b
			} else if s, ok := v.(string); ok {
				config.RocketMQ.Enabled = strings.ToLower(s) == "true"
			}
		},
		"ROCKETMQ_NAME_SERVERS": func(v interface{}) {
			if s, ok := v.(string); ok && s != "" {
				config.RocketMQ.NameServers = strings.Split(s, ",")
			} else if arr, ok := v.([]interface{}); ok {
				servers := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						servers = append(servers, s)
					}
				}
				config.RocketMQ.NameServers = servers
			}
		},
	}

	// 應用映射
	for key, value := range configMap {
		if mapFunc, exists := configMappings[key]; exists {
			mapFunc(value)
		}
	}
}

// NewRedisClient 創建 Redis 客戶端
func NewRedisClient(config *AppConfig, logger *zap.Logger) (*redis.Client, error) {
	// 首先嘗試使用 cache 套件的 RedisInit
	redisConfig := &cache.RedisConfig{
		Host:      config.Redis.Host,
		Port:      config.Redis.Port,
		Password:  config.Redis.Password,
		DB:        config.Redis.DB,
		IsCluster: false, // 預設為非集群模式
	}

	// 檢查是否為集群模式
	if isCluster, ok := config.Redis.Extra["is_cluster"]; ok {
		if isClusterStr, isString := isCluster.(string); isString {
			redisConfig.IsCluster = isClusterStr == "true"
		}
	}

	// 檢查是否有節點信息
	if nodes, ok := config.Redis.Extra["nodes"]; ok {
		if nodeList, isNodeList := nodes.([]string); isNodeList && len(nodeList) > 0 {
			logger.Info("使用 Redis 集群模式", zap.Strings("nodes", nodeList))
		}
	}

	// 初始化 Redis
	err := cache.RedisInit(*redisConfig, 3*time.Second)
	if err != nil {
		logger.Warn("使用 cache 套件初始化 Redis 失敗，將使用直接連接方式", zap.Error(err))
	} else {
		logger.Info("使用 cache 套件初始化 Redis 成功")
	}

	// 無論 cache 套件是否成功，都再創建一個直接的客戶端
	// 這樣可以確保我們有一個可以返回的客戶端實例
	redisAddr := fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port)
	redisOptions := &redis.Options{
		Addr:     redisAddr,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	}

	// 如果有用戶名，設置用戶名
	if config.Redis.Username != "" {
		redisOptions.Username = config.Redis.Username
	}

	// 創建直接的 Redis 客戶端
	redisClient := redis.NewClient(redisOptions)

	// 測試連接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		logger.Error("直接連接 Redis 失敗", zap.Error(err))
		return nil, err
	}

	logger.Info("Redis 連接成功",
		zap.String("host", config.Redis.Host),
		zap.Int("port", config.Redis.Port),
		zap.String("addr", redisAddr))

	return redisClient, nil
}

// NewDatabaseManager 創建數據庫管理器
func NewDatabaseManager(config *AppConfig, logger *zap.Logger) (*db.DBMgr, error) {
	dbConfig := db.DBConfig{
		Name:     config.AppName,
		Type:     config.Database.Driver,
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		Username: config.Database.Username,
		Password: config.Database.Password,
		Database: config.Database.DBName,
	}

	dbMgr, err := db.NewDBMgr(dbConfig)
	if err != nil {
		logger.Error("初始化數據庫失敗", zap.Error(err))
		return nil, err
	}

	logger.Info("數據庫連接成功",
		zap.String("name", dbConfig.Name),
		zap.String("type", dbConfig.Type),
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port))

	return dbMgr, nil
}

// NewRocketMQResolver 創建 RocketMQ DNS 解析器
func NewRocketMQResolver(config *AppConfig, logger *zap.Logger) (*msgqueue.DnsResolver, error) {
	// 預設使用空解析器
	if !config.RocketMQ.Enabled || len(config.RocketMQ.NameServers) == 0 {
		logger.Info("RocketMQ 未啟用或未配置 NameServers")
		return msgqueue.NewDnsResolver([]msgqueue.NamesrvNode{}), nil
	}

	// 從配置中獲取名稱服務器列表並轉換為 NamesrvNode 結構
	var namesrvNodes []msgqueue.NamesrvNode
	for _, server := range config.RocketMQ.NameServers {
		// 解析 host:port 格式
		parts := strings.Split(server, ":")
		host := parts[0]
		port := 9876 // 默認端口

		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				port = p
			}
		}

		namesrvNodes = append(namesrvNodes, msgqueue.NamesrvNode{
			Host: host,
			Port: port,
		})
	}

	// 創建 DNS 解析器
	dnsResolver := msgqueue.NewDnsResolver(namesrvNodes)
	logger.Info("RocketMQ DNS 解析器創建成功", zap.Strings("nameServers", config.RocketMQ.NameServers))

	return dnsResolver, nil
}

// LogConnectionStatus 輸出連接成功訊息
func LogConnectionStatus(
	logger *zap.Logger,
	config *AppConfig,
	redisClient *redis.Client,
	dbMgr *db.DBMgr,
	dnsResolver *msgqueue.DnsResolver,
) {
	logger.Info("所有連接初始化完成")

	logger.Info("應用配置信息",
		zap.String("appName", config.AppName),
		zap.Bool("debug", config.Debug),
		zap.Int("httpPort", config.Server.Port),
		zap.Int("wsPort", config.Websocket.Port))

	if redisClient != nil {
		logger.Info("Redis 連接狀態", zap.String("status", "已連接"))
	} else {
		logger.Error("Redis 連接狀態", zap.String("status", "未連接"))
	}

	if dbMgr != nil {
		logger.Info("數據庫連接狀態", zap.String("status", "已連接"))
	} else {
		logger.Error("數據庫連接狀態", zap.String("status", "未連接"))
	}

	if dnsResolver != nil {
		logger.Info("RocketMQ DNS 解析器狀態", zap.String("status", "已創建"))
		if config.RocketMQ.Enabled {
			logger.Info("RocketMQ 配置",
				zap.Strings("nameServers", config.RocketMQ.NameServers),
				zap.String("producerGroup", config.RocketMQ.ProducerGroup),
				zap.String("consumerGroup", config.RocketMQ.ConsumerGroup))
		} else {
			logger.Info("RocketMQ 未啟用")
		}
	} else {
		logger.Error("RocketMQ DNS 解析器狀態", zap.String("status", "創建失敗"))
	}
}

// ===== Nacos 相關函數 =====

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

// ParseConfig 從配置內容解析應用配置
func ParseConfig(data []byte) (*AppConfig, error) {
	// 嘗試檢測配置格式
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

// detectAndParseConfig 檢測數據格式（JSON 或 XML）並解析
func detectAndParseConfig(data string) (map[string]interface{}, error) {
	// 檢查是否為 JSON 格式
	if strings.TrimSpace(data)[0] == '{' {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(data), &result)
		if err != nil {
			return nil, fmt.Errorf("JSON 解析失敗: %w", err)
		}
		return result, nil
	}

	// 檢查是否為 XML 格式
	if strings.Contains(data, "<?xml") || strings.Contains(data, "<config>") {
		// 定義一個通用的 XML 結構來解析頂層標籤
		type XMLConfig struct {
			XMLName xml.Name
			Items   []xml.Name `xml:",any"`
			Content string     `xml:",innerxml"`
		}

		var config XMLConfig
		err := xml.Unmarshal([]byte(data), &config)
		if err != nil {
			return nil, fmt.Errorf("XML 解析失敗: %w", err)
		}

		// 將 XML 轉換為 map
		result := make(map[string]interface{})
		result["_format"] = "xml"
		result["_root"] = config.XMLName.Local
		result["_content"] = config.Content

		// 使用正則表達式匹配所有標籤和值
		pattern := `<([^>/]+)>([^<]+)</([^>]+)>`
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(data, -1)

		for _, match := range matches {
			if len(match) >= 3 {
				key := strings.TrimSpace(match[1])
				value := strings.TrimSpace(match[2])
				log.Printf("從 XML 提取鍵值對: %s = %s", key, value)

				// 嘗試轉換值到適當的類型
				if intVal, err := strconv.Atoi(value); err == nil {
					result[key] = intVal
				} else if boolVal, err := strconv.ParseBool(value); err == nil && (value == "true" || value == "false") {
					result[key] = boolVal
				} else {
					result[key] = value
				}
			}
		}

		return result, nil
	}

	// 嘗試作為簡單的 key=value 格式
	log.Printf("嘗試解析為簡單的 key=value 格式...")
	lines := strings.Split(data, "\n")
	result := make(map[string]interface{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// 嘗試轉換值到適當的類型
			if intVal, err := strconv.Atoi(value); err == nil {
				result[key] = intVal
			} else if boolVal, err := strconv.ParseBool(value); err == nil && (value == "true" || value == "false") {
				result[key] = boolVal
			} else {
				result[key] = value
			}
		}
	}

	if len(result) > 0 {
		log.Printf("成功解析為 key=value 格式，包含 %d 個配置項", len(result))
		return result, nil
	}

	return nil, fmt.Errorf("無法解析配置: 不支持的格式")
}

// createDefaultConfig 創建預設配置
func createDefaultConfig() *AppConfig {
	return &AppConfig{
		AppName: getEnv("APP_NAME", "g38_host_service"),
		Debug:   getEnvAsBool("DEBUG", true),
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvAsInt("SERVER_PORT", 8081),
			Version:         getEnv("SERVER_VERSION", "v1"),
			ServiceName:     getEnv("SERVICE_NAME", "host-service"),
			ServiceID:       getEnv("SERVICE_ID", "host-service-1"),
			ServiceIP:       getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort:     getEnvAsInt("SERVICE_PORT", 8081),
			RegisterService: getEnvAsBool("REGISTER_SERVICE", true),
			ShutdownTimeout: getEnvAsInt("SHUTDOWN_TIMEOUT", 30),
		},
		Websocket: WebsocketConfig{
			Port: getEnvAsInt("WEBSOCKET_PORT", 9001),
			Path: getEnv("WEBSOCKET_PATH", "/ws"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Username: getEnv("REDIS_USERNAME", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			Extra:    make(map[string]interface{}),
		},
		Database: DatabaseConfig{
			Driver:   getEnv("DB_DRIVER", "mysql"),
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("DB_PORT", 4000),
			Username: getEnv("DB_USERNAME", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "host"),
		},
		Nacos: NacosConfig{
			Enabled:     getEnvAsBool("ENABLE_NACOS", true),
			Host:        getEnv("NACOS_HOST", "127.0.0.1"),
			Port:        uint64(getEnvAsInt("NACOS_PORT", 8848)),
			Namespace:   getEnv("NACOS_NAMESPACE", "public"),
			Group:       getEnv("NACOS_GROUP", "DEFAULT_GROUP"),
			DataId:      getEnv("NACOS_DATAID", "lotterysvr.xml"),
			Username:    getEnv("NACOS_USERNAME", "nacos"),
			Password:    getEnv("NACOS_PASSWORD", "nacos"),
			ServiceIP:   getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort: getEnvAsInt("SERVICE_PORT", 8081),
		},
		RocketMQ: RocketMQConfig{
			Enabled:       getEnvAsBool("ROCKETMQ_ENABLED", false),
			NameServers:   []string{getEnv("ROCKETMQ_NAME_SERVERS", "localhost:9876")},
			AccessKey:     getEnv("ROCKETMQ_ACCESS_KEY", ""),
			SecretKey:     getEnv("ROCKETMQ_SECRET_KEY", ""),
			ProducerGroup: getEnv("ROCKETMQ_PRODUCER_GROUP", "host-producer"),
			ConsumerGroup: getEnv("ROCKETMQ_CONSUMER_GROUP", "host-consumer"),
		},
	}
}

// mapJsonMapToAppConfig 將 map[string]interface{} 格式的配置映射到 AppConfig 結構
func mapJsonMapToAppConfig(configMap map[string]interface{}, config *AppConfig) error {
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
			if port, ok := parseIntValue(v); ok {
				config.Server.Port = port
			}
		},
		"WEBSOCKET_PORT": func(v interface{}) {
			if port, ok := parseIntValue(v); ok {
				config.Websocket.Port = port
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
		"DB_USERNAME": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.Username = s
			}
		},
		"DB_PASSWORD": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Database.Password = s
			}
		},
		"ROCKETMQ_ENABLED": func(v interface{}) {
			if b, ok := v.(bool); ok {
				config.RocketMQ.Enabled = b
			} else if s, ok := v.(string); ok {
				config.RocketMQ.Enabled = strings.ToLower(s) == "true"
			}
		},
		"ROCKETMQ_NAME_SERVERS": func(v interface{}) {
			if s, ok := v.(string); ok && s != "" {
				config.RocketMQ.NameServers = strings.Split(s, ",")
			} else if arr, ok := v.([]interface{}); ok {
				servers := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						servers = append(servers, s)
					}
				}
				config.RocketMQ.NameServers = servers
			}
		},
	}

	// 應用映射
	for key, value := range configMap {
		if mapFunc, exists := configMappings[key]; exists {
			mapFunc(value)
		}
	}

	return nil
}

// parseIntValue 嘗試將接口值解析為整數
func parseIntValue(v interface{}) (int, bool) {
	switch value := v.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue, true
		}
	case json.Number:
		if intValue, err := value.Int64(); err == nil {
			return int(intValue), true
		}
	}
	return 0, false
}

// parseRedisXmlConfig 解析 Redis XML 配置
func parseRedisXmlConfig(xmlContent string, cfg *AppConfig) error {
	var redisConfig RedisXMLConfig
	err := xml.Unmarshal([]byte(xmlContent), &redisConfig)
	if err != nil {
		return fmt.Errorf("Redis XML 解析失敗: %w", err)
	}

	// 解析端口
	port, err := strconv.Atoi(redisConfig.Redis.Port)
	if err != nil {
		return fmt.Errorf("Redis 端口號格式不正確: %s", redisConfig.Redis.Port)
	}

	// 解析數據庫索引
	db := 0
	if redisConfig.Redis.DB != "" {
		db, err = strconv.Atoi(redisConfig.Redis.DB)
		if err != nil {
			return fmt.Errorf("Redis 數據庫索引格式不正確: %s", redisConfig.Redis.DB)
		}
	}

	// 填充 Redis 配置
	cfg.Redis.Host = redisConfig.Redis.Host
	cfg.Redis.Port = port
	cfg.Redis.Username = redisConfig.Redis.Username
	cfg.Redis.Password = redisConfig.Redis.Password
	cfg.Redis.DB = db

	// 解析集群設置
	if cfg.Redis.Extra == nil {
		cfg.Redis.Extra = make(map[string]interface{})
	}

	isCluster := false
	if redisConfig.Redis.IsCluster == "true" {
		isCluster = true
	}
	cfg.Redis.Extra["is_cluster"] = redisConfig.Redis.IsCluster

	// 解析節點列表
	var nodes []string
	if isCluster && len(redisConfig.Redis.Nodes.NodeList) > 0 {
		nodes = redisConfig.Redis.Nodes.NodeList
	}
	cfg.Redis.Extra["nodes"] = nodes

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
			(serviceName == "g38_host_service" && db.Name == "g38_hostsvr_service") ||
			(serviceName == "g38_hostsvr_service" && db.Name == "g38_host_service") {
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

// RegisterService 將服務註冊到 Nacos
func RegisterService(config *AppConfig, logger *zap.Logger) error {
	if !config.Server.RegisterService {
		logger.Info("服務註冊已禁用")
		return nil
	}

	// 如果 Nacos 未啟用，則不進行註冊
	if !config.Nacos.Enabled {
		logger.Info("Nacos 未啟用，跳過服務註冊")
		return nil
	}

	// 如果服務 IP 或端口未配置，則不進行註冊
	if config.Server.ServiceIP == "" || config.Server.ServicePort == 0 {
		logger.Warn("服務 IP 或端口未配置，無法註冊服務",
			zap.String("serviceIP", config.Server.ServiceIP),
			zap.Int("servicePort", config.Server.ServicePort))
		return fmt.Errorf("服務 IP 或端口未配置")
	}

	logger.Info("準備註冊服務到 Nacos",
		zap.String("serviceName", config.Server.ServiceName),
		zap.String("serviceIP", config.Server.ServiceIP),
		zap.Int("servicePort", config.Server.ServicePort))

	// 創建 Nacos 客戶端
	nacosClient := nacosmgr.NewNacosClient(
		config.Nacos.Host,
		strconv.FormatUint(config.Nacos.Port, 10),
		config.Nacos.Namespace,
		config.Nacos.Username,
		config.Nacos.Password,
	)

	// 獲取 Nacos 命名服務客戶端
	namingClient := nacosClient.GetNamingClient()
	if namingClient == nil {
		logger.Error("無法獲取 Nacos 命名服務客戶端")
		return fmt.Errorf("無法獲取 Nacos 命名服務客戶端")
	}

	// 註冊服務實例
	success, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
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

	if err != nil {
		logger.Error("註冊服務到 Nacos 失敗", zap.Error(err))
		return fmt.Errorf("註冊服務失敗: %w", err)
	}

	if success {
		logger.Info("成功註冊服務到 Nacos",
			zap.String("serviceName", config.Server.ServiceName),
			zap.String("serviceIP", config.Server.ServiceIP),
			zap.Int("servicePort", config.Server.ServicePort))
	} else {
		logger.Warn("註冊服務返回失敗",
			zap.String("serviceName", config.Server.ServiceName))
		return fmt.Errorf("註冊服務返回失敗")
	}

	return nil
}
