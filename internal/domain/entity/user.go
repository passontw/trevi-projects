package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID               int        `gorm:"primaryKey;column:id" json:"id" example:"1"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;default:now()" json:"created_at" example:"2025-02-16T16:05:00.763995Z"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;not null;default:now()" json:"updated_at" example:"2025-02-16T16:05:00.763995Z"`
	DeletedAt        *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty" example:"2025-02-16T16:05:00.763995Z"`
	Name             string     `gorm:"column:name;type:varchar(20);not null" json:"name" binding:"required,max=20" example:"testdemo001"`
	Phone            string     `gorm:"column:phone;type:varchar(20);not null;uniqueIndex:idx_users_phone,where:deleted_at IS NULL" json:"phone" binding:"required,max=20" example:"0987654321"`
	Password         string     `gorm:"column:password;type:varchar(200);not null" json:"-"`
	AvailableBalance float64    `gorm:"column:available_balance;type:decimal(20,2);not null;default:0.00;check:available_balance >= 0" json:"available_balance" example:"1000.00"`
	FrozenBalance    float64    `gorm:"column:frozen_balance;type:decimal(20,2);not null;default:0.00;check:frozen_balance >= 0" json:"frozen_balance" example:"0.00"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.AvailableBalance < 0 {
		return errors.New("available balance cannot be negative")
	}
	if u.FrozenBalance < 0 {
		return errors.New("frozen balance cannot be negative")
	}

	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	return nil
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) ValidateBalance(amount float64, operation string) error {
	switch operation {
	case "deduct", "freeze":
		if u.AvailableBalance < amount {
			return errors.New("insufficient available balance")
		}
	case "unfreeze":
		if u.FrozenBalance < amount {
			return errors.New("insufficient frozen balance")
		}
	}
	return nil
}

func (u *User) ProcessBalanceChange(tx *gorm.DB, operationType string, amount float64, description string, operator string, referenceID string, remark JSON) error {
	if amount < 0 {
		return errors.New("amount must be positive")
	}

	beforeBalance := u.AvailableBalance
	beforeFrozen := u.FrozenBalance
	var afterBalance, afterFrozen float64

	switch operationType {
	case BalanceOperationAdd:
		afterBalance = beforeBalance + amount
		afterFrozen = beforeFrozen

	case BalanceOperationDeduct:
		if beforeBalance < amount {
			return errors.New("insufficient balance")
		}
		afterBalance = beforeBalance - amount
		afterFrozen = beforeFrozen

	case BalanceOperationFreeze:
		if beforeBalance < amount {
			return errors.New("insufficient balance to freeze")
		}
		afterBalance = beforeBalance - amount
		afterFrozen = beforeFrozen + amount

	case BalanceOperationUnfreeze:
		if beforeFrozen < amount {
			return errors.New("insufficient frozen balance")
		}
		afterBalance = beforeBalance + amount
		afterFrozen = beforeFrozen - amount

	default:
		return errors.New("invalid operation type")
	}

	// 創建餘額變動記錄
	record := BalanceRecord{
		UserID:        u.ID,
		Type:          operationType,
		Amount:        amount,
		BeforeBalance: beforeBalance,
		AfterBalance:  afterBalance,
		BeforeFrozen:  beforeFrozen,
		AfterFrozen:   afterFrozen,
		Description:   description,
		Operator:      operator,
		ReferenceID:   referenceID,
		Remark:        remark,
	}

	if err := tx.Create(&record).Error; err != nil {
		return err
	}

	u.AvailableBalance = afterBalance
	u.FrozenBalance = afterFrozen
	return tx.Save(u).Error
}

func (u *User) AddBalance(tx *gorm.DB, amount float64, description string, operator string, referenceID string, remark JSON) error {
	return u.ProcessBalanceChange(tx, BalanceOperationAdd, amount, description, operator, referenceID, remark)
}

func (u *User) DeductBalance(tx *gorm.DB, amount float64, description string, operator string, referenceID string, remark JSON) error {
	return u.ProcessBalanceChange(tx, BalanceOperationDeduct, amount, description, operator, referenceID, remark)
}

func (u *User) FreezeBalance(tx *gorm.DB, amount float64, description string, operator string, referenceID string, remark JSON) error {
	return u.ProcessBalanceChange(tx, BalanceOperationFreeze, amount, description, operator, referenceID, remark)
}

func (u *User) UnfreezeBalance(tx *gorm.DB, amount float64, description string, operator string, referenceID string, remark JSON) error {
	return u.ProcessBalanceChange(tx, BalanceOperationUnfreeze, amount, description, operator, referenceID, remark)
}
