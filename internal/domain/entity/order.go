package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"g38_lottery_servic/pkg/utils"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type OrderType string
type OrderStatus string

const (
	OrderTypeSlotGame OrderType = "SLOT_GAME" // 老虎機遊戲
	OrderTypeDeposit  OrderType = "DEPOSIT"   // 存款
	OrderTypeWithdraw OrderType = "WITHDRAW"  // 提款
	OrderTypeBonus    OrderType = "BONUS"     // 獎金

	OrderPrefixSlotGame              = "SLOT"      // 老虎機遊戲
	OrderPrefixDeposit               = "DEP"       // 存款
	OrderPrefixWithdraw              = "WD"        // 提款
	OrderPrefixBonus                 = "BONUS"     // 獎金
	OrderPrefixDefault               = "ORD"       // 預設前綴
	OrderStatusPending   OrderStatus = "PENDING"   // 待處理
	OrderStatusCompleted OrderStatus = "COMPLETED" // 已完成
	OrderStatusCancelled OrderStatus = "CANCELLED" // 已取消
	OrderStatusFailed    OrderStatus = "FAILED"    // 失敗
)

type CreateOrderParams struct {
	UserID           int             `json:"user_id"`
	Type             OrderType       `json:"type"`
	BetAmount        float64         `json:"bet_amount"`
	WinAmount        float64         `json:"win_amount"`
	GameResult       json.RawMessage `json:"game_result"`
	BalanceRecordIDs []int64         `json:"balance_record_ids,omitempty"`
}

func (p *CreateOrderParams) Validate() error {
	if p.UserID <= 0 {
		return errors.New("invalid user ID")
	}

	if !p.Type.IsValid() {
		return errors.New("invalid order type")
	}

	if p.BetAmount <= 0 {
		return errors.New("bet amount must be positive")
	}

	if p.WinAmount < 0 {
		return errors.New("win amount cannot be negative")
	}

	if len(p.GameResult) == 0 {
		return errors.New("game result is required")
	}

	return nil
}

func (o *Order) getOrderPrefix() string {
	switch o.Type {
	case OrderTypeSlotGame:
		return OrderPrefixSlotGame
	case OrderTypeDeposit:
		return OrderPrefixDeposit
	case OrderTypeWithdraw:
		return OrderPrefixWithdraw
	case OrderTypeBonus:
		return OrderPrefixBonus
	default:
		return OrderPrefixDefault
	}
}

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusCompleted, OrderStatusCancelled, OrderStatusFailed:
		return true
	default:
		return false
	}
}

func (s OrderStatus) String() string {
	return string(s)
}

func (t OrderType) IsValid() bool {
	switch t {
	case OrderTypeSlotGame:
		return true
	default:
		return false
	}
}

func (t OrderType) String() string {
	return string(t)
}

var OrderStatusTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending: {
		OrderStatusCompleted,
		OrderStatusCancelled,
		OrderStatusFailed,
	},
	OrderStatusCompleted: {}, // 最終狀態
	OrderStatusCancelled: {}, // 最終狀態
	OrderStatusFailed:    {}, // 最終狀態
}

func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	if !s.IsValid() || !target.IsValid() {
		return false
	}

	allowedTransitions, exists := OrderStatusTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == target {
			return true
		}
	}
	return false
}

type Order struct {
	ID               int64           `gorm:"primaryKey;column:id" json:"id"`
	OrderID          string          `gorm:"column:order_id;uniqueIndex" json:"order_id"`
	UserID           int             `gorm:"column:user_id;not null" json:"user_id"`
	Type             OrderType       `gorm:"column:type;not null" json:"type"`
	Status           OrderStatus     `gorm:"column:status;not null" json:"status"`
	BetAmount        float64         `gorm:"column:bet_amount;not null" json:"bet_amount"`
	WinAmount        float64         `gorm:"column:win_amount;not null" json:"win_amount"`
	GameResult       json.RawMessage `gorm:"column:game_result;type:jsonb;not null" json:"game_result"`
	BalanceRecordIDs pq.Int64Array   `gorm:"column:balance_record_ids;type:integer[]" json:"balance_record_ids,omitempty"`
	CreatedAt        time.Time       `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;not null" json:"updated_at"`
	CompletedAt      *time.Time      `gorm:"column:completed_at" json:"completed_at,omitempty"`
	Remark           json.RawMessage `gorm:"column:remark;type:jsonb" json:"remark,omitempty"`

	User           *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	BalanceRecords []BalanceRecord `gorm:"many2many:order_balance_records;" json:"balance_records,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	// 生成雪花算法 ID
	snowflakeID, err := utils.GetNextID()
	if err != nil {
		return err
	}
	o.ID = snowflakeID

	// 生成業務訂單號
	if o.OrderID == "" {
		o.OrderID = fmt.Sprintf("%s%s%s",
			o.getOrderPrefix(),
			time.Now().Format("20060102"),
			fmt.Sprintf("%04d", snowflakeID%10000))
	}

	return nil
}

func CreateOrder(tx *gorm.DB, params CreateOrderParams) (*Order, error) {
	// 驗證參數
	if err := params.Validate(); err != nil {
		return nil, err
	}

	order := &Order{
		UserID:           params.UserID,
		Type:             params.Type,
		Status:           OrderStatusPending,
		BetAmount:        params.BetAmount,
		WinAmount:        params.WinAmount,
		GameResult:       params.GameResult,
		BalanceRecordIDs: params.BalanceRecordIDs,
	}

	err := tx.Transaction(func(tx *gorm.DB) error {
		// 創建訂單
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// 如果有餘額記錄ID，更新訂單
		if len(params.BalanceRecordIDs) > 0 {
			if err := tx.Save(order).Error; err != nil {
				return fmt.Errorf("failed to update balance records: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (o *Order) GetByID(db *gorm.DB, id int64) error {
	return db.First(o, id).Error
}

func (o *Order) FindByOrderID(db *gorm.DB, orderID string) error {
	return db.Where("order_id = ?", orderID).First(o).Error
}
