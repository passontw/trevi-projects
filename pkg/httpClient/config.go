package httpClient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 定義客戶端配置
type Config struct {
	ServiceEndpoints map[string]ServiceEndpoint `json:"serviceEndpoints"`
	DefaultTimeout   int                        `json:"defaultTimeout"` // 秒
}

// LoadConfig 從文件中加載配置
func LoadConfig(configPath string) (*Config, error) {
	// 確保文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found at: %s", configPath)
	}

	// 讀取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %v", err)
	}

	// 解析配置
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %v", err)
	}

	// 設置默認值
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 30 // 默認30秒
	}

	return &config, nil
}

// SaveConfig 將配置保存到文件
func SaveConfig(config *Config, configPath string) error {
	// 確保目錄存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %v", err)
	}

	// 寫入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %v", err)
	}

	return nil
}

// UpdateEndpoint 更新或添加服務端點
func (c *Config) UpdateEndpoint(endpoint ServiceEndpoint) {
	if c.ServiceEndpoints == nil {
		c.ServiceEndpoints = make(map[string]ServiceEndpoint)
	}
	c.ServiceEndpoints[endpoint.Name] = endpoint
}

// RemoveEndpoint 刪除服務端點
func (c *Config) RemoveEndpoint(serviceName string) bool {
	if c.ServiceEndpoints == nil {
		return false
	}

	if _, exists := c.ServiceEndpoints[serviceName]; !exists {
		return false
	}

	delete(c.ServiceEndpoints, serviceName)
	return true
}

// CreateDefaultConfig 創建默認配置
func CreateDefaultConfig() *Config {
	return &Config{
		ServiceEndpoints: make(map[string]ServiceEndpoint),
		DefaultTimeout:   30,
	}
}
