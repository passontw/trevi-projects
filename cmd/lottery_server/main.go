package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/pkg/databaseManager"
	"g38_lottery_service/pkg/redisManager"
	"g38_lottery_service/pkg/websocketManager"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 全局變數
var (
	appConfig *config.Config
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

// 初始化配置
func initConfig() (*config.Config, error) {
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

// 初始化資料庫連接
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
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

	return db, nil
}

// 初始化Redis連接
func initRedis() (redisManager.RedisManager, error) {
	// 檢查配置有效性
	if appConfig.Redis.Addr == "" {
		appConfig.Redis.Addr = "localhost:6379" // 使用默認地址
		log.Printf("[警告] Redis地址未配置，使用默認地址: %s", appConfig.Redis.Addr)
	}

	redisConfig := &redisManager.RedisConfig{
		Addr:         appConfig.Redis.Addr,
		Username:     appConfig.Redis.Username,
		Password:     appConfig.Redis.Password,
		DB:           appConfig.Redis.DB,
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
			appConfig.Redis.Addr, appConfig.Redis.Username, appConfig.Redis.DB)
		client.Close() // 關閉失敗的連接
		return nil, fmt.Errorf("無法連接到Redis: %w", err)
	}

	// 使用成功連接的客戶端創建 RedisManager
	redis := redisManager.ProvideRedisManager(client)

	log.Printf("[成功] 已連接到Redis %s", appConfig.Redis.Addr)
	return redis, nil
}

// 認證函數 - 根據您的需求自行實現
func authenticateToken(token string) (uint, error) {
	// 模擬認證，實際場景請替換為您的認證邏輯
	return 1000, nil
}

// 獲取當前主機IP
func getIpAddress() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return hostname
}

// 設定CORS中間件
func setupCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// 設定遊戲端路由
func setupPlayerRouter(wsHandler *websocketManager.DualWebSocketHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(setupCORS())

	// 健康檢查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"type":   "player",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 版本資訊
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": appConfig.Server.Version,
		})
	})

	// WebSocket 連接端點
	router.GET("/ws", wsHandler.HandlePlayerConnection)

	// 認證端點
	router.POST("/auth", wsHandler.HandlePlayerAuthRequest)

	return router
}

// 設定荷官端路由
func setupDealerRouter(wsHandler *websocketManager.DualWebSocketHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(setupCORS())

	// 健康檢查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"type":   "dealer",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 版本資訊
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": appConfig.Server.Version,
		})
	})

	// WebSocket 連接端點
	router.GET("/ws", wsHandler.HandleDealerConnection)

	// 認證端點
	router.POST("/auth", wsHandler.HandleDealerAuthRequest)

	return router
}

// 啟動 HTTP 服務器
func startServer(ctx context.Context, router *gin.Engine, port uint64, serverType string) *http.Server {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		log.Printf("[%s] 伺服器開始監聽 http://%s:%d", serverType, appConfig.Server.Host, port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[%s] 伺服器啟動失敗: %v", serverType, err)
		}
	}()

	return server
}

// 遊戲服務介面
type GameService interface {
	Initialize() error
	Shutdown(ctx context.Context) error
}

// 簡易遊戲服務實現
type SimpleGameService struct {
	wsService *websocketManager.DualWebSocketService
	db        *gorm.DB
	logger    *SimpleLogger
}

func NewGameService(wsService *websocketManager.DualWebSocketService, db *gorm.DB) GameService {
	return &SimpleGameService{
		wsService: wsService,
		db:        db,
		logger:    NewSimpleLogger("[遊戲服務] "),
	}
}

func (s *SimpleGameService) Initialize() error {
	s.logger.Info("初始化遊戲服務")
	// 初始化遊戲狀態、載入配置等
	return nil
}

func (s *SimpleGameService) Shutdown(ctx context.Context) error {
	s.logger.Info("關閉遊戲服務")
	// 保存遊戲狀態、清理資源等
	return nil
}

func main() {
	// 設置日誌格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("樂透遊戲服務初始化中...")

	var err error
	// 初始化配置
	appConfig, err = initConfig()
	if err != nil {
		log.Fatalf("初始化配置失敗: %v", err)
	}

	// 初始化資料庫
	db, err := initDatabase(appConfig)
	if err != nil {
		log.Fatalf("初始化資料庫失敗: %v", err)
	}

	// 初始化 Redis
	_, err = initRedis()
	if err != nil {
		log.Printf("[警告] Redis 初始化失敗: %v", err)
		// 繼續執行，即使 Redis 連接失敗
	}

	// 創建 WebSocket 服務
	wsService := websocketManager.NewDualWebSocketService(
		authenticateToken,
		authenticateToken)

	// 初始化遊戲服務
	gameService := NewGameService(wsService, db)
	if err := gameService.Initialize(); err != nil {
		log.Printf("[警告] 遊戲服務初始化失敗: %v", err)
	}

	// 創建 WebSocket 處理器
	wsHandler := websocketManager.NewDualWebSocketHandler(wsService)

	// 設置玩家路由
	playerRouter := setupPlayerRouter(wsHandler)

	// 設置莊家路由
	dealerRouter := setupDealerRouter(wsHandler)

	// 創建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 啟動玩家服務器
	playerServer := startServer(ctx, playerRouter, uint64(appConfig.Server.PlayerWSPort), "玩家")

	// 啟動莊家服務器
	dealerServer := startServer(ctx, dealerRouter, uint64(appConfig.Server.DealerWSPort), "莊家")

	// 設置優雅關閉服務
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在關閉服務...")

	// 設置關閉超時
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 關閉玩家服務器
	if err := playerServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("玩家服務器關閉失敗: %v", err)
	}

	// 關閉莊家服務器
	if err := dealerServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("莊家服務器關閉失敗: %v", err)
	}

	// 關閉遊戲服務
	if err := gameService.Shutdown(shutdownCtx); err != nil {
		log.Printf("[警告] 遊戲服務關閉時發生錯誤: %v", err)
	}

	log.Println("服務已關閉")
}
