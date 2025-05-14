package nacosManager

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// ServiceConfig 通用服務配置
type ServiceConfig struct {
	APIPort      int `json:"API_PORT"`
	DealerWSPort int `json:"DEALER_WS_PORT"`
	GRPCPort     int `json:"GRPC_PORT"`
	RocketMQInfo RocketMQInfo
}

// RocketMQInfo RocketMQ 相關配置
type RocketMQInfo struct {
	Enabled       bool     `json:"ROCKETMQ_ENABLED"`
	NameServers   []string `json:"ROCKETMQ_NAME_SERVERS"`
	ProducerGroup string   `json:"ROCKETMQ_PRODUCER_GROUP"`
	ConsumerGroup string   `json:"ROCKETMQ_CONSUMER_GROUP"`
}

// GenericXMLConfig 通用 XML 配置結構
type GenericXMLConfig struct {
	XMLName xml.Name     `xml:"config"`
	Items   []ConfigItem `xml:",any"`
}

// ConfigItem 配置項結構
type ConfigItem struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// ParseGenericXMLConfig 解析通用 XML 格式的配置
func ParseGenericXMLConfig(xmlContent string) (map[string]interface{}, error) {
	if strings.TrimSpace(xmlContent) == "" {
		return nil, errors.New("XML content is empty")
	}

	var config GenericXMLConfig
	err := xml.Unmarshal([]byte(xmlContent), &config)
	if err != nil {
		log.Printf("解析通用 XML 配置失敗: %v", err)
		return nil, fmt.Errorf("解析通用 XML 配置失敗: %w", err)
	}

	// 轉換為 map
	result := make(map[string]interface{})
	for _, item := range config.Items {
		// 提取標籤名作為鍵
		key := item.XMLName.Local

		// 嘗試解析數值型別
		if val, err := strconv.Atoi(item.Value); err == nil {
			result[key] = val
			continue
		}

		// 嘗試解析布林型別
		if strings.ToLower(item.Value) == "true" {
			result[key] = true
			continue
		} else if strings.ToLower(item.Value) == "false" {
			result[key] = false
			continue
		}

		// 其他情況保持為字符串
		result[key] = item.Value
	}

	return result, nil
}

// ParseJSONConfig 解析 JSON 格式的配置
func ParseJSONConfig(jsonContent string) (map[string]interface{}, error) {
	if strings.TrimSpace(jsonContent) == "" {
		return nil, errors.New("JSON content is empty")
	}

	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonContent), &result)
	if err != nil {
		log.Printf("解析 JSON 配置失敗: %v", err)
		return nil, fmt.Errorf("解析 JSON 配置失敗: %w", err)
	}

	return result, nil
}

// DetectAndParseConfig 檢測配置格式並解析
func DetectAndParseConfig(content string) (map[string]interface{}, error) {
	// 去除空白
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return nil, errors.New("配置內容為空")
	}

	// 檢測是否為 XML
	if strings.HasPrefix(trimmedContent, "<?xml") || strings.HasPrefix(trimmedContent, "<config") {
		return ParseGenericXMLConfig(content)
	}

	// 檢測是否為 JSON
	if strings.HasPrefix(trimmedContent, "{") && strings.HasSuffix(trimmedContent, "}") {
		return ParseJSONConfig(content)
	}

	return nil, errors.New("無法識別的配置格式")
}

// GetServiceConfigFromNacos 從 Nacos 獲取服務配置
func GetServiceConfigFromNacos(client NacosClient, dataId, group string) (map[string]interface{}, error) {
	// 從 Nacos 獲取配置
	content, err := client.GetConfig(dataId, group)
	if err != nil {
		return nil, fmt.Errorf("從 Nacos 獲取服務配置失敗: %w", err)
	}

	// 檢測並解析配置
	config, err := DetectAndParseConfig(content)
	if err != nil {
		return nil, err
	}

	return config, nil
}
