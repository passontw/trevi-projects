package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

// ServiceConfig 通用服務配置
type ServiceConfig struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
}

// ServerConfig 服務器配置
type ServerConfig struct {
	Port        string `json:"port"`
	Environment string `json:"environment"`
}

// DatabaseConfig 資料庫配置
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// LoadConfig 從配置源加載配置
func LoadConfig(nacosClient *NacosClient) (*ServiceConfig, error) {
	if nacosClient == nil || !nacosClient.Config.EnableNacos {
		// 如果未啟用Nacos，從環境變量加載
		return loadLocalConfig(), nil
	}

	// 從Nacos加載配置
	config, err := nacosClient.GetServiceConfig()
	if err != nil {
		log.Printf("Failed to get config from Nacos: %v, using local config", err)
		return loadLocalConfig(), nil
	}

	return config, nil
}

// 從環境變量加載本地配置
func loadLocalConfig() *ServiceConfig {
	return &ServiceConfig{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "shoppingcart"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
	}
}

// DSN 構建PostgreSQL連接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.Name)
}

// RedisAddr 構建Redis地址
func (c *RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// 從環境變量獲取值，如不存在則使用默認值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// 獲取整數類型的環境變量
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetOutboundIP 獲取主機對外IP地址
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
