package config

import (
	"context"
	"fmt"
	"time"

	"git.trevi.cc/server/go_gamecommon/cache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 處理 Redis 配置相關的操作

// ProvideRedisConfig 從 XML 配置轉換為 go_gamecommon/cache 的 RedisConfig
func ProvideRedisConfig(appConfig *AppConfig, logger *zap.Logger) (*cache.RedisConfig, error) {
	// 將 AppConfig 中的 Redis 配置轉換為 go_gamecommon/cache 的 RedisConfig

	// 構建節點列表
	var nodes []string

	// 檢查是否有配置集群模式
	isCluster := false
	if val, ok := appConfig.Redis.Extra["is_cluster"]; ok {
		if strVal, ok := val.(string); ok && strVal == "true" {
			isCluster = true
		}
	}

	// 如果是集群模式，獲取節點列表
	if isCluster && appConfig.Redis.Extra != nil {
		if nodesVal, ok := appConfig.Redis.Extra["nodes"]; ok {
			if nodesList, ok := nodesVal.([]string); ok {
				nodes = nodesList
			} else if nodesSlice, ok := nodesVal.([]interface{}); ok {
				for _, node := range nodesSlice {
					if nodeStr, ok := node.(string); ok {
						nodes = append(nodes, nodeStr)
					}
				}
			}
		}
	}

	// 如果節點列表為空但是集群模式，則添加主節點
	if isCluster && len(nodes) == 0 {
		host := appConfig.Redis.Host
		if host == "" {
			host = "localhost"
		}
		port := appConfig.Redis.Port
		if port <= 0 {
			port = 6379
		}
		nodes = append(nodes, fmt.Sprintf("%s:%d", host, port))
	}

	redisConfig := &cache.RedisConfig{
		Host:      appConfig.Redis.Host,
		Port:      appConfig.Redis.Port,
		Password:  appConfig.Redis.Password,
		DB:        appConfig.Redis.DB,
		IsCluster: isCluster,
		Nodes:     cache.NodeList{Nodes: nodes},
	}

	logger.Info("Redis 配置轉換完成",
		zap.String("host", redisConfig.Host),
		zap.Int("port", redisConfig.Port),
		zap.Int("db", redisConfig.DB),
		zap.Bool("isCluster", redisConfig.IsCluster),
		zap.Strings("nodes", redisConfig.Nodes.Nodes),
	)

	return redisConfig, nil
}

// InitRedis 使用 go_gamecommon/cache 初始化 Redis 連線
func InitRedis(lc fx.Lifecycle, redisConfig *cache.RedisConfig, logger *zap.Logger) error {
	// 初始化 Redis 連線
	logger.Info("初始化 Redis 連線")

	err := cache.RedisInit(*redisConfig, 3*time.Second)
	if err != nil {
		logger.Error("Redis 連線初始化失敗", zap.Error(err))
		return err
	}

	logger.Info("Redis 連線初始化成功")

	// 在應用關閉時關閉 Redis 連線
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("關閉 Redis 連線")
			return cache.RedisClose()
		},
	})

	return nil
}

// Module Redis 模塊
var RedisModule = fx.Options(
	fx.Provide(ProvideRedisConfig),
	fx.Invoke(InitRedis),
)
