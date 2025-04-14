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

	cfg.Server.Host = getEnv("SERVER_HOST", "localhost")
	portStr := getEnv("SERVER_PORT", "8080")
	portInt, err := strconv.Atoi(portStr)
	if err == nil {
		cfg.Server.Port = uint64(portInt)
	} else {
		cfg.Server.Port = 8080
	}
	cfg.Server.APIHost = getEnv("API_HOST", "localhost:8080")
	cfg.Server.Version = getEnv("VERSION", "1.0.0")

	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnvAsInt("DB_PORT", 5432)
	cfg.Database.User = getEnv("DB_USER", "postgres")
	cfg.Database.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.Database.Name = getEnv("DB_NAME", "postgres")

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	cfg.Redis.Addr = redisHost + ":" + redisPort
	cfg.Redis.Username = getEnv("REDIS_USERNAME", "")
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvAsInt("REDIS_DB", 0)

	cfg.EnableNacos = getEnvAsBool("ENABLE_NACOS", false)
	cfg.Nacos.Host = getEnv("NACOS_HOST", "localhost")
	nacosPort := getEnvAsInt("NACOS_PORT", 8848)
	cfg.Nacos.Port = uint64(nacosPort)
	cfg.Nacos.NamespaceId = getEnv("NACOS_NAMESPACE", "")
	cfg.Nacos.Group = getEnv("NACOS_GROUP", "DEFAULT_GROUP")
	cfg.Nacos.DataId = getEnv("NACOS_DATAID", "slot_game_config") // 使用環境變量獲取DataId
	cfg.Nacos.Username = getEnv("NACOS_USERNAME", "")
	cfg.Nacos.Password = getEnv("NACOS_PASSWORD", "")

	cfg.JWT.Secret = getEnv("JWT_SECRET", "your-secret-key")
	expiresInStr := getEnv("JWT_EXPIRES_IN", "24h")
	duration, err := time.ParseDuration(expiresInStr)
	if err == nil {
		cfg.JWT.ExpiresIn = duration
	} else {
		cfg.JWT.ExpiresIn = 24 * time.Hour
	}

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
