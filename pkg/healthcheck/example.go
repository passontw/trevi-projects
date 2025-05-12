// 這個文件包含使用健康檢查包的示例代碼
package healthcheck

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ExampleBasicUsage 展示基本使用方法
func ExampleBasicUsage() {
	// 創建日誌記錄器
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 創建健康檢查管理器
	health := New(Config{
		Logger: logger,
	})

	// 添加自定義健康檢查
	health.AddReadinessCheck(&CustomChecker{
		Name_: "example-check",
		CheckFunc: func(r *http.Request) error {
			// 執行一些業務邏輯檢查
			return nil
		},
	})

	// 添加 Redis 健康檢查
	health.AddReadinessCheck(&RedisChecker{
		Name_: "example-redis",
		PingFunc: func(ctx context.Context) error {
			// 在實際應用中，您會使用真實的 Redis 客戶端
			// 例如: return redisClient.Ping(ctx).Err()
			return nil
		},
	})

	// 創建 HTTP 路由
	mux := http.NewServeMux()

	// 註冊自定義路由
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// 安裝健康檢查路由
	health.InstallHandlers(mux)

	// 創建並啟動服務器
	server := NewServer(ServerConfig{
		Addr:          ":8080",
		Handler:       mux,
		HealthManager: health,
		Logger:        logger,
	})

	// 啟動服務器
	err := server.Start(func(ctx context.Context) error {
		// 在這裡執行初始化操作
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	if err != nil {
		logger.Fatal("服務器錯誤", zap.Error(err))
	}
}

// ExampleFxUsage 展示與 fx 框架集成的使用方法
func ExampleFxUsage() {
	// 創建並啟動 fx 應用
	app := fx.New(
		// 提供日誌記錄器
		fx.Provide(func() (*zap.Logger, error) {
			return zap.NewProduction()
		}),

		// 引入健康檢查模組
		Module,

		// 提供 HTTP 服務器
		fx.Provide(func(health *Manager, logger *zap.Logger) *http.ServeMux {
			mux := http.NewServeMux()

			// 註冊自定義路由
			mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Hello from fx app!"))
			})

			// 安裝健康檢查路由
			health.InstallHandlers(mux)

			return mux
		}),

		// 提供數據庫連接
		fx.Provide(func() (*sql.DB, error) {
			// 創建並返回數據庫連接
			return sql.Open("mysql", "user:password@/dbname")
		}),

		// 註冊數據庫健康檢查
		fx.Invoke(func(health *Manager, db *sql.DB) {
			health.AddReadinessCheck(&DatabaseChecker{
				Name_: "mysql-db",
				DB:    db,
			})
		}),

		// 註冊 Redis 健康檢查 (使用自定義 ping 函數)
		fx.Invoke(func(health *Manager) {
			health.AddReadinessCheck(&RedisChecker{
				Name_: "redis-cache",
				PingFunc: func(ctx context.Context) error {
					// 在真實應用中，這裡會使用注入的 Redis 客戶端
					return nil
				},
			})
		}),

		// 啟動 HTTP 服務器
		fx.Invoke(func(mux *http.ServeMux, health *Manager, logger *zap.Logger, lc fx.Lifecycle) {
			server := NewServer(ServerConfig{
				Addr:          ":8080",
				Handler:       mux,
				HealthManager: health,
				Logger:        logger,
			})

			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// 啟動服務器
					go func() {
						if err := server.Start(nil); err != nil {
							logger.Error("服務器錯誤", zap.Error(err))
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					// 關閉服務器
					server.Shutdown()
					return nil
				},
			})
		}),
	)

	// 啟動應用
	if err := app.Start(context.Background()); err != nil {
		fmt.Printf("啟動錯誤: %v\n", err)
		return
	}

	// 運行應用
	<-app.Done()
}

// ExampleCustomRoute 展示自定義路由的使用方法
func ExampleCustomRoute() {
	// 創建日誌記錄器
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 創建健康檢查管理器，啟用自定義路由
	health := New(Config{
		Logger:       logger,
		CustomRoutes: true,
	})

	// 添加健康檢查
	health.AddLivenessCheck(&PingChecker{})
	health.AddReadinessCheck(&CustomChecker{
		Name_: "custom-ready",
		CheckFunc: func(r *http.Request) error {
			return nil
		},
	})

	// 創建 HTTP 路由
	mux := http.NewServeMux()

	// 註冊自定義路由
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, Custom Routes!"))
	})

	// 為健康檢查註冊自定義路徑
	customRoutes := map[string]string{
		"/custom/ping":   "liveness",
		"/custom/ready":  "readiness",
		"/custom/health": "health",
	}

	// 安裝自定義健康檢查路由
	health.InstallCustomHandlers(mux, customRoutes)

	// 創建並啟動服務器
	server := NewServer(ServerConfig{
		Addr:          ":8080",
		Handler:       mux,
		HealthManager: health,
		Logger:        logger,
	})

	// 啟動服務器
	err := server.Start(func(ctx context.Context) error {
		// 在這裡執行初始化操作
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	if err != nil {
		logger.Fatal("服務器錯誤", zap.Error(err))
	}
}
