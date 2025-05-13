package dealer

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"
	"g38_lottery_service/internal/lottery_service/gameflow"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DealerServiceAdapter 是將舊的 API 轉換到新的 API 的適配器
type DealerServiceAdapter struct {
	logger      *zap.Logger
	gameManager *gameflow.GameManager
}

// NewDealerServiceAdapter 創建一個新的 dealer 服務適配器
func NewDealerServiceAdapter(
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
) *DealerServiceAdapter {
	return &DealerServiceAdapter{
		logger:      logger.Named("dealer_service_adapter"),
		gameManager: gameManager,
	}
}

// StartNewRound 處理開始新遊戲回合的請求
func (a *DealerServiceAdapter) StartNewRound(ctx context.Context, req *dealerpb.StartNewRoundRequest) (*dealerpb.StartNewRoundResponse, error) {
	// 默認房間ID (因為 dealer 的 StartNewRoundRequest 是空的)
	roomID := "SG01"

	a.logger.Info("收到開始新遊戲回合請求", zap.String("roomID", roomID))

	// 檢查房間是否支持
	supportedRooms := a.gameManager.GetSupportedRooms()
	isSupportedRoom := false
	for _, supported := range supportedRooms {
		if supported == roomID {
			isSupportedRoom = true
			break
		}
	}

	if !isSupportedRoom {
		a.logger.Warn("無法開始新局，不支持的房間ID", zap.String("roomID", roomID))
		return nil, status.Errorf(codes.InvalidArgument, "不支持的房間ID: %s", roomID)
	}

	// 檢查該房間的遊戲是否處於準備階段
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame != nil && currentGame.CurrentStage != gameflow.StagePreparation {
		a.logger.Warn("無法開始新局，當前房間的遊戲不在準備階段",
			zap.String("roomID", roomID),
			zap.String("currentStage", string(currentGame.CurrentStage)))
		return nil, gameflow.ErrInvalidStage
	}

	// 為指定房間創建新遊戲
	gameID, err := a.gameManager.CreateNewGameForRoom(ctx, roomID)
	if err != nil {
		a.logger.Error("創建新遊戲失敗", zap.String("roomID", roomID), zap.Error(err))
		return nil, err
	}

	// 獲取當前遊戲數據
	game := a.gameManager.GetCurrentGameByRoom(roomID)
	if game == nil {
		a.logger.Error("獲取新創建的遊戲失敗", zap.String("roomID", roomID))
		return nil, gameflow.ErrGameNotFound
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:        gameID,
		RoomId:    roomID,
		Stage:     convertGameflowStageToPb(game.CurrentStage),
		Status:    dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 在後台推進階段，不阻塞 RPC 響應
	go func() {
		// 創建新的上下文，因為原始上下文可能會在 RPC 返回後被取消
		newCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 為指定房間推進到新局階段
		if err := a.gameManager.AdvanceStageForRoom(newCtx, roomID, true); err != nil {
			a.logger.Error("推進到新局階段失敗", zap.String("roomID", roomID), zap.Error(err))
			// 無法返回錯誤，只能記錄
		} else {
			a.logger.Info("成功開始新局並推進階段",
				zap.String("roomID", roomID),
				zap.String("gameID", gameID),
				zap.String("stage", string(a.gameManager.GetCurrentStage())))
		}
	}()

	// 構建回應，只包含 gameData，因為 StartNewRoundResponse 只有這一個字段
	resp := &dealerpb.StartNewRoundResponse{
		GameData: gameData,
	}

	a.logger.Info("正在返回 StartNewRound 響應，後台繼續處理階段推進", zap.String("roomID", roomID))
	return resp, nil
}

// DrawBall 處理抽球請求
func (a *DealerServiceAdapter) DrawBall(ctx context.Context, req *dealerpb.DrawBallRequest) (*dealerpb.DrawBallResponse, error) {
	a.logger.Info("收到抽球請求 (新 API)")

	// 模擬抽球
	randomBall := &dealerpb.Ball{
		Id:      generateRandomString(8),
		Number:  int32(rand.Intn(75) + 1), // 1-75 之間的隨機數字
		Color:   "red",
		IsOdd:   rand.Intn(2) == 1,
		IsSmall: rand.Intn(2) == 1,
	}

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", generateRandomString(8)),
		RoomId:    "SG01",
		Stage:     commonpb.GameStage_GAME_STAGE_DRAWING_START,
		Status:    dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{
			randomBall,
		},
	}

	// 構建回應
	newResp := &dealerpb.DrawBallResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// DrawExtraBall 處理抽取額外球的請求
func (a *DealerServiceAdapter) DrawExtraBall(ctx context.Context, req *dealerpb.DrawExtraBallRequest) (*dealerpb.DrawExtraBallResponse, error) {
	a.logger.Info("收到抽取額外球請求 (新 API)")

	// 模擬抽取額外球
	extraBall := &dealerpb.Ball{
		Id:      generateRandomString(8),
		Number:  int32(rand.Intn(75) + 1), // 1-75 之間的隨機數字
		Color:   "blue",
		IsOdd:   rand.Intn(2) == 1,
		IsSmall: rand.Intn(2) == 1,
	}

	// 確定側邊
	side := "left"
	if req.Side == commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT {
		side = "right"
	}

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", generateRandomString(8)),
		RoomId:    "SG01",
		Stage:     commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START,
		Status:    dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		ExtraBalls: map[string]*dealerpb.Ball{
			side: extraBall,
		},
	}

	// 構建回應
	newResp := &dealerpb.DrawExtraBallResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// DrawJackpotBall 處理抽取頭獎球的請求
func (a *DealerServiceAdapter) DrawJackpotBall(ctx context.Context, req *dealerpb.DrawJackpotBallRequest) (*dealerpb.DrawJackpotBallResponse, error) {
	a.logger.Info("收到抽取頭獎球請求 (新 API)")

	// 模擬抽取頭獎球
	jackpotBall := &dealerpb.Ball{
		Id:      generateRandomString(8),
		Number:  int32(rand.Intn(75) + 1), // 1-75 之間的隨機數字
		Color:   "gold",
		IsOdd:   rand.Intn(2) == 1,
		IsSmall: rand.Intn(2) == 1,
	}

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:          fmt.Sprintf("G%s", generateRandomString(8)),
		RoomId:      "SG01",
		Stage:       commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START,
		Status:      dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:    "system",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		JackpotBall: jackpotBall,
	}

	// 構建回應
	newResp := &dealerpb.DrawJackpotBallResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// DrawLuckyBall 處理抽取幸運球的請求
func (a *DealerServiceAdapter) DrawLuckyBall(ctx context.Context, req *dealerpb.DrawLuckyBallRequest) (*dealerpb.DrawLuckyBallResponse, error) {
	a.logger.Info("收到抽取幸運球請求 (新 API)")

	// 決定要抽取多少幸運球
	count := int(req.Count)
	if count <= 0 {
		count = 1 // 默認至少抽取 1 個
	}
	if count > 5 {
		count = 5 // 最多抽取 5 個
	}

	// 模擬抽取幸運球
	luckyBalls := make([]*dealerpb.Ball, 0, count)
	for i := 0; i < count; i++ {
		luckyBall := &dealerpb.Ball{
			Id:      generateRandomString(8),
			Number:  int32(rand.Intn(75) + 1), // 1-75 之間的隨機數字
			Color:   "rainbow",
			IsOdd:   rand.Intn(2) == 1,
			IsSmall: rand.Intn(2) == 1,
		}
		luckyBalls = append(luckyBalls, luckyBall)
	}

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:         fmt.Sprintf("G%s", generateRandomString(8)),
		RoomId:     "SG01",
		Stage:      commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START,
		Status:     dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:   "system",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
		LuckyBalls: luckyBalls,
	}

	// 構建回應
	newResp := &dealerpb.DrawLuckyBallResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// CancelGame 處理取消遊戲的請求
func (a *DealerServiceAdapter) CancelGame(ctx context.Context, req *dealerpb.CancelGameRequest) (*dealerpb.CancelGameResponse, error) {
	a.logger.Info("收到取消遊戲請求 (新 API)")

	// 構建一個基本的 GameData
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", generateRandomString(8)),
		RoomId:    "SG01",
		Status:    dealerpb.GameStatus_GAME_STATUS_CANCELLED,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 構建取消遊戲的回應
	newResp := &dealerpb.CancelGameResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// GetGameStatus 處理獲取遊戲狀態的請求
func (a *DealerServiceAdapter) GetGameStatus(ctx context.Context, req *dealerpb.GetGameStatusRequest) (*dealerpb.GetGameStatusResponse, error) {
	a.logger.Info("收到獲取遊戲狀態請求 (新 API)")

	// 模擬隨機遊戲狀態
	randStage := commonpb.GameStage(rand.Intn(int(commonpb.GameStage_GAME_STAGE_GAME_OVER)))
	randStatus := dealerpb.GameStatus_GAME_STATUS_RUNNING
	if randStage == commonpb.GameStage_GAME_STAGE_PREPARATION {
		randStatus = dealerpb.GameStatus_GAME_STATUS_NOT_STARTED
	} else if randStage == commonpb.GameStage_GAME_STAGE_GAME_OVER {
		randStatus = dealerpb.GameStatus_GAME_STATUS_COMPLETED
	}

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", generateRandomString(8)),
		RoomId:    "SG01",
		Stage:     randStage,
		Status:    randStatus,
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 構建回應
	newResp := &dealerpb.GetGameStatusResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// StartJackpotRound 處理開始頭獎回合的請求
func (a *DealerServiceAdapter) StartJackpotRound(ctx context.Context, req *dealerpb.StartJackpotRoundRequest) (*dealerpb.StartJackpotRoundResponse, error) {
	a.logger.Info("收到開始頭獎回合請求 (新 API)")

	// 保存原始房間ID
	roomId := "SG01"
	if req.RoomId != "" {
		roomId = req.RoomId
	}

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", generateRandomString(8)),
		RoomId:    roomId,
		Stage:     commonpb.GameStage_GAME_STAGE_JACKPOT_START,
		Status:    dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 構建回應
	newResp := &dealerpb.StartJackpotRoundResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// SubscribeGameEvents 處理訂閱遊戲事件的請求
func (a *DealerServiceAdapter) SubscribeGameEvents(req *dealerpb.SubscribeGameEventsRequest, stream dealerpb.DealerService_SubscribeGameEventsServer) error {
	a.logger.Info("收到訂閱遊戲事件請求 (新 API)")

	// 每隔5秒發送一個心跳事件
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 發送初始遊戲狀態事件
	initialEvent := &dealerpb.GameEvent{
		Type:      commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
		Timestamp: time.Now().Unix(),
		EventData: &dealerpb.GameEvent_GameData{
			GameData: &dealerpb.GameData{
				Id:        fmt.Sprintf("G%s", generateRandomString(8)),
				RoomId:    "SG01",
				Stage:     commonpb.GameStage_GAME_STAGE_NEW_ROUND,
				Status:    dealerpb.GameStatus_GAME_STATUS_RUNNING,
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			},
		},
	}

	if err := stream.Send(initialEvent); err != nil {
		a.logger.Error("發送初始事件失敗", zap.Error(err))
		return err
	}

	// 持續發送心跳事件，直到上下文被取消
	for {
		select {
		case <-ticker.C:
			// 創建心跳事件
			heartbeatEvent := &dealerpb.GameEvent{
				Type:      commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
				Timestamp: time.Now().Unix(),
			}

			// 發送心跳事件
			if err := stream.Send(heartbeatEvent); err != nil {
				a.logger.Error("發送心跳事件失敗", zap.Error(err))
				return err
			}

		case <-stream.Context().Done():
			a.logger.Info("客戶端斷開連接或上下文被取消")
			return nil
		}
	}
}

// convertGameflowStageToPb 將 gameflow.GameStage 轉換為 commonpb.GameStage
func convertGameflowStageToPb(stage gameflow.GameStage) commonpb.GameStage {
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

// generateRandomString 生成指定長度的隨機字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
