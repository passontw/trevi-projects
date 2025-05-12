package dealer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	newpb "g38_lottery_service/internal/generated/api/v1/lottery"
	commonpb "g38_lottery_service/internal/generated/common"
	oldpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"
)

func TestConvertGameStage(t *testing.T) {
	tests := []struct {
		name     string
		newStage commonpb.GameStage
		oldStage oldpb.GameStage
	}{
		{
			name:     "準備階段",
			newStage: commonpb.GameStage_GAME_STAGE_PREPARATION,
			oldStage: oldpb.GameStage_GAME_STAGE_PREPARATION,
		},
		{
			name:     "新局開始",
			newStage: commonpb.GameStage_GAME_STAGE_NEW_ROUND,
			oldStage: oldpb.GameStage_GAME_STAGE_NEW_ROUND,
		},
		{
			name:     "遊戲結束",
			newStage: commonpb.GameStage_GAME_STAGE_GAME_OVER,
			oldStage: oldpb.GameStage_GAME_STAGE_GAME_OVER,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertGameStage(tt.newStage)
			assert.Equal(t, tt.oldStage, result)

			// 反向轉換
			reverse := ConvertGameStageToNew(tt.oldStage)
			assert.Equal(t, tt.newStage, reverse)
		})
	}
}

func TestConvertExtraBallSide(t *testing.T) {
	tests := []struct {
		name    string
		newSide commonpb.ExtraBallSide
		oldSide oldpb.ExtraBallSide
	}{
		{
			name:    "未指定",
			newSide: commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED,
			oldSide: oldpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED,
		},
		{
			name:    "左側",
			newSide: commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT,
			oldSide: oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT,
		},
		{
			name:    "右側",
			newSide: commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT,
			oldSide: oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertExtraBallSide(tt.newSide)
			assert.Equal(t, tt.oldSide, result)

			// 反向轉換
			reverse := ConvertExtraBallSideToNew(tt.oldSide)
			assert.Equal(t, tt.newSide, reverse)
		})
	}
}

func TestConvertBall(t *testing.T) {
	now := time.Now()
	ts := timestamppb.New(now)

	newBall := &newpb.Ball{
		Number:    42,
		Type:      newpb.BallType_BALL_TYPE_REGULAR,
		IsLast:    true,
		Timestamp: ts,
	}

	t.Run("轉換球資訊", func(t *testing.T) {
		result := ConvertBall(newBall)
		assert.Equal(t, int32(42), result.Number)
		assert.Equal(t, oldpb.BallType_BALL_TYPE_REGULAR, result.Type)
		assert.True(t, result.IsLast)
		assert.Equal(t, ts, result.Timestamp)

		// 反向轉換
		reverse := ConvertBallToNew(result)
		assert.Equal(t, newBall.Number, reverse.Number)
		assert.Equal(t, newBall.Type, reverse.Type)
		assert.Equal(t, newBall.IsLast, reverse.IsLast)
		assert.Equal(t, newBall.Timestamp, reverse.Timestamp)
	})

	t.Run("處理空值", func(t *testing.T) {
		result := ConvertBall(nil)
		assert.Nil(t, result)

		reverseResult := ConvertBallToNew(nil)
		assert.Nil(t, reverseResult)
	})
}

func TestConvertGameStatus(t *testing.T) {
	newStatus := &commonpb.GameStatus{
		Stage:   commonpb.GameStage_GAME_STAGE_DRAWING_START,
		Message: "開始抽獎",
	}

	t.Run("轉換遊戲狀態", func(t *testing.T) {
		result := ConvertGameStatus(newStatus)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_DRAWING_START, result.Stage)
		assert.Equal(t, "開始抽獎", result.Message)

		// 反向轉換
		reverse := ConvertGameStatusToNew(result)
		assert.Equal(t, newStatus.Stage, reverse.Stage)
		assert.Equal(t, newStatus.Message, reverse.Message)
	})

	t.Run("處理空值", func(t *testing.T) {
		result := ConvertGameStatus(nil)
		assert.Nil(t, result)

		reverseResult := ConvertGameStatusToNew(nil)
		assert.Nil(t, reverseResult)
	})
}

func TestConvertGameData(t *testing.T) {
	now := time.Now()
	ts := timestamppb.New(now)

	regularBall := &newpb.Ball{
		Number:    1,
		Type:      newpb.BallType_BALL_TYPE_REGULAR,
		IsLast:    false,
		Timestamp: ts,
	}

	extraBall := &newpb.Ball{
		Number:    42,
		Type:      newpb.BallType_BALL_TYPE_EXTRA,
		IsLast:    true,
		Timestamp: ts,
	}

	jackpotBall := &newpb.Ball{
		Number:    75,
		Type:      newpb.BallType_BALL_TYPE_JACKPOT,
		IsLast:    true,
		Timestamp: ts,
	}

	luckyBall := &newpb.Ball{
		Number:    88,
		Type:      newpb.BallType_BALL_TYPE_LUCKY,
		IsLast:    true,
		Timestamp: ts,
	}

	newGameData := &newpb.GameData{
		GameId:                "game-123",
		RoomId:                "room-456",
		StartTime:             ts,
		EndTime:               ts,
		CurrentStage:          commonpb.GameStage_GAME_STAGE_DRAWING_START,
		RegularBalls:          []*newpb.Ball{regularBall},
		ExtraBall:             extraBall,
		JackpotBall:           jackpotBall,
		LuckyBalls:            []*newpb.Ball{luckyBall},
		SelectedExtraBallSide: commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT,
		Status:                &commonpb.GameStatus{Stage: commonpb.GameStage_GAME_STAGE_DRAWING_START, Message: "抽獎中"},
		IsValid:               true,
		CancelReason:          "",
	}

	t.Run("轉換遊戲數據", func(t *testing.T) {
		result := ConvertGameData(newGameData)
		assert.Equal(t, "game-123", result.GameId)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_DRAWING_START, result.CurrentStage)
		assert.Equal(t, ts, result.StartTime)
		assert.Equal(t, ts, result.EndTime)
		assert.Len(t, result.RegularBalls, 1)
		assert.Equal(t, int32(1), result.RegularBalls[0].Number)
		assert.Len(t, result.ExtraBalls, 1)
		assert.Equal(t, int32(42), result.ExtraBalls[0].Number)
		assert.Len(t, result.JackpotBalls, 1)
		assert.Equal(t, int32(75), result.JackpotBalls[0].Number)
		assert.Len(t, result.LuckyBalls, 1)
		assert.Equal(t, int32(88), result.LuckyBalls[0].Number)
		assert.Equal(t, oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT, result.SelectedSide)
	})

	t.Run("處理空值", func(t *testing.T) {
		result := ConvertGameData(nil)
		assert.Nil(t, result)
	})
}

func TestConvertResponseTypes(t *testing.T) {
	now := time.Now()
	ts := timestamppb.New(now)

	ball := &newpb.Ball{
		Number:    42,
		Type:      newpb.BallType_BALL_TYPE_REGULAR,
		IsLast:    true,
		Timestamp: ts,
	}

	status := &commonpb.GameStatus{
		Stage:   commonpb.GameStage_GAME_STAGE_DRAWING_START,
		Message: "開始抽獎",
	}

	t.Run("轉換 StartNewRoundResponse", func(t *testing.T) {
		newResp := &newpb.StartNewRoundResponse{
			GameId:       "game-123",
			StartTime:    ts,
			CurrentStage: commonpb.GameStage_GAME_STAGE_NEW_ROUND,
		}

		result := ConvertStartNewRoundResponse(newResp)
		assert.Equal(t, "game-123", result.GameId)
		assert.Equal(t, ts, result.StartTime)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_NEW_ROUND, result.CurrentStage)
	})

	t.Run("轉換 DrawBallResponse", func(t *testing.T) {
		newResp := &newpb.DrawBallResponse{
			GameId: "game-123",
			Balls:  []*newpb.Ball{ball},
			Status: status,
		}

		result := ConvertDrawBallResponse(newResp)
		assert.Len(t, result.Balls, 1)
		assert.Equal(t, int32(42), result.Balls[0].Number)
		assert.Equal(t, oldpb.BallType_BALL_TYPE_REGULAR, result.Balls[0].Type)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_DRAWING_START, result.GameStatus.Stage)
	})

	t.Run("轉換 DrawExtraBallResponse", func(t *testing.T) {
		newResp := &newpb.DrawExtraBallResponse{
			GameId:    "game-123",
			ExtraBall: ball,
			Status:    status,
		}

		result := ConvertDrawExtraBallResponse(newResp)
		assert.Len(t, result.Balls, 1)
		assert.Equal(t, int32(42), result.Balls[0].Number)
	})
}

func TestConvertGameEvent(t *testing.T) {
	now := time.Now()
	ts := timestamppb.New(now)
	gameId := "game-123"

	t.Run("測試 BallDrawn 事件", func(t *testing.T) {
		ball := &newpb.Ball{
			Number:    42,
			Type:      newpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    true,
			Timestamp: ts,
		}

		newEvent := &newpb.GameEvent{
			Type:      commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
			Timestamp: ts,
			GameId:    gameId,
			EventData: &newpb.GameEvent_BallDrawn{
				BallDrawn: &newpb.BallDrawnEvent{
					Ball: ball,
				},
			},
		}

		result := ConvertGameEvent(newEvent)
		assert.Equal(t, oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT, result.EventType)
		assert.Equal(t, ts, result.Timestamp)
		assert.Equal(t, gameId, result.GameId)

		ballDrawnEvent, ok := result.EventData.(*oldpb.GameEvent_BallDrawn)
		assert.True(t, ok, "應該轉換為 BallDrawn 事件")
		assert.Equal(t, int32(42), ballDrawnEvent.BallDrawn.Ball.Number)
		assert.Equal(t, oldpb.BallType_BALL_TYPE_REGULAR, ballDrawnEvent.BallDrawn.Ball.Type)
		assert.True(t, ballDrawnEvent.BallDrawn.Ball.IsLast)
	})

	t.Run("測試 StageChanged 事件", func(t *testing.T) {
		newEvent := &newpb.GameEvent{
			Type:      commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
			Timestamp: ts,
			GameId:    gameId,
			EventData: &newpb.GameEvent_StageChanged{
				StageChanged: &newpb.StageChangedEvent{
					OldStage: commonpb.GameStage_GAME_STAGE_PREPARATION,
					NewStage: commonpb.GameStage_GAME_STAGE_DRAWING_START,
				},
			},
		}

		result := ConvertGameEvent(newEvent)
		assert.Equal(t, oldpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION, result.EventType)

		stageChangedEvent, ok := result.EventData.(*oldpb.GameEvent_StageChanged)
		assert.True(t, ok, "應該轉換為 StageChanged 事件")
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_PREPARATION, stageChangedEvent.StageChanged.OldStage)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_DRAWING_START, stageChangedEvent.StageChanged.NewStage)
	})

	t.Run("測試 NewGame 事件", func(t *testing.T) {
		regularBall := &newpb.Ball{
			Number:    1,
			Type:      newpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    false,
			Timestamp: ts,
		}

		gameData := &newpb.GameData{
			GameId:       gameId,
			StartTime:    ts,
			CurrentStage: commonpb.GameStage_GAME_STAGE_NEW_ROUND,
			RegularBalls: []*newpb.Ball{regularBall},
		}

		newEvent := &newpb.GameEvent{
			Type:      commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
			Timestamp: ts,
			GameId:    gameId,
			EventData: &newpb.GameEvent_NewGame{
				NewGame: &newpb.NewGameEvent{
					GameData: gameData,
				},
			},
		}

		result := ConvertGameEvent(newEvent)
		assert.Equal(t, oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT, result.EventType)

		gameCreatedEvent, ok := result.EventData.(*oldpb.GameEvent_GameCreated)
		assert.True(t, ok, "應該轉換為 GameCreated 事件")
		assert.Equal(t, gameId, gameCreatedEvent.GameCreated.InitialState.GameId)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_NEW_ROUND, gameCreatedEvent.GameCreated.InitialState.CurrentStage)
		assert.Len(t, gameCreatedEvent.GameCreated.InitialState.RegularBalls, 1)
	})

	t.Run("測試 GameCancelled 事件", func(t *testing.T) {
		cancelReason := "操作員取消"

		newEvent := &newpb.GameEvent{
			Type:      commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
			Timestamp: ts,
			GameId:    gameId,
			EventData: &newpb.GameEvent_GameCancelled{
				GameCancelled: &newpb.GameCancelledEvent{
					Reason: cancelReason,
				},
			},
		}

		result := ConvertGameEvent(newEvent)
		assert.Equal(t, oldpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION, result.EventType)

		gameCancelledEvent, ok := result.EventData.(*oldpb.GameEvent_GameCancelled)
		assert.True(t, ok, "應該轉換為 GameCancelled 事件")
		assert.Equal(t, cancelReason, gameCancelledEvent.GameCancelled.Reason)
		assert.NotNil(t, gameCancelledEvent.GameCancelled.CancelTime)
	})

	t.Run("測試 ExtraBallSideSelected 事件", func(t *testing.T) {
		newEvent := &newpb.GameEvent{
			Type:      commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
			Timestamp: ts,
			GameId:    gameId,
			EventData: &newpb.GameEvent_ExtraBallSideSelected{
				ExtraBallSideSelected: &newpb.ExtraBallSideSelectedEvent{
					Side: commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT,
				},
			},
		}

		result := ConvertGameEvent(newEvent)
		assert.Equal(t, oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT, result.EventType)

		sideSelectedEvent, ok := result.EventData.(*oldpb.GameEvent_ExtraBallSideSelected)
		assert.True(t, ok, "應該轉換為 ExtraBallSideSelected 事件")
		assert.Equal(t, oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT, sideSelectedEvent.ExtraBallSideSelected.SelectedSide)
	})

	t.Run("測試 Heartbeat 事件", func(t *testing.T) {
		count := int32(42)

		newEvent := &newpb.GameEvent{
			Type:      commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
			Timestamp: ts,
			GameId:    gameId,
			EventData: &newpb.GameEvent_Heartbeat{
				Heartbeat: &newpb.HeartbeatEvent{
					Count: count,
				},
			},
		}

		result := ConvertGameEvent(newEvent)
		assert.Equal(t, oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT, result.EventType)

		heartbeatEvent, ok := result.EventData.(*oldpb.GameEvent_Heartbeat)
		assert.True(t, ok, "應該轉換為 Heartbeat 事件")
		assert.Contains(t, heartbeatEvent.Heartbeat.Message, "42")
	})

	t.Run("處理空值", func(t *testing.T) {
		result := ConvertGameEvent(nil)
		assert.Nil(t, result)
	})
}

func TestAdditionalConversionFunctions(t *testing.T) {
	now := time.Now()
	ts := timestamppb.New(now)
	gameId := "game-123"

	ball := &newpb.Ball{
		Number:    42,
		Type:      newpb.BallType_BALL_TYPE_JACKPOT,
		IsLast:    true,
		Timestamp: ts,
	}

	regularBall := &newpb.Ball{
		Number:    1,
		Type:      newpb.BallType_BALL_TYPE_REGULAR,
		IsLast:    false,
		Timestamp: ts,
	}

	gameData := &newpb.GameData{
		GameId:       gameId,
		StartTime:    ts,
		CurrentStage: commonpb.GameStage_GAME_STAGE_NEW_ROUND,
		RegularBalls: []*newpb.Ball{regularBall},
	}

	t.Run("測試 ConvertDrawJackpotBallResponse", func(t *testing.T) {
		newResp := &newpb.DrawJackpotBallResponse{
			GameId:      gameId,
			JackpotBall: ball,
		}

		result := ConvertDrawJackpotBallResponse(newResp)
		assert.Len(t, result.Balls, 1)
		assert.Equal(t, int32(42), result.Balls[0].Number)
		assert.Equal(t, oldpb.BallType_BALL_TYPE_JACKPOT, result.Balls[0].Type)
		assert.True(t, result.Balls[0].IsLast)
	})

	t.Run("測試 ConvertDrawLuckyBallResponse", func(t *testing.T) {
		newResp := &newpb.DrawLuckyBallResponse{
			GameId:     gameId,
			LuckyBalls: []*newpb.Ball{ball},
		}

		result := ConvertDrawLuckyBallResponse(newResp)
		assert.Len(t, result.Balls, 1)
		assert.Equal(t, int32(42), result.Balls[0].Number)
		assert.Equal(t, oldpb.BallType_BALL_TYPE_JACKPOT, result.Balls[0].Type)
	})

	t.Run("測試 ConvertGetGameStatusResponse", func(t *testing.T) {
		newResp := &newpb.GetGameStatusResponse{
			GameData: gameData,
		}

		result := ConvertGetGameStatusResponse(newResp)
		assert.NotNil(t, result.GameData)
		assert.Equal(t, gameId, result.GameData.GameId)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_NEW_ROUND, result.GameData.CurrentStage)
	})

	t.Run("測試 ConvertStartJackpotRoundResponse", func(t *testing.T) {
		newResp := &newpb.StartJackpotRoundResponse{
			GameId:       gameId,
			CurrentStage: commonpb.GameStage_GAME_STAGE_JACKPOT_START,
		}

		result := ConvertStartJackpotRoundResponse(newResp)
		assert.True(t, result.Success)
		assert.Equal(t, gameId, result.GameId)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT, result.OldStage)
		assert.Equal(t, oldpb.GameStage_GAME_STAGE_JACKPOT_START, result.NewStage)
	})

	t.Run("處理空值", func(t *testing.T) {
		assert.Nil(t, ConvertDrawJackpotBallResponse(nil))
		assert.Nil(t, ConvertDrawLuckyBallResponse(nil))
		assert.Nil(t, ConvertGetGameStatusResponse(nil))
		assert.Nil(t, ConvertStartJackpotRoundResponse(nil))
	})
}

// ConvertGameStatusToNew 將舊的 GameStatus 轉換為新的 GameStatus (測試用)
func ConvertGameStatusToNew(status *oldpb.GameStatus) *commonpb.GameStatus {
	if status == nil {
		return nil
	}
	return &commonpb.GameStatus{
		Stage:   ConvertGameStageToNew(status.Stage),
		Message: status.Message,
	}
}
