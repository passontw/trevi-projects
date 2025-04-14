package config

import (
	"strconv"
	"time"

	"github.com/nacos-group/nacos-sdk-go/vo"
)

func updateConfigFromNacos(cfg *Config, nacosConfig *NacosAppConfig) {
	if nacosConfig.Port != "" {
		portInt, err := strconv.Atoi(nacosConfig.Port)
		if err == nil {
			cfg.Server.Port = uint64(portInt)
		}
	}
	if nacosConfig.APIHost != "" {
		cfg.Server.APIHost = nacosConfig.APIHost
	}
	if nacosConfig.Version != "" {
		cfg.Server.Version = nacosConfig.Version
	}

	if nacosConfig.DBHost != "" {
		cfg.Database.Host = nacosConfig.DBHost
	}
	if nacosConfig.DBPort != 0 {
		cfg.Database.Port = nacosConfig.DBPort
	}
	if nacosConfig.DBName != "" {
		cfg.Database.Name = nacosConfig.DBName
	}
	if nacosConfig.DBUser != "" {
		cfg.Database.User = nacosConfig.DBUser
	}
	if nacosConfig.DBPassword != "" {
		cfg.Database.Password = nacosConfig.DBPassword
	}

	if nacosConfig.RedisHost != "" || nacosConfig.RedisPort != "" {
		host := nacosConfig.RedisHost
		if host == "" {
			host = "localhost"
		}

		port := nacosConfig.RedisPort
		if port == "" {
			port = "6379"
		}

		cfg.Redis.Addr = host + ":" + port
	}
	if nacosConfig.RedisUsername != "" {
		cfg.Redis.Username = nacosConfig.RedisUsername
	}
	if nacosConfig.RedisPassword != "" {
		cfg.Redis.Password = nacosConfig.RedisPassword
	}
	if nacosConfig.RedisDB != 0 {
		cfg.Redis.DB = nacosConfig.RedisDB
	}

	if nacosConfig.JWTSecret != "" {
		cfg.JWT.Secret = nacosConfig.JWTSecret
	}
	if nacosConfig.JWTExpiresIn != "" {
		duration, err := time.ParseDuration(nacosConfig.JWTExpiresIn)
		if err == nil {
			cfg.JWT.ExpiresIn = duration
		}
	}
}

func createServiceRegistrationParam(cfg *Config) vo.RegisterInstanceParam {
	serviceName := getEnv("NACOS_SERVICE_NAME", "slot-game1")

	return vo.RegisterInstanceParam{
		Ip:          cfg.Server.Host,
		Port:        cfg.Server.Port,
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"version": cfg.Server.Version},
		GroupName:   cfg.Nacos.Group,
	}
}
