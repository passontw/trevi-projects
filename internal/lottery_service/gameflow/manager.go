package gameflow

import (
	"context"
	"fmt"
	"g38_lottery_service/internal/lottery_service/mq"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebSocketNotifier 接口
type WebSocketNotifier interface {
	OnStageChanged(gameID string, oldStage, newStage GameStage, game *GameData)
}

// GameManager 遊戲流程管理器
type GameManager struct {
	repo            GameRepository         // 資料儲存庫介面
	logger          *zap.Logger            // 日誌記錄器
	sidePicker      *SidePicker            // 選邊器
	currentGame     *GameData              // 當前遊戲
	stageMutex      sync.RWMutex           // 階段讀寫鎖
	stageTimers     map[string]*time.Timer // 階段計時器
	stageTimerMutex sync.Mutex             // 計時器鎖
	mqProducer      *mq.MessageProducer    // RocketMQ 生產者
	wsNotifier      WebSocketNotifier      // WebSocket 通知器

	// 事件處理回調
	onStageChanged          func(gameID string, oldStage, newStage GameStage) // 階段變更回調
	onGameCreated           func(gameID string)                               // 遊戲創建回調
	onGameCancelled         func(gameID string, reason string)                // 遊戲取消回調
	onGameCompleted         func(gameID string)                               // 遊戲完成回調
	onBallDrawn             func(gameID string, ball Ball)                    // 球抽取回調
	onExtraBallSideSelected func(gameID string, side ExtraBallSide)           // 額外球選邊回調
}

// NewGameManager 創建新的遊戲流程管理器
func NewGameManager(repo GameRepository, logger *zap.Logger, mqProducer *mq.MessageProducer) *GameManager {
	return &GameManager{
		repo:        repo,
		logger:      logger.With(zap.String("component", "game_manager")),
		sidePicker:  NewSidePicker(),
		stageTimers: make(map[string]*time.Timer),
		mqProducer:  mqProducer,
	}
}

// Start 啟動遊戲管理器，初始化系統狀態
func (m *GameManager) Start(ctx context.Context) error {
	m.logger.Info("啟動遊戲管理器")

	// 檢查並初始化幸運號碼
	luckyBalls, err := m.repo.GetLuckyBalls(ctx)
	if err != nil {
		// 檢查是否是「不存在」錯誤
		if strings.Contains(err.Error(), "does not exist") {
			m.logger.Warn("幸運號碼不存在，將創建新的幸運號碼")
			// 繼續執行，會在下面創建新的幸運號碼
		} else {
			// 其他錯誤，返回
			m.logger.Error("獲取幸運號碼失敗", zap.Error(err))
			return err
		}
	}

	// 如果沒有幸運號碼，則生成新的幸運號碼
	if luckyBalls == nil || len(luckyBalls) == 0 {
		m.logger.Info("未找到幸運號碼，生成新的幸運號碼")
		luckyBalls, err = GenerateLuckyBalls()
		if err != nil {
			m.logger.Error("生成幸運號碼失敗", zap.Error(err))
			return err
		}

		// 保存幸運號碼
		if err := m.repo.SaveLuckyBalls(ctx, luckyBalls); err != nil {
			m.logger.Error("保存幸運號碼失敗", zap.Error(err))
			return err
		}
	}

	m.logger.Info("幸運號碼設定完成", zap.Int("數量", len(luckyBalls)))

	// 檢查是否有未完成的遊戲
	currentGame, err := m.repo.GetCurrentGame(ctx)
	if err != nil {
		// 檢查是否是「不存在」錯誤
		if strings.Contains(err.Error(), "does not exist") {
			m.logger.Warn("當前遊戲不存在，將創建新遊戲")
			// 繼續執行，會在下面創建新遊戲
			currentGame = nil
		} else {
			// 其他錯誤，返回
			m.logger.Error("獲取當前遊戲失敗", zap.Error(err))
			return err
		}
	}

	// 如果有未完成的遊戲，則恢復該遊戲
	if currentGame != nil {
		m.logger.Info("發現未完成的遊戲，恢復遊戲狀態",
			zap.String("gameID", currentGame.GameID),
			zap.String("stage", string(currentGame.CurrentStage)))

		m.stageMutex.Lock()
		m.currentGame = currentGame
		m.stageMutex.Unlock()

		// 設定階段計時器
		if err := m.setupStageTimer(ctx, currentGame.GameID, currentGame.CurrentStage); err != nil {
			m.logger.Error("設定階段計時器失敗", zap.Error(err))
			return err
		}
	} else {
		// 沒有未完成的遊戲，初始化為準備階段
		m.logger.Info("未發現未完成的遊戲，設定為準備階段")
		newGameID := fmt.Sprintf("game_%s", uuid.New().String())
		newGame := NewGameData(newGameID)

		m.stageMutex.Lock()
		m.currentGame = newGame
		m.stageMutex.Unlock()

		// 保存初始狀態
		if err := m.repo.SaveGame(ctx, newGame); err != nil {
			m.logger.Error("保存初始遊戲狀態失敗", zap.Error(err))
			return err
		}
	}

	return nil
}

// GetCurrentGame 獲取當前遊戲狀態
func (m *GameManager) GetCurrentGame() *GameData {
	m.logger.Info("GetCurrentGame-開始獲取當前遊戲狀態")

	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()

	if m.currentGame == nil {
		m.logger.Info("GetCurrentGame-當前沒有遊戲")
		return nil
	}

	// 返回一個複本，避免外部修改
	gameCopy := *m.currentGame

	m.logger.Info("GetCurrentGame-成功獲取遊戲狀態",
		zap.String("gameID", gameCopy.GameID),
		zap.String("stage", string(gameCopy.CurrentStage)),
		zap.Int("extraBallCount", gameCopy.ExtraBallCount))

	return &gameCopy
}

// GetCurrentStage 獲取當前遊戲階段
func (m *GameManager) GetCurrentStage() GameStage {
	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()

	if m.currentGame == nil {
		return StagePreparation
	}

	return m.currentGame.CurrentStage
}

// CreateNewGame 創建新遊戲
func (m *GameManager) CreateNewGame(ctx context.Context) (string, error) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	// 檢查是否有正在進行的遊戲
	if m.currentGame != nil && m.currentGame.CurrentStage != StagePreparation && m.currentGame.CurrentStage != StageGameOver {
		return "", NewGameFlowErrorWithFormat("GAME_IN_PROGRESS",
			"已有遊戲在進行中，無法創建新遊戲。當前遊戲ID: %s, 階段: %s",
			m.currentGame.GameID, m.currentGame.CurrentStage)
	}

	// 創建新遊戲
	gameID := fmt.Sprintf("game_%s", uuid.New().String())
	newGame := NewGameData(gameID)

	// 保存遊戲狀態
	if err := m.repo.SaveGame(ctx, newGame); err != nil {
		m.logger.Error("保存新遊戲失敗",
			zap.String("gameID", gameID),
			zap.Error(err))
		return "", err
	}

	m.logger.Info("已創建新遊戲",
		zap.String("gameID", gameID),
		zap.Int("extraBallCount", newGame.ExtraBallCount))
	m.currentGame = newGame

	// 觸發事件
	if m.onGameCreated != nil {
		m.onGameCreated(gameID)
	}

	return gameID, nil
}

// RegisterWebSocketNotifier 註冊 WebSocket 通知器
func (m *GameManager) RegisterWebSocketNotifier(notifier WebSocketNotifier) {
	m.wsNotifier = notifier
	m.logger.Info("已註冊 WebSocket 通知器")
}

// AdvanceStage 將遊戲推進到下一階段
func (m *GameManager) AdvanceStage(ctx context.Context, autoAdvance bool) error {
	m.stageMutex.Lock()
	if m.currentGame == nil {
		m.stageMutex.Unlock()
		return ErrGameNotFound
	}

	// 記錄當前階段
	previousStage := m.currentGame.CurrentStage

	// 如果是 StagePayoutSettlement 階段，進行幸運球檢查並決定下一階段
	var nextStage GameStage
	if previousStage == StagePayoutSettlement {
		// 從 Redis 獲取當前幸運球
		luckyBalls, err := m.repo.GetLuckyBalls(ctx)
		if err != nil {
			m.logger.Error("獲取幸運球失敗，使用標準流程轉換",
				zap.String("gameID", m.currentGame.GameID),
				zap.Error(err))
			nextStage = GetNextStage(previousStage, m.currentGame.HasJackpot)
		} else {
			// 記錄幸運球號碼
			var luckyBallNumbers []int
			for _, ball := range luckyBalls {
				luckyBallNumbers = append(luckyBallNumbers, ball.Number)
			}

			// 記錄已抽出的球號碼
			var regularBallNumbers, extraBallNumbers []int
			for _, ball := range m.currentGame.RegularBalls {
				regularBallNumbers = append(regularBallNumbers, ball.Number)
			}
			for _, ball := range m.currentGame.ExtraBalls {
				extraBallNumbers = append(extraBallNumbers, ball.Number)
			}

			// 進行幸運球檢查
			var checkResult *LuckyBallCheckResult
			nextStage, checkResult = GetNextStageWithGameDetailed(previousStage, m.currentGame, luckyBalls)

			m.logger.Info("檢查幸運球是否在抽出球中",
				zap.String("gameID", m.currentGame.GameID),
				zap.Int("luckyBallsCount", len(luckyBalls)),
				zap.Ints("luckyBallNumbers", luckyBallNumbers),
				zap.Ints("regularBallNumbers", regularBallNumbers),
				zap.Ints("extraBallNumbers", extraBallNumbers),
				zap.Bool("allLuckyBallsDrawn", checkResult != nil && checkResult.AllMatched),
				zap.Ints("matchedLuckyBalls", func() []int {
					if checkResult != nil {
						return checkResult.MatchedBalls
					}
					return nil
				}()),
				zap.Ints("unmatchedLuckyBalls", func() []int {
					if checkResult != nil {
						return checkResult.UnmatchedBalls
					}
					return nil
				}()))

			// 記錄階段轉換結果
			if nextStage == StageJackpotPreparation {
				if m.currentGame.HasJackpot {
					m.logger.Info("遊戲有JP標記，進入JP階段",
						zap.String("gameID", m.currentGame.GameID))
				} else {
					m.logger.Info("所有幸運球都在已抽出球中，進入JP階段",
						zap.String("gameID", m.currentGame.GameID),
						zap.Ints("matchedLuckyBalls", checkResult.MatchedBalls))
				}
			} else if nextStage == StageGameOver {
				m.logger.Info("存在幸運球不在已抽出球中，直接進入遊戲結束階段",
					zap.String("gameID", m.currentGame.GameID),
					zap.Ints("unmatchedLuckyBalls", checkResult.UnmatchedBalls))
			}
		}
	} else {
		// 其他階段使用標準流程轉換
		nextStage = GetNextStage(previousStage, m.currentGame.HasJackpot)
	}

	m.logger.Info("推進遊戲階段",
		zap.String("gameID", m.currentGame.GameID),
		zap.String("currentStage", string(previousStage)),
		zap.String("nextStage", string(nextStage)),
		zap.Bool("autoAdvance", autoAdvance))

	// 如果提前手動推進，先停止計時器
	if !autoAdvance {
		m.stopStageTimer(m.currentGame.GameID)
	}

	// 特殊處理: 當前階段是 StageGameOver，下一階段是 StagePreparation
	if previousStage == StageGameOver && nextStage == StagePreparation {
		// 設置遊戲結束時間
		m.currentGame.EndTime = time.Now()

		// 保存遊戲歷史記錄
		gameToFinalize := *m.currentGame // 複製當前遊戲以便於後續操作

		// 創建新遊戲替換當前遊戲
		newGameID := fmt.Sprintf("game_%s", uuid.New().String())
		newGame := NewGameData(newGameID)
		m.currentGame = newGame

		// 解鎖，避免長時間持有鎖
		m.stageMutex.Unlock()

		// 保存遊戲歷史到 TiDB
		if err := m.repo.SaveGameHistory(ctx, &gameToFinalize); err != nil {
			m.logger.Error("保存遊戲歷史記錄失敗",
				zap.String("gameID", gameToFinalize.GameID),
				zap.Error(err))
			// 不返回錯誤，繼續執行
		} else {
			m.logger.Info("遊戲歷史記錄已保存",
				zap.String("gameID", gameToFinalize.GameID))
		}

		// 保存新遊戲狀態到 Redis
		if err := m.repo.SaveGame(ctx, newGame); err != nil {
			m.logger.Error("保存新遊戲狀態失敗",
				zap.String("gameID", newGame.GameID),
				zap.Error(err))
			return err
		}

		// 觸發遊戲完成事件
		if m.onGameCompleted != nil {
			m.onGameCompleted(gameToFinalize.GameID)
		}

		// 觸發階段變更事件
		if m.onStageChanged != nil {
			m.onStageChanged(newGame.GameID, previousStage, nextStage)
		}

		// 使用 WebSocket 通知器
		if m.wsNotifier != nil {
			m.wsNotifier.OnStageChanged(newGame.GameID, previousStage, nextStage, newGame)
		}

		return nil // 提前返回，避免執行下面的代碼
	}

	// 更新遊戲階段
	oldStage := m.currentGame.CurrentStage
	m.currentGame.CurrentStage = nextStage
	m.currentGame.LastUpdateTime = time.Now()

	// 執行階段轉換邏輯
	if nextStage == StageJackpotDrawingStart {
		// 在轉入JP開獎階段時初始化Jackpot
		if m.currentGame.Jackpot == nil && m.currentGame.HasJackpot {
			// 初始化Jackpot
			jpID := fmt.Sprintf("jackpot_%s", uuid.New().String())

			// 從資料庫獲取幸運號碼球
			luckyBalls, err := m.repo.GetLuckyBalls(ctx)
			if err != nil {
				m.logger.Error("獲取幸運號碼失敗",
					zap.String("gameID", m.currentGame.GameID),
					zap.Error(err))
				luckyBalls = make([]Ball, 0)
			}

			// 創建JP遊戲
			m.currentGame.Jackpot = &JackpotGame{
				ID:         jpID,
				StartTime:  time.Now(),
				LuckyBalls: luckyBalls,
				DrawnBalls: make([]Ball, 0),
			}

			m.logger.Info("在階段轉換中初始化JP遊戲",
				zap.String("gameID", m.currentGame.GameID),
				zap.String("jackpotID", jpID),
				zap.Int("luckyNumbersCount", len(luckyBalls)))
		}
	}

	// 保存遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		m.logger.Error("保存遊戲狀態失敗",
			zap.String("gameID", m.currentGame.GameID),
			zap.String("stage", string(nextStage)),
			zap.Error(err))
		m.stageMutex.Unlock()
		return err
	}

	// 建立階段計時器
	if err := m.setupStageTimer(ctx, m.currentGame.GameID, nextStage); err != nil {
		m.logger.Error("設置階段計時器失敗",
			zap.String("gameID", m.currentGame.GameID),
			zap.String("stage", string(nextStage)),
			zap.Error(err))
		// 不返回錯誤，繼續執行
	}

	// 複製當前遊戲狀態以供事件處理使用
	gameCopy := *m.currentGame
	m.stageMutex.Unlock()

	// 執行階段特定操作
	if err := m.executeStageSpecificActions(ctx, nextStage); err != nil {
		m.logger.Error("執行階段特定操作失敗",
			zap.String("stage", string(nextStage)),
			zap.Error(err))
		// 不返回錯誤，繼續執行
	}

	// 觸發階段變更事件
	if m.onStageChanged != nil {
		m.onStageChanged(gameCopy.GameID, oldStage, nextStage)
	}

	// 使用 WebSocket 通知器
	if m.wsNotifier != nil {
		m.wsNotifier.OnStageChanged(gameCopy.GameID, oldStage, nextStage, &gameCopy)
	}

	return nil
}

// 設置階段計時器
func (m *GameManager) setupStageTimer(ctx context.Context, gameID string, stage GameStage) error {
	config := GetStageConfig(stage)

	// 如果超時設置為-1，表示無限期等待，不設置計時器
	if config.Timeout < 0 {
		m.logger.Debug("階段無自動超時",
			zap.String("gameID", gameID),
			zap.String("stage", string(stage)))
		return nil
	}

	m.stageTimerMutex.Lock()
	defer m.stageTimerMutex.Unlock()

	// 先停止已有的計時器
	if timer, exists := m.stageTimers[gameID]; exists && timer != nil {
		timer.Stop()
		delete(m.stageTimers, gameID)
		m.logger.Debug("停止並清除現有計時器", zap.String("gameID", gameID))
	}

	// 創建新計時器
	m.logger.Info("設置階段計時器",
		zap.String("gameID", gameID),
		zap.String("stage", string(stage)),
		zap.Duration("timeout", config.Timeout))

	// 為計時器創建獨立的 context，不使用傳入的 ctx，避免在計時器觸發時 context 已經過期
	timerCtx := context.Background()

	timer := time.AfterFunc(config.Timeout, func() {
		// 從 background context 創建新的 context，確保計時器觸發時有有效的 context
		timeoutCtx, cancel := context.WithTimeout(timerCtx, 5*time.Second)
		defer cancel()

		// 在計時器觸發時，推進遊戲階段
		m.logger.Info("階段計時器觸發，自動推進遊戲階段",
			zap.String("gameID", gameID),
			zap.String("stage", string(stage)),
			zap.Duration("timeout", config.Timeout))

		// 首先移除計時器，避免死鎖
		// 這裡在 AdvanceStage 之前清理計時器，避免潛在死鎖
		func() {
			m.stageTimerMutex.Lock()
			defer m.stageTimerMutex.Unlock()
			delete(m.stageTimers, gameID)
			m.logger.Debug("計時器觸發前已刪除", zap.String("gameID", gameID))
		}()

		// 然後推進階段
		if err := m.AdvanceStage(timeoutCtx, true); err != nil {
			m.logger.Error("計時器觸發推進階段失敗",
				zap.String("gameID", gameID),
				zap.String("stage", string(stage)),
				zap.Error(err))
		} else {
			m.logger.Info("計時器觸發階段推進成功",
				zap.String("gameID", gameID),
				zap.String("stage", string(stage)))
		}
	})

	m.stageTimers[gameID] = timer
	m.logger.Info("階段計時器設置完成",
		zap.String("gameID", gameID),
		zap.String("stage", string(stage)),
		zap.Duration("timeout", config.Timeout))
	return nil
}

// 停止階段計時器
func (m *GameManager) stopStageTimer(gameID string) {
	m.stageTimerMutex.Lock()
	defer m.stageTimerMutex.Unlock()

	if timer, exists := m.stageTimers[gameID]; exists && timer != nil {
		timer.Stop()
		delete(m.stageTimers, gameID)
		m.logger.Debug("停止階段計時器", zap.String("gameID", gameID))
	}
}

// executeStageSpecificActions 執行階段特定操作
func (m *GameManager) executeStageSpecificActions(ctx context.Context, stage GameStage) error {
	switch stage {
	case StageNewRound:
		// 新局開始階段的特殊處理
		m.logger.Info("執行新局開始特定操作")

	case StageExtraBallSideSelectBettingStart:
		// 選擇額外球邊的特殊處理
		side, err := m.sidePicker.PickSide()
		if err != nil {
			m.logger.Error("選擇額外球邊失敗", zap.Error(err))
			return err
		}

		// 由於此方法在 AdvanceStage 中已釋放鎖後調用
		// 此處需要獲取鎖以安全地更新遊戲狀態
		m.stageMutex.Lock()
		if m.currentGame == nil {
			m.stageMutex.Unlock()
			return ErrGameNotFound
		}

		m.currentGame.SelectedSide = side

		// 保存遊戲狀態
		err = m.repo.SaveGame(ctx, m.currentGame)

		// 獲取遊戲ID以在解鎖後使用
		gameID := m.currentGame.GameID

		// 釋放鎖
		m.stageMutex.Unlock()

		if err != nil {
			m.logger.Error("保存選擇的額外球邊失敗", zap.Error(err))
			return err
		}

		m.logger.Info("自動選擇額外球邊",
			zap.String("gameID", gameID),
			zap.String("side", string(side)))

		// 觸發選邊事件
		if m.onExtraBallSideSelected != nil {
			m.onExtraBallSideSelected(gameID, side)
		}

	case StageJackpotDrawingStart:
		// 在進入JP開獎階段時初始化Jackpot
		m.stageMutex.Lock()
		if m.currentGame == nil {
			m.stageMutex.Unlock()
			return ErrGameNotFound
		}

		// 如果Jackpot未初始化且遊戲有JP，則創建
		if m.currentGame.Jackpot == nil && m.currentGame.HasJackpot {
			jpID := fmt.Sprintf("jackpot_%s", uuid.New().String())

			// 從資料庫獲取幸運號碼球
			luckyBalls, err := m.repo.GetLuckyBalls(ctx)
			if err != nil {
				m.logger.Error("獲取幸運號碼失敗",
					zap.String("gameID", m.currentGame.GameID),
					zap.Error(err))
				luckyBalls = make([]Ball, 0)
			}

			// 創建JP遊戲
			m.currentGame.Jackpot = &JackpotGame{
				ID:         jpID,
				StartTime:  time.Now(),
				LuckyBalls: luckyBalls,
				DrawnBalls: make([]Ball, 0),
			}

			// 保存遊戲狀態
			err = m.repo.SaveGame(ctx, m.currentGame)

			m.logger.Info("初始化JP遊戲",
				zap.String("gameID", m.currentGame.GameID),
				zap.String("jackpotID", jpID),
				zap.Int("luckyNumbersCount", len(luckyBalls)))

			if err != nil {
				m.logger.Error("保存JP初始化狀態失敗", zap.Error(err))
				m.stageMutex.Unlock()
				return err
			}
		}

		m.stageMutex.Unlock()
	}

	return nil
}

// 最終處理遊戲
func (m *GameManager) finalizeGame(ctx context.Context) error {
	if m.currentGame == nil {
		return ErrGameNotFound
	}

	gameID := m.currentGame.GameID
	m.logger.Info("開始最終處理遊戲", zap.String("gameID", gameID))

	// 設置遊戲結束時間
	m.currentGame.EndTime = time.Now()

	// 保存遊戲歷史記錄
	if err := m.repo.SaveGameHistory(ctx, m.currentGame); err != nil {
		m.logger.Error("保存遊戲歷史記錄失敗",
			zap.String("gameID", gameID),
			zap.Error(err))
		return err
	}

	m.logger.Info("遊戲歷史記錄已保存", zap.String("gameID", gameID))

	// 觸發遊戲完成事件
	if m.onGameCompleted != nil {
		m.onGameCompleted(gameID)
	}

	// 清理當前遊戲數據，準備下一局
	if err := m.prepareForNextGame(ctx, false); err != nil {
		m.logger.Error("準備下一局失敗", zap.Error(err))
		return err
	}

	return nil
}

// prepareForNextGame 準備下一局遊戲
func (m *GameManager) prepareForNextGame(ctx context.Context, alreadyHoldingLock bool) error {
	// 從Redis刪除當前遊戲
	if err := m.repo.DeleteCurrentGame(ctx); err != nil {
		return fmt.Errorf("刪除當前遊戲數據失敗: %w", err)
	}

	// 創建新的遊戲
	newGameID := fmt.Sprintf("game_%s", uuid.New().String())
	newGame := NewGameData(newGameID)

	if !alreadyHoldingLock {
		m.stageMutex.Lock()
	}

	m.currentGame = newGame

	if !alreadyHoldingLock {
		m.stageMutex.Unlock()
	}

	// 保存新遊戲狀態
	if err := m.repo.SaveGame(ctx, newGame); err != nil {
		return fmt.Errorf("保存新遊戲狀態失敗: %w", err)
	}

	m.logger.Info("準備下一局遊戲完成", zap.String("newGameID", newGameID))
	return nil
}

// HandleDrawExtraBall 處理額外球抽取
func (m *GameManager) HandleDrawExtraBall(ctx context.Context, balls []Ball) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return ErrGameNotFound
	}

	// 確認當前階段允許抽取額外球
	if m.currentGame.CurrentStage != StageExtraBallDrawingStart {
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_EXTRA_BALL",
			"當前階段 %s 不允許抽取額外球", m.currentGame.CurrentStage)
	}

	// 確認球的數量不超過 ExtraBallCount
	if len(balls) > m.currentGame.ExtraBallCount {
		return NewGameFlowErrorWithFormat("INVALID_EXTRA_BALL_COUNT",
			"額外球數量 %d 超過了設定的最大值 %d", len(balls), m.currentGame.ExtraBallCount)
	}

	// 創建一個新的球陣列，以便我們可以修改它們
	newBalls := make([]Ball, len(balls))
	for i, ball := range balls {
		// 驗證球號是否合法
		if err := ValidateBallNumber(ball.Number); err != nil {
			return err
		}

		// 檢查是否與常規球重複
		if IsBallDuplicate(ball.Number, m.currentGame.RegularBalls) {
			return fmt.Errorf("額外球號 %d 與常規球重複", ball.Number)
		}

		// 複製並修改球
		newBalls[i] = Ball{
			Number:    ball.Number,
			Type:      BallTypeExtra,
			IsLast:    false, // 預設為 false，稍後會設置最後一個球
			Timestamp: ball.Timestamp,
		}

		// 如果時間戳是零值，設置為當前時間
		if newBalls[i].Timestamp.IsZero() {
			newBalls[i].Timestamp = time.Now()
		}
	}

	// 設置最後一個球的 isLast 標誌（如果陣列長度等於 ExtraBallCount）
	if len(newBalls) > 0 && len(newBalls) == m.currentGame.ExtraBallCount {
		newBalls[len(newBalls)-1].IsLast = true
	}

	// 直接覆蓋陣列
	m.currentGame.ExtraBalls = newBalls
	m.currentGame.LastUpdateTime = time.Now()

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("更新額外球陣列",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballCount", len(newBalls)),
		zap.Int("extraBallCount", m.currentGame.ExtraBallCount))

	// 觸發所有球的事件
	if m.onBallDrawn != nil {
		for _, ball := range newBalls {
			m.onBallDrawn(m.currentGame.GameID, ball)
		}
	}

	// 如果球的數量等於 ExtraBallCount，則自動推進到下一階段
	if len(newBalls) == m.currentGame.ExtraBallCount {
		// 保存當前遊戲ID
		gameID := m.currentGame.GameID

		// 在 goroutine 中推進階段
		go func(gameID string) {
			// 創建新的上下文
			advanceCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// 嘗試推進階段
			if err := m.AdvanceStage(advanceCtx, true); err != nil {
				m.logger.Error("更新額外球後自動推進階段失敗",
					zap.String("gameID", gameID),
					zap.Error(err))
			}
		}(gameID)
	}

	return nil
}

// HandleDrawJackpotBall 處理JP球抽取
func (m *GameManager) HandleDrawJackpotBall(ctx context.Context, number int, isLast bool) (*Ball, error) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return nil, ErrGameNotFound
	}

	// 確認當前階段允許抽取JP球
	if m.currentGame.CurrentStage != StageJackpotDrawingStart {
		return nil, NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_JP_BALL",
			"當前階段 %s 不允許抽取JP球", m.currentGame.CurrentStage)
	}

	// 檢查遊戲是否啟用JP
	if !m.currentGame.HasJackpot {
		return nil, ErrJackpotNotEnabled
	}

	// 如果Jackpot為空，初始化它
	if m.currentGame.Jackpot == nil {
		jpID := fmt.Sprintf("jackpot_%s", uuid.New().String())
		m.currentGame.Jackpot = &JackpotGame{
			ID:         jpID,
			StartTime:  time.Now(),
			LuckyBalls: make([]Ball, 0),
			DrawnBalls: make([]Ball, 0),
		}
		m.logger.Info("初始化JP遊戲",
			zap.String("gameID", m.currentGame.GameID),
			zap.String("jackpotID", jpID))
	}

	// 創建新球
	newBall := Ball{
		Number:    number,
		Type:      BallTypeJackpot,
		IsLast:    isLast,
		Timestamp: time.Now(),
	}

	// 檢查是否是重複球
	if IsBallDuplicate(number, m.currentGame.Jackpot.DrawnBalls) {
		return nil, fmt.Errorf("重複的JP球號: %d", number)
	}

	// 添加球到JP球數組
	m.currentGame.Jackpot.DrawnBalls = append(m.currentGame.Jackpot.DrawnBalls, newBall)
	m.currentGame.LastUpdateTime = time.Now()

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return nil, fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("抽取JP球",
		zap.String("gameID", m.currentGame.GameID),
		zap.String("jackpotID", m.currentGame.Jackpot.ID),
		zap.Int("ballNumber", number),
		zap.Bool("isLast", isLast))

	// 觸發球抽取事件
	if m.onBallDrawn != nil {
		m.onBallDrawn(m.currentGame.GameID, newBall)
	}

	// 如果是最後一顆球，安排自動推進到下一階段，但不在持有鎖的情況下執行
	if isLast {
		// 設置JP結束時間
		m.currentGame.Jackpot.EndTime = time.Now()

		// 再次保存遊戲狀態
		if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
			m.logger.Error("保存JP結束時間失敗",
				zap.String("gameID", m.currentGame.GameID),
				zap.Error(err))
		}

		// 保存當前遊戲ID，用於在goroutine中使用
		gameID := m.currentGame.GameID

		// 在goroutine中執行階段推進，但首先釋放當前持有的鎖
		go func(gameID string) {
			// 創建新的上下文
			advanceCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// 嘗試推進階段
			if err := m.AdvanceStage(advanceCtx, true); err != nil {
				m.logger.Error("最後一顆JP球抽取後自動推進階段失敗",
					zap.String("gameID", gameID),
					zap.Error(err))
			}
		}(gameID)
	}

	return &newBall, nil
}

// HandleDrawLuckyBall 處理幸運號碼球抽取
func (m *GameManager) HandleDrawLuckyBall(ctx context.Context, number int, isLast bool) (*Ball, error) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return nil, ErrGameNotFound
	}

	// 確認當前階段允許抽取幸運號碼球
	if m.currentGame.CurrentStage != StageDrawingLuckyBallsStart {
		return nil, NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_LUCKY_BALL",
			"當前階段 %s 不允許抽取幸運號碼球", m.currentGame.CurrentStage)
	}

	// 添加球
	ball, err := AddBall(m.currentGame, number, BallTypeLucky, isLast)
	if err != nil {
		return nil, err
	}

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return nil, fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	// 同時更新全局幸運號碼記錄
	if isLast && m.currentGame.Jackpot != nil && len(m.currentGame.Jackpot.LuckyBalls) == 7 {
		if err := m.repo.SaveLuckyBalls(ctx, m.currentGame.Jackpot.LuckyBalls); err != nil {
			m.logger.Error("更新全局幸運號碼記錄失敗", zap.Error(err))
			// 不返回錯誤，繼續執行
		}
	}

	m.logger.Info("抽取幸運號碼球",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballNumber", number),
		zap.Bool("isLast", isLast))

	// 觸發球抽取事件
	if m.onBallDrawn != nil {
		m.onBallDrawn(m.currentGame.GameID, *ball)
	}

	// 如果是最後一顆球，安排自動推進到下一階段，但不在持有鎖的情況下執行
	if isLast {
		// 保存當前遊戲ID，用於在goroutine中使用
		gameID := m.currentGame.GameID

		// 在goroutine中執行階段推進，但首先釋放當前持有的鎖
		go func(gameID string) {
			// 創建新的上下文
			advanceCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// 嘗試推進階段
			if err := m.AdvanceStage(advanceCtx, true); err != nil {
				m.logger.Error("最後一顆幸運號碼球抽取後自動推進階段失敗",
					zap.String("gameID", gameID),
					zap.Error(err))
			}
		}(gameID)
	}

	return ball, nil
}

// SetHasJackpot 設置遊戲是否啟用JP
func (m *GameManager) SetHasJackpot(ctx context.Context, hasJackpot bool) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return ErrGameNotFound
	}

	// 只允許在特定階段設置JP狀態
	if m.currentGame.CurrentStage != StagePreparation &&
		m.currentGame.CurrentStage != StageNewRound &&
		m.currentGame.CurrentStage != StageCardPurchaseOpen {
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_JP_SETTING",
			"當前階段 %s 不允許設置JP狀態", m.currentGame.CurrentStage)
	}

	m.currentGame.HasJackpot = hasJackpot

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("設置遊戲JP狀態",
		zap.String("gameID", m.currentGame.GameID),
		zap.Bool("hasJackpot", hasJackpot))

	return nil
}

// isStageAllowedToBeCancelled 檢查遊戲階段是否允許取消
func isStageAllowedToBeCancelled(stage GameStage) bool {
	// 使用 StageConfig 中的 AllowCanceling 設定來決定是否允許取消遊戲
	config := GetStageConfig(stage)
	return config.AllowCanceling
}

// CancelGame 取消當前遊戲
func (m *GameManager) CancelGame(ctx context.Context, reason string) error {
	m.stageMutex.Lock()

	// 檢查條件並設置遊戲狀態
	if m.currentGame == nil {
		m.stageMutex.Unlock()
		return ErrGameNotFound
	}

	// 檢查當前遊戲是否已經被取消
	if m.currentGame.IsCancelled {
		m.stageMutex.Unlock()
		return NewGameFlowError("GAME_ALREADY_CANCELLED", "遊戲已被取消")
	}

	// 檢查遊戲階段，不是所有階段都可以被取消
	stage := m.currentGame.CurrentStage
	if !isStageAllowedToBeCancelled(stage) {
		m.stageMutex.Unlock()
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_CANCELLATION",
			"當前階段 %s 不允許取消遊戲", stage)
	}

	// 設置取消狀態
	m.currentGame.IsCancelled = true
	m.currentGame.CancelReason = reason
	m.currentGame.CancelTime = time.Now()
	m.currentGame.EndTime = time.Now()

	// 保存遊戲狀態副本和ID以便在釋放鎖後使用
	gameCopy := *m.currentGame
	gameID := m.currentGame.GameID

	// 解鎖，後續操作不需要鎖定
	m.stageMutex.Unlock()

	// 保存遊戲狀態到 Redis (可能較快的操作)
	if err := m.repo.SaveGame(ctx, &gameCopy); err != nil {
		return fmt.Errorf("保存取消狀態失敗: %w", err)
	}

	// 保存歷史記錄到 TiDB
	if err := m.repo.SaveGameHistory(ctx, &gameCopy); err != nil {
		m.logger.Error("保存取消遊戲的歷史記錄失敗", zap.Error(err))
	} else {
		if err := m.repo.DeleteCurrentGame(ctx); err != nil {
			m.logger.Error("刪除當前遊戲數據失敗", zap.Error(err))
		}
	}

	// 觸發事件
	if m.onGameCancelled != nil {
		m.onGameCancelled(gameID, reason)
	}

	// 準備下一局
	if err := m.prepareForNextGame(ctx, false); err != nil {
		m.logger.Error("準備下一局失敗", zap.Error(err))
		return err
	}

	return nil
}

// SetEventHandlers 設置事件處理回調
func (m *GameManager) SetEventHandlers(
	onStageChanged func(gameID string, oldStage, newStage GameStage),
	onGameCreated func(gameID string),
	onGameCancelled func(gameID string, reason string),
	onGameCompleted func(gameID string),
	onBallDrawn func(gameID string, ball Ball),
	onExtraBallSideSelected func(gameID string, side ExtraBallSide),
) {
	m.onStageChanged = onStageChanged
	m.onGameCreated = onGameCreated
	m.onGameCancelled = onGameCancelled
	m.onGameCompleted = onGameCompleted
	m.onBallDrawn = onBallDrawn
	m.onExtraBallSideSelected = onExtraBallSideSelected
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
	if m.mqProducer == nil || m.currentGame == nil {
		return nil
	}
	snapshot := BuildGameStatusResponse(m.currentGame)
	mapData, err := mq.StructToMap(snapshot)
	if err != nil {
		m.logger.Error("遊戲快照序列化失敗", zap.Error(err))
		return err
	}
	return m.mqProducer.SendGameSnapshot(m.currentGame.GameID, mapData)
}

// GetOnBallDrawnCallback 獲取球抽取事件回調函數
func (m *GameManager) GetOnBallDrawnCallback() func(gameID string, ball Ball) {
	return m.onBallDrawn
}

// UpdateRegularBalls 直接更新遊戲的常規球陣列
func (m *GameManager) UpdateRegularBalls(ctx context.Context, balls []Ball) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return ErrGameNotFound
	}

	// 確認當前階段允許抽取常規球
	if m.currentGame.CurrentStage != StageDrawingStart {
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_DRAW",
			"當前階段 %s 不允許更新常規球", m.currentGame.CurrentStage)
	}

	// 驗證所有球是否合法
	for _, ball := range balls {
		if err := ValidateBallNumber(ball.Number); err != nil {
			return err
		}
	}

	// 直接覆蓋陣列
	m.currentGame.RegularBalls = balls
	m.currentGame.LastUpdateTime = time.Now()

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("更新常規球陣列",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballCount", len(balls)))

	return nil
}

// UpdateExtraBalls 直接更新遊戲的額外球陣列
func (m *GameManager) UpdateExtraBalls(ctx context.Context, balls []Ball) error {
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

	// 確認球的數量不超過 ExtraBallCount
	if len(balls) > m.currentGame.ExtraBallCount {
		return NewGameFlowErrorWithFormat("INVALID_EXTRA_BALL_COUNT",
			"額外球數量 %d 超過了設定的最大值 %d", len(balls), m.currentGame.ExtraBallCount)
	}

	// 驗證所有球是否合法
	for _, ball := range balls {
		if err := ValidateBallNumber(ball.Number); err != nil {
			return err
		}

		// 檢查是否與常規球重複
		if IsBallDuplicate(ball.Number, m.currentGame.RegularBalls) {
			return fmt.Errorf("額外球號 %d 與常規球重複", ball.Number)
		}
	}

	// 直接覆蓋陣列
	m.currentGame.ExtraBalls = balls
	m.currentGame.LastUpdateTime = time.Now()

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("更新額外球陣列",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballCount", len(balls)))

	// 如果球的數量等於 ExtraBallCount，則自動推進到下一階段
	if len(balls) == m.currentGame.ExtraBallCount {
		// 保存當前遊戲ID
		gameID := m.currentGame.GameID

		// 在 goroutine 中推進階段
		go func(gameID string) {
			// 創建新的上下文
			advanceCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// 嘗試推進階段
			if err := m.AdvanceStage(advanceCtx, true); err != nil {
				m.logger.Error("更新額外球後自動推進階段失敗",
					zap.String("gameID", gameID),
					zap.Error(err))
			}
		}(gameID)
	}

	return nil
}
