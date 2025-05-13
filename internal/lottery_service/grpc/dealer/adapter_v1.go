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
	// 使用硬編碼的房間ID
	roomID := "SG01"

	a.logger.Info("收到抽球請求 (新 API)", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 檢查遊戲階段是否允許抽球
	if currentGame.CurrentStage != gameflow.StageDrawingStart {
		a.logger.Warn("當前遊戲階段不允許抽球",
			zap.String("roomID", roomID),
			zap.String("currentStage", string(currentGame.CurrentStage)))
		return nil, fmt.Errorf("當前遊戲階段不允許抽球，當前階段: %s", currentGame.CurrentStage)
	}

	// 準備要添加的球
	var drawnBalls []*dealerpb.Ball
	var gameflowBalls []gameflow.Ball

	// 由於 dealer 服務的 DrawBallRequest 沒有 Balls 字段，始終使用隨機生成球邏輯
	randNum := rand.Intn(75) + 1 // 1-75 之間的隨機數字

	// 創建內部球對象
	ball := gameflow.Ball{
		Number:    randNum,
		Type:      gameflow.BallTypeRegular,
		IsLast:    true,
		Timestamp: time.Now(),
	}

	// 將球添加到遊戲流程中
	err := a.gameManager.UpdateRegularBalls(ctx, roomID, ball)
	if err != nil {
		a.logger.Error("添加常規球失敗",
			zap.String("roomID", roomID),
			zap.Int("number", ball.Number),
			zap.Error(err))
		return nil, fmt.Errorf("添加常規球失敗: %w", err)
	}

	gameflowBalls = append(gameflowBalls, ball)

	// 創建返回的球對象
	drawnBall := &dealerpb.Ball{
		Id:      fmt.Sprintf("ball_%d", len(currentGame.RegularBalls)+1),
		Number:  int32(randNum),
		IsOdd:   randNum%2 == 1,
		IsSmall: randNum <= 38,
	}
	drawnBalls = append(drawnBalls, drawnBall)

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
		DrawnBalls: drawnBalls,
	}

	// 為之前已經存在的球添加到響應中
	for i, ball := range updatedGame.RegularBalls {
		// 跳過我們剛剛添加的球，以避免重複
		isNewBall := false
		for _, newBall := range gameflowBalls {
			if ball.Number == newBall.Number {
				isNewBall = true
				break
			}
		}

		if !isNewBall {
			gameData.DrawnBalls = append(gameData.DrawnBalls, &dealerpb.Ball{
				Id:      fmt.Sprintf("ball_%d", i+1),
				Number:  int32(ball.Number),
				IsOdd:   ball.Number%2 == 1,
				IsSmall: ball.Number <= 38,
			})
		}
	}

	// 構建回應
	response := &dealerpb.DrawBallResponse{
		GameData: gameData,
	}

	return response, nil
}

// DrawExtraBall 處理抽取額外球的請求
func (a *DealerServiceAdapter) DrawExtraBall(ctx context.Context, req *dealerpb.DrawExtraBallRequest) (*dealerpb.DrawExtraBallResponse, error) {
	// 使用硬編碼的房間ID
	roomID := "SG01"

	a.logger.Info("收到抽取額外球請求 (新 API)", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 檢查遊戲階段是否允許抽取額外球
	if currentGame.CurrentStage != gameflow.StageExtraBallDrawingStart {
		a.logger.Warn("當前遊戲階段不允許抽取額外球",
			zap.String("roomID", roomID),
			zap.String("currentStage", string(currentGame.CurrentStage)))
		return nil, fmt.Errorf("當前遊戲階段不允許抽取額外球，當前階段: %s", currentGame.CurrentStage)
	}

	// 準備抽取的額外球
	randNum := rand.Intn(75) + 1 // 1-75 之間的隨機數字

	// 創建內部球對象
	ball := gameflow.Ball{
		Number:    randNum,
		Type:      gameflow.BallTypeExtra,
		IsLast:    true,
		Timestamp: time.Now(),
	}

	// 將球添加到遊戲流程中
	err := a.gameManager.UpdateExtraBalls(ctx, roomID, ball)
	if err != nil {
		a.logger.Error("添加額外球失敗",
			zap.String("roomID", roomID),
			zap.Int("number", ball.Number),
			zap.Error(err))
		return nil, fmt.Errorf("添加額外球失敗: %w", err)
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
		ExtraBalls: make(map[string]*dealerpb.Ball),
	}

	// 如果存在額外球（應該存在），添加到響應中
	if len(updatedGame.ExtraBalls) > 0 {
		latestBall := updatedGame.ExtraBalls[len(updatedGame.ExtraBalls)-1]

		// 將額外球添加到遊戲數據的 map 中
		ballId := fmt.Sprintf("extra_ball_%d", len(updatedGame.ExtraBalls))
		gameData.ExtraBalls[ballId] = &dealerpb.Ball{
			Id:      ballId,
			Number:  int32(latestBall.Number),
			IsOdd:   latestBall.Number%2 == 1,
			IsSmall: latestBall.Number <= 38,
		}
	}

	// 使用 onBallDrawn 回調而不是 DispatchEvent
	if callback := a.gameManager.GetOnBallDrawnCallback(); callback != nil {
		callback(roomID, ball)
		a.logger.Info("已發送額外球抽取事件通知",
			zap.String("room_id", roomID),
			zap.String("game_id", gameData.Id),
			zap.Any("ball", ball))
	}

	// 構建回應
	response := &dealerpb.DrawExtraBallResponse{
		GameData: gameData,
	}

	return response, nil
}

// DrawJackpotBall 處理抽取頭獎球的請求
func (a *DealerServiceAdapter) DrawJackpotBall(ctx context.Context, req *dealerpb.DrawJackpotBallRequest) (*dealerpb.DrawJackpotBallResponse, error) {
	// 使用硬編碼的房間ID
	roomID := "SG01"

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
	randNum := rand.Intn(75) + 1 // 1-75 之間的隨機數字

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
	// 使用硬編碼的房間ID，因為請求中沒有包含房間ID
	roomID := "SG01"

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
		randNum := rand.Intn(75) + 1

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
	// 使用硬編碼的房間ID
	roomID := "SG01"

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
	// 使用硬編碼的房間ID
	roomID := "SG01"

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
	// 使用硬編碼的房間ID，因為dealer服務不需要在請求中指定房間
	roomID := "SG01"

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
	statusResp, err := a.GetGameStatus(context.Background(), &dealerpb.GetGameStatusRequest{})

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
	return "subscriber_" + roomID + "_" + generateRandomString(8)
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
