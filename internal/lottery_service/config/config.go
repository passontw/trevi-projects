package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"g38_lottery_service/pkg/nacosManager"

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
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Version         string `json:"version"`
	ServiceName     string `json:"serviceName"`
	ServiceID       string `json:"serviceId"`
	ServiceIP       string `json:"serviceIp"`
	ServicePort     int    `json:"servicePort"`
	DealerWsPort    int    `json:"dealerWsPort"` // 荷官 WebSocket 端口
	GrpcPort        int    `json:"grpcPort"`     // gRPC 服務端口
	RegisterService bool   `json:"registerService"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DB       int    `json:"db"`
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
	NameServers   []string `json:"nameServers"`
	AccessKey     string   `json:"accessKey"`
	SecretKey     string   `json:"secretKey"`
	ProducerGroup string   `json:"producerGroup"`
	ConsumerGroup string   `json:"consumerGroup"`
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

// ===== JSON 處理函數 =====

// isValidJson 檢查字符串是否為有效的 JSON
func isValidJson(str string) bool {
	var js interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}

// preprocessJsonContent 預處理 JSON 內容，移除註釋和處理特殊字符
func preprocessJsonContent(content string) string {
	if content == "" {
		return "{}"
	}

	// 如果內容以 / 開頭，可能是註釋或非 JSON 格式
	if len(content) > 0 && content[0] == '/' {
		log.Println("檢測到疑似非 JSON 內容，將返回空 JSON 對象")
		return "{}"
	}

	// 移除行內註釋（//...）和處理空行
	lines := []string{}
	for _, line := range strings.Split(content, "\n") {
		// 查找行內註釋的位置
		commentIdx := strings.Index(line, "//")
		if commentIdx >= 0 {
			// 只保留註釋前的內容
			line = line[:commentIdx]
		}

		// 只添加非空行
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	// 重新組合內容
	return strings.Join(lines, "\n")
}

// ===== Nacos 相關函數 =====

// uploadConfigToNacos 上傳配置到 Nacos
func uploadConfigToNacos(client nacosManager.NacosClient, dataId, group, content string) (bool, error) {
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

// ===== 配置加載主要函數 =====

// LoadConfig 加載配置，優先使用 .env 中的 Nacos 設定取回配置
// 當無法從 Nacos 獲取有效配置時，返回錯誤
func LoadConfig(nacosClient nacosManager.NacosClient) (*AppConfig, error) {
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

		// 從 Nacos 獲取配置
		content, err := nacosClient.GetConfig(config.Nacos.DataId, config.Nacos.Group)
		if err != nil {
			log.Printf("從 Nacos 獲取配置失敗: %v", err)
			return nil, fmt.Errorf("無法從 Nacos 獲取配置: %w", err)
		}

		// 預處理 JSON 內容
		content = preprocessJsonContent(content)

		// 檢查 JSON 有效性
		if !isValidJson(content) {
			log.Printf("從 Nacos 獲取的配置不是有效的 JSON")
			return nil, fmt.Errorf("從 Nacos 獲取的配置不是有效的 JSON")
		}

		// 嘗試解析配置
		if err := parseNacosConfig(content, config); err != nil {
			log.Printf("解析 Nacos 配置失敗: %v", err)
			return nil, fmt.Errorf("解析 Nacos 配置失敗: %w", err)
		}

		log.Println("成功從 Nacos 獲取配置並合併")

		// 保留 Nacos 連接信息
		preserveNacosConnectionInfo(config)
	} else {
		// Nacos 未啟用或客戶端無效
		log.Println("Nacos 未啟用或客戶端無效，無法獲取配置")
		return nil, fmt.Errorf("Nacos 未啟用或客戶端無效，無法獲取配置")
	}

	return config, nil
}

// createDefaultConfig, 創建默認配置, 但不從環境變量讀取 Redis 配置
func createDefaultConfig() *AppConfig {
	// 定義 Nacos 相關設置
	nacosEnabled := getEnvAsBool("ENABLE_NACOS", true)

	return &AppConfig{
		AppName: getEnv("APP_NAME", "g38_lottery_service"),
		Debug:   getEnvAsBool("DEBUG", true),
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvAsInt("SERVER_PORT", 8000),
			Version:         getEnv("SERVER_VERSION", "v1"),
			ServiceName:     getEnv("SERVICE_NAME", "lottery-service"),
			ServiceID:       getEnv("SERVICE_ID", "lottery-service-1"),
			ServiceIP:       getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort:     getEnvAsInt("SERVICE_PORT", 8080),
			DealerWsPort:    getEnvAsInt("DEALER_WS_PORT", 9000), // 默認荷官 WebSocket 端口
			GrpcPort:        getEnvAsInt("GRPC_PORT", 9100),      // 默認 gRPC 端口
			RegisterService: getEnvAsBool("REGISTER_SERVICE", true),
		},
		Redis: RedisConfig{
			Host:     "", // 空值，強制從 Nacos 讀取
			Port:     0,  // 0值，強制從 Nacos 讀取
			Username: "",
			Password: "",
			DB:       0,
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
			Host:        getEnv("NACOS_HOST", "127.0.0.1"),
			Port:        uint64(getEnvAsInt("NACOS_PORT", 8848)),
			Namespace:   getEnv("NACOS_NAMESPACE", "public"),
			Group:       getEnv("NACOS_GROUP", "DEFAULT_GROUP"),
			DataId:      getEnv("NACOS_DATAID", "g38_lottery"),
			Username:    getEnv("NACOS_USERNAME", "nacos"),
			Password:    getEnv("NACOS_PASSWORD", "nacos"),
			ServiceIP:   getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort: getEnvAsInt("SERVICE_PORT", 8080),
		},
		RocketMQ: RocketMQConfig{
			NameServers:   []string{getEnv("ROCKETMQ_NAME_SERVERS", "localhost:9876")},
			AccessKey:     getEnv("ROCKETMQ_ACCESS_KEY", ""),
			SecretKey:     getEnv("ROCKETMQ_SECRET_KEY", ""),
			ProducerGroup: getEnv("ROCKETMQ_PRODUCER_GROUP", "default"),
			ConsumerGroup: getEnv("ROCKETMQ_CONSUMER_GROUP", "default"),
		},
	}
}

// parseNacosConfig 解析從 Nacos 獲取的配置
func parseNacosConfig(content string, config *AppConfig) error {
	// 保存原始的WebSocket端口設置
	dealerWsPort := config.Server.DealerWsPort
	grpcPort := config.Server.GrpcPort

	// 預處理 JSON 內容，處理註釋和換行符
	processedContent := preprocessJsonContent(content)

	// 檢查 JSON 有效性
	if !isValidJson(processedContent) {
		return fmt.Errorf("從 Nacos 獲取的配置不是有效的 JSON")
	}

	// 解析 JSON
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(processedContent), &jsonMap); err != nil {
		return fmt.Errorf("解析 Nacos 配置失敗: %w", err)
	}

	// 打印 Nacos 中的 DEALER_WS_PORT 配置
	if dealerPort, ok := jsonMap["DEALER_WS_PORT"]; ok {
		log.Printf("從 Nacos 獲取的 DEALER_WS_PORT 配置: %v (類型: %T)", dealerPort, dealerPort)
	} else {
		log.Printf("Nacos 配置中未找到 DEALER_WS_PORT 設定")
	}

	// 打印 Nacos 中的 Redis 配置
	log.Printf("從 Nacos 獲取的 Redis 配置項:")
	for _, key := range []string{"REDIS_HOST", "REDIS_PORT", "REDIS_USERNAME", "REDIS_PASSWORD", "REDIS_DB"} {
		if val, ok := jsonMap[key]; ok {
			log.Printf("  %s = %v (類型: %T)", key, val, val)
		} else {
			log.Printf("  %s: 未設定", key)
		}
	}

	// 將 JSON 映射到 AppConfig 結構
	if err := mapJsonToAppConfig(jsonMap, config); err != nil {
		return fmt.Errorf("映射 Nacos 配置到 AppConfig 失敗: %w", err)
	}

	// 恢復 WebSocket 端口設置
	restoreWebSocketPorts(config, dealerWsPort, grpcPort)

	// 打印從 Nacos 獲取的 Redis 配置
	log.Printf("從 Nacos 獲取的 Redis 配置: Host=%s, Port=%d, Username=%s, PasswordLen=%d, DB=%d",
		config.Redis.Host, config.Redis.Port, config.Redis.Username,
		len(config.Redis.Password), config.Redis.DB)

	log.Printf("成功從 Nacos 獲取配置並合併")
	return nil
}

// restoreWebSocketPorts 恢復 WebSocket 端口和 gRPC 端口
func restoreWebSocketPorts(config *AppConfig, dealerWsPort int, grpcPort int) {
	// 保存原始的荷官WebSocket端口
	if dealerWsPort > 0 {
		config.Server.DealerWsPort = dealerWsPort
	} else {
		envPort := getEnvAsInt("DEALER_WS_PORT", 0)
		if envPort > 0 {
			config.Server.DealerWsPort = envPort
		}
	}

	// 保存原始的gRPC端口
	if grpcPort > 0 {
		config.Server.GrpcPort = grpcPort
	} else {
		envPort := getEnvAsInt("GRPC_PORT", 0)
		if envPort > 0 {
			config.Server.GrpcPort = envPort
		}
	}
}

// preserveNacosConnectionInfo 保留 Nacos 連接信息
func preserveNacosConnectionInfo(config *AppConfig) {
	// 保存原始的WebSocket端口和gRPC端口設置
	dealerWsPort := config.Server.DealerWsPort
	grpcPort := config.Server.GrpcPort

	// 設置Nacos連接信息
	config.Nacos = NacosConfig{
		Enabled:     true,
		Host:        getEnv("NACOS_HOST", "127.0.0.1"),
		Port:        uint64(getEnvAsInt("NACOS_PORT", 8848)),
		Namespace:   getEnv("NACOS_NAMESPACE", "public"),
		Group:       getEnv("NACOS_GROUP", "DEFAULT_GROUP"),
		DataId:      getEnv("NACOS_DATAID", "g38_lottery"),
		Username:    getEnv("NACOS_USERNAME", "nacos"),
		Password:    getEnv("NACOS_PASSWORD", "nacos"),
		ServiceIP:   getEnv("SERVICE_IP", "127.0.0.1"),
		ServicePort: getEnvAsInt("SERVICE_PORT", 8080),
	}

	// 保留原始的 DealerWsPort 設定，如果它有效
	if dealerWsPort > 0 {
		config.Server.DealerWsPort = dealerWsPort
	}

	// 保留原始的 GrpcPort 設定，如果它有效
	if grpcPort > 0 {
		config.Server.GrpcPort = grpcPort
	}

	// 確保 WebSocket 端口設置不為0
	if config.Server.DealerWsPort <= 0 {
		config.Server.DealerWsPort = 3002 // 默認值
	}

	// 確保 gRPC 端口設置不為0
	if config.Server.GrpcPort <= 0 {
		config.Server.GrpcPort = 9100 // 默認值
	}

	// 若端口為 3002，嘗試使用備用端口 3033，因為 3002 可能被佔用
	if config.Server.DealerWsPort == 3002 {
		log.Printf("初始化階段檢測到使用可能被佔用的端口 3002，切換至備用端口 3033")
		config.Server.DealerWsPort = 3033
	}

	log.Printf("Nacos 初始配置設置完成，DealerWsPort=%d, GrpcPort=%d", config.Server.DealerWsPort, config.Server.GrpcPort)
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
		"PORT": func(v interface{}) {
			if port, ok := parseIntValue(v); ok {
				config.Server.Port = port
			}
		},
		"DEALER_WS_PORT": func(v interface{}) {
			if port, ok := parseIntValue(v); ok {
				config.Server.DealerWsPort = port
			}
		},
		"API_HOST": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.Server.Host = s
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

		"JWT_SECRET": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.JWT.Secret = s
			}
		},
		"ADMIN_JWT_SECRET": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.JWT.AdminSecret = s
			}
		},
		"JWT_EXPIRES_IN": func(v interface{}) {
			if s, ok := v.(string); ok {
				if duration, err := time.ParseDuration(s); err == nil {
					config.JWT.ExpiresIn = duration
				}
			}
		},
		// RocketMQ 配置映射
		"ROCKETMQ_ENABLED": func(v interface{}) {
			if b, ok := v.(bool); ok {
				// 只保存為記錄，實際不影響配置
				log.Printf("ROCKETMQ_ENABLED 設定: %v", b)
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
		"ROCKETMQ_ACCESS_KEY": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.RocketMQ.AccessKey = s
			}
		},
		"ROCKETMQ_SECRET_KEY": func(v interface{}) {
			if s, ok := v.(string); ok {
				config.RocketMQ.SecretKey = s
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
	switch val := v.(type) {
	case int:
		return val, true
	case float64:
		return int(val), true
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i, true
		}
	}
	return 0, false
}

// ===== 依賴注入相關函數 =====

// ProvideAppConfig 提供應用配置，用於 fx 依賴注入
func ProvideAppConfig(nacosClient nacosManager.NacosClient) (*AppConfig, error) {
	return LoadConfig(nacosClient)
}

// RegisterService 在 Nacos 註冊服務
func RegisterService(config *AppConfig, nacosClient nacosManager.NacosClient) error {
	if !config.Server.RegisterService || !config.Nacos.Enabled || nacosClient == nil {
		log.Println("服務註冊未啟用或 Nacos 客戶端不可用")
		return nil
	}

	// 嘗試將當前配置上傳到 Nacos（如果尚不存在或無效）
	if err := ensureValidConfigInNacos(config, nacosClient); err != nil {
		log.Printf("確保 Nacos 中有有效配置時發生錯誤: %v", err)
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

// ensureValidConfigInNacos 確保 Nacos 中存在有效配置
func ensureValidConfigInNacos(config *AppConfig, nacosClient nacosManager.NacosClient) error {
	content, err := nacosClient.GetConfig(config.Nacos.DataId, config.Nacos.Group)
	if err != nil || content == "" || !isValidJson(content) {
		// 如果有內容但不是有效的 JSON，嘗試修復
		if content != "" && !isValidJson(content) {
			log.Println("檢測到 Nacos 中的配置不是有效的 JSON，嘗試修復...")

			// 預處理 JSON 內容
			fixedContent := preprocessJsonContent(content)

			// 檢查修復後的內容是否有效
			if isValidJson(fixedContent) {
				log.Println("成功修復 Nacos 配置，會使用修復後的配置")

				// 解析修復後的 JSON
				var tmpConfig map[string]interface{}
				if err := json.Unmarshal([]byte(fixedContent), &tmpConfig); err == nil {
					// 上傳修復後的配置
					return uploadFixedConfig(config, nacosClient, tmpConfig)
				}
			}
		}

		log.Println("檢測到 Nacos 中不存在有效的配置，將上傳默認配置...")
		return uploadDefaultConfig(config, nacosClient)
	}

	return nil
}

// uploadFixedConfig 上傳修復後的配置
func uploadFixedConfig(config *AppConfig, nacosClient nacosManager.NacosClient, fixedConfig map[string]interface{}) error {
	// 構建完整配置
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	var baseConfig map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &baseConfig); err != nil {
		return err
	}

	// 合併修復後的配置
	for k, v := range fixedConfig {
		// 添加頂層配置項
		baseConfig[k] = v
	}

	// 序列化為 JSON
	mergedBytes, err := json.MarshalIndent(baseConfig, "", "  ")
	if err != nil {
		return err
	}

	// 上傳到 Nacos
	success, err := uploadConfigToNacos(nacosClient, config.Nacos.DataId, config.Nacos.Group, string(mergedBytes))
	if err != nil {
		return err
	}

	if success {
		log.Println("成功將修復後的配置上傳到 Nacos")
	}

	return nil
}

// uploadDefaultConfig 上傳默認配置
func uploadDefaultConfig(config *AppConfig, nacosClient nacosManager.NacosClient) error {
	configJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失敗: %w", err)
	}

	success, err := uploadConfigToNacos(nacosClient, config.Nacos.DataId, config.Nacos.Group, string(configJson))
	if err != nil {
		return fmt.Errorf("上傳配置到 Nacos 失敗: %w", err)
	}

	if success {
		log.Println("成功將默認配置上傳到 Nacos")
	}

	return nil
}

// registerServiceInstance 註冊服務實例
func registerServiceInstance(config *AppConfig, nacosClient nacosManager.NacosClient) (bool, error) {
	return nacosClient.RegisterInstance(vo.RegisterInstanceParam{
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
