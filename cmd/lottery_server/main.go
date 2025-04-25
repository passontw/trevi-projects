// 樂透遊戲服務主程序
package main

// @title          G38 Lottery Service API
// @version        1.0
// @description    樂透遊戲服務 API 文檔
// @termsOfService http://swagger.io/terms/

// @contact.name  API Support
// @contact.url   http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url  http://www.apache.org/licenses/LICENSE-2.0.html

// @host     localhost:3001
// @BasePath /

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"g38_lottery_service/game"
	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/handlers"
	"g38_lottery_service/internal/services"
	"g38_lottery_service/pkg/databaseManager"
	"g38_lottery_service/pkg/redisManager"
	"g38_lottery_service/pkg/websocketManager"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"

	_ "g38_lottery_service/docs/swagger" // 導入swagger文檔
)

// 自定義簡易日誌記錄器
type SimpleLogger struct {
	prefix string
}

func NewSimpleLogger(prefix string) *SimpleLogger {
	return &SimpleLogger{prefix: prefix}
}

func (l *SimpleLogger) Info(format string, args ...interface{}) {
	log.Printf(l.prefix+"INFO: "+format, args...)
}

func (l *SimpleLogger) Warn(format string, args ...interface{}) {
	log.Printf(l.prefix+"WARN: "+format, args...)
}

func (l *SimpleLogger) Error(format string, args ...interface{}) {
	log.Printf(l.prefix+"ERROR: "+format, args...)
}

func (l *SimpleLogger) Debug(format string, args ...interface{}) {
	log.Printf(l.prefix+"DEBUG: "+format, args...)
}

// 初始化配置模組
func provideConfig() (*config.Config, error) {
	// 設置默認環境變量 - TiDB 配置
	os.Setenv("MYSQL_HOST", "127.0.0.1")
	os.Setenv("MYSQL_PORT", "4000")
	os.Setenv("MYSQL_DATABASE", "g38_lottery")
	os.Setenv("MYSQL_USER", "root")
	os.Setenv("MYSQL_PASSWORD", "")

	// 載入配置
	cfg := &config.Config{}
	cfg.Server.Host = getIpAddress()
	cfg.Server.Port = 8080
	cfg.Server.PlayerWSPort = 3001
	cfg.Server.DealerWSPort = 3002
	cfg.Server.Version = "1.0.0"

	// 設置 Nacos 配置
	cfg.Nacos.Host = "172.237.27.51"
	cfg.Nacos.Port = 8848
	cfg.Nacos.NamespaceId = "test_golang"
	cfg.Nacos.Group = "TEST_GOLANG_ENVS"
	cfg.Nacos.DataId = "g38_lottery"
	cfg.Nacos.Username = "username"
	cfg.Nacos.Password = "password"
	cfg.EnableNacos = true // 啟用 Nacos

	// 設置備用配置（如果 Nacos 無法連接時使用）
	cfg.Database.Host = "127.0.0.1"
	cfg.Database.Port = 4000
	cfg.Database.User = "root"
	cfg.Database.Password = ""
	cfg.Database.Name = "g38_lottery"

	// 設置 Redis 備用配置
	cfg.Redis.Addr = "localhost:6379"
	cfg.Redis.DB = 0

	// 嘗試從 Nacos 載入配置
	if cfg.EnableNacos {
		// 包裝在 recover 中，避免 Nacos 配置錯誤導致程序崩潰
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[嚴重錯誤] Nacos 配置載入發生嚴重錯誤: %v", r)
					log.Printf("[警告] 將使用備用配置繼續運行")
				}
			}()

			err := config.LoadFromNacos(cfg)
			if err != nil {
				log.Printf("[錯誤] 從 Nacos 載入配置失敗: %v", err)
				log.Printf("[警告] 將使用備用配置: Host=%s, Port=%d, DB=%s",
					cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
			} else {
				log.Printf("[成功] 從 Nacos 載入配置成功")
				log.Printf("[信息] 數據庫配置: Host=%s, Port=%d, DB=%s",
					cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
			}
		}()
	}

	// 驗證必要配置是否存在
	if cfg.Database.Host == "" || cfg.Database.Port == 0 || cfg.Database.Name == "" {
		log.Printf("[嚴重錯誤] 資料庫配置不完整，將使用預設配置")
		// 使用預設配置
		cfg.Database.Host = "127.0.0.1"
		cfg.Database.Port = 4000
		cfg.Database.User = "root"
		cfg.Database.Password = ""
		cfg.Database.Name = "g38_lottery"
	}

	// 顯示最終配置
	log.Printf("MySQL配置: Host=%s, Port=%d, User=%s, Name=%s, 密碼%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Name,
		func() string {
			if cfg.Database.Password == "" {
				return "未設置"
			} else {
				return "已設置"
			}
		}(),
	)

	// 顯示資料庫連接字符串
	dsn := fmt.Sprintf("%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&allowNativePasswords=true",
		cfg.Database.User,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)
	log.Printf("MySQL連接字符串: %s", dsn)

	return cfg, nil
}

// 初始化資料庫模組
func provideDatabaseManager(cfg *config.Config) (databaseManager.DatabaseManager, error) {
	dbConfig := &databaseManager.MySQLConfig{
		Host:      cfg.Database.Host,
		Port:      cfg.Database.Port,
		User:      cfg.Database.User,
		Password:  cfg.Database.Password,
		Name:      cfg.Database.Name,
		Charset:   "utf8mb4",
		ParseTime: true,
		Loc:       "Local",
	}

	manager, err := databaseManager.NewMySQLManager(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化數據庫失敗: %w", err)
	}

	db := manager.GetDB()

	// 嘗試連接資料庫
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("獲取資料庫連接失敗: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("無法連接到數據庫: %w", err)
	}

	log.Printf("[成功] 已連接到數據庫 %s@%s:%d/%s",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	return manager, nil
}

// 提供 GORM DB 實例
func provideGormDB(dbManager databaseManager.DatabaseManager) *gorm.DB {
	return dbManager.GetDB()
}

// 初始化Redis模組
func provideRedisManager(cfg *config.Config) (redisManager.RedisManager, error) {
	// 檢查配置有效性
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379" // 使用默認地址
		log.Printf("[警告] Redis地址未配置，使用默認地址: %s", cfg.Redis.Addr)
	}

	redisConfig := &redisManager.RedisConfig{
		Addr:         cfg.Redis.Addr,
		Username:     cfg.Redis.Username,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	}

	// 使用安全的方式創建 Redis 客戶端
	// 避免在連接失敗時直接 Fatal
	options := &redis.Options{
		Addr:         redisConfig.Addr,
		Username:     redisConfig.Username,
		Password:     redisConfig.Password,
		DB:           redisConfig.DB,
		DialTimeout:  redisConfig.DialTimeout,
		ReadTimeout:  redisConfig.ReadTimeout,
		WriteTimeout: redisConfig.WriteTimeout,
		PoolSize:     redisConfig.PoolSize,
		MinIdleConns: redisConfig.MinIdleConns,
	}

	client := redis.NewClient(options)

	// 測試連接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[警告] Redis連接測試失敗: %v", err)
		log.Printf("[警告] Redis配置: 地址=%s, 用戶=%s, DB=%d",
			cfg.Redis.Addr, cfg.Redis.Username, cfg.Redis.DB)
		client.Close() // 關閉失敗的連接
		return nil, fmt.Errorf("無法連接到Redis: %w", err)
	}

	// 使用成功連接的客戶端創建 RedisManager
	redis := redisManager.ProvideRedisManager(client)

	log.Printf("[成功] 已連接到Redis %s", cfg.Redis.Addr)
	return redis, nil
}

// 處理Redis模組初始化錯誤的回退方案
func provideRedisManagerWithFallback(lc fx.Lifecycle, cfg *config.Config) (redisManager.RedisManager, error) {
	redis, err := provideRedisManager(cfg)
	if err != nil {
		log.Printf("[警告] 使用無操作的Redis替代實現")
		// 提供一個空的實現，避免系統崩潰
		return &noOpRedisManager{}, nil
	}

	return redis, nil
}

// 空操作的Redis實現
type noOpRedisManager struct{}

func (r *noOpRedisManager) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (r *noOpRedisManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

func (r *noOpRedisManager) Delete(ctx context.Context, keys ...string) error {
	return nil
}

func (r *noOpRedisManager) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (r *noOpRedisManager) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return true, nil
}

func (r *noOpRedisManager) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (r *noOpRedisManager) HSet(ctx context.Context, key string, field string, value interface{}) error {
	return nil
}

func (r *noOpRedisManager) HGet(ctx context.Context, key string, field string) (string, error) {
	return "", nil
}

func (r *noOpRedisManager) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return nil, nil
}

func (r *noOpRedisManager) HDel(ctx context.Context, key string, fields ...string) error {
	return nil
}

func (r *noOpRedisManager) LPush(ctx context.Context, key string, values ...interface{}) error {
	return nil
}

func (r *noOpRedisManager) RPush(ctx context.Context, key string, values ...interface{}) error {
	return nil
}

func (r *noOpRedisManager) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return nil, nil
}

func (r *noOpRedisManager) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return nil
}

func (r *noOpRedisManager) SMembers(ctx context.Context, key string) ([]string, error) {
	return nil, nil
}

func (r *noOpRedisManager) SRem(ctx context.Context, key string, members ...interface{}) error {
	return nil
}

func (r *noOpRedisManager) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return nil
}

func (r *noOpRedisManager) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return nil, nil
}

func (r *noOpRedisManager) Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error {
	return nil
}

func (r *noOpRedisManager) Incr(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (r *noOpRedisManager) GetClient() *redis.Client {
	return nil
}

func (r *noOpRedisManager) Close() error {
	return nil
}

func (r *noOpRedisManager) Ping(ctx context.Context) error {
	return nil
}

// 提供認證函數 - 根據您的需求自行實現
func provideAuthFunc() func(string) (uint, error) {
	return func(token string) (uint, error) {
		// 模擬認證，實際場景請替換為您的認證邏輯
		return 1000, nil
	}
}

// 提供WebSocket服務模組
func provideWebSocketService(authFunc func(string) (uint, error)) *websocketManager.DualWebSocketService {
	return websocketManager.NewDualWebSocketService(authFunc, authFunc)
}

// 提供WebSocket處理器模組
func provideWebSocketHandler(service *websocketManager.DualWebSocketService) *websocketManager.DualWebSocketHandler {
	return websocketManager.NewDualWebSocketHandler(service)
}

// 提供Gin路由器模組
func providePlayerRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(handlers.SetupGinCORS())
	return router
}

func provideDealerRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(handlers.SetupGinCORS())
	return router
}

// 註冊應用生命週期管理
func registerHooks(
	lc fx.Lifecycle,
	cfg *config.Config,
	playerRouter *gin.Engine,
	dealerRouter *gin.Engine,
	gameService services.GameService,
) {
	// 創建上下文用於取消
	_, cancel := context.WithCancel(context.Background())

	// 啟動階段
	lc.Append(fx.Hook{
		OnStart: func(appCtx context.Context) error {
			// 啟動遊戲服務
			if err := gameService.Initialize(); err != nil {
				log.Printf("[警告] 遊戲服務初始化失敗: %v", err)
				// 不返回錯誤，允許系統繼續運行
			}

			// 啟動玩家服務器
			go func() {
				addr := fmt.Sprintf(":%d", cfg.Server.PlayerWSPort)
				log.Printf("[玩家] 伺服器開始監聽 http://%s:%d", cfg.Server.Host, cfg.Server.PlayerWSPort)
				log.Printf("[玩家] Swagger API 文檔可訪問: http://%s:%d/swagger/index.html", cfg.Server.Host, cfg.Server.PlayerWSPort)

				server := &http.Server{
					Addr:    addr,
					Handler: playerRouter,
				}

				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("[玩家] 伺服器啟動失敗: %v", err)
				}
			}()

			// 啟動莊家服務器
			go func() {
				addr := fmt.Sprintf(":%d", cfg.Server.DealerWSPort)
				log.Printf("[莊家] 伺服器開始監聽 http://%s:%d", cfg.Server.Host, cfg.Server.DealerWSPort)
				log.Printf("[莊家] Swagger API 文檔可訪問: http://%s:%d/swagger/index.html", cfg.Server.Host, cfg.Server.DealerWSPort)

				server := &http.Server{
					Addr:    addr,
					Handler: dealerRouter,
				}

				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("[莊家] 伺服器啟動失敗: %v", err)
				}
			}()

			// 設置信號處理
			go func() {
				quit := make(chan os.Signal, 1)
				signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
				<-quit
				log.Println("接收到關閉信號...")
				cancel() // 取消上下文，觸發 OnStop
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("正在關閉服務...")

			// 關閉遊戲服務
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			if err := gameService.Shutdown(shutdownCtx); err != nil {
				log.Printf("[警告] 遊戲服務關閉時發生錯誤: %v", err)
			}

			log.Println("服務已關閉")
			return nil
		},
	})
}

// 註冊路由處理
func registerRoutes(
	playerHandler *handlers.PlayerHandler,
	dealerHandler *handlers.DealerHandler,
	playerRouter *gin.Engine,
	dealerRouter *gin.Engine,
) {
	playerHandler.RegisterRoutes(playerRouter)
	dealerHandler.RegisterRoutes(dealerRouter)
}

// 獲取當前主機IP
func getIpAddress() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return hostname
}

// 提供必要的配置項目
func provideConfigParams(cfg *config.Config) (
	host string,
	playerPort int,
	dealerPort int,
	version string,
) {
	return cfg.Server.Host, int(cfg.Server.PlayerWSPort), int(cfg.Server.DealerWSPort), cfg.Server.Version
}

func main() {
	// 設置日誌格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("樂透遊戲服務初始化中...")
	log.Println("Swagger API 文檔可在 http://localhost:3001/swagger/index.html 和 http://localhost:3002/swagger/index.html 訪問")

	app := fx.New(
		// 註冊模組
		game.Module,
		services.Module,
		handlers.Module,

		// 提供基礎組件
		fx.Provide(
			provideConfig,
			provideDatabaseManager,
			provideGormDB,
			provideRedisManagerWithFallback,
			provideAuthFunc,
			provideWebSocketService,
			provideWebSocketHandler,
			fx.Annotate(
				providePlayerRouter,
				fx.ResultTags(`name:"playerRouter"`),
			),
			fx.Annotate(
				provideDealerRouter,
				fx.ResultTags(`name:"dealerRouter"`),
			),
		),

		// 提供配置參數
		fx.Provide(
			fx.Annotate(
				provideConfigParams,
				fx.ResultTags(`name:"serverHost"`, `name:"playerPort"`, `name:"dealerPort"`, `name:"serverVersion"`),
			),
		),

		// 註冊路由和生命週期鉤子
		fx.Invoke(
			fx.Annotate(
				registerRoutes,
				fx.ParamTags(``, ``, `name:"playerRouter"`, `name:"dealerRouter"`),
			),
			fx.Annotate(
				registerHooks,
				fx.ParamTags(``, ``, `name:"playerRouter"`, `name:"dealerRouter"`, ``),
			),
		),
	)

	// 啟動應用程式
	app.Run()
}
