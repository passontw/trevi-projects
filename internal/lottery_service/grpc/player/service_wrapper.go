package player

import (
	"context"
	"time"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
	newpb "g38_lottery_service/internal/generated/api/v1/player"
	commonpb "g38_lottery_service/internal/generated/common"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/internal/lottery_service/grpc/dealer"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PlayerCommunicationServiceWrapper 是新 API 到舊 API 的包裝器
type PlayerCommunicationServiceWrapper struct {
	newpb.UnimplementedPlayerCommunicationServiceServer
	logger        *zap.Logger
	gameManager   *gameflow.GameManager
	dealerService *dealer.DealerService
}

// NewPlayerCommunicationServiceWrapper 創建一個新的 PlayerCommunicationServiceWrapper 實例
func NewPlayerCommunicationServiceWrapper(
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
	dealerService *dealer.DealerService,
) *PlayerCommunicationServiceWrapper {
	return &PlayerCommunicationServiceWrapper{
		logger:        logger.Named("player_communication_wrapper"),
		gameManager:   gameManager,
		dealerService: dealerService,
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
		gameData = w.gameManager.GetGameByID(req.GameId)
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
		dealerGameData = dealer.ConvertGameDataFromOldPb(nil) // 需要實現這個函數
	}

	// 創建玩家資訊（這裡使用模擬數據，實際應用中應從用戶服務獲取）
	playerInfo := &newpb.PlayerInfo{
		Id:         req.PlayerId,
		Nickname:   "Player_" + req.PlayerId,
		Balance:    1000.0,
		CardsCount: 0,
		Preferences: &newpb.PlayerPreference{
			ReceiveGameNotifications: true,
			ReceiveChatMessages:      true,
			ShowOtherPlayersBets:     true,
			UiTheme:                  "default",
			Language:                 "zh-TW",
		},
	}

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
	w.logger.Info("收到訂閱遊戲事件請求",
		zap.String("roomID", req.RoomId),
		zap.String("playerID", req.PlayerId))

	// 檢查是否包含遊戲事件訂閱類型
	hasGameEvents := false
	for _, subType := range req.SubscriptionTypes {
		if subType == newpb.SubscriptionType_SUBSCRIPTION_TYPE_GAME_EVENTS {
			hasGameEvents = true
			break
		}
	}

	if hasGameEvents {
		// 從房間獲取當前遊戲
		gameData := w.gameManager.GetCurrentGameByRoom(req.RoomId)
		if gameData != nil {
			// 發送當前遊戲狀態事件
			event := createGameDataEvent(gameData)
			if err := stream.Send(event); err != nil {
				w.logger.Error("發送初始遊戲數據失敗",
					zap.String("playerID", req.PlayerId),
					zap.Error(err))
				return err
			}
		}
	}

	// 保持連接開啟
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("客戶端連接關閉",
				zap.String("playerID", req.PlayerId))
			return ctx.Err()
		case <-time.After(30 * time.Second):
			// 發送一個心跳事件
			event := &newpb.GameEvent{
				Id:        uuid.New().String(),
				Type:      commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
				Timestamp: time.Now().Unix(),
			}
			if err := stream.Send(event); err != nil {
				w.logger.Error("發送心跳事件失敗",
					zap.String("playerID", req.PlayerId),
					zap.Error(err))
				return err
			}
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

	// 模擬玩家資訊數據
	playerInfo := &newpb.PlayerInfo{
		Id:         req.PlayerId,
		Nickname:   "Player_" + req.PlayerId,
		Balance:    1000.0,
		CardsCount: 0,
		Preferences: &newpb.PlayerPreference{
			ReceiveGameNotifications: true,
			ReceiveChatMessages:      true,
			ShowOtherPlayersBets:     true,
			UiTheme:                  "default",
			Language:                 "zh-TW",
		},
	}

	// 模擬遊戲歷史數據
	gameHistory := make([]*newpb.GameHistoryItem, 0)
	// 這裡可以從數據庫獲取真實的遊戲歷史

	resp := &newpb.GetPlayerStatusResponse{
		PlayerInfo:  playerInfo,
		GameHistory: gameHistory,
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

	// 模擬儲存偏好設置
	// 實際應用中應存入數據庫

	// 模擬玩家資訊數據
	playerInfo := &newpb.PlayerInfo{
		Id:          req.PlayerId,
		Nickname:    "Player_" + req.PlayerId,
		Balance:     1000.0,
		CardsCount:  0,
		Preferences: req.Preferences,
	}

	resp := &newpb.UpdatePlayerPreferenceResponse{
		PlayerInfo:   playerInfo,
		Success:      true,
		ErrorMessage: "",
	}

	return resp, nil
}

// 輔助函數：創建包含遊戲數據的事件
func createGameDataEvent(gameData *gameflow.GameData) *newpb.GameEvent {
	// 轉換遊戲數據
	dealerGameData := dealer.ConvertGameDataFromOldPb(nil) // 需要實現這個函數

	event := &newpb.GameEvent{
		Id:        uuid.New().String(),
		Type:      commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
		Timestamp: time.Now().Unix(),
		EventData: &newpb.GameEvent_GameData{
			GameData: dealerGameData,
		},
	}

	return event
}
