package dealer

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"
	"g38_lottery_service/internal/lottery_service/gameflow"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DealerServiceAdapter 實現 DealerService 介面的服務適配器
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

// StartNewRound 處理開始新遊戲回合的請求
func (a *DealerServiceAdapter) StartNewRound(ctx context.Context, req *dealerpb.StartNewRoundRequest) (*dealerpb.StartNewRoundResponse, error) {
	a.logger.Info("收到開始新回合的請求 (新 API)")

	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomId := req.RoomId

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", GenerateRandomString(8)),
		RoomId:    roomId,
		Stage:     commonpb.GameStage_GAME_STAGE_JACKPOT_START,
		Status:    dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		// 其他字段將在遊戲進行中更新
	}

	resp := &dealerpb.StartNewRoundResponse{
		GameData: gameData,
	}

	return resp, nil
}

// DrawBall 處理抽取球的請求
func (a *DealerServiceAdapter) DrawBall(ctx context.Context, req *dealerpb.DrawBallRequest) (*dealerpb.DrawBallResponse, error) {
	a.logger.Info("收到抽取球的請求 (新 API)")

	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomId := req.RoomId

	// 隨機生成一個球號碼
	ballNumber := mathrand.Intn(80) + 1

	// 構建球對象
	ball := &dealerpb.Ball{
		Id:      fmt.Sprintf("ball_%d", ballNumber),
		Number:  int32(ballNumber),
		Color:   "red", // 顏色是字符串類型
		IsOdd:   (ballNumber % 2) == 1,
		IsSmall: ballNumber <= 40,
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", GenerateRandomString(8)),
		RoomId:    roomId,
		Stage:     commonpb.GameStage_GAME_STAGE_DRAWING_START,
		Status:    dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{
			ball,
		},
	}

	// 構建回應
	resp := &dealerpb.DrawBallResponse{
		GameData: gameData,
	}

	return resp, nil
}

// DrawExtraBall 實現抽取額外球流程
func (a *DealerServiceAdapter) DrawExtraBall(ctx context.Context, req *dealerpb.DrawExtraBallRequest) (*dealerpb.DrawExtraBallResponse, error) {
	a.logger.Info("收到抽取額外球請求 (新 API)")

	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	// 獲取當前遊戲數據
	roomID := req.RoomId
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		return nil, status.Errorf(codes.Internal, "無法獲取當前遊戲數據")
	}

	// 驗證遊戲階段，確保我們處於可以抽取額外球的階段
	if currentGame.CurrentStage != gameflow.StageExtraBallDrawingStart {
		return nil, status.Errorf(codes.FailedPrecondition, "遊戲階段不正確，當前階段: %v，需要階段: %v",
			currentGame.CurrentStage, gameflow.StageExtraBallDrawingStart)
	}

	// 生成一個隨機球號碼 (1-80)
	ballNumber := mathrand.Intn(80) + 1

	// 創建 gameflow 球對象
	gfBall := gameflow.Ball{
		Number:    ballNumber,
		Type:      gameflow.BallTypeExtra,
		IsLast:    true,
		Timestamp: time.Now(),
	}

	// 添加額外球到遊戲流程
	err := a.gameManager.UpdateExtraBalls(ctx, roomID, gfBall)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "添加額外球失敗: %v", err)
	}

	// 獲取更新後的遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame == nil {
		return nil, status.Errorf(codes.Internal, "無法獲取更新後的遊戲數據")
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:         updatedGame.GameID,
		RoomId:     roomID,
		Stage:      convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:     dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:   "system",
		CreatedAt:  updatedGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
		ExtraBalls: make(map[string]*dealerpb.Ball),
	}

	// 添加所有額外球到響應中
	for i, extraBall := range updatedGame.ExtraBalls {
		ballID := fmt.Sprintf("extra_%d", i+1)
		gameData.ExtraBalls[ballID] = &dealerpb.Ball{
			Id:      ballID,
			Number:  int32(extraBall.Number),
			Color:   "red",
			IsOdd:   extraBall.Number%2 == 1,
			IsSmall: extraBall.Number <= 40,
		}
	}

	// 構建回應
	resp := &dealerpb.DrawExtraBallResponse{
		GameData: gameData,
	}

	return resp, nil
}

// DrawJackpotBall 處理抽取頭獎球的請求
func (a *DealerServiceAdapter) DrawJackpotBall(ctx context.Context, req *dealerpb.DrawJackpotBallRequest) (*dealerpb.DrawJackpotBallResponse, error) {
	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomID := req.RoomId
	a.logger.Info("收到抽取頭獎球請求 (新 API)", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 檢查遊戲階段是否允許抽取頭獎球
	if currentGame.CurrentStage != gameflow.StageJackpotDrawingStart {
		a.logger.Warn("當前遊戲階段不允許抽取頭獎球",
			zap.String("roomID", roomID),
			zap.String("currentStage", string(currentGame.CurrentStage)))
		return nil, fmt.Errorf("當前遊戲階段不允許抽取頭獎球，當前階段: %s", currentGame.CurrentStage)
	}

	// 準備抽取的頭獎球
	randNum := mathrand.Intn(75) + 1 // 1-75 之間的隨機數字

	// 創建內部球對象
	ball := gameflow.Ball{
		Number:    randNum,
		Type:      gameflow.BallTypeJackpot,
		IsLast:    true,
		Timestamp: time.Now(),
	}

	// 將球添加到遊戲流程中
	err := a.gameManager.UpdateJackpotBalls(ctx, roomID, ball)
	if err != nil {
		a.logger.Error("添加頭獎球失敗",
			zap.String("roomID", roomID),
			zap.Int("number", ball.Number),
			zap.Error(err))
		return nil, fmt.Errorf("添加頭獎球失敗: %w", err)
	}

	// 更新後重新獲取遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame == nil {
		updatedGame = currentGame // 如果無法獲取則使用原始數據
	}

	// 確定遊戲狀態
	status := dealerpb.GameStatus_GAME_STATUS_RUNNING

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:        updatedGame.GameID,
		RoomId:    roomID,
		Stage:     convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:    status,
		DealerId:  "system",
		CreatedAt: updatedGame.StartTime.Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 如果存在頭獎球且有Jackpot數據，添加到響應中
	if updatedGame.Jackpot != nil && len(updatedGame.Jackpot.DrawnBalls) > 0 {
		latestBall := updatedGame.Jackpot.DrawnBalls[len(updatedGame.Jackpot.DrawnBalls)-1]

		// 設置頭獎球
		gameData.JackpotBall = &dealerpb.Ball{
			Id:      fmt.Sprintf("jackpot_ball_%d", len(updatedGame.Jackpot.DrawnBalls)),
			Number:  int32(latestBall.Number),
			IsOdd:   latestBall.Number%2 == 1,
			IsSmall: latestBall.Number <= 38,
		}
	}

	// 構建回應
	response := &dealerpb.DrawJackpotBallResponse{
		GameData: gameData,
	}

	return response, nil
}

// DrawLuckyBall 處理抽取幸運球的請求
func (a *DealerServiceAdapter) DrawLuckyBall(ctx context.Context, req *dealerpb.DrawLuckyBallRequest) (*dealerpb.DrawLuckyBallResponse, error) {
	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomID := req.RoomId
	a.logger.Info("收到抽取幸運球請求 (新 API)", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 檢查遊戲階段是否允許抽取幸運球
	if currentGame.CurrentStage != gameflow.StageDrawingLuckyBallsStart {
		a.logger.Warn("當前遊戲階段不允許抽取幸運球",
			zap.String("roomID", roomID),
			zap.String("currentStage", string(currentGame.CurrentStage)))
		return nil, fmt.Errorf("當前遊戲階段不允許抽取幸運球，當前階段: %s", currentGame.CurrentStage)
	}

	// 決定要抽取的幸運球數量
	count := int(req.Count)
	if count <= 0 {
		count = 1 // 默認至少抽取1個
	}

	// 確保不超過7個幸運球的限制
	if count > 7 {
		count = 7
	}

	// 檢查是否已經存在幸運球
	var existingLuckyBallsCount int
	if currentGame.Jackpot != nil {
		existingLuckyBallsCount = len(currentGame.Jackpot.LuckyBalls)
	}

	// 確定還可以抽取多少球
	remainingBallsToAdd := 7 - existingLuckyBallsCount
	if count > remainingBallsToAdd {
		count = remainingBallsToAdd
	}

	// 如果無法再添加球，返回錯誤
	if count <= 0 {
		a.logger.Warn("已達到最大幸運號碼球數量(7球)",
			zap.String("roomID", roomID),
			zap.Int("currentCount", existingLuckyBallsCount))
		return nil, fmt.Errorf("已達到最大幸運號碼球數量(7球)")
	}

	a.logger.Info("準備抽取幸運球",
		zap.String("roomID", roomID),
		zap.Int("existingCount", existingLuckyBallsCount),
		zap.Int("countToAdd", count))

	// 生成並添加幸運球
	var lastBall *gameflow.Ball // 用來記錄最後一個球，以便稍後決定是否推進階段

	for i := 0; i < count; i++ {
		// 生成隨機數 (1-75)
		randNum := mathrand.Intn(75) + 1

		// 檢查最後一個球的標記
		isLast := false
		if existingLuckyBallsCount+i+1 == 7 { // 第7個球是最後一個
			isLast = true
		}

		// 添加球到遊戲
		newBall, err := gameflow.AddBall(currentGame, randNum, gameflow.BallTypeLucky, isLast)
		if err != nil {
			a.logger.Error("添加幸運球失敗",
				zap.String("roomID", roomID),
				zap.Int("number", randNum),
				zap.Error(err))
			return nil, fmt.Errorf("添加幸運球失敗: %w", err)
		}

		// 記錄最後一個球
		if isLast {
			lastBall = newBall
		}

		a.logger.Info("成功添加幸運球",
			zap.String("roomID", roomID),
			zap.Int("ballNumber", newBall.Number))
	}

	// 如果有最後一個球，嘗試推進遊戲階段
	if lastBall != nil && lastBall.IsLast {
		go func() {
			// 使用無超時的 context
			ctx := context.Background()
			// 嘗試推進階段
			err := a.gameManager.AdvanceStageForRoom(ctx, roomID, true)
			if err != nil {
				a.logger.Error("自動推進階段失敗",
					zap.String("roomID", roomID),
					zap.Error(err))
			}
		}()
	}

	// 更新後重新獲取遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame == nil {
		updatedGame = currentGame // 如果無法獲取則使用原始數據
	}

	// 確定遊戲狀態
	status := dealerpb.GameStatus_GAME_STATUS_RUNNING

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:         updatedGame.GameID,
		RoomId:     roomID,
		Stage:      convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:     status,
		DealerId:   "system",
		CreatedAt:  updatedGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		LuckyBalls: make([]*dealerpb.Ball, 0),
	}

	// 取出所有幸運球
	if updatedGame.Jackpot != nil && len(updatedGame.Jackpot.LuckyBalls) > 0 {
		for i, luckyBall := range updatedGame.Jackpot.LuckyBalls {
			ball := &dealerpb.Ball{
				Id:      fmt.Sprintf("lucky_ball_%d", i+1),
				Number:  int32(luckyBall.Number),
				IsOdd:   luckyBall.Number%2 == 1,
				IsSmall: luckyBall.Number <= 38,
			}
			gameData.LuckyBalls = append(gameData.LuckyBalls, ball)
		}
	}

	// 構建回應
	response := &dealerpb.DrawLuckyBallResponse{
		GameData: gameData,
	}

	return response, nil
}

// CancelGame 處理取消遊戲的請求
func (a *DealerServiceAdapter) CancelGame(ctx context.Context, req *dealerpb.CancelGameRequest) (*dealerpb.CancelGameResponse, error) {
	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomID := req.RoomId
	a.logger.Info("收到取消遊戲請求 (新 API)", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return &dealerpb.CancelGameResponse{
			GameData: &dealerpb.GameData{
				Id:        "",
				RoomId:    roomID,
				Stage:     commonpb.GameStage_GAME_STAGE_GAME_OVER,
				Status:    dealerpb.GameStatus_GAME_STATUS_CANCELLED,
				DealerId:  "system",
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			},
		}, nil
	}

	// 更新遊戲狀態，我們不能直接使用 CancelGameForRoom，因為該方法不存在
	// 相反，我們需要手動更新遊戲狀態
	// 先記錄原始遊戲 ID 和信息
	gameID := currentGame.GameID

	// 嘗試重置遊戲狀態（相當於取消當前遊戲並準備新遊戲）
	_, err := a.gameManager.ResetGameForRoom(ctx, roomID)
	if err != nil {
		a.logger.Error("重置遊戲失敗", zap.String("roomID", roomID), zap.Error(err))
		return nil, fmt.Errorf("取消遊戲失敗: %w", err)
	}

	// 構建取消後的遊戲數據
	gameData := &dealerpb.GameData{
		Id:         gameID,
		RoomId:     roomID,
		Stage:      commonpb.GameStage_GAME_STAGE_GAME_OVER,
		Status:     dealerpb.GameStatus_GAME_STATUS_CANCELLED,
		DealerId:   "system",
		CreatedAt:  currentGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
	}

	// 填充已抽取的球
	for i, ball := range currentGame.RegularBalls {
		gameData.DrawnBalls = append(gameData.DrawnBalls, &dealerpb.Ball{
			Id:      fmt.Sprintf("ball_%d", i+1),
			Number:  int32(ball.Number),
			IsOdd:   ball.Number%2 == 1,
			IsSmall: ball.Number <= 38,
		})
	}

	// 填充額外球
	gameData.ExtraBalls = make(map[string]*dealerpb.Ball)
	if len(currentGame.ExtraBalls) > 0 {
		side := "left"
		if currentGame.SelectedSide == gameflow.ExtraBallSideRight {
			side = "right"
		}
		gameData.ExtraBalls[side] = &dealerpb.Ball{
			Id:      "extra_ball",
			Number:  int32(currentGame.ExtraBalls[0].Number),
			IsOdd:   currentGame.ExtraBalls[0].Number%2 == 1,
			IsSmall: currentGame.ExtraBalls[0].Number <= 38,
		}
	}

	// 構建回應
	newResp := &dealerpb.CancelGameResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// GetGameStatus 處理獲取遊戲狀態的請求
func (a *DealerServiceAdapter) GetGameStatus(ctx context.Context, req *dealerpb.GetGameStatusRequest) (*dealerpb.GetGameStatusResponse, error) {
	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomID := req.RoomId
	a.logger.Info("收到獲取遊戲狀態請求 (新 API)", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return &dealerpb.GetGameStatusResponse{
			GameData: &dealerpb.GameData{
				Id:        "",
				RoomId:    roomID,
				Stage:     commonpb.GameStage_GAME_STAGE_PREPARATION,
				Status:    dealerpb.GameStatus_GAME_STATUS_NOT_STARTED,
				DealerId:  "system",
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			},
		}, nil
	}

	// 根據遊戲階段確定狀態
	status := dealerpb.GameStatus_GAME_STATUS_RUNNING
	if currentGame.CurrentStage == gameflow.StagePreparation {
		status = dealerpb.GameStatus_GAME_STATUS_NOT_STARTED
	} else if currentGame.CurrentStage == gameflow.StageGameOver {
		status = dealerpb.GameStatus_GAME_STATUS_COMPLETED
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:         currentGame.GameID,
		RoomId:     roomID,
		Stage:      convertGameflowStageToPb(currentGame.CurrentStage),
		Status:     status,
		DealerId:   "system",
		CreatedAt:  currentGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
	}

	// 填充已抽取的球
	for i, ball := range currentGame.RegularBalls {
		gameData.DrawnBalls = append(gameData.DrawnBalls, &dealerpb.Ball{
			Id:      fmt.Sprintf("ball_%d", i+1),
			Number:  int32(ball.Number),
			IsOdd:   ball.Number%2 == 1,
			IsSmall: ball.Number <= 38,
		})
	}

	// 填充額外球
	gameData.ExtraBalls = make(map[string]*dealerpb.Ball)
	if len(currentGame.ExtraBalls) > 0 {
		side := "left"
		if currentGame.SelectedSide == gameflow.ExtraBallSideRight {
			side = "right"
		}
		gameData.ExtraBalls[side] = &dealerpb.Ball{
			Id:      "extra_ball",
			Number:  int32(currentGame.ExtraBalls[0].Number),
			IsOdd:   currentGame.ExtraBalls[0].Number%2 == 1,
			IsSmall: currentGame.ExtraBalls[0].Number <= 38,
		}
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

	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	// 使用請求中的房間ID
	roomId := req.RoomId

	// 構建基本的遊戲數據
	gameData := &dealerpb.GameData{
		Id:        fmt.Sprintf("G%s", GenerateRandomString(8)),
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
	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomID := req.RoomId
	a.logger.Info("收到訂閱遊戲事件請求 (新 API)", zap.String("roomID", roomID))

	// 生成訂閱者ID
	subscriberID := generateSubscriberID(roomID)

	// 創建事件頻道
	eventChan := make(chan *dealerpb.GameEvent, 100)

	// 註冊訂閱者 - 這裡我們需要增加對訂閱者的管理
	// 由於 DealerServiceAdapter 目前沒有內建訂閱者管理功能，我們將採用一個簡單的方式
	// 在實際項目中，應當將訂閱者管理相關的代碼抽象為一個共用的服務或模塊

	defer func() {
		// 取消訂閱並關閉通道
		close(eventChan)
		a.logger.Info("關閉訂閱者連接",
			zap.String("subscriberID", subscriberID),
			zap.String("roomID", roomID))
	}()

	a.logger.Info("成功註冊訂閱者",
		zap.String("subscriberID", subscriberID),
		zap.String("roomID", roomID))

	// 主動發送當前遊戲狀態
	a.logger.Info("用戶請求獲取當前遊戲狀態", zap.String("roomID", roomID))

	// 獲取當前遊戲狀態並發送
	statusResp, err := a.GetGameStatus(context.Background(), &dealerpb.GetGameStatusRequest{
		RoomId: roomID,
	})

	if err == nil && statusResp != nil && statusResp.GameData != nil {
		gameData := statusResp.GameData

		// 創建事件時間戳
		now := time.Now().Unix()

		// 創建事件並發送
		event := &dealerpb.GameEvent{
			Type:      commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
			Timestamp: now,
			EventData: &dealerpb.GameEvent_GameData{
				GameData: gameData,
			},
		}

		// 嘗試發送
		select {
		case eventChan <- event:
			a.logger.Info("已發送當前遊戲狀態給新訂閱者",
				zap.String("subscriberID", subscriberID),
				zap.String("roomID", roomID),
				zap.String("gameID", gameData.Id))
		default:
			a.logger.Warn("訂閱者通道已滿，無法發送當前遊戲狀態",
				zap.String("subscriberID", subscriberID),
				zap.String("roomID", roomID))
		}
	}

	// 另外開一個 goroutine 定期發送心跳事件
	stopHeartbeat := make(chan bool)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 創建心跳事件
				heartbeatEvent := &dealerpb.GameEvent{
					Type:      commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
					Timestamp: time.Now().Unix(),
				}

				// 發送心跳事件
				select {
				case eventChan <- heartbeatEvent:
					// 成功發送心跳
				default:
					a.logger.Warn("無法發送心跳事件，頻道已滿",
						zap.String("subscriberID", subscriberID))
				}
			case <-stopHeartbeat:
				return
			}
		}
	}()

	// 處理事件流
	for event := range eventChan {
		if err := stream.Send(event); err != nil {
			a.logger.Error("發送事件到客戶端失敗",
				zap.String("subscriberID", subscriberID),
				zap.Error(err))
			close(stopHeartbeat) // 停止心跳
			return err
		}
	}

	close(stopHeartbeat) // 停止心跳
	return nil
}

// 生成訂閱者 ID
func generateSubscriberID(roomID string) string {
	return "subscriber_" + roomID + "_" + GenerateRandomString(8)
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
