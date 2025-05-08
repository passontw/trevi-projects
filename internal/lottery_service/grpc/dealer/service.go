package dealer

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"g38_lottery_service/internal/lottery_service/dealerWebsocket"
	"g38_lottery_service/internal/lottery_service/gameflow"
	pb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DealerService 實現 gRPC 中定義的 DealerService 接口
type DealerService struct {
	pb.UnimplementedDealerServiceServer
	logger         *zap.Logger
	gameManager    *gameflow.GameManager
	dealerServer   *dealerWebsocket.DealerServer
	subscribers    map[string]chan *pb.GameEvent // 訂閱者映射表
	subscribersMux sync.RWMutex                  // 訂閱者鎖
}

// NewDealerService 創建新的 DealerService 實例
func NewDealerService(
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
	dealerServer *dealerWebsocket.DealerServer,
) *DealerService {
	// 創建服務實例
	service := &DealerService{
		logger:       logger.With(zap.String("component", "dealer_service")),
		gameManager:  gameManager,
		dealerServer: dealerServer,
		subscribers:  make(map[string]chan *pb.GameEvent),
	}

	// 註冊事件處理函數
	service.registerEventHandlers()

	return service
}

// 註冊事件處理函數
func (s *DealerService) registerEventHandlers() {
	s.logger.Info("正在註冊事件處理函數")

	// 註冊階段變更事件處理函數
	s.gameManager.SetEventHandlers(
		s.onStageChanged,          // 階段變更
		s.onGameCreated,           // 遊戲創建
		s.onGameCancelled,         // 遊戲取消
		s.onGameCompleted,         // 遊戲完成
		s.onBallDrawn,             // 球抽取
		s.onExtraBallSideSelected, // 額外球選邊
	)

	s.logger.Info("事件處理函數註冊完成")
}

// StartNewRound 實現 DealerService.StartNewRound RPC 方法
func (s *DealerService) StartNewRound(ctx context.Context, req *pb.StartNewRoundRequest) (*pb.StartNewRoundResponse, error) {
	s.logger.Info("收到開始新局請求")

	// 檢查當前階段是否為準備階段
	currentStage := s.gameManager.GetCurrentStage()
	if currentStage != gameflow.StagePreparation {
		s.logger.Warn("無法開始新局，當前階段不是準備階段",
			zap.String("currentStage", string(currentStage)))
		return nil, gameflow.ErrInvalidStage
	}
	// 創建新遊戲
	gameID, err := s.gameManager.CreateNewGame(ctx)
	if err != nil {
		s.logger.Error("創建新遊戲失敗", zap.Error(err))
		return nil, err
	}
	// 獲取當前遊戲數據
	game := s.gameManager.GetCurrentGame()
	if game == nil {
		s.logger.Error("獲取新創建的遊戲失敗")
		return nil, gameflow.ErrGameNotFound
	}

	// 構建響應
	response := &pb.StartNewRoundResponse{
		GameId:       gameID,
		StartTime:    timestamppb.New(game.StartTime),
		CurrentStage: convertGameStageToPb(game.CurrentStage),
	}

	// 創建一個副本作為響應返回
	responseCopy := *response

	// 在後台推進階段，不阻塞 RPC 響應
	go func() {
		// 創建新的上下文，因為原始上下文可能會在 RPC 返回後被取消
		newCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 推進到新局階段
		if err := s.gameManager.AdvanceStage(newCtx, true); err != nil {
			s.logger.Error("推進到新局階段失敗", zap.Error(err))
			// 無法返回錯誤，只能記錄
		} else {
			s.logger.Info("成功開始新局並推進階段",
				zap.String("gameID", gameID),
				zap.String("stage", string(s.gameManager.GetCurrentStage())))

			// 通過 WebSocket 廣播新遊戲開始事件
			// 已經移至 goroutine 中，不會阻塞
			s.broadcastNewGameEvent(gameID, s.gameManager.GetCurrentGame())
		}
	}()

	s.logger.Info("正在返回 StartNewRound 響應，後台繼續處理階段推進")
	return &responseCopy, nil
}

// DrawBall 實現 DealerService.DrawBall RPC 方法
func (s *DealerService) DrawBall(ctx context.Context, req *pb.DrawBallRequest) (*pb.DrawBallResponse, error) {
	// 獲取當前遊戲
	game := s.gameManager.GetCurrentGame()
	if game == nil {
		return nil, gameflow.ErrGameNotFound
	}

	// 確認當前階段允許抽取常規球
	if game.CurrentStage != gameflow.StageDrawingStart {
		return nil, gameflow.NewGameFlowErrorWithFormat("INVALID_STAGE_FOR_DRAW",
			"當前階段 %s 不允許抽取常規球", game.CurrentStage)
	}

	// 從請求中獲取球列表
	reqBalls := req.GetBalls()

	// 將 proto balls 轉換為 gameflow balls
	gameBalls := make([]gameflow.Ball, 0, len(reqBalls))

	// 檢查最後一顆球是否標記為最後一顆
	var isLastBall bool
	if len(reqBalls) > 0 {
		lastBall := reqBalls[len(reqBalls)-1]
		isLastBall = lastBall.IsLast
	}

	// 轉換所有球
	for i, ball := range reqBalls {
		gameBall := gameflow.Ball{
			Number:    int(ball.Number),
			Type:      gameflow.BallTypeRegular,
			IsLast:    ball.IsLast,
			Timestamp: ball.Timestamp.AsTime(),
		}
		gameBalls = append(gameBalls, gameBall)

		// 如果是新添加的球，觸發球抽取事件
		if i >= len(game.RegularBalls) {
			if s.gameManager.GetOnBallDrawnCallback() != nil {
				s.gameManager.GetOnBallDrawnCallback()(game.GameID, gameBall)
			}
		}
	}

	// 獲取管理器內部的遊戲，並直接覆蓋 RegularBalls 陣列
	if err := s.gameManager.UpdateRegularBalls(ctx, gameBalls); err != nil {
		return nil, err
	}

	// 如果最後一顆球標記為最後一顆，同步執行階段推進
	var gameStatus *pb.GameStatus
	if isLastBall {
		// 同步執行階段推進，確保 Redis 中的狀態立即更新
		if err := s.gameManager.AdvanceStage(ctx, true); err != nil {
			s.logger.Error("最後一顆常規球處理後推進階段失敗", zap.Error(err))
		} else {
			// 階段推進成功，更新遊戲狀態
			gameStatus = &pb.GameStatus{
				Stage:   pb.GameStage_GAME_STAGE_DRAWING_CLOSE,
				Message: "最後一顆球已標記，抽球環節結束，進入下一階段",
			}
		}
	}

	// 獲取更新後的遊戲數據
	updatedGame := s.gameManager.GetCurrentGame()

	// 將 gameflow balls 轉換回 proto balls
	updatedBalls := make([]*pb.Ball, 0, len(updatedGame.RegularBalls))
	for _, ball := range updatedGame.RegularBalls {
		pbBall := &pb.Ball{
			Number:    int32(ball.Number),
			Type:      pb.BallType_BALL_TYPE_REGULAR,
			IsLast:    ball.IsLast,
			Timestamp: timestamppb.New(ball.Timestamp),
		}
		updatedBalls = append(updatedBalls, pbBall)
	}

	response := &pb.DrawBallResponse{
		Balls:      updatedBalls,
		GameStatus: gameStatus,
	}

	return response, nil
}

// DrawExtraBall 實現 DealerService.DrawExtraBall RPC 方法
func (s *DealerService) DrawExtraBall(ctx context.Context, req *pb.DrawExtraBallRequest) (*pb.DrawExtraBallResponse, error) {
	// 檢查是否有已存在的球
	existingBalls := req.GetBalls()

	// 將 proto 球轉換為 gameflow 球
	gameBalls := make([]gameflow.Ball, 0, len(existingBalls))
	for _, pbBall := range existingBalls {
		gameBalls = append(gameBalls, gameflow.Ball{
			Number:    int(pbBall.Number),
			Type:      gameflow.BallTypeExtra,
			IsLast:    pbBall.IsLast,
			Timestamp: pbBall.Timestamp.AsTime(),
		})
	}

	// 處理額外球請求 - 使用新的 HandleDrawExtraBall 函數，接受整個球陣列
	err := s.gameManager.HandleDrawExtraBall(ctx, gameBalls)
	if err != nil {
		return nil, err
	}

	// 獲取當前遊戲狀態以獲取更新後的球陣列
	gameData := s.gameManager.GetCurrentGame()
	if gameData == nil {
		return nil, gameflow.ErrGameNotFound
	}

	// 將更新後的額外球轉換為 proto 球
	updatedBalls := make([]*pb.Ball, len(gameData.ExtraBalls))
	for i, ball := range gameData.ExtraBalls {
		updatedBalls[i] = convertBallToPb(ball)
	}

	return &pb.DrawExtraBallResponse{
		Balls: updatedBalls,
	}, nil
}

// DrawJackpotBall 實現 DealerService.DrawJackpotBall RPC 方法
func (s *DealerService) DrawJackpotBall(ctx context.Context, req *pb.DrawJackpotBallRequest) (*pb.DrawJackpotBallResponse, error) {
	// 檢查是否有已存在的球
	existingBalls := req.GetBalls()

	// 獲取最後一個球的數字（如果存在）
	var lastBallNumber int
	var isLast bool

	if len(existingBalls) > 0 {
		lastBall := existingBalls[len(existingBalls)-1]
		lastBallNumber = int(lastBall.Number)
		isLast = lastBall.IsLast
	}

	// 抽取JP球
	ball, err := s.gameManager.HandleDrawJackpotBall(ctx, lastBallNumber, isLast)
	if err != nil {
		return nil, err
	}

	// 創建新的球對象
	pbBall := &pb.Ball{
		Number:    int32(ball.Number),
		Type:      pb.BallType_BALL_TYPE_JACKPOT,
		IsLast:    ball.IsLast,
		Timestamp: timestamppb.New(ball.Timestamp),
	}

	// 將新球添加到現有球列表
	updatedBalls := append(existingBalls, pbBall)

	return &pb.DrawJackpotBallResponse{
		Balls: updatedBalls,
	}, nil
}

// DrawLuckyBall 實現 DealerService.DrawLuckyBall RPC 方法
func (s *DealerService) DrawLuckyBall(ctx context.Context, req *pb.DrawLuckyBallRequest) (*pb.DrawLuckyBallResponse, error) {
	// 檢查是否有已存在的球
	existingBalls := req.GetBalls()

	// 獲取當前遊戲狀態來檢查實際幸運球數量
	gameData := s.gameManager.GetCurrentGame()
	if gameData == nil || gameData.Jackpot == nil {
		s.logger.Error("獲取遊戲狀態失敗或Jackpot為空")
		return nil, fmt.Errorf("獲取遊戲狀態失敗")
	}

	// 檢查實際LuckyBalls數量，如果是6個但用戶沒有提供新球，自動添加第7個
	currentLuckyBallsCount := len(gameData.Jackpot.LuckyBalls)
	s.logger.Info("當前幸運號碼球數量",
		zap.Int("luckyBalls數量", currentLuckyBallsCount),
		zap.Int("請求中的球數量", len(existingBalls)))

	// 如果當前有6個幸運球但請求中沒有任何球，我們需要添加第7個球
	if currentLuckyBallsCount == 6 && len(existingBalls) == 0 {
		s.logger.Info("檢測到已有6個幸運號碼球，自動添加第7個球以完成抽取")

		// 生成隨機球號（1-75）
		// Go 1.22 中不再需要顯式設置隨機種子，math/rand 現在自動處理
		randomBallNumber := rand.Intn(75) + 1

		// 創建新的幸運球
		newBall := &pb.Ball{
			Number:    int32(randomBallNumber),
			Type:      pb.BallType_BALL_TYPE_LUCKY,
			IsLast:    true,
			Timestamp: timestamppb.New(time.Now()),
		}

		// 添加到請求中
		existingBalls = append(existingBalls, newBall)
		s.logger.Info("已自動添加第7個幸運號碼球",
			zap.Int("球號", randomBallNumber))
	}

	// 檢查是否已達7球上限
	if len(existingBalls) > 7 {
		s.logger.Warn("幸運號碼球數量超過7個，將截取前7個",
			zap.Int("當前球數", len(existingBalls)))
		existingBalls = existingBalls[:7]
	} else if len(existingBalls) == 7 {
		s.logger.Info("已達到幸運號碼球的上限數量(7球)",
			zap.Int("當前球數", len(existingBalls)))
	}

	// 將 protobuf 的球轉換為內部類型的球
	luckyBalls := make([]gameflow.Ball, len(existingBalls))
	for i, pbBall := range existingBalls {
		luckyBalls[i] = gameflow.Ball{
			Number:    int(pbBall.Number),
			Type:      gameflow.BallTypeLucky,
			IsLast:    pbBall.IsLast,
			Timestamp: pbBall.Timestamp.AsTime(),
		}
	}

	// 確保最後一個球被標記為最後一個
	if len(luckyBalls) > 0 {
		// 先將所有球設為非最後一球
		for i := range luckyBalls {
			luckyBalls[i].IsLast = false
		}
		// 設置最後一個球為最後一個
		luckyBalls[len(luckyBalls)-1].IsLast = true
	}

	s.logger.Info("調用 HandleDrawLuckyBall",
		zap.Int("球數量", len(luckyBalls)))

	// 調用新的處理方法，直接使用整個球陣列
	err := s.gameManager.HandleDrawLuckyBall(ctx, luckyBalls)
	if err != nil {
		s.logger.Error("處理幸運號碼球失敗", zap.Error(err))
		return nil, err
	}

	// 獲取更新後的遊戲狀態
	gameData = s.gameManager.GetCurrentGame()
	if gameData == nil || gameData.Jackpot == nil {
		s.logger.Error("獲取更新後的遊戲狀態失敗或Jackpot為空")
		return nil, fmt.Errorf("獲取更新後的遊戲狀態失敗")
	}

	// 轉換所有幸運號碼球為 proto 類型
	updatedBalls := make([]*pb.Ball, 0, len(gameData.Jackpot.LuckyBalls))
	for _, ball := range gameData.Jackpot.LuckyBalls {
		updatedBalls = append(updatedBalls, convertBallToPb(ball))
	}

	// 確保只返回最多7顆球
	if len(updatedBalls) > 7 {
		updatedBalls = updatedBalls[:7]
		s.logger.Warn("修正返回的幸運號碼球數量，限制為7球",
			zap.Int("原球數", len(gameData.Jackpot.LuckyBalls)),
			zap.Int("修正後", len(updatedBalls)))
	}

	s.logger.Info("已處理幸運號碼球",
		zap.Int("當前幸運球總數", len(updatedBalls)))

	return &pb.DrawLuckyBallResponse{
		Balls: updatedBalls,
	}, nil
}

// CancelGame 實現 DealerService.CancelGame RPC 方法
func (s *DealerService) CancelGame(ctx context.Context, req *pb.CancelGameRequest) (*pb.GameData, error) {
	// 取消遊戲
	err := s.gameManager.CancelGame(ctx, req.Reason)
	if err != nil {
		return nil, err
	}

	// 返回更新後的遊戲數據
	return convertGameDataToPb(s.gameManager.GetCurrentGame()), nil
}

// AdvanceStage 實現 DealerService.AdvanceStage RPC 方法
// NOTE: 此方法在最新的proto定義中不存在，暫時註釋掉
/*
func (s *DealerService) AdvanceStage(ctx context.Context, req *pb.AdvanceStageRequest) (*pb.AdvanceStageResponse, error) {
	// 獲取當前階段
	oldStage := s.gameManager.GetCurrentStage()

	// 推進遊戲階段
	err := s.gameManager.AdvanceStage(ctx, req.Force)
	if err != nil {
		return nil, err
	}

	// 獲取新階段
	newStage := s.gameManager.GetCurrentStage()

	return &pb.AdvanceStageResponse{
		OldStage: convertGameStageToPb(oldStage),
		NewStage: convertGameStageToPb(newStage),
	}, nil
}
*/

// GetGameStatus 實現 DealerService.GetGameStatus RPC 方法
func (s *DealerService) GetGameStatus(ctx context.Context, req *pb.GetGameStatusRequest) (*pb.GetGameStatusResponse, error) {
	s.logger.Info("收到 GetGameStatus 請求")

	// 獲取當前遊戲狀態
	gameData := s.gameManager.GetCurrentGame()
	if gameData == nil {
		return nil, gameflow.ErrGameNotFound
	}

	// 將內部 GameData 轉換為 proto GameData
	pbGameData := convertGameDataToPb(gameData)

	// 返回 gRPC 響應
	return &pb.GetGameStatusResponse{
		GameData: pbGameData,
	}, nil
}

// sendNotificationToSubscribers 發送通知給所有訂閱者
func (s *DealerService) sendNotificationToSubscribers(notification *pb.GameEvent) {
	s.subscribersMux.RLock()
	defer s.subscribersMux.RUnlock()

	// 遍歷所有訂閱者並發送通知
	for subID, subChan := range s.subscribers {
		// 使用非阻塞方式發送，避免一個訂閱者阻塞其他訂閱者
		select {
		case subChan <- notification:
			s.logger.Debug("已發送通知事件到訂閱者",
				zap.String("subscriberID", subID),
				zap.String("gameID", notification.GameId),
				zap.String("eventType", notification.EventType.String()))
		default:
			s.logger.Warn("訂閱者通道已滿，無法發送通知",
				zap.String("subscriberID", subID),
				zap.String("gameID", notification.GameId))
		}
	}
}

// 添加和移除訂閱者
func (s *DealerService) addSubscriber(subscriberID string, channel chan *pb.GameEvent) {
	s.subscribersMux.Lock()
	defer s.subscribersMux.Unlock()
	s.subscribers[subscriberID] = channel
	s.logger.Info("添加新訂閱者", zap.String("subscriberID", subscriberID))
}

func (s *DealerService) removeSubscriber(subscriberID string) {
	s.subscribersMux.Lock()
	defer s.subscribersMux.Unlock()
	if _, exists := s.subscribers[subscriberID]; exists {
		delete(s.subscribers, subscriberID)
		s.logger.Info("移除訂閱者", zap.String("subscriberID", subscriberID))
	}
}

// SubscribeGameEvents 實現 DealerService.SubscribeGameEvents RPC 方法 (流式 RPC)
func (s *DealerService) SubscribeGameEvents(req *pb.SubscribeGameEventsRequest, stream pb.DealerService_SubscribeGameEventsServer) error {
	// 創建一個唯一的訂閱 ID
	subscriptionID := uuid.New().String()
	s.logger.Info("收到新的事件訂閱請求",
		zap.String("subscriptionID", subscriptionID))

	// 創建通道以接收事件
	eventChan := make(chan *pb.GameEvent, 100)

	// 創建一個取消的 context
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// 預設訂閱 UNSPECIFIED, HEARTBEAT 和 NOTIFICATION 事件
	hasHeartbeatSubscription := true
	hasNotificationSubscription := true

	// 注意：由於我們不再使用 req.EventTypes，所以不需要檢查它

	// 如果訂閱了通知事件，註冊為訂閱者
	if hasNotificationSubscription {
		s.addSubscriber(subscriptionID, eventChan)
		// 確保在函數結束時移除訂閱者
		defer s.removeSubscriber(subscriptionID)

		// 立即發送當前遊戲狀態作為通知
		s.logger.Info("客戶端訂閱了通知事件，立即發送當前遊戲狀態",
			zap.String("subscriptionID", subscriptionID))

		// 獲取當前遊戲狀態
		currentGame := s.gameManager.GetCurrentGame()
		if currentGame != nil {
			// 建立通知事件
			notificationEvent := &pb.GameEvent{
				EventType: pb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
				GameId:    currentGame.GameID,
				Timestamp: timestamppb.Now(),
				EventData: &pb.GameEvent_Notification{
					Notification: &pb.NotificationEvent{
						GameData: convertGameDataToPb(currentGame),
						Message:  "訂閱時的遊戲狀態",
					},
				},
			}

			// 直接發送到事件通道
			select {
			case eventChan <- notificationEvent:
				s.logger.Debug("已發送初始通知事件到通道",
					zap.String("subscriptionID", subscriptionID))
			case <-ctx.Done():
				return ctx.Err()
			default:
				s.logger.Warn("事件通道已滿，無法發送初始通知事件",
					zap.String("subscriptionID", subscriptionID))
			}
		}
	}

	// 如果訂閱了心跳事件，啟動定時器每 10 秒發送一次心跳訊息
	if hasHeartbeatSubscription {
		s.logger.Info("客戶端訂閱了心跳事件，將每 10 秒發送一次心跳訊息",
			zap.String("subscriptionID", subscriptionID))

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// 建立心跳事件
					heartbeatEvent := &pb.GameEvent{
						EventType: pb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT,
						GameId:    "system",
						Timestamp: timestamppb.Now(),
						EventData: &pb.GameEvent_Heartbeat{
							Heartbeat: &pb.HeartbeatEvent{
								Message: "hello",
							},
						},
					}

					// 發送到事件通道
					select {
					case eventChan <- heartbeatEvent:
						s.logger.Debug("已發送心跳事件到通道",
							zap.String("subscriptionID", subscriptionID))
					case <-ctx.Done():
						return
					default:
						s.logger.Warn("事件通道已滿，無法發送心跳事件",
							zap.String("subscriptionID", subscriptionID))
					}
				}
			}
		}()
	}

	// 在 goroutine 中處理收到的事件
	go func() {
		defer close(eventChan)

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("客戶端關閉了訂閱",
					zap.String("subscriptionID", subscriptionID),
					zap.Error(ctx.Err()))
				return
			case event := <-eventChan:
				if event == nil {
					s.logger.Warn("收到空事件")
					continue
				}

				// 檢查事件類型，允許處理 UNSPECIFIED、HEARTBEAT 和 NOTIFICATION 事件
				if event.EventType != pb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED &&
					event.EventType != pb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT &&
					event.EventType != pb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION {
					// 跳過不需要的事件類型
					s.logger.Debug("跳過不支持的事件類型",
						zap.String("subscriptionID", subscriptionID),
						zap.String("eventType", event.EventType.String()))
					continue
				}

				// 發送事件到客戶端
				if err := stream.Send(event); err != nil {
					s.logger.Error("發送事件到客戶端失敗",
						zap.String("subscriptionID", subscriptionID),
						zap.Any("event", event),
						zap.Error(err))
					return
				} else {
					s.logger.Debug("成功發送事件到客戶端",
						zap.String("subscriptionID", subscriptionID),
						zap.String("eventType", event.EventType.String()),
						zap.String("gameID", event.GameId))
				}
			}
		}
	}()

	// 等待流關閉
	<-ctx.Done()
	s.logger.Info("事件訂閱結束", zap.String("subscriptionID", subscriptionID))
	return ctx.Err()
}

// 事件處理函數

// onStageChanged 處理階段變更事件
func (s *DealerService) onStageChanged(gameID string, oldStage, newStage gameflow.GameStage) {
	s.logger.Info("遊戲階段變更",
		zap.String("gameID", gameID),
		zap.String("oldStage", string(oldStage)),
		zap.String("newStage", string(newStage)))

	// 將廣播事件放入 goroutine 中執行，避免阻塞回調函數
	go func() {
		// 廣播階段變更事件
		event := map[string]interface{}{
			"type": "stage_changed",
			"data": map[string]interface{}{
				"game_id":   gameID,
				"old_stage": string(oldStage),
				"new_stage": string(newStage),
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		// 使用 WebSocket 廣播事件
		if err := s.broadcastEvent(event); err != nil {
			s.logger.Error("廣播階段變更事件失敗",
				zap.String("gameID", gameID),
				zap.String("oldStage", string(oldStage)),
				zap.String("newStage", string(newStage)),
				zap.Error(err))
		}

		// 發送通知事件給訂閱 NOTIFICATION 的客戶端
		// 獲取當前遊戲狀態
		currentGame := s.gameManager.GetCurrentGame()
		if currentGame != nil {
			notificationEvent := &pb.GameEvent{
				EventType: pb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
				GameId:    gameID,
				Timestamp: timestamppb.Now(),
				EventData: &pb.GameEvent_Notification{
					Notification: &pb.NotificationEvent{
						GameData: convertGameDataToPb(currentGame),
						Message:  fmt.Sprintf("遊戲階段從 %s 變更為 %s", oldStage, newStage),
					},
				},
			}

			// 這裡應該將通知事件發送到通知發送系統
			// 實際實現中，您可能需要添加一個通知管理器來處理訂閱和發布
			s.sendNotificationToSubscribers(notificationEvent)
		}
	}()
}

// onGameCreated 處理遊戲創建事件
func (s *DealerService) onGameCreated(gameID string) {
	s.logger.Info("遊戲創建", zap.String("gameID", gameID))

	// 將廣播事件放入 goroutine 中執行，避免阻塞回調函數
	go func() {
		// 廣播遊戲創建事件
		event := map[string]interface{}{
			"type": "game_created",
			"data": map[string]interface{}{
				"game_id":   gameID,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		// 使用 WebSocket 廣播事件
		if err := s.broadcastEvent(event); err != nil {
			s.logger.Error("廣播遊戲創建事件失敗", zap.String("gameID", gameID), zap.Error(err))
		}
	}()
}

// onGameCancelled 處理遊戲取消事件
func (s *DealerService) onGameCancelled(gameID string, reason string) {
	s.logger.Info("遊戲取消",
		zap.String("gameID", gameID),
		zap.String("reason", reason))

	// 將廣播事件放入 goroutine 中執行，避免阻塞回調函數
	go func() {
		// 廣播遊戲取消事件
		event := map[string]interface{}{
			"type": "game_cancelled",
			"data": map[string]interface{}{
				"game_id":   gameID,
				"reason":    reason,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		// 使用 WebSocket 廣播事件
		if err := s.broadcastEvent(event); err != nil {
			s.logger.Error("廣播遊戲取消事件失敗", zap.String("gameID", gameID), zap.String("reason", reason), zap.Error(err))
		}
	}()
}

// onGameCompleted 處理遊戲完成事件
func (s *DealerService) onGameCompleted(gameID string) {
	s.logger.Info("遊戲完成", zap.String("gameID", gameID))

	// 將廣播事件放入 goroutine 中執行，避免阻塞回調函數
	go func() {
		// 廣播遊戲完成事件
		event := map[string]interface{}{
			"type": "game_completed",
			"data": map[string]interface{}{
				"game_id":   gameID,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		// 使用 WebSocket 廣播事件
		if err := s.broadcastEvent(event); err != nil {
			s.logger.Error("廣播遊戲完成事件失敗", zap.String("gameID", gameID), zap.Error(err))
		}
	}()
}

// onBallDrawn 處理球抽取事件
func (s *DealerService) onBallDrawn(gameID string, ball gameflow.Ball) {
	s.logger.Info("球抽取",
		zap.String("gameID", gameID),
		zap.Int("number", ball.Number),
		zap.String("type", string(ball.Type)),
		zap.Bool("isLast", ball.IsLast))

	// 將廣播事件放入 goroutine 中執行，避免阻塞回調函數
	go func() {
		// 廣播球抽取事件
		event := map[string]interface{}{
			"type": "ball_drawn",
			"data": map[string]interface{}{
				"game_id":   gameID,
				"number":    ball.Number,
				"ball_type": string(ball.Type),
				"is_last":   ball.IsLast,
				"timestamp": ball.Timestamp.Format(time.RFC3339),
			},
		}

		// 使用 WebSocket 廣播事件
		if err := s.broadcastEvent(event); err != nil {
			s.logger.Error("廣播球抽取事件失敗",
				zap.String("gameID", gameID),
				zap.Int("number", ball.Number),
				zap.String("type", string(ball.Type)),
				zap.Error(err))
		}
	}()
}

// onExtraBallSideSelected 處理額外球選邊事件
func (s *DealerService) onExtraBallSideSelected(gameID string, side gameflow.ExtraBallSide) {
	s.logger.Info("額外球選邊",
		zap.String("gameID", gameID),
		zap.String("side", string(side)))

	// 將廣播事件放入 goroutine 中執行，避免阻塞回調函數
	go func() {
		// 廣播額外球選邊事件
		event := map[string]interface{}{
			"type": "extra_ball_side_selected",
			"data": map[string]interface{}{
				"game_id":   gameID,
				"side":      string(side),
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		// 使用 WebSocket 廣播事件
		if err := s.broadcastEvent(event); err != nil {
			s.logger.Error("廣播額外球選邊事件失敗", zap.String("gameID", gameID), zap.String("side", string(side)), zap.Error(err))
		}
	}()
}

// broadcastEvent 廣播事件到所有連接的荷官端
func (s *DealerService) broadcastEvent(event map[string]interface{}) error {
	// 通過 dealerServer 廣播事件
	return s.dealerServer.BroadcastMessage(event)
}

// broadcastNewGameEvent 廣播新遊戲事件
func (s *DealerService) broadcastNewGameEvent(gameID string, game *gameflow.GameData) {
	// 將廣播事件放入 goroutine 中執行，避免阻塞
	go func() {
		event := map[string]interface{}{
			"type": "new_round_started",
			"data": map[string]interface{}{
				"game_id":   gameID,
				"stage":     string(game.CurrentStage),
				"timestamp": game.StartTime.Format(time.RFC3339),
			},
		}

		// 使用 WebSocket 廣播事件
		if err := s.broadcastEvent(event); err != nil {
			s.logger.Error("廣播新遊戲事件失敗", zap.String("gameID", gameID), zap.Error(err))
		}
	}()
}

// 輔助函數：轉換 GameStage 到 proto GameStage
func convertGameStageToPb(stage gameflow.GameStage) pb.GameStage {
	switch stage {
	case gameflow.StagePreparation:
		return pb.GameStage_GAME_STAGE_PREPARATION
	case gameflow.StageNewRound:
		return pb.GameStage_GAME_STAGE_NEW_ROUND
	case gameflow.StageCardPurchaseOpen:
		return pb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case gameflow.StageCardPurchaseClose:
		return pb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case gameflow.StageDrawingStart:
		return pb.GameStage_GAME_STAGE_DRAWING_START
	case gameflow.StageDrawingClose:
		return pb.GameStage_GAME_STAGE_DRAWING_CLOSE
	case gameflow.StageExtraBallPrepare:
		return pb.GameStage_GAME_STAGE_EXTRA_BALL_PREPARE
	case gameflow.StageExtraBallSideSelectBettingStart:
		return pb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_START
	case gameflow.StageExtraBallSideSelectBettingClosed:
		return pb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED
	case gameflow.StageExtraBallWaitClaim:
		return pb.GameStage_GAME_STAGE_EXTRA_BALL_WAIT_CLAIM
	case gameflow.StageExtraBallDrawingStart:
		return pb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START
	case gameflow.StageExtraBallDrawingClose:
		return pb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_CLOSE
	case gameflow.StagePayoutSettlement:
		return pb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	case gameflow.StageJackpotPreparation:
		return pb.GameStage_GAME_STAGE_JACKPOT_START
	case gameflow.StageJackpotDrawingStart:
		return pb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START
	case gameflow.StageJackpotDrawingClosed:
		return pb.GameStage_GAME_STAGE_JACKPOT_DRAWING_CLOSED
	case gameflow.StageJackpotSettlement:
		return pb.GameStage_GAME_STAGE_JACKPOT_SETTLEMENT
	case gameflow.StageDrawingLuckyBallsStart:
		return pb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START
	case gameflow.StageDrawingLuckyBallsClosed:
		return pb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_CLOSED
	case gameflow.StageGameOver:
		return pb.GameStage_GAME_STAGE_GAME_OVER
	default:
		return pb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// 輔助函數：轉換 ExtraBallSide 到 proto ExtraBallSide
func convertExtraBallSideToPb(side gameflow.ExtraBallSide) pb.ExtraBallSide {
	switch side {
	case gameflow.ExtraBallSideLeft:
		return pb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case gameflow.ExtraBallSideRight:
		return pb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return pb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// 輔助函數：轉換 Ball 到 proto Ball
func convertBallToPb(ball gameflow.Ball) *pb.Ball {
	var ballType pb.BallType

	switch ball.Type {
	case gameflow.BallTypeRegular:
		ballType = pb.BallType_BALL_TYPE_REGULAR
	case gameflow.BallTypeExtra:
		ballType = pb.BallType_BALL_TYPE_EXTRA
	case gameflow.BallTypeJackpot:
		ballType = pb.BallType_BALL_TYPE_JACKPOT
	case gameflow.BallTypeLucky:
		ballType = pb.BallType_BALL_TYPE_LUCKY
	default:
		ballType = pb.BallType_BALL_TYPE_UNSPECIFIED
	}

	return &pb.Ball{
		Number:    int32(ball.Number),
		Type:      ballType,
		IsLast:    ball.IsLast,
		Timestamp: timestamppb.New(ball.Timestamp),
	}
}

// 輔助函數：轉換 GameData 到 proto GameData
func convertGameDataToPb(game *gameflow.GameData) *pb.GameData {
	if game == nil {
		return nil
	}

	// 初始化球數組
	regularBalls := make([]*pb.Ball, len(game.RegularBalls))
	for i, ball := range game.RegularBalls {
		regularBalls[i] = convertBallToPb(ball)
	}

	extraBalls := make([]*pb.Ball, len(game.ExtraBalls))
	for i, ball := range game.ExtraBalls {
		extraBalls[i] = convertBallToPb(ball)
	}

	// 初始化JP球和幸運球
	var jackpotBalls []*pb.Ball
	var luckyBalls []*pb.Ball

	if game.Jackpot != nil {
		// 如果有 Jackpot 數據，從 Jackpot 結構中獲取
		jackpotBalls = make([]*pb.Ball, len(game.Jackpot.DrawnBalls))
		for i, ball := range game.Jackpot.DrawnBalls {
			jackpotBalls[i] = convertBallToPb(ball)
		}

		luckyBalls = make([]*pb.Ball, len(game.Jackpot.LuckyBalls))
		for i, ball := range game.Jackpot.LuckyBalls {
			luckyBalls[i] = convertBallToPb(ball)
		}
	} else {
		jackpotBalls = make([]*pb.Ball, 0)
		luckyBalls = make([]*pb.Ball, 0)
	}

	// 創建 proto GameData
	pbGame := &pb.GameData{
		GameId:         game.GameID,
		CurrentStage:   convertGameStageToPb(game.CurrentStage),
		StartTime:      timestamppb.New(game.StartTime),
		RegularBalls:   regularBalls,
		ExtraBalls:     extraBalls,
		JackpotBalls:   jackpotBalls,
		LuckyBalls:     luckyBalls,
		SelectedSide:   convertExtraBallSideToPb(game.SelectedSide),
		HasJackpot:     game.HasJackpot,
		ExtraBallCount: int32(game.ExtraBallCount),
		LastUpdateTime: timestamppb.New(game.LastUpdateTime),
	}

	// 處理可選欄位
	if !game.EndTime.IsZero() {
		pbGame.EndTime = timestamppb.New(game.EndTime)
	}

	if !game.LastUpdateTime.IsZero() {
		pbGame.LastUpdateTime = timestamppb.New(game.LastUpdateTime)
	}

	return pbGame
}

// StartJackpotRound 開始JP回合，從 StageJackpotPreparation 進入到 StageJackpotDrawingStart
func (s *DealerService) StartJackpotRound(ctx context.Context, req *pb.StartJackpotRoundRequest) (*pb.StartJackpotRoundResponse, error) {
	s.logger.Info("接收到開始JP回合請求")

	// 取得當前遊戲
	game := s.gameManager.GetCurrentGame()
	if game == nil {
		return nil, status.Errorf(codes.NotFound, "找不到進行中的遊戲")
	}

	// 確認當前階段是否為 StageJackpotPreparation
	if game.CurrentStage != gameflow.StageJackpotPreparation {
		return nil, status.Errorf(codes.FailedPrecondition, "當前階段 %s 不是 JP準備階段", game.CurrentStage)
	}

	// 記錄原始階段
	oldStage := game.CurrentStage

	// 手動推進遊戲階段
	err := s.gameManager.AdvanceStage(ctx, false)
	if err != nil {
		s.logger.Error("推進遊戲階段失敗", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "推進遊戲階段失敗: %v", err)
	}

	// 重新獲取最新的遊戲狀態
	game = s.gameManager.GetCurrentGame()

	// 傳送通知
	s.broadcastNewGameEvent(game.GameID, game)

	// 返回成功響應
	return &pb.StartJackpotRoundResponse{
		Success:  true,
		GameId:   game.GameID,
		OldStage: convertGameStageToPb(oldStage),
		NewStage: convertGameStageToPb(game.CurrentStage),
	}, nil
}
