package dealer

import (
	"fmt"
	"math/rand"

	"google.golang.org/protobuf/types/known/timestamppb"

	oldpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	newpb "g38_lottery_service/internal/generated/api/v1/lottery"
	commonpb "g38_lottery_service/internal/generated/common"
)

// 這個文件作為過渡期間的適配器，將新的 proto 定義映射到舊的 proto 定義
// 在完全遷移到新的 proto 之前，這個適配器允許新舊代碼共存

// ConvertGameStage 將新的 GameStage 轉換為舊的 GameStage
func ConvertGameStage(stage commonpb.GameStage) oldpb.GameStage {
	return oldpb.GameStage(stage)
}

// ConvertGameStageToNew 將舊的 GameStage 轉換為新的 GameStage
func ConvertGameStageToNew(stage oldpb.GameStage) commonpb.GameStage {
	return commonpb.GameStage(stage)
}

// ConvertExtraBallSide 將新的 ExtraBallSide 轉換為舊的 ExtraBallSide
func ConvertExtraBallSide(side commonpb.ExtraBallSide) oldpb.ExtraBallSide {
	return oldpb.ExtraBallSide(side)
}

// ConvertExtraBallSideToNew 將舊的 ExtraBallSide 轉換為新的 ExtraBallSide
func ConvertExtraBallSideToNew(side oldpb.ExtraBallSide) commonpb.ExtraBallSide {
	return commonpb.ExtraBallSide(side)
}

// ConvertGameEventType 將新的 GameEventType 轉換為舊的 GameEventType
func ConvertGameEventType(eventType commonpb.GameEventType) oldpb.GameEventType {
	return oldpb.GameEventType(eventType)
}

// ConvertBall 將新的 Ball 轉換為舊的 Ball
func ConvertBall(ball *newpb.Ball) *oldpb.Ball {
	if ball == nil {
		return nil
	}
	return &oldpb.Ball{
		Number:    ball.Number,
		Type:      oldpb.BallType(ball.Type),
		IsLast:    ball.IsLast,
		Timestamp: ball.Timestamp,
	}
}

// ConvertBallToNew 將舊的 Ball 轉換為新的 Ball
func ConvertBallToNew(ball *oldpb.Ball) *newpb.Ball {
	if ball == nil {
		return nil
	}
	return &newpb.Ball{
		Number:    ball.Number,
		Type:      newpb.BallType(ball.Type),
		IsLast:    ball.IsLast,
		Timestamp: ball.Timestamp,
	}
}

// ConvertGameData 將新的 GameData 轉換為舊的 GameData
func ConvertGameData(gameData *newpb.GameData) *oldpb.GameData {
	if gameData == nil {
		return nil
	}

	regularBalls := make([]*oldpb.Ball, 0, len(gameData.RegularBalls))
	for _, ball := range gameData.RegularBalls {
		regularBalls = append(regularBalls, ConvertBall(ball))
	}

	luckyBalls := make([]*oldpb.Ball, 0, len(gameData.LuckyBalls))
	for _, ball := range gameData.LuckyBalls {
		luckyBalls = append(luckyBalls, ConvertBall(ball))
	}

	result := &oldpb.GameData{
		GameId:       gameData.GameId,
		CurrentStage: ConvertGameStage(gameData.CurrentStage),
		StartTime:    gameData.StartTime,
		EndTime:      gameData.EndTime,
		RegularBalls: regularBalls,
		SelectedSide: ConvertExtraBallSide(gameData.SelectedExtraBallSide),
	}

	// 如果有額外球，轉換額外球
	if gameData.ExtraBall != nil {
		result.ExtraBalls = []*oldpb.Ball{ConvertBall(gameData.ExtraBall)}
	}

	// 如果有頭獎球，轉換頭獎球
	if gameData.JackpotBall != nil {
		result.JackpotBalls = []*oldpb.Ball{ConvertBall(gameData.JackpotBall)}
	}

	// 設置幸運球
	result.LuckyBalls = luckyBalls

	return result
}

// ConvertGameStatus 將新的 GameStatus 轉換為舊的 GameStatus
func ConvertGameStatus(status *commonpb.GameStatus) *oldpb.GameStatus {
	if status == nil {
		return nil
	}
	return &oldpb.GameStatus{
		Stage:   ConvertGameStage(status.Stage),
		Message: status.Message,
	}
}

// ConvertGameEvent 將新的 GameEvent 轉換為舊的 GameEvent
func ConvertGameEvent(event *newpb.GameEvent) *oldpb.GameEvent {
	if event == nil {
		return nil
	}

	result := &oldpb.GameEvent{
		EventType: ConvertGameEventType(event.Type),
		Timestamp: event.Timestamp,
		GameId:    event.GameId,
	}

	// 根據事件類型轉換對應的事件數據
	switch x := event.EventData.(type) {
	case *newpb.GameEvent_BallDrawn:
		result.EventData = &oldpb.GameEvent_BallDrawn{
			BallDrawn: &oldpb.BallDrawnEvent{
				Ball: ConvertBall(x.BallDrawn.Ball),
			},
		}
	case *newpb.GameEvent_StageChanged:
		result.EventData = &oldpb.GameEvent_StageChanged{
			StageChanged: &oldpb.StageChangedEvent{
				OldStage: ConvertGameStage(x.StageChanged.OldStage),
				NewStage: ConvertGameStage(x.StageChanged.NewStage),
			},
		}
	case *newpb.GameEvent_NewGame:
		result.EventData = &oldpb.GameEvent_GameCreated{
			GameCreated: &oldpb.GameCreatedEvent{
				InitialState: ConvertGameData(x.NewGame.GameData),
			},
		}
	case *newpb.GameEvent_GameCancelled:
		result.EventData = &oldpb.GameEvent_GameCancelled{
			GameCancelled: &oldpb.GameCancelledEvent{
				Reason:     x.GameCancelled.Reason,
				CancelTime: timestamppb.Now(),
			},
		}
	case *newpb.GameEvent_ExtraBallSideSelected:
		result.EventData = &oldpb.GameEvent_ExtraBallSideSelected{
			ExtraBallSideSelected: &oldpb.ExtraBallSideSelectedEvent{
				SelectedSide: ConvertExtraBallSide(x.ExtraBallSideSelected.Side),
			},
		}
	case *newpb.GameEvent_Heartbeat:
		result.EventData = &oldpb.GameEvent_Heartbeat{
			Heartbeat: &oldpb.HeartbeatEvent{
				Message: fmt.Sprintf("Heartbeat: %d", x.Heartbeat.Count),
			},
		}
	}

	return result
}

// ConvertStartNewRoundResponse 將新的 StartNewRoundResponse 轉換為舊的 StartNewRoundResponse
func ConvertStartNewRoundResponse(resp *newpb.StartNewRoundResponse) *oldpb.StartNewRoundResponse {
	if resp == nil {
		return nil
	}
	return &oldpb.StartNewRoundResponse{
		GameId:       resp.GameId,
		StartTime:    resp.StartTime,
		CurrentStage: ConvertGameStage(resp.CurrentStage),
	}
}

// ConvertDrawBallResponse 將新的 DrawBallResponse 轉換為舊的 DrawBallResponse
func ConvertDrawBallResponse(resp *newpb.DrawBallResponse) *oldpb.DrawBallResponse {
	if resp == nil {
		return nil
	}

	balls := make([]*oldpb.Ball, 0, len(resp.Balls))
	for _, ball := range resp.Balls {
		balls = append(balls, ConvertBall(ball))
	}

	return &oldpb.DrawBallResponse{
		Balls:      balls,
		GameStatus: ConvertGameStatus(resp.Status),
	}
}

// ConvertDrawExtraBallResponse 將新的 DrawExtraBallResponse 轉換為舊的 DrawExtraBallResponse
func ConvertDrawExtraBallResponse(resp *newpb.DrawExtraBallResponse) *oldpb.DrawExtraBallResponse {
	if resp == nil {
		return nil
	}

	balls := []*oldpb.Ball{}
	if resp.ExtraBall != nil {
		balls = append(balls, ConvertBall(resp.ExtraBall))
	}

	return &oldpb.DrawExtraBallResponse{
		Balls: balls,
	}
}

// ConvertDrawJackpotBallResponse 將新的 DrawJackpotBallResponse 轉換為舊的 DrawJackpotBallResponse
func ConvertDrawJackpotBallResponse(resp *newpb.DrawJackpotBallResponse) *oldpb.DrawJackpotBallResponse {
	if resp == nil {
		return nil
	}

	balls := []*oldpb.Ball{}
	if resp.JackpotBall != nil {
		balls = append(balls, ConvertBall(resp.JackpotBall))
	}

	return &oldpb.DrawJackpotBallResponse{
		Balls: balls,
	}
}

// ConvertDrawLuckyBallResponse 將新的 DrawLuckyBallResponse 轉換為舊的 DrawLuckyBallResponse
func ConvertDrawLuckyBallResponse(resp *newpb.DrawLuckyBallResponse) *oldpb.DrawLuckyBallResponse {
	if resp == nil {
		return nil
	}

	balls := make([]*oldpb.Ball, 0, len(resp.LuckyBalls))
	for _, ball := range resp.LuckyBalls {
		balls = append(balls, ConvertBall(ball))
	}

	return &oldpb.DrawLuckyBallResponse{
		Balls: balls,
	}
}

// ConvertGetGameStatusResponse 將新的 GetGameStatusResponse 轉換為舊的 GetGameStatusResponse
func ConvertGetGameStatusResponse(resp *newpb.GetGameStatusResponse) *oldpb.GetGameStatusResponse {
	if resp == nil {
		return nil
	}
	return &oldpb.GetGameStatusResponse{
		GameData: ConvertGameData(resp.GameData),
	}
}

// ConvertStartJackpotRoundResponse 將新的 StartJackpotRoundResponse 轉換為舊的 StartJackpotRoundResponse
func ConvertStartJackpotRoundResponse(resp *newpb.StartJackpotRoundResponse) *oldpb.StartJackpotRoundResponse {
	if resp == nil {
		return nil
	}
	return &oldpb.StartJackpotRoundResponse{
		Success:  true,
		GameId:   resp.GameId,
		OldStage: oldpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT,
		NewStage: ConvertGameStage(resp.CurrentStage),
	}
}

// 以下是用於將舊版 proto 轉換為新版 proto 的函數

// ConvertBallReverse 將新的 Ball 轉換為舊的 Ball (保留為向後兼容)
func ConvertBallReverse(ball *newpb.Ball) *oldpb.Ball {
	return ConvertBall(ball)
}

// GenerateRandomString 生成隨機字符串，用作 ID
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
