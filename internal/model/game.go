package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 遊戲狀態常量
const (
	GameStateCreated       = "CREATED"        // 遊戲創建
	GameStateReady         = "READY"          // 遊戲準備中
	GameStateStarted       = "STARTED"        // 遊戲開始
	GameStateShowLuckyNums = "SHOW_LUCKYNUMS" // 顯示幸運號碼階段
	GameStateLuckySet      = "LUCKY_SET"      // 已設置幸運號碼
	GameStateDrawing       = "DRAWING"        // 抽球中
	GameStateExtraBall     = "EXTRA_BALL"     // 額外球階段
	GameStateJackpot       = "JACKPOT"        // JP遊戲階段
	GameStateEnded         = "ENDED"          // 遊戲結束
)

// Game 遊戲模型，對應 games 資料表
type Game struct {
	ID               string     `gorm:"primarykey;type:varchar(36)" json:"id"`
	State            string     `gorm:"type:varchar(20);not null;index" json:"state"`
	StartTime        time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;index" json:"start_time"`
	EndTime          *time.Time `gorm:"index" json:"end_time,omitempty"`
	HasJackpot       bool       `gorm:"type:tinyint(1);not null;default:0" json:"has_jackpot"`
	ExtraBallCount   int        `gorm:"not null;default:0" json:"extra_ball_count"`
	StateStartTime   *time.Time `gorm:"index;default:null" json:"state_start_time,omitempty"`
	MaxTimeout       int        `gorm:"not null;default:60" json:"max_timeout"`
	LuckyNumbersJSON *string    `gorm:"type:json;column:lucky_numbers_json" json:"lucky_numbers_json,omitempty"`
	DrawnBallsJSON   *string    `gorm:"type:json;column:drawn_balls_json" json:"drawn_balls_json,omitempty"`
	ExtraBallsJSON   *string    `gorm:"type:json;column:extra_balls_json" json:"extra_balls_json,omitempty"`
	JackpotInfoJSON  *string    `gorm:"type:json;column:jackpot_info_json" json:"jackpot_info_json,omitempty"`
	CreatedAt        time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// 關聯模型
	LuckyNumbers []LuckyNumber `gorm:"foreignKey:GameID" json:"lucky_numbers,omitempty"`
	DrawnBalls   []DrawnBall   `gorm:"foreignKey:GameID" json:"drawn_balls,omitempty"`
	ExtraBalls   []ExtraBall   `gorm:"foreignKey:GameID" json:"extra_balls,omitempty"`
	JPGames      []JPGame      `gorm:"foreignKey:GameID" json:"jp_games,omitempty"`
	Bets         []Bet         `gorm:"foreignKey:GameID" json:"bets,omitempty"`
}

// TableName 指定資料表名稱
func (Game) TableName() string {
	return "games"
}

// LuckyNumber 幸運號碼模型
type LuckyNumber struct {
	ID        uint      `gorm:"primarykey;autoIncrement" json:"id"`
	GameID    string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	Number    int       `gorm:"not null" json:"number"`
	Sequence  int       `gorm:"not null" json:"sequence"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	// 關聯
	Game *Game `gorm:"foreignKey:GameID" json:"-"`
}

// TableName 指定資料表名稱
func (LuckyNumber) TableName() string {
	return "lucky_numbers"
}

// DrawnBall 開獎球模型
type DrawnBall struct {
	ID          uint      `gorm:"primarykey;autoIncrement" json:"id"`
	GameID      string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	Number      int       `gorm:"not null" json:"number"`
	Sequence    int       `gorm:"not null;index" json:"sequence"`
	DrawnTime   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"drawn_time"`
	IsExtraBall bool      `gorm:"type:tinyint(1);not null;default:0" json:"is_extra_ball"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	// 關聯
	Game *Game `gorm:"foreignKey:GameID" json:"-"`
}

// TableName 指定資料表名稱
func (DrawnBall) TableName() string {
	return "drawn_balls"
}

// ExtraBall 額外球模型
type ExtraBall struct {
	ID        uint      `gorm:"primarykey;autoIncrement" json:"id"`
	GameID    string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	Number    int       `gorm:"not null" json:"number"`
	Side      string    `gorm:"type:varchar(10);not null" json:"side"` // LEFT或RIGHT
	DrawnTime time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"drawn_time"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	// 關聯
	Game *Game `gorm:"foreignKey:GameID" json:"-"`
}

// TableName 指定資料表名稱
func (ExtraBall) TableName() string {
	return "extra_balls"
}

// JPGame JP遊戲模型
type JPGame struct {
	ID            string     `gorm:"primarykey;type:varchar(36)" json:"id"`
	GameID        string     `gorm:"type:varchar(36);not null;index" json:"game_id"`
	StartTime     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"start_time"`
	EndTime       *time.Time `json:"end_time"`
	JackpotAmount int64      `gorm:"not null;default:0" json:"jackpot_amount"`
	Active        bool       `gorm:"type:tinyint(1);not null;default:0" json:"active"`
	CreatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// 關聯
	Game    *Game      `gorm:"foreignKey:GameID" json:"-"`
	JPBalls []JPBall   `gorm:"foreignKey:JPGameID" json:"jp_balls,omitempty"`
	Winners []JPWinner `gorm:"foreignKey:JPGameID" json:"winners,omitempty"`
}

// TableName 指定資料表名稱
func (JPGame) TableName() string {
	return "jp_games"
}

// JPBall JP球模型
type JPBall struct {
	ID        uint      `gorm:"primarykey;autoIncrement" json:"id"`
	JPGameID  string    `gorm:"type:varchar(36);not null;index" json:"jp_game_id"`
	Number    int       `gorm:"not null" json:"number"`
	Sequence  int       `gorm:"not null;index" json:"sequence"`
	DrawnTime time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"drawn_time"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	// 關聯
	JPGame *JPGame `gorm:"foreignKey:JPGameID" json:"-"`
}

// TableName 指定資料表名稱
func (JPBall) TableName() string {
	return "jp_balls"
}

// JPWinner JP獲勝者模型
type JPWinner struct {
	ID        uint      `gorm:"primarykey;autoIncrement" json:"id"`
	JPGameID  string    `gorm:"type:varchar(36);not null;index" json:"jp_game_id"`
	UserID    string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	CardID    string    `gorm:"type:varchar(50);not null;index" json:"card_id"`
	WinTime   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"win_time"`
	Amount    int64     `gorm:"not null" json:"amount"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	// 關聯
	JPGame *JPGame `gorm:"foreignKey:JPGameID" json:"-"`
}

// TableName 指定資料表名稱
func (JPWinner) TableName() string {
	return "jp_winners"
}

// Bet 投注記錄模型
type Bet struct {
	ID              uint      `gorm:"primarykey;autoIncrement" json:"id"`
	GameID          string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	UserID          string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	BetAmount       int       `gorm:"not null" json:"bet_amount"`
	SelectedNumbers string    `gorm:"type:varchar(300);not null" json:"selected_numbers"`
	BetTime         time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"bet_time"`
	WinAmount       int       `gorm:"default:0" json:"win_amount"`
	IsExtraBet      bool      `gorm:"type:tinyint(1);not null;default:0" json:"is_extra_bet"`
	ExtraSide       *string   `gorm:"type:varchar(10)" json:"extra_side"`
	Status          string    `gorm:"type:varchar(20);not null;default:'PENDING';index" json:"status"`
	CreatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// 關聯
	Game *Game `gorm:"foreignKey:GameID" json:"-"`
}

// TableName 指定資料表名稱
func (Bet) TableName() string {
	return "bets"
}

// GameResponse 遊戲響應模型
type GameResponse struct {
	ID             string     `json:"id"`
	State          string     `json:"state"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	HasJackpot     bool       `json:"has_jackpot"`
	ExtraBallCount int        `json:"extra_ball_count"`
}

// ToResponse 將完整遊戲模型轉換為響應模型
func (g *Game) ToResponse() GameResponse {
	return GameResponse{
		ID:             g.ID,
		State:          g.State,
		StartTime:      g.StartTime,
		EndTime:        g.EndTime,
		HasJackpot:     g.HasJackpot,
		ExtraBallCount: g.ExtraBallCount,
	}
}

// WebSocketResponse WebSocket響應模型
type WebSocketResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Type    string      `json:"type,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse 創建成功響應
func NewSuccessResponse(responseType string, message string, data interface{}) WebSocketResponse {
	return WebSocketResponse{
		Success: true,
		Message: message,
		Type:    responseType,
		Data:    data,
	}
}

// NewErrorResponse 創建錯誤響應
func NewErrorResponse(message string) WebSocketResponse {
	return WebSocketResponse{
		Success: false,
		Message: message,
		Type:    "ERROR",
	}
}

// NewGame 創建新遊戲
func NewGame() *Game {
	now := time.Now()
	stateStartTime := now

	return &Game{
		ID:             uuid.New().String(),
		State:          GameStateCreated,
		StartTime:      now,
		HasJackpot:     false,
		ExtraBallCount: 0,
		StateStartTime: &stateStartTime,
		MaxTimeout:     60,
	}
}

// BeforeCreate GORM 前置鉤子，確保ID已設置
func (g *Game) BeforeCreate(tx *gorm.DB) error {
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	return nil
}

// CreateGameStartResponse 創建遊戲開始響應
func CreateGameStartResponse(game *Game) WebSocketResponse {
	return NewSuccessResponse(
		"GAME_START_RESPONSE",
		"遊戲已成功創建",
		map[string]interface{}{
			"game_id":     game.ID,
			"state":       game.State,
			"has_jackpot": game.HasJackpot,
			"start_time":  game.StartTime,
		},
	)
}
