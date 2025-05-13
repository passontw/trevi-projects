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

// StartNewRound 處理開始新一輪遊戲的請求
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
			GameId:       fmt.Sprintf("G%s", GenerateRandomString(8)),
			RoomId:       roomId,
			Stage:        commonpb.GameStage_GAME_STAGE_JACKPOT_START,
			Status:       dealerpb.GameStatus_GAME_STATUS_CREATED,
			DealerId:     "system",
			CreatedAt:    time.Now().Unix(),
			UpdatedAt:    time.Now().Unix(),
			RegularBalls: []*dealerpb.Ball{},
			ExtraBalls:   []*dealerpb.Ball{},
			LuckyBalls:   []*dealerpb.Ball{},
		}

		resp := &dealerpb.StartNewRoundResponse{
			GameData: gameData,
		}

		return resp, nil
	}

	// 根據遊戲階段確定狀態
	status := dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS
	if currentGame.CurrentStage == gameflow.StagePreparation {
		status = dealerpb.GameStatus_GAME_STATUS_CREATED
	} else if currentGame.CurrentStage == gameflow.StageGameOver {
		status = dealerpb.GameStatus_GAME_STATUS_COMPLETED
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		GameId:       currentGame.GameID,
		RoomId:       roomId,
		Stage:        a.convertGameflowStageToPb(currentGame.CurrentStage),
		Status:       status,
		DealerId:     "system",
		CreatedAt:    currentGame.StartTime.Unix(),
		UpdatedAt:    time.Now().Unix(),
		RegularBalls: []*dealerpb.Ball{},
		ExtraBalls:   []*dealerpb.Ball{},
		LuckyBalls:   []*dealerpb.Ball{},
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
	roomId := req.GetRoomId()
	if roomId == "" {
		a.logger.Warn("room_id 不能為空")
		return nil, fmt.Errorf("room_id 不能為空")
	}

	a.logger.Info("處理抽球請求", zap.String("roomId", roomId))

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

	// 處理請求中的球
	var isLastBallPresent bool
	var balls []gameflow.Ball

	// 如果請求中包含球數據
	if len(req.Balls) > 0 {
		a.logger.Info("使用請求中的預定義球陣列", zap.Int("球數量", len(req.Balls)))

		// 檢查請求中是否有重複球號
		numberSet := make(map[int]bool)

		// 轉換所有請求中的球
		for i, reqBall := range req.Balls {
			ballNumber := int(reqBall.Number)

			// 檢查球號是否有效 (1-80)
			if ballNumber < 1 || ballNumber > 80 {
				return nil, fmt.Errorf("無效的球號: %d, 球號必須在1到80之間", ballNumber)
			}

			// 檢查請求中是否有重複
			if numberSet[ballNumber] {
				return nil, fmt.Errorf("請求中包含重複的球號: %d", ballNumber)
			}
			numberSet[ballNumber] = true

			// 處理 isLast 標誌 (只有最後一個球可能是最後一個)
			isLast := reqBall.IsLast
			if isLast {
				isLastBallPresent = true
			}

			// 創建球對象
			ball := gameflow.Ball{
				Number:    ballNumber,
				Type:      gameflow.BallTypeRegular,
				IsLast:    isLast,
				Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond), // 確保時間戳不同
			}

			balls = append(balls, ball)
			a.logger.Info("處理預定義球",
				zap.Int("索引", i),
				zap.Int("號碼", ballNumber),
				zap.Bool("isLast", isLast))
		}

		// 批量替換常規球 (而不是一個一個添加)
		err := a.replaceBallsForRoom(ctx, roomId, balls)
		if err != nil {
			a.logger.Error("替換球失敗", zap.Error(err))
			return nil, fmt.Errorf("替換球失敗: %w", err)
		}
	} else {
		// 沒有預定義球，生成隨機球號
		n, err := rand.Int(rand.Reader, big.NewInt(80))
		if err != nil {
			return nil, fmt.Errorf("生成隨機數失敗: %w", err)
		}
		ballNumber := int(n.Int64()) + 1

		// 創建球對象
		ball := gameflow.Ball{
			Number:    ballNumber,
			Type:      gameflow.BallTypeRegular,
			IsLast:    false,
			Timestamp: time.Now(),
		}

		// 添加單個球到遊戲流程中
		err = a.gameManager.UpdateRegularBalls(ctx, roomId, ball)
		if err != nil {
			a.logger.Error("添加球失敗", zap.Error(err))
			return nil, fmt.Errorf("添加球失敗: %w", err)
		}

		a.logger.Info("生成隨機球號", zap.Int("number", ballNumber))
	}

	// 如果有最後一個球，自動推進到下一個階段
	if isLastBallPresent {
		a.logger.Info("收到最後一個球，自動推進到下一個階段",
			zap.String("roomId", roomId))

		go func() {
			// 創建新的上下文以避免使用已取消的上下文
			newCtx := context.Background()
			err := a.gameManager.AdvanceStageForRoom(newCtx, roomId, true)
			if err != nil {
				a.logger.Error("自動推進到下一個階段失敗",
					zap.String("roomId", roomId),
					zap.Error(err))
			} else {
				a.logger.Info("成功推進到下一個階段",
					zap.String("roomId", roomId))
			}
		}()
	}

	// 獲取更新後的遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomId)
	if updatedGame == nil {
		a.logger.Error("無法獲取更新後的遊戲數據")
		return nil, fmt.Errorf("無法獲取更新後的遊戲數據")
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		GameId:       updatedGame.GameID,
		RoomId:       roomId,
		Stage:        a.convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:       dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS,
		DealerId:     "system",
		CreatedAt:    updatedGame.StartTime.Unix(),
		UpdatedAt:    time.Now().Unix(),
		RegularBalls: []*dealerpb.Ball{},
		ExtraBalls:   []*dealerpb.Ball{},
		LuckyBalls:   []*dealerpb.Ball{},
	}

	// 添加所有已抽取的球到響應中
	for i, regularBall := range updatedGame.RegularBalls {
		// 添加球到列表
		gameData.RegularBalls = append(gameData.RegularBalls, &dealerpb.Ball{
			Number: int32(regularBall.Number),
			Type:   dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast: regularBall.IsLast,
			Timestamp: &timestamppb.Timestamp{
				Seconds: regularBall.Timestamp.Unix(),
				Nanos:   int32(regularBall.Timestamp.Nanosecond()),
			},
		})

		// 在日誌中輸出前10個球
		if i < 10 {
			a.logger.Debug("添加常規球", zap.Int("number", regularBall.Number))
		}
	}

	// 構建回應
	resp := &dealerpb.DrawBallResponse{
		GameData: gameData,
	}

	return resp, nil
}

// 新添加的方法 - 批量替換房間的球
func (a *DealerServiceAdapter) replaceBallsForRoom(ctx context.Context, roomID string, balls []gameflow.Ball) error {
	a.logger.Info("批量替換球", zap.String("roomID", roomID), zap.Int("球數量", len(balls)))

	// 直接使用 GameManager 的 ReplaceBalls 方法
	err := a.gameManager.ReplaceBalls(ctx, roomID, balls)
	if err != nil {
		a.logger.Error("替換球失敗", zap.Error(err))
		return fmt.Errorf("替換球失敗: %w", err)
	}

	return nil
}

// CustomDrawBall 處理自定義球請求（用於兼容客戶端發送的球陣列格式）
func (a *DealerServiceAdapter) CustomDrawBall(ctx context.Context, reqData map[string]interface{}) (*dealerpb.DrawBallResponse, error) {
	a.logger.Info("收到自定義抽球請求")

	// 檢查 roomId 是否存在
	roomId, ok := reqData["roomId"].(string)
	if !ok || roomId == "" {
		return nil, status.Error(codes.InvalidArgument, "roomId 不能為空")
	}

	a.logger.Info("處理自定義抽球請求", zap.String("roomId", roomId))

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

	// 處理請求中的球
	var isLastBallPresent bool
	var balls []gameflow.Ball

	// 檢查請求中是否包含球陣列
	ballsData, hasBalls := reqData["balls"].([]interface{})
	if hasBalls && len(ballsData) > 0 {
		a.logger.Info("使用請求中的預定義球陣列", zap.Int("球數量", len(ballsData)))

		// 檢查請求中是否有重複球號
		numberSet := make(map[int]bool)

		// 處理所有請求中的球
		for i, ballDataInterface := range ballsData {
			ballData, ok := ballDataInterface.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("球數據格式無效，第 %d 個球", i)
			}

			// 嘗試獲取球號
			numFloat, ok := ballData["number"].(float64)
			if !ok {
				return nil, fmt.Errorf("球號缺失或無效，第 %d 個球", i)
			}

			ballNumber := int(numFloat)

			// 檢查球號是否有效 (1-80)
			if ballNumber < 1 || ballNumber > 80 {
				return nil, fmt.Errorf("無效的球號: %d, 球號必須在1到80之間", ballNumber)
			}

			// 檢查請求中是否有重複
			if numberSet[ballNumber] {
				return nil, fmt.Errorf("請求中包含重複的球號: %d", ballNumber)
			}
			numberSet[ballNumber] = true

			// 嘗試獲取 isLast 標誌
			isLast, _ := ballData["isLast"].(bool)
			if isLast {
				isLastBallPresent = true
			}

			// 創建球對象
			ball := gameflow.Ball{
				Number:    ballNumber,
				Type:      gameflow.BallTypeRegular,
				IsLast:    isLast,
				Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond), // 確保時間戳不同
			}

			balls = append(balls, ball)
			a.logger.Info("處理預定義球",
				zap.Int("索引", i),
				zap.Int("號碼", ballNumber),
				zap.Bool("isLast", isLast))
		}

		// 批量替換常規球 (而不是一個一個添加)
		err := a.replaceBallsForRoom(ctx, roomId, balls)
		if err != nil {
			a.logger.Error("替換球失敗", zap.Error(err))
			return nil, fmt.Errorf("替換球失敗: %w", err)
		}
	} else {
		// 沒有有效的預定義球，生成隨機球號
		n, err := rand.Int(rand.Reader, big.NewInt(80))
		if err != nil {
			return nil, fmt.Errorf("生成隨機數失敗: %w", err)
		}
		ballNumber := int(n.Int64()) + 1
		a.logger.Info("生成隨機球號", zap.Int("number", ballNumber))

		// 創建球對象
		ball := gameflow.Ball{
			Number:    ballNumber,
			Type:      gameflow.BallTypeRegular,
			IsLast:    false,
			Timestamp: time.Now(),
		}

		// 添加單個球到遊戲流程中
		err = a.gameManager.UpdateRegularBalls(ctx, roomId, ball)
		if err != nil {
			a.logger.Error("添加球失敗", zap.Error(err))
			return nil, fmt.Errorf("添加球失敗: %w", err)
		}
	}

	// 如果有最後一個球，自動推進到下一個階段
	if isLastBallPresent {
		a.logger.Info("收到最後一個球，自動推進到下一個階段",
			zap.String("roomId", roomId))

		go func() {
			// 創建新的上下文以避免使用已取消的上下文
			newCtx := context.Background()
			err := a.gameManager.AdvanceStageForRoom(newCtx, roomId, true)
			if err != nil {
				a.logger.Error("自動推進到下一個階段失敗",
					zap.String("roomId", roomId),
					zap.Error(err))
			} else {
				a.logger.Info("成功推進到下一個階段",
					zap.String("roomId", roomId))
			}
		}()
	}

	// 獲取更新後的遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomId)
	if updatedGame == nil {
		a.logger.Error("無法獲取更新後的遊戲數據")
		return nil, fmt.Errorf("無法獲取更新後的遊戲數據")
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		GameId:       updatedGame.GameID,
		RoomId:       roomId,
		Stage:        a.convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:       dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS,
		DealerId:     "system",
		CreatedAt:    updatedGame.StartTime.Unix(),
		UpdatedAt:    time.Now().Unix(),
		RegularBalls: []*dealerpb.Ball{},
		ExtraBalls:   []*dealerpb.Ball{},
		LuckyBalls:   []*dealerpb.Ball{},
	}

	// 添加所有已抽取的球到響應中
	for i, regularBall := range updatedGame.RegularBalls {
		// 添加球到列表
		gameData.RegularBalls = append(gameData.RegularBalls, &dealerpb.Ball{
			Number: int32(regularBall.Number),
			Type:   dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast: regularBall.IsLast,
			Timestamp: &timestamppb.Timestamp{
				Seconds: regularBall.Timestamp.Unix(),
				Nanos:   int32(regularBall.Timestamp.Nanosecond()),
			},
		})

		// 在日誌中輸出前10個球
		if i < 10 {
			a.logger.Debug("添加常規球", zap.Int("number", regularBall.Number))
		}
	}

	// 構建回應
	resp := &dealerpb.DrawBallResponse{
		GameData: gameData,
	}

	return resp, nil
}

// DrawExtraBall 處理抽取額外球的請求
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

	// 檢查請求中是否包含預定義的球
	var balls []gameflow.Ball
	if len(req.Balls) > 0 {
		a.logger.Info("檢測到請求中包含預定義的球",
			zap.Int("球數量", len(req.Balls)),
			zap.String("roomID", roomID))

		// 檢查預定義球數量是否超過允許的額外球數量
		if len(req.Balls) > int(currentGame.ExtraBallCount) {
			return nil, status.Errorf(codes.InvalidArgument,
				"預定義球數量 %d 超過了允許的額外球數量 %d",
				len(req.Balls), currentGame.ExtraBallCount)
		}

		// 轉換請求中的球為 gameflow.Ball
		for i, reqBall := range req.Balls {
			ballNumber := int(reqBall.Number)

			// 檢查球號是否有效
			if err := gameflow.ValidateBallNumber(ballNumber); err != nil {
				return nil, status.Errorf(codes.InvalidArgument,
					"無效的球號 %d: %v", ballNumber, err)
			}

			// 檢查是否與已抽出的常規球或已有的額外球重複
			if gameflow.IsBallDuplicate(ballNumber, currentGame.RegularBalls) {
				return nil, status.Errorf(codes.InvalidArgument,
					"球號 %d 與已抽出的常規球重複", ballNumber)
			}

			if gameflow.IsBallDuplicate(ballNumber, currentGame.ExtraBalls) {
				return nil, status.Errorf(codes.InvalidArgument,
					"球號 %d 與已抽出的額外球重複", ballNumber)
			}

			// 創建 Ball 對象
			ball := gameflow.Ball{
				Number:    ballNumber,
				Type:      gameflow.BallTypeExtra,
				IsLast:    i == len(req.Balls)-1,                               // 最後一個球標記為 IsLast
				Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond), // 確保時間戳不同
			}

			balls = append(balls, ball)
		}

		// 處理預定義的球
		for _, ball := range balls {
			err := a.gameManager.UpdateExtraBalls(ctx, roomID, ball)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "添加預定義額外球失敗: %v", err)
			}
		}

		// 如果預定義球數量等於總共允許的額外球數量，則表示額外球已抽完
		if len(req.Balls) == int(currentGame.ExtraBallCount) {
			a.logger.Info("已抽出所有額外球，自動進入下一階段",
				zap.String("roomID", roomID),
				zap.Int("extraBallCount", int(currentGame.ExtraBallCount)))

			// 在後台執行，避免阻塞當前請求
			go func() {
				// 創建新的上下文以避免使用已取消的上下文
				newCtx := context.Background()
				if err := a.gameManager.AdvanceStageForRoom(newCtx, roomID, true); err != nil {
					a.logger.Error("自動進入下一階段失敗",
						zap.String("roomID", roomID),
						zap.Error(err))
				}
			}()
		}
	} else {
		// 如果請求中不包含預定義球，則生成一個隨機球號碼 (1-80)
		a.logger.Info("請求中不包含預定義球，將生成隨機球號", zap.String("roomID", roomID))

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
	}

	// 獲取更新後的遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame == nil {
		return nil, status.Errorf(codes.Internal, "無法獲取更新後的遊戲數據")
	}

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		GameId:         updatedGame.GameID,
		RoomId:         roomID,
		Stage:          a.convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:         dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS,
		DealerId:       "system",
		CreatedAt:      updatedGame.StartTime.Unix(),
		UpdatedAt:      time.Now().Unix(),
		RegularBalls:   []*dealerpb.Ball{},
		ExtraBalls:     []*dealerpb.Ball{},
		LuckyBalls:     []*dealerpb.Ball{},
		ExtraBallCount: int32(updatedGame.ExtraBallCount),
	}

	// 添加所有已抽取的額外球到響應中
	for _, extraBall := range updatedGame.ExtraBalls {
		// 創建 timestamp proto
		timestamp := extraBall.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls = append(gameData.ExtraBalls, &dealerpb.Ball{
			Number:    int32(extraBall.Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    extraBall.IsLast,
			Timestamp: protoTimestamp,
		})
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

	// 確保 Jackpot 存在
	if currentGame.Jackpot == nil {
		currentGame.Jackpot = &gameflow.JackpotGame{
			ID:         fmt.Sprintf("jackpot_%s", time.Now().Format("20060102150405")),
			StartTime:  time.Now(),
			DrawnBalls: make([]gameflow.Ball, 0),
			LuckyBalls: make([]gameflow.Ball, 0),
			Active:     true,
			Amount:     500000, // 默認JP金額
		}
	}

	// 檢查是否使用客戶端預定義的球
	if req.Balls != nil && len(req.Balls) > 0 {
		// 檢查預定義球是否有重複球號
		numberSet := make(map[int]bool)
		for _, ball := range req.Balls {
			num := int(ball.Number)
			if numberSet[num] {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("預定義球中包含重複球號: %d", num))
			}
			numberSet[num] = true
		}

		// 使用預定義球替換 DrawnBalls
		newBalls := make([]gameflow.Ball, 0, len(req.Balls))
		for i, predefinedBall := range req.Balls {
			// 創建 Ball 對象
			ball := gameflow.Ball{
				Number:    int(predefinedBall.Number),
				Type:      gameflow.BallTypeJackpot,
				IsLast:    predefinedBall.IsLast,
				Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond), // 確保時間戳不同
			}
			newBalls = append(newBalls, ball)
		}

		// 替換 DrawnBalls 陣列
		currentGame.Jackpot.DrawnBalls = newBalls
		currentGame.LastUpdateTime = time.Now()

		a.logger.Info("使用預定義的球覆蓋頭獎球數據",
			zap.String("roomID", roomID),
			zap.Int("ballCount", len(newBalls)))

		// 保存遊戲狀態
		if err := a.gameManager.GetRepository().SaveGame(ctx, currentGame); err != nil {
			a.logger.Error("保存遊戲狀態失敗",
				zap.String("roomID", roomID),
				zap.Error(err))
			return nil, fmt.Errorf("保存遊戲狀態失敗: %w", err)
		}
	} else {
		// 使用隨機生成的球
		randNum := mathrand.Intn(75) + 1 // 1-75 之間的隨機數字
		isLast := true                   // 單個頭獎球通常是最後一個

		// 創建內部球對象
		ball := gameflow.Ball{
			Number:    randNum,
			Type:      gameflow.BallTypeJackpot,
			IsLast:    isLast,
			Timestamp: time.Now(),
		}

		a.logger.Info("生成隨機頭獎球數據",
			zap.String("roomID", roomID),
			zap.Int("number", randNum))

		// 直接使用隨機球替換 DrawnBalls
		currentGame.Jackpot.DrawnBalls = []gameflow.Ball{ball}
		currentGame.LastUpdateTime = time.Now()

		// 保存遊戲狀態
		if err := a.gameManager.GetRepository().SaveGame(ctx, currentGame); err != nil {
			a.logger.Error("保存遊戲狀態失敗",
				zap.String("roomID", roomID),
				zap.Error(err))
			return nil, fmt.Errorf("保存遊戲狀態失敗: %w", err)
		}
	}

	// 檢查是否有最後一個球，自動推進到下一階段
	hasLastBall := false
	for _, ball := range currentGame.Jackpot.DrawnBalls {
		if ball.IsLast {
			hasLastBall = true
			break
		}
	}

	if hasLastBall {
		a.logger.Info("檢測到最後一顆頭獎球，自動推進到下一階段",
			zap.String("roomID", roomID))

		go func() {
			// 創建新的上下文以避免使用已取消的上下文
			newCtx := context.Background()
			err := a.gameManager.AdvanceStageForRoom(newCtx, roomID, true)
			if err != nil {
				a.logger.Error("自動推進到下一階段失敗",
					zap.String("roomID", roomID),
					zap.Error(err))
			} else {
				a.logger.Info("成功推進到下一階段",
					zap.String("roomID", roomID))
			}
		}()
	}

	// 更新後重新獲取遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame == nil {
		updatedGame = currentGame // 如果無法獲取則使用原始數據
	}

	// 確定遊戲狀態
	status := dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		GameId:       updatedGame.GameID,
		RoomId:       roomID,
		Stage:        a.convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:       status,
		DealerId:     "system",
		CreatedAt:    updatedGame.StartTime.Unix(),
		UpdatedAt:    time.Now().Unix(),
		RegularBalls: []*dealerpb.Ball{},
		ExtraBalls:   []*dealerpb.Ball{},
		LuckyBalls:   []*dealerpb.Ball{},
	}

	// 填充額外球
	if len(currentGame.ExtraBalls) > 0 {
		// 創建 timestamp proto
		timestamp := currentGame.ExtraBalls[0].Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls = append(gameData.ExtraBalls, &dealerpb.Ball{
			Number:    int32(currentGame.ExtraBalls[0].Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    currentGame.ExtraBalls[0].IsLast,
			Timestamp: protoTimestamp,
		})
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

	// 初始化 Jackpot（如果不存在）
	if currentGame.Jackpot == nil {
		currentGame.Jackpot = &gameflow.JackpotGame{
			ID:         fmt.Sprintf("jackpot_%s", time.Now().Format("20060102150405")),
			StartTime:  time.Now(),
			DrawnBalls: make([]gameflow.Ball, 0),
			LuckyBalls: make([]gameflow.Ball, 0),
			Active:     true,
			Amount:     500000, // 默認JP金額
		}
	}

	// 處理預定義球或生成隨機球
	if req.Balls != nil && len(req.Balls) > 0 {
		a.logger.Info("檢測到請求中包含預定義的幸運球",
			zap.Int("球數量", len(req.Balls)),
			zap.String("roomID", roomID))

		// 檢查預定義球是否有重複球號
		numberSet := make(map[int]bool)
		for _, ball := range req.Balls {
			num := int(ball.Number)
			if numberSet[num] {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("預定義球中包含重複球號: %d", num))
			}
			numberSet[num] = true
		}

		// 檢查是否超過最大球數
		maxLuckyBalls := 7
		if len(currentGame.Jackpot.LuckyBalls)+len(req.Balls) > maxLuckyBalls {
			return nil, status.Errorf(codes.InvalidArgument,
				"幸運球總數將超過上限 %d 個（當前 %d 個，新增 %d 個）",
				maxLuckyBalls, len(currentGame.Jackpot.LuckyBalls), len(req.Balls))
		}

		// 處理所有預定義幸運球
		for i, predefinedBall := range req.Balls {
			// 確定是否為最後一個球
			isLast := predefinedBall.IsLast

			// 創建球對象
			ball := gameflow.Ball{
				Number:    int(predefinedBall.Number),
				Type:      gameflow.BallTypeLucky,
				IsLast:    isLast,
				Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond), // 確保時間戳不同
			}

			// 添加球到遊戲
			_, err := gameflow.AddBall(currentGame, ball.Number, ball.Type, ball.IsLast)
			if err != nil {
				a.logger.Error("添加預定義幸運球失敗",
					zap.Int("number", ball.Number),
					zap.Error(err))
				return nil, fmt.Errorf("添加預定義幸運球失敗: %w", err)
			}
		}

		// 檢查是否有 isLast 為 true 的球，如果有，自動進入下一階段
		hasLastBall := false
		for _, reqBall := range req.Balls {
			if reqBall.IsLast {
				hasLastBall = true
				break
			}
		}

		if hasLastBall {
			a.logger.Info("檢測到最後一個幸運球，自動推進到下一階段",
				zap.String("roomID", roomID))

			go func() {
				// 創建新的上下文以避免使用已取消的上下文
				newCtx := context.Background()
				err := a.gameManager.AdvanceStageForRoom(newCtx, roomID, true)
				if err != nil {
					a.logger.Error("自動推進到下一階段失敗",
						zap.String("roomID", roomID),
						zap.Error(err))
				} else {
					a.logger.Info("成功推進到下一階段",
						zap.String("roomID", roomID))
				}
			}()
		}
	} else {
		// 生成隨機球號
		n, err := rand.Int(rand.Reader, big.NewInt(75))
		if err != nil {
			a.logger.Error("生成隨機數失敗", zap.Error(err))
			return nil, fmt.Errorf("生成隨機數失敗: %w", err)
		}
		ballNumber := int(n.Int64()) + 1

		// 檢查是否是最後一個幸運號碼球
		isLast := false
		if currentGame.Jackpot != nil {
			isLast = len(currentGame.Jackpot.LuckyBalls) == 6 // 如果已有6個，則下一個是第7個，也是最後一個
		}

		// 添加球到遊戲
		_, err = gameflow.AddBall(currentGame, ballNumber, gameflow.BallTypeLucky, isLast)
		if err != nil {
			a.logger.Error("添加幸運號碼球失敗", zap.Error(err))
			return nil, fmt.Errorf("添加幸運號碼球失敗: %w", err)
		}
	}

	// 獲取更新後的遊戲數據
	updatedGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if updatedGame == nil {
		a.logger.Error("無法獲取更新後的遊戲數據", zap.String("roomID", roomID))
		return nil, fmt.Errorf("無法獲取更新後的遊戲數據")
	}

	// 構建回應數據
	gameData := &dealerpb.GameData{
		GameId:       updatedGame.GameID,
		RoomId:       roomID,
		Stage:        a.convertGameflowStageToPb(updatedGame.CurrentStage),
		Status:       dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS,
		DealerId:     "system",
		CreatedAt:    updatedGame.StartTime.Unix(),
		UpdatedAt:    time.Now().Unix(),
		RegularBalls: []*dealerpb.Ball{},
		ExtraBalls:   []*dealerpb.Ball{},
		LuckyBalls:   []*dealerpb.Ball{},
	}

	// 填充已抽取的球
	for _, ball := range updatedGame.RegularBalls {
		// 創建 timestamp proto
		timestamp := ball.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.RegularBalls = append(gameData.RegularBalls, &dealerpb.Ball{
			Number:    int32(ball.Number),
			Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    ball.IsLast,
			Timestamp: protoTimestamp,
		})
	}

	// 填充額外球
	if len(updatedGame.ExtraBalls) > 0 {
		// 創建 timestamp proto
		timestamp := updatedGame.ExtraBalls[0].Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls = append(gameData.ExtraBalls, &dealerpb.Ball{
			Number:    int32(updatedGame.ExtraBalls[0].Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    updatedGame.ExtraBalls[0].IsLast,
			Timestamp: protoTimestamp,
		})
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

			gameData.LuckyBalls = append(gameData.LuckyBalls, &dealerpb.Ball{
				Number:    int32(luckyBall.Number),
				Type:      dealerpb.BallType_BALL_TYPE_LUCKY,
				IsLast:    luckyBall.IsLast,
				Timestamp: protoTimestamp,
			})
		}
	}

	response := &dealerpb.DrawLuckyBallResponse{
		GameData: gameData,
	}

	return response, nil
}

// CancelGame 取消當前遊戲
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
		GameId:       currentGame.GameID,
		RoomId:       roomID,
		Stage:        a.convertGameflowStageToPb(currentGame.CurrentStage),
		Status:       dealerpb.GameStatus_GAME_STATUS_CANCELED,
		DealerId:     "system",
		CreatedAt:    currentGame.StartTime.Unix(),
		UpdatedAt:    time.Now().Unix(),
		RegularBalls: []*dealerpb.Ball{},
		ExtraBalls:   []*dealerpb.Ball{},
		LuckyBalls:   []*dealerpb.Ball{},
	}

	// 填充已抽取的球
	for _, ball := range currentGame.RegularBalls {
		// 創建 timestamp proto
		timestamp := ball.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.RegularBalls = append(gameData.RegularBalls, &dealerpb.Ball{
			Number:    int32(ball.Number),
			Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    ball.IsLast,
			Timestamp: protoTimestamp,
		})
	}

	// 填充額外球
	if len(currentGame.ExtraBalls) > 0 {
		// 創建 timestamp proto
		timestamp := currentGame.ExtraBalls[0].Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls = append(gameData.ExtraBalls, &dealerpb.Ball{
			Number:    int32(currentGame.ExtraBalls[0].Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    currentGame.ExtraBalls[0].IsLast,
			Timestamp: protoTimestamp,
		})
	}

	// 構建回應
	newResp := &dealerpb.CancelGameResponse{
		GameData: gameData,
	}

	return newResp, nil
}

// GetGameStatus 實現獲取遊戲狀態
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
		a.logger.Warn("無法找到指定房間的遊戲",
			zap.String("roomID", roomID))
		return nil, status.Error(codes.NotFound, "找不到指定房間的遊戲")
	}

	// 轉換 GameStage 和 GameStatus
	stage := a.convertGameflowStageToPb(currentGame.CurrentStage)
	status := dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS

	// 構建遊戲數據
	gameData := &dealerpb.GameData{
		GameId:         currentGame.GameID,
		RoomId:         roomID,
		Stage:          stage,
		Status:         status,
		DealerId:       "system",
		CreatedAt:      currentGame.StartTime.Unix(),
		UpdatedAt:      time.Now().Unix(),
		RegularBalls:   make([]*dealerpb.Ball, 0, len(currentGame.RegularBalls)),
		ExtraBalls:     make([]*dealerpb.Ball, 0, len(currentGame.ExtraBalls)),
		LuckyBalls:     make([]*dealerpb.Ball, 0, 7),
		ExtraBallCount: int32(currentGame.ExtraBallCount),
	}

	// 添加已抽出的常規球到響應中
	a.logger.Debug("添加已抽出的常規球到響應中",
		zap.String("roomID", roomID),
		zap.Int("ballCount", len(currentGame.RegularBalls)))

	for i, ball := range currentGame.RegularBalls {
		// 創建 timestamp proto
		timestamp := ball.Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		// 設置 IsLast 標誌，只有最後一個球被標記為最後一個
		isLast := i == len(currentGame.RegularBalls)-1

		gameData.RegularBalls = append(gameData.RegularBalls, &dealerpb.Ball{
			Number:    int32(ball.Number),
			Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    isLast,
			Timestamp: protoTimestamp,
		})
	}

	// 填充額外球
	if len(currentGame.ExtraBalls) > 0 {
		// 創建 timestamp proto
		timestamp := currentGame.ExtraBalls[0].Timestamp
		protoTimestamp := &timestamppb.Timestamp{
			Seconds: timestamp.Unix(),
			Nanos:   int32(timestamp.Nanosecond()),
		}

		gameData.ExtraBalls = append(gameData.ExtraBalls, &dealerpb.Ball{
			Number:    int32(currentGame.ExtraBalls[0].Number),
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    currentGame.ExtraBalls[0].IsLast,
			Timestamp: protoTimestamp,
		})
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

	// 可選：添加幸運號碼球（如果有）
	if currentGame.Jackpot != nil && len(currentGame.Jackpot.LuckyBalls) > 0 {
		for _, luckyBall := range currentGame.Jackpot.LuckyBalls {
			// 創建 timestamp proto
			timestamp := luckyBall.Timestamp
			protoTimestamp := &timestamppb.Timestamp{
				Seconds: timestamp.Unix(),
				Nanos:   int32(timestamp.Nanosecond()),
			}

			gameData.LuckyBalls = append(gameData.LuckyBalls, &dealerpb.Ball{
				Number:    int32(luckyBall.Number),
				Type:      dealerpb.BallType_BALL_TYPE_LUCKY,
				IsLast:    luckyBall.IsLast,
				Timestamp: protoTimestamp,
			})
		}
	}

	// 構建最終的響應
	resp := &dealerpb.GetGameStatusResponse{
		GameData:       gameData,
		ExtraBallCount: int32(currentGame.ExtraBallCount),
	}

	a.logger.Debug("返回遊戲狀態響應",
		zap.String("roomID", roomID),
		zap.String("gameID", gameData.GameId),
		zap.String("stage", gameData.Stage.String()),
		zap.Int("drawnBallsCount", len(gameData.RegularBalls)),
		zap.Int("extraBallsCount", len(gameData.ExtraBalls)),
		zap.Int("luckyBallsCount", len(gameData.LuckyBalls)))

	return resp, nil
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
		GameId:       "jackpot_" + roomID,
		RoomId:       roomID,
		Stage:        commonpb.GameStage_GAME_STAGE_JACKPOT_START,
		Status:       dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS,
		DealerId:     "system",
		CreatedAt:    createdAt.Unix(),
		UpdatedAt:    time.Now().Unix(),
		RegularBalls: []*dealerpb.Ball{},
		ExtraBalls:   []*dealerpb.Ball{},
		LuckyBalls:   []*dealerpb.Ball{},
	}

	return &dealerpb.StartJackpotRoundResponse{
		GameData: gameData,
	}, nil
}

// SubscribeGameEvents 處理訂閱遊戲事件的請求
func (a *DealerServiceAdapter) SubscribeGameEvents(req *dealerpb.SubscribeGameEventsRequest, stream dealerpb.DealerService_SubscribeGameEventsServer) error {
	// 檢查 room_id 是否為空
	roomID := req.RoomId
	if roomID == "" {
		a.logger.Warn("訂閱請求的 room_id 為空")
		return fmt.Errorf("room_id 不能為空")
	}

	a.logger.Info("處理遊戲事件訂閱請求",
		zap.String("roomID", roomID))

	// 生成訂閱者 ID
	subscriberID := a.generateSubscriberID(roomID)
	a.logger.Debug("為訂閱者生成 ID",
		zap.String("subscriberID", subscriberID),
		zap.String("roomID", roomID))

	// 創建事件通道
	eventChan := make(chan *dealerpb.GameEvent, 10)
	defer close(eventChan)

	// 一開始就獲取遊戲狀態並發送給訂閱者
	currentGame := a.gameManager.GetCurrentGameByRoom(roomID)
	if currentGame != nil {
		a.logger.Info("訂閱時發送當前遊戲狀態",
			zap.String("roomID", roomID),
			zap.String("gameID", currentGame.GameID),
			zap.String("stage", string(currentGame.CurrentStage)))

		// 構建遊戲數據
		gameData := &dealerpb.GameData{
			GameId:       currentGame.GameID,
			RoomId:       roomID,
			Stage:        a.convertGameflowStageToPb(currentGame.CurrentStage),
			Status:       dealerpb.GameStatus_GAME_STATUS_IN_PROGRESS,
			DealerId:     "system",
			CreatedAt:    currentGame.StartTime.Unix(),
			UpdatedAt:    time.Now().Unix(),
			RegularBalls: make([]*dealerpb.Ball, 0, len(currentGame.RegularBalls)),
			ExtraBalls:   make([]*dealerpb.Ball, 0, len(currentGame.ExtraBalls)),
		}

		// 添加所有已抽取的常規球
		for i, ball := range currentGame.RegularBalls {
			isLast := (i == len(currentGame.RegularBalls)-1)

			// 創建 timestamp proto
			timestamp := ball.Timestamp
			protoTimestamp := &timestamppb.Timestamp{
				Seconds: timestamp.Unix(),
				Nanos:   int32(timestamp.Nanosecond()),
			}

			ballPb := &dealerpb.Ball{
				Number:    int32(ball.Number),
				Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
				IsLast:    isLast,
				Timestamp: protoTimestamp,
			}

			gameData.RegularBalls = append(gameData.RegularBalls, ballPb)
		}
		a.logger.Debug("添加常規球到遊戲資料",
			zap.Int("球數量", len(gameData.RegularBalls)))
	}

	return nil
}

func (a *DealerServiceAdapter) generateSubscriberID(roomID string) string {
	return fmt.Sprintf("%s_%s", roomID, GenerateRandomString(8))
}

func (a *DealerServiceAdapter) getLuckyBallsCapacity(game *gameflow.GameData) int {
	return 7 // 默認幸運球數量為7
}

func (a *DealerServiceAdapter) convertGameflowStageToPb(stage gameflow.GameStage) commonpb.GameStage {
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
