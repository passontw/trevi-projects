package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	BalanceOperationAdd      = "ADD"
	BalanceOperationDeduct   = "DEDUCT"
	BalanceOperationFreeze   = "FREEZE"
	BalanceOperationUnfreeze = "UNFREEZE"
)

type JSON map[string]interface{}

type BalanceRecord struct {
	ID            int       `gorm:"primaryKey;column:id" json:"id"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	UserID        int       `gorm:"column:user_id;not null;index:idx_balance_records_user" json:"user_id"`
	Type          string    `gorm:"column:type;type:balance_operation_type;not null" json:"type"`
	Amount        float64   `gorm:"column:amount;type:decimal(20,2);not null" json:"amount"`
	BeforeBalance float64   `gorm:"column:before_balance;type:decimal(20,2);not null" json:"before_balance"`
	AfterBalance  float64   `gorm:"column:after_balance;type:decimal(20,2);not null" json:"after_balance"`
	BeforeFrozen  float64   `gorm:"column:before_frozen;type:decimal(20,2);not null" json:"before_frozen"`
	AfterFrozen   float64   `gorm:"column:after_frozen;type:decimal(20,2);not null" json:"after_frozen"`
	Description   string    `gorm:"column:description;type:text" json:"description"`
	Operator      string    `gorm:"column:operator;type:varchar(50)" json:"operator"`
	ReferenceID   string    `gorm:"column:reference_id;type:varchar(100);index:idx_balance_records_reference" json:"reference_id"`
	Remark        JSON      `gorm:"column:remark;type:jsonb" json:"remark"`
	User          *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (BalanceRecord) TableName() string {
	return "balance_records"
}

func (b *BalanceRecord) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now
	return nil
}

func (b *BalanceRecord) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = time.Now()
	return nil
}

func (b *BalanceRecord) Validate() error {
	switch b.Type {
	case BalanceOperationAdd, BalanceOperationDeduct, BalanceOperationFreeze, BalanceOperationUnfreeze:
	default:
		return errors.New("invalid balance operation type")
	}

	if b.Amount <= 0 {
		return errors.New("amount must be positive")
	}

	if b.UserID <= 0 {
		return errors.New("invalid user ID")
	}

	return nil
}
