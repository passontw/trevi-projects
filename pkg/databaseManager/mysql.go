package databaseManager

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MySQLConfig 存儲 MySQL 連接的配置項
type MySQLConfig struct {
	Host      string
	Port      int
	User      string
	Password  string
	Name      string
	Charset   string
	ParseTime bool
	Loc       string
}

// DatabaseManager 提供數據庫操作的介面
type DatabaseManager interface {
	// 獲取 GORM DB 實例
	GetDB() *gorm.DB
	// 關閉數據庫連接
	Close() error
}

// mysqlManagerImpl 是 DatabaseManager 介面的實作
type mysqlManagerImpl struct {
	db *gorm.DB
}

// GetDB 返回 GORM DB 實例
func (m *mysqlManagerImpl) GetDB() *gorm.DB {
	return m.db
}

// Close 關閉數據庫連接
func (m *mysqlManagerImpl) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

// NewMySQLManager 創建一個新的 MySQL 數據庫管理器
func NewMySQLManager(config *MySQLConfig) (DatabaseManager, error) {
	// 打印配置信息，便於診斷
	fmt.Printf("MySQL配置: Host=%s, Port=%d, User=%s, Name=%s, 密碼%s\n",
		config.Host, config.Port, config.User, config.Name,
		func(pwd string) string {
			if pwd == "" {
				return "未設置"
			}
			return "已設置"
		}(config.Password))

	// 檢查端口是否有效，如果無效則使用默認值
	port := config.Port
	if port <= 0 || port > 65535 {
		port = 3306 // 使用默認的MySQL端口
		fmt.Printf("檢測到無效端口 %d，使用默認端口 3306\n", config.Port)
	}

	// MySQL 的 DSN 格式
	var dsn string
	if config.Password == "" {
		// 無密碼連接格式: username@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
		dsn = fmt.Sprintf("%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&allowNativePasswords=true",
			config.User,
			config.Host,
			port,
			config.Name,
			config.Charset,
			config.ParseTime,
			config.Loc,
		)
	} else {
		// 有密碼連接格式: username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&allowNativePasswords=true",
			config.User,
			config.Password,
			config.Host,
			port,
			config.Name,
			config.Charset,
			config.ParseTime,
			config.Loc,
		)
	}

	fmt.Printf("MySQL連接字符串: %s\n", dsn)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("數據庫連接失敗: %v\n", err)
		fmt.Printf("嘗試診斷問題: \n")
		fmt.Printf("- 請確認 TiDB/MySQL 服務器正在運行於 %s:%d\n", config.Host, port)
		fmt.Printf("- 請確認 TiDB/MySQL 用戶 '%s' 存在且有權限訪問數據庫 '%s'\n", config.User, config.Name)
		fmt.Printf("- 請確認密碼設置正確\n")
		fmt.Printf("- 請確認 TiDB/MySQL 服務器允許來自 %s 的連接\n", config.Host)

		// 嘗試不使用數據庫名稱連接
		fmt.Printf("嘗試不指定數據庫名稱連接...\n")
		dsnWithoutDB := fmt.Sprintf("%s@tcp(%s:%d)/?charset=%s&parseTime=%t&loc=%s&allowNativePasswords=true",
			config.User,
			config.Host,
			port,
			config.Charset,
			config.ParseTime,
			config.Loc,
		)
		db, testErr := gorm.Open(mysql.Open(dsnWithoutDB), &gorm.Config{})
		if testErr == nil {
			fmt.Printf("不指定數據庫名稱連接成功! 嘗試創建數據庫 '%s'...\n", config.Name)

			// 創建數據庫
			result := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", config.Name))
			if result.Error != nil {
				fmt.Printf("創建數據庫失敗: %v\n", result.Error)
			} else {
				fmt.Printf("數據庫創建成功或已存在，嘗試重新連接...\n")

				// 重新嘗試原始連接
				db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
				if err == nil {
					fmt.Printf("連接成功！\n")
					sqlDB, _ := db.DB()
					sqlDB.SetMaxOpenConns(10)
					sqlDB.SetMaxIdleConns(5)
					sqlDB.SetConnMaxLifetime(time.Hour)
					return &mysqlManagerImpl{db: db}, nil
				}
			}
		} else {
			fmt.Printf("不指定數據庫名稱連接也失敗: %v\n", testErr)
		}

		// 嘗試不帶密碼連接
		if config.Password != "" {
			fmt.Printf("嘗試不帶密碼連接...\n")
			dsnWithoutPassword := fmt.Sprintf("%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&allowNativePasswords=true",
				config.User,
				config.Host,
				port,
				config.Name,
				config.Charset,
				config.ParseTime,
				config.Loc,
			)
			_, testErr := gorm.Open(mysql.Open(dsnWithoutPassword), &gorm.Config{})
			if testErr == nil {
				fmt.Printf("無密碼連接成功! 請將密碼設置為空字符串\n")
			} else {
				fmt.Printf("無密碼連接也失敗: %v\n", testErr)
			}
		}

		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	db.Debug()

	// 設置連接池
	sqlDB.SetMaxOpenConns(10)           // 最大連接數
	sqlDB.SetMaxIdleConns(5)            // 最大空閒連接數
	sqlDB.SetConnMaxLifetime(time.Hour) // 連接最大生命週期

	// 驗證連接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &mysqlManagerImpl{db: db}, nil
}

// ProvideMySQLConfig 提供 MySQL 配置，用於 fx
func ProvideMySQLConfig(cfg interface{}) *MySQLConfig {
	// 這裡需要根據實際的配置結構進行調整
	// 假設 cfg 是一個包含 Database 字段的結構
	config, ok := cfg.(interface {
		GetDatabaseHost() string
		GetDatabasePort() int
		GetDatabaseUser() string
		GetDatabasePassword() string
		GetDatabaseName() string
	})

	if !ok {
		// 如果無法轉換，返回默認配置
		return &MySQLConfig{
			Host:      "127.0.0.1",
			Port:      3306,
			User:      "root",
			Password:  "",
			Name:      "test",
			Charset:   "utf8mb4",
			ParseTime: true,
			Loc:       "Local",
		}
	}

	// 如果主機名是 localhost，替換為 127.0.0.1 以強制使用 IPv4
	host := config.GetDatabaseHost()
	if host == "localhost" {
		host = "127.0.0.1"
	}

	return &MySQLConfig{
		Host:      host,
		Port:      config.GetDatabasePort(),
		User:      config.GetDatabaseUser(),
		Password:  config.GetDatabasePassword(),
		Name:      config.GetDatabaseName(),
		Charset:   "utf8mb4",
		ParseTime: true,
		Loc:       "Local",
	}
}

// ProvideMySQLDatabaseManager 提供 MySQL DatabaseManager 實例，用於 fx
func ProvideMySQLDatabaseManager(lc fx.Lifecycle, config *MySQLConfig) (DatabaseManager, error) {
	manager, err := NewMySQLManager(config)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Println("MySQL database connected successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("Closing MySQL database connection...")
			return manager.Close()
		},
	})

	return manager, nil
}
