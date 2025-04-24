package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// 在 package 級別初始化隨機數生成器
func init() {
	// 在 Go 1.20+ 中不需要手動設置種子，但為了向後兼容
	rand.Seed(time.Now().UnixNano())
}

// GameState 代表遊戲的不同狀態
type GameState string

const (
	StateInitial         GameState = "INITIAL"           // 初始狀態
	StateStandby         GameState = "STANDBY"           // 待機狀態
	StateReady           GameState = "READY"             // 待機狀態 對到 GAME_READY 等待開局
	StateShowLuckyNums   GameState = "SHOW_LUCKYNUMS"    // 開七個幸運球的狀態
	StateBetting         GameState = "BETTING"           // 投注狀態
	StateDrawing         GameState = "DRAWING"           // 抽球狀態
	StateShowBalls       GameState = "SHOW_BALLS"        // 開獎狀態
	StateExtraBet        GameState = "EXTRA_BET"         // 額外球投注狀態
	StateExtraDraw       GameState = "EXTRA_DRAW"        // 額外球抽取狀態
	StateChooseExtraBall GameState = "CHOOSE_EXTRA_BALL" // 額外球投注狀態 對到GAME_CHOOSE_EXTRA_BALL
	StateShowExtraBalls  GameState = "SHOW_EXTRA_BALLS"  // 額外球開獎狀態
	StateResult          GameState = "RESULT"            // 結算狀態
	StateJPStandby       GameState = "JP_STANDBY"        // JP待機狀態
	StateJPBetting       GameState = "JP_BETTING"        // JP投注狀態
	StateJPDrawing       GameState = "JP_DRAWING"        // JP抽球狀態
	StateJPResult        GameState = "JP_RESULT"         // JP結算狀態
	StateJPShowBalls     GameState = "JP_SHOW_BALLS"     // JP開獎狀態
	StateCompleted       GameState = "COMPLETED"         // 遊戲完成狀態
)

// DrawResult 代表抽球的結果
type DrawResult struct {
	BallNumber int       `json:"ball_number"`
	DrawTime   time.Time `json:"draw_time"`
	OrderIndex int       `json:"order_index"`
}

// DataFlowController 控制遊戲流程和狀態
type DataFlowController struct {
	mu sync.RWMutex

	// 遊戲狀態管理
	currentState GameState   // 當前遊戲狀態
	stateHistory []GameState // 狀態歷史記錄

	// 球池管理
	sourceBalls []int        // 原始球池 (例如: 1-75)
	drawnBalls  []DrawResult // 已抽出的球
	extraBalls  []DrawResult // 額外球

	// 遊戲設定
	totalBalls    int // 總球數
	mainDrawCount int // 主遊戲抽球數
	maxExtraBalls int // 最大額外球數

	// 其他設定
	jpTriggerNumbers []int  // JP觸發號碼
	currentGameID    string // 當前遊戲ID
	isJPTriggered    bool   // 是否觸發JP
}

// NewDataFlowController 創建一個新的DataFlowController實例
func NewDataFlowController() *DataFlowController {
	controller := &DataFlowController{
		currentState:     StateInitial,
		stateHistory:     make([]GameState, 0),
		sourceBalls:      make([]int, 0),
		drawnBalls:       make([]DrawResult, 0),
		extraBalls:       make([]DrawResult, 0),
		totalBalls:       75, // 預設75球
		mainDrawCount:    30, // 預設主遊戲抽30球
		maxExtraBalls:    3,  // 預設最多3顆額外球
		jpTriggerNumbers: make([]int, 0),
		isJPTriggered:    false,
	}

	controller.initializeBallPool()
	return controller
}

// initializeBallPool 初始化球池
func (dfc *DataFlowController) initializeBallPool() {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	dfc.sourceBalls = make([]int, dfc.totalBalls)
	for i := 0; i < dfc.totalBalls; i++ {
		dfc.sourceBalls[i] = i + 1
	}
}

// ChangeState 改變遊戲狀態
func (dfc *DataFlowController) ChangeState(newState GameState) error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	// 檢查狀態轉換是否合法
	if !dfc.isValidStateTransition(dfc.currentState, newState) {
		return fmt.Errorf("invalid state transition from %s to %s", dfc.currentState, newState)
	}

	dfc.stateHistory = append(dfc.stateHistory, dfc.currentState)
	dfc.currentState = newState

	// 如果進入新遊戲，重置相關數據
	if newState == StateStandby {
		dfc.resetGame()
	}

	return nil
}

// DrawBall 從球池中抽出一顆球
func (dfc *DataFlowController) DrawBall() (*DrawResult, error) {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	// 檢查當前狀態是否允許抽球
	if dfc.currentState != StateDrawing && dfc.currentState != StateJPDrawing {
		return nil, fmt.Errorf("cannot draw ball in current state: %s", dfc.currentState)
	}

	// 檢查是否還有球可抽
	if len(dfc.drawnBalls) >= dfc.totalBalls {
		return nil, fmt.Errorf("no more balls available")
	}

	// 計算剩餘可抽的球
	remainingBalls := make([]int, 0)
	for _, ball := range dfc.sourceBalls {
		found := false
		for _, drawn := range dfc.drawnBalls {
			if drawn.BallNumber == ball {
				found = true
				break
			}
		}
		if !found {
			remainingBalls = append(remainingBalls, ball)
		}
	}

	if len(remainingBalls) == 0 {
		return nil, fmt.Errorf("no more balls remaining")
	}

	// 隨機抽一顆球
	selectedBall := remainingBalls[rand.Intn(len(remainingBalls))]

	// 創建抽球結果
	result := DrawResult{
		BallNumber: selectedBall,
		DrawTime:   time.Now(),
		OrderIndex: len(dfc.drawnBalls) + 1,
	}

	dfc.drawnBalls = append(dfc.drawnBalls, result)

	// 檢查是否匹配JP觸發號碼
	if dfc.currentState == StateDrawing {
		dfc.checkJPTrigger(selectedBall)
	}

	return &result, nil
}

// DrawExtraBall 抽取額外球
func (dfc *DataFlowController) DrawExtraBall() (*DrawResult, error) {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	// 檢查當前狀態是否允許抽額外球
	if dfc.currentState != StateExtraDraw {
		return nil, fmt.Errorf("cannot draw extra ball in current state: %s", dfc.currentState)
	}

	// 檢查是否超過最大額外球數
	if len(dfc.extraBalls) >= dfc.maxExtraBalls {
		return nil, fmt.Errorf("maximum extra balls reached")
	}

	// 計算剩餘可抽的球（主球和額外球都需要排除）
	usedBalls := make(map[int]bool)
	for _, drawn := range dfc.drawnBalls {
		usedBalls[drawn.BallNumber] = true
	}
	for _, extra := range dfc.extraBalls {
		usedBalls[extra.BallNumber] = true
	}

	remainingBalls := make([]int, 0)
	for _, ball := range dfc.sourceBalls {
		if !usedBalls[ball] {
			remainingBalls = append(remainingBalls, ball)
		}
	}

	if len(remainingBalls) == 0 {
		return nil, fmt.Errorf("no more balls remaining for extra draw")
	}

	// 隨機抽一顆額外球
	selectedBall := remainingBalls[rand.Intn(len(remainingBalls))]

	// 創建額外球結果
	result := DrawResult{
		BallNumber: selectedBall,
		DrawTime:   time.Now(),
		OrderIndex: len(dfc.extraBalls) + 1,
	}

	dfc.extraBalls = append(dfc.extraBalls, result)

	return &result, nil
}

// GetCurrentState 獲取當前遊戲狀態
func (dfc *DataFlowController) GetCurrentState() GameState {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()
	return dfc.currentState
}

// GetDrawnBalls 獲取已抽出的球
func (dfc *DataFlowController) GetDrawnBalls() []DrawResult {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()

	result := make([]DrawResult, len(dfc.drawnBalls))
	copy(result, dfc.drawnBalls)
	return result
}

// GetExtraBalls 獲取額外球
func (dfc *DataFlowController) GetExtraBalls() []DrawResult {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()

	result := make([]DrawResult, len(dfc.extraBalls))
	copy(result, dfc.extraBalls)
	return result
}

// SetJPTriggerNumbers 設置JP觸發號碼
func (dfc *DataFlowController) SetJPTriggerNumbers(numbers []int) {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	dfc.jpTriggerNumbers = make([]int, len(numbers))
	copy(dfc.jpTriggerNumbers, numbers)
}

// VerifyTwoBalls API 功能：驗證前端輸入的兩顆球
func (dfc *DataFlowController) VerifyTwoBalls(ball1, ball2 int) bool {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()

	// 檢查球號是否在有效範圍內
	if ball1 < 1 || ball1 > dfc.totalBalls || ball2 < 1 || ball2 > dfc.totalBalls {
		return false
	}

	// 檢查是否為不同的球
	if ball1 == ball2 {
		return false
	}

	return true
}

// GetGameStatus API 功能：取回現在的遊戲狀態
func (dfc *DataFlowController) GetGameStatus() *GameStatusResponse {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()

	// 創建時間數據
	now := time.Now()

	// 模擬狀態開始時間（實際應用中應該記錄狀態變更時間）
	stateStartTime := now.Add(-time.Duration(rand.Intn(300)) * time.Second)

	// 創建一個模擬的結束時間（如果遊戲已結束）
	var endTime *time.Time
	if dfc.currentState == StateCompleted {
		t := now.Add(-time.Minute)
		endTime = &t
	}

	// 將DrawResult轉換為BallInfo
	drawnBalls := make([]BallInfo, 0, len(dfc.drawnBalls))
	for _, ball := range dfc.drawnBalls {
		drawnBalls = append(drawnBalls, BallInfo{
			Number:    ball.BallNumber,
			DrawnTime: ball.DrawTime,
			Sequence:  ball.OrderIndex,
		})
	}

	// 將額外球DrawResult轉換為ExtraBall
	extraBalls := make([]ExtraBall, 0, len(dfc.extraBalls))
	for i, ball := range dfc.extraBalls {
		// 模擬左右位置
		side := "LEFT"
		if i%2 == 1 {
			side = "RIGHT"
		}

		extraBalls = append(extraBalls, ExtraBall{
			Number:    ball.BallNumber,
			DrawnTime: ball.DrawTime,
			Sequence:  ball.OrderIndex,
			Side:      side,
		})
	}

	// 創建模擬的幸運數字
	luckyNumbers := make([]int, 0, 7)
	for i := 0; i < 7; i++ {
		luckyNumbers = append(luckyNumbers, 1+rand.Intn(75))
	}

	// 創建模擬的JP數據
	jackpotInfo := JackpotInfo{
		Active:     dfc.isJPTriggered,
		GameID:     nil,
		Amount:     500000.0,
		StartTime:  nil,
		EndTime:    nil,
		DrawnBalls: []BallInfo{},
		Winner:     nil,
	}

	// 如果JP被觸發，添加更多數據
	if dfc.isJPTriggered {
		jpGameID := "JP" + dfc.currentGameID[1:]
		jpStart := now.Add(-time.Minute * 5)
		jackpotInfo.GameID = &jpGameID
		jackpotInfo.StartTime = &jpStart

		// 如果是JP結果狀態，添加結束時間和獲勝者
		if dfc.currentState == StateJPResult {
			jpEnd := now.Add(-time.Minute)
			winner := "U" + fmt.Sprintf("%d", rand.Intn(999999))
			jackpotInfo.EndTime = &jpEnd
			jackpotInfo.Winner = &winner
		}
	}

	// 創建模擬玩家數據
	topPlayers := []PlayerInfo{
		{
			UserID:    "U123456",
			Nickname:  "幸運星",
			WinAmount: 25000,
			BetAmount: 5000,
			Cards:     3,
		},
		{
			UserID:    "U789012",
			Nickname:  "好運連連",
			WinAmount: 18500,
			BetAmount: 3500,
			Cards:     2,
		},
		{
			UserID:    "U345678",
			Nickname:  "財神到",
			WinAmount: 12000,
			BetAmount: 2000,
			Cards:     1,
		},
	}

	// 計算剩餘時間
	remainingTime := 60 - int(now.Sub(stateStartTime).Seconds())
	if remainingTime < 0 {
		remainingTime = 0
	}

	// 構建完整的遊戲狀態響應
	response := &GameStatusResponse{
		Game: GameInfo{
			ID:             dfc.currentGameID,
			State:          string(dfc.currentState),
			StartTime:      stateStartTime.Add(-time.Minute * 5),
			EndTime:        endTime,
			HasJackpot:     dfc.isJPTriggered,
			ExtraBallCount: dfc.maxExtraBalls,
			Timeline: TimelineInfo{
				CurrentTime:    now,
				StateStartTime: stateStartTime,
				RemainingTime:  remainingTime,
				MaxTimeout:     60,
			},
		},
		LuckyNumbers:   luckyNumbers,
		DrawnBalls:     drawnBalls,
		ExtraBalls:     extraBalls,
		Jackpot:        jackpotInfo,
		TopPlayers:     topPlayers,
		TotalWinAmount: 75800.0,
	}

	return response
}

// Private helper methods

// isValidStateTransition 檢查狀態轉換是否合法
func (dfc *DataFlowController) isValidStateTransition(from, to GameState) bool {
	validTransitions := map[GameState][]GameState{
		StateInitial:   {StateStandby, StateReady},
		StateStandby:   {StateBetting},
		StateReady:     {StateBetting},
		StateBetting:   {StateDrawing},
		StateDrawing:   {StateExtraBet, StateJPStandby},
		StateExtraBet:  {StateExtraDraw},
		StateExtraDraw: {StateResult},
		StateResult:    {StateStandby, StateCompleted},
		StateJPStandby: {StateJPBetting},
		StateJPBetting: {StateJPDrawing},
		StateJPDrawing: {StateJPResult},
		StateJPResult:  {StateStandby, StateCompleted},
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}

	return false
}

// resetGame 重置遊戲狀態
func (dfc *DataFlowController) resetGame() {
	dfc.drawnBalls = make([]DrawResult, 0)
	dfc.extraBalls = make([]DrawResult, 0)
	dfc.isJPTriggered = false
	dfc.currentGameID = fmt.Sprintf("G%d", time.Now().UnixNano())
}

// checkJPTrigger 檢查是否觸發JP
func (dfc *DataFlowController) checkJPTrigger(ballNumber int) {
	if len(dfc.jpTriggerNumbers) == 0 {
		return
	}

	matchedCount := 0
	for _, triggerNum := range dfc.jpTriggerNumbers {
		for _, drawnBall := range dfc.drawnBalls {
			if drawnBall.BallNumber == triggerNum {
				matchedCount++
				break
			}
		}
	}

	// 如果所有JP觸發號碼都被抽中
	if matchedCount == len(dfc.jpTriggerNumbers) {
		dfc.isJPTriggered = true
	}
}
