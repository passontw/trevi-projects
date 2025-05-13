package gameflow

import (
	"context"
	"sync"
	"time"
)

// GetStageMutex 返回遊戲管理器的階段互斥鎖，可用於確保線程安全
func (m *GameManager) GetStageMutex() *sync.RWMutex {
	return &m.stageMutex
}

// GetRepository 返回遊戲管理器的數據存儲庫
func (m *GameManager) GetRepository() GameRepository {
	return m.repo
}

// ReplaceBalls 批量替換指定房間的常規球
// 這是為了支持批量替換功能而添加的方法
func (m *GameManager) ReplaceBalls(ctx context.Context, roomID string, balls []Ball) error {
	// 鎖定以確保線程安全
	m.stageMutex.Lock()
	defer m.stageMutex.Unlock()

	// 獲取遊戲數據
	game, exists := m.currentGames[roomID]
	if !exists || game == nil {
		return ErrGameNotFound
	}

	// 驗證遊戲階段
	if game.CurrentStage != StageDrawingStart {
		return NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_REPLACE_BALLS",
			"當前階段 %s 不允許替換球", game.CurrentStage)
	}

	// 檢查請求中是否有重複球號
	numberSet := make(map[int]bool)
	for _, ball := range balls {
		if numberSet[ball.Number] {
			return NewGameFlowErrorWithFormat("DUPLICATE_BALL_NUMBER",
				"球號 %d 重複", ball.Number)
		}
		numberSet[ball.Number] = true

		// 確保球的類型是常規球
		ball.Type = BallTypeRegular
	}

	// 直接替換球陣列
	game.RegularBalls = balls
	game.LastUpdateTime = time.Now()

	// 保存遊戲狀態
	if err := m.repo.SaveGame(ctx, game); err != nil {
		return NewGameFlowErrorWithFormat("SAVE_GAME_FAILED",
			"保存遊戲狀態失敗: %v", err)
	}

	// 調用球抽取事件回調
	if m.onBallDrawn != nil {
		for _, ball := range balls {
			go m.onBallDrawn(game.GameID, ball)
		}
	}

	return nil
}
