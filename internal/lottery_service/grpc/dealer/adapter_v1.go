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
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// 嘗試在遊戲管理器中創建新遊戲
	gameID, err := a.gameManager.CreateNewGameForRoom(ctx, roomId)
	if err != nil {
		a.logger.Error("創建新遊戲失敗",
			zap.String("roomID", roomId),
			zap.Error(err))
	} else {
		a.logger.Info("成功創建新遊戲",
			zap.String("roomID", roomId),
			zap.String("gameID", gameID))
	}

	// 獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomId)

	// 如果無法從遊戲管理器獲取遊戲，則使用模擬數據
	if currentGame == nil {
		// 構建基本的遊戲數據
		gameData := &dealerpb.GameData{
			Id:         fmt.Sprintf("G%s", GenerateRandomString(8)),
			RoomId:     roomId,
			Stage:      commonpb.GameStage_GAME_STAGE_JACKPOT_START,
			Status:     dealerpb.GameStatus_GAME_STATUS_RUNNING,
			DealerId:   "system",
			CreatedAt:  time.Now().Unix(),
			UpdatedAt:  time.Now().Unix(),
			DrawnBalls: []*dealerpb.Ball{},              // 初始化空陣列
			ExtraBalls: make(map[string]*dealerpb.Ball), // 初始化空 map
			LuckyBalls: []*dealerpb.Ball{},              // 初始化空陣列
		}

		resp := &dealerpb.StartNewRoundResponse{
			GameData: gameData,
		}

		return resp, nil
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
		RoomId:     roomId,
		Stage:      convertGameflowStageToPb(currentGame.CurrentStage),
		Status:     status,
		DealerId:   "system",
		CreatedAt:  currentGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
		ExtraBalls: make(map[string]*dealerpb.Ball),
		LuckyBalls: []*dealerpb.Ball{},
	}

	// 構建回應
	resp := &dealerpb.StartNewRoundResponse{
		GameData: gameData,
	}

	return resp, nil
}

// DrawBall 處理抽球請求
func (a *DealerServiceAdapter) DrawBall(ctx context.Context, req *dealerpb.DrawBallRequest) (*dealerpb.DrawBallResponse, error) {
	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomId := req.RoomId
	a.logger.Info("收到抽球請求", zap.String("roomId", roomId))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomId)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomId", roomId))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 驗證遊戲階段是否適合抽取常規球
	if currentGame.CurrentStage != gameflow.StageDrawingStart {
		a.logger.Warn("遊戲階段不允許抽取常規球",
			zap.String("roomId", roomId),
			zap.String("currentStage", string(currentGame.CurrentStage)))
		return nil, fmt.Errorf("遊戲階段不允許抽取常規球，當前階段: %s", currentGame.CurrentStage)
	}

	// 生成隨機球號
	n, err := rand.Int(rand.Reader, big.NewInt(80))
	if err != nil {
		return nil, fmt.Errorf("生成隨機數失敗: %w", err)
	}
	ballNumber := int(n.Int64()) + 1

	// 創建球對象
	gfBall := gameflow.Ball{
		Number:    ballNumber,
		Type:      gameflow.BallTypeRegular,
		IsLast:    false,
		Timestamp: time.Now(),
	}

	// 添加球到遊戲流程中
	err = a.gameManager.UpdateRegularBalls(ctx, roomId, gfBall)
	if err != nil {
		a.logger.Error("添加球失敗", zap.Error(err))
		return nil, fmt.Errorf("添加球失敗: %w", err)
	}

	// 獲取更新後的遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomId)
	if updatedGame == nil {
		a.logger.Error("無法獲取更新後的遊戲數據")
		return nil, fmt.Errorf("無法獲取更新後的遊戲數據")
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:         updatedGame.GameID,
		RoomId:     roomId,
		Stage:      convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:     dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:   "system",
		CreatedAt:  updatedGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
		LuckyBalls: []*dealerpb.Ball{},
	}

	// 添加所有已抽取的球到響應中
	for _, regularBall := range updatedGame.RegularBalls {
		// 創建 timestamp proto
		timestamp := regularBall.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		// 添加球到列表
		gameData.DrawnBalls = append(gameData.DrawnBalls, &dealerpb.Ball{
			Number:    int32(regularBall.Number),
			Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    regularBall.IsLast,
			Timestamp: protoTimestamp,
		})
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
		ballKey := fmt.Sprintf("extra_%d", i+1)

		// 使用當前的時間戳為 proto timestamp
		timestamp := extraBall.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls[ballKey] = &dealerpb.Ball{
			Number:    int32(extraBall.Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    extraBall.IsLast,
			Timestamp: protoTimestamp,
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
		Id:         updatedGame.GameID,
		RoomId:     roomID,
		Stage:      convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:     status,
		DealerId:   "system",
		CreatedAt:  updatedGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
		ExtraBalls: make(map[string]*dealerpb.Ball),
		LuckyBalls: []*dealerpb.Ball{},
	}

	// 如果存在頭獎球且有Jackpot數據，添加到響應中
	if updatedGame.Jackpot != nil && len(updatedGame.Jackpot.DrawnBalls) > 0 {
		latestBall := updatedGame.Jackpot.DrawnBalls[len(updatedGame.Jackpot.DrawnBalls)-1]

		// 創建 timestamp proto
		timestamp := latestBall.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		// 設置頭獎球
		gameData.JackpotBall = &dealerpb.Ball{
			Number:    int32(latestBall.Number),
			Type:      dealerpb.BallType_BALL_TYPE_JACKPOT,
			IsLast:    latestBall.IsLast,
			Timestamp: protoTimestamp,
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
	a.logger.Info("收到幸運號碼球抽取請求", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 確認遊戲階段是否適合抽取幸運號碼球
	if currentGame.CurrentStage != gameflow.StageDrawingLuckyBallsStart {
		return nil, fmt.Errorf("目前遊戲階段不允許抽取幸運號碼球: %s", currentGame.CurrentStage)
	}

	// 生成隨機球號
	n, err := rand.Int(rand.Reader, big.NewInt(75))
	if err != nil {
		a.logger.Error("生成隨機數失敗", zap.Error(err))
		return nil, fmt.Errorf("生成隨機數失敗: %w", err)
	}
	ballNumber := int(n.Int64()) + 1

	// 檢查是否是最後一個幸運號碼球
	var isLast bool
	if currentGame.Jackpot != nil {
		isLast = len(currentGame.Jackpot.LuckyBalls) == 6 // 如果已有6個，則下一個是第7個，也是最後一個
	}

	// 添加球到遊戲
	_, err = gameflow.AddBall(currentGame, ballNumber, gameflow.BallTypeLucky, isLast)
	if err != nil {
		a.logger.Error("添加幸運號碼球失敗", zap.Error(err))
		return nil, fmt.Errorf("添加幸運號碼球失敗: %w", err)
	}

	// 獲取更新後的遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame == nil {
		a.logger.Error("無法獲取更新後的遊戲數據", zap.String("roomID", roomID))
		return nil, fmt.Errorf("無法獲取更新後的遊戲數據")
	}

	// 構建回應數據
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
		LuckyBalls: []*dealerpb.Ball{},
	}

	// 添加所有已抽出的球到響應中
	for _, ball := range updatedGame.RegularBalls {
		// 創建 timestamp proto
		timestamp := ball.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.DrawnBalls = append(gameData.DrawnBalls, &dealerpb.Ball{
			Number:    int32(ball.Number),
			Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    ball.IsLast,
			Timestamp: protoTimestamp,
		})
	}

	// 添加額外球到響應中（如果存在）
	if len(updatedGame.ExtraBalls) > 0 {
		side := "LEFT"
		if updatedGame.SelectedSide == gameflow.ExtraBallSideRight {
			side = "RIGHT"
		}

		// 創建 timestamp proto
		timestamp := updatedGame.ExtraBalls[0].Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls[side] = &dealerpb.Ball{
			Number:    int32(updatedGame.ExtraBalls[0].Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    updatedGame.ExtraBalls[0].IsLast,
			Timestamp: protoTimestamp,
		}
	}

	// 添加幸運號碼球到響應中
	if updatedGame.Jackpot != nil && len(updatedGame.Jackpot.LuckyBalls) > 0 {
		for _, luckyBall := range updatedGame.Jackpot.LuckyBalls {
			// 創建 timestamp proto
			timestamp := luckyBall.Timestamp
			protoTimestamp := &timestamppb.Timestamp{
				Seconds: timestamp.Unix(),
				Nanos:   int32(timestamp.Nanosecond()),
			}

			ball := &dealerpb.Ball{
				Number:    int32(luckyBall.Number),
				Type:      dealerpb.BallType_BALL_TYPE_LUCKY,
				IsLast:    luckyBall.IsLast,
				Timestamp: protoTimestamp,
			}
			gameData.LuckyBalls = append(gameData.LuckyBalls, ball)
		}
	}

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
	a.logger.Info("收到取消遊戲請求", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 取消遊戲
	_, err := a.gameManager.ResetGameForRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("無法取消遊戲: %w", err)
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:         currentGame.GameID,
		RoomId:     roomID,
		Stage:      commonpb.GameStage_GAME_STAGE_GAME_OVER,
		Status:     dealerpb.GameStatus_GAME_STATUS_CANCELLED,
		DealerId:   "system",
		CreatedAt:  currentGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
		ExtraBalls: make(map[string]*dealerpb.Ball),
		LuckyBalls: []*dealerpb.Ball{},
	}

	// 填充已抽取的球
	for _, ball := range currentGame.RegularBalls {
		// 創建 timestamp proto
		timestamp := ball.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.DrawnBalls = append(gameData.DrawnBalls, &dealerpb.Ball{
			Number:    int32(ball.Number),
			Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    ball.IsLast,
			Timestamp: protoTimestamp,
		})
	}

	// 填充額外球
	if len(currentGame.ExtraBalls) > 0 {
		side := "LEFT"
		if currentGame.SelectedSide == gameflow.ExtraBallSideRight {
			side = "RIGHT"
		}

		// 創建 timestamp proto
		timestamp := currentGame.ExtraBalls[0].Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls[side] = &dealerpb.Ball{
			Number:    int32(currentGame.ExtraBalls[0].Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    currentGame.ExtraBalls[0].IsLast,
			Timestamp: protoTimestamp,
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
	a.logger.Info("收到獲取遊戲狀態請求", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame == nil {
		a.logger.Warn("找不到指定房間的遊戲", zap.String("roomID", roomID))
		return nil, fmt.Errorf("找不到指定房間的遊戲")
	}

	// 確定遊戲階段和狀態
	stage := convertGameflowStageToPb(currentGame.CurrentStage)
	status := dealerpb.GameStatus_GAME_STATUS_RUNNING

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:         currentGame.GameID,
		RoomId:     roomID,
		Stage:      stage,
		Status:     status,
		DealerId:   "system",
		CreatedAt:  currentGame.StartTime.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
		ExtraBalls: make(map[string]*dealerpb.Ball),
		LuckyBalls: []*dealerpb.Ball{},
	}

	// 添加已抽出的球到響應中
	if len(currentGame.RegularBalls) > 0 {
		for _, ball := range currentGame.RegularBalls {
			// 創建 timestamp proto
			timestamp := ball.Timestamp
			protoTimestamp := &timestamppb.Timestamp{
				Seconds: timestamp.Unix(),
				Nanos:   int32(timestamp.Nanosecond()),
			}

			gameData.DrawnBalls = append(gameData.DrawnBalls, &dealerpb.Ball{
				Number:    int32(ball.Number),
				Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
				IsLast:    ball.IsLast,
				Timestamp: protoTimestamp,
			})
		}
	}

	// 添加額外球到響應中（如果存在）
	if len(currentGame.ExtraBalls) > 0 {
		side := "LEFT"
		if currentGame.SelectedSide == gameflow.ExtraBallSideRight {
			side = "RIGHT"
		}

		// 設置額外球
		for _, ball := range currentGame.ExtraBalls {
			// 創建 timestamp proto
			timestamp := ball.Timestamp
			protoTimestamp := &timestamppb.Timestamp{
				Seconds: timestamp.Unix(),
				Nanos:   int32(timestamp.Nanosecond()),
			}

			gameData.ExtraBalls[side] = &dealerpb.Ball{
				Number:    int32(ball.Number),
				Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
				IsLast:    ball.IsLast,
				Timestamp: protoTimestamp,
			}
			break // 目前僅支持一個額外球
		}
	}

	// 添加頭獎球到響應中（如果存在）
	if currentGame.Jackpot != nil && len(currentGame.Jackpot.DrawnBalls) > 0 {
		latestBall := currentGame.Jackpot.DrawnBalls[len(currentGame.Jackpot.DrawnBalls)-1]

		// 創建 timestamp proto
		timestamp := latestBall.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.JackpotBall = &dealerpb.Ball{
			Number:    int32(latestBall.Number),
			Type:      dealerpb.BallType_BALL_TYPE_JACKPOT,
			IsLast:    latestBall.IsLast,
			Timestamp: protoTimestamp,
		}
	}

	// 添加幸運球到響應中（如果存在）
	if currentGame.Jackpot != nil && len(currentGame.Jackpot.LuckyBalls) > 0 {
		for _, luckyBall := range currentGame.Jackpot.LuckyBalls {
			// 創建 timestamp proto
			timestamp := luckyBall.Timestamp
			protoTimestamp := &timestamppb.Timestamp{
				Seconds: timestamp.Unix(),
				Nanos:   int32(timestamp.Nanosecond()),
			}

			ball := &dealerpb.Ball{
				Number:    int32(luckyBall.Number),
				Type:      dealerpb.BallType_BALL_TYPE_LUCKY,
				IsLast:    luckyBall.IsLast,
				Timestamp: protoTimestamp,
			}
			gameData.LuckyBalls = append(gameData.LuckyBalls, ball)
		}
	}

	// 構建響應
	response := &dealerpb.GetGameStatusResponse{
		GameData: gameData,
	}

	return response, nil
}

// StartJackpotRound 處理開始頭獎回合的請求
func (a *DealerServiceAdapter) StartJackpotRound(ctx context.Context, req *dealerpb.StartJackpotRoundRequest) (*dealerpb.StartJackpotRoundResponse, error) {
	// 檢查 room_id 是否為空
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id 不能為空")
	}

	roomID := req.RoomId
	a.logger.Info("收到開始頭獎回合請求", zap.String("roomID", roomID))

	// 從遊戲管理器獲取當前遊戲數據
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	var currentStage string = string(gameflow.StageJackpotDrawingStart)
	var createdAt time.Time = time.Now()

	if currentGame != nil {
		currentStage = string(currentGame.CurrentStage)
		createdAt = currentGame.StartTime
	}

	// 準備前進到頭獎回合
	err := a.gameManager.AdvanceStageForRoom(ctx, roomID, true)
	if err != nil {
		a.logger.Error("無法前進到頭獎回合",
			zap.String("roomID", roomID),
			zap.String("currentStage", currentStage),
			zap.Error(err))
		return nil, fmt.Errorf("無法前進到頭獎回合: %w", err)
	}

	a.logger.Info("成功前進到頭獎回合",
		zap.String("roomID", roomID),
		zap.String("previousStage", currentStage))

	// 再次獲取遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame != nil {
		currentStage = string(updatedGame.CurrentStage)
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		Id:         "jackpot_" + roomID,
		RoomId:     roomID,
		Stage:      commonpb.GameStage_GAME_STAGE_JACKPOT_START,
		Status:     dealerpb.GameStatus_GAME_STATUS_RUNNING,
		DealerId:   "system",
		CreatedAt:  createdAt.Unix(),
		UpdatedAt:  time.Now().Unix(),
		DrawnBalls: []*dealerpb.Ball{},
		ExtraBalls: make(map[string]*dealerpb.Ball),
		LuckyBalls: []*dealerpb.Ball{},
	}

	return &dealerpb.StartJackpotRoundResponse{
		GameData: gameData,
	}, nil
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
