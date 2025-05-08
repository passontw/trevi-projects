package gameflow

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// BallType 代表球的類型
type BallType string

const (
	BallTypeRegular BallType = "REGULAR" // 常規球
	BallTypeExtra   BallType = "EXTRA"   // 額外球
	BallTypeJackpot BallType = "JACKPOT" // Jackpot球
	BallTypeLucky   BallType = "LUCKY"   // 幸運號碼球
)

// Ball 代表一顆球
type Ball struct {
	Number    int       `json:"number"`    // 球號
	Type      BallType  `json:"type"`      // 球類型
	IsLast    bool      `json:"is_last"`   // 是否是該類型中的最後一顆球
	Timestamp time.Time `json:"timestamp"` // 抽取時間
}

// ExtraBallSide 額外球選邊
type ExtraBallSide string

const (
	ExtraBallSideLeft  ExtraBallSide = "LEFT"
	ExtraBallSideRight ExtraBallSide = "RIGHT"
)

// JackpotGame 代表JP遊戲數據
type JackpotGame struct {
	ID         string    `json:"id"`                 // JP遊戲ID
	StartTime  time.Time `json:"start_time"`         // JP開始時間
	EndTime    time.Time `json:"end_time,omitempty"` // JP結束時間
	LuckyBalls []Ball    `json:"lucky_numbers"`      // JP幸運號碼球
	DrawnBalls []Ball    `json:"drawn_balls"`        // JP抽出的球
}

// GameData 代表遊戲數據
type GameData struct {
	GameID       string    `json:"game_id"`            // 遊戲ID
	CurrentStage GameStage `json:"current_stage"`      // 當前階段
	StartTime    time.Time `json:"start_time"`         // 開始時間
	EndTime      time.Time `json:"end_time,omitempty"` // 結束時間
	RegularBalls []Ball    `json:"regular_balls"`      // 常規球
	ExtraBalls   []Ball    `json:"extra_balls"`        // 額外球

	SelectedSide   ExtraBallSide `json:"selected_side"`     // 選擇的額外球一側
	ExtraBallCount int           `json:"extra_ball_count"`  // 額外球數量，範圍是1~3
	HasJackpot     bool          `json:"has_jackpot"`       // 是否有JP
	Jackpot        *JackpotGame  `json:"jackpot,omitempty"` // JP遊戲數據
	IsCancelled    bool          `json:"is_cancelled"`      // 是否已取消
	CancelReason   string        `json:"cancel_reason"`     // 取消原因
	CancelTime     time.Time     `json:"cancel_time"`       // 取消時間
	LastUpdateTime time.Time     `json:"last_update_time"`  // 最後更新時間
}

// 建立一個新的遊戲
func NewGameData(gameID string) *GameData {
	now := time.Now()

	// 隨機生成額外球數量 (1-3)
	n, err := rand.Int(rand.Reader, big.NewInt(3))
	extraBallCount := 1
	if err == nil {
		extraBallCount = int(n.Int64()) + 1 // 加1確保範圍是1-3
	}

	return &GameData{
		GameID:         gameID,
		CurrentStage:   StagePreparation,
		StartTime:      now,
		RegularBalls:   make([]Ball, 0),
		ExtraBalls:     make([]Ball, 0),
		ExtraBallCount: extraBallCount,
		HasJackpot:     true, // 將所有遊戲設為有 JP
		IsCancelled:    false,
		LastUpdateTime: now,
	}
}

// ValidateBallNumber 驗證球號是否有效
func ValidateBallNumber(number int) error {
	if number < 1 || number > 75 {
		return fmt.Errorf("無效的球號: %d，球號必須在 1-75 之間", number)
	}
	return nil
}

// IsBallDuplicate 檢查是否是重複球
func IsBallDuplicate(number int, existingBalls []Ball) bool {
	for _, ball := range existingBalls {
		if ball.Number == number {
			return true
		}
	}
	return false
}

// IsBallDuplicateAcrossAllTypes 檢查球是否在所有類型中出現過
func IsBallDuplicateAcrossAllTypes(number int, game *GameData) bool {
	// 檢查常規球和額外球
	if IsBallDuplicate(number, game.RegularBalls) ||
		IsBallDuplicate(number, game.ExtraBalls) {
		return true
	}

	// 檢查JP球（如果存在）
	if game.Jackpot != nil {
		if IsBallDuplicate(number, game.Jackpot.DrawnBalls) ||
			IsBallDuplicate(number, game.Jackpot.LuckyBalls) {
			return true
		}
	}

	return false
}

// GenerateLuckyBalls 生成7顆幸運號碼球
func GenerateLuckyBalls() ([]Ball, error) {
	var luckyBalls []Ball
	usedNumbers := make(map[int]bool)

	for len(luckyBalls) < 7 {
		// 產生1-75的隨機數
		n, err := rand.Int(rand.Reader, big.NewInt(75))
		if err != nil {
			return nil, fmt.Errorf("生成隨機數失敗: %w", err)
		}

		num := int(n.Int64()) + 1

		// 檢查是否重複
		if !usedNumbers[num] {
			usedNumbers[num] = true

			isLast := len(luckyBalls) == 6 // 第七顆球是最後一顆

			luckyBalls = append(luckyBalls, Ball{
				Number:    num,
				Type:      BallTypeLucky,
				IsLast:    isLast,
				Timestamp: time.Now(),
			})

			// 加入小延遲以確保不同的時間戳
			time.Sleep(1 * time.Millisecond)
		}
	}

	return luckyBalls, nil
}

// AddBall 添加一顆球到遊戲中
func AddBall(game *GameData, number int, ballType BallType, isLast bool) (*Ball, error) {
	// 驗證球號
	if err := ValidateBallNumber(number); err != nil {
		return nil, err
	}

	// 檢查該階段是否允許抽球
	if !IsBallDrawingStage(game.CurrentStage) {
		return nil, fmt.Errorf("當前階段 %s 不允許抽球", game.CurrentStage)
	}

	// 根據球類型和當前階段執行特定的檢查
	switch ballType {
	case BallTypeRegular:
		if game.CurrentStage != StageDrawingStart {
			return nil, fmt.Errorf("常規球只能在 %s 階段抽取", StageDrawingStart)
		}
		if IsBallDuplicate(number, game.RegularBalls) {
			return nil, fmt.Errorf("重複的常規球號: %d", number)
		}
		if len(game.RegularBalls) >= 75 {
			return nil, fmt.Errorf("已達到最大常規球數量")
		}

	case BallTypeExtra:
		if game.CurrentStage != StageExtraBallDrawingStart {
			return nil, fmt.Errorf("額外球只能在 %s 階段抽取", StageExtraBallDrawingStart)
		}
		if IsBallDuplicate(number, game.ExtraBalls) {
			return nil, fmt.Errorf("重複的額外球號: %d", number)
		}
		if IsBallDuplicate(number, game.RegularBalls) {
			return nil, fmt.Errorf("額外球號 %d 與常規球重複", number)
		}
		if len(game.ExtraBalls) >= game.ExtraBallCount {
			return nil, fmt.Errorf("已達到最大額外球數量 %d", game.ExtraBallCount)
		}

	case BallTypeJackpot:
		if game.CurrentStage != StageJackpotDrawingStart {
			return nil, fmt.Errorf("JP球只能在 %s 階段抽取", StageJackpotDrawingStart)
		}
		if game.Jackpot == nil {
			return nil, fmt.Errorf("JP遊戲未初始化")
		}
		if IsBallDuplicate(number, game.Jackpot.DrawnBalls) {
			return nil, fmt.Errorf("重複的JP球號: %d", number)
		}
		if len(game.Jackpot.DrawnBalls) >= 75 {
			return nil, fmt.Errorf("已達到最大JP球數量")
		}

	case BallTypeLucky:
		if game.CurrentStage != StageDrawingLuckyBallsStart {
			return nil, fmt.Errorf("幸運號碼球只能在 %s 階段抽取", StageDrawingLuckyBallsStart)
		}
		// 幸運號碼球現在應該直接添加到 Jackpot.LuckyBalls
		if game.Jackpot == nil {
			// 如果 Jackpot 不存在，初始化它
			game.Jackpot = &JackpotGame{
				ID:         fmt.Sprintf("jackpot_%s", time.Now().Format("20060102150405")),
				StartTime:  time.Now(),
				LuckyBalls: make([]Ball, 0),
				DrawnBalls: make([]Ball, 0),
			}
		}
		if IsBallDuplicate(number, game.Jackpot.LuckyBalls) {
			return nil, fmt.Errorf("重複的幸運號碼球號: %d", number)
		}
		if len(game.Jackpot.LuckyBalls) >= 7 {
			return nil, fmt.Errorf("已達到最大幸運號碼球數量")
		}

	default:
		return nil, fmt.Errorf("未知的球類型: %s", ballType)
	}

	// 創建新球
	newBall := Ball{
		Number:    number,
		Type:      ballType,
		IsLast:    isLast,
		Timestamp: time.Now(),
	}

	// 添加球到相應的數組中
	switch ballType {
	case BallTypeRegular:
		game.RegularBalls = append(game.RegularBalls, newBall)
	case BallTypeExtra:
		game.ExtraBalls = append(game.ExtraBalls, newBall)
	case BallTypeJackpot:
		if game.Jackpot != nil {
			game.Jackpot.DrawnBalls = append(game.Jackpot.DrawnBalls, newBall)
		}
	case BallTypeLucky:
		// 幸運號碼球直接添加到 Jackpot.LuckyBalls
		if game.Jackpot != nil {
			game.Jackpot.LuckyBalls = append(game.Jackpot.LuckyBalls, newBall)
		}
	}

	// 更新遊戲最後更新時間
	game.LastUpdateTime = time.Now()

	return &newBall, nil
}
