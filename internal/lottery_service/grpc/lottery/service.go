package lottery

import (
	"context"
	"sync"

	"g38_lottery_service/internal/generated/api/v1/lottery"
	"g38_lottery_service/internal/generated/common"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/internal/lottery_service/grpc/dealer"
	dealerpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LotteryService 實現 lottery.LotteryServiceServer 接口
type LotteryService struct {
	lottery.UnimplementedLotteryServiceServer
	logger          *zap.Logger
	gameManager     *gameflow.GameManager
	dealerService   *dealer.DealerService
	subscribers     map[string]chan *lottery.GameEvent // 訂閱者映射表
	subscribersMux  sync.RWMutex                       // 訂閱者鎖
	roomSubscribers map[string]map[string]bool         // 房間到訂閱者的映射表，存儲房間下有哪些訂閱者
	roomSubMux      sync.RWMutex                       // 房間訂閱者鎖
}

// NewLotteryService 創建新的 LotteryService 實例
func NewLotteryService(
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
	dealerService *dealer.DealerService,
) *LotteryService {
	service := &LotteryService{
		logger:          logger.Named("lottery_service"),
		gameManager:     gameManager,
		dealerService:   dealerService,
		subscribers:     make(map[string]chan *lottery.GameEvent),
		roomSubscribers: make(map[string]map[string]bool),
	}

	// 註冊事件處理函數
	service.registerEventHandlers()

	return service
}

// 註冊事件處理函數
func (s *LotteryService) registerEventHandlers() {
	s.logger.Info("註冊事件處理函數")
	// 使用 dealerService 已經註冊的事件處理函數，這裡不再重複註冊
	s.logger.Info("事件處理函數註冊完成")
}

// StartNewRound 實現 LotteryService.StartNewRound RPC 方法
func (s *LotteryService) StartNewRound(ctx context.Context, req *lottery.StartNewRoundRequest) (*lottery.StartNewRoundResponse, error) {
	s.logger.Info("收到開始新局請求", zap.String("roomID", req.RoomId))

	// 轉換請求
	dealerReq := &dealerpb.StartNewRoundRequest{
		RoomId: req.RoomId,
	}

	// 調用 DealerService
	dealerResp, err := s.dealerService.StartNewRound(ctx, dealerReq)
	if err != nil {
		s.logger.Error("開始新局失敗", zap.Error(err))
		return nil, err
	}

	// 轉換響應
	resp := &lottery.StartNewRoundResponse{
		GameId:       dealerResp.GameId,
		StartTime:    dealerResp.StartTime,
		CurrentStage: ConvertGameStageFromDealerPb(dealerResp.CurrentStage),
	}

	return resp, nil
}

// DrawBall 實現 LotteryService.DrawBall RPC 方法
func (s *LotteryService) DrawBall(ctx context.Context, req *lottery.DrawBallRequest) (*lottery.DrawBallResponse, error) {
	s.logger.Info("收到抽取常規球請求", zap.String("roomID", req.RoomId))

	// 轉換請求
	dealerReq := &dealerpb.DrawBallRequest{
		RoomId: req.RoomId,
	}

	// 轉換球數據
	for _, ball := range req.Balls {
		dealerBall := ConvertBallToDealerPb(ball)
		dealerReq.Balls = append(dealerReq.Balls, dealerBall)
	}

	// 調用 DealerService
	dealerResp, err := s.dealerService.DrawBall(ctx, dealerReq)
	if err != nil {
		s.logger.Error("抽取常規球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換響應
	resp := ConvertDrawBallResponse(dealerResp)
	// 設置 GameId，因為舊版 DrawBallResponse 中沒有 GameId
	// 這裡使用從 GameManager 獲取的當前遊戲 ID
	currentGame := s.gameManager.GetCurrentGameByRoom(req.RoomId)
	if currentGame != nil {
		resp.GameId = currentGame.GameID
	}

	return resp, nil
}

// DrawExtraBall 實現 LotteryService.DrawExtraBall RPC 方法
func (s *LotteryService) DrawExtraBall(ctx context.Context, req *lottery.DrawExtraBallRequest) (*lottery.DrawExtraBallResponse, error) {
	s.logger.Info("收到抽取額外球請求", zap.String("roomID", req.RoomId))

	// 轉換請求
	dealerReq := &dealerpb.DrawExtraBallRequest{
		RoomId: req.RoomId,
	}

	// 如果有球號，創建一個 Ball 並添加到請求中
	if req.BallNumber > 0 {
		dealerReq.Balls = []*dealerpb.Ball{
			{
				Number:    req.BallNumber,
				Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
				IsLast:    true,
				Timestamp: nil, // 會在 DealerService 中設置
			},
		}
	}

	// 調用 DealerService
	dealerResp, err := s.dealerService.DrawExtraBall(ctx, dealerReq)
	if err != nil {
		s.logger.Error("抽取額外球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換響應
	resp := ConvertDrawExtraBallResponse(dealerResp)
	// 設置 GameId
	currentGame := s.gameManager.GetCurrentGameByRoom(req.RoomId)
	if currentGame != nil {
		resp.GameId = currentGame.GameID
	}

	return resp, nil
}

// DrawJackpotBall 實現 LotteryService.DrawJackpotBall RPC 方法
func (s *LotteryService) DrawJackpotBall(ctx context.Context, req *lottery.DrawJackpotBallRequest) (*lottery.DrawJackpotBallResponse, error) {
	s.logger.Info("收到抽取頭獎球請求", zap.String("roomID", req.RoomId))

	// 轉換請求
	dealerReq := &dealerpb.DrawJackpotBallRequest{
		RoomId: req.RoomId,
	}

	// 如果有球號，創建一個 Ball 並添加到請求中
	if req.BallNumber > 0 {
		dealerReq.Balls = []*dealerpb.Ball{
			{
				Number:    req.BallNumber,
				Type:      dealerpb.BallType_BALL_TYPE_JACKPOT,
				IsLast:    true,
				Timestamp: nil, // 會在 DealerService 中設置
			},
		}
	}

	// 調用 DealerService
	dealerResp, err := s.dealerService.DrawJackpotBall(ctx, dealerReq)
	if err != nil {
		s.logger.Error("抽取頭獎球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換響應
	resp := ConvertDrawJackpotBallResponse(dealerResp)
	// 設置 GameId
	currentGame := s.gameManager.GetCurrentGameByRoom(req.RoomId)
	if currentGame != nil {
		resp.GameId = currentGame.GameID
	}

	return resp, nil
}

// DrawLuckyBall 實現 LotteryService.DrawLuckyBall RPC 方法
func (s *LotteryService) DrawLuckyBall(ctx context.Context, req *lottery.DrawLuckyBallRequest) (*lottery.DrawLuckyBallResponse, error) {
	s.logger.Info("收到抽取幸運球請求", zap.String("roomID", req.RoomId))

	// 轉換請求
	dealerReq := &dealerpb.DrawLuckyBallRequest{
		RoomId: req.RoomId,
	}

	// 如果有球號，創建 Ball 並添加到請求中
	if len(req.BallNumbers) > 0 {
		dealerReq.Balls = make([]*dealerpb.Ball, 0, len(req.BallNumbers))
		for i, num := range req.BallNumbers {
			isLast := i == len(req.BallNumbers)-1
			dealerReq.Balls = append(dealerReq.Balls, &dealerpb.Ball{
				Number:    num,
				Type:      dealerpb.BallType_BALL_TYPE_LUCKY,
				IsLast:    isLast,
				Timestamp: nil, // 會在 DealerService 中設置
			})
		}
	}

	// 調用 DealerService
	dealerResp, err := s.dealerService.DrawLuckyBall(ctx, dealerReq)
	if err != nil {
		s.logger.Error("抽取幸運球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換響應
	resp := ConvertDrawLuckyBallResponse(dealerResp)
	// 設置 GameId
	currentGame := s.gameManager.GetCurrentGameByRoom(req.RoomId)
	if currentGame != nil {
		resp.GameId = currentGame.GameID
	}

	return resp, nil
}

// CancelGame 實現 LotteryService.CancelGame RPC 方法
func (s *LotteryService) CancelGame(ctx context.Context, req *lottery.CancelGameRequest) (*lottery.GameData, error) {
	s.logger.Info("收到取消遊戲請求", zap.String("roomID", req.RoomId))

	// 轉換請求
	dealerReq := &dealerpb.CancelGameRequest{
		RoomId: req.RoomId,
		Reason: req.Reason,
	}

	// 調用 DealerService
	_, err := s.dealerService.CancelGame(ctx, dealerReq)
	if err != nil {
		s.logger.Error("取消遊戲失敗", zap.Error(err))
		return nil, err
	}

	// 創建遊戲數據
	currentGame := s.gameManager.GetCurrentGameByRoom(req.RoomId)
	if currentGame == nil {
		return nil, status.Errorf(codes.NotFound, "未找到房間 %s 的當前遊戲", req.RoomId)
	}

	// 獲取遊戲狀態
	statusResp, err := s.GetGameStatus(ctx, &lottery.GetGameStatusRequest{RoomId: req.RoomId})
	if err != nil {
		s.logger.Error("取消遊戲後獲取遊戲狀態失敗", zap.Error(err))
	}

	var gameData *lottery.GameData
	if statusResp != nil && statusResp.GameData != nil {
		gameData = statusResp.GameData
		// 設置取消標誌
		gameData.IsValid = false
		gameData.CancelReason = req.Reason
	} else {
		// 創建一個基本的遊戲數據
		gameData = &lottery.GameData{
			GameId:       currentGame.GameID,
			RoomId:       req.RoomId,
			IsValid:      false,
			CancelReason: req.Reason,
			CurrentStage: common.GameStage_GAME_STAGE_GAME_OVER,
			Status: &common.GameStatus{
				Stage:   common.GameStage_GAME_STAGE_GAME_OVER,
				Message: "遊戲已取消: " + req.Reason,
			},
		}
	}

	return gameData, nil
}

// GetGameStatus 實現 LotteryService.GetGameStatus RPC 方法
func (s *LotteryService) GetGameStatus(ctx context.Context, req *lottery.GetGameStatusRequest) (*lottery.GetGameStatusResponse, error) {
	s.logger.Info("收到獲取遊戲狀態請求", zap.String("roomID", req.RoomId))

	// 轉換請求
	dealerReq := &dealerpb.GetGameStatusRequest{
		RoomId: req.RoomId,
	}

	// 調用 DealerService
	dealerResp, err := s.dealerService.GetGameStatus(ctx, dealerReq)
	if err != nil {
		s.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換響應
	resp := ConvertGetGameStatusResponse(dealerResp)
	return resp, nil
}

// StartJackpotRound 實現 LotteryService.StartJackpotRound RPC 方法
func (s *LotteryService) StartJackpotRound(ctx context.Context, req *lottery.StartJackpotRoundRequest) (*lottery.StartJackpotRoundResponse, error) {
	s.logger.Info("收到開始頭獎回合請求", zap.String("roomID", req.RoomId))

	// 轉換請求 - 注意 DealerService.StartJackpotRound 不需要 RoomId
	dealerReq := &dealerpb.StartJackpotRoundRequest{}

	// 調用 DealerService
	dealerResp, err := s.dealerService.StartJackpotRound(ctx, dealerReq)
	if err != nil {
		s.logger.Error("開始頭獎回合失敗", zap.Error(err))
		return nil, err
	}

	// 轉換響應
	resp := ConvertStartJackpotRoundResponse(dealerResp)
	return resp, nil
}

// SubscribeGameEvents 實現 LotteryService.SubscribeGameEvents RPC 方法
func (s *LotteryService) SubscribeGameEvents(req *lottery.SubscribeGameEventsRequest, stream lottery.LotteryService_SubscribeGameEventsServer) error {
	s.logger.Info("收到訂閱遊戲事件請求", zap.String("roomID", req.RoomId))

	if req.RoomId == "" {
		return status.Errorf(codes.InvalidArgument, "房間ID不能為空")
	}

	// 生成訂閱者ID
	subscriberID := s.generateSubscriberID(req.RoomId)

	// 創建事件頻道
	eventChan := make(chan *lottery.GameEvent, 100)

	// 註冊訂閱者
	s.addSubscriber(subscriberID, req.RoomId, eventChan)

	defer func() {
		// 取消訂閱並關閉通道
		s.removeSubscriber(subscriberID, req.RoomId)
		close(eventChan)
		s.logger.Info("關閉訂閱者連接",
			zap.String("subscriberID", subscriberID),
			zap.String("roomID", req.RoomId))
	}()

	s.logger.Info("成功註冊訂閱者",
		zap.String("subscriberID", subscriberID),
		zap.String("roomID", req.RoomId))

	// 主動發送當前遊戲狀態
	s.logger.Info("用戶請求獲取當前遊戲狀態", zap.String("roomID", req.RoomId))

	// 獲取當前遊戲狀態並發送
	statusResp, err := s.GetGameStatus(context.Background(), &lottery.GetGameStatusRequest{
		RoomId: req.RoomId,
	})

	if err == nil && statusResp != nil && statusResp.GameData != nil {
		gameData := statusResp.GameData

		// 創建事件時間戳
		now := timestamppb.Now()

		// 創建事件並發送
		event := &lottery.GameEvent{
			EventId:   GenerateRandomString(10),
			GameId:    gameData.GameId,
			Type:      common.GameEventType_GAME_EVENT_TYPE_NOTIFICATION,
			Timestamp: now,
			RoomId:    req.RoomId,
			Stage:     gameData.CurrentStage,
		}

		// 設置事件數據
		event.EventData = &lottery.GameEvent_NewGame{
			NewGame: &lottery.NewGameEvent{
				GameData: gameData,
			},
		}

		// 嘗試發送
		select {
		case eventChan <- event:
			s.logger.Info("已發送當前遊戲狀態給新訂閱者",
				zap.String("subscriberID", subscriberID),
				zap.String("roomID", req.RoomId),
				zap.String("gameID", gameData.GameId))
		default:
			s.logger.Warn("訂閱者通道已滿，無法發送當前遊戲狀態",
				zap.String("subscriberID", subscriberID),
				zap.String("roomID", req.RoomId))
		}
	}

	// 處理事件流
	for event := range eventChan {
		if err := stream.Send(event); err != nil {
			s.logger.Error("發送事件到客戶端失敗",
				zap.String("subscriberID", subscriberID),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// 生成訂閱者 ID
func (s *LotteryService) generateSubscriberID(roomID string) string {
	return "subscriber_" + roomID + "_" + GenerateRandomString(8)
}

// 新增訂閱者
func (s *LotteryService) addSubscriber(subscriberID string, roomID string, channel chan *lottery.GameEvent) {
	// 添加到訂閱者映射表
	s.subscribersMux.Lock()
	s.subscribers[subscriberID] = channel
	s.subscribersMux.Unlock()

	// 添加到房間-訂閱者映射表
	s.roomSubMux.Lock()
	if _, exists := s.roomSubscribers[roomID]; !exists {
		s.roomSubscribers[roomID] = make(map[string]bool)
	}
	s.roomSubscribers[roomID][subscriberID] = true
	s.roomSubMux.Unlock()
}

// 移除訂閱者
func (s *LotteryService) removeSubscriber(subscriberID string, roomID string) {
	// 從訂閱者映射表中移除
	s.subscribersMux.Lock()
	delete(s.subscribers, subscriberID)
	s.subscribersMux.Unlock()

	// 從房間-訂閱者映射表中移除
	s.roomSubMux.Lock()
	if subs, exists := s.roomSubscribers[roomID]; exists {
		delete(subs, subscriberID)
		if len(subs) == 0 {
			delete(s.roomSubscribers, roomID)
		}
	}
	s.roomSubMux.Unlock()
}
