package service

import (
	"g38_lottery_servic/pkg/databaseManager"

	"gorm.io/gorm"
)

func ProvideGormDB(dbManager databaseManager.DatabaseManager) *gorm.DB {
	return dbManager.GetDB()
}
