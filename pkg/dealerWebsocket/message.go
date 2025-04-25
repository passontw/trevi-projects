package dealerWebsocket

import (
	"encoding/json"
	"log"
	"time"
)

// 消息類型常量
const (
	// 系統消息類型
	MessageTypeHeartbeat      = "heartbeat"      // 心跳消息
	MessageTypeAuthentication = "authentication" // 認證消息
	MessageTypeAuthSuccess    = "auth_success"   // 認證成功
	MessageTypeAuthFailure    = "auth_failure"   // 認證失敗
	MessageTypeSystemNotice   = "system_notice"  // 系統通知
	MessageTypeError          = "error"          // 錯誤消息

	// 業務消息類型
	MessageTypeTicketPurchase = "ticket_purchase" // 票券購買消息
	MessageTypeTicketStatus   = "ticket_status"   // 票券狀態更新
	MessageTypeDrawResult     = "draw_result"     // 開獎結果
	MessageTypeAccountUpdate  = "account_update"  // 賬戶更新
)

// 基礎消息結構 - 使用不同名稱避免衝突
type BasicMessage struct {
	Type      string      `json:"type"`           // 消息類型
	Timestamp time.Time   `json:"timestamp"`      // 消息時間戳
	Data      interface{} `json:"data,omitempty"` // 消息數據
}

// 認證請求消息
type AuthMessage struct {
	Token string `json:"token"` // 認證令牌
}

// 認證回應消息
type AuthResponseMessage struct {
	Success bool   `json:"success"`           // 認證是否成功
	UserID  uint   `json:"user_id,omitempty"` // 用戶ID（認證成功時）
	Message string `json:"message,omitempty"` // 消息（認證失敗時的錯誤信息）
}

// 心跳消息 - 使用不同名稱避免衝突
type ServerHeartbeatMessage struct {
	ServerTime time.Time `json:"server_time"` // 服務器時間
}

// 錯誤消息
type ErrorMessage struct {
	Code    int    `json:"code"`    // 錯誤代碼
	Message string `json:"message"` // 錯誤信息
}

// 票券購買消息
type TicketPurchaseMessage struct {
	OrderID      string    `json:"order_id"`      // 訂單ID
	TicketCount  int       `json:"ticket_count"`  // 購買票數
	TotalAmount  float64   `json:"total_amount"`  // 總金額
	PurchaseTime time.Time `json:"purchase_time"` // 購買時間
}

// 票券狀態更新消息
type TicketStatusMessage struct {
	TicketID  string    `json:"ticket_id"`  // 票券ID
	Status    string    `json:"status"`     // 票券狀態
	UpdatedAt time.Time `json:"updated_at"` // 更新時間
}

// 開獎結果消息
type DrawResultMessage struct {
	DrawID         string      `json:"draw_id"`         // 開獎ID
	DrawTime       time.Time   `json:"draw_time"`       // 開獎時間
	WinningNumbers []int       `json:"winning_numbers"` // 中獎號碼
	PrizeInfo      interface{} `json:"prize_info"`      // 獎品信息
}

// 賬戶更新消息
type AccountUpdateMessage struct {
	UserID      uint      `json:"user_id"`     // 用戶ID
	Balance     float64   `json:"balance"`     // 賬戶餘額
	UpdatedAt   time.Time `json:"updated_at"`  // 更新時間
	UpdateType  string    `json:"update_type"` // 更新類型（例如：充值、消費、獎金）
	Amount      float64   `json:"amount"`      // 變動金額
	Description string    `json:"description"` // 描述
}

// 創建新消息
func NewMessage(messageType string, data interface{}) *BasicMessage {
	return &BasicMessage{
		Type:      messageType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// 解析通用消息
func ParseMessage(messageData []byte) (*BasicMessage, error) {
	var message BasicMessage
	err := json.Unmarshal(messageData, &message)
	if err != nil {
		log.Printf("Dealer WebSocket Message: Failed to parse message: %v", err)
		return nil, err
	}
	return &message, nil
}

// 創建心跳回應消息
func NewHeartbeatMessage() *BasicMessage {
	heartbeat := ServerHeartbeatMessage{
		ServerTime: time.Now(),
	}
	return NewMessage(MessageTypeHeartbeat, heartbeat)
}

// 創建認證成功消息
func NewAuthSuccessMessage(userID uint) *BasicMessage {
	authResponse := AuthResponseMessage{
		Success: true,
		UserID:  userID,
	}
	return NewMessage(MessageTypeAuthSuccess, authResponse)
}

// 創建認證失敗消息
func NewAuthFailureMessage(message string) *BasicMessage {
	authResponse := AuthResponseMessage{
		Success: false,
		Message: message,
	}
	return NewMessage(MessageTypeAuthFailure, authResponse)
}

// 創建錯誤消息
func NewErrorMessage(code int, message string) *BasicMessage {
	errorData := ErrorMessage{
		Code:    code,
		Message: message,
	}
	return NewMessage(MessageTypeError, errorData)
}

// 將消息轉換為JSON
func (m *BasicMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
