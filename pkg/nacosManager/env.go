package nacosManager

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	APIHost string
	Port    string
	Version string
}

type RedisENVConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DB       int
}

type EnvConfig struct {
	NACOS_HOST      string
	NACOS_PORT      uint64
	NACOS_NAMESPACE string
	NACOS_GROUP     string
	NACOS_USERNAME  string
	NACOS_PASSWORD  string
	NACOS_DATAID    string
}

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
}

// ConfigWithNacos 包含啟用 Nacos 的配置設定
type ConfigWithNacos interface {
	IsNacosEnabled() bool
	GetNacosGroup() string
	GetNacosDataId() string
}

func LoadEnv() *EnvConfig {
	// 嘗試加載 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	config := &EnvConfig{
		NACOS_HOST:      getEnv("NACOS_HOST", "127.0.0.1"),
		NACOS_PORT:      uint64(getEnvAsInt("NACOS_PORT", 8488)),
		NACOS_NAMESPACE: getEnv("NACOS_NAMESPACE", "public"),
		NACOS_GROUP:     getEnv("NACOS_GROUP", "DEFAULT_GROUP"),
		NACOS_USERNAME:  getEnv("NACOS_USERNAME", "username"),
		NACOS_PASSWORD:  getEnv("NACOS_PASSWORD", "password"),
		NACOS_DATAID:    getEnv("NACOS_DATAID", "dataid"),
	}

	return config
}

// LoadFromEnv 從環境變量加載配置
func LoadFromEnv(cfg interface{}, nacosClient NacosClient) error {
	// 嘗試加載 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// 這裡需要根據 cfg 的具體類型進行處理
	// 由於原始代碼沒有完整實現，這裡提供一個簡單的實現

	// 獲取環境變量的一般邏輯...

	// 如果需要從 Nacos 獲取配置，則進行如下處理
	configurable, ok := cfg.(ConfigWithNacos)
	if ok && configurable.IsNacosEnabled() && nacosClient != nil {
		content, err := nacosClient.GetConfig(
			configurable.GetNacosDataId(),
			configurable.GetNacosGroup(),
		)
		if err != nil {
			return fmt.Errorf("failed to get config from Nacos: %w", err)
		}

		// 解析配置內容並更新 cfg...
		// 這裡需要根據實際配置格式進行處理
		_ = content // 暫時不處理，避免未使用錯誤
	}

	return nil
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnv 獲取環境變數，如果不存在則返回默認值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
