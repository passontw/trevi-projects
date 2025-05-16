package config

import (
	"encoding/xml"
	"strconv"
)

// LotteryServiceNacosConfig 是從 Nacos 服務器獲取的彩票服務配置
type LotteryServiceNacosConfig struct {
	XMLName               xml.Name `xml:"config"`
	APIPORT               string   `xml:"API_PORT"`
	APIHealthPort         string   `xml:"API_HEALTH_PORT"`
	GRPCPort              string   `xml:"GRPC_PORT"`
	RocketMQEnabled       string   `xml:"ROCKETMQ_ENABLED"`
	RocketMQProducerGroup string   `xml:"ROCKETMQ_PRODUCER_GROUP"`
	RocketMQConsumerGroup string   `xml:"ROCKETMQ_CONSUMER_GROUP"`
}

// ParseLotteryServiceConfig 解析 XML 配置
func ParseLotteryServiceConfig(data []byte) (*LotteryServiceNacosConfig, error) {
	var config LotteryServiceNacosConfig
	if err := xml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ApplyToAppConfig 將彩票服務配置應用到應用程序配置
func (n *LotteryServiceNacosConfig) ApplyToAppConfig(config *AppConfig) error {
	// 設置 API 端口
	if n.APIPORT != "" {
		if port, err := strconv.Atoi(n.APIPORT); err == nil {
			config.Server.Port = port
		}
	}

	// 設置 GRPC 端口
	if n.GRPCPort != "" {
		if port, err := strconv.Atoi(n.GRPCPort); err == nil {
			config.Server.GrpcPort = port
		}
	}

	// 設置 RocketMQ 配置
	if n.RocketMQEnabled != "" {
		enabled, _ := strconv.ParseBool(n.RocketMQEnabled)
		config.RocketMQ.Enabled = enabled
	}

	if n.RocketMQProducerGroup != "" {
		config.RocketMQ.ProducerGroup = n.RocketMQProducerGroup
	}

	if n.RocketMQConsumerGroup != "" {
		config.RocketMQ.ConsumerGroup = n.RocketMQConsumerGroup
	}

	return nil
}

// GetAPIPort 獲取 API 端口
func (n *LotteryServiceNacosConfig) GetAPIPort(defaultPort int) int {
	if n.APIPORT != "" {
		if port, err := strconv.Atoi(n.APIPORT); err == nil {
			return port
		}
	}
	return defaultPort
}

// GetAPIHealthPort 獲取 API 健康檢查端口
func (n *LotteryServiceNacosConfig) GetAPIHealthPort(defaultPort int) int {
	if n.APIHealthPort != "" {
		if port, err := strconv.Atoi(n.APIHealthPort); err == nil {
			return port
		}
	}
	return defaultPort
}

// GetGRPCPort 獲取 GRPC 端口
func (n *LotteryServiceNacosConfig) GetGRPCPort(defaultPort int) int {
	if n.GRPCPort != "" {
		if port, err := strconv.Atoi(n.GRPCPort); err == nil {
			return port
		}
	}
	return defaultPort
}

// IsRocketMQEnabled 檢查 RocketMQ 是否啟用
func (n *LotteryServiceNacosConfig) IsRocketMQEnabled(defaultValue bool) bool {
	if n.RocketMQEnabled != "" {
		enabled, err := strconv.ParseBool(n.RocketMQEnabled)
		if err == nil {
			return enabled
		}
	}
	return defaultValue
}
