package config

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"g38_lottery_service/pkg/logger"
	"g38_lottery_service/pkg/nacosManager"

	"go.uber.org/fx"
)

var Module = fx.Module("config",
	fx.Provide(
		ProvideConfig,
	),
)

func ProvideConfig(lc fx.Lifecycle, nacosClient nacosManager.NacosClient, logger logger.Logger) (*Config, error) {
	cfg := initializeConfig()

	logger.Info(fmt.Sprintf("Nacos配置: Host=%s, Port=%d, Namespace=%s, Group=%s, DataId=%s, EnableNacos=%v",
		cfg.Nacos.Host, cfg.Nacos.Port, cfg.Nacos.NamespaceId, cfg.Nacos.Group, cfg.Nacos.DataId, cfg.EnableNacos))

	if !cfg.EnableNacos {
		logger.Info("Nacos配置未啟用，使用本地配置")
		return cfg, nil
	}

	return configureWithNacos(lc, nacosClient, logger, cfg)
}

func configureWithNacos(lc fx.Lifecycle, nacosClient nacosManager.NacosClient, logger logger.Logger, cfg *Config) (*Config, error) {
	logger.Info("嘗試從Nacos獲取配置...")

	content, err := nacosClient.GetConfig(cfg.Nacos.DataId, cfg.Nacos.Group)
	if err != nil {
		logger.Info(fmt.Sprintf("從Nacos獲取配置失敗: %v", err))
		logger.Info("使用本地默認配置繼續運行")
		return cfg, nil
	}

	logger.Info(fmt.Sprintf("成功從Nacos獲取配置: %s", content))

	// 嘗試直接提取關鍵配置值，繞過 JSON 解析問題
	nacosAppConfig := extractConfig(content, logger)

	logger.Info(fmt.Sprintf("原始數據庫配置: Host=%s, Port=%d, User=%s, Name=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name))

	// 使用提取的配置更新應用設置
	updateConfigFromExtracted(cfg, nacosAppConfig, logger)

	logger.Info(fmt.Sprintf("更新後數據庫配置: Host=%s, Port=%d, User=%s, Name=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name))
	logger.Info("成功從Nacos加載配置並應用")

	// 啟用配置變更監聽
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 監聽配置變更
			setupConfigListener(nacosClient, logger, cfg)
			// 註冊服務到Nacos
			registerServiceToNacos(nacosClient, logger, cfg)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return cfg, nil
}

// extractConfig 從 JSON 字符串中提取配置值
func extractConfig(jsonStr string, logger logger.Logger) *NacosAppConfig {
	config := &NacosAppConfig{}

	// 移除註解
	cleanStr := removeJSONComments(jsonStr)
	logger.Info(fmt.Sprintf("清理註解後: %s", cleanStr))

	// 嘗試正常解析
	err := json.Unmarshal([]byte(cleanStr), config)
	if err == nil {
		return config
	}

	logger.Info(fmt.Sprintf("標準解析失敗: %v，嘗試手動解析配置項", err))

	// 手動解析主要配置項
	config.Port = extractStringValue(cleanStr, `"PORT":\s*"([^"]+)"`)
	config.PlayerWSPort = extractStringValue(cleanStr, `"PLAYER_WS_PORT":\s*"([^"]+)"`)
	config.DealerWSPort = extractStringValue(cleanStr, `"DEALER_WS_PORT":\s*"([^"]+)"`)
	config.DBHost = extractStringValue(cleanStr, `"DB_HOST":\s*"([^"]+)"`)
	dbPortStr := extractStringValue(cleanStr, `"DB_PORT":\s*(\d+)`)
	if dbPortStr != "" {
		if dbPort, err := strconv.Atoi(dbPortStr); err == nil {
			config.DBPort = dbPort
		}
	}
	config.DBName = extractStringValue(cleanStr, `"DB_NAME":\s*"([^"]+)"`)
	config.DBUser = extractStringValue(cleanStr, `"DB_USER":\s*"([^"]+)"`)
	config.DBPassword = extractStringValue(cleanStr, `"DB_PASSWORD":\s*"([^"]+)"`)
	config.RedisHost = extractStringValue(cleanStr, `"REDIS_HOST":\s*"([^"]+)"`)
	config.RedisPort = extractStringValue(cleanStr, `"REDIS_PORT":\s*"([^"]+)"`)
	config.RedisUsername = extractStringValue(cleanStr, `"REDIS_USERNAME":\s*"([^"]+)"`)
	config.RedisPassword = extractStringValue(cleanStr, `"REDIS_PASSWORD":\s*"([^"]+)"`)
	redisDBStr := extractStringValue(cleanStr, `"REDIS_DB":\s*(\d+)`)
	if redisDBStr != "" {
		if redisDB, err := strconv.Atoi(redisDBStr); err == nil {
			config.RedisDB = redisDB
		}
	}

	// 列出提取的值，便於調試
	logger.Info(fmt.Sprintf("手動提取的配置: PORT=%s, DB_HOST=%s, DB_PORT=%d, DB_NAME=%s, DB_USER=%s",
		config.Port, config.DBHost, config.DBPort, config.DBName, config.DBUser))

	return config
}

// extractStringValue 從文本中提取符合正則表達式的值
func extractStringValue(text, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// updateConfigFromExtracted 使用提取的配置更新應用設置
func updateConfigFromExtracted(cfg *Config, nacosConfig *NacosAppConfig, logger logger.Logger) {
	// 更新服務器配置
	if nacosConfig.Port != "" {
		portInt, err := strconv.Atoi(nacosConfig.Port)
		if err == nil {
			logger.Info(fmt.Sprintf("更新服務器端口: %d -> %d", cfg.Server.Port, portInt))
			cfg.Server.Port = uint64(portInt)
		}
	}

	// 更新數據庫配置
	if nacosConfig.DBHost != "" {
		logger.Info(fmt.Sprintf("更新數據庫主機: %s -> %s", cfg.Database.Host, nacosConfig.DBHost))
		cfg.Database.Host = nacosConfig.DBHost
		// 如果主機名是 localhost，改為 127.0.0.1 以避免 IPv6 問題
		if cfg.Database.Host == "localhost" {
			cfg.Database.Host = "127.0.0.1"
		}
	}

	if nacosConfig.DBPort != 0 {
		logger.Info(fmt.Sprintf("更新數據庫端口: %d -> %d", cfg.Database.Port, nacosConfig.DBPort))
		cfg.Database.Port = nacosConfig.DBPort
	}

	if nacosConfig.DBName != "" {
		logger.Info(fmt.Sprintf("更新數據庫名稱: %s -> %s", cfg.Database.Name, nacosConfig.DBName))
		cfg.Database.Name = nacosConfig.DBName
	}

	if nacosConfig.DBUser != "" {
		logger.Info(fmt.Sprintf("更新數據庫用戶: %s -> %s", cfg.Database.User, nacosConfig.DBUser))
		cfg.Database.User = nacosConfig.DBUser
	}

	if nacosConfig.DBPassword != "" {
		logger.Info("更新數據庫密碼")
		cfg.Database.Password = nacosConfig.DBPassword
	}

	// 更新 Redis 配置
	if nacosConfig.RedisHost != "" || nacosConfig.RedisPort != "" {
		host := nacosConfig.RedisHost
		if host == "" {
			host = "localhost"
		}

		port := nacosConfig.RedisPort
		if port == "" {
			port = "6379"
		}

		addr := host + ":" + port
		logger.Info(fmt.Sprintf("更新Redis地址: %s -> %s", cfg.Redis.Addr, addr))
		cfg.Redis.Addr = addr
	}

	if nacosConfig.RedisUsername != "" {
		logger.Info(fmt.Sprintf("更新Redis用戶名: %s -> %s", cfg.Redis.Username, nacosConfig.RedisUsername))
		cfg.Redis.Username = nacosConfig.RedisUsername
	}

	if nacosConfig.RedisPassword != "" {
		logger.Info("更新Redis密碼")
		cfg.Redis.Password = nacosConfig.RedisPassword
	}

	if nacosConfig.RedisDB != 0 {
		logger.Info(fmt.Sprintf("更新Redis數據庫: %d -> %d", cfg.Redis.DB, nacosConfig.RedisDB))
		cfg.Redis.DB = nacosConfig.RedisDB
	}
}

// removeJSONComments 使用正則表達式移除JSON字符串中的JavaScript樣式註解
func removeJSONComments(jsonStr string) string {
	// 移除單行註解 (// 後面的內容直到行尾)
	singleLineCommentRegex := regexp.MustCompile(`//.*`)
	jsonClean := singleLineCommentRegex.ReplaceAllString(jsonStr, "")

	// 分行，移除空行和隻包含空白字符的行
	lines := strings.Split(jsonClean, "\n")
	cleanLines := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	// 重新組合成字符串
	jsonClean = strings.Join(cleanLines, "\n")

	// 修復可能的JSON語法問題 (例如尾隨逗號)
	jsonClean = strings.Replace(jsonClean, ",\n}", "\n}", -1)
	jsonClean = strings.Replace(jsonClean, ",\n]", "\n]", -1)

	return jsonClean
}

func setupConfigListener(nacosClient nacosManager.NacosClient, logger logger.Logger, cfg *Config) {
	err := nacosClient.ListenConfig(cfg.Nacos.DataId, cfg.Nacos.Group, func(newContent string) {
		logger.Info("Nacos配置已更改，開始更新...")
		logger.Info(fmt.Sprintf("接收到的新配置: %s", newContent))

		// 使用相同的提取邏輯處理新配置
		newNacosConfig := extractConfig(newContent, logger)

		// 記錄更新前的設定
		logger.Info(fmt.Sprintf("更新前數據庫配置: Host=%s, Port=%d, User=%s, Name=%s",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name))

		// 更新配置
		updateConfigFromExtracted(cfg, newNacosConfig, logger)

		// 記錄更新後的設定
		logger.Info(fmt.Sprintf("更新後數據庫配置: Host=%s, Port=%d, User=%s, Name=%s",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name))
		logger.Info("配置已動態更新完成")
	})

	if err != nil {
		logger.Info(fmt.Sprintf("設置Nacos配置監聽失敗: %v", err))
	} else {
		logger.Info("成功設置Nacos配置監聽，將自動接收配置變更")
	}
}

func registerServiceToNacos(nacosClient nacosManager.NacosClient, logger logger.Logger, cfg *Config) {
	param := createServiceRegistrationParam(cfg)

	success, err := nacosClient.RegisterInstance(param)
	if err != nil {
		logger.Info(fmt.Sprintf("註冊服務到Nacos失敗: %v", err))
	} else if success {
		logger.Info("服務已成功註冊到Nacos")
	}
}
