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
	PlayerWsPort    int    `json:"playerWsPort"` // 玩家 WebSocket 端口
	DealerWsPort    int    `json:"dealerWsPort"` // 荷官 WebSocket 端口
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
			log.Println("將使用本地/環境變量配置")
		} else {
			// 預處理 JSON 內容
			content = preprocessJsonContent(content)

			// 嘗試解析配置
			if err := parseNacosConfig(content, config); err != nil {
				log.Printf("解析 Nacos 配置失敗: %v", err)
				log.Println("將使用本地/環境變量配置")
			} else {
				log.Println("成功從 Nacos 獲取配置並合併")

				// 保留 Nacos 連接信息
				preserveNacosConnectionInfo(config)
			}
		}
	}

	return config, nil
}

// createDefaultConfig 創建默認配置
func createDefaultConfig() *AppConfig {
	// 定義 Nacos 相關設置
	nacosEnabled := getEnvAsBool("ENABLE_NACOS", false)

	return &AppConfig{
		AppName: getEnv("APP_NAME", "app_service"),
		Debug:   getEnvAsBool("DEBUG", false),
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvAsInt("SERVER_PORT", 8080),
			Version:         getEnv("SERVER_VERSION", "1.0.0"),
			ServiceName:     getEnv("SERVICE_NAME", "app_service"),
			ServiceID:       getEnv("SERVICE_ID", "app_service"),
			ServiceIP:       getEnv("SERVICE_IP", "127.0.0.1"),
			ServicePort:     getEnvAsInt("SERVICE_PORT", 8080),
			PlayerWsPort:    getEnvAsInt("PLAYER_WS_PORT", 3001),
			DealerWsPort:    getEnvAsInt("DEALER_WS_PORT", 3002),
			RegisterService: getEnvAsBool("REGISTER_SERVICE", true),
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
	}
}

// parseNacosConfig 解析 Nacos 配置並更新 AppConfig
func parseNacosConfig(content string, config *AppConfig) error {
	// 保存原始的WebSocket端口設置
	playerWsPort := config.Server.PlayerWsPort
	dealerWsPort := config.Server.DealerWsPort

	// 首先嘗試直接解析為 AppConfig 結構
	var nacosConfig AppConfig
	if err := json.Unmarshal([]byte(content), &nacosConfig); err == nil {
		// 合併 Nacos 配置與本地配置
		*config = nacosConfig

		// 恢復端口設置
		restoreWebSocketPorts(config, playerWsPort, dealerWsPort)
		return nil
	}

	// 如果直接解析失敗，嘗試解析為扁平結構的 map
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonMap); err != nil {
		return err
	}

	// 將扁平結構轉換為 AppConfig
	if err := mapJsonToAppConfig(jsonMap, config); err != nil {
		return err
	}

	// 恢復端口設置
	restoreWebSocketPorts(config, playerWsPort, dealerWsPort)
	return nil
}

// restoreWebSocketPorts 恢復WebSocket端口設置
func restoreWebSocketPorts(config *AppConfig, playerWsPort, dealerWsPort int) {
	// 優先使用環境變量中的值
	if envPlayerPort := getEnvAsInt("PLAYER_WS_PORT", 0); envPlayerPort > 0 {
		config.Server.PlayerWsPort = envPlayerPort
	} else if playerWsPort > 0 {
		config.Server.PlayerWsPort = playerWsPort
	}

	if envDealerPort := getEnvAsInt("DEALER_WS_PORT", 0); envDealerPort > 0 {
		config.Server.DealerWsPort = envDealerPort
	} else if dealerWsPort > 0 {
		config.Server.DealerWsPort = dealerWsPort
	}

	// 確保端口設置不為0
	if config.Server.PlayerWsPort <= 0 {
		config.Server.PlayerWsPort = 3001 // 默認值
	}

	if config.Server.DealerWsPort <= 0 {
		config.Server.DealerWsPort = 3002 // 默認值
	}

	// 記錄當前使用的端口
	log.Printf("使用的玩家端WebSocket端口: %d", config.Server.PlayerWsPort)
	log.Printf("使用的荷官端WebSocket端口: %d", config.Server.DealerWsPort)
}

// preserveNacosConnectionInfo 保留 Nacos 連接信息
func preserveNacosConnectionInfo(config *AppConfig) {
	// 保存原始的WebSocket端口設置
	playerWsPort := config.Server.PlayerWsPort
	dealerWsPort := config.Server.DealerWsPort

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

	// 恢復WebSocket端口設置，優先使用環境變量中的值
	if envPlayerPort := getEnvAsInt("PLAYER_WS_PORT", 0); envPlayerPort > 0 {
		config.Server.PlayerWsPort = envPlayerPort
	} else if playerWsPort > 0 {
		config.Server.PlayerWsPort = playerWsPort
	}

	if envDealerPort := getEnvAsInt("DEALER_WS_PORT", 0); envDealerPort > 0 {
		config.Server.DealerWsPort = envDealerPort
	} else if dealerWsPort > 0 {
		config.Server.DealerWsPort = dealerWsPort
	}

	// 確保端口設置不為0
	if config.Server.PlayerWsPort <= 0 {
		config.Server.PlayerWsPort = 3001 // 默認值
	}

	if config.Server.DealerWsPort <= 0 {
		config.Server.DealerWsPort = 3002 // 默認值
	}
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
		"PLAYER_WS_PORT": func(v interface{}) {
			if port, ok := parseIntValue(v); ok {
				config.Server.PlayerWsPort = port
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
