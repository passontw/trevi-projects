package player

import (
	"context"
	"time"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
	newpb "g38_lottery_service/internal/generated/api/v1/player"
	commonpb "g38_lottery_service/internal/generated/common"
	"g38_lottery_service/internal/lottery_service/gameflow"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PlayerCommunicationServiceWrapper 是新 API 到舊 API 的包裝器
type PlayerCommunicationServiceWrapper struct {
	newpb.UnimplementedPlayerCommunicationServiceServer
	logger      *zap.Logger
	gameManager *gameflow.GameManager
}

// NewPlayerCommunicationServiceWrapper 創建一個新的 PlayerCommunicationServiceWrapper 實例
func NewPlayerCommunicationServiceWrapper(
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
) *PlayerCommunicationServiceWrapper {
	return &PlayerCommunicationServiceWrapper{
		logger:      logger.Named("player_communication_wrapper"),
		gameManager: gameManager,
	}
}

// ConnectToGame 處理連接到遊戲的請求
func (w *PlayerCommunicationServiceWrapper) ConnectToGame(
	ctx context.Context,
	req *newpb.ConnectToGameRequest,
) (*newpb.ConnectToGameResponse, error) {
	w.logger.Info("收到連接遊戲請求",
		zap.String("roomID", req.RoomId),
		zap.String("playerID", req.PlayerId),
		zap.String("gameID", req.GameId))

	// 獲取遊戲數據
	var gameData *gameflow.GameData
	if req.GameId != "" {
		// 如果提供了遊戲ID，從遊戲管理器獲取該遊戲
		gameData = w.gameManager.GetCurrentGameByRoom(req.RoomId)
	} else if req.RoomId != "" {
		// 否則獲取房間當前的遊戲
		gameData = w.gameManager.GetCurrentGameByRoom(req.RoomId)
	} else {
		// 如果都沒有提供，使用默認房間的遊戲
		gameData = w.gameManager.GetCurrentGame()
	}

	// 創建連接ID
	connectionID := uuid.New().String()

	// 轉換遊戲數據
	var dealerGameData *dealerpb.GameData
	if gameData != nil {
		dealerGameData = convertGameDataToNewPb(gameData)
	}

	// 創建玩家資訊（這裡使用模擬數據，實際應用中應從用戶服務獲取）
	playerInfo := convertPlayerInfoToNewPb(req.PlayerId, "Player_"+req.PlayerId, 1000.0)

	// 構建回應
	resp := &newpb.ConnectToGameResponse{
		ConnectionId: connectionID,
		GameData:     dealerGameData,
		PlayerInfo:   playerInfo,
	}

	return resp, nil
}

// SubscribeToGameEvents 處理訂閱遊戲事件的請求
func (w *PlayerCommunicationServiceWrapper) SubscribeToGameEvents(
	req *newpb.SubscribeToGameEventsRequest,
	stream newpb.PlayerCommunicationService_SubscribeToGameEventsServer,
) error {
	// 創建一個隨機的訂閱者ID
	subscriberID := uuid.New().String()

	w.logger.Info("收到訂閱遊戲事件請求",
		zap.String("roomID", req.RoomId),
		zap.String("playerID", req.PlayerId),
		zap.String("subscriberID", subscriberID))

	// 訂閱事件
	// 這裡你需要實現一個訂閱機制，可能需要與 dealerService 協調
	// 以下僅為示例代碼，實際上你需要根據你的訂閱機制進行調整

	// 每 2 秒發送一個模擬事件
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 獲取當前遊戲數據
			gameData := w.gameManager.GetCurrentGameByRoom(req.RoomId)
			if gameData == nil {
				continue
			}

			// 轉換遊戲數據
			dealerGameData := convertGameDataToNewPb(gameData)

			// 創建一個包含遊戲數據的事件
			event := &newpb.GameEvent{
				Id:        uuid.New().String(),
				Type:      commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
				Timestamp: time.Now().Unix(),
				EventData: &newpb.GameEvent_GameData{
					GameData: dealerGameData,
				},
			}

			// 發送事件
			if err := stream.Send(event); err != nil {
				w.logger.Error("發送遊戲事件失敗",
					zap.String("roomID", req.RoomId),
					zap.String("playerID", req.PlayerId),
					zap.String("subscriberID", subscriberID),
					zap.Error(err))
				return err
			}
		case <-stream.Context().Done():
			// 客戶端斷開連接
			w.logger.Info("客戶端斷開連接",
				zap.String("roomID", req.RoomId),
				zap.String("playerID", req.PlayerId),
				zap.String("subscriberID", subscriberID))
			return nil
		}
	}
}

// GetPlayerStatus 處理獲取玩家狀態的請求
func (w *PlayerCommunicationServiceWrapper) GetPlayerStatus(
	ctx context.Context,
	req *newpb.GetPlayerStatusRequest,
) (*newpb.GetPlayerStatusResponse, error) {
	w.logger.Info("收到獲取玩家狀態請求",
		zap.String("playerID", req.PlayerId))

	// 創建一個示例玩家狀態（在實際應用中，應該從用戶服務或數據庫獲取）
	playerInfo := convertPlayerInfoToNewPb(req.PlayerId, "Player_"+req.PlayerId, 1000.0)
	playerInfo.CardsCount = 5
	playerInfo.Preferences.UiTheme = "dark"

	// 創建響應
	resp := &newpb.GetPlayerStatusResponse{
		PlayerInfo: playerInfo,
	}

	return resp, nil
}

// UpdatePlayerPreference 處理更新玩家偏好設置的請求
func (w *PlayerCommunicationServiceWrapper) UpdatePlayerPreference(
	ctx context.Context,
	req *newpb.UpdatePlayerPreferenceRequest,
) (*newpb.UpdatePlayerPreferenceResponse, error) {
	w.logger.Info("收到更新玩家偏好設置請求",
		zap.String("playerID", req.PlayerId))

	// 在實際應用中，應該將這些偏好設置保存到數據庫
	// 這裡僅返回一個成功響應

	resp := &newpb.UpdatePlayerPreferenceResponse{
		Success: true,
		PlayerInfo: &newpb.PlayerInfo{
			Id:          req.PlayerId,
			Nickname:    "Player_" + req.PlayerId,
			Balance:     1000.0,
			CardsCount:  5,
			Preferences: req.Preferences,
		},
	}

	return resp, nil
}

// convertGameDataToNewPb 將 gameflow.GameData 轉換為新版本的 Proto 結構
func convertGameDataToNewPb(gameData *gameflow.GameData) *dealerpb.GameData {
	if gameData == nil {
		return nil
	}

	// 轉換所有球
	// 使用 RegularBalls 作為 DrawnBalls
	drawnBalls := make([]*dealerpb.Ball, 0, len(gameData.RegularBalls))
	for _, ball := range gameData.RegularBalls {
		drawnBalls = append(drawnBalls, convertBallToNewPb(ball))
	}

	// ExtraBalls需要轉換為map格式
	extraBallsMap := make(map[string]*dealerpb.Ball)
	for i, ball := range gameData.ExtraBalls {
		key := "extra_" + string(rune('a'+i))
		if i == 0 {
			key = "left"
		} else if i == 1 {
			key = "right"
		}
		extraBallsMap[key] = convertBallToNewPb(ball)
	}

	// 處理頭獎球
	var jackpotBall *dealerpb.Ball
	if gameData.Jackpot != nil && len(gameData.Jackpot.DrawnBalls) > 0 {
		jackpotBall = convertBallToNewPb(gameData.Jackpot.DrawnBalls[0])
	}

	// 處理幸運球
	luckyBalls := make([]*dealerpb.Ball, 0)
	if gameData.Jackpot != nil && len(gameData.Jackpot.LuckyBalls) > 0 {
		for _, ball := range gameData.Jackpot.LuckyBalls {
			luckyBalls = append(luckyBalls, convertBallToNewPb(ball))
		}
	}

	// 轉換 game stage
	gameStage := convertGameStageToCommonPb(gameData.CurrentStage)

	// 使用最後更新時間作為更新時間
	updatedAt := time.Now().Unix()
	if !gameData.LastUpdateTime.IsZero() {
		updatedAt = gameData.LastUpdateTime.Unix()
	}

	// 返回新的GameData結構，根據 game.proto 中的 GameData 定義
	return &dealerpb.GameData{
		Id:          gameData.GameID,
		RoomId:      gameData.RoomID,
		Stage:       gameStage,
		Status:      getGameStatusFromStage(gameStage),
		DrawnBalls:  drawnBalls,
		ExtraBalls:  extraBallsMap,
		JackpotBall: jackpotBall,
		LuckyBalls:  luckyBalls,
		CreatedAt:   gameData.StartTime.Unix(),
		UpdatedAt:   updatedAt,
		DealerId:    "system", // 使用默認值
	}
}

// convertBallToNewPb 將 gameflow.Ball 轉換為新版本的 Proto 結構
func convertBallToNewPb(ball gameflow.Ball) *dealerpb.Ball {
	// 生成球ID
	ballID := uuid.New().String()[:8]

	// 獲取球號顏色
	color := "white"
	number := ball.Number
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

	return &dealerpb.Ball{
		Id:      ballID,
		Number:  int32(number),
		Color:   color,
		IsOdd:   number%2 == 1,
		IsSmall: number <= 20,
	}
}

// convertGameStageToCommonPb 將 gameflow.GameStage 轉換為 commonpb.GameStage
func convertGameStageToCommonPb(stage gameflow.GameStage) commonpb.GameStage {
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

// getGameStatusFromStage 根據遊戲階段獲取遊戲狀態
func getGameStatusFromStage(stage commonpb.GameStage) dealerpb.GameStatus {
	// 根據階段判斷狀態
	switch {
	case stage == commonpb.GameStage_GAME_STAGE_UNSPECIFIED:
		return dealerpb.GameStatus_GAME_STATUS_UNSPECIFIED
	case stage == commonpb.GameStage_GAME_STAGE_PREPARATION:
		return dealerpb.GameStatus_GAME_STATUS_NOT_STARTED
	case stage == commonpb.GameStage_GAME_STAGE_GAME_OVER:
		return dealerpb.GameStatus_GAME_STATUS_COMPLETED
	case stage >= commonpb.GameStage_GAME_STAGE_NEW_ROUND && stage < commonpb.GameStage_GAME_STAGE_GAME_OVER:
		return dealerpb.GameStatus_GAME_STATUS_RUNNING
	default:
		return dealerpb.GameStatus_GAME_STATUS_UNSPECIFIED
	}
}

// convertPlayerInfoToNewPb 將玩家資訊轉換為新版本的 Proto 結構
func convertPlayerInfoToNewPb(playerID string, nickname string, balance float64) *newpb.PlayerInfo {
	return &newpb.PlayerInfo{
		Id:       playerID,
		Nickname: nickname,
		Balance:  balance,
		Preferences: &newpb.PlayerPreference{
			ReceiveGameNotifications: true,
			ReceiveChatMessages:      true,
			ShowOtherPlayersBets:     true,
			UiTheme:                  "default",
			Language:                 "zh-TW",
		},
	}
}
