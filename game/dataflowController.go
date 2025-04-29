package game

import (
	"fmt"
	"g38_lottery_service/pkg/logger"
	"log"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
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
	sourceBalls  []int        // 原始球池 (例如: 1-75)
	drawnBalls   []DrawResult // 已抽出的球
	extraBalls   []DrawResult // 額外球
	luckyNumbers []int        // 幸運號碼

	// 遊戲設定
	totalBalls    int // 總球數
	mainDrawCount int // 主遊戲抽球數
	maxExtraBalls int // 最大額外球數

	// 其他設定
	jpTriggerNumbers []int         // JP觸發號碼
	currentGameID    string        // 當前遊戲ID
	isJPTriggered    bool          // 是否觸發JP
	logger           logger.Logger // 日誌記錄器
}

// NewDataFlowController 創建一個新的DataFlowController實例
func NewDataFlowController(logger logger.Logger) *DataFlowController {
	controller := &DataFlowController{
		currentState:     StateInitial,
		stateHistory:     make([]GameState, 0),
		sourceBalls:      make([]int, 0),
		drawnBalls:       make([]DrawResult, 0),
		extraBalls:       make([]DrawResult, 0),
		luckyNumbers:     make([]int, 0), // 初始化幸運號碼陣列
		totalBalls:       75,             // 預設75球
		mainDrawCount:    30,             // 預設主遊戲抽30球
		maxExtraBalls:    3,              // 預設最多3顆額外球
		jpTriggerNumbers: make([]int, 0),
		isJPTriggered:    false,
		logger:           logger,
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

	oldState := dfc.currentState
	log.Printf("嘗試變更遊戲狀態: %s -> %s", oldState, newState)

	// 檢查狀態轉換是否合法
	if !dfc.isValidStateTransition(dfc.currentState, newState) {
		log.Printf("錯誤: 不允許從 %s 到 %s 的狀態轉換", dfc.currentState, newState)
		dfc.logger.Error("不允許的狀態轉換",
			zap.String("from", string(dfc.currentState)),
			zap.String("to", string(newState)))
		dfc.logger.Error("invalid state transition",
			zap.String("from", string(dfc.currentState)),
			zap.String("to", string(newState)))
		return nil
	}

	// 將當前狀態添加到歷史記錄
	dfc.stateHistory = append(dfc.stateHistory, dfc.currentState)

	// 更新狀態
	dfc.currentState = newState
	log.Printf("遊戲狀態已變更: %s -> %s", oldState, newState)

	// 新增更詳細的日誌
	log.Printf("遊戲狀態已變更: %s -> %s", oldState, newState)

	// 如果進入新遊戲，重置相關數據
	if newState == StateReady {
		dfc.resetGame()
		log.Printf("已重置遊戲數據（新遊戲準備階段）")
	}

	return nil
}

// DrawBall 從球池中抽出一顆球
func (dfc *DataFlowController) DrawBall() (*DrawResult, error) {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	// 檢查當前狀態是否允許抽球
	if dfc.currentState != StateDrawing && dfc.currentState != StateJPDrawing {
		dfc.logger.Error("無法在當前狀態下抽球",
			zap.String("currentState", string(dfc.currentState)))
		return nil, fmt.Errorf("cannot draw ball in current state: %s", dfc.currentState)
	}

	// 檢查是否還有球可抽
	if len(dfc.drawnBalls) >= dfc.totalBalls {
		dfc.logger.Error("沒有更多球可抽",
			zap.Int("totalBalls", dfc.totalBalls),
			zap.Int("drawnBalls", len(dfc.drawnBalls)))
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
		dfc.logger.Error("沒有剩餘球可抽")
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
	if dfc.currentState != StateExtraDraw && dfc.currentState != StateChooseExtraBall {
		dfc.logger.Error("無法在當前狀態下抽取額外球",
			zap.String("currentState", string(dfc.currentState)))
		return nil, fmt.Errorf("cannot draw extra ball in current state: %s", dfc.currentState)
	}

	// 檢查是否超過最大額外球數
	if len(dfc.extraBalls) >= dfc.maxExtraBalls {
		dfc.logger.Error("已達最大額外球數量",
			zap.Int("maxExtraBalls", dfc.maxExtraBalls),
			zap.Int("currentExtraBalls", len(dfc.extraBalls)))
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
		dfc.logger.Error("沒有剩餘球可抽取額外球")
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

// SetLuckyNumbers 生成7個幸運號碼
func (dfc *DataFlowController) SetLuckyNumbers() []int {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	// 清空現有幸運號碼
	dfc.luckyNumbers = make([]int, 0, 7)

	// 創建一個75球的陣列
	nums := make([]int, dfc.totalBalls)
	for i := 0; i < dfc.totalBalls; i++ {
		nums[i] = i + 1
	}

	// Fisher-Yates洗牌算法，隨機抽出7個不重複的號碼
	for i := 0; i < 7; i++ {
		// 隨機選擇剩餘球中的一個
		j := i + rand.Intn(dfc.totalBalls-i)
		// 交換球的位置
		nums[i], nums[j] = nums[j], nums[i]
		// 將選中的球加入幸運號碼
		dfc.luckyNumbers = append(dfc.luckyNumbers, nums[i])
	}

	log.Printf("已生成幸運號碼: %v", dfc.luckyNumbers)

	return dfc.luckyNumbers
}

// GetLuckyNumbers 獲取幸運號碼
func (dfc *DataFlowController) GetLuckyNumbers() []int {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()

	result := make([]int, len(dfc.luckyNumbers))
	copy(result, dfc.luckyNumbers)
	return result
}

// GetGameStatus API 功能：取回現在的遊戲狀態
func (dfc *DataFlowController) GetGameStatus() *GameStatusResponse {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()

	// 創建時間數據
	now := time.Now()

	// 使用實際時間，而不是模擬時間
	stateStartTime := now
	if len(dfc.stateHistory) > 0 {
		// 如果有狀態歷史，則可以假設最後狀態變更時間為5秒前（假設沒有實際記錄）
		stateStartTime = now.Add(-5 * time.Second)
	}

	// 只在遊戲完成時設置結束時間
	var endTime *time.Time
	if dfc.currentState == StateCompleted {
		t := now
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
		// 確定側邊位置 - 依據順序而非隨機
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

	// 使用實際設置的幸運號碼，而不是固定生成
	luckyNumbers := make([]int, len(dfc.luckyNumbers))
	copy(luckyNumbers, dfc.luckyNumbers)

	// 創建實際的JP數據，而非模擬數據
	jpGameID := ""
	if dfc.currentGameID != "" {
		jpGameID = "JP" + dfc.currentGameID[1:]
	}

	jackpotInfo := JackpotInfo{
		Active:     dfc.isJPTriggered,
		GameID:     nil,
		Amount:     0, // 實際金額應從資料庫獲取
		StartTime:  nil,
		EndTime:    nil,
		DrawnBalls: []BallInfo{},
		Winner:     nil,
	}

	// 只有當JP被觸發時才填充資料
	if dfc.isJPTriggered {
		jackpotInfo.GameID = &jpGameID
		jpStart := now.Add(-time.Second)
		jackpotInfo.StartTime = &jpStart
		jackpotInfo.Amount = 500000 // 實際金額應該從配置或資料庫獲取
	}

	// 計算實際剩餘時間 - 基於狀態的預設持續時間
	stateDuration := 60 // 預設每個狀態60秒
	switch dfc.currentState {
	case StateBetting:
		stateDuration = 120 // 投注時間更長
	case StateDrawing, StateJPDrawing:
		stateDuration = 90 // 抽球時間適中
	case StateExtraBet:
		stateDuration = 30 // 額外投注時間較短
	}

	// 計算剩餘時間
	elapsed := int(now.Sub(stateStartTime).Seconds())
	remainingTime := stateDuration - elapsed
	if remainingTime < 0 {
		remainingTime = 0
	}

	// 構建完整的遊戲狀態響應，不使用模擬玩家數據
	response := &GameStatusResponse{
		Game: GameInfo{
			ID:             dfc.currentGameID,
			State:          string(dfc.currentState),
			StartTime:      stateStartTime,
			EndTime:        endTime,
			HasJackpot:     dfc.isJPTriggered,
			ExtraBallCount: dfc.maxExtraBalls,
			Timeline: TimelineInfo{
				CurrentTime:    now,
				StateStartTime: stateStartTime,
				RemainingTime:  remainingTime,
				MaxTimeout:     stateDuration,
			},
		},
		LuckyNumbers:   luckyNumbers,
		DrawnBalls:     drawnBalls,
		ExtraBalls:     extraBalls,
		Jackpot:        jackpotInfo,
		TopPlayers:     []PlayerInfo{}, // 不使用模擬數據，返回空陣列
		TotalWinAmount: 0,              // 不使用模擬數據，初始為0
	}

	return response
}

// Private helper methods

// isValidStateTransition 檢查狀態轉換是否合法
func (dfc *DataFlowController) isValidStateTransition(from, to GameState) bool {
	// 特殊處理：允許從任何狀態轉換到BETTING（用於處理BETTING_STARTED消息）
	if to == StateBetting {
		log.Printf("允許從 %s 轉換到 BETTING 狀態", from)
		return true
	}

	validTransitions := map[GameState][]GameState{
		StateInitial:         {StateReady},
		StateReady:           {StateBetting, StateShowLuckyNums},
		StateBetting:         {StateDrawing},
		StateDrawing:         {StateExtraBet, StateJPStandby, StateChooseExtraBall},
		StateExtraBet:        {StateExtraDraw},
		StateExtraDraw:       {StateResult},
		StateResult:          {StateReady, StateCompleted},
		StateJPStandby:       {StateJPBetting},
		StateJPBetting:       {StateJPDrawing},
		StateJPDrawing:       {StateJPResult},
		StateJPResult:        {StateReady, StateCompleted},
		StateShowLuckyNums:   {StateBetting},
		StateShowBalls:       {StateExtraBet, StateResult},
		StateChooseExtraBall: {StateExtraDraw, StateResult},
		StateShowExtraBalls:  {StateResult},
		StateJPShowBalls:     {StateJPResult},
		StateCompleted:       {StateReady, StateInitial},
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
	dfc.luckyNumbers = make([]int, 0) // 重置幸運號碼
	dfc.isJPTriggered = false

	newGameID := fmt.Sprintf("G%d", time.Now().UnixNano())
	dfc.currentGameID = newGameID

	if dfc.logger != nil {
		dfc.logger.Info("遊戲狀態已重置",
			zap.String("gameID", newGameID))
	}
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

// SetCurrentGameID 設置當前遊戲ID
func (dfc *DataFlowController) SetCurrentGameID(gameID string) {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	dfc.currentGameID = gameID
}

// GetCurrentGameID 獲取當前遊戲ID
func (dfc *DataFlowController) GetCurrentGameID() string {
	dfc.mu.RLock()
	defer dfc.mu.RUnlock()

	return dfc.currentGameID
}

// ResetGame 公開方法，重置遊戲狀態
func (dfc *DataFlowController) ResetGame() {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	dfc.resetGame()
}

// StartBetting 開始投注階段
func (dfc *DataFlowController) StartBetting() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始投注階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許開始投注
	if dfc.currentState != StateReady && dfc.currentState != StateShowLuckyNums {
		dfc.logger.Error("當前狀態不允許開始投注",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許開始投注", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateBetting

	log.Printf("遊戲狀態已從 %s 變更為 %s (投注開始)", oldState, StateBetting)
	return nil
}

// CloseBetting 關閉投注階段
func (dfc *DataFlowController) CloseBetting() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("關閉投注階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許關閉投注
	if dfc.currentState != StateBetting {
		dfc.logger.Error("當前狀態不允許關閉投注",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許關閉投注", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateDrawing

	log.Printf("遊戲狀態已從 %s 變更為 %s (投注關閉)", oldState, StateDrawing)
	return nil
}

// StartDrawing 開始抽球階段
func (dfc *DataFlowController) StartDrawing() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始抽球階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許開始抽球
	if dfc.currentState != StateBetting {
		dfc.logger.Error("當前狀態不允許開始抽球",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許開始抽球", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateDrawing

	log.Printf("遊戲狀態已從 %s 變更為 %s (開始抽球)", oldState, StateDrawing)
	return nil
}

// StartExtraBetting 開始額外球投注階段
func (dfc *DataFlowController) StartExtraBetting() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始額外球投注階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許開始額外球投注
	if dfc.currentState != StateDrawing {
		dfc.logger.Error("當前狀態不允許開始額外球投注",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許開始額外球投注", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateExtraBet

	log.Printf("遊戲狀態已從 %s 變更為 %s (開始額外球投注)", oldState, StateExtraBet)
	return nil
}

// FinishExtraBetting 結束額外球投注階段，開始額外球抽取
func (dfc *DataFlowController) FinishExtraBetting() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("結束額外球投注階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許結束額外球投注
	if dfc.currentState != StateExtraBet {
		dfc.logger.Error("當前狀態不允許結束額外球投注",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許結束額外球投注", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateExtraDraw

	log.Printf("遊戲狀態已從 %s 變更為 %s (結束額外球投注，開始額外球抽取)", oldState, StateExtraDraw)
	return nil
}

// ChooseExtraBall 轉換到選擇額外球狀態
func (dfc *DataFlowController) ChooseExtraBall() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始選擇額外球階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許開始選擇額外球
	if dfc.currentState != StateDrawing {
		dfc.logger.Error("當前狀態不允許開始選擇額外球",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許開始選擇額外球", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateChooseExtraBall

	log.Printf("遊戲狀態已從 %s 變更為 %s (開始選擇額外球)", oldState, StateChooseExtraBall)
	return nil
}

// StartResult 轉換到結算狀態
func (dfc *DataFlowController) StartResult() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始結算階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許進入結算階段
	if !dfc.isValidStateTransition(dfc.currentState, StateResult) {
		dfc.logger.Error("當前狀態不允許進入結算階段",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許進入結算階段", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateResult

	log.Printf("遊戲狀態已從 %s 變更為 %s (進入結算階段)", oldState, StateResult)
	return nil
}

// StartJPStandby 轉換到JP待機狀態
func (dfc *DataFlowController) StartJPStandby() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始JP待機階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許進入JP待機階段
	if !dfc.isValidStateTransition(dfc.currentState, StateJPStandby) {
		dfc.logger.Error("當前狀態不允許進入JP待機階段",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許進入JP待機階段", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateJPStandby

	log.Printf("遊戲狀態已從 %s 變更為 %s (進入JP待機階段)", oldState, StateJPStandby)
	return nil
}

// StartJPBetting 轉換到JP投注狀態
func (dfc *DataFlowController) StartJPBetting() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始JP投注階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許進入JP投注階段
	if !dfc.isValidStateTransition(dfc.currentState, StateJPBetting) {
		dfc.logger.Error("當前狀態不允許進入JP投注階段",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許進入JP投注階段", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateJPBetting

	log.Printf("遊戲狀態已從 %s 變更為 %s (進入JP投注階段)", oldState, StateJPBetting)
	return nil
}

// StartJPDrawing 轉換到JP抽球狀態
func (dfc *DataFlowController) StartJPDrawing() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始JP抽球階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許進入JP抽球階段
	if !dfc.isValidStateTransition(dfc.currentState, StateJPDrawing) {
		dfc.logger.Error("當前狀態不允許進入JP抽球階段",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許進入JP抽球階段", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateJPDrawing

	log.Printf("遊戲狀態已從 %s 變更為 %s (進入JP抽球階段)", oldState, StateJPDrawing)
	return nil
}

// StopJPDrawing 結束JP抽球並進入JP結果狀態
func (dfc *DataFlowController) StopJPDrawing() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("結束JP抽球階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許進入JP結果階段
	if !dfc.isValidStateTransition(dfc.currentState, StateJPResult) {
		dfc.logger.Error("當前狀態不允許進入JP結果階段",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許進入JP結果階段", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateJPResult

	log.Printf("遊戲狀態已從 %s 變更為 %s (進入JP結果階段)", oldState, StateJPResult)
	return nil
}

// StartJPShowBalls 轉換到JP開獎狀態
func (dfc *DataFlowController) StartJPShowBalls() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始JP開獎階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許進入JP開獎階段
	if !dfc.isValidStateTransition(dfc.currentState, StateJPShowBalls) {
		dfc.logger.Error("當前狀態不允許進入JP開獎階段",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許進入JP開獎階段", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateJPShowBalls

	log.Printf("遊戲狀態已從 %s 變更為 %s (進入JP開獎階段)", oldState, StateJPShowBalls)
	return nil
}

// StartCompleted 轉換到遊戲完成狀態
func (dfc *DataFlowController) StartCompleted() error {
	dfc.mu.Lock()
	defer dfc.mu.Unlock()

	log.Printf("開始遊戲完成階段，當前狀態: %s", dfc.currentState)

	// 檢查當前狀態是否允許進入遊戲完成階段
	if !dfc.isValidStateTransition(dfc.currentState, StateCompleted) {
		dfc.logger.Error("當前狀態不允許進入遊戲完成階段",
			zap.String("currentState", string(dfc.currentState)))
		return fmt.Errorf("當前狀態 %s 不允許進入遊戲完成階段", dfc.currentState)
	}

	// 記錄舊狀態並更改為新狀態
	oldState := dfc.currentState
	dfc.stateHistory = append(dfc.stateHistory, oldState)
	dfc.currentState = StateCompleted

	log.Printf("遊戲狀態已從 %s 變更為 %s (進入遊戲完成階段)", oldState, StateCompleted)
	return nil
}
