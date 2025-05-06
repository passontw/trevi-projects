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
	CurrentTime    string `json:"currentTime"`
	StateStartTime string `json:"stateStartTime"`
	RemainingTime  int    `json:"remainingTime"`
	MaxTimeout     int    `json:"maxTimeout"`
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

type GameStatusJackpot struct {
	Active     bool                  `json:"active"`
	GameID     *string               `json:"gameId"`
	Amount     int64                 `json:"amount"`
	StartTime  *string               `json:"startTime"`
	EndTime    *string               `json:"endTime"`
	DrawnBalls []GameStatusDrawnBall `json:"drawnBalls"`
	Winner     *string               `json:"winner"`
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
	for _, b := range game.LuckyBalls {
		luckyNumbers = append(luckyNumbers, b.Number)
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
	for i, b := range game.JackpotBalls {
		jackpotBalls = append(jackpotBalls, GameStatusDrawnBall{
			Number:    b.Number,
			DrawnTime: b.Timestamp.Format(time.RFC3339),
			Sequence:  i + 1,
		})
	}
	var jackpotGameID *string
	if game.HasJackpot {
		jackpotGameID = &game.GameID
	}
	var jackpotWinner *string
	if game.JackpotWinner != "" {
		jackpotWinner = &game.JackpotWinner
	}
	var jackpotStartTime, jackpotEndTime *string
	if !game.StartTime.IsZero() {
		s := game.StartTime.Format(time.RFC3339)
		jackpotStartTime = &s
	}
	if !game.EndTime.IsZero() {
		s := game.EndTime.Format(time.RFC3339)
		jackpotEndTime = &s
	}
	jackpot := GameStatusJackpot{
		Active:     game.HasJackpot,
		GameID:     jackpotGameID,
		Amount:     500000, // TODO: 從資料庫或配置取得
		StartTime:  jackpotStartTime,
		EndTime:    jackpotEndTime,
		DrawnBalls: jackpotBalls,
		Winner:     jackpotWinner,
	}

	// 5. 轉換 topPlayers, totalWinAmount (如有排行榜資料，否則可留空)
	var topPlayers []GameStatusTopPlayer
	var totalWin int64 = 0

	// 6. Timeline
	timeline := GameStatusTimeline{
		CurrentTime:    time.Now().Format(time.RFC3339),
		StateStartTime: game.LastUpdateTime.Format(time.RFC3339),
		RemainingTime:  0, // TODO: 計算剩餘秒數
		MaxTimeout:     0, // TODO: 取自階段設定
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
		ExtraBallCount: len(game.ExtraBalls),
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
