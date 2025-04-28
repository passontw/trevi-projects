package dealerWebsocket

import (
	"encoding/json"
	"log"
	"time"
)

// 消息類型常量
const (
	// 系統消息類型
	MessageTypeHeartbeat      = "HEARTBEAT"      // 心跳消息
	MessageTypeAuthentication = "authentication" // 認證消息
	MessageTypeAuthSuccess    = "auth_success"   // 認證成功
	MessageTypeAuthFailure    = "auth_failure"   // 認證失敗
	MessageTypeSystemNotice   = "system_notice"  // 系統通知
	MessageTypeError          = "error"          // 錯誤消息

	// 業務消息類型
	MessageTypeTicketPurchase = "ticket_purchase" // 票券購買消息
	MessageTypeTicketStatus   = "ticket_status"   // 票券狀態更新
	MessageTypeDrawResult     = "DRAW_RESULT"     // 開獎結果
	MessageTypeAccountUpdate  = "account_update"  // 賬戶更新

	// 遊戲命令類型
	MessageTypeGameStart          = "GAME_START"              // 遊戲開始命令
	MessageTypeGameResponse       = "GAME_START_RESPONSE"     // 遊戲開始回應
	MessageTypeShowLuckyNumbers   = "SHOW_LUCKY_NUMBERS"      // 顯示幸運號碼命令
	MessageTypeDrawBall           = "DRAW_BALL"               // 抽球命令
	MessageTypeDrawExtraBall      = "DRAW_EXTRA_BALL"         // 抽額外球命令
	MessageTypeChooseExtraBall    = "STATE_CHOOSE_EXTRA_BALL" // 選擇額外球狀態命令
	MessageTypeStartJPGame        = "START_JP_GAME"           // 開始JP遊戲命令
	MessageTypeDrawJPBall         = "DRAW_JP_BALL"            // 抽JP球命令
	MessageTypeBettingStarted     = "BETTING_STARTED"         // 投注開始命令
	MessageTypeBettingClosed      = "BETTING_CLOSED"          // 投注關閉命令
	MessageTypeStartExtraBetting  = "START_EXTRA_BETTING"     // 額外球投注開始命令
	MessageTypeFinishExtraBetting = "FINISH_EXTRA_BETTING"    // 額外球投注結束命令

	// 遊戲事件類型
	MessageTypeLuckyNumbersSet = "LUCKY_NUMBERS_SET" // 幸運號碼設置事件
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

// 遊戲開始命令消息
type GameStartMessage struct {
	// 沒有額外參數，僅使用基本的Type欄位
}

// 遊戲開始回應消息
type GameStartResponseMessage struct {
	GameID     string    `json:"game_id"`     // 遊戲ID
	State      string    `json:"state"`       // 遊戲狀態
	HasJackpot bool      `json:"has_jackpot"` // 是否有JP遊戲
	StartTime  time.Time `json:"start_time"`  // 開始時間
}

// WebSocket回應結構
type WebSocketResponse struct {
	Success bool        `json:"success"`           // 是否成功
	Message string      `json:"message,omitempty"` // 消息
	Type    string      `json:"type,omitempty"`    // 類型
	Data    interface{} `json:"data,omitempty"`    // 數據
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

// 創建遊戲開始回應消息
func NewGameStartResponseMessage(gameID string, state string, hasJackpot bool, startTime time.Time) *BasicMessage {
	responseData := GameStartResponseMessage{
		GameID:     gameID,
		State:      state,
		HasJackpot: hasJackpot,
		StartTime:  startTime,
	}
	return NewMessage(MessageTypeGameResponse, responseData)
}

// 創建成功的WebSocket回應
func NewSuccessResponse(responseType string, message string, data interface{}) WebSocketResponse {
	return WebSocketResponse{
		Success: true,
		Message: message,
		Type:    responseType,
		Data:    data,
	}
}

// 創建錯誤的WebSocket回應
func NewErrorResponse(message string) WebSocketResponse {
	return WebSocketResponse{
		Success: false,
		Message: message,
		Type:    "ERROR",
	}
}

// 將消息轉換為JSON
func (m *BasicMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// 將WebSocket回應轉換為JSON
func (r *WebSocketResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
