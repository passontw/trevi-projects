package dealer

import (
	"crypto/rand"
	"math/big"
	"time"

	newpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"
	oldpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// 這個文件作為適配器層，將新版的 API 格式轉換為舊的 proto 格式，反之亦然

// GenerateRandomString 生成指定長度的隨機字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// ConvertGameStageToDealerPb 將遊戲階段轉換為舊版 oldpb.GameStage
func ConvertGameStageToDealerPb(stage commonpb.GameStage) oldpb.GameStage {
	switch stage {
	case commonpb.GameStage_GAME_STAGE_UNSPECIFIED:
		return oldpb.GameStage_GAME_STAGE_UNSPECIFIED
	case commonpb.GameStage_GAME_STAGE_PREPARATION:
		return oldpb.GameStage_GAME_STAGE_PREPARATION
	case commonpb.GameStage_GAME_STAGE_NEW_ROUND:
		return oldpb.GameStage_GAME_STAGE_NEW_ROUND
	case commonpb.GameStage_GAME_STAGE_GAME_OVER:
		return oldpb.GameStage_GAME_STAGE_GAME_OVER
	case commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		return oldpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE:
		return oldpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case commonpb.GameStage_GAME_STAGE_DRAWING_START:
		return oldpb.GameStage_GAME_STAGE_DRAWING_START
	case commonpb.GameStage_GAME_STAGE_DRAWING_CLOSE:
		return oldpb.GameStage_GAME_STAGE_DRAWING_CLOSE
	case commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		return oldpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	case commonpb.GameStage_GAME_STAGE_JACKPOT_START:
		return oldpb.GameStage_GAME_STAGE_JACKPOT_START
	default:
		return oldpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// ConvertGameStageFromDealerPb 將舊版 oldpb.GameStage 轉換為 commonpb.GameStage
func ConvertGameStageFromDealerPb(stage oldpb.GameStage) commonpb.GameStage {
	switch stage {
	case oldpb.GameStage_GAME_STAGE_UNSPECIFIED:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	case oldpb.GameStage_GAME_STAGE_PREPARATION:
		return commonpb.GameStage_GAME_STAGE_PREPARATION
	case oldpb.GameStage_GAME_STAGE_NEW_ROUND:
		return commonpb.GameStage_GAME_STAGE_NEW_ROUND
	case oldpb.GameStage_GAME_STAGE_GAME_OVER:
		return commonpb.GameStage_GAME_STAGE_GAME_OVER
	case oldpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case oldpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case oldpb.GameStage_GAME_STAGE_DRAWING_START:
		return commonpb.GameStage_GAME_STAGE_DRAWING_START
	case oldpb.GameStage_GAME_STAGE_DRAWING_CLOSE:
		return commonpb.GameStage_GAME_STAGE_DRAWING_CLOSE
	case oldpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		return commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	case oldpb.GameStage_GAME_STAGE_JACKPOT_START:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_START
	default:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// ConvertExtraBallSideToDealerPb 將額外球邊轉換為舊版 oldpb.ExtraBallSide
func ConvertExtraBallSideToDealerPb(side commonpb.ExtraBallSide) oldpb.ExtraBallSide {
	switch side {
	case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return oldpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// ConvertExtraBallSideFromDealerPb 將舊版 oldpb.ExtraBallSide 轉換為 commonpb.ExtraBallSide
func ConvertExtraBallSideFromDealerPb(side oldpb.ExtraBallSide) commonpb.ExtraBallSide {
	switch side {
	case oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// ConvertGameEventTypeToDealerPb 將事件類型轉換為舊版 oldpb.GameEventType
func ConvertGameEventTypeToDealerPb(eventType commonpb.GameEventType) oldpb.GameEventType {
	switch eventType {
	case commonpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED:
		return oldpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	case commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION:
		return oldpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION
	case commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT:
		return oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT
	default:
		return oldpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	}
}

// ConvertGameEventTypeFromDealerPb 將舊版 oldpb.GameEventType 轉換為 commonpb.GameEventType
func ConvertGameEventTypeFromDealerPb(eventType oldpb.GameEventType) commonpb.GameEventType {
	switch eventType {
	case oldpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED:
		return commonpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	case oldpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION:
		return commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION
	case oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT:
		return commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT
	default:
		return commonpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	}
}

// ConvertGameStatusToDealerPb 將遊戲狀態轉換為舊版 oldpb.GameStatus
func ConvertGameStatusToDealerPb(status *commonpb.GameStatus) *oldpb.GameStatus {
	if status == nil {
		return nil
	}
	return &oldpb.GameStatus{
		Stage:   ConvertGameStageToDealerPb(status.Stage),
		Message: status.Message,
	}
}

// ConvertGameStatusFromDealerPb 將舊版 oldpb.GameStatus 轉換為 commonpb.GameStatus
func ConvertGameStatusFromDealerPb(status *oldpb.GameStatus) *commonpb.GameStatus {
	if status == nil {
		return nil
	}
	return &commonpb.GameStatus{
		Stage:   ConvertGameStageFromDealerPb(status.Stage),
		Message: status.Message,
	}
}

// ConvertBallToDealerPb 將球轉換為舊版 oldpb.Ball
func ConvertBallToDealerPb(number int32, ballType oldpb.BallType, isLast bool) *oldpb.Ball {
	return &oldpb.Ball{
		Number:    number,
		Type:      ballType,
		IsLast:    isLast,
		Timestamp: timestamppb.Now(),
	}
}

// ConvertBallsFromDealerPb 將舊版 oldpb.Ball 陣列轉換為新版 Ball 陣列
func ConvertBallsFromDealerPb(balls []*oldpb.Ball) []*oldpb.Ball {
	if balls == nil {
		return nil
	}
	result := make([]*oldpb.Ball, 0, len(balls))
	for _, ball := range balls {
		result = append(result, ball)
	}
	return result
}

// ConvertBallFromDealerPb 將舊版 oldpb.Ball 轉換為新版 newpb.Ball
func ConvertBallFromDealerPb(ball *oldpb.Ball) *newpb.Ball {
	if ball == nil {
		return nil
	}

	// 獲取球號顏色
	color := "white"
	if ball.Number <= 15 {
		color = "red"
	} else if ball.Number <= 30 {
		color = "yellow"
	} else if ball.Number <= 45 {
		color = "blue"
	} else if ball.Number <= 60 {
		color = "green"
	} else {
		color = "purple"
	}

	// 生成球ID
	ballID := GenerateRandomString(8)

	return &newpb.Ball{
		Id:      ballID,
		Number:  ball.Number,
		Color:   color,
		IsOdd:   ball.Number%2 == 1,
		IsSmall: ball.Number <= 20,
	}
}

// ConvertNewBallToDealerPb 將新版 newpb.Ball 轉換為舊版 oldpb.Ball
func ConvertNewBallToDealerPb(ball *newpb.Ball, ballType oldpb.BallType) *oldpb.Ball {
	if ball == nil {
		return nil
	}

	return &oldpb.Ball{
		Number:    ball.Number,
		Type:      ballType,
		IsLast:    false, // 默認不是最後一個球
		Timestamp: timestamppb.Now(),
	}
}

// ConvertBallsToDealerPb 將新版 newpb.Ball 陣列轉換為舊版 oldpb.Ball 陣列
func ConvertBallsToDealerPb(balls []*newpb.Ball, ballType oldpb.BallType) []*oldpb.Ball {
	if balls == nil {
		return nil
	}

	result := make([]*oldpb.Ball, 0, len(balls))
	for i, ball := range balls {
		isLast := (i == len(balls)-1) // 最後一個球標記為最後一個

		oldBall := &oldpb.Ball{
			Number:    ball.Number,
			Type:      ballType,
			IsLast:    isLast,
			Timestamp: timestamppb.Now(),
		}
		result = append(result, oldBall)
	}

	return result
}

// ConvertGameDataToNewPb 將舊版 oldpb.GameData 轉換為新版 newpb.GameData
func ConvertGameDataToNewPb(oldData *oldpb.GameData) (*newpb.GameData, error) {
	if oldData == nil {
		return nil, nil
	}

	// 建立遊戲數據
	newData := &newpb.GameData{
		Id:        oldData.GameId,
		RoomId:    "SG01", // 使用默認房間ID，可以從遊戲ID中解析
		Stage:     ConvertGameStageFromDealerPb(oldData.CurrentStage),
		Status:    newpb.GameStatus_GAME_STATUS_RUNNING, // 根據階段設置狀態
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 設置創建和更新時間
	if oldData.StartTime != nil {
		newData.CreatedAt = oldData.StartTime.AsTime().Unix()
	}

	if oldData.LastUpdateTime != nil {
		newData.UpdatedAt = oldData.LastUpdateTime.AsTime().Unix()
	}

	// 根據階段更精確地設置狀態
	switch oldData.CurrentStage {
	case oldpb.GameStage_GAME_STAGE_PREPARATION:
		newData.Status = newpb.GameStatus_GAME_STATUS_NOT_STARTED
	case oldpb.GameStage_GAME_STAGE_GAME_OVER:
		newData.Status = newpb.GameStatus_GAME_STATUS_COMPLETED
	}

	// 處理已抽出的常規球
	if len(oldData.RegularBalls) > 0 {
		newData.DrawnBalls = make([]*newpb.Ball, 0, len(oldData.RegularBalls))
		for _, ball := range oldData.RegularBalls {
			newBall := ConvertBallFromDealerPb(ball)
			if newBall != nil {
				newData.DrawnBalls = append(newData.DrawnBalls, newBall)
			}
		}
	}

	// 處理額外球
	if len(oldData.ExtraBalls) > 0 {
		newData.ExtraBalls = make(map[string]*newpb.Ball)
		for i, ball := range oldData.ExtraBalls {
			// 使用索引作為臨時鍵
			key := ""
			if i == 0 {
				key = "left"
			} else if i == 1 {
				key = "right"
			} else {
				key = "extra_" + string(rune('a'+i))
			}

			newBall := ConvertBallFromDealerPb(ball)
			if newBall != nil {
				newData.ExtraBalls[key] = newBall
			}
		}
	}

	// 處理頭獎球
	if len(oldData.JackpotBalls) > 0 && len(oldData.JackpotBalls) > 0 {
		newData.JackpotBall = ConvertBallFromDealerPb(oldData.JackpotBalls[0])
	}

	// 處理幸運球
	if len(oldData.LuckyBalls) > 0 {
		newData.LuckyBalls = make([]*newpb.Ball, 0, len(oldData.LuckyBalls))
		for _, ball := range oldData.LuckyBalls {
			newBall := ConvertBallFromDealerPb(ball)
			if newBall != nil {
				newData.LuckyBalls = append(newData.LuckyBalls, newBall)
			}
		}
	}

	return newData, nil
}

// ConvertGameDataToDealerPb 將新版 newpb.GameData 轉換為舊版 oldpb.GameData
func ConvertGameDataToDealerPb(newData *newpb.GameData) (*oldpb.GameData, error) {
	if newData == nil {
		return nil, nil
	}

	// 建立舊版遊戲數據結構
	oldData := &oldpb.GameData{
		GameId:         newData.Id,
		CurrentStage:   ConvertGameStageToDealerPb(newData.Stage),
		HasJackpot:     newData.JackpotBall != nil,
		ExtraBallCount: int32(len(newData.ExtraBalls)), // 根據新版數據中的額外球數量動態設置
	}

	// 確保合理的額外球數量
	if oldData.ExtraBallCount < 1 {
		oldData.ExtraBallCount = 2 // 默認有兩個額外球
	}

	// 設置時間相關欄位
	oldData.StartTime = timestamppb.New(time.Unix(newData.CreatedAt, 0))
	oldData.LastUpdateTime = timestamppb.New(time.Unix(newData.UpdatedAt, 0))

	// 如果狀態為已完成，設置結束時間
	if newData.Status == newpb.GameStatus_GAME_STATUS_COMPLETED {
		oldData.EndTime = timestamppb.New(time.Unix(newData.UpdatedAt, 0))
	}

	// 處理已抽出的常規球
	if len(newData.DrawnBalls) > 0 {
		oldData.RegularBalls = make([]*oldpb.Ball, 0, len(newData.DrawnBalls))
		for i, ball := range newData.DrawnBalls {
			isLast := (i == len(newData.DrawnBalls)-1) // 最後一個球標記為最後一個

			oldBall := &oldpb.Ball{
				Number:    ball.Number,
				Type:      oldpb.BallType_BALL_TYPE_REGULAR,
				IsLast:    isLast,
				Timestamp: timestamppb.New(time.Unix(newData.UpdatedAt, 0)), // 使用更新時間作為球時間戳
			}
			oldData.RegularBalls = append(oldData.RegularBalls, oldBall)
		}
	}

	// 處理額外球
	if len(newData.ExtraBalls) > 0 {
		oldData.ExtraBalls = make([]*oldpb.Ball, 0, len(newData.ExtraBalls))

		// 因為舊版API將extra球存儲為數組，而新版API存儲為map，這裡需要統一轉換
		for key, ball := range newData.ExtraBalls {
			ballType := oldpb.BallType_BALL_TYPE_EXTRA
			if key == "left" {
				oldData.SelectedSide = oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
			} else if key == "right" {
				oldData.SelectedSide = oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
			}

			oldBall := &oldpb.Ball{
				Number:    ball.Number,
				Type:      ballType,
				IsLast:    false, // 默認不是最後一個球
				Timestamp: timestamppb.New(time.Unix(newData.UpdatedAt, 0)),
			}
			oldData.ExtraBalls = append(oldData.ExtraBalls, oldBall)
		}
	}

	// 處理頭獎球
	if newData.JackpotBall != nil {
		jackpotBall := &oldpb.Ball{
			Number:    newData.JackpotBall.Number,
			Type:      oldpb.BallType_BALL_TYPE_JACKPOT,
			IsLast:    true, // JP球通常是唯一的，所以標記為最後一個
			Timestamp: timestamppb.New(time.Unix(newData.UpdatedAt, 0)),
		}
		oldData.JackpotBalls = []*oldpb.Ball{jackpotBall}
		oldData.HasJackpot = true
	}

	// 處理幸運球
	if len(newData.LuckyBalls) > 0 {
		oldData.LuckyBalls = make([]*oldpb.Ball, 0, len(newData.LuckyBalls))
		for i, ball := range newData.LuckyBalls {
			isLast := (i == len(newData.LuckyBalls)-1) // 最後一個球標記為最後一個

			oldBall := &oldpb.Ball{
				Number:    ball.Number,
				Type:      oldpb.BallType_BALL_TYPE_LUCKY,
				IsLast:    isLast,
				Timestamp: timestamppb.New(time.Unix(newData.UpdatedAt, 0)),
			}
			oldData.LuckyBalls = append(oldData.LuckyBalls, oldBall)
		}
	}

	return oldData, nil
}
