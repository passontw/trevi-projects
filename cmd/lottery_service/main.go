package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

// 遊戲階段定義
type GameStage string

const (
	// 主遊戲流程
	StagePreparation       GameStage = "PREPARATION"
	StageNewRound          GameStage = "NEW_ROUND"
	StageCardPurchaseOpen  GameStage = "CARD_PURCHASE_OPEN"
	StageCardPurchaseClose GameStage = "CARD_PURCHASE_CLOSE"
	StageDrawingStart      GameStage = "DRAWING_START"
	StageDrawingClose      GameStage = "DRAWING_CLOSE"

	// 額外球流程
	StageExtraBallPrepare                 GameStage = "EXTRA_BALL_PREPARE"
	StageExtraBallSideSelectBettingStart  GameStage = "EXTRA_BALL_SIDE_SELECT_BETTING_START"
	StageExtraBallSideSelectBettingClosed GameStage = "EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED"
	StageExtraBallWaitClaim               GameStage = "EXTRA_BALL_WAIT_CLAIM"
	StageExtraBallDrawingStart            GameStage = "EXTRA_BALL_DRAWING_START"
	StageExtraBallDrawingClose            GameStage = "EXTRA_BALL_DRAWING_CLOSE"

	// 派彩與JP流程
	StagePayoutSettlement        GameStage = "PAYOUT_SETTLEMENT"
	StageJackpotStart            GameStage = "JACKPOT_START"
	StageJackpotDrawingStart     GameStage = "JACKPOT_DRAWING_START"
	StageJackpotDrawingClosed    GameStage = "JACKPOT_DRAWING_CLOSED"
	StageJackpotSettlement       GameStage = "JACKPOT_SETTLEMENT"
	StageDrawingLuckyBallsStart  GameStage = "DRAWING_LUCKY_BALLS_START"
	StageDrawingLuckyBallsClosed GameStage = "DRAWING_LUCKY_BALLS_CLOSED"

	// 結束階段
	StageGameOver GameStage = "GAMEOVER"
)

// 階段顯示名稱
var StageDisplayNames = map[GameStage]string{
	StagePreparation:       "遊戲準備中",
	StageNewRound:          "新局開始",
	StageCardPurchaseOpen:  "開始購卡",
	StageCardPurchaseClose: "購卡結束",
	StageDrawingStart:      "開始抽球",
	StageDrawingClose:      "抽球結束",

	StageExtraBallPrepare:                 "準備額外球",
	StageExtraBallSideSelectBettingStart:  "額外球選邊開始",
	StageExtraBallSideSelectBettingClosed: "額外球選邊結束",
	StageExtraBallWaitClaim:               "等待額外球兌獎",
	StageExtraBallDrawingStart:            "額外球抽球開始",
	StageExtraBallDrawingClose:            "額外球抽球結束",

	StagePayoutSettlement:        "派彩結算",
	StageJackpotStart:            "JP 遊戲開始",
	StageJackpotDrawingStart:     "JP 抽球開始",
	StageJackpotDrawingClosed:    "JP 抽球結束",
	StageJackpotSettlement:       "JP 結算",
	StageDrawingLuckyBallsStart:  "幸運號碼抽球開始",
	StageDrawingLuckyBallsClosed: "幸運號碼抽球結束",

	StageGameOver: "遊戲結束",
}

// 命令類型
type CommandType string

const (
	// 管理命令
	CmdStartNewRound CommandType = "START_NEW_ROUND"
	CmdEndGame       CommandType = "END_GAME"
	CmdCancelGame    CommandType = "CANCEL_GAME"

	// 抽球命令
	CmdDrawBall        CommandType = "DRAW_BALL"
	CmdDrawExtraBall   CommandType = "DRAW_EXTRA_BALL"
	CmdDrawJackpotBall CommandType = "DRAW_JACKPOT_BALL"
	CmdDrawLuckyBall   CommandType = "DRAW_LUCKY_BALL"

	// 其他命令
	CmdForceAdvanceStage CommandType = "FORCE_ADVANCE_STAGE"
	CmdResetGame         CommandType = "RESET_GAME"
)

// 事件類型
type EventType string

const (
	// 狀態事件
	EventStageChanged     EventType = "STAGE_CHANGED"
	EventCountdownUpdated EventType = "COUNTDOWN_UPDATED"

	// 遊戲事件
	EventGameStarted   EventType = "GAME_STARTED"
	EventGameCancelled EventType = "GAME_CANCELLED"
	EventGameEnded     EventType = "GAME_ENDED"

	// 球事件
	EventBallDrawn        EventType = "BALL_DRAWN"
	EventExtraBallDrawn   EventType = "EXTRA_BALL_DRAWN"
	EventJackpotBallDrawn EventType = "JACKPOT_BALL_DRAWN"
	EventLuckyBallDrawn   EventType = "LUCKY_BALL_DRAWN"

	// 其他事件
	EventError EventType = "ERROR"
)

// 消息類型
type MessageType string

const (
	MsgTypeCommand   MessageType = "COMMAND"
	MsgTypeEvent     MessageType = "EVENT"
	MsgTypeGameState MessageType = "GAME_STATE"
	MsgTypeError     MessageType = "ERROR"
)

// WebSocket 消息
type Message struct {
	Type      MessageType     `json:"type"`
	Command   CommandType     `json:"command,omitempty"`
	Event     EventType       `json:"event,omitempty"`
	Stage     GameStage       `json:"stage,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// 球數據
type Ball struct {
	ID         string    `json:"id"`
	Number     int       `json:"number"`
	Type       string    `json:"type"` // "normal", "extra", "jackpot", "lucky"
	IsLastBall bool      `json:"is_last_ball"`
	DrawnTime  time.Time `json:"drawn_time"`
}

// 階段配置
type StageConfig struct {
	AutoAdvanceTimeout time.Duration // 自動進階時間 (-1 表示無限，需人工控制)
	RequireDealer      bool          // 需要荷官確認
	RequireGame        bool          // 需要遊戲端確認
	AllowBallDraw      bool          // 允許抽球
	MaxBalls           int           // 最大抽球數量 (若適用)
	Description        string        // 階段描述
}

// 自然階段轉換映射
var naturalStageTransition = map[GameStage]GameStage{
	StagePreparation:       StageNewRound,
	StageNewRound:          StageCardPurchaseOpen,
	StageCardPurchaseOpen:  StageCardPurchaseClose,
	StageCardPurchaseClose: StageDrawingStart,
	StageDrawingStart:      StageDrawingClose,
	StageDrawingClose:      StageExtraBallPrepare,

	StageExtraBallPrepare:                 StageExtraBallSideSelectBettingStart,
	StageExtraBallSideSelectBettingStart:  StageExtraBallSideSelectBettingClosed,
	StageExtraBallSideSelectBettingClosed: StageExtraBallDrawingStart,
	StageExtraBallDrawingStart:            StageExtraBallDrawingClose,
	StageExtraBallDrawingClose:            StagePayoutSettlement,

	StagePayoutSettlement: StageJackpotStart, // 有JP時
	// StagePayoutSettlement:              StageGameOver,  // 無JP時

	StageJackpotStart:            StageJackpotDrawingStart,
	StageJackpotDrawingStart:     StageJackpotDrawingClosed,
	StageJackpotDrawingClosed:    StageJackpotSettlement,
	StageJackpotSettlement:       StageDrawingLuckyBallsStart,
	StageDrawingLuckyBallsStart:  StageDrawingLuckyBallsClosed,
	StageDrawingLuckyBallsClosed: StageGameOver,

	StageGameOver: StagePreparation,
}

// 階段配置映射
var stageConfigs = map[GameStage]StageConfig{
	StagePreparation: {
		AutoAdvanceTimeout: -1, // 等待人工干預
		RequireDealer:      true,
		RequireGame:        false,
		Description:        "遊戲準備中",
	},
	StageNewRound: {
		AutoAdvanceTimeout: 2 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "新局開始",
	},
	StageCardPurchaseOpen: {
		AutoAdvanceTimeout: 12 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "開始購卡",
	},
	StageCardPurchaseClose: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "購卡結束",
	},
	StageDrawingStart: {
		AutoAdvanceTimeout: -1, // 由荷官控制
		RequireDealer:      true,
		RequireGame:        false,
		AllowBallDraw:      true,
		MaxBalls:           75,
		Description:        "開始抽球",
	},
	StageDrawingClose: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "抽球結束",
	},
	StageExtraBallPrepare: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "準備額外球",
	},
	StageExtraBallSideSelectBettingStart: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "額外球選邊開始",
	},
	StageExtraBallSideSelectBettingClosed: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "額外球選邊結束",
	},
	StageExtraBallDrawingStart: {
		AutoAdvanceTimeout: -1, // 由荷官控制
		RequireDealer:      true,
		RequireGame:        false,
		AllowBallDraw:      true,
		MaxBalls:           3,
		Description:        "額外球抽球開始",
	},
	StageExtraBallDrawingClose: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "額外球抽球結束",
	},
	StagePayoutSettlement: {
		AutoAdvanceTimeout: 3 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "派彩結算",
	},
	StageJackpotStart: {
		AutoAdvanceTimeout: 3 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "JP 遊戲開始",
	},
	StageJackpotDrawingStart: {
		AutoAdvanceTimeout: -1, // 由遊戲端控制
		RequireDealer:      true,
		RequireGame:        true,
		AllowBallDraw:      true,
		Description:        "JP 抽球開始",
	},
	StageJackpotDrawingClosed: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "JP 抽球結束",
	},
	StageJackpotSettlement: {
		AutoAdvanceTimeout: 3 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "JP 結算",
	},
	StageDrawingLuckyBallsStart: {
		AutoAdvanceTimeout: -1, // 由荷官控制
		RequireDealer:      true,
		RequireGame:        false,
		AllowBallDraw:      true,
		MaxBalls:           7,
		Description:        "幸運號碼抽球開始",
	},
	StageDrawingLuckyBallsClosed: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "幸運號碼抽球結束",
	},
	StageGameOver: {
		AutoAdvanceTimeout: 1 * time.Second,
		RequireDealer:      false,
		RequireGame:        false,
		Description:        "遊戲結束",
	},
}

// 賓果遊戲數據
type BingoGameData struct {
	GameID         string    `json:"game_id"`
	CurrentStage   GameStage `json:"current_stage"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time,omitempty"`
	RegularBalls   []Ball    `json:"regular_balls"`
	ExtraBalls     []Ball    `json:"extra_balls"`
	JackpotBalls   []Ball    `json:"jackpot_balls"`
	LuckyBalls     []Ball    `json:"lucky_balls"`
	HasJackpot     bool      `json:"has_jackpot"`
	JackpotWinners int       `json:"jackpot_winners"`
	ExtraBallSide  string    `json:"extra_ball_side"` // "left" 或 "right"
	Cancelled      bool      `json:"cancelled"`
	CancelReason   string    `json:"cancel_reason,omitempty"`
}

// 賓果開獎服務
type BingoDrawService struct {
	mu             sync.RWMutex
	currentGame    *BingoGameData
	currentStage   GameStage
	stageTimers    map[GameStage]*time.Timer
	dealerClients  map[*websocket.Conn]bool
	gameClients    map[*websocket.Conn]bool
	dealerApproved bool
	gameApproved   bool
	db             *sql.DB
	redis          *redis.Client
	ctx            context.Context
	shutdown       chan struct{}
}

// 創建新的賓果開獎服務
func NewBingoDrawService(db *sql.DB, redisClient *redis.Client) *BingoDrawService {
	ctx := context.Background()

	service := &BingoDrawService{
		currentStage:  StagePreparation,
		stageTimers:   make(map[GameStage]*time.Timer),
		dealerClients: make(map[*websocket.Conn]bool),
		gameClients:   make(map[*websocket.Conn]bool),
		db:            db,
		redis:         redisClient,
		ctx:           ctx,
		shutdown:      make(chan struct{}),
	}

	// 初始化服務
	go service.initialize()

	return service
}

// 初始化服務
func (bds *BingoDrawService) initialize() {
	// 檢查是否有未完成的遊戲
	if gameData, err := bds.loadGameFromRedis(); err == nil && gameData != nil {
		// 恢復遊戲狀態
		bds.mu.Lock()
		bds.currentGame = gameData
		bds.currentStage = gameData.CurrentStage
		bds.mu.Unlock()

		log.Printf("恢復上次遊戲狀態，當前階段: %s", gameData.CurrentStage)
		bds.broadcastGameState()
	} else {
		// 沒有未完成的遊戲，初始化為準備階段
		bds.mu.Lock()
		bds.currentStage = StagePreparation
		bds.mu.Unlock()

		// 檢查數據庫中是否有幸運號碼，沒有則生成
		bds.ensureLuckyBallsExist()
	}
}

// 確保幸運號碼存在
func (bds *BingoDrawService) ensureLuckyBallsExist() {
	// 檢查 TiDB 中是否有幸運號碼
	var count int
	err := bds.db.QueryRowContext(bds.ctx, "SELECT COUNT(*) FROM lucky_balls WHERE active = 1").Scan(&count)
	if err != nil || count == 0 {
		// 生成新的幸運號碼
		luckyBalls := bds.generateLuckyBalls(7) // 生成7個幸運號碼

		// 將幸運號碼寫入數據庫
		_, err := bds.db.ExecContext(bds.ctx, `
			INSERT INTO lucky_balls (
				draw_date, number1, number2, number3, number4, number5, number6, number7, active
			) VALUES (NOW(), ?, ?, ?, ?, ?, ?, ?, 1)`,
			luckyBalls[0], luckyBalls[1], luckyBalls[2], luckyBalls[3],
			luckyBalls[4], luckyBalls[5], luckyBalls[6],
		)
		if err != nil {
			log.Printf("插入幸運號碼錯誤: %v", err)
			return
		}

		log.Printf("成功生成並存儲7個幸運號碼")
	}
}

// 生成隨機幸運號碼
func (bds *BingoDrawService) generateLuckyBalls(count int) []int {
	luckyBalls := make([]int, 0, count)
	used := make(map[int]bool)

	rand.Seed(time.Now().UnixNano())

	for len(luckyBalls) < count {
		num := rand.Intn(75) + 1
		if !used[num] {
			used[num] = true
			luckyBalls = append(luckyBalls, num)
		}
	}

	return luckyBalls
}

// 從 Redis 讀取遊戲數據
func (bds *BingoDrawService) loadGameFromRedis() (*BingoGameData, error) {
	gameDataJSON, err := bds.redis.Get(bds.ctx, "bingo:current_game").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 沒有數據
		}
		return nil, err
	}

	var gameData BingoGameData
	if err := json.Unmarshal([]byte(gameDataJSON), &gameData); err != nil {
		return nil, err
	}

	return &gameData, nil
}

// 保存遊戲數據到 Redis
func (bds *BingoDrawService) saveGameToRedis() error {
	bds.mu.RLock()
	defer bds.mu.RUnlock()

	if bds.currentGame == nil {
		return nil
	}

	gameDataJSON, err := json.Marshal(bds.currentGame)
	if err != nil {
		return err
	}

	return bds.redis.Set(bds.ctx, "bingo:current_game", gameDataJSON, 24*time.Hour).Err()
}

// 註冊客戶端
func (bds *BingoDrawService) RegisterClient(conn *websocket.Conn, clientType string) {
	bds.mu.Lock()
	defer bds.mu.Unlock()

	if clientType == "dealer" {
		bds.dealerClients[conn] = true
	} else {
		bds.gameClients[conn] = true
	}

	// 向客戶端發送當前遊戲狀態
	bds.sendGameState(conn)
}

// 取消註冊客戶端
func (bds *BingoDrawService) UnregisterClient(conn *websocket.Conn, clientType string) {
	bds.mu.Lock()
	defer bds.mu.Unlock()

	if clientType == "dealer" {
		delete(bds.dealerClients, conn)
	} else {
		delete(bds.gameClients, conn)
	}
}

// 處理命令
func (bds *BingoDrawService) HandleCommand(msg Message, conn *websocket.Conn, clientType string) {
	bds.mu.Lock()
	defer bds.mu.Unlock()

	switch msg.Command {
	case CmdStartNewRound:
		bds.handleStartNewRound()
	case CmdDrawBall:
		bds.handleDrawBall(msg.Data)
	case CmdDrawExtraBall:
		bds.handleDrawExtraBall(msg.Data)
	case CmdDrawJackpotBall:
		bds.handleDrawJackpotBall(msg.Data)
	case CmdDrawLuckyBall:
		bds.handleDrawLuckyBall(msg.Data)
	case CmdCancelGame:
		bds.handleCancelGame(msg.Data)
	case CmdForceAdvanceStage:
		bds.handleForceAdvanceStage()
	case CmdResetGame:
		bds.handleResetGame()
	}
}

// 處理遊戲開始命令
func (bds *BingoDrawService) handleStartNewRound() {
	if bds.currentStage != StagePreparation {
		bds.sendError("遊戲不在準備階段，無法開始新局")
		return
	}

	// 創建新的遊戲數據
	bds.currentGame = &BingoGameData{
		GameID:       fmt.Sprintf("BG-%d", time.Now().Unix()),
		CurrentStage: StageNewRound,
		StartTime:    time.Now(),
		RegularBalls: make([]Ball, 0),
		ExtraBalls:   make([]Ball, 0),
		JackpotBalls: make([]Ball, 0),
		LuckyBalls:   make([]Ball, 0),
		HasJackpot:   true, // 預設有 Jackpot，可根據實際情況調整
	}

	// 更新階段並廣播
	bds.advanceToStage(StageNewRound)
}

// 處理抽球命令
func (bds *BingoDrawService) handleDrawBall(data json.RawMessage) {
	if bds.currentStage != StageDrawingStart {
		bds.sendError("遊戲不在抽球階段，無法抽球")
		return
	}

	var ballData Ball
	if err := json.Unmarshal(data, &ballData); err != nil {
		bds.sendError("解析球數據錯誤")
		return
	}

	// 檢查號碼是否有效
	if ballData.Number < 1 || ballData.Number > 75 {
		bds.sendError("無效的球號")
		return
	}

	// 檢查是否已經抽過
	for _, ball := range bds.currentGame.RegularBalls {
		if ball.Number == ballData.Number {
			bds.sendError("此球已經被抽過")
			return
		}
	}

	// 設置球類型和抽出時間
	ballData.Type = "normal"
	ballData.DrawnTime = time.Now()
	ballData.ID = fmt.Sprintf("%s-R-%d", bds.currentGame.GameID, len(bds.currentGame.RegularBalls)+1)

	// 添加到抽球列表
	bds.currentGame.RegularBalls = append(bds.currentGame.RegularBalls, ballData)

	// 廣播球信息
	bds.broadcastBallDrawn(ballData)

	// 如果是最後一顆球，進入抽球結束階段
	if ballData.IsLastBall {
		bds.advanceToStage(StageDrawingClose)
	}
}

// 處理抽額外球命令
func (bds *BingoDrawService) handleDrawExtraBall(data json.RawMessage) {
	if bds.currentStage != StageExtraBallDrawingStart {
		bds.sendError("遊戲不在額外球抽球階段，無法抽額外球")
		return
	}

	var ballData Ball
	if err := json.Unmarshal(data, &ballData); err != nil {
		bds.sendError("解析球數據錯誤")
		return
	}

	// 檢查號碼是否有效
	if ballData.Number < 1 || ballData.Number > 75 {
		bds.sendError("無效的球號")
		return
	}

	// 檢查是否已經抽過 (在常規球和額外球中)
	for _, ball := range bds.currentGame.RegularBalls {
		if ball.Number == ballData.Number {
			bds.sendError("此球已經被抽過")
			return
		}
	}

	for _, ball := range bds.currentGame.ExtraBalls {
		if ball.Number == ballData.Number {
			bds.sendError("此球已經被抽過")
			return
		}
	}

	// 設置球類型和抽出時間
	ballData.Type = "extra"
	ballData.DrawnTime = time.Now()
	ballData.ID = fmt.Sprintf("%s-E-%d", bds.currentGame.GameID, len(bds.currentGame.ExtraBalls)+1)

	// 添加到額外球列表
	bds.currentGame.ExtraBalls = append(bds.currentGame.ExtraBalls, ballData)

	// 廣播球信息
	bds.broadcastExtraBallDrawn(ballData)

	// 如果是最後一顆球，進入額外球抽球結束階段
	if ballData.IsLastBall {
		bds.advanceToStage(StageExtraBallDrawingClose)
	}
}

// 處理抽JP球命令
func (bds *BingoDrawService) handleDrawJackpotBall(data json.RawMessage) {
	if bds.currentStage != StageJackpotDrawingStart {
		bds.sendError("遊戲不在JP抽球階段，無法抽JP球")
		return
	}

	var ballData Ball
	if err := json.Unmarshal(data, &ballData); err != nil {
		bds.sendError("解析球數據錯誤")
		return
	}

	// 檢查號碼是否有效
	if ballData.Number < 1 || ballData.Number > 75 {
		bds.sendError("無效的球號")
		return
	}

	// 檢查是否已經抽過 (僅在JP球中檢查)
	for _, ball := range bds.currentGame.JackpotBalls {
		if ball.Number == ballData.Number {
			bds.sendError("此JP球已經被抽過")
			return
		}
	}

	// 設置球類型和抽出時間
	ballData.Type = "jackpot"
	ballData.DrawnTime = time.Now()
	ballData.ID = fmt.Sprintf("%s-J-%d", bds.currentGame.GameID, len(bds.currentGame.JackpotBalls)+1)

	// 添加到JP球列表
	bds.currentGame.JackpotBalls = append(bds.currentGame.JackpotBalls, ballData)

	// 廣播球信息
	bds.broadcastJackpotBallDrawn(ballData)

	// 如果是最後一顆球，進入JP抽球結束階段
	if ballData.IsLastBall {
		bds.advanceToStage(StageJackpotDrawingClosed)
	}
}

// 處理抽幸運球命令
func (bds *BingoDrawService) handleDrawLuckyBall(data json.RawMessage) {
	if bds.currentStage != StageDrawingLuckyBallsStart {
		bds.sendError("遊戲不在幸運號碼抽球階段，無法抽幸運球")
		return
	}

	var ballData Ball
	if err := json.Unmarshal(data, &ballData); err != nil {
		bds.sendError("解析球數據錯誤")
		return
	}

	// 檢查號碼是否有效
	if ballData.Number < 1 || ballData.Number > 75 {
		bds.sendError("無效的球號")
		return
	}

	// 檢查是否已經抽過 (在幸運球中)
	for _, ball := range bds.currentGame.LuckyBalls {
		if ball.Number == ballData.Number {
			bds.sendError("此幸運球已經被抽過")
			return
		}
	}

	// 設置球類型和抽出時間
	ballData.Type = "lucky"
	ballData.DrawnTime = time.Now()
	ballData.ID = fmt.Sprintf("%s-L-%d", bds.currentGame.GameID, len(bds.currentGame.LuckyBalls)+1)

	// 添加到幸運球列表
	bds.currentGame.LuckyBalls = append(bds.currentGame.LuckyBalls, ballData)

	// 廣播球信息
	bds.broadcastLuckyBallDrawn(ballData)

	// 如果是最後一顆球，進入幸運號碼抽球結束階段
	if ballData.IsLastBall || len(bds.currentGame.LuckyBalls) >= 7 {
		bds.advanceToStage(StageDrawingLuckyBallsClosed)
	}
}

// 處理取消遊戲命令
func (bds *BingoDrawService) handleCancelGame(data json.RawMessage) {
	var cancelData struct {
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal(data, &cancelData); err != nil {
		bds.sendError("解析取消原因錯誤")
		return
	}

	// 確保遊戲存在
	if bds.currentGame == nil {
		bds.sendError("沒有進行中的遊戲")
		return
	}

	// 標記遊戲為已取消
	bds.currentGame.Cancelled = true
	bds.currentGame.CancelReason = cancelData.Reason
	bds.currentGame.EndTime = time.Now()

	// 保存遊戲數據到 TiDB
	bds.saveGameToTiDB()

	// 清除 Redis 中的遊戲數據
	bds.redis.Del(bds.ctx, "bingo:current_game")

	// 廣播遊戲取消事件
	bds.broadcastGameCancelled(cancelData.Reason)

	// 重置狀態為準備階段
	bds.currentStage = StagePreparation
	bds.currentGame = nil
	bds.broadcastGameState()
}

// 處理強制進階命令
func (bds *BingoDrawService) handleForceAdvanceStage() {
	nextStage := bds.getNextStage(bds.currentStage)
	bds.advanceToStage(nextStage)
}

// 處理重置遊戲命令
func (bds *BingoDrawService) handleResetGame() {
	// 清除所有計時器
	for _, timer := range bds.stageTimers {
		if timer != nil {
			timer.Stop()
		}
	}
	bds.stageTimers = make(map[GameStage]*time.Timer)

	// 清除 Redis 中的遊戲數據
	bds.redis.Del(bds.ctx, "bingo:current_game")

	// 重置狀態
	bds.currentStage = StagePreparation
	bds.currentGame = nil

	// 廣播新狀態
	bds.broadcastGameState()
}

// 進階到指定階段
func (bds *BingoDrawService) advanceToStage(newStage GameStage) {
	// 停止當前階段的計時器
	if timer, exists := bds.stageTimers[bds.currentStage]; exists && timer != nil {
		timer.Stop()
		delete(bds.stageTimers, bds.currentStage)
	}

	// 更新遊戲狀態
	bds.currentStage = newStage
	if bds.currentGame != nil {
		bds.currentGame.CurrentStage = newStage
	}

	// 執行階段特定邏輯
	switch newStage {
	case StageExtraBallSideSelectBettingStart:
		// 自動選邊
		bds.autoSelectExtraBallSide()
	case StageGameOver:
		// 遊戲結束，保存數據並清理
		if bds.currentGame != nil {
			bds.currentGame.EndTime = time.Now()
			bds.saveGameToTiDB()
			// 清除 Redis 中的遊戲數據
			bds.redis.Del(bds.ctx, "bingo:current_game")
		}
	}

	// 保存遊戲數據到 Redis (除非遊戲結束)
	if newStage != StageGameOver && bds.currentGame != nil {
		bds.saveGameToRedis()
	}

	// 廣播新狀態
	bds.broadcastGameState()

	// 設置自動進階計時器 (如果配置允許)
	config := stageConfigs[newStage]
	if config.AutoAdvanceTimeout > 0 {
		bds.stageTimers[newStage] = time.AfterFunc(config.AutoAdvanceTimeout, func() {
			bds.mu.Lock()
			defer bds.mu.Unlock()

			// 檢查是否還是同一階段 (防止舊計時器觸發)
			if bds.currentStage == newStage {
				nextStage := bds.getNextStage(newStage)
				log.Printf("階段 %s 超時，自動進階到 %s", newStage, nextStage)
				bds.advanceToStage(nextStage)
			}
		})
	}
}

// 自動選擇額外球邊
func (bds *BingoDrawService) autoSelectExtraBallSide() {
	// 隨機選擇左或右
	sides := []string{"left", "right"}
	rand.Seed(time.Now().UnixNano())
	selectedSide := sides[rand.Intn(len(sides))]

	bds.currentGame.ExtraBallSide = selectedSide

	// 向遊戲客戶端廣播選邊結果
	sideData, _ := json.Marshal(map[string]string{
		"side": selectedSide,
	})

	event := Message{
		Type:      MsgTypeEvent,
		Event:     "EXTRA_BALL_SIDE_SELECTED",
		Data:      sideData,
		Timestamp: time.Now().Unix(),
	}

	bds.broadcast(event)
}

// 獲取下一階段
func (bds *BingoDrawService) getNextStage(currentStage GameStage) GameStage {
	// 檢查是否有特殊轉換邏輯
	if currentStage == StagePayoutSettlement && (!bds.currentGame.HasJackpot || bds.currentGame.JackpotWinners > 0) {
		// 如果沒有JP遊戲或已經有JP獲勝者，直接進入遊戲結束階段
		return StageGameOver
	}

	// 檢查自然轉換映射
	if nextStage, exists := naturalStageTransition[currentStage]; exists {
		return nextStage
	}

	// 預設回到準備階段
	return StagePreparation
}

// 保存遊戲數據到 TiDB
func (bds *BingoDrawService) saveGameToTiDB() {
	if bds.currentGame == nil {
		return
	}

	// 開始事務
	tx, err := bds.db.BeginTx(bds.ctx, nil)
	if err != nil {
		log.Printf("開始事務錯誤: %v", err)
		return
	}

	// 保存遊戲基本信息
	gameStmt, err := tx.PrepareContext(bds.ctx, `
		INSERT INTO games (
			game_id, start_time, end_time, state, has_jackpot, 
			total_players, extra_ball_count, cancelled, cancel_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Printf("準備遊戲插入語句錯誤: %v", err)
		tx.Rollback()
		return
	}
	defer gameStmt.Close()

	_, err = gameStmt.ExecContext(
		bds.ctx,
		bds.currentGame.GameID,
		bds.currentGame.StartTime,
		bds.currentGame.EndTime,
		string(bds.currentGame.CurrentStage),
		bds.currentGame.HasJackpot,
		0,                               // total_players - 目前未實作玩家追蹤
		len(bds.currentGame.ExtraBalls), // extra_ball_count
		bds.currentGame.Cancelled,
		bds.currentGame.CancelReason,
	)
	if err != nil {
		log.Printf("插入遊戲數據錯誤: %v", err)
		tx.Rollback()
		return
	}

	// 保存球數據
	ballStmt, err := tx.PrepareContext(bds.ctx, `
		INSERT INTO drawn_balls (
			game_id, number, sequence, ball_type, is_last_ball, drawn_time
		) VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Printf("準備球插入語句錯誤: %v", err)
		tx.Rollback()
		return
	}
	defer ballStmt.Close()

	// 保存常規球
	for i, ball := range bds.currentGame.RegularBalls {
		_, err = ballStmt.ExecContext(
			bds.ctx,
			bds.currentGame.GameID,
			ball.Number,
			i+1,       // sequence
			"REGULAR", // ball_type
			ball.IsLastBall,
			ball.DrawnTime,
		)
		if err != nil {
			log.Printf("插入常規球數據錯誤: %v", err)
			tx.Rollback()
			return
		}
	}

	// 保存額外球
	for i, ball := range bds.currentGame.ExtraBalls {
		_, err = ballStmt.ExecContext(
			bds.ctx,
			bds.currentGame.GameID,
			ball.Number,
			i+1,     // sequence
			"EXTRA", // ball_type
			ball.IsLastBall,
			ball.DrawnTime,
		)
		if err != nil {
			log.Printf("插入額外球數據錯誤: %v", err)
			tx.Rollback()
			return
		}
	}

	// 保存JP球
	for i, ball := range bds.currentGame.JackpotBalls {
		_, err = ballStmt.ExecContext(
			bds.ctx,
			bds.currentGame.GameID,
			ball.Number,
			i+1,       // sequence
			"JACKPOT", // ball_type
			ball.IsLastBall,
			ball.DrawnTime,
		)
		if err != nil {
			log.Printf("插入JP球數據錯誤: %v", err)
			tx.Rollback()
			return
		}
	}

	// 保存幸運球
	if len(bds.currentGame.LuckyBalls) > 0 {
		// 先將舊的幸運號碼設為非活躍
		_, err = tx.ExecContext(bds.ctx, "UPDATE lucky_balls SET active = 0 WHERE active = 1")
		if err != nil {
			log.Printf("更新舊幸運號碼狀態錯誤: %v", err)
			tx.Rollback()
			return
		}

		// 確保有7個幸運號碼
		luckyNumbers := make([]int, 7)
		for i := 0; i < 7; i++ {
			if i < len(bds.currentGame.LuckyBalls) {
				luckyNumbers[i] = bds.currentGame.LuckyBalls[i].Number
			} else {
				// 如果不足7個，隨機生成其他號碼
				luckyNumbers[i] = rand.Intn(75) + 1
			}
		}

		// 插入新的幸運號碼
		_, err = tx.ExecContext(bds.ctx, `
			INSERT INTO lucky_balls (
				game_id, draw_date, number1, number2, number3, number4, number5, number6, number7, active
			) VALUES (?, NOW(), ?, ?, ?, ?, ?, ?, ?, 1)`,
			bds.currentGame.GameID,
			luckyNumbers[0], luckyNumbers[1], luckyNumbers[2], luckyNumbers[3],
			luckyNumbers[4], luckyNumbers[5], luckyNumbers[6],
		)
		if err != nil {
			log.Printf("插入幸運號碼錯誤: %v", err)
			tx.Rollback()
			return
		}
	}

	// 提交事務
	if err := tx.Commit(); err != nil {
		log.Printf("提交事務錯誤: %v", err)
		tx.Rollback()
		return
	}

	log.Printf("成功保存遊戲數據到 TiDB，遊戲ID: %s", bds.currentGame.GameID)
}

// 廣播球抽出事件
func (bds *BingoDrawService) broadcastBallDrawn(ball Ball) {
	ballData, _ := json.Marshal(ball)

	event := Message{
		Type:      MsgTypeEvent,
		Event:     EventBallDrawn,
		Data:      ballData,
		Timestamp: time.Now().Unix(),
	}

	bds.broadcast(event)
}

// 廣播額外球抽出事件
func (bds *BingoDrawService) broadcastExtraBallDrawn(ball Ball) {
	ballData, _ := json.Marshal(ball)

	event := Message{
		Type:      MsgTypeEvent,
		Event:     EventExtraBallDrawn,
		Data:      ballData,
		Timestamp: time.Now().Unix(),
	}

	bds.broadcast(event)
}

// 廣播JP球抽出事件
func (bds *BingoDrawService) broadcastJackpotBallDrawn(ball Ball) {
	ballData, _ := json.Marshal(ball)

	event := Message{
		Type:      MsgTypeEvent,
		Event:     EventJackpotBallDrawn,
		Data:      ballData,
		Timestamp: time.Now().Unix(),
	}

	bds.broadcast(event)
}

// 廣播幸運球抽出事件
func (bds *BingoDrawService) broadcastLuckyBallDrawn(ball Ball) {
	ballData, _ := json.Marshal(ball)

	event := Message{
		Type:      MsgTypeEvent,
		Event:     EventLuckyBallDrawn,
		Data:      ballData,
		Timestamp: time.Now().Unix(),
	}

	bds.broadcast(event)
}

// 廣播遊戲取消事件
func (bds *BingoDrawService) broadcastGameCancelled(reason string) {
	cancelData, _ := json.Marshal(map[string]string{
		"reason":  reason,
		"game_id": bds.currentGame.GameID,
	})

	event := Message{
		Type:      MsgTypeEvent,
		Event:     EventGameCancelled,
		Data:      cancelData,
		Timestamp: time.Now().Unix(),
	}

	bds.broadcast(event)
}

// 廣播遊戲狀態
func (bds *BingoDrawService) broadcastGameState() {
	for conn := range bds.dealerClients {
		bds.sendGameState(conn)
	}

	for conn := range bds.gameClients {
		bds.sendGameState(conn)
	}
}

// 發送遊戲狀態到單一客戶端
func (bds *BingoDrawService) sendGameState(conn *websocket.Conn) {
	var gameStateData map[string]interface{}

	if bds.currentGame != nil {
		// 遊戲進行中，發送完整狀態
		gameStateData = map[string]interface{}{
			"game_id":         bds.currentGame.GameID,
			"current_stage":   bds.currentStage,
			"stage_name":      StageDisplayNames[bds.currentStage],
			"regular_balls":   bds.currentGame.RegularBalls,
			"extra_balls":     bds.currentGame.ExtraBalls,
			"jackpot_balls":   bds.currentGame.JackpotBalls,
			"lucky_balls":     bds.currentGame.LuckyBalls,
			"has_jackpot":     bds.currentGame.HasJackpot,
			"extra_ball_side": bds.currentGame.ExtraBallSide,
		}
	} else {
		// 無遊戲，僅發送基本狀態
		gameStateData = map[string]interface{}{
			"current_stage": bds.currentStage,
			"stage_name":    StageDisplayNames[bds.currentStage],
		}
	}

	// 計算並包含倒數時間
	if timer, exists := bds.stageTimers[bds.currentStage]; exists && timer != nil {
		// 未來可增加計算剩餘時間的邏輯
		gameStateData["has_countdown"] = true
	} else {
		gameStateData["has_countdown"] = false
	}

	stateData, _ := json.Marshal(gameStateData)

	message := Message{
		Type:      MsgTypeGameState,
		Stage:     bds.currentStage,
		Data:      stateData,
		Timestamp: time.Now().Unix(),
	}

	messageData, _ := json.Marshal(message)

	if err := conn.WriteMessage(websocket.TextMessage, messageData); err != nil {
		log.Printf("發送遊戲狀態錯誤: %v", err)
	}
}

// 發送錯誤消息
func (bds *BingoDrawService) sendError(errMsg string) {
	errorData, _ := json.Marshal(map[string]string{
		"message": errMsg,
	})

	event := Message{
		Type:      MsgTypeError,
		Data:      errorData,
		Timestamp: time.Now().Unix(),
	}

	bds.broadcast(event)
	log.Printf("錯誤: %s", errMsg)
}

// 廣播消息
func (bds *BingoDrawService) broadcast(msg Message) {
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("序列化消息錯誤: %v", err)
		return
	}

	// 發送給荷官客戶端
	for conn := range bds.dealerClients {
		if err := conn.WriteMessage(websocket.TextMessage, msgData); err != nil {
			log.Printf("發送給荷官錯誤: %v", err)
			bds.UnregisterClient(conn, "dealer")
		}
	}

	// 發送給遊戲客戶端
	for conn := range bds.gameClients {
		if err := conn.WriteMessage(websocket.TextMessage, msgData); err != nil {
			log.Printf("發送給遊戲端錯誤: %v", err)
			bds.UnregisterClient(conn, "game")
		}
	}
}

func main() {
	// 配置 TiDB 連接
	db, err := sql.Open("mysql", "root:a12345678@tcp(127.0.0.1:4000)/g38_lottery?parseTime=true")
	if err != nil {
		log.Fatalf("連接 TiDB 失敗: %v", err)
	}
	defer db.Close()

	// 檢查數據庫連接
	if err := db.Ping(); err != nil {
		log.Fatalf("無法 ping TiDB: %v", err)
	}

	// 確保必要的表格存在
	if err := ensureTablesExist(db); err != nil {
		log.Fatalf("確保表格存在失敗: %v", err)
	}

	// 配置 Redis 連接
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "172.237.27.51:6379",
		Username: "passontw",
		Password: "1qaz@WSX3edc", // 如果有密碼設定
		DB:       3,              // 使用默認數據庫
	})

	// 檢查 Redis 連接
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("連接 Redis 失敗: %v", err)
	}
	defer redisClient.Close()

	// 創建開獎服務
	drawService := NewBingoDrawService(db, redisClient)

	// 設置 WebSocket 升級器
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 開發階段允許所有來源，生產環境應該更嚴格
		},
	}

	// 荷官端 WebSocket 處理
	http.HandleFunc("/dealer", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("升級荷官連接錯誤: %v", err)
			return
		}

		// 註冊荷官客戶端
		drawService.RegisterClient(conn, "dealer")

		// 處理荷官訊息
		go handleWebSocketConnection(conn, drawService, "dealer")
	})

	// 遊戲端 WebSocket 處理
	http.HandleFunc("/game", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("升級遊戲連接錯誤: %v", err)
			return
		}

		// 註冊遊戲客戶端
		drawService.RegisterClient(conn, "game")

		// 處理遊戲訊息
		go handleWebSocketConnection(conn, drawService, "game")
	})

	// 啟動 HTTP 服務器
	log.Println("開獎服務已啟動，監聽 8080 端口...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("啟動 HTTP 服務器失敗: %v", err)
	}
}

// 處理 WebSocket 連接
func handleWebSocketConnection(conn *websocket.Conn, service *BingoDrawService, clientType string) {
	defer func() {
		service.UnregisterClient(conn, clientType)
		conn.Close()
	}()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("讀取 WebSocket 消息錯誤: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(msgData, &msg); err != nil {
			log.Printf("解析消息錯誤: %v", err)
			continue
		}

		// 處理命令
		if msg.Type == MsgTypeCommand {
			service.HandleCommand(msg, conn, clientType)
		}
	}
}

// 確保必要的表格存在
func ensureTablesExist(db *sql.DB) error {
	// 檢查並創建 games 表
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS games (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		game_id VARCHAR(50) NOT NULL UNIQUE,
		state VARCHAR(30) NOT NULL,           
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NULL,
		has_jackpot BOOLEAN NOT NULL DEFAULT FALSE,
		jackpot_amount DECIMAL(18, 2) DEFAULT 0,
		extra_ball_count INT NOT NULL DEFAULT 3,
		current_state_start_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		max_timeout INT NOT NULL DEFAULT 60,
		total_cards INT DEFAULT 0,
		total_players INT DEFAULT 0,
		total_bet_amount DECIMAL(18, 2) DEFAULT 0,
		total_win_amount DECIMAL(18, 2) DEFAULT 0,
		cancelled BOOLEAN NOT NULL DEFAULT FALSE,
		cancel_time TIMESTAMP NULL,
		cancelled_by VARCHAR(50) NULL,
		cancel_reason VARCHAR(255) NULL,
		game_snapshot JSON NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_state (state),
		INDEX idx_start_time (start_time)
	)`)
	if err != nil {
		return fmt.Errorf("創建 games 表錯誤: %v", err)
	}

	// 檢查並創建 drawn_balls 表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS drawn_balls (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		game_id VARCHAR(50) NOT NULL,
		number INT NOT NULL,
		sequence INT NOT NULL,
		ball_type ENUM('REGULAR', 'EXTRA', 'JACKPOT', 'LUCKY') NOT NULL,
		drawn_time TIMESTAMP NOT NULL,
		side VARCHAR(10) NULL,
		is_last_ball BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uk_game_type_sequence (game_id, ball_type, sequence),
		INDEX idx_game_id (game_id)
	)`)
	if err != nil {
		return fmt.Errorf("創建 drawn_balls 表錯誤: %v", err)
	}

	// 檢查並創建 lucky_balls 表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS lucky_balls (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		game_id VARCHAR(50) NULL,
		draw_date TIMESTAMP NOT NULL,
		number1 INT NOT NULL,
		number2 INT NOT NULL,
		number3 INT NOT NULL,
		number4 INT NOT NULL,
		number5 INT NOT NULL,
		number6 INT NOT NULL,
		number7 INT NOT NULL,
		active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_active (active),
		INDEX idx_game_id (game_id)
	)`)
	if err != nil {
		return fmt.Errorf("創建 lucky_balls 表錯誤: %v", err)
	}

	return nil
}
