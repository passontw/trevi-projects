package database

import (
	"log"
	"os"
	"time"

	"g38_lottery_servic/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresDB 初始化 PostgreSQL 連接
func NewPostgresDB(cfg config.DatabaseConfig) *gorm.DB {
	// 配置 GORM 日誌
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)

	// 連接數據庫
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 獲取通用數據庫對象以配置連接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection pool: %v", err)
	}

	// 設置連接池參數
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connection established")
	return db
}
