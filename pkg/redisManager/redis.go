package redisManager

import (
	"context"
	"fmt"
	"log"
	"time"

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
	Addr     string
	Username string
	Password string
	DB       int
	// 自定義連接超時設定
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

// RedisAddr 返回格式化後的 Redis 地址
func (c *RedisConfig) RedisAddr() string {
	return c.Addr
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
	
	// 獲取原始客戶端
	GetClient() *redis.Client
}

// redisManagerImpl 是 RedisManager 介面的實作
type redisManagerImpl struct {
	client *redis.Client
}

// NewRedisClient 初始化 Redis 客戶端 (從 databaseManager/redis.go 整合)
func NewRedisClient(cfg *RedisConfig) *redis.Client {
	options := &redis.Options{
		Addr:     cfg.RedisAddr(),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	
	// 加入自定義連接參數，如果有設定的話
	if cfg.DialTimeout > 0 {
		options.DialTimeout = cfg.DialTimeout
	} else {
		options.DialTimeout = 5 * time.Second
	}
	
	if cfg.ReadTimeout > 0 {
		options.ReadTimeout = cfg.ReadTimeout
	} else {
		options.ReadTimeout = 3 * time.Second
	}
	
	if cfg.WriteTimeout > 0 {
		options.WriteTimeout = cfg.WriteTimeout
	} else {
		options.WriteTimeout = 3 * time.Second
	}
	
	if cfg.PoolSize > 0 {
		options.PoolSize = cfg.PoolSize
	} else {
		options.PoolSize = 20
	}
	
	if cfg.MinIdleConns > 0 {
		options.MinIdleConns = cfg.MinIdleConns
	} else {
		options.MinIdleConns = 10
	}

	client := redis.NewClient(options)

	// 測試連接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis connection established")
	return client
}

// NewRedisManager 創建一個新的 Redis 管理器
func NewRedisManager(config *RedisConfig) RedisManager {
	client := NewRedisClient(config)
	return &redisManagerImpl{
		client: client,
	}
}

// ProvideRedisClient 提供 Redis 客戶端實例，用於 fx
func ProvideRedisClient(lc fx.Lifecycle, config *RedisConfig) *redis.Client {
	client := NewRedisClient(config)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := client.Ping(ctx).Err(); err != nil {
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

	return client
}

// ProvideRedisManager 提供 RedisManager 實例，用於 fx
func ProvideRedisManager(client *redis.Client) RedisManager {
	return &redisManagerImpl{
		client: client,
	}
}

// 創建 fx 模組，包含所有 Redis 相關組件
// 移除模块定义，它会在 core 包和主程序中使用

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
	return r.client.Watch(ctx, fn, keys...)
}

// 實作連接管理
func (r *redisManagerImpl) Close() error {
	return r.client.Close()
}

func (r *redisManagerImpl) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetClient 獲取原始的 Redis 客戶端
func (r *redisManagerImpl) GetClient() *redis.Client {
	return r.client
}
