package gameflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// JackpotInfo 結構
type JackpotInfo struct {
	JackpotID        string    // JP ID
	Amount           int       // JP金額
	CombinationBalls []int     // 符合的球號組合
	CreatedAt        time.Time // 創建時間
}

// GameManager 遊戲流程管理器
type GameManager struct {
	repo              GameRepository
	logger            *zap.Logger
	sidePicker        *SidePicker
	stageMutex        sync.RWMutex                // 使用 RWMutex 代替 Mutex
	currentGames      map[string]*GameData        // 以房間ID為鍵的遊戲映射
	gameTimers        map[string]*time.Timer      // 以遊戲ID為鍵的計時器映射
	stageTimeDefaults map[GameStage]time.Duration // 各階段默認時間
	onGameCreated     func(string)                // 遊戲創建時的回調
	onGameCancelled   func(string, string)        // 遊戲取消時的回調
	onGameOver        func(string)                // 遊戲結束時的回調
	onBallDrawn       func(string, Ball)          // 球抽取事件的回調
	currentGame       *GameData                   // 向後兼容，使用默認房間的遊戲

	// 多房間相關配置
	defaultRoom    string   // 默認房間ID
	supportedRooms []string // 支持的房間ID列表
}

// NewGameManager 創建新的遊戲管理器
func NewGameManager(repo GameRepository, logger *zap.Logger) *GameManager {
	// 遊戲各階段默認時間設置
	stageTimeDefaults := map[GameStage]time.Duration{
		StagePreparation:                      24 * time.Hour,   // 準備階段默認24小時
		StageNewRound:                         10 * time.Second, // 新局階段默認10秒
		StageCardPurchaseOpen:                 10 * time.Minute, // 購買卡片開放階段默認10分鐘
		StageCardPurchaseClose:                10 * time.Second, // 購買卡片結束階段默認10秒
		StageDrawingStart:                     5 * time.Minute,  // 開始抽球階段默認5分鐘
		StageDrawingClose:                     10 * time.Second, // 結束抽球階段默認10秒
		StageExtraBallPrepare:                 10 * time.Second, // 額外球準備階段默認10秒
		StageExtraBallSideSelectBettingStart:  2 * time.Minute,  // 額外球選邊開始階段默認2分鐘
		StageExtraBallSideSelectBettingClosed: 10 * time.Second, // 額外球選邊結束階段默認10秒
		StageExtraBallWaitClaim:               30 * time.Second, // 額外球等待認領階段默認30秒
		StageExtraBallDrawingStart:            2 * time.Minute,  // 額外球開始抽取階段默認2分鐘
		StageExtraBallDrawingClose:            10 * time.Second, // 額外球結束抽取階段默認10秒
		StagePayoutSettlement:                 30 * time.Second, // 派彩結算階段默認30秒
		StageJackpotPreparation:               30 * time.Second, // JP準備階段默認30秒
		StageJackpotDrawingStart:              3 * time.Minute,  // JP開始抽球階段默認3分鐘
		StageJackpotDrawingClosed:             10 * time.Second, // JP結束抽球階段默認10秒
		StageJackpotSettlement:                30 * time.Second, // JP結算階段默認30秒
		StageDrawingLuckyBallsStart:           3 * time.Minute,  // 幸運號碼球開始抽取階段默認3分鐘
		StageDrawingLuckyBallsClosed:          10 * time.Second, // 幸運號碼球結束抽取階段默認10秒
		StageGameOver:                         1 * time.Hour,    // 遊戲結束階段默認1小時
	}

	return &GameManager{
		repo:              repo,
		logger:            logger.With(zap.String("component", "game_manager")),
		sidePicker:        NewSidePicker(),
		stageMutex:        sync.RWMutex{},
		currentGames:      make(map[string]*GameData),
		gameTimers:        make(map[string]*time.Timer),
		stageTimeDefaults: stageTimeDefaults,
		defaultRoom:       "SG01",
		supportedRooms:    []string{"SG01", "SG02"}, // 初始支持的房間列表
		onBallDrawn:       nil,                      // 初始化為空
	}
}

// Start 啟動遊戲管理器
func (m *GameManager) Start(ctx context.Context) error {
	m.logger.Info("啟動遊戲管理器")

	// 初始化所有支持的房間
	for _, roomID := range m.supportedRooms {
		if err := m.startRoomGameManager(ctx, roomID); err != nil {
			m.logger.Error("初始化房間遊戲管理器失敗",
				zap.String("roomID", roomID),
				zap.Error(err))
			return err
		}
	}

	// 向後兼容：將默認房間的遊戲賦值給 currentGame
	if game, exists := m.currentGames[m.defaultRoom]; exists {
		m.currentGame = game
	}

	return nil
}

// startRoomGameManager 初始化特定房間的遊戲管理器
func (m *GameManager) startRoomGameManager(ctx context.Context, roomID string) error {
	m.logger.Info("初始化房間遊戲管理器", zap.String("roomID", roomID))

	// 檢查並初始化幸運號碼
	luckyBalls, err := m.repo.GetLuckyBallsByRoom(ctx, roomID)
	if err != nil {
		// 檢查是否是「不存在」錯誤
		if strings.Contains(err.Error(), "does not exist") {
			m.logger.Warn("幸運號碼不存在，將創建新的幸運號碼", zap.String("roomID", roomID))
			// 繼續執行，會在下面創建新的幸運號碼
		} else {
			// 其他錯誤，返回
			m.logger.Error("獲取幸運號碼失敗", zap.String("roomID", roomID), zap.Error(err))
			return err
		}
	}

	// 如果沒有幸運號碼，則生成新的幸運號碼
	if luckyBalls == nil || len(luckyBalls) == 0 {
		m.logger.Info("未找到幸運號碼，生成新的幸運號碼", zap.String("roomID", roomID))
		luckyBalls, err = GenerateLuckyBalls()
		if err != nil {
			m.logger.Error("生成幸運號碼失敗", zap.String("roomID", roomID), zap.Error(err))
			return err
		}

		// 保存幸運號碼
		if err := m.repo.SaveLuckyBallsToRoom(ctx, roomID, luckyBalls); err != nil {
			m.logger.Error("保存幸運號碼失敗", zap.String("roomID", roomID), zap.Error(err))
			return err
		}
	}

	m.logger.Info("幸運號碼設定完成", zap.String("roomID", roomID), zap.Int("數量", len(luckyBalls)))

	// 檢查是否有未完成的遊戲
	currentGame, err := m.repo.GetCurrentGameByRoom(ctx, roomID)
	if err != nil {
		// 檢查是否是「不存在」錯誤
		if strings.Contains(err.Error(), "does not exist") {
			m.logger.Warn("當前遊戲不存在，將創建新遊戲", zap.String("roomID", roomID))
			// 繼續執行，會在下面創建新遊戲
			currentGame = nil
		} else {
			// 其他錯誤，返回
			m.logger.Error("獲取當前遊戲失敗", zap.String("roomID", roomID), zap.Error(err))
			return err
		}
	}

	// 如果有未完成的遊戲，則恢復該遊戲
	if currentGame != nil {
		m.logger.Info("發現未完成的遊戲，恢復此遊戲",
			zap.String("roomID", roomID),
			zap.String("gameID", currentGame.GameID),
			zap.String("stage", string(currentGame.CurrentStage)))

		// 存儲到當前遊戲映射
		m.currentGames[roomID] = currentGame

		// 根據當前階段設置計時器
		m.setupStageTimer(ctx, currentGame, roomID)
	} else {
		// 如果沒有未完成的遊戲，創建新遊戲
		m.logger.Info("創建新遊戲", zap.String("roomID", roomID))
		if _, err := m.CreateNewGameForRoom(ctx, roomID); err != nil {
			m.logger.Error("創建新遊戲失敗",
				zap.String("roomID", roomID),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// CreateNewGame 創建新遊戲（使用默認房間）
func (m *GameManager) CreateNewGame(ctx context.Context) (string, error) {
	return m.CreateNewGameForRoom(ctx, m.defaultRoom)
}

// CreateNewGameForRoom 為特定房間創建新遊戲
func (m *GameManager) CreateNewGameForRoom(ctx context.Context, roomID string) (string, error) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	// 檢查是否支持此房間
	if !m.isRoomSupported(roomID) {
		return "", NewGameFlowErrorWithFormat("ROOM_NOT_SUPPORTED",
			"不支持的房間ID: %s", roomID)
	}

	// 檢查是否有正在進行的遊戲
	if game, exists := m.currentGames[roomID]; exists &&
		game.CurrentStage != StagePreparation &&
		game.CurrentStage != StageGameOver {
		return "", NewGameFlowErrorWithFormat("GAME_IN_PROGRESS",
			"房間 %s 已有遊戲在進行中，無法創建新遊戲。當前遊戲ID: %s, 階段: %s",
			roomID, game.GameID, game.CurrentStage)
	}

	// 創建新遊戲，包含房間ID
	gameID := fmt.Sprintf("room_%s_game_%s", roomID, uuid.New().String())
	newGame := NewGameData(gameID, roomID)

	// 保存遊戲狀態
	if err := m.repo.SaveGame(ctx, newGame); err != nil {
		m.logger.Error("保存新遊戲失敗",
			zap.String("roomID", roomID),
			zap.String("gameID", gameID),
			zap.Error(err))
		return "", err
	}

	m.logger.Info("已創建新遊戲",
		zap.String("roomID", roomID),
		zap.String("gameID", gameID),
		zap.Int("extraBallCount", newGame.ExtraBallCount))

	// 更新當前遊戲映射
	m.currentGames[roomID] = newGame

	// 如果是默認房間，同時更新 currentGame 以保持向後兼容
	if roomID == m.defaultRoom {
		m.currentGame = newGame
	}

	// 觸發事件
	if m.onGameCreated != nil {
		m.onGameCreated(gameID)
	}

	return gameID, nil
}

// 檢查房間是否受支持
func (m *GameManager) isRoomSupported(roomID string) bool {
	for _, supportedRoom := range m.supportedRooms {
		if supportedRoom == roomID {
			return true
		}
	}
	return false
}

// GetCurrentGame 獲取當前遊戲（使用默認房間）
func (m *GameManager) GetCurrentGame() *GameData {
	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()

	return m.currentGame
}

// GetCurrentGameByRoom 獲取特定房間的當前遊戲
func (m *GameManager) GetCurrentGameByRoom(roomID string) *GameData {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if game, exists := m.currentGames[roomID]; exists {
		return game
	}
	return nil
}

// AdvanceStageForRoom 將特定房間的遊戲推進到下一階段
func (m *GameManager) AdvanceStageForRoom(ctx context.Context, roomID string, autoAdvance bool) error {
	m.stageMutex.Lock()

	// 獲取指定房間的遊戲
	game, exists := m.currentGames[roomID]
	if !exists || game == nil {
		m.stageMutex.Unlock()
		return ErrGameNotFound
	}

	// 記錄當前階段
	previousStage := game.CurrentStage

	// 如果是 StagePayoutSettlement 階段，進行幸運球檢查並決定下一階段
	var nextStage GameStage
	if previousStage == StagePayoutSettlement {
		// 從 Redis 獲取當前幸運球
		luckyBalls, err := m.repo.GetLuckyBallsByRoom(ctx, roomID)
		if err != nil {
			m.logger.Error("獲取幸運球失敗，使用標準流程轉換",
				zap.String("roomID", roomID),
				zap.String("gameID", game.GameID),
				zap.Error(err))
			nextStage = GetNextStage(previousStage, game.HasJackpot)
		} else {
			// 記錄幸運球號碼
			var luckyBallNumbers []int
			for _, ball := range luckyBalls {
				luckyBallNumbers = append(luckyBallNumbers, ball.Number)
			}

			// 記錄已抽出的球號碼
			var regularBallNumbers, extraBallNumbers []int
			for _, ball := range game.RegularBalls {
				regularBallNumbers = append(regularBallNumbers, ball.Number)
			}
			for _, ball := range game.ExtraBalls {
				extraBallNumbers = append(extraBallNumbers, ball.Number)
			}

			// 進行幸運球檢查
			var checkResult *LuckyBallCheckResult
			nextStage, checkResult = GetNextStageWithGameDetailed(previousStage, game, luckyBalls)

			// 記錄 JP 檢查結果
			if checkResult != nil && checkResult.AllMatched {
				m.logger.Info("檢測到 Jackpot 符合條件！",
					zap.String("roomID", roomID),
					zap.String("gameID", game.GameID),
					zap.Ints("luckyBalls", luckyBallNumbers),
					zap.Ints("regularBalls", regularBallNumbers),
					zap.Ints("extraBalls", extraBallNumbers),
					zap.Int("jackpotAmount", 500000)) // 使用默認值

				// 設置 JP 相關資訊到 Jackpot 字段
				if game.Jackpot == nil {
					game.Jackpot = &JackpotGame{
						ID:         fmt.Sprintf("jp_%s", uuid.New().String()),
						StartTime:  time.Now(),
						DrawnBalls: make([]Ball, 0),
						LuckyBalls: make([]Ball, 0),
						Amount:     500000,
						Active:     true,
					}
				}
				// 更新 JP 信息
				game.Jackpot.Active = true
				game.Jackpot.Amount = 500000
			} else {
				m.logger.Info("幸運球檢查完成，未符合 Jackpot 條件",
					zap.String("roomID", roomID),
					zap.String("gameID", game.GameID),
					zap.Ints("luckyBalls", luckyBallNumbers),
					zap.Ints("regularBalls", regularBallNumbers),
					zap.Ints("extraBalls", extraBallNumbers))
			}
		}
	} else {
		// 其他階段使用標準轉換
		nextStage = GetNextStage(previousStage, game.HasJackpot)
	}

	m.logger.Info("遊戲階段轉換",
		zap.String("roomID", roomID),
		zap.String("gameID", game.GameID),
		zap.String("from", string(previousStage)),
		zap.String("to", string(nextStage)),
		zap.Bool("autoAdvance", autoAdvance))

	// 更新遊戲階段
	game.CurrentStage = nextStage
	game.LastUpdateTime = time.Now()

	if nextStage == StageGameOver {
		// 遊戲結束，保存歷史記錄
		if err := m.repo.SaveGameHistory(ctx, game); err != nil {
			m.logger.Error("保存遊戲歷史記錄失敗",
				zap.String("roomID", roomID),
				zap.String("gameID", game.GameID),
				zap.Error(err))
			// 繼續執行，不要中斷遊戲流程
		}
	}

	// 保存狀態到緩存
	gameCopy := *game // 創建副本以避免競態條件
	if err := m.repo.SaveGame(ctx, &gameCopy); err != nil {
		m.stageMutex.Unlock()
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	// 設置階段計時器
	if autoAdvance && nextStage != StageGameOver {
		// 計算此階段的時間
		stageDuration := m.calculateStageDuration(nextStage)
		m.setupTimerForGame(ctx, game, roomID, stageDuration)
	}

	// 釋放鎖，因為我們已經完成了關鍵部分
	m.stageMutex.Unlock()

	m.logger.Info("階段已變更，完成推進",
		zap.String("roomID", roomID),
		zap.String("gameID", game.GameID),
		zap.String("from", string(previousStage)),
		zap.String("to", string(nextStage)))

	// 如果是 GameOver 階段，觸發事件並準備下一局
	if nextStage == StageGameOver {
		// 觸發遊戲結束事件
		if m.onGameOver != nil {
			m.onGameOver(game.GameID)
		}

		// 準備下一局
		if err := m.prepareForNextGameInRoom(ctx, roomID, autoAdvance); err != nil {
			m.logger.Error("準備下一局失敗",
				zap.String("roomID", roomID),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// setupStageTimer 根據遊戲階段設置計時器
func (m *GameManager) setupStageTimer(ctx context.Context, game *GameData, roomID string) {
	// 如果遊戲已結束或在準備階段，不需要設置計時器
	if game.CurrentStage == StageGameOver || game.CurrentStage == StagePreparation {
		m.logger.Debug("不為階段設置計時器",
			zap.String("roomID", roomID),
			zap.String("gameID", game.GameID),
			zap.String("stage", string(game.CurrentStage)))
		return
	}

	// 計算此階段的時間
	stageDuration := m.calculateStageDuration(game.CurrentStage)
	m.setupTimerForGame(ctx, game, roomID, stageDuration)
}

// setupTimerForGame 為特定遊戲設置計時器
func (m *GameManager) setupTimerForGame(ctx context.Context, game *GameData, roomID string, duration time.Duration) {
	// 創建定時器ID
	timerID := fmt.Sprintf("%s_%s", roomID, game.GameID)

	// 如果已存在計時器，先取消它
	if existingTimer, exists := m.gameTimers[timerID]; exists {
		existingTimer.Stop()
		delete(m.gameTimers, timerID)
	}

	// 創建新的計時器
	m.logger.Debug("為遊戲設置計時器",
		zap.String("roomID", roomID),
		zap.String("gameID", game.GameID),
		zap.String("stage", string(game.CurrentStage)),
		zap.Duration("duration", duration))

	timer := time.AfterFunc(duration, func() {
		// 創建獨立的上下文，不受原始請求上下文的影響
		timerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		m.logger.Info("階段計時器觸發，自動推進到下一階段",
			zap.String("roomID", roomID),
			zap.String("gameID", game.GameID),
			zap.String("stage", string(game.CurrentStage)))

		// 定義重試邏輯的函數
		retryAdvanceStage := func() error {
			// 添加重試邏輯
			maxRetries := 3
			var lastErr error

			for attempt := 1; attempt <= maxRetries; attempt++ {
				// 推進到下一階段
				if err := m.AdvanceStageForRoom(timerCtx, roomID, true); err != nil {
					lastErr = err
					m.logger.Warn("推進階段失敗，準備重試",
						zap.String("roomID", roomID),
						zap.String("gameID", game.GameID),
						zap.Int("attempt", attempt),
						zap.Int("maxRetries", maxRetries),
						zap.Error(err))

					// 如果不是最後一次嘗試，稍等一下再重試
					if attempt < maxRetries {
						time.Sleep(time.Duration(attempt*200) * time.Millisecond)
						continue
					}
					return fmt.Errorf("重試 %d 次後，推進階段仍然失敗: %w", maxRetries, lastErr)
				}
				// 成功推進階段，跳出循環
				m.logger.Info("成功推進階段",
					zap.String("roomID", roomID),
					zap.String("gameID", game.GameID),
					zap.Int("attempt", attempt))
				return nil
			}
			return lastErr
		}

		// 執行重試邏輯
		if err := retryAdvanceStage(); err != nil {
			m.logger.Error("經過多次重試後，自動推進階段失敗",
				zap.String("roomID", roomID),
				zap.String("gameID", game.GameID),
				zap.Error(err))
		}
	})

	// 存儲計時器
	m.gameTimers[timerID] = timer
}

// prepareForNextGameInRoom 為特定房間準備下一局遊戲
func (m *GameManager) prepareForNextGameInRoom(ctx context.Context, roomID string, autoStart bool) error {
	// 刪除當前遊戲
	if err := m.repo.DeleteCurrentGameByRoom(ctx, roomID); err != nil {
		m.logger.Error("刪除當前遊戲失敗",
			zap.String("roomID", roomID),
			zap.Error(err))
		// 這不是致命錯誤，繼續執行
	}

	// 清除當前遊戲
	m.stageMutex.Lock()
	delete(m.currentGames, roomID)

	// 如果是默認房間，同時更新 currentGame 以保持向後兼容
	if roomID == m.defaultRoom {
		m.currentGame = nil
	}

	m.stageMutex.Unlock()

	// 如果設置為自動開始，則創建新遊戲
	if autoStart {
		_, err := m.CreateNewGameForRoom(ctx, roomID)
		if err != nil {
			return fmt.Errorf("創建新遊戲失敗: %w", err)
		}
	}

	return nil
}

// SetSupportedRooms 設置支持的房間列表
func (m *GameManager) SetSupportedRooms(rooms []string) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	m.supportedRooms = make([]string, len(rooms))
	copy(m.supportedRooms, rooms)
}

// GetSupportedRooms 獲取支持的房間列表
func (m *GameManager) GetSupportedRooms() []string {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	rooms := make([]string, len(m.supportedRooms))
	copy(rooms, m.supportedRooms)
	return rooms
}

// SetDefaultRoom 設置默認房間
func (m *GameManager) SetDefaultRoom(roomID string) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	// 檢查該房間是否受支持
	if !m.isRoomSupported(roomID) {
		return NewGameFlowErrorWithFormat("ROOM_NOT_SUPPORTED",
			"無法設置為默認房間：不支持的房間ID: %s", roomID)
	}

	m.defaultRoom = roomID

	// 更新 currentGame 以保持向後兼容
	if game, exists := m.currentGames[roomID]; exists {
		m.currentGame = game
	} else {
		m.currentGame = nil
	}

	return nil
}

// GetCurrentStage 獲取當前遊戲階段
func (m *GameManager) GetCurrentStage() GameStage {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return StagePreparation
	}

	return m.currentGame.CurrentStage
}

// GetExtraBallSide 獲取當前選擇的額外球邊
func (m *GameManager) GetExtraBallSide() ExtraBallSide {
	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()

	if m.currentGame == nil {
		return ""
	}

	return m.currentGame.SelectedSide
}

// GetGameStatistics 獲取遊戲統計數據
func (m *GameManager) GetGameStatistics(ctx context.Context) (map[string]interface{}, error) {
	// 獲取總遊戲數和取消的遊戲數
	repo, ok := m.repo.(*CompositeRepository)
	if !ok {
		return nil, fmt.Errorf("儲存庫類型不支持此操作")
	}

	totalGames, err := repo.persistent.GetTotalGamesCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("獲取總遊戲數失敗: %w", err)
	}

	cancelledGames, err := repo.persistent.GetCancelledGamesCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("獲取取消遊戲數失敗: %w", err)
	}

	// 獲取當前遊戲狀態
	currentGame := m.GetCurrentGame()
	currentStage := "無遊戲"
	if currentGame != nil {
		currentStage = string(currentGame.CurrentStage)
	}

	return map[string]interface{}{
		"total_games":     totalGames,
		"cancelled_games": cancelledGames,
		"completion_rate": float64(totalGames-cancelledGames) / float64(totalGames) * 100,
		"current_stage":   currentStage,
	}, nil
}

// pushGameSnapshot 推送遊戲快照到 RocketMQ
func (m *GameManager) pushGameSnapshot(ctx context.Context) error {
	// 此處僅做日誌記錄，實際功能已被移除
	m.logger.Debug("pushGameSnapshot 被調用但功能已被移除")
	return nil
}

// GetOnBallDrawnCallback 獲取球抽取事件回調函數
func (m *GameManager) GetOnBallDrawnCallback() func(string, Ball) {
	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()
	return m.onBallDrawn
}

// SetOnBallDrawnCallback 設置球抽取事件回調函數
func (m *GameManager) SetOnBallDrawnCallback(callback func(string, Ball)) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()
	m.onBallDrawn = callback
}

// UpdateRegularBalls 直接更新指定房間遊戲的常規球陣列
func (m *GameManager) UpdateRegularBalls(ctx context.Context, roomID string, ball Ball) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	// 獲取指定房間的遊戲
	game, exists := m.currentGames[roomID]
	if !exists || game == nil {
		return ErrGameNotFound
	}

	// 確認當前階段允許抽取常規球
	if game.CurrentStage != StageDrawingStart {
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_REGULAR_BALL",
			"當前階段 %s 不允許更新常規球", game.CurrentStage)
	}

	// 驗證球號
	if err := ValidateBallNumber(ball.Number); err != nil {
		return err
	}

	// 檢查是否重複
	if IsBallDuplicate(ball.Number, game.RegularBalls) {
		return fmt.Errorf("重複的球號: %d", ball.Number)
	}

	// 設置球的類型為常規球
	ball.Type = BallTypeRegular

	// 添加球到遊戲數據
	game.RegularBalls = append(game.RegularBalls, ball)
	game.LastUpdateTime = time.Now()

	// 保存遊戲狀態
	if err := m.repo.SaveGame(ctx, game); err != nil {
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("已添加常規球",
		zap.String("roomID", roomID),
		zap.String("gameID", game.GameID),
		zap.Int("ballNumber", ball.Number))

	// 調用 onBallDrawn 回調（如果有的話）
	gameID := game.GameID
	if m.onBallDrawn != nil {
		go m.onBallDrawn(gameID, ball)
	}

	return nil
}

// UpdateExtraBalls 直接更新遊戲的額外球陣列
func (m *GameManager) UpdateExtraBalls(ctx context.Context, roomID string, ball Ball) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return ErrGameNotFound
	}

	// 確認當前階段允許抽取額外球
	if m.currentGame.CurrentStage != StageExtraBallDrawingStart {
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_EXTRA_BALL",
			"當前階段 %s 不允許更新額外球", m.currentGame.CurrentStage)
	}

	// 驗證球號
	if err := ValidateBallNumber(ball.Number); err != nil {
		return err
	}

	// 檢查是否重複
	if IsBallDuplicate(ball.Number, m.currentGame.ExtraBalls) {
		return fmt.Errorf("重複的球號: %d", ball.Number)
	}

	// 設置球的類型為額外球
	ball.Type = BallTypeExtra

	// 添加球到遊戲數據
	m.currentGame.ExtraBalls = append(m.currentGame.ExtraBalls, ball)
	m.currentGame.LastUpdateTime = time.Now()

	// 保存遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("已添加額外球",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballNumber", ball.Number))

	// 調用 onBallDrawn 回調（如果有的話）
	gameID := m.currentGame.GameID
	if m.onBallDrawn != nil {
		go m.onBallDrawn(gameID, ball)
	}

	// 如果額外球數量達到配置的數量，則自動前進到下一階段
	if len(m.currentGame.ExtraBalls) == m.currentGame.ExtraBallCount {
		go func() {
			// 創建新的上下文以避免使用已取消的上下文
			newCtx := context.Background()
			err := m.AdvanceStageForRoom(newCtx, roomID, true)
			if err != nil {
				m.logger.Error("自動前進到下一階段失敗",
					zap.String("gameID", gameID),
					zap.Error(err))
			}
		}()
	}

	return nil
}

// calculateStageDuration 計算指定階段的持續時間
func (m *GameManager) calculateStageDuration(stage GameStage) time.Duration {
	// 先嘗試從 stageTimeDefaults 獲取配置的時間
	if duration, exists := m.stageTimeDefaults[stage]; exists {
		return duration
	}

	// 如果沒有配置，從 GetStageConfig 獲取默認配置
	config := GetStageConfig(stage)
	if config.Timeout > 0 {
		return config.Timeout
	}

	// 如果 timeout 為 -1 或不存在配置，返回默認時間
	return 30 * time.Second
}

// NotifyGameStageChanged 通知遊戲階段已變更
func (m *GameManager) NotifyGameStageChanged(roomID, gameID, newStage string) {
	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()

	m.logger.Info("收到遊戲階段變更通知",
		zap.String("roomID", roomID),
		zap.String("gameID", gameID),
		zap.String("newStage", newStage))

	// 查找對應的遊戲
	game, exists := m.currentGames[roomID]
	if !exists || game == nil {
		m.logger.Warn("找不到指定的遊戲",
			zap.String("roomID", roomID),
			zap.String("gameID", gameID))
		return
	}

	m.logger.Info("遊戲階段已設置為",
		zap.String("roomID", roomID),
		zap.String("gameID", gameID),
		zap.String("oldStage", string(game.CurrentStage)),
		zap.String("newStage", newStage))
}

// GetAllOpenRooms 獲取所有開放的房間
func (m *GameManager) GetAllOpenRooms(ctx context.Context) []string {
	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()

	// 返回所有支持的房間
	return m.supportedRooms
}

// ResetGameForRoom 將特定房間的遊戲重置到初始狀態
// 這會保存當前遊戲到 TiDB，刪除舊遊戲並創建一個新的遊戲在 StagePreparation 階段
func (m *GameManager) ResetGameForRoom(ctx context.Context, roomID string) (*GameData, error) {
	m.stageMutex.Lock()

	// 獲取指定房間的當前遊戲
	game, exists := m.currentGames[roomID]
	if !exists || game == nil {
		m.stageMutex.Unlock()
		return nil, ErrGameNotFound
	}

	// 儲存當前遊戲到歷史記錄
	if err := m.repo.SaveGameHistory(ctx, game); err != nil {
		m.logger.Error("保存遊戲歷史記錄失敗",
			zap.String("roomID", roomID),
			zap.String("gameID", game.GameID),
			zap.Error(err))
		// 繼續執行，不要因為歷史記錄保存失敗而中斷操作
	}

	// 刪除當前遊戲
	if err := m.repo.DeleteCurrentGameByRoom(ctx, roomID); err != nil {
		m.logger.Error("刪除當前遊戲失敗",
			zap.String("roomID", roomID),
			zap.Error(err))
		// 繼續執行，不要因為刪除失敗而中斷操作
	}

	// 清除當前遊戲記錄
	delete(m.currentGames, roomID)

	// 如果是默認房間，同時更新 currentGame 以保持向後兼容
	if roomID == m.defaultRoom {
		m.currentGame = nil
	}

	// 釋放锁，以便創建新遊戲
	m.stageMutex.Unlock()

	// 創建新遊戲
	gameID, err := m.CreateNewGameForRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("重置遊戲時創建新遊戲失敗: %w", err)
	}

	// 再次獲取锁，以便讀取新創建的遊戲
	m.stageMutex.Lock()
	newGame := m.currentGames[roomID]
	m.stageMutex.Unlock()

	// 記錄重置操作
	m.logger.Info("已重置房間遊戲狀態",
		zap.String("roomID", roomID),
		zap.String("oldGameID", game.GameID),
		zap.String("newGameID", gameID))

	return newGame, nil
}
