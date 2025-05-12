package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"g38_lottery_service/internal/lottery_service/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// 導出 redis.Nil 以便使用者可以處理找不到鍵的情況
var Nil = redis.Nil

// 輔助函數，檢查錯誤是否表示鍵不存在
func IsKeyNotExist(err error) bool {
	return err == redis.Nil
}

// RedisConfig 存儲 Redis 連接的配置項
type RedisConfig struct {
	Addr      string
	Username  string
	Password  string
	DB        int
	IsCluster bool
	Addrs     []string // 集群模式下的所有節點地址
}

// RedisManager 提供 Redis 操作的介面
type RedisManager interface {
	// 基本操作
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// 過期時間操作
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)

	// 哈希操作
	HSet(ctx context.Context, key string, field string, value interface{}) error
	HGet(ctx context.Context, key string, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error

	// 列表操作
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPush(ctx context.Context, key string, values ...interface{}) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)

	// 集合操作
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SRem(ctx context.Context, key string, members ...interface{}) error

	// 有序集合操作
	ZAdd(ctx context.Context, key string, score float64, member string) error
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)

	// 事務操作
	Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error

	// 連接管理
	Close() error
	Ping(ctx context.Context) error
}

// 抽象的 Redis 客戶端介面，同時支援單節點和集群
type redisClientInterface interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

// redisManagerImpl 是 RedisManager 介面的實作
type redisManagerImpl struct {
	client redisClientInterface
}

// NewRedisManager 創建一個新的 Redis 管理器
func NewRedisManager(config *RedisConfig) RedisManager {
	var client redisClientInterface

	if config.IsCluster {
		// 使用集群客戶端
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    config.Addrs,
			Username: config.Username,
			Password: config.Password,
		})
		fmt.Printf("建立 Redis 集群連接，節點數: %d\n", len(config.Addrs))
	} else {
		// 使用單節點客戶端
		client = redis.NewClient(&redis.Options{
			Addr:     config.Addr,
			Username: config.Username,
			Password: config.Password,
			DB:       config.DB,
		})
		fmt.Printf("建立 Redis 單節點連接: %s\n", config.Addr)
	}

	return &redisManagerImpl{
		client: client,
	}
}

// ProvideRedisConfig 從應用配置提供 Redis 配置
func ProvideRedisConfig(cfg *config.AppConfig) *RedisConfig {
	// 檢查是否為集群模式
	isCluster := false
	var addrs []string

	// 日誌顯示 Extra 字段中的所有鍵
	fmt.Println("檢查 Redis 配置的 Extra 字段:")
	if cfg.Redis.Extra != nil {
		for k, v := range cfg.Redis.Extra {
			fmt.Printf("  找到 Extra 鍵: %s = %v (類型: %T)\n", k, v, v)
		}
	} else {
		fmt.Println("  Redis.Extra 為 nil")
	}

	// 從環境變數讀取集群配置
	if clustered, ok := cfg.Redis.Extra["is_cluster"]; ok {
		fmt.Printf("檢測到 is_cluster 設定: %v (類型: %T)\n", clustered, clustered)

		// 轉換為字符串並檢查是否為 "true"
		if clusterStr, ok := clustered.(string); ok && strings.ToLower(clusterStr) == "true" {
			isCluster = true
			fmt.Printf("已開啟 Redis 集群模式 (is_cluster = %s)\n", clusterStr)

			// 檢查節點配置
			if nodes, ok := cfg.Redis.Extra["nodes"]; ok {
				fmt.Printf("找到集群節點配置: %v (類型: %T)\n", nodes, nodes)

				// 嘗試將節點轉換為字符串列表
				if nodeList, ok := nodes.([]string); ok {
					addrs = nodeList
					fmt.Printf("已解析出 %d 個集群節點\n", len(addrs))
				} else if nodeSlice, ok := nodes.([]interface{}); ok {
					// 處理 interface{} 切片的情況
					for _, node := range nodeSlice {
						if nodeStr, ok := node.(string); ok {
							addrs = append(addrs, nodeStr)
						}
					}
					fmt.Printf("從 interface{} 切片解析出 %d 個集群節點\n", len(addrs))
				} else if nodeStr, ok := nodes.(string); ok && nodeStr != "" {
					// 如果是單一字符串，嘗試分割
					addrs = strings.Split(nodeStr, ",")
					fmt.Printf("從字符串解析出 %d 個集群節點\n", len(addrs))
				} else {
					fmt.Printf("無法解析節點配置，類型: %T\n", nodes)
				}
			} else {
				fmt.Println("沒有找到集群節點配置")
			}
		} else {
			fmt.Printf("集群模式未啟用 (is_cluster = %v)\n", clustered)
		}
	} else {
		fmt.Println("沒有找到 is_cluster 設定，使用單節點模式")
	}

	// 如果集群模式但沒有節點，使用主節點作為首個節點
	if isCluster && len(addrs) == 0 {
		host := cfg.Redis.Host
		if host == "" {
			host = "localhost"
		}

		port := cfg.Redis.Port
		if port <= 0 {
			port = 6379
		}

		// 添加配置的主節點
		addrs = append(addrs, fmt.Sprintf("%s:%d", host, port))
		fmt.Printf("Redis 集群模式啟用，但沒有額外節點配置，使用主節點: %s:%d\n", host, port)
	}

	// 如果是集群模式，嘗試自動增加可能的節點
	if isCluster && len(addrs) > 0 {
		// 從現有節點提取主機名
		firstNode := addrs[0]
		parts := strings.Split(firstNode, ":")
		if len(parts) == 2 {
			host := parts[0]

			// 檢查是否已經有多個端口
			hasPort := make(map[string]bool)
			for _, addr := range addrs {
				hasPort[addr] = true
			}

			// 在 Redis Cluster 中，常見的端口範圍是 7000-7005 或 6379-6384
			// 嘗試添加可能存在的節點
			for _, port := range []int{7000, 7001, 7002, 7003, 7004, 7005} {
				newAddr := fmt.Sprintf("%s:%d", host, port)
				if !hasPort[newAddr] {
					addrs = append(addrs, newAddr)
					fmt.Printf("自動添加可能的集群節點: %s\n", newAddr)
				}
			}
		}
	}

	// 處理 Username 欄位 - 強制為空
	username := ""

	// 記錄配置信息
	fmt.Printf("Redis 配置: 集群模式=%v, 節點數=%d\n", isCluster, len(addrs))
	if isCluster {
		for i, addr := range addrs {
			fmt.Printf("  節點 %d: %s\n", i+1, addr)
		}
	} else {
		// 檢查主機名
		host := cfg.Redis.Host
		if host == "" {
			host = "localhost"
			fmt.Println("Redis Host 為空，使用默認值 localhost")
		}

		// 確保端口有效
		port := cfg.Redis.Port
		if port <= 0 {
			port = 6379
			fmt.Println("Redis Port 無效，使用默認值 6379")
		}

		addr := fmt.Sprintf("%s:%d", host, port)
		fmt.Printf("最終 Redis 地址: %s\n", addr)

		return &RedisConfig{
			Addr:      addr,
			Username:  username,
			Password:  cfg.Redis.Password,
			DB:        cfg.Redis.DB,
			IsCluster: false,
		}
	}

	return &RedisConfig{
		Addr:      "", // 集群模式不使用此字段
		Username:  username,
		Password:  cfg.Redis.Password,
		DB:        cfg.Redis.DB,
		IsCluster: isCluster,
		Addrs:     addrs,
	}
}

// ProvideRedisClient 提供 Redis 客戶端實例，用於 fx
func ProvideRedisClient(lc fx.Lifecycle, config *RedisConfig) (redisClientInterface, error) {
	var client redisClientInterface

	// 不論配置如何，始終使用集群客戶端
	// 這是因為 MOVED 錯誤表明需要支持集群模式的重定向
	fmt.Printf("始終使用 Redis 集群客戶端模式，以支持自動重定向\n")

	// 如果沒有節點配置，使用主節點的地址
	addrs := config.Addrs
	if len(addrs) == 0 {
		addrs = []string{config.Addr}
		fmt.Printf("使用主節點作為唯一集群節點: %s\n", config.Addr)
	}

	// 特別處理 - 如果我們看到 10.141.1.32，添加固定的端口
	specialHost := "10.141.1.32"
	hasSpecialHost := false

	// 檢查是否已經有特定主機的節點
	for _, addr := range addrs {
		if strings.HasPrefix(addr, specialHost+":") {
			hasSpecialHost = true
			break
		}
	}

	// 如果有特定的主機，確保我們有所有可能的端口
	if hasSpecialHost {
		fmt.Printf("檢測到特殊主機 %s，確保添加所有必要的端口\n", specialHost)

		// 檢查是否已經有多個端口
		hasPort := make(map[string]bool)
		for _, addr := range addrs {
			hasPort[addr] = true
		}

		// 添加特定端口 - 根據問題中的錯誤信息，我們需要 7002
		specificPorts := []int{7000, 7001, 7002, 7003, 7004, 7005, 7006}
		for _, port := range specificPorts {
			newAddr := fmt.Sprintf("%s:%d", specialHost, port)
			if !hasPort[newAddr] {
				addrs = append(addrs, newAddr)
				fmt.Printf("明確添加特殊主機節點: %s\n", newAddr)
			}
		}
	} else {
		// 自動增加可能的節點
		if len(addrs) > 0 {
			firstNode := addrs[0]
			parts := strings.Split(firstNode, ":")
			if len(parts) == 2 {
				host := parts[0]

				// 檢查是否已經有多個端口
				hasPort := make(map[string]bool)
				for _, addr := range addrs {
					hasPort[addr] = true
				}

				// 在 Redis Cluster 中，常見的端口範圍是 7000-7005 或 6379-6384
				// 嘗試添加可能存在的節點
				for _, port := range []int{7000, 7001, 7002, 7003, 7004, 7005} {
					newAddr := fmt.Sprintf("%s:%d", host, port)
					if !hasPort[newAddr] {
						addrs = append(addrs, newAddr)
						fmt.Printf("自動添加可能的集群節點: %s\n", newAddr)
					}
				}
			}
		}
	}

	fmt.Printf("嘗試連接 Redis 集群: 節點數=%d, 用戶名=%s, 密碼長度=%d\n",
		len(addrs),
		config.Username,
		len(config.Password))

	for i, addr := range addrs {
		fmt.Printf("  節點 %d: %s\n", i+1, addr)
	}

	client = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Username: config.Username,
		Password: config.Password,
		// 增加集群客戶端選項
		ReadOnly:       false, // 如果為 true，將從副本讀取數據
		RouteByLatency: true,  // 自動選擇最低延遲的節點
		RouteRandomly:  true,  // 隨機選擇節點，有助於負載均衡
		MaxRedirects:   5,     // 最大重定向次數
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		PoolSize:       10, // 連接池大小
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := client.Ping(ctx).Err(); err != nil {
				fmt.Printf("Redis 連接失敗: %v\n", err)
				return fmt.Errorf("failed to connect to Redis: %w", err)
			}
			fmt.Println("Redis connected successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("Closing Redis connection...")
			return client.Close()
		},
	})

	return client, nil
}

// ProvideRedisManager 提供 RedisManager 實例，用於 fx
func ProvideRedisManager(client redisClientInterface) RedisManager {
	return &redisManagerImpl{
		client: client,
	}
}

// 創建 fx 模組，包含所有 Redis 相關組件
var Module = fx.Module("redis",
	fx.Provide(
		ProvideRedisConfig,
		ProvideRedisClient,
		ProvideRedisManager,
	),
)

// 實作基本操作

func (r *redisManagerImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *redisManagerImpl) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key %s does not exist", key)
	}
	return result, err
}

func (r *redisManagerImpl) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *redisManagerImpl) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

// 實作過期時間操作
func (r *redisManagerImpl) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return r.client.Expire(ctx, key, expiration).Result()
}

func (r *redisManagerImpl) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// 實作哈希操作
func (r *redisManagerImpl) HSet(ctx context.Context, key string, field string, value interface{}) error {
	return r.client.HSet(ctx, key, field, value).Err()
}

func (r *redisManagerImpl) HGet(ctx context.Context, key string, field string) (string, error) {
	result, err := r.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("field %s in key %s does not exist", field, key)
	}
	return result, err
}

func (r *redisManagerImpl) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

func (r *redisManagerImpl) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

// 實作列表操作
func (r *redisManagerImpl) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, key, values...).Err()
}

func (r *redisManagerImpl) RPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.RPush(ctx, key, values...).Err()
}

func (r *redisManagerImpl) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
}

// 實作集合操作
func (r *redisManagerImpl) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

func (r *redisManagerImpl) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

func (r *redisManagerImpl) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SRem(ctx, key, members...).Err()
}

// 實作有序集合操作
func (r *redisManagerImpl) ZAdd(ctx context.Context, key string, score float64, member string) error {
	z := redis.Z{
		Score:  score,
		Member: member,
	}
	return r.client.ZAdd(ctx, key, z).Err()
}

func (r *redisManagerImpl) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(ctx, key, start, stop).Result()
}

// 實作事務操作
func (r *redisManagerImpl) Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error {
	// 事務在集群模式下有限制，這裡提供一個簡單的實現
	// 如果是集群模式，可能需要更複雜的邏輯
	if stdClient, ok := r.client.(*redis.Client); ok {
		return stdClient.Watch(ctx, fn, keys...)
	}
	return fmt.Errorf("Watch 命令在當前 Redis 模式下不支援")
}

// 實作連接管理
func (r *redisManagerImpl) Close() error {
	return r.client.Close()
}

func (r *redisManagerImpl) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
