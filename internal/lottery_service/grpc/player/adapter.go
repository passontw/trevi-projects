package player

import (
	"time"

	dealerPb "g38_lottery_service/internal/generated/api/v1/dealer"
	playerPb "g38_lottery_service/internal/generated/api/v1/player"
	commonpb "g38_lottery_service/internal/generated/common"
	"g38_lottery_service/internal/lottery_service/gameflow"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// 添加自定義Stage類型，用於在轉換過程中使用
type Stage int

// 遊戲階段常量
const (
	StageUnspecified Stage = iota
	StagePrepare
	StageRoundStart
	StageDrawing
	StageRoundEnd
	StageJackpotStart
	StageJackpotDraw
	StageJackpotEnd
	StageAwaitingStart
)

// ConvertGameDataToNewPb 將 gameflow.GameData 轉換為新版本的 Proto 結構
func ConvertGameDataToNewPb(gameData *gameflow.GameData) *dealerPb.GameData {
	if gameData == nil {
		return nil
	}

	// 轉換所有球
	// 使用 RegularBalls 作為 DrawnBalls
	drawnBalls := make([]*dealerPb.Ball, 0, len(gameData.RegularBalls))
	for _, ball := range gameData.RegularBalls {
		drawnBalls = append(drawnBalls, ConvertBallToNewPb(ball))
	}

	// ExtraBalls需要轉換為map格式
	extraBallsMap := make(map[string]*dealerPb.Ball)
	for i, ball := range gameData.ExtraBalls {
		key := "extra_" + string(rune('a'+i))
		if i == 0 {
			key = "left"
		} else if i == 1 {
			key = "right"
		}
		extraBallsMap[key] = ConvertBallToNewPb(ball)
	}

	// 處理頭獎球
	var jackpotBall *dealerPb.Ball
	if gameData.Jackpot != nil && len(gameData.Jackpot.DrawnBalls) > 0 {
		jackpotBall = ConvertBallToNewPb(gameData.Jackpot.DrawnBalls[0])
	}

	// 處理幸運球
	luckyBalls := make([]*dealerPb.Ball, 0)
	if gameData.Jackpot != nil && len(gameData.Jackpot.LuckyBalls) > 0 {
		for _, ball := range gameData.Jackpot.LuckyBalls {
			luckyBalls = append(luckyBalls, ConvertBallToNewPb(ball))
		}
	}

	// 轉換 game stage
	gameStage := ConvertGameStageToCommonPb(gameData.CurrentStage)

	// 使用最後更新時間作為更新時間
	updatedAt := ConvertTimestampToUnix(gameData.LastUpdateTime)

	// 返回新的GameData結構，根據 game.proto 中的 GameData 定義
	return &dealerPb.GameData{
		Id:          gameData.GameID,
		RoomId:      gameData.RoomID,
		Stage:       gameStage,
		Status:      GetGameStatusFromStage(gameStage),
		DrawnBalls:  drawnBalls,
		ExtraBalls:  extraBallsMap,
		JackpotBall: jackpotBall,
		LuckyBalls:  luckyBalls,
		CreatedAt:   ConvertTimestampToUnix(gameData.StartTime),
		UpdatedAt:   updatedAt,
		DealerId:    "system", // 使用默認值
	}
}

// ConvertBallToNewPb 將 gameflow.Ball 轉換為新版本的 Proto 結構
func ConvertBallToNewPb(ball gameflow.Ball) *dealerPb.Ball {
	return &dealerPb.Ball{
		Id:      generateRandomId(), // 生成一個隨機ID
		Number:  int32(ball.Number),
		Color:   getBallColor(ball.Number),
		IsOdd:   ball.Number%2 != 0,
		IsSmall: ball.Number <= 40,
	}
}

// getBallColor 根據球號獲取顏色
func getBallColor(number int) string {
	switch {
	case number >= 1 && number <= 16:
		return "red"
	case number >= 17 && number <= 32:
		return "blue"
	case number >= 33 && number <= 48:
		return "green"
	case number >= 49 && number <= 64:
		return "yellow"
	case number >= 65 && number <= 80:
		return "purple"
	default:
		return "unknown"
	}
}

// 生成隨機ID
func generateRandomId() string {
	return uuid.New().String()
}

// ConvertGameStageToCommonPb 將 gameflow.GameStage 轉換為 dealer.GameStage
func ConvertGameStageToCommonPb(stage gameflow.GameStage) commonpb.GameStage {
	// 將字符串類型的GameStage映射到新版本中的枚舉
	switch stage {
	case gameflow.StagePreparation:
		return commonpb.GameStage_GAME_STAGE_PREPARATION
	case gameflow.StageNewRound:
		return commonpb.GameStage_GAME_STAGE_NEW_ROUND
	case gameflow.StageCardPurchaseOpen:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case gameflow.StageCardPurchaseClose:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case gameflow.StageDrawingStart:
		return commonpb.GameStage_GAME_STAGE_DRAWING_START
	case gameflow.StageDrawingClose:
		return commonpb.GameStage_GAME_STAGE_DRAWING_CLOSE
	case gameflow.StageExtraBallPrepare:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_PREPARE
	case gameflow.StageExtraBallSideSelectBettingStart:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_START
	case gameflow.StageExtraBallSideSelectBettingClosed:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED
	case gameflow.StageExtraBallWaitClaim:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_WAIT_CLAIM
	case gameflow.StageExtraBallDrawingStart:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START
	case gameflow.StageExtraBallDrawingClose:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_CLOSE
	case gameflow.StagePayoutSettlement:
		return commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	case gameflow.StageJackpotPreparation:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_START
	case gameflow.StageJackpotDrawingStart:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START
	case gameflow.StageJackpotDrawingClosed:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_CLOSED
	case gameflow.StageJackpotSettlement:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_SETTLEMENT
	case gameflow.StageDrawingLuckyBallsStart:
		return commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START
	case gameflow.StageDrawingLuckyBallsClosed:
		return commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_CLOSED
	case gameflow.StageGameOver:
		return commonpb.GameStage_GAME_STAGE_GAME_OVER
	default:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// GetGameStatusFromStage 根據遊戲階段獲取遊戲狀態
func GetGameStatusFromStage(stage commonpb.GameStage) dealerPb.GameStatus {
	// 根據階段判斷狀態
	switch {
	case stage == commonpb.GameStage_GAME_STAGE_UNSPECIFIED:
		return dealerPb.GameStatus_GAME_STATUS_UNSPECIFIED
	case stage == commonpb.GameStage_GAME_STAGE_PREPARATION:
		return dealerPb.GameStatus_GAME_STATUS_NOT_STARTED
	case stage == commonpb.GameStage_GAME_STAGE_GAME_OVER:
		return dealerPb.GameStatus_GAME_STATUS_COMPLETED
	case stage >= commonpb.GameStage_GAME_STAGE_NEW_ROUND && stage < commonpb.GameStage_GAME_STAGE_GAME_OVER:
		return dealerPb.GameStatus_GAME_STATUS_RUNNING
	default:
		return dealerPb.GameStatus_GAME_STATUS_UNSPECIFIED
	}
}

// ConvertTimestampToUnix 將時間戳轉換為 Unix 時間戳（秒）
func ConvertTimestampToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// ConvertExtraBallSideToCommonPb 將遊戲邏輯層 ExtraBallSide 轉換為新版 ExtraBallSide
func ConvertExtraBallSideToCommonPb(side gameflow.ExtraBallSide) commonpb.ExtraBallSide {
	switch side {
	case gameflow.ExtraBallSideLeft:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case gameflow.ExtraBallSideRight:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// ConvertPlayerInfoToNewPb 將玩家資訊轉換為新版格式
func ConvertPlayerInfoToNewPb(playerID string, nickname string, balance float64) *playerPb.PlayerInfo {
	return &playerPb.PlayerInfo{
		Id:         playerID,
		Nickname:   nickname,
		Balance:    balance,
		CardsCount: 0, // 默認值
		Preferences: &playerPb.PlayerPreference{
			ReceiveGameNotifications: true,
			ReceiveChatMessages:      true,
			ShowOtherPlayersBets:     true,
			UiTheme:                  "default",
			Language:                 "zh-TW",
		},
	}
}

// CreateGameHistoryItem 創建遊戲歷史記錄項
func CreateGameHistoryItem(gameID string, gameTime time.Time, winAmount float64, betAmount float64, resultSummary string) *playerPb.GameHistoryItem {
	// 將時間轉換為 timestamp.Timestamp
	gameTimeProto := &timestamppb.Timestamp{
		Seconds: gameTime.Unix(),
		Nanos:   int32(gameTime.Nanosecond()),
	}

	return &playerPb.GameHistoryItem{
		GameId:        gameID,
		GameTime:      gameTimeProto,
		WinAmount:     winAmount,
		BetAmount:     betAmount,
		ResultSummary: resultSummary,
	}
}
