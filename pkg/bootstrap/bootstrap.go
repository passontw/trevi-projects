package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"g38_lottery_servic/pkg/config"
	"g38_lottery_servic/pkg/database"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module 返回應用的基本依賴模塊
func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			// 配置相關
			loadEnv,
			config.LoadNacosConfig,
			config.NewNacosClient,
			loadServiceConfig,

			// 數據庫相關
			database.NewPostgresDB,
			database.NewRedisClient,

			// 服務註冊
			registerService,
		),
	)
}

// loadEnv 加載環境變量
func loadEnv() error {
	return godotenv.Load()
}

// loadServiceConfig 加載服務配置
func loadServiceConfig(nacosClient *config.NacosClient) (*config.ServiceConfig, error) {
	return config.LoadConfig(nacosClient)
}

// registerService 註冊服務到Nacos
func registerService(lc fx.Lifecycle, nacosClient *config.NacosClient, cfg *config.ServiceConfig) error {
	if nacosClient == nil || !nacosClient.Config.EnableNacos {
		log.Println("Nacos is disabled, skipping service registration")
		return nil
	}

	ip := config.GetOutboundIP()
	port, err := strconv.ParseUint(cfg.Server.Port, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse port: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			success, err := nacosClient.RegisterService(ip, port)
			if err != nil || !success {
				return fmt.Errorf("failed to register service: %w", err)
			}
			log.Printf("Service registered to Nacos: %s:%d", ip, port)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			success, err := nacosClient.DeregisterService(ip, port)
			if err != nil || !success {
				return fmt.Errorf("failed to deregister service: %w", err)
			}
			log.Printf("Service deregistered from Nacos: %s:%d", ip, port)
			return nil
		},
	})

	return nil
}

// SetupGracefulShutdown 設置優雅關閉
func SetupGracefulShutdown(lc fx.Lifecycle, db *gorm.DB) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Println("Shutting down application...")
			if db != nil {
				sqlDB, err := db.DB()
				if err != nil {
					log.Printf("Error getting DB instance: %v", err)
				} else {
					if err := sqlDB.Close(); err != nil {
						log.Printf("Error closing database connection: %v", err)
					} else {
						log.Println("Database connection closed")
					}
				}
			}
			return nil
		},
	})
}

// WaitForSignal 等待信號以優雅關閉
func WaitForSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Received shutdown signal")
}
