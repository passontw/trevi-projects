package config

import (
	"context"
	"fmt"

	"g38_lottery_service/pkg/logger"
	"g38_lottery_service/pkg/nacosManager"

	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
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
		return cfg, nil
	}

	var nacosAppConfig NacosAppConfig
	if err = yaml.Unmarshal([]byte(content), &nacosAppConfig); err != nil {
		logger.Info(fmt.Sprintf("解析Nacos配置失敗: %v", err))
		return cfg, nil
	}

	logger.Info(fmt.Sprintf("原始數據庫配置: Host=%s, Port=%d, User=%s, Name=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name))

	updateConfigFromNacos(cfg, &nacosAppConfig)

	logger.Info(fmt.Sprintf("更新後數據庫配置: Host=%s, Port=%d, User=%s, Name=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name))
	logger.Info("成功從Nacos加載配置")

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			setupConfigListener(nacosClient, logger, cfg)
			registerServiceToNacos(nacosClient, logger, cfg)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return cfg, nil
}

func setupConfigListener(nacosClient nacosManager.NacosClient, logger logger.Logger, cfg *Config) {
	err := nacosClient.ListenConfig(cfg.Nacos.DataId, cfg.Nacos.Group, func(newContent string) {
		logger.Info("Nacos配置已更改")

		var newNacosConfig NacosAppConfig
		if err := yaml.Unmarshal([]byte(newContent), &newNacosConfig); err != nil {
			logger.Info(fmt.Sprintf("解析新的Nacos配置失敗: %v", err))
			return
		}

		updateConfigFromNacos(cfg, &newNacosConfig)
		logger.Info("配置已動態更新")
	})

	if err != nil {
		logger.Info(fmt.Sprintf("設置Nacos配置監聽失敗: %v", err))
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
