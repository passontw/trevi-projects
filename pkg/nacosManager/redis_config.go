package nacosManager

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"strings"
)

// RedisXMLConfig XML 格式的 Redis 配置結構
type RedisXMLConfig struct {
	XMLName xml.Name        `xml:"config"`
	Redis   RedisNodeConfig `xml:"redis"`
}

// RedisNodeConfig Redis 節點配置
type RedisNodeConfig struct {
	Host      string `xml:"host,attr"`
	Port      string `xml:"port,attr"`
	Username  string `xml:"username,attr"`
	Password  string `xml:"password,attr"`
	DB        string `xml:"db,attr"`
	IsCluster string `xml:"is_cluster,attr"`
	Nodes     struct {
		NodeList []string `xml:"node"`
	} `xml:"nodes"`
}

// RedisClusterConfig 表示解析後的 Redis 群集配置
type RedisClusterConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	DB        int
	IsCluster bool
	Nodes     []string
}

// ParseRedisXMLConfig 解析 XML 格式的 Redis 配置
func ParseRedisXMLConfig(xmlContent string) (*RedisClusterConfig, error) {
	if strings.TrimSpace(xmlContent) == "" {
		return nil, errors.New("XML content is empty")
	}

	var config RedisXMLConfig
	err := xml.Unmarshal([]byte(xmlContent), &config)
	if err != nil {
		log.Printf("解析 Redis XML 配置失敗: %v", err)
		return nil, fmt.Errorf("解析 Redis XML 配置失敗: %w", err)
	}

	// 構建 Redis 配置
	redisConfig := &RedisClusterConfig{
		Host:      config.Redis.Host,
		Port:      config.Redis.Port,
		Username:  config.Redis.Username,
		Password:  config.Redis.Password,
		DB:        0,     // 默認值，稍後會嘗試轉換
		IsCluster: false, // 默認值，稍後會嘗試轉換
		Nodes:     make([]string, 0),
	}

	// 嘗試轉換 DB 值
	if config.Redis.DB != "" {
		var dbValue int
		_, err := fmt.Sscanf(config.Redis.DB, "%d", &dbValue)
		if err == nil {
			redisConfig.DB = dbValue
		}
	}

	// 嘗試轉換 IsCluster 值
	if strings.ToLower(config.Redis.IsCluster) == "true" {
		redisConfig.IsCluster = true
		// 只有在群集模式下才處理節點列表
		redisConfig.Nodes = config.Redis.Nodes.NodeList
	}

	return redisConfig, nil
}

// GetRedisConfigFromNacos 從 Nacos 獲取 Redis 配置
func GetRedisConfigFromNacos(client NacosClient, dataId, group string) (*RedisClusterConfig, error) {
	// 從 Nacos 獲取 XML 配置
	xmlContent, err := client.GetConfig(dataId, group)
	if err != nil {
		return nil, fmt.Errorf("從 Nacos 獲取 Redis 配置失敗: %w", err)
	}

	// 解析 XML 配置
	redisConfig, err := ParseRedisXMLConfig(xmlContent)
	if err != nil {
		return nil, err
	}

	return redisConfig, nil
}

// RedisConfigToMap 將 Redis 配置轉換為 map
func RedisConfigToMap(config *RedisClusterConfig) map[string]interface{} {
	result := make(map[string]interface{})

	// 只有在非群集模式下才設置主機和端口
	if !config.IsCluster {
		if config.Host != "" {
			result["REDIS_HOST"] = config.Host
		}
		if config.Port != "" {
			result["REDIS_PORT"] = config.Port
		}
	} else {
		// 使用第一個節點作為主機和端口（向下兼容）
		if len(config.Nodes) > 0 {
			parts := strings.Split(config.Nodes[0], ":")
			if len(parts) == 2 {
				result["REDIS_HOST"] = parts[0]
				result["REDIS_PORT"] = parts[1]
			}
		}
	}

	// 設置其他配置項
	if config.Username != "" {
		result["REDIS_USERNAME"] = config.Username
	} else {
		result["REDIS_USERNAME"] = ""
	}

	result["REDIS_PASSWORD"] = config.Password
	result["REDIS_DB"] = config.DB
	result["REDIS_CLUSTER_ENABLED"] = config.IsCluster

	if config.IsCluster {
		result["REDIS_CLUSTER_NODES"] = config.Nodes
	}

	return result
}
