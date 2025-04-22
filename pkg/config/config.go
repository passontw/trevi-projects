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
	Port     int    `json:"port"`
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
			Port:     getEnvAsInt("DB_PORT", 3306),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "lottery_service"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
	}
}

// DSN 構建MySQL連接字符串
func (db *DatabaseConfig) DSN() string {
	// MySQL DSN 格式: username:password@tcp(host:port)/database
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.User,
		db.Password,
		db.Host,
		db.Port,
		db.Name,
	)
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
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// 獲取布爾類型的環境變量
func getEnvAsBool(key string, defaultValue bool) bool {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.ParseBool(valueStr); err == nil {
			return value
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

// GetLocalIP 獲取本地IP地址
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// 檢查IP地址類型
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("無法獲取本地IP地址")
}
