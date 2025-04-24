package game

import "time"

// GameStatusResponse 代表API的響應數據結構
// @Description 遊戲狀態完整響應
type GameStatusResponse struct {
	// 遊戲基本信息
	// @example {"id":"G20240619001","state":"BETTING","startTime":"2024-06-19T08:00:00Z","endTime":null,"hasJackpot":false,"extraBallCount":3,"timeline":{"currentTime":"2024-06-19T08:05:30Z","stateStartTime":"2024-06-19T08:05:00Z","remainingTime":25,"maxTimeout":60}}
	Game GameInfo `json:"game"`

	// 七個幸運號碼
	// @example [1,12,23,34,45,56,67]
	LuckyNumbers []int `json:"luckyNumbers"`

	// 已抽出的球列表
	// @example [{"number":5,"drawnTime":"2024-06-19T08:06:00Z","sequence":1},{"number":17,"drawnTime":"2024-06-19T08:06:01Z","sequence":2}]
	DrawnBalls []BallInfo `json:"drawnBalls"`

	// 額外球列表
	// @example [{"number":42,"drawnTime":"2024-06-19T08:10:00Z","sequence":1,"side":"LEFT"}]
	ExtraBalls []ExtraBall `json:"extraBalls"`

	// JP遊戲信息
	// @example {"active":false,"gameId":null,"amount":500000,"startTime":null,"endTime":null,"drawnBalls":[],"winner":null}
	Jackpot JackpotInfo `json:"jackpot"`

	// 前三名玩家
	// @example [{"userId":"U123456","nickname":"幸運星","winAmount":25000,"betAmount":5000,"cards":3},{"userId":"U789012","nickname":"好運連連","winAmount":18500,"betAmount":3500,"cards":2}]
	TopPlayers []PlayerInfo `json:"topPlayers"`

	// 所有玩家贏取的總金額
	// @example 75800
	TotalWinAmount float64 `json:"totalWinAmount"`
}

// GameInfo 代表遊戲基本資訊
// @Description 遊戲基本信息
type GameInfo struct {
	// 遊戲唯一ID
	// @example G20240619001
	ID string `json:"id"`

	// 遊戲當前狀態
	// @example BETTING
	State string `json:"state"`

	// 遊戲開始時間
	// @example 2024-06-19T08:00:00Z
	StartTime time.Time `json:"startTime"`

	// 遊戲結束時間，未結束為null
	// @example null
	EndTime *time.Time `json:"endTime"`

	// 是否有JP遊戲
	// @example false
	HasJackpot bool `json:"hasJackpot"`

	// 額外球數量
	// @example 3
	ExtraBallCount int `json:"extraBallCount"`

	// 時間相關信息
	// @example {"currentTime":"2024-06-19T08:05:30Z","stateStartTime":"2024-06-19T08:05:00Z","remainingTime":25,"maxTimeout":60}
	Timeline TimelineInfo `json:"timeline"`
}

// TimelineInfo 代表時間相關資訊
// @Description 時間相關信息
type TimelineInfo struct {
	// 當前時間
	// @example 2024-06-19T08:05:30Z
	CurrentTime time.Time `json:"currentTime"`

	// 當前狀態開始時間
	// @example 2024-06-19T08:05:00Z
	StateStartTime time.Time `json:"stateStartTime"`

	// 當前狀態剩餘秒數
	// @example 25
	RemainingTime int `json:"remainingTime"`

	// 最大超時時間(秒)
	// @example 60
	MaxTimeout int `json:"maxTimeout"`
}

// BallInfo 代表已抽出的球資訊
// @Description 已抽出的球信息
type BallInfo struct {
	// 球號
	// @example 5
	Number int `json:"number"`

	// 抽出時間
	// @example 2024-06-19T08:06:00Z
	DrawnTime time.Time `json:"drawnTime"`

	// 抽出順序
	// @example 1
	Sequence int `json:"sequence"`
}

// ExtraBall 代表額外球資訊
// @Description 額外球信息
type ExtraBall struct {
	// 球號
	// @example 42
	Number int `json:"number"`

	// 抽出時間
	// @example 2024-06-19T08:10:00Z
	DrawnTime time.Time `json:"drawnTime"`

	// 抽出順序
	// @example 1
	Sequence int `json:"sequence"`

	// 球的位置（LEFT或RIGHT）
	// @example LEFT
	Side string `json:"side"`
}

// JackpotInfo 代表JP遊戲資訊
// @Description JP遊戲信息
type JackpotInfo struct {
	// JP遊戲是否啟用
	// @example false
	Active bool `json:"active"`

	// JP遊戲ID
	// @example null
	GameID *string `json:"gameId"`

	// JP獎金金額
	// @example 500000
	Amount float64 `json:"amount"`

	// JP遊戲開始時間
	// @example null
	StartTime *time.Time `json:"startTime"`

	// JP遊戲結束時間
	// @example null
	EndTime *time.Time `json:"endTime"`

	// JP遊戲中抽出的球
	// @example []
	DrawnBalls []BallInfo `json:"drawnBalls"`

	// JP獲勝者資訊
	// @example null
	Winner *string `json:"winner"`
}

// PlayerInfo 代表玩家資訊
// @Description 玩家信息
type PlayerInfo struct {
	// 用戶ID
	// @example U123456
	UserID string `json:"userId"`

	// 用戶暱稱
	// @example 幸運星
	Nickname string `json:"nickname"`

	// 贏取金額
	// @example 25000
	WinAmount float64 `json:"winAmount"`

	// 投注金額
	// @example 5000
	BetAmount float64 `json:"betAmount"`

	// 購買的卡片數量
	// @example 3
	Cards int `json:"cards"`
}
