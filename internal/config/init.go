package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func initializeConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or cannot be loaded: %v", err)
	}

	cfg := &Config{}

	// 服務器基本設定（使用默認值，等待 Nacos 覆蓋）
	cfg.Server.Host = getEnv("SERVER_HOST", "localhost")
	cfg.Server.Port = 8080
	cfg.Server.APIHost = "localhost:8080"
	cfg.Server.Version = getEnv("VERSION", "1.0.0")

	// 數據庫設定（使用默認值，等待 Nacos 覆蓋）
	// 默認 TiDB 連接參數
	cfg.Database.Host = "127.0.0.1"
	cfg.Database.Port = 4000 // TiDB 默認端口
	cfg.Database.User = "root"
	cfg.Database.Password = "a12345678"
	cfg.Database.Name = "g38_lottery"

	// Redis 設定（使用默認值，等待 Nacos 覆蓋）
	cfg.Redis.Addr = "172.237.27.51:6379" // 使用實際 Redis 服務器地址
	cfg.Redis.Username = "passontw"
	cfg.Redis.Password = "1qaz@WSX3edc"
	cfg.Redis.DB = 2

	// JWT 設定（使用默認值，等待 Nacos 覆蓋）
	cfg.JWT.Secret = "default-secret-key"
	cfg.JWT.ExpiresIn = 24 * time.Hour

	// Nacos 設定（從環境變量讀取）
	cfg.EnableNacos = getEnvAsBool("ENABLE_NACOS", false)
	cfg.Nacos.Host = getEnv("NACOS_HOST", "localhost")
	nacosPort := getEnvAsInt("NACOS_PORT", 8848)
	cfg.Nacos.Port = uint64(nacosPort)
	cfg.Nacos.NamespaceId = getEnv("NACOS_NAMESPACE", "public")
	cfg.Nacos.Group = getEnv("NACOS_GROUP", "DEFAULT_GROUP")
	cfg.Nacos.DataId = getEnv("NACOS_DATAID", "g38_lottery")
	cfg.Nacos.Username = getEnv("NACOS_USERNAME", "nacos")
	cfg.Nacos.Password = getEnv("NACOS_PASSWORD", "nacos")

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
