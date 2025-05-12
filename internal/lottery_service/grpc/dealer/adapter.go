package dealer

import (
	"crypto/rand"
	"math/big"

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
