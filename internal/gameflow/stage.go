package gameflow

import (
	"time"
)

// GameStage 代表遊戲階段
type GameStage string

const (
	// 主遊戲流程階段
	StagePreparation       GameStage = "PREPARATION"
	StageNewRound          GameStage = "NEW_ROUND"
	StageCardPurchaseOpen  GameStage = "CARD_PURCHASE_OPEN"
	StageCardPurchaseClose GameStage = "CARD_PURCHASE_CLOSE"
	StageDrawingStart      GameStage = "DRAWING_START"
	StageDrawingClose      GameStage = "DRAWING_CLOSE"

	// 額外球流程階段
	StageExtraBallPrepare                 GameStage = "EXTRA_BALL_PREPARE"
	StageExtraBallSideSelectBettingStart  GameStage = "EXTRA_BALL_SIDE_SELECT_BETTING_START"
	StageExtraBallSideSelectBettingClosed GameStage = "EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED"
	StageExtraBallWaitClaim               GameStage = "EXTRA_BALL_WAIT_CLAIM"
	StageExtraBallDrawingStart            GameStage = "EXTRA_BALL_DRAWING_START"
	StageExtraBallDrawingClose            GameStage = "EXTRA_BALL_DRAWING_CLOSE"

	// 派彩與JP流程階段
	StagePayoutSettlement        GameStage = "PAYOUT_SETTLEMENT"
	StageJackpotPreparation      GameStage = "JACKPOT_PREPARATION"
	StageJackpotDrawingStart     GameStage = "JACKPOT_DRAWING_START"
	StageJackpotDrawingClosed    GameStage = "JACKPOT_DRAWING_CLOSED"
	StageJackpotSettlement       GameStage = "JACKPOT_SETTLEMENT"
	StageDrawingLuckyBallsStart  GameStage = "DRAWING_LUCKY_BALLS_START"
	StageDrawingLuckyBallsClosed GameStage = "DRAWING_LUCKY_BALLS_CLOSED"

	// 結束階段
	StageGameOver GameStage = "GAME_OVER"
)

// StageConfig 代表一個遊戲階段的配置
type StageConfig struct {
	Timeout        time.Duration // 階段超時時間，-1表示無限
	RequireDealer  bool          // 是否需要荷官確認才能進入下一階段
	RequireGame    bool          // 是否需要遊戲端確認才能進入下一階段
	AllowDrawBall  bool          // 是否允許抽球操作
	MaxBalls       int           // 最大球數
	AllowCanceling bool          // 是否允許取消遊戲
}

// 自然階段轉換映射表 - 定義標準的階段轉換路徑
var naturalStageTransition = map[GameStage]GameStage{
	StagePreparation:                      StageNewRound,
	StageNewRound:                         StageCardPurchaseOpen,
	StageCardPurchaseOpen:                 StageCardPurchaseClose,
	StageCardPurchaseClose:                StageDrawingStart,
	StageDrawingStart:                     StageDrawingClose,
	StageDrawingClose:                     StageExtraBallPrepare,
	StageExtraBallPrepare:                 StageExtraBallSideSelectBettingStart,
	StageExtraBallSideSelectBettingStart:  StageExtraBallSideSelectBettingClosed,
	StageExtraBallSideSelectBettingClosed: StageExtraBallDrawingStart,
	StageExtraBallDrawingStart:            StageExtraBallDrawingClose,
	StageExtraBallDrawingClose:            StagePayoutSettlement,
	StagePayoutSettlement:                 StageJackpotPreparation, // 默認轉換，實際中會根據是否有JP條件判斷
	StageJackpotPreparation:               StageJackpotDrawingStart,
	StageJackpotDrawingStart:              StageJackpotDrawingClosed,
	StageJackpotDrawingClosed:             StageJackpotSettlement,
	StageJackpotSettlement:                StageDrawingLuckyBallsStart,
	StageDrawingLuckyBallsStart:           StageDrawingLuckyBallsClosed,
	StageDrawingLuckyBallsClosed:          StageGameOver,
	StageGameOver:                         StagePreparation,
}

// GetStageConfig 獲取特定遊戲階段的配置
func GetStageConfig(stage GameStage) StageConfig {
	configs := map[GameStage]StageConfig{
		StagePreparation: {
			Timeout:        -1,   // 無限，等待荷官手動開始
			RequireDealer:  true, // 需要荷官確認
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: false,
		},
		StageNewRound: {
			Timeout:        2 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageCardPurchaseOpen: {
			Timeout:        12 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageCardPurchaseClose: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageDrawingStart: {
			Timeout:        -1, // 無限，由荷官控制進度
			RequireDealer:  true,
			RequireGame:    false,
			AllowDrawBall:  true,
			MaxBalls:       75,
			AllowCanceling: true,
		},
		StageDrawingClose: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageExtraBallPrepare: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageExtraBallSideSelectBettingStart: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageExtraBallSideSelectBettingClosed: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageExtraBallDrawingStart: {
			Timeout:        -1, // 無限，由荷官控制進度
			RequireDealer:  true,
			RequireGame:    false,
			AllowDrawBall:  true,
			MaxBalls:       3,
			AllowCanceling: true,
		},
		StageExtraBallDrawingClose: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StagePayoutSettlement: {
			Timeout:        3 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageJackpotPreparation: {
			Timeout:        3 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageJackpotDrawingStart: {
			Timeout:        -1, // 無限，由荷官控制或直到有人中獎
			RequireDealer:  true,
			RequireGame:    true, // 需要遊戲端確認（有人中獎）
			AllowDrawBall:  true,
			MaxBalls:       75,
			AllowCanceling: true,
		},
		StageJackpotDrawingClosed: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageJackpotSettlement: {
			Timeout:        3 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: true,
		},
		StageDrawingLuckyBallsStart: {
			Timeout:        -1, // 無限，由荷官控制進度
			RequireDealer:  true,
			RequireGame:    false,
			AllowDrawBall:  true,
			MaxBalls:       7,
			AllowCanceling: false, // 已經到幸運號碼階段，不允許取消
		},
		StageDrawingLuckyBallsClosed: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: false,
		},
		StageGameOver: {
			Timeout:        1 * time.Second,
			RequireDealer:  false,
			RequireGame:    false,
			AllowDrawBall:  false,
			MaxBalls:       0,
			AllowCanceling: false,
		},
	}

	if config, ok := configs[stage]; ok {
		return config
	}

	// 默認配置
	return StageConfig{
		Timeout:        1 * time.Second,
		RequireDealer:  false,
		RequireGame:    false,
		AllowDrawBall:  false,
		MaxBalls:       0,
		AllowCanceling: true,
	}
}

// GetNextStage 根據當前階段和遊戲數據計算下一個階段
func GetNextStage(currentStage GameStage, hasJackpot bool) GameStage {
	// 特殊轉換規則
	if currentStage == StagePayoutSettlement && !hasJackpot {
		return StageDrawingLuckyBallsStart
	}

	// 使用自然轉換表
	if nextStage, ok := naturalStageTransition[currentStage]; ok {
		return nextStage
	}

	// 找不到轉換規則，返回準備階段
	return StagePreparation
}

// IsBallDrawingStage 判斷當前階段是否允許抽球
func IsBallDrawingStage(stage GameStage) bool {
	config := GetStageConfig(stage)
	return config.AllowDrawBall
}

// IsLastStageInSequence 判斷是否是序列中的最後一個階段
func IsLastStageInSequence(stage GameStage) bool {
	return stage == StageGameOver
}
