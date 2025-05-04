package gameflow

import (
	"context"
	"fmt"
	"g38_lottery_service/internal/mq"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
	m.stageMutex.RLock()
	defer m.stageMutex.RUnlock()

	if m.currentGame == nil {
		return nil
	}

	// 返回一個複本，避免外部修改
	gameCopy := *m.currentGame
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

	m.logger.Info("已創建新遊戲", zap.String("gameID", gameID))
	m.currentGame = newGame

	// 觸發事件
	if m.onGameCreated != nil {
		m.onGameCreated(gameID)
	}

	return gameID, nil
}

// AdvanceStage 推進遊戲階段
func (m *GameManager) AdvanceStage(ctx context.Context, force bool) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return ErrGameNotFound
	}

	currentStage := m.currentGame.CurrentStage

	// 檢查當前階段配置
	config := GetStageConfig(currentStage)

	// 如果不是強制推進且需要荷官確認，則阻止自動推進
	if !force && config.RequireDealer {
		return NewGameFlowErrorWithFormat("REQUIRE_DEALER_CONFIRMATION",
			"階段 %s 需要荷官確認才能推進", currentStage)
	}

	// 計算下一階段
	nextStage := GetNextStage(currentStage, m.currentGame.HasJackpot)

	// 切換到新階段前，先停止舊計時器
	m.stopStageTimer(m.currentGame.GameID)

	oldStage := m.currentGame.CurrentStage
	m.currentGame.CurrentStage = nextStage
	m.currentGame.LastUpdateTime = time.Now()

	// 保存新狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		// 保存失敗，回滾階段
		m.currentGame.CurrentStage = oldStage
		m.logger.Error("保存新階段失敗，回滾階段",
			zap.String("gameID", m.currentGame.GameID),
			zap.String("oldStage", string(oldStage)),
			zap.String("nextStage", string(nextStage)),
			zap.Error(err))
		return err
	}

	m.logger.Info("遊戲階段已推進",
		zap.String("gameID", m.currentGame.GameID),
		zap.String("oldStage", string(oldStage)),
		zap.String("newStage", string(nextStage)))

	// 設置新階段的計時器
	if err := m.setupStageTimer(ctx, m.currentGame.GameID, nextStage); err != nil {
		m.logger.Error("設置新階段計時器失敗", zap.Error(err))
		// 不回滾階段，因為階段已經成功推進並保存
	}

	// 對於某些階段，執行額外操作
	if err := m.executeStageSpecificActions(ctx, nextStage); err != nil {
		m.logger.Error("執行階段特定操作失敗",
			zap.String("stage", string(nextStage)),
			zap.Error(err))
		// 不回滾階段，繼續執行
	}

	// 觸發事件
	if m.onStageChanged != nil {
		m.onStageChanged(m.currentGame.GameID, oldStage, nextStage)
	}

	// 自動推送快照
	if m.mqProducer != nil {
		go m.pushGameSnapshot(ctx)
	}

	// 如果是遊戲結束階段，則執行遊戲結束操作
	if nextStage == StageGameOver {
		if err := m.finalizeGame(ctx); err != nil {
			m.logger.Error("遊戲結束操作失敗", zap.Error(err))
			return err
		}
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
	}

	// 創建新計時器
	m.logger.Debug("設置階段計時器",
		zap.String("gameID", gameID),
		zap.String("stage", string(stage)),
		zap.Duration("timeout", config.Timeout))

	timer := time.AfterFunc(config.Timeout, func() {
		// 在計時器觸發時，推進遊戲階段
		m.logger.Info("階段計時器觸發，自動推進遊戲階段",
			zap.String("gameID", gameID),
			zap.String("stage", string(stage)))

		if err := m.AdvanceStage(ctx, true); err != nil {
			m.logger.Error("計時器觸發推進階段失敗",
				zap.String("gameID", gameID),
				zap.String("stage", string(stage)),
				zap.Error(err))
		}

		// 計時器觸發後自動刪除
		m.stageTimerMutex.Lock()
		delete(m.stageTimers, gameID)
		m.stageTimerMutex.Unlock()
	})

	m.stageTimers[gameID] = timer
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

// 執行階段特定操作
func (m *GameManager) executeStageSpecificActions(ctx context.Context, stage GameStage) error {
	switch stage {
	case StageNewRound:
		// 新局開始階段的特殊處理
		m.logger.Info("執行新局開始特定操作", zap.String("gameID", m.currentGame.GameID))

	case StageExtraBallSideSelectBettingStart:
		// 選擇額外球邊的特殊處理
		side, err := m.sidePicker.PickSide()
		if err != nil {
			m.logger.Error("選擇額外球邊失敗", zap.Error(err))
			return err
		}

		m.stageMutex.Lock()
		m.currentGame.SelectedSide = side
		m.stageMutex.Unlock()

		// 保存遊戲狀態
		if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
			m.logger.Error("保存選擇的額外球邊失敗", zap.Error(err))
			return err
		}

		m.logger.Info("自動選擇額外球邊",
			zap.String("gameID", m.currentGame.GameID),
			zap.String("side", string(side)))

		// 觸發選邊事件
		if m.onExtraBallSideSelected != nil {
			m.onExtraBallSideSelected(m.currentGame.GameID, side)
		}
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
	if err := m.prepareForNextGame(ctx); err != nil {
		m.logger.Error("準備下一局失敗", zap.Error(err))
		return err
	}

	return nil
}

// prepareForNextGame 準備下一局遊戲
func (m *GameManager) prepareForNextGame(ctx context.Context) error {
	// 從Redis刪除當前遊戲
	if err := m.repo.DeleteCurrentGame(ctx); err != nil {
		return fmt.Errorf("刪除當前遊戲數據失敗: %w", err)
	}

	// 創建新的遊戲，設置為準備階段
	newGameID := fmt.Sprintf("game_%s", uuid.New().String())
	newGame := NewGameData(newGameID)

	m.stageMutex.Lock()
	m.currentGame = newGame
	m.stageMutex.Unlock()

	// 保存新遊戲狀態
	if err := m.repo.SaveGame(ctx, newGame); err != nil {
		return fmt.Errorf("保存新遊戲狀態失敗: %w", err)
	}

	m.logger.Info("準備下一局遊戲完成", zap.String("newGameID", newGameID))
	return nil
}

// HandleDrawBall 處理常規球抽取
func (m *GameManager) HandleDrawBall(ctx context.Context, number int, isLast bool) (*Ball, error) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return nil, ErrGameNotFound
	}

	// 確認當前階段允許抽取常規球
	if m.currentGame.CurrentStage != StageDrawingStart {
		return nil, NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_DRAW",
			"當前階段 %s 不允許抽取常規球", m.currentGame.CurrentStage)
	}

	// 添加球
	ball, err := AddBall(m.currentGame, number, BallTypeRegular, isLast)
	if err != nil {
		return nil, err
	}

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return nil, fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("抽取常規球",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballNumber", number),
		zap.Bool("isLast", isLast))

	// 觸發球抽取事件
	if m.onBallDrawn != nil {
		m.onBallDrawn(m.currentGame.GameID, *ball)
	}

	// 如果是最後一顆球，自動推進到下一階段
	if isLast {
		go func() {
			if err := m.AdvanceStage(ctx, true); err != nil {
				m.logger.Error("最後一顆常規球抽取後自動推進階段失敗", zap.Error(err))
			}
		}()
	}

	return ball, nil
}

// HandleDrawExtraBall 處理額外球抽取
func (m *GameManager) HandleDrawExtraBall(ctx context.Context, number int, isLast bool) (*Ball, error) {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return nil, ErrGameNotFound
	}

	// 確認當前階段允許抽取額外球
	if m.currentGame.CurrentStage != StageExtraBallDrawingStart {
		return nil, NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_EXTRA_BALL",
			"當前階段 %s 不允許抽取額外球", m.currentGame.CurrentStage)
	}

	// 添加球
	ball, err := AddBall(m.currentGame, number, BallTypeExtra, isLast)
	if err != nil {
		return nil, err
	}

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return nil, fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("抽取額外球",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballNumber", number),
		zap.Bool("isLast", isLast))

	// 觸發球抽取事件
	if m.onBallDrawn != nil {
		m.onBallDrawn(m.currentGame.GameID, *ball)
	}

	// 如果是最後一顆球，自動推進到下一階段
	if isLast {
		go func() {
			if err := m.AdvanceStage(ctx, true); err != nil {
				m.logger.Error("最後一顆額外球抽取後自動推進階段失敗", zap.Error(err))
			}
		}()
	}

	return ball, nil
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

	// 添加球
	ball, err := AddBall(m.currentGame, number, BallTypeJackpot, isLast)
	if err != nil {
		return nil, err
	}

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return nil, fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("抽取JP球",
		zap.String("gameID", m.currentGame.GameID),
		zap.Int("ballNumber", number),
		zap.Bool("isLast", isLast))

	// 觸發球抽取事件
	if m.onBallDrawn != nil {
		m.onBallDrawn(m.currentGame.GameID, *ball)
	}

	// 如果是最後一顆球，自動推進到下一階段
	if isLast {
		go func() {
			if err := m.AdvanceStage(ctx, true); err != nil {
				m.logger.Error("最後一顆JP球抽取後自動推進階段失敗", zap.Error(err))
			}
		}()
	}

	return ball, nil
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
	if isLast && len(m.currentGame.LuckyBalls) == 7 {
		if err := m.repo.SaveLuckyBalls(ctx, m.currentGame.LuckyBalls); err != nil {
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

	// 如果是最後一顆球，自動推進到下一階段
	if isLast {
		go func() {
			if err := m.AdvanceStage(ctx, true); err != nil {
				m.logger.Error("最後一顆幸運號碼球抽取後自動推進階段失敗", zap.Error(err))
			}
		}()
	}

	return ball, nil
}

// NotifyJackpotWinner 通知JP獲獎者
func (m *GameManager) NotifyJackpotWinner(ctx context.Context, winnerID string) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return ErrGameNotFound
	}

	// 確認當前階段是JP抽球階段
	if m.currentGame.CurrentStage != StageJackpotDrawingStart {
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_JP_WINNER",
			"當前階段 %s 不允許設置JP獲獎者", m.currentGame.CurrentStage)
	}

	// 設置JP獲獎者
	m.currentGame.JackpotWinner = winnerID

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return fmt.Errorf("保存遊戲狀態失敗: %w", err)
	}

	m.logger.Info("設置JP獲獎者",
		zap.String("gameID", m.currentGame.GameID),
		zap.String("winnerID", winnerID))

	// 有人中獎，自動推進到下一階段
	go func() {
		if err := m.AdvanceStage(ctx, true); err != nil {
			m.logger.Error("JP獲獎後自動推進階段失敗", zap.Error(err))
		}
	}()

	return nil
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

// CancelGame 取消當前遊戲
func (m *GameManager) CancelGame(ctx context.Context, reason string) error {
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	if m.currentGame == nil {
		return ErrGameNotFound
	}

	// 檢查是否已經取消
	if m.currentGame.IsCancelled {
		return ErrGameAlreadyCancelled
	}

	// 檢查當前階段是否允許取消
	stageConfig := GetStageConfig(m.currentGame.CurrentStage)
	if !stageConfig.AllowCanceling {
		return NewGameFlowErrorWithFormat("CANNOT_CANCEL_AT_STAGE",
			"當前階段 %s 不允許取消遊戲", m.currentGame.CurrentStage)
	}

	// 設置取消狀態
	m.currentGame.IsCancelled = true
	m.currentGame.CancelReason = reason
	m.currentGame.CancelTime = time.Now()
	m.currentGame.EndTime = time.Now()

	// 保存更新後的遊戲狀態
	if err := m.repo.SaveGame(ctx, m.currentGame); err != nil {
		return fmt.Errorf("保存取消狀態失敗: %w", err)
	}

	// 保存遊戲歷史記錄
	if err := m.repo.SaveGameHistory(ctx, m.currentGame); err != nil {
		m.logger.Error("保存取消遊戲的歷史記錄失敗", zap.Error(err))
		// 繼續執行，不返回錯誤
	}

	m.logger.Info("遊戲已取消",
		zap.String("gameID", m.currentGame.GameID),
		zap.String("reason", reason))

	// 觸發取消事件
	if m.onGameCancelled != nil {
		m.onGameCancelled(m.currentGame.GameID, reason)
	}

	// 準備下一局
	if err := m.prepareForNextGame(ctx); err != nil {
		m.logger.Error("取消遊戲後準備下一局失敗", zap.Error(err))
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
func (m *GameManager) pushGameSnapshot(ctx context.Context) {
	if m.mqProducer == nil || m.currentGame == nil {
		return
	}
	snapshot := BuildGameStatusResponse(m.currentGame)
	mapData, err := mq.StructToMap(snapshot)
	if err != nil {
		m.logger.Error("遊戲快照序列化失敗", zap.Error(err))
		return
	}
	err = m.mqProducer.SendGameSnapshot(m.currentGame.GameID, mapData)
	if err != nil {
		m.logger.Error("推送遊戲快照到 RocketMQ 失敗", zap.Error(err))
	}
}
