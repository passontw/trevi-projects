package databaseManager

import "gorm.io/gorm"

// DatabaseManager 提供數據庫操作的介面
type DatabaseManager interface {
	// 獲取 GORM DB 實例
	GetDB() *gorm.DB
	// 關閉數據庫連接
	Close() error
}
