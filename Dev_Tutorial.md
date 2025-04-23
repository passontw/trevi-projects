# 樂透開獎服務開發教學

## 目錄

1. [專案概述](#專案概述)
2. [系統架構](#系統架構)
3. [開發環境設置](#開發環境設置)
4. [開發步驟](#開發步驟)
5. [單元測試](#單元測試)
6. [荷官端與遊戲端模擬方案](#荷官端與遊戲端模擬方案)

## 專案概述

樂透開獎服務是一個基於 WebSocket 的實時通訊服務，主要負責管理樂透遊戲的開獎流程、處理荷官端的抽球輸入、維護遊戲狀態並向遊戲端推送結果。系統包括主遊戲流程與 JP (Jackpot) 遊戲流程兩個主要部分。

### 主要功能

- 荷官端與遊戲端的 WebSocket 連接管理
- 遊戲狀態管理與轉換
- 幸運球設定與傳播
- 開獎球號處理與廣播
- 額外球處理機制
- JP 遊戲支援
- 計時器管理
- 心跳機制維護

### 遊戲狀態

系統內部定義了以下遊戲狀態：

```go
type GameState string

const (
    StateInitial         GameState = "INITIAL"           // 初始狀態
    StateReady           GameState = "READY"             // 待機狀態
    StateShowLuckyNums   GameState = "SHOW_LUCKYNUMS"    // 開七個幸運球的狀態
    StateBetting         GameState = "BETTING"           // 投注狀態
    StateShowBalls       GameState = "SHOW_BALLS"        // 開獎狀態
    StateChooseExtraBall GameState = "CHOOSE_EXTRA_BALL" // 額外球投注狀態
    StateShowExtraBalls  GameState = "SHOW_EXTRA_BALLS"  // 額外球開獎狀態
    StateResult          GameState = "MG_CONCLUDE"       // 結算狀態
    StateJPReady         GameState = "JP_READY"          // JP待機狀態
    StateJPShowBalls     GameState = "JP_SHOW_BALLS"     // JP開獎狀態
    StateJPConclude      GameState = "JP_CONCLUDE"       // JP結算狀態
)
```

這些狀態將驅動整個遊戲流程的轉換與行為。

## 系統架構

開獎服務採用三層架構：

1. **API 層**：處理 WebSocket 連接、消息解析與分發
2. **服務層**：包含遊戲邏輯、狀態管理、計時器管理
3. **領域層**：定義核心遊戲模型、狀態機制與持久化邏輯 

## 開發環境設置

### 必要工具

- Go 1.16+ (建議使用 Go 1.18+)
- Git
- Redis (用於緩存遊戲狀態)
- MySQL 或 PostgreSQL (用於持久化遊戲數據)
- WebSocket 客戶端測試工具 (如 WebSocket King 或自定義測試頁面)

### 項目初始化

```bash
# 創建項目目錄
mkdir -p g38_lottery_service
cd g38_lottery_service

# 初始化 Go 模塊
go mod init github.com/your-username/g38_lottery_service

# 創建基本目錄結構
mkdir -p cmd/server
mkdir -p internal/{api,service,domain,config,websocket,utils}
mkdir -p pkg
mkdir -p test
mkdir -p wsclient
```

### 安裝依賴

```bash
# WebSocket 處理庫
go get github.com/gorilla/websocket

# 網絡框架
go get github.com/gin-gonic/gin

# 資料庫處理
go get gorm.io/gorm
go get gorm.io/driver/mysql

# 配置管理
go get github.com/spf13/viper

# UUID 生成
go get github.com/google/uuid

# 測試工具
go get github.com/stretchr/testify
```

## 開發步驟

### 步驟 1：定義基本模型與狀態

首先創建遊戲基本模型與狀態定義：

```go
// internal/domain/models.go
package domain

import (
	"time"
)

// 遊戲狀態定義
type GameState string

const (
    StateInitial         GameState = "INITIAL"           // 初始狀態
    StateReady           GameState = "READY"             // 待機狀態
    StateShowLuckyNums   GameState = "SHOW_LUCKYNUMS"    // 開七個幸運球的狀態
    StateBetting         GameState = "BETTING"           // 投注狀態
    StateShowBalls       GameState = "SHOW_BALLS"        // 開獎狀態
    StateChooseExtraBall GameState = "CHOOSE_EXTRA_BALL" // 額外球投注狀態
    StateShowExtraBalls  GameState = "SHOW_EXTRA_BALLS"  // 額外球開獎狀態
    StateResult          GameState = "MG_CONCLUDE"       // 結算狀態
    StateJPReady         GameState = "JP_READY"          // JP待機狀態
    StateJPShowBalls     GameState = "JP_SHOW_BALLS"     // JP開獎狀態
    StateJPConclude      GameState = "JP_CONCLUDE"       // JP結算狀態
)

// 球結構
type Ball struct {
    Number     int       `json:"number"`
    DrawnTime  time.Time `json:"drawn_time"`
    Sequence   int       `json:"sequence"`
}

// 遊戲結構
type Game struct {
    ID              string     `json:"id"`
    State           GameState  `json:"state"`
    StartTime       time.Time  `json:"start_time"`
    EndTime         time.Time  `json:"end_time,omitempty"`
    LuckyNumbers    []int      `json:"lucky_numbers,omitempty"`
    DrawnBalls      []Ball     `json:"drawn_balls"`
    ExtraBalls      []Ball     `json:"extra_balls"`
    ExtraBallCount  int        `json:"extra_ball_count"`
    HasJackpot      bool       `json:"has_jackpot"`
}

// JP遊戲結構
type JPGame struct {
    ID             string     `json:"id"`
    GameID         string     `json:"game_id"`
    StartTime      time.Time  `json:"start_time"`
    EndTime        time.Time  `json:"end_time,omitempty"`
    DrawnBalls     []Ball     `json:"drawn_balls"`
    JackpotAmount  int64      `json:"jackpot_amount"`
    Winner         JPWinner   `json:"winner,omitempty"`
}

// JP獲獎者
type JPWinner struct {
    UserID       uint       `json:"user_id"`
    CardID       string     `json:"card_id"`
    WinTime      time.Time  `json:"win_time"`
    Amount       int64      `json:"amount"`
}
```

### 步驟 2：定義消息結構與命令類型

為 WebSocket 通訊定義消息格式與命令類型：

```go
// internal/domain/messages.go
package domain

// 命令類型
type CommandType string

const (
    // 荷官端命令
    CmdGameStart        CommandType = "GAME_START"
    CmdShowLuckyNumbers CommandType = "SHOW_LUCKY_NUMBERS"
    CmdDrawBall         CommandType = "DRAW_BALL"
    CmdDrawExtraBall    CommandType = "DRAW_EXTRA_BALL"
    CmdStartJPGame      CommandType = "START_JP_GAME"
    CmdDrawJPBall       CommandType = "DRAW_JP_BALL"
    
    // 遊戲端命令
    CmdPlaceBet         CommandType = "PLACE_BET"
    CmdExtraBet         CommandType = "EXTRA_BET"
    CmdJoinJP           CommandType = "JOIN_JP"
    CmdGameResult       CommandType = "GAME_RESULT"
    CmdJPWinner         CommandType = "JP_WINNER"
    
    // 通用命令
    CmdHeartbeat        CommandType = "HEARTBEAT"
    CmdRequestGameState CommandType = "REQUEST_GAME_STATE"
)

// 事件類型
type EventType string

const (
    EventGameStateChanged EventType = "GAME_STATE_CHANGED"
    EventGameReady        EventType = "GAME_READY"
    EventLuckyNumbersSet  EventType = "LUCKY_NUMBERS_SET"
    EventBallDrawn        EventType = "BALL_DRAWN"
    EventExtraBallDrawn   EventType = "EXTRA_BALL_DRAWN"
    EventGameResult       EventType = "GAME_RESULT"
    EventJPGameStart      EventType = "JP_GAME_START"
    EventJPBallDrawn      EventType = "JP_BALL_DRAWN"
    EventJPWinner         EventType = "JP_WINNER"
    EventJPGameResult     EventType = "JP_GAME_RESULT"
    EventCountdownUpdate  EventType = "COUNTDOWN_UPDATE"
    EventError            EventType = "ERROR"
)

// 通用消息結構
type Message struct {
    Type      string      `json:"type"`
    Data      interface{} `json:"data,omitempty"`
    Timestamp int64       `json:"timestamp"`
}

// 命令結構
type Command struct {
    Type      CommandType `json:"type"`
    Data      interface{} `json:"data,omitempty"`
    ClientID  string      `json:"client_id,omitempty"`
    Timestamp int64       `json:"timestamp"`
}

// 事件結構
type Event struct {
    Type      EventType   `json:"type"`
    Data      interface{} `json:"data,omitempty"`
    Timestamp int64       `json:"timestamp"`
}
```

### 步驟 3：實現 WebSocket 連接管理

建立 WebSocket 連接與客戶端管理：

```go
// internal/websocket/manager.go
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/your-username/g38_lottery_service/internal/domain"
)

// WebSocket連接管理器
type Manager struct {
	dealerClients map[string]*Client
	playerClients map[string]*Client
	broadcast     chan []byte
	register      chan *Client
	unregister    chan *Client
	mutex         sync.RWMutex
}

// 客戶端類型
type ClientType string

const (
	ClientTypeDealer ClientType = "DEALER"
	ClientTypePlayer ClientType = "PLAYER"
)

// 客戶端連接
type Client struct {
	ID             string
	Type           ClientType
	UserID         uint
	Conn           *websocket.Conn
	Send           chan []byte
	Manager        *Manager
	LastActivity   time.Time
	HeartbeatTimer *time.Ticker
	closeChan      chan struct{}
}

// 創建新的WebSocket管理器
func NewManager() *Manager {
	return &Manager{
		dealerClients: make(map[string]*Client),
		playerClients: make(map[string]*Client),
		broadcast:     make(chan []byte, 256),
		register:      make(chan *Client, 10),
		unregister:    make(chan *Client, 10),
		mutex:         sync.RWMutex{},
	}
}

// 啟動WebSocket管理器
func (m *Manager) Start(ctx context.Context) {
    // 實現啟動邏輯，處理註冊、註銷、廣播等
}

// 註冊客戶端
func (m *Manager) RegisterClient(client *Client) {
    // 實現客戶端註冊
}

// 註銷客戶端
func (m *Manager) UnregisterClient(client *Client) {
    // 實現客戶端註銷
}

// 廣播給所有客戶端
func (m *Manager) BroadcastToAll(message interface{}) error {
    // 實現廣播
}

// 廣播給所有荷官端
func (m *Manager) BroadcastToDealers(message interface{}) error {
    // 實現廣播給荷官端
}

// 廣播給所有遊戲端
func (m *Manager) BroadcastToPlayers(message interface{}) error {
    // 實現廣播給遊戲端
}

// 給指定客戶端發送訊息
func (m *Manager) SendToClient(clientID string, message interface{}) error {
    // 實現發送給指定客戶端
}

// 客戶端讀取泵
func (c *Client) ReadPump() {
    // 實現讀取泵
}

// 客戶端寫入泵
func (c *Client) WritePump() {
    // 實現寫入泵
}

// 啟動心跳
func (c *Client) StartHeartbeat() {
    // 實現心跳機制
}
```

### 步驟 4：實現遊戲服務

實現核心遊戲邏輯：

```go
// internal/service/game_service.go
package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/websocket"
)

// 遊戲服務
type GameService struct {
	currentGame      *domain.Game
	currentJPGame    *domain.JPGame
	wsManager        *websocket.Manager
	timerService     *TimerService
	mutex            sync.RWMutex
	stateChangeHooks map[domain.GameState]func()
}

// 創建新的遊戲服務
func NewGameService(wsManager *websocket.Manager, timerService *TimerService) *GameService {
    // 實現創建
}

// 初始化遊戲
func (s *GameService) InitializeGame() error {
    // 實現初始化邏輯
}

// 開始遊戲
func (s *GameService) StartGame() (string, error) {
    // 實現開始遊戲邏輯
}

// 設置幸運號碼
func (s *GameService) ShowLuckyNumbers(numbers []int) error {
    // 實現設置幸運號碼邏輯
}

// 開始投注
func (s *GameService) StartBetting(seconds int) error {
    // 實現開始投注邏輯
}

// 抽出球號
func (s *GameService) DrawBall(number int) error {
    // 實現抽球邏輯
}

// 開始額外球投注
func (s *GameService) StartExtraBetting(extraBallCount int) error {
    // 實現開始額外球投注邏輯
}

// 抽出額外球
func (s *GameService) DrawExtraBall(number int) error {
    // 實現抽額外球邏輯
}

// 結束遊戲
func (s *GameService) FinishGame() error {
    // 實現結束遊戲邏輯
}

// 開始JP遊戲
func (s *GameService) StartJPGame() (string, error) {
    // 實現開始JP遊戲邏輯
}

// 抽出JP球
func (s *GameService) DrawJPBall(number int) error {
    // 實現抽JP球邏輯
}

// 登記JP獲勝者
func (s *GameService) RegisterJPWinner(winner domain.JPWinner) error {
    // 實現登記JP獲勝者邏輯
}

// 結束JP遊戲
func (s *GameService) FinishJPGame() error {
    // 實現結束JP遊戲邏輯
}

// 切換遊戲狀態
func (s *GameService) changeGameState(newState domain.GameState) error {
    // 實現狀態切換邏輯
}

// 獲取當前遊戲
func (s *GameService) GetCurrentGame() (*domain.Game, error) {
    // 實現獲取當前遊戲
}

// 獲取當前JP遊戲
func (s *GameService) GetCurrentJPGame() (*domain.JPGame, error) {
    // 實現獲取當前JP遊戲
}

// 獲取當前遊戲狀態
func (s *GameService) GetGameState() domain.GameState {
    // 實現獲取當前狀態
}
```

### 步驟 5：實現計時器服務

創建用於倒計時與計時功能的服務：

```go
// internal/service/timer_service.go
package service

import (
	"sync"
	"time"
)

// 計時器服務
type TimerService struct {
	currentTimer    *time.Timer
	remainingTime   int
	callbackFunc    func()
	mutex           sync.Mutex
	isRunning       bool
	tickerChan      chan int
}

// 創建新計時器服務
func NewTimerService() *TimerService {
    // 實現創建
}

// 開始倒計時
func (s *TimerService) StartCountdown(seconds int, callback func()) {
    // 實現開始倒計時
}

// 停止倒計時
func (s *TimerService) StopCountdown() {
    // 實現停止倒計時
}

// 獲取剩餘秒數
func (s *TimerService) GetRemainingSeconds() int {
    // 實現獲取剩餘秒數
}

// 訂閱倒計時更新
func (s *TimerService) SubscribeToTicks() <-chan int {
    // 實現訂閱倒計時更新
}
```

### 步驟 6：實現 WebSocket 處理器

創建處理 WebSocket 連接與消息的處理器：

```go
// internal/api/websocket_handler.go
package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/service"
	ws "github.com/your-username/g38_lottery_service/internal/websocket"
)

// WebSocket升級器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允許所有來源
	},
}

// WebSocket處理器
type WebSocketHandler struct {
	gameService *service.GameService
	wsManager   *ws.Manager
}

// 創建WebSocket處理器
func NewWebSocketHandler(gameService *service.GameService, wsManager *ws.Manager) *WebSocketHandler {
    // 實現創建
}

// 處理荷官端連接
func (h *WebSocketHandler) HandleDealerConnection(c *gin.Context) {
    // 實現荷官端連接處理
}

// 處理遊戲端連接
func (h *WebSocketHandler) HandlePlayerConnection(c *gin.Context) {
    // 實現遊戲端連接處理
}

// 處理荷官端命令
func (h *WebSocketHandler) HandleDealerCommand(client *ws.Client, rawMessage []byte) {
    // 實現荷官端命令處理
}

// 處理遊戲端命令
func (h *WebSocketHandler) HandlePlayerCommand(client *ws.Client, rawMessage []byte) {
    // 實現遊戲端命令處理
}
```

### 步驟 7：實現命令處理器

創建處理具體命令的處理器：

```go
// internal/api/command_handler.go
package api

import (
	"encoding/json"
	"log"
	"time"

	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/service"
	"github.com/your-username/g38_lottery_service/internal/websocket"
)

// 命令處理器
type CommandHandler struct {
	gameService  *service.GameService
	timerService *service.TimerService
	wsManager    *websocket.Manager
}

// 創建命令處理器
func NewCommandHandler(gameService *service.GameService, timerService *service.TimerService, wsManager *websocket.Manager) *CommandHandler {
    // 實現創建
}

// 處理開始遊戲命令
func (h *CommandHandler) HandleGameStart(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理開始遊戲命令
}

// 處理設置幸運號碼命令
func (h *CommandHandler) HandleShowLuckyNumbers(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理設置幸運號碼命令
}

// 處理抽球命令
func (h *CommandHandler) HandleDrawBall(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理抽球命令
}

// 處理抽額外球命令
func (h *CommandHandler) HandleDrawExtraBall(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理抽額外球命令
}

// 處理開始JP遊戲命令
func (h *CommandHandler) HandleStartJPGame(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理開始JP遊戲命令
}

// 處理抽JP球命令
func (h *CommandHandler) HandleDrawJPBall(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理抽JP球命令
}

// 處理投注命令
func (h *CommandHandler) HandlePlaceBet(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理投注命令
}

// 處理額外球投注命令
func (h *CommandHandler) HandleExtraBet(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理額外球投注命令
}

// 處理加入JP遊戲命令
func (h *CommandHandler) HandleJoinJP(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理加入JP遊戲命令
}

// 處理遊戲結果命令
func (h *CommandHandler) HandleGameResult(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理遊戲結果命令
}

// 處理JP獲勝者命令
func (h *CommandHandler) HandleJPWinner(client *websocket.Client, data map[string]interface{}) error {
    // 實現處理JP獲勝者命令
}
```

### 步驟 8：實現事件通知

創建用於向客戶端發送事件通知的服務：

```go
// internal/service/notification_service.go
package service

import (
	"time"

	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/websocket"
)

// 通知服務
type NotificationService struct {
	wsManager *websocket.Manager
}

// 創建通知服務
func NewNotificationService(wsManager *websocket.Manager) *NotificationService {
    return &NotificationService{
        wsManager: wsManager,
    }
}

// 通知遊戲狀態改變
func (s *NotificationService) NotifyGameStateChanged(state domain.GameState, remainingTime int) error {
    event := domain.Event{
        Type:      domain.EventGameStateChanged,
        Data: map[string]interface{}{
            "state":         state,
            "remainingTime": remainingTime,
        },
        Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
    }
    
    // 廣播給所有客戶端
    return s.wsManager.BroadcastToAll(event)
}

// 通知遊戲準備就緒
func (s *NotificationService) NotifyGameReady(gameID string) error {
    event := domain.Event{
        Type:      domain.EventGameReady,
        Data: map[string]interface{}{
            "gameId": gameID,
        },
        Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
    }
    
    return s.wsManager.BroadcastToAll(event)
}

// 通知幸運號碼設置
func (s *NotificationService) NotifyLuckyNumbersSet(numbers []int) error {
    // 實現通知幸運號碼設置
}

// 通知球抽出
func (s *NotificationService) NotifyBallDrawn(number int, sequence int) error {
    // 實現通知球抽出
}

// 通知額外球抽出
func (s *NotificationService) NotifyExtraBallDrawn(number int, side string) error {
    // 實現通知額外球抽出
}

// 通知遊戲結果
func (s *NotificationService) NotifyGameResult(game *domain.Game, topPlayers []map[string]interface{}) error {
    // 實現通知遊戲結果
}

// 通知開始JP遊戲
func (s *NotificationService) NotifyJPGameStart(jpGame *domain.JPGame) error {
    // 實現通知開始JP遊戲
}

// 通知JP球抽出
func (s *NotificationService) NotifyJPBallDrawn(number int, sequence int) error {
    // 實現通知JP球抽出
}

// 通知JP獲勝者
func (s *NotificationService) NotifyJPWinner(winner domain.JPWinner) error {
    // 實現通知JP獲勝者
}

// 通知JP遊戲結果
func (s *NotificationService) NotifyJPGameResult(jpGame *domain.JPGame) error {
    // 實現通知JP遊戲結果
}

// 通知倒計時更新
func (s *NotificationService) NotifyCountdownUpdate(seconds int) error {
    // 實現通知倒計時更新
}

// 通知錯誤
func (s *NotificationService) NotifyError(code string, message string) error {
    // 實現通知錯誤
}
```

### 步驟 9：設置路由與啟動服務

創建 Gin 路由配置並啟動服務：

```go
// internal/api/router.go
package api

import (
	"github.com/gin-gonic/gin"
)

// 設置路由
func SetupRouter(wsHandler *WebSocketHandler) *gin.Engine {
	r := gin.Default()
	
	// CORS中間件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
	
	// WebSocket 端點
	r.GET("/ws", func(c *gin.Context) {
		clientType := c.Query("type")
		if clientType == "dealer" {
			wsHandler.HandleDealerConnection(c)
		} else if clientType == "player" {
			wsHandler.HandlePlayerConnection(c)
		} else {
			c.JSON(400, gin.H{"error": "Invalid client type"})
		}
	})
	
	// 健康檢查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	
	return r
}
```

### 步驟 10：創建主程序

創建主入口點啟動服務：

```go
// cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-username/g38_lottery_service/internal/api"
	"github.com/your-username/g38_lottery_service/internal/service"
	"github.com/your-username/g38_lottery_service/internal/websocket"
)

func main() {
	// 創建WebSocket管理器
	wsManager := websocket.NewManager()
	
	// 創建計時器服務
	timerService := service.NewTimerService()
	
	// 創建遊戲服務
	gameService := service.NewGameService(wsManager, timerService)
	
	// 創建通知服務
	notificationService := service.NewNotificationService(wsManager)
	
	// 創建命令處理器
	commandHandler := api.NewCommandHandler(gameService, timerService, wsManager)
	
	// 創建WebSocket處理器
	wsHandler := api.NewWebSocketHandler(gameService, wsManager)
	
	// 設置路由
	router := api.SetupRouter(wsHandler)
	
	// 在背景啟動WebSocket管理器
	ctx, cancel := context.WithCancel(context.Background())
	go wsManager.Start(ctx)
	
	// 配置HTTP服務器
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	
	// 在背景啟動服務器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	
	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	// 取消WebSocket管理器上下文
	cancel()
	
	// 創建關閉上下文
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 關閉HTTP服務器
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	
	log.Println("Server exiting")
}
```

## 單元測試

### 測試結構

為確保系統穩定性，應針對關鍵組件編寫單元測試。以下是測試分類：

1. **模型測試**：測試核心模型邏輯
2. **服務測試**：測試遊戲服務、通知服務和計時器服務
3. **命令處理測試**：測試各類命令處理邏輯
4. **整合流程測試**：測試完整遊戲流程

### 模擬與依賴注入

在測試中使用模擬（Mock）對象和依賴注入，便於測試隔離和控制：

```go
// test/mocks/ws_manager_mock.go
package mocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/websocket"
)

// WSManagerMock 模擬WebSocket管理器
type WSManagerMock struct {
	mock.Mock
}

func (m *WSManagerMock) RegisterClient(client *websocket.Client) {
	m.Called(client)
}

func (m *WSManagerMock) UnregisterClient(client *websocket.Client) {
	m.Called(client)
}

func (m *WSManagerMock) BroadcastToAll(message interface{}) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *WSManagerMock) BroadcastToDealers(message interface{}) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *WSManagerMock) BroadcastToPlayers(message interface{}) error {
	args := m.Called(message)
	return args.Error(0)
}
```

### 測試案例列舉

以下是針對各個關鍵組件的測試案例列舉：

#### 1. 遊戲服務測試

```go
// test/service/game_service_test.go
package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/service"
	"github.com/your-username/g38_lottery_service/test/mocks"
)

func TestInitializeGame(t *testing.T) {
	// 設置
	wsManagerMock := new(mocks.WSManagerMock)
	timerServiceMock := new(mocks.TimerServiceMock)
	gameService := service.NewGameService(wsManagerMock, timerServiceMock)
	
	// 執行
	err := gameService.InitializeGame()
	
	// 驗證
	assert.NoError(t, err)
	assert.Equal(t, domain.StateInitial, gameService.GetGameState())
}

func TestStartGame(t *testing.T) {
	// 測試案例實現
}

func TestShowLuckyNumbers(t *testing.T) {
	// 測試案例實現
}

func TestStartBetting(t *testing.T) {
	// 測試案例實現
}

func TestDrawBall(t *testing.T) {
	// 測試球號抽出
}

func TestStartExtraBetting(t *testing.T) {
	// 測試額外球投注開始
}

func TestDrawExtraBall(t *testing.T) {
	// 測試額外球抽出
}

func TestFinishGame(t *testing.T) {
	// 測試遊戲結束
}

func TestStartJPGame(t *testing.T) {
	// 測試JP遊戲開始
}

func TestDrawJPBall(t *testing.T) {
	// 測試JP球抽出
}

func TestRegisterJPWinner(t *testing.T) {
	// 測試JP獲勝者註冊
}

func TestFinishJPGame(t *testing.T) {
	// 測試JP遊戲結束
}

func TestChangeGameState(t *testing.T) {
	// 測試遊戲狀態切換
}
```

#### 2. 計時器服務測試

```go
// test/service/timer_service_test.go
package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/your-username/g38_lottery_service/internal/service"
)

func TestStartCountdown(t *testing.T) {
	// 設置
	timerService := service.NewTimerService()
	callbackCalled := false
	
	// 執行
	timerService.StartCountdown(1, func() {
		callbackCalled = true
	})
	
	// 等待回調觸發
	time.Sleep(1100 * time.Millisecond)
	
	// 驗證
	assert.True(t, callbackCalled)
	assert.Equal(t, 0, timerService.GetRemainingSeconds())
}

func TestStopCountdown(t *testing.T) {
	// 測試停止倒計時
}

func TestGetRemainingSeconds(t *testing.T) {
	// 測試獲取剩餘秒數
}
```

#### 3. 命令處理器測試

```go
// test/api/command_handler_test.go
package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/your-username/g38_lottery_service/internal/api"
	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/websocket"
	"github.com/your-username/g38_lottery_service/test/mocks"
)

func TestHandleGameStart(t *testing.T) {
	// 設置
	gameServiceMock := new(mocks.GameServiceMock)
	timerServiceMock := new(mocks.TimerServiceMock)
	wsManagerMock := new(mocks.WSManagerMock)
	
	gameServiceMock.On("StartGame").Return("test-game-1", nil)
	
	commandHandler := api.NewCommandHandler(gameServiceMock, timerServiceMock, wsManagerMock)
	client := &websocket.Client{ID: "test-dealer", Type: websocket.ClientTypeDealer}
	
	// 執行
	err := commandHandler.HandleGameStart(client, nil)
	
	// 驗證
	assert.NoError(t, err)
	gameServiceMock.AssertExpectations(t)
}

func TestHandleShowLuckyNumbers(t *testing.T) {
	// 測試處理顯示幸運號碼命令
}

func TestHandleDrawBall(t *testing.T) {
	// 測試處理抽球命令
}

func TestHandleDrawExtraBall(t *testing.T) {
	// 測試處理抽額外球命令
}

func TestHandleStartJPGame(t *testing.T) {
	// 測試處理開始JP遊戲命令
}

func TestHandleDrawJPBall(t *testing.T) {
	// 測試處理抽JP球命令
}
```

#### 4. 遊戲流程整合測試

```go
// test/integration/game_flow_test.go
package integration_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/your-username/g38_lottery_service/internal/domain"
	"github.com/your-username/g38_lottery_service/internal/service"
	"github.com/your-username/g38_lottery_service/internal/websocket"
)

func TestMainGameFlow(t *testing.T) {
	// 設置
	wsManager := websocket.NewManager()
	timerService := service.NewTimerService()
	gameService := service.NewGameService(wsManager, timerService)
	
	// 初始化遊戲
	err := gameService.InitializeGame()
	assert.NoError(t, err)
	assert.Equal(t, domain.StateInitial, gameService.GetGameState())
	
	// 啟動遊戲
	gameId, err := gameService.StartGame()
	assert.NoError(t, err)
	assert.NotEmpty(t, gameId)
	assert.Equal(t, domain.StateReady, gameService.GetGameState())
	
	// 設置幸運號碼
	err = gameService.ShowLuckyNumbers([]int{1, 2, 3, 4, 5, 6, 7})
	assert.NoError(t, err)
	assert.Equal(t, domain.StateShowLuckyNums, gameService.GetGameState())
	
	// 開始投注
	err = gameService.StartBetting(2) // 2秒倒計時以加速測試
	assert.NoError(t, err)
	assert.Equal(t, domain.StateBetting, gameService.GetGameState())
	
	// 等待投注階段結束
	time.Sleep(2500 * time.Millisecond)
	
	// 抽球
	for i := 1; i <= 30; i++ {
		err = gameService.DrawBall(i)
		assert.NoError(t, err)
	}
	
	// 開始額外球投注
	err = gameService.StartExtraBetting(2)
	assert.NoError(t, err)
	assert.Equal(t, domain.StateChooseExtraBall, gameService.GetGameState())
	
	// 等待額外球投注階段結束
	time.Sleep(2500 * time.Millisecond)
	
	// 抽出額外球
	err = gameService.DrawExtraBall(31)
	assert.NoError(t, err)
	err = gameService.DrawExtraBall(32)
	assert.NoError(t, err)
	
	// 結束遊戲
	err = gameService.FinishGame()
	assert.NoError(t, err)
	assert.Equal(t, domain.StateResult, gameService.GetGameState())
}

func TestJPGameFlow(t *testing.T) {
	// 測試完整JP遊戲流程
}
```

## 荷官端與遊戲端模擬方案

為了測試開獎服務，我們需要模擬荷官端和遊戲端的行為。以下提供兩種模擬方案：

### 1. 命令行工具模擬

創建一個簡單的命令行工具，模擬荷官端和遊戲端的操作：

```go
// cmd/simulator/main.go
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// 解析命令行參數
	var role string
	var serverURL string
	
	flag.StringVar(&role, "role", "DEALER", "客戶端角色 (DEALER 或 PLAYER)")
	flag.StringVar(&serverURL, "url", "ws://localhost:8080/ws", "WebSocket 服務器 URL")
	flag.Parse()
	
	if role != "DEALER" && role != "PLAYER" {
		log.Fatal("角色必須是 DEALER 或 PLAYER")
	}
	
	// 連接 WebSocket
	u, err := url.Parse(serverURL)
	if err != nil {
		log.Fatal("無效的 URL:", err)
	}
	
	// 添加角色參數
	q := u.Query()
	q.Set("type", strings.ToLower(role))
	u.RawQuery = q.Encode()
	
	// 連接 WebSocket
	log.Printf("正在連接到 %s 作為 %s...\n", u.String(), role)
	
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("連接失敗:", err)
	}
	defer c.Close()
	
	log.Println("已連接到服務器")
	
	// 啟動接收消息的 goroutine
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("讀取錯誤:", err)
				return
			}
			
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, message, "", "  "); err != nil {
				log.Printf("收到: %s\n", message)
			} else {
				log.Printf("收到:\n%s\n", prettyJSON.String())
			}
		}
	}()
	
	// 啟動心跳
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				heartbeat := map[string]interface{}{
					"type": "HEARTBEAT",
					"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
				}
				
				if err := c.WriteJSON(heartbeat); err != nil {
					log.Println("發送心跳失敗:", err)
					return
				}
				
				log.Println("已發送心跳")
			}
		}
	}()
	
	// 讀取用戶輸入的命令
	reader := bufio.NewReader(os.Stdin)
	
	// 顯示幫助信息
	if role == "DEALER" {
		fmt.Println(`
荷官端命令:
  start                   - 開始新遊戲
  lucky 1,2,3,4,5,6,7     - 設置幸運號碼
  draw 42                 - 抽出球號
  extra 3                 - 開始額外球 (參數為數量)
  draw_extra 18           - 抽出額外球
  jp                      - 開始JP遊戲
  draw_jp 33              - 抽出JP球
  exit                    - 退出
`)
	} else {
		fmt.Println(`
遊戲端命令:
  bet 1,5,10,15,20,25,30 100  - 投注 (號碼和金額)
  bet_extra LEFT 50           - 額外球投注
  join_jp 2                   - 參與JP遊戲 (卡片數量)
  result                      - 發送遊戲結果
  jp_winner 12345 JP001 5000  - 模擬JP獲勝
  exit                        - 退出
`)
	}
	
	// 主循環
	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		
		if input == "exit" {
			break
		}
		
		command := parseCommand(input, role)
		if command == nil {
			continue
		}
		
		if err := c.WriteJSON(command); err != nil {
			log.Println("發送失敗:", err)
		} else {
			commandJSON, _ := json.MarshalIndent(command, "", "  ")
			log.Printf("已發送:\n%s\n", string(commandJSON))
		}
	}
}

// 解析輸入命令
func parseCommand(input string, role string) map[string]interface{} {
	// 根據角色和輸入解析命令
	// 實現解析邏輯 (根據上面幫助信息中的命令格式)
	// 返回格式化的命令對象
}
```

### 2. Web 界面模擬

使用簡單的 HTML/JS 頁面模擬荷官端和遊戲端。在 `wsclient` 目錄下創建以下文件：

1. **荷官端模擬器 (dealer-simulator.html)**：模擬荷官控制頁面
2. **遊戲端模擬器 (player-simulator.html)**：模擬玩家頁面

這些頁面提供用戶友好的界面，發送命令並接收和顯示服務器發回的消息。

```html
<!-- 荷官端和遊戲端模擬器 HTML 代碼 -->
<!-- 基本包含連接控制、命令按鈕和消息顯示區域 -->
```

### 建議的測試流程

1. 啟動開獎服務：
   ```bash
   go run cmd/server/main.go
   ```

2. 啟動模擬器：

   命令行模擬器：
   ```bash
   # 荷官端
   go run cmd/simulator/main.go -role=DEALER
   
   # 遊戲端
   go run cmd/simulator/main.go -role=PLAYER
   ```

   Web 界面模擬器：
   使用瀏覽器打開 `wsclient/dealer-simulator.html` 和 `wsclient/player-simulator.html`

3. 按照遊戲流程進行測試：
   - 連接服務
   - 開始遊戲
   - 設置幸運號碼
   - 進行投注
   - 抽出球號
   - 進行額外球投注
   - 抽出額外球
   - 結束遊戲
   - 如需要，進行 JP 遊戲測試

4. 觀察服務日誌和模擬器輸出，檢查消息流程和狀態轉換是否符合預期 

## 資料庫設計與初始化

為了支援樂透開獎服務的持久化需求，我們需要一個完善的資料庫結構來儲存遊戲進行相關的資料，包括遊戲記錄、抽出的球號、玩家投注和 JP (Jackpot) 遊戲資訊。本節將提供完整的資料庫初始化 SQL 和詳細說明。

### 資料庫設計概述

樂透開獎服務資料庫設計遵循了關聯式資料庫的最佳實踐，確保資料完整性和查詢效能。主要資料表包括遊戲記錄、球號資訊、玩家資料、投注記錄以及 JP 遊戲相關資訊。這些表格之間透過外鍵關聯，形成完整的資料結構，支援遊戲的全流程管理。 

### 資料庫初始化 SQL

```sql
-- 創建資料庫
CREATE DATABASE IF NOT EXISTS g38_lottery;
USE g38_lottery;

-- 設置字符集和排序規則
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 遊戲記錄表
CREATE TABLE IF NOT EXISTS `games` (
  `id` VARCHAR(36) NOT NULL COMMENT '遊戲唯一標識符',
  `state` VARCHAR(20) NOT NULL COMMENT '遊戲狀態',
  `start_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '遊戲開始時間',
  `end_time` TIMESTAMP NULL DEFAULT NULL COMMENT '遊戲結束時間',
  `has_jackpot` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否有JP遊戲',
  `extra_ball_count` INT NOT NULL DEFAULT 0 COMMENT '額外球數量',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  INDEX `idx_state` (`state`),
  INDEX `idx_start_time` (`start_time`),
  INDEX `idx_end_time` (`end_time`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '遊戲記錄表';

-- 幸運號碼表
CREATE TABLE IF NOT EXISTS `lucky_numbers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `number` INT NOT NULL COMMENT '幸運號碼',
  `sequence` INT NOT NULL COMMENT '序號',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  CONSTRAINT `fk_lucky_numbers_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '幸運號碼表';

-- 開獎球表
CREATE TABLE IF NOT EXISTS `drawn_balls` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `number` INT NOT NULL COMMENT '球號',
  `sequence` INT NOT NULL COMMENT '抽出順序',
  `drawn_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '抽出時間',
  `is_extra_ball` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否為額外球',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  INDEX `idx_sequence` (`sequence`),
  CONSTRAINT `fk_drawn_balls_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '開獎球表';

-- 額外球表
CREATE TABLE IF NOT EXISTS `extra_balls` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `number` INT NOT NULL COMMENT '球號',
  `side` VARCHAR(10) NOT NULL COMMENT '左側或右側',
  `drawn_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '抽出時間',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  CONSTRAINT `fk_extra_balls_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '額外球表';

-- 玩家表
CREATE TABLE IF NOT EXISTS `players` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `nickname` VARCHAR(50) NULL DEFAULT NULL COMMENT '玩家暱稱',
  `balance` BIGINT NOT NULL DEFAULT 0 COMMENT '玩家餘額',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_user_id` (`user_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '玩家表';

-- 投注記錄表
CREATE TABLE IF NOT EXISTS `bets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `bet_amount` INT NOT NULL COMMENT '投注金額',
  `selected_numbers` VARCHAR(300) NOT NULL COMMENT '選擇的號碼，逗號分隔',
  `bet_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '投注時間',
  `win_amount` INT NULL DEFAULT 0 COMMENT '贏取金額',
  `is_extra_bet` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否為額外球投注',
  `extra_side` VARCHAR(10) NULL DEFAULT NULL COMMENT '額外球左側或右側',
  `status` VARCHAR(20) NOT NULL DEFAULT 'PENDING' COMMENT '狀態：PENDING, SETTLED, CANCELED',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_status` (`status`),
  CONSTRAINT `fk_bets_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '投注記錄表';

-- JP遊戲表
CREATE TABLE IF NOT EXISTS `jp_games` (
  `id` VARCHAR(36) NOT NULL COMMENT 'JP遊戲唯一標識符',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的主遊戲ID',
  `start_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '開始時間',
  `end_time` TIMESTAMP NULL DEFAULT NULL COMMENT '結束時間',
  `jackpot_amount` BIGINT NOT NULL DEFAULT 0 COMMENT 'JP金額',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  CONSTRAINT `fk_jp_games_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP遊戲表';

-- JP球表
CREATE TABLE IF NOT EXISTS `jp_balls` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `jp_game_id` VARCHAR(36) NOT NULL COMMENT '關聯的JP遊戲ID',
  `number` INT NOT NULL COMMENT '球號',
  `sequence` INT NOT NULL COMMENT '抽出順序',
  `drawn_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '抽出時間',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_jp_game_id` (`jp_game_id`),
  INDEX `idx_sequence` (`sequence`),
  CONSTRAINT `fk_jp_balls_jp_game_id` FOREIGN KEY (`jp_game_id`) REFERENCES `jp_games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP球表';

-- JP參與記錄表
CREATE TABLE IF NOT EXISTS `jp_participations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `jp_game_id` VARCHAR(36) NOT NULL COMMENT '關聯的JP遊戲ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `card_id` VARCHAR(50) NOT NULL COMMENT '卡片ID',
  `join_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '參與時間',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_jp_game_id` (`jp_game_id`),
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_card_id` (`card_id`),
  CONSTRAINT `fk_jp_participations_jp_game_id` FOREIGN KEY (`jp_game_id`) REFERENCES `jp_games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP參與記錄表';

-- JP獲勝者表
CREATE TABLE IF NOT EXISTS `jp_winners` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `jp_game_id` VARCHAR(36) NOT NULL COMMENT '關聯的JP遊戲ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `card_id` VARCHAR(50) NOT NULL COMMENT '卡片ID',
  `win_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '獲勝時間',
  `amount` BIGINT NOT NULL COMMENT '贏取金額',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_jp_game_id` (`jp_game_id`),
  INDEX `idx_user_id` (`user_id`),
  CONSTRAINT `fk_jp_winners_jp_game_id` FOREIGN KEY (`jp_game_id`) REFERENCES `jp_games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP獲勝者表';

-- 系統配置表
CREATE TABLE IF NOT EXISTS `system_configs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `key` VARCHAR(100) NOT NULL COMMENT '配置鍵',
  `value` TEXT NOT NULL COMMENT '配置值',
  `description` VARCHAR(255) NULL DEFAULT NULL COMMENT '描述',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_key` (`key`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '系統配置表';

-- 連接日誌表
CREATE TABLE IF NOT EXISTS `connection_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `client_id` VARCHAR(36) NOT NULL COMMENT '客戶端ID',
  `client_type` VARCHAR(10) NOT NULL COMMENT '客戶端類型：DEALER或PLAYER',
  `user_id` VARCHAR(36) NULL DEFAULT NULL COMMENT '玩家ID（僅玩家類型）',
  `connect_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '連接時間',
  `disconnect_time` TIMESTAMP NULL DEFAULT NULL COMMENT '斷開時間',
  `ip_address` VARCHAR(50) NOT NULL COMMENT 'IP地址',
  `user_agent` VARCHAR(255) NULL DEFAULT NULL COMMENT '用戶代理',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_client_id` (`client_id`),
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_connect_time` (`connect_time`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '連接日誌表';

-- 插入基本配置數據
INSERT INTO `system_configs` (`key`, `value`, `description`) VALUES
('default_betting_time', '60', '默認投注時間（秒）'),
('default_extra_ball_time', '30', '默認額外球投注時間（秒）'),
('default_jp_min_amount', '10000', '默認JP最小金額'),
('max_extra_balls', '5', '最大額外球數量'),
('heartbeat_interval', '10', '心跳間隔（秒）'),
('connection_timeout', '30', '連接超時時間（秒）'),
('inactivity_timeout', '300', '不活動超時時間（秒）');

SET FOREIGN_KEY_CHECKS = 1;
```

### 資料表說明

#### games 表
儲存主遊戲的基本資訊，包括遊戲狀態、開始和結束時間、是否有JP遊戲等。每個遊戲都有一個唯一的UUID作為主鍵，便於在整個系統中識別。

#### lucky_numbers 表
儲存幸運號碼。在遊戲開始前，會設定7個幸運號碼，這些號碼會顯示給玩家，但不參與實際開獎。

#### drawn_balls 表
記錄遊戲中抽出的球號，包括序號和抽出時間。這些是主要的開獎號碼。

#### extra_balls 表
記錄額外球的抽出資訊。額外球是遊戲的特殊環節，允許玩家在主要開獎之後進行額外投注。

#### players 表
儲存玩家資訊，包括ID、暱稱和餘額等。

#### bets 表
記錄玩家的投注資訊，包括投注金額、所選號碼和投注結果。還包含額外球投注的相關資訊。

#### jp_games 表
儲存JP遊戲的資訊，包括開始和結束時間、JP金額等。每個JP遊戲都關聯到一個主遊戲。

#### jp_balls 表
記錄JP遊戲中抽出的球號。

#### jp_participations 表
記錄玩家參與JP遊戲的資訊，包括卡片ID和參與時間。

#### jp_winners 表
記錄JP遊戲的獲勝者資訊，包括獲勝時間和獲獎金額。

#### system_configs 表
儲存系統配置，如默認投注時間、心跳間隔等。

#### connection_logs 表
記錄客戶端連接和斷開的詳細資訊，用於監控和分析。 

### 資料關聯圖

```
games
└── drawn_balls (一對多)
└── lucky_numbers (一對多)
└── extra_balls (一對多)
└── bets (一對多)
└── jp_games (一對一)
    └── jp_balls (一對多)
    └── jp_participations (一對多)
    └── jp_winners (一對多)
```

### 設計考量

#### 索引策略
- 對頻繁查詢的欄位（如 game_id、user_id、status 等）建立了索引，以提高查詢效能。
- 為時間相關欄位（如 start_time、end_time、drawn_time 等）建立了索引，便於時間範圍查詢。
- 使用複合索引優化複雜查詢。

#### 外鍵關聯
- 實施了完整的外鍵約束，確保參照完整性。
- 使用 `ON DELETE CASCADE ON UPDATE CASCADE` 確保當主記錄被刪除或更新時，相關記錄也會自動處理。

#### 時間戳記和審計欄位
- 每個表格都包含 `created_at`，大多數表格還有 `updated_at`，用於追蹤記錄的創建和修改時間。
- 重要事件（如球抽出、投注、獲勝等）都記錄了精確的時間戳記。

#### 狀態追蹤
- 遊戲狀態和投注狀態都採用了明確的狀態欄位，便於追蹤和查詢。
- 使用 ENUM 或預定義的狀態值，確保狀態的一致性。 

### 資料存取模式

#### 常見查詢模式
- 獲取當前進行中的遊戲資訊
- 獲取特定遊戲的所有抽出球號
- 獲取特定玩家的投注記錄
- 計算特定遊戲的總投注金額和獲獎金額
- 查詢JP遊戲的獲勝者

#### 存取權限考量
- 建議為不同角色（管理員、荷官、玩家）設定不同的資料庫存取權限。
- 確保敏感操作（如修改遊戲狀態、設定獲勝者）只能由授權用戶執行。

### 擴展性考量

#### 未來可能的擴展
- 新增更多遊戲統計和分析表格
- 整合玩家忠誠度和獎勵系統
- 支援更多遊戲類型和變體
- 增強資料安全性和稽核功能

#### 擴展建議
- 採用分區策略（如按月或按遊戲ID分區）處理增長中的資料
- 考慮實施讀寫分離以提高效能
- 定期檢視和優化索引策略
- 實施資料歸檔機制，保持活動表格的精簡

### 資料安全考量

#### 安全建議
- 實施嚴格的存取控制，限制資料庫帳號權限
- 對敏感資料（如玩家資訊）實施加密
- 定期備份資料庫
- 記錄和監控敏感操作
- 實施資料庫安全掃描和稽核

### 效能優化

#### 優化策略
- 定期分析並優化慢查詢
- 考慮快取常用查詢結果
- 對大型表格實施分區策略
- 定期維護索引和統計資訊
- 監控資料庫負載和資源使用

### 結論

所設計的資料庫結構符合樂透開獎服務的需求，能夠有效地支援所有必要的操作和查詢。透過精心設計的表格結構、索引策略和關聯，資料庫能夠在確保資料完整性的同時提供良好的查詢效能。

提供的初始化 SQL 可直接用於設置開發、測試或生產環境，並包含了基本的系統配置資料，使系統能夠立即投入使用。