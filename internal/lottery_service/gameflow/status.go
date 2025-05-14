package gameflow

import (
	"time"
)

// GameStatusResponse 對外推送的遊戲狀態格式
// 參考 gamstatus.md JSON 範例

type GameStatusResponse struct {
	Game         GameStatusGameInfo    `json:"game"`
	LuckyNumbers []int                 `json:"luckyNumbers"`
	DrawnBalls   []GameStatusDrawnBall `json:"drawnBalls"`
	ExtraBalls   []GameStatusExtraBall `json:"extraBalls"`
	Jackpot      GameStatusJackpot     `json:"jackpot"`
	TopPlayers   []GameStatusTopPlayer `json:"topPlayers"`
	TotalWin     int64                 `json:"totalWinAmount"`
}

type GameStatusGameInfo struct {
	ID             string             `json:"id"`
	State          string             `json:"state"`
	StartTime      string             `json:"startTime"`
	EndTime        *string            `json:"endTime,omitempty"`
	HasJackpot     bool               `json:"hasJackpot"`
	ExtraBallCount int                `json:"extraBallCount"`
	Timeline       GameStatusTimeline `json:"timeline"`
}

type GameStatusTimeline struct {
	CurrentTime     string `json:"currentTime"`
	StateStartTime  string `json:"stateStartTime"`
	RemainingTime   int    `json:"remainingTime"`
	MaxTimeout      int    `json:"maxTimeout"`
	StageExpireTime string `json:"stageExpireTime"` // ISO8601格式的階段過期時間
}

type GameStatusDrawnBall struct {
	Number    int    `json:"number"`
	DrawnTime string `json:"drawnTime"`
	Sequence  int    `json:"sequence"`
}

type GameStatusExtraBall struct {
	Number    int    `json:"number"`
	DrawnTime string `json:"drawnTime"`
	Sequence  int    `json:"sequence"`
	Side      string `json:"side"`
}

// GameStatusJackpot 表示JP狀態響應
type GameStatusJackpot struct {
	Active     bool                  `json:"active"`           // 是否啟用（僅內存使用，不保存到資料庫）
	GameID     *string               `json:"gameId"`           // 遊戲ID
	Amount     int64                 `json:"amount"`           // 獎金金額（僅內存使用，不保存到資料庫）
	StartTime  *string               `json:"startTime"`        // 開始時間
	EndTime    *string               `json:"endTime"`          // 結束時間
	DrawnBalls []GameStatusDrawnBall `json:"drawnBalls"`       // 已抽出的JP球
	Winner     *string               `json:"winner,omitempty"` // 獲獎者信息（僅內存使用，不保存到資料庫）
}

type GameStatusTopPlayer struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	WinAmount int64  `json:"winAmount"`
	BetAmount int64  `json:"betAmount"`
	Cards     int    `json:"cards"`
}

// BuildGameStatusResponse 將 GameData 轉為 GameStatusResponse
func BuildGameStatusResponse(game *GameData) *GameStatusResponse {
	// 1. 轉換 luckyNumbers
	var luckyNumbers []int
	if game.Jackpot != nil {
		for _, b := range game.Jackpot.LuckyBalls {
			luckyNumbers = append(luckyNumbers, b.Number)
		}
	}

	// 2. 轉換 drawnBalls
	var drawnBalls []GameStatusDrawnBall
	for i, b := range game.RegularBalls {
		drawnBalls = append(drawnBalls, GameStatusDrawnBall{
			Number:    b.Number,
			DrawnTime: b.Timestamp.Format(time.RFC3339),
			Sequence:  i + 1,
		})
	}

	// 3. 轉換 extraBalls
	var extraBalls []GameStatusExtraBall
	for i, b := range game.ExtraBalls {
		extraBalls = append(extraBalls, GameStatusExtraBall{
			Number:    b.Number,
			DrawnTime: b.Timestamp.Format(time.RFC3339),
			Sequence:  i + 1,
			Side:      string(game.SelectedSide),
		})
	}

	// 4. 轉換 jackpot
	var jackpotBalls []GameStatusDrawnBall
	var jackpotGameID *string
	var jackpotStartTime, jackpotEndTime *string

	jackpot := GameStatusJackpot{
		Active: game.HasJackpot,
		Amount: 500000, // TODO: 從資料庫或配置取得
	}

	// 如果有Jackpot實例，填充其他欄位
	if game.HasJackpot && game.Jackpot != nil {
		// JP遊戲ID
		jackpotGameID = &game.Jackpot.ID
		jackpot.GameID = jackpotGameID

		// JP開始/結束時間
		if !game.Jackpot.StartTime.IsZero() {
			s := game.Jackpot.StartTime.Format(time.RFC3339)
			jackpotStartTime = &s
			jackpot.StartTime = jackpotStartTime
		}

		if !game.Jackpot.EndTime.IsZero() {
			s := game.Jackpot.EndTime.Format(time.RFC3339)
			jackpotEndTime = &s
			jackpot.EndTime = jackpotEndTime
		}

		// JP抽出的球
		for i, b := range game.Jackpot.DrawnBalls {
			jackpotBalls = append(jackpotBalls, GameStatusDrawnBall{
				Number:    b.Number,
				DrawnTime: b.Timestamp.Format(time.RFC3339),
				Sequence:  i + 1,
			})
		}
		jackpot.DrawnBalls = jackpotBalls
	}

	// 5. 轉換 topPlayers, totalWinAmount (如有排行榜資料，否則可留空)
	var topPlayers []GameStatusTopPlayer
	var totalWin int64 = 0

	// 6. Timeline
	now := time.Now()
	var remainingTime int
	var stageExpireTimeStr string

	// 處理階段過期時間
	if !game.StageExpireTime.IsZero() {
		stageExpireTimeStr = game.StageExpireTime.Format(time.RFC3339)

		// 計算剩餘時間（秒）
		if game.StageExpireTime.After(now) {
			remainingTime = int(time.Until(game.StageExpireTime).Seconds())
		} else {
			remainingTime = 0
		}
	} else {
		// 如果沒有設置過期時間，使用默認值
		stageExpireTimeStr = ""
		remainingTime = 0
	}

	// 獲取該階段的最大超時時間（以秒為單位）
	config := GetStageConfig(game.CurrentStage)
	maxTimeout := 0
	if config.Timeout > 0 {
		maxTimeout = int(config.Timeout.Seconds())
	}

	timeline := GameStatusTimeline{
		CurrentTime:     now.Format(time.RFC3339),
		StateStartTime:  game.LastUpdateTime.Format(time.RFC3339),
		RemainingTime:   remainingTime,
		MaxTimeout:      maxTimeout,
		StageExpireTime: stageExpireTimeStr,
	}

	// 7. GameInfo
	var endTime *string
	if !game.EndTime.IsZero() {
		s := game.EndTime.Format(time.RFC3339)
		endTime = &s
	}
	gameInfo := GameStatusGameInfo{
		ID:             game.GameID,
		State:          string(game.CurrentStage),
		StartTime:      game.StartTime.Format(time.RFC3339),
		EndTime:        endTime,
		HasJackpot:     game.HasJackpot,
		ExtraBallCount: game.ExtraBallCount,
		Timeline:       timeline,
	}

	return &GameStatusResponse{
		Game:         gameInfo,
		LuckyNumbers: luckyNumbers,
		DrawnBalls:   drawnBalls,
		ExtraBalls:   extraBalls,
		Jackpot:      jackpot,
		TopPlayers:   topPlayers,
		TotalWin:     totalWin,
	}
}
