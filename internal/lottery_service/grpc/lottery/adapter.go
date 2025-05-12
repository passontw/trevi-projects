package lottery

import (
	"crypto/rand"
	"math/big"

	lotterypb "g38_lottery_service/internal/generated/api/v1/lottery"
	commonpb "g38_lottery_service/internal/generated/common"
	dealerpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// 這個文件作為適配器層，將 dealerpb 類型轉換為 lottery 類型，反之亦然

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

// ConvertBallToDealerPb 將球轉換為 dealerpb.Ball
func ConvertBallToDealerPb(ball *lotterypb.Ball) *dealerpb.Ball {
	if ball == nil {
		return nil
	}
	return &dealerpb.Ball{
		Number:    ball.Number,
		Type:      dealerpb.BallType(ball.Type),
		IsLast:    ball.IsLast,
		Timestamp: ball.Timestamp,
	}
}

// ConvertBallFromDealerPb 將 dealerpb.Ball 轉換為 Ball
func ConvertBallFromDealerPb(ball *dealerpb.Ball) *lotterypb.Ball {
	if ball == nil {
		return nil
	}
	return &lotterypb.Ball{
		Number:    ball.Number,
		Type:      lotterypb.BallType(ball.Type),
		IsLast:    ball.IsLast,
		Timestamp: ball.Timestamp,
	}
}

// ConvertGameStageToDealerPb 將遊戲階段轉換為 dealerpb.GameStage
func ConvertGameStageToDealerPb(stage commonpb.GameStage) dealerpb.GameStage {
	switch stage {
	case commonpb.GameStage_GAME_STAGE_UNSPECIFIED:
		return dealerpb.GameStage_GAME_STAGE_UNSPECIFIED
	case commonpb.GameStage_GAME_STAGE_PREPARATION:
		return dealerpb.GameStage_GAME_STAGE_PREPARATION
	case commonpb.GameStage_GAME_STAGE_NEW_ROUND:
		return dealerpb.GameStage_GAME_STAGE_NEW_ROUND
	case commonpb.GameStage_GAME_STAGE_GAME_OVER:
		return dealerpb.GameStage_GAME_STAGE_GAME_OVER
	case commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		return dealerpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE:
		return dealerpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case commonpb.GameStage_GAME_STAGE_DRAWING_START:
		return dealerpb.GameStage_GAME_STAGE_DRAWING_START
	case commonpb.GameStage_GAME_STAGE_DRAWING_CLOSE:
		return dealerpb.GameStage_GAME_STAGE_DRAWING_CLOSE
	case commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		return dealerpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	default:
		return dealerpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// ConvertGameStageFromDealerPb 將 dealerpb.GameStage 轉換為 GameStage
func ConvertGameStageFromDealerPb(stage dealerpb.GameStage) commonpb.GameStage {
	switch stage {
	case dealerpb.GameStage_GAME_STAGE_UNSPECIFIED:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	case dealerpb.GameStage_GAME_STAGE_PREPARATION:
		return commonpb.GameStage_GAME_STAGE_PREPARATION
	case dealerpb.GameStage_GAME_STAGE_NEW_ROUND:
		return commonpb.GameStage_GAME_STAGE_NEW_ROUND
	case dealerpb.GameStage_GAME_STAGE_GAME_OVER:
		return commonpb.GameStage_GAME_STAGE_GAME_OVER
	case dealerpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case dealerpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case dealerpb.GameStage_GAME_STAGE_DRAWING_START:
		return commonpb.GameStage_GAME_STAGE_DRAWING_START
	case dealerpb.GameStage_GAME_STAGE_DRAWING_CLOSE:
		return commonpb.GameStage_GAME_STAGE_DRAWING_CLOSE
	case dealerpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		return commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	default:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// ConvertExtraBallSideToDealerPb 將額外球邊轉換為 dealerpb.ExtraBallSide
func ConvertExtraBallSideToDealerPb(side commonpb.ExtraBallSide) dealerpb.ExtraBallSide {
	switch side {
	case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return dealerpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return dealerpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return dealerpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// ConvertExtraBallSideFromDealerPb 將 dealerpb.ExtraBallSide 轉換為 ExtraBallSide
func ConvertExtraBallSideFromDealerPb(side dealerpb.ExtraBallSide) commonpb.ExtraBallSide {
	switch side {
	case dealerpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case dealerpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// ConvertGameDataFromDealerPb 將 dealerpb.GameData 轉換為 GameData
func ConvertGameDataFromDealerPb(gameData *dealerpb.GameData) *lotterypb.GameData {
	if gameData == nil {
		return nil
	}

	result := &lotterypb.GameData{
		GameId:       gameData.GameId,
		StartTime:    gameData.StartTime,
		EndTime:      gameData.EndTime,
		CurrentStage: ConvertGameStageFromDealerPb(gameData.CurrentStage),
		IsValid:      true,
	}

	// 轉換一般球
	regularBalls := make([]*lotterypb.Ball, 0, len(gameData.RegularBalls))
	for _, ball := range gameData.RegularBalls {
		regularBalls = append(regularBalls, ConvertBallFromDealerPb(ball))
	}
	result.RegularBalls = regularBalls

	// 轉換幸運球
	luckyBalls := make([]*lotterypb.Ball, 0, len(gameData.LuckyBalls))
	for _, ball := range gameData.LuckyBalls {
		luckyBalls = append(luckyBalls, ConvertBallFromDealerPb(ball))
	}
	result.LuckyBalls = luckyBalls

	// 設置額外球 (取第一個)
	if len(gameData.ExtraBalls) > 0 {
		result.ExtraBall = ConvertBallFromDealerPb(gameData.ExtraBalls[0])
	}

	// 設置頭獎球 (取第一個)
	if len(gameData.JackpotBalls) > 0 {
		result.JackpotBall = ConvertBallFromDealerPb(gameData.JackpotBalls[0])
	}

	// 設置選中的額外球邊
	result.SelectedExtraBallSide = ConvertExtraBallSideFromDealerPb(gameData.SelectedSide)

	// 設置遊戲狀態
	result.Status = &commonpb.GameStatus{
		Stage:   ConvertGameStageFromDealerPb(gameData.CurrentStage),
		Message: "",
	}

	return result
}

// ConvertGameDataToDealerPb 將 GameData 轉換為 dealerpb.GameData
func ConvertGameDataToDealerPb(gameData *lotterypb.GameData) *dealerpb.GameData {
	if gameData == nil {
		return nil
	}

	result := &dealerpb.GameData{
		GameId:       gameData.GameId,
		StartTime:    gameData.StartTime,
		EndTime:      gameData.EndTime,
		CurrentStage: ConvertGameStageToDealerPb(gameData.CurrentStage),
		RegularBalls: make([]*dealerpb.Ball, 0, len(gameData.RegularBalls)),
		LuckyBalls:   make([]*dealerpb.Ball, 0, len(gameData.LuckyBalls)),
		ExtraBalls:   make([]*dealerpb.Ball, 0, 1),
		JackpotBalls: make([]*dealerpb.Ball, 0, 1),
		SelectedSide: ConvertExtraBallSideToDealerPb(gameData.SelectedExtraBallSide),
	}

	// 轉換一般球
	for _, ball := range gameData.RegularBalls {
		result.RegularBalls = append(result.RegularBalls, ConvertBallToDealerPb(ball))
	}

	// 轉換幸運球
	for _, ball := range gameData.LuckyBalls {
		result.LuckyBalls = append(result.LuckyBalls, ConvertBallToDealerPb(ball))
	}

	// 設置額外球
	if gameData.ExtraBall != nil {
		result.ExtraBalls = append(result.ExtraBalls, ConvertBallToDealerPb(gameData.ExtraBall))
	}

	// 設置頭獎球
	if gameData.JackpotBall != nil {
		result.JackpotBalls = append(result.JackpotBalls, ConvertBallToDealerPb(gameData.JackpotBall))
	}

	// 設置 LastUpdateTime 為當前時間
	result.LastUpdateTime = timestamppb.Now()

	return result
}

// ConvertGameStatusToDealerPb 將新版 common.GameStatus 轉換為舊版 dealerpb.GameStatus
func ConvertGameStatusToDealerPb(status *commonpb.GameStatus) *dealerpb.GameStatus {
	if status == nil {
		return nil
	}
	return &dealerpb.GameStatus{
		Stage:   ConvertGameStageToDealerPb(status.Stage),
		Message: status.Message,
	}
}

// ConvertGameStatusFromDealerPb 將舊版 dealerpb.GameStatus 轉換為新版 common.GameStatus
func ConvertGameStatusFromDealerPb(status *dealerpb.GameStatus) *commonpb.GameStatus {
	if status == nil {
		return nil
	}
	return &commonpb.GameStatus{
		Stage:   ConvertGameStageFromDealerPb(status.Stage),
		Message: status.Message,
	}
}

// ConvertDrawBallResponse 將舊版 DrawBallResponse 轉換為新版
func ConvertDrawBallResponse(resp *dealerpb.DrawBallResponse) *lotterypb.DrawBallResponse {
	if resp == nil {
		return nil
	}

	result := &lotterypb.DrawBallResponse{
		GameId: "", // 舊版沒有 GameId
	}

	if len(resp.Balls) > 0 {
		result.Balls = make([]*lotterypb.Ball, 0, len(resp.Balls))
		for _, ball := range resp.Balls {
			result.Balls = append(result.Balls, ConvertBallFromDealerPb(ball))
		}
	}

	if resp.GameStatus != nil {
		result.Status = ConvertGameStatusFromDealerPb(resp.GameStatus)
	}

	return result
}

// ConvertDrawExtraBallResponse 將舊版 DrawExtraBallResponse 轉換為新版
func ConvertDrawExtraBallResponse(resp *dealerpb.DrawExtraBallResponse) *lotterypb.DrawExtraBallResponse {
	if resp == nil {
		return nil
	}

	result := &lotterypb.DrawExtraBallResponse{
		GameId: "", // 舊版中無此字段
	}

	if len(resp.Balls) > 0 {
		result.ExtraBall = ConvertBallFromDealerPb(resp.Balls[0])
	}

	return result
}

// ConvertDrawJackpotBallResponse 將舊版 DrawJackpotBallResponse 轉換為新版
func ConvertDrawJackpotBallResponse(resp *dealerpb.DrawJackpotBallResponse) *lotterypb.DrawJackpotBallResponse {
	if resp == nil {
		return nil
	}

	result := &lotterypb.DrawJackpotBallResponse{
		GameId: "", // 舊版無此字段
	}

	if len(resp.Balls) > 0 {
		result.JackpotBall = ConvertBallFromDealerPb(resp.Balls[0])
	}

	return result
}

// ConvertDrawLuckyBallResponse 將舊版 DrawLuckyBallResponse 轉換為新版
func ConvertDrawLuckyBallResponse(resp *dealerpb.DrawLuckyBallResponse) *lotterypb.DrawLuckyBallResponse {
	if resp == nil {
		return nil
	}

	result := &lotterypb.DrawLuckyBallResponse{
		GameId: "", // 舊版無此字段
	}

	if len(resp.Balls) > 0 {
		result.LuckyBalls = make([]*lotterypb.Ball, 0, len(resp.Balls))
		for _, ball := range resp.Balls {
			result.LuckyBalls = append(result.LuckyBalls, ConvertBallFromDealerPb(ball))
		}
	}

	return result
}

// ConvertGetGameStatusResponse 將舊版 GetGameStatusResponse 轉換為新版
func ConvertGetGameStatusResponse(resp *dealerpb.GetGameStatusResponse) *lotterypb.GetGameStatusResponse {
	if resp == nil {
		return nil
	}

	result := &lotterypb.GetGameStatusResponse{}

	if resp.GameData != nil {
		result.GameData = ConvertGameDataFromDealerPb(resp.GameData)
		result.GameId = resp.GameData.GameId
		result.CurrentStage = ConvertGameStageFromDealerPb(resp.GameData.CurrentStage)
		result.Status = &commonpb.GameStatus{
			Stage:   ConvertGameStageFromDealerPb(resp.GameData.CurrentStage),
			Message: "",
		}
	}

	return result
}

// ConvertStartJackpotRoundResponse 將舊版 StartJackpotRoundResponse 轉換為新版
func ConvertStartJackpotRoundResponse(resp *dealerpb.StartJackpotRoundResponse) *lotterypb.StartJackpotRoundResponse {
	if resp == nil {
		return nil
	}

	return &lotterypb.StartJackpotRoundResponse{
		GameId:       resp.GameId,
		CurrentStage: ConvertGameStageFromDealerPb(resp.NewStage),
		Status:       ConvertGameStatusFromDealerPb(&dealerpb.GameStatus{Stage: resp.NewStage}),
	}
}
