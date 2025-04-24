package config

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"g38_lottery_service/pkg/nacosManager"

	"gopkg.in/yaml.v3"
)

// LoadFromNacos 從 Nacos 載入配置
func LoadFromNacos(cfg *Config) error {
	// 創建 Nacos 配置
	nacosConfig := &nacosManager.NacosConfig{
		IpAddr:      cfg.Nacos.Host,
		Port:        cfg.Nacos.Port,
		NamespaceId: cfg.Nacos.NamespaceId,
		Group:       cfg.Nacos.Group,
		DataId:      cfg.Nacos.DataId,
		LogDir:      "/tmp/nacos/log",
		CacheDir:    "/tmp/nacos/cache",
		Username:    cfg.Nacos.Username,
		Password:    cfg.Nacos.Password,
	}

	// 創建 Nacos 客戶端
	client, err := nacosManager.NewNacosClient(nacosConfig)
	if err != nil {
		return fmt.Errorf("建立 Nacos 客戶端失敗: %w", err)
	}

	// 獲取配置內容
	content, err := client.GetConfig(cfg.Nacos.DataId, cfg.Nacos.Group)
	if err != nil {
		return fmt.Errorf("從 Nacos 獲取配置失敗: %w", err)
	}

	log.Printf("從 Nacos 獲取到的配置內容: %s", content)

	// 預處理 JSON 字符串，去除註解
	content = removeJSONComments(content)
	log.Printf("預處理後的配置內容: %s", content)

	var nacosAppConfig NacosAppConfig

	// 先嘗試 JSON 格式解析
	err = json.Unmarshal([]byte(content), &nacosAppConfig)
	if err != nil {
		log.Printf("JSON 解析失敗: %v，嘗試使用 YAML 解析...", err)

		// 如果 JSON 解析失敗，嘗試 YAML 格式解析
		yaml_err := yaml.Unmarshal([]byte(content), &nacosAppConfig)
		if yaml_err != nil {
			return fmt.Errorf("解析 Nacos 配置失敗 (JSON: %v, YAML: %v)", err, yaml_err)
		}
	}

	// 配置解析成功，打印關鍵配置
	log.Printf("成功解析配置，數據庫配置: Host=%s, Port=%d, Name=%s",
		nacosAppConfig.DBHost, nacosAppConfig.DBPort, nacosAppConfig.DBName)

	// 解析端口配置
	port, err := strconv.ParseUint(nacosAppConfig.Port, 10, 64)
	if err != nil {
		return fmt.Errorf("解析主端口配置失敗: %w", err)
	}
	cfg.Server.Port = port

	// 解析遊戲端 WebSocket 端口配置
	if nacosAppConfig.PlayerWSPort != "" {
		playerWSPort, err := strconv.ParseUint(nacosAppConfig.PlayerWSPort, 10, 64)
		if err != nil {
			return fmt.Errorf("解析遊戲端 WebSocket 端口配置失敗: %w", err)
		}
		cfg.Server.PlayerWSPort = playerWSPort
	} else {
		// 預設值為 3001
		cfg.Server.PlayerWSPort = 3001
	}

	// 解析荷官端 WebSocket 端口配置
	if nacosAppConfig.DealerWSPort != "" {
		dealerWSPort, err := strconv.ParseUint(nacosAppConfig.DealerWSPort, 10, 64)
		if err != nil {
			return fmt.Errorf("解析荷官端 WebSocket 端口配置失敗: %w", err)
		}
		cfg.Server.DealerWSPort = dealerWSPort
	} else {
		// 預設值為 3002
		cfg.Server.DealerWSPort = 3002
	}

	// 更新其他配置...
	updateConfigFromNacos(cfg, &nacosAppConfig)

	return nil
}

// removeJSONComments 移除 JSON 字符串中的 JavaScript 風格註解
func removeJSONComments(jsonStr string) string {
	// 移除單行註解 (// 註解內容)
	singleLineCommentRegex := regexp.MustCompile(`//.*`)
	noSingleLineComments := singleLineCommentRegex.ReplaceAllString(jsonStr, "")

	// 移除多行註解 (/* 註解內容 */)
	multiLineCommentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	noComments := multiLineCommentRegex.ReplaceAllString(noSingleLineComments, "")

	// 去除可能留下的多餘逗號
	trailingCommaRegex := regexp.MustCompile(`,\s*}`)
	noTrailingCommas := trailingCommaRegex.ReplaceAllString(noComments, "}")

	trailingCommaArrayRegex := regexp.MustCompile(`,\s*\]`)
	result := trailingCommaArrayRegex.ReplaceAllString(noTrailingCommas, "]")

	return result
}
