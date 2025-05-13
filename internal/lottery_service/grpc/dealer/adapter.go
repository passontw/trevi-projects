package dealer

import (
	"crypto/rand"
	"math/big"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"
)

// 這個文件作為適配器層，用於處理不同API格式之間的轉換

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

// 簡化的轉換函數，直接返回原始值
// 因為新版的proto結構已經替代舊版本

// ConvertGameStageToDealerPb 將遊戲階段轉換
func ConvertGameStageToDealerPb(stage commonpb.GameStage) commonpb.GameStage {
	return stage
}

// ConvertGameStageFromDealerPb 將遊戲階段轉換
func ConvertGameStageFromDealerPb(stage commonpb.GameStage) commonpb.GameStage {
	return stage
}

// ConvertExtraBallSideToDealerPb 將額外球邊轉換
func ConvertExtraBallSideToDealerPb(side commonpb.ExtraBallSide) commonpb.ExtraBallSide {
	return side
}

// ConvertExtraBallSideFromDealerPb 將額外球邊轉換
func ConvertExtraBallSideFromDealerPb(side commonpb.ExtraBallSide) commonpb.ExtraBallSide {
	return side
}

// ConvertGameEventTypeToDealerPb 將事件類型轉換
func ConvertGameEventTypeToDealerPb(eventType commonpb.GameEventType) commonpb.GameEventType {
	return eventType
}

// ConvertGameEventTypeFromDealerPb 將事件類型轉換
func ConvertGameEventTypeFromDealerPb(eventType commonpb.GameEventType) commonpb.GameEventType {
	return eventType
}

// ConvertGameStatusToDealerPb 將遊戲狀態轉換
func ConvertGameStatusToDealerPb(status *commonpb.GameStatus) *commonpb.GameStatus {
	return status
}

// ConvertGameStatusFromDealerPb 將遊戲狀態轉換
func ConvertGameStatusFromDealerPb(status *commonpb.GameStatus) *commonpb.GameStatus {
	return status
}

// ConvertBallToDealerPb 將球轉換為 dealerpb.Ball
func ConvertBallToDealerPb(number int32) *dealerpb.Ball {
	// 獲取球號顏色
	color := "white"
	if number <= 15 {
		color = "red"
	} else if number <= 30 {
		color = "yellow"
	} else if number <= 45 {
		color = "blue"
	} else if number <= 60 {
		color = "green"
	} else {
		color = "purple"
	}

	// 生成球ID
	ballID := GenerateRandomString(8)

	return &dealerpb.Ball{
		Id:      ballID,
		Number:  number,
		Color:   color,
		IsOdd:   number%2 == 1,
		IsSmall: number <= 20,
	}
}

// ConvertBallFromDealerPb 將 dealerpb.Ball 轉換
func ConvertBallFromDealerPb(ball *dealerpb.Ball) *dealerpb.Ball {
	return ball
}

// ConvertBallsToDealerPb 將 Ball 陣列轉換為 dealerpb.Ball 陣列
func ConvertBallsToDealerPb(balls []*dealerpb.Ball) []*dealerpb.Ball {
	return balls
}

// ConvertGameDataToNewPb 將 dealerpb.GameData 轉換
func ConvertGameDataToNewPb(data *dealerpb.GameData) (*dealerpb.GameData, error) {
	return data, nil
}

// ConvertGameDataToDealerPb 將 dealerpb.GameData 轉換
func ConvertGameDataToDealerPb(data *dealerpb.GameData) (*dealerpb.GameData, error) {
	return data, nil
}
