package dealer

import (
	"context"
	"time"

	"g38_lottery_service/internal/dealerWebsocket"
	"g38_lottery_service/internal/gameflow"
	pb "g38_lottery_service/internal/proto/dealer"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DealerService 實現 gRPC 中定義的 DealerService 接口
type DealerService struct {
	pb.UnimplementedDealerServiceServer
	logger       *zap.Logger
	gameManager  *gameflow.GameManager
	dealerServer *dealerWebsocket.DealerServer
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
	}

	// 註冊事件處理函數
	service.registerEventHandlers()

	return service
}

// 註冊事件處理函數
func (s *DealerService) registerEventHandlers() {
	// 註冊階段變更事件處理函數
	s.gameManager.SetEventHandlers(
		s.onStageChanged,          // 階段變更
		s.onGameCreated,           // 遊戲創建
		s.onGameCancelled,         // 遊戲取消
		s.onGameCompleted,         // 遊戲完成
		s.onBallDrawn,             // 球抽取
		s.onExtraBallSideSelected, // 額外球選邊
	)
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

	// 推進到新局階段
	err = s.gameManager.AdvanceStage(ctx, true)
	if err != nil {
		s.logger.Error("推進到新局階段失敗", zap.Error(err))
		return nil, err
	}

	// 構建響應
	response := &pb.StartNewRoundResponse{
		GameId:       gameID,
		StartTime:    timestamppb.New(game.StartTime),
		CurrentStage: convertGameStageToPb(game.CurrentStage),
	}

	s.logger.Info("成功開始新局",
		zap.String("gameID", gameID),
		zap.String("stage", string(game.CurrentStage)))

	// 通過 WebSocket 廣播新遊戲開始事件
	s.broadcastNewGameEvent(gameID, game)

	return response, nil
}

// DrawBall 實現 DealerService.DrawBall RPC 方法
func (s *DealerService) DrawBall(ctx context.Context, req *pb.DrawBallRequest) (*pb.DrawBallResponse, error) {
	// 抽取常規球
	ball, err := s.gameManager.HandleDrawBall(ctx, int(req.Number), req.IsLast)
	if err != nil {
		return nil, err
	}

	return &pb.DrawBallResponse{
		Ball: &pb.Ball{
			Number:    int32(ball.Number),
			Type:      pb.BallType_BALL_TYPE_REGULAR,
			IsLast:    ball.IsLast,
			Timestamp: timestamppb.New(ball.Timestamp),
		},
	}, nil
}

// DrawExtraBall 實現 DealerService.DrawExtraBall RPC 方法
func (s *DealerService) DrawExtraBall(ctx context.Context, req *pb.DrawExtraBallRequest) (*pb.DrawExtraBallResponse, error) {
	// 抽取額外球
	ball, err := s.gameManager.HandleDrawExtraBall(ctx, int(req.Number), req.IsLast)
	if err != nil {
		return nil, err
	}

	return &pb.DrawExtraBallResponse{
		Ball: &pb.Ball{
			Number:    int32(ball.Number),
			Type:      pb.BallType_BALL_TYPE_EXTRA,
			IsLast:    ball.IsLast,
			Timestamp: timestamppb.New(ball.Timestamp),
		},
	}, nil
}

// DrawJackpotBall 實現 DealerService.DrawJackpotBall RPC 方法
func (s *DealerService) DrawJackpotBall(ctx context.Context, req *pb.DrawJackpotBallRequest) (*pb.DrawJackpotBallResponse, error) {
	// 抽取JP球
	ball, err := s.gameManager.HandleDrawJackpotBall(ctx, int(req.Number), req.IsLast)
	if err != nil {
		return nil, err
	}

	return &pb.DrawJackpotBallResponse{
		Ball: &pb.Ball{
			Number:    int32(ball.Number),
			Type:      pb.BallType_BALL_TYPE_JACKPOT,
			IsLast:    ball.IsLast,
			Timestamp: timestamppb.New(ball.Timestamp),
		},
	}, nil
}

// DrawLuckyBall 實現 DealerService.DrawLuckyBall RPC 方法
func (s *DealerService) DrawLuckyBall(ctx context.Context, req *pb.DrawLuckyBallRequest) (*pb.DrawLuckyBallResponse, error) {
	// 抽取幸運號碼球
	ball, err := s.gameManager.HandleDrawLuckyBall(ctx, int(req.Number), req.IsLast)
	if err != nil {
		return nil, err
	}

	return &pb.DrawLuckyBallResponse{
		Ball: &pb.Ball{
			Number:    int32(ball.Number),
			Type:      pb.BallType_BALL_TYPE_LUCKY,
			IsLast:    ball.IsLast,
			Timestamp: timestamppb.New(ball.Timestamp),
		},
	}, nil
}

// NotifyJackpotWinner 實現 DealerService.NotifyJackpotWinner RPC 方法
func (s *DealerService) NotifyJackpotWinner(ctx context.Context, req *pb.NotifyJackpotWinnerRequest) (*pb.GameData, error) {
	// 通知JP獲獎者
	err := s.gameManager.NotifyJackpotWinner(ctx, req.WinnerId)
	if err != nil {
		return nil, err
	}

	// 返回更新後的遊戲數據
	return convertGameDataToPb(s.gameManager.GetCurrentGame()), nil
}

// SetHasJackpot 實現 DealerService.SetHasJackpot RPC 方法
func (s *DealerService) SetHasJackpot(ctx context.Context, req *pb.SetHasJackpotRequest) (*pb.GameData, error) {
	// 設置遊戲是否啟用JP
	err := s.gameManager.SetHasJackpot(ctx, req.HasJackpot)
	if err != nil {
		return nil, err
	}

	// 返回更新後的遊戲數據
	return convertGameDataToPb(s.gameManager.GetCurrentGame()), nil
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

// GetGameStatus 實現 DealerService.GetGameStatus RPC 方法
func (s *DealerService) GetGameStatus(ctx context.Context, req *pb.GetGameStatusRequest) (*pb.GetGameStatusResponse, error) {
	// 獲取當前遊戲狀態
	gameData := s.gameManager.GetCurrentGame()
	if gameData == nil {
		return nil, gameflow.ErrGameNotFound
	}

	return &pb.GetGameStatusResponse{
		GameData: convertGameDataToPb(gameData),
	}, nil
}

// SubscribeGameEvents 實現 DealerService.SubscribeGameEvents RPC 方法 (流式 RPC)
func (s *DealerService) SubscribeGameEvents(req *pb.SubscribeGameEventsRequest, stream pb.DealerService_SubscribeGameEventsServer) error {
	// 創建一個唯一的訂閱 ID
	subscriptionID := uuid.New().String()
	s.logger.Info("收到新的事件訂閱請求",
		zap.String("subscriptionID", subscriptionID),
		zap.Any("eventTypes", req.EventTypes))

	// 創建通道以接收事件
	eventChan := make(chan *pb.GameEvent, 100)

	// TODO: 在實際實現中，可以使用更複雜的事件訂閱系統，
	// 例如使用 Redis Pub/Sub 或其他消息代理。
	// 這裡只是一個簡單的示例。

	// 創建一個取消的 context
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

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

				// 檢查事件類型是否匹配訂閱的類型
				if len(req.EventTypes) > 0 {
					found := false
					for _, t := range req.EventTypes {
						if t == event.EventType {
							found = true
							break
						}
					}
					if !found {
						continue // 跳過不匹配的事件類型
					}
				}

				// 發送事件到客戶端
				if err := stream.Send(event); err != nil {
					s.logger.Error("發送事件到客戶端失敗",
						zap.String("subscriptionID", subscriptionID),
						zap.Any("event", event),
						zap.Error(err))
					return
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
	s.broadcastEvent(event)
}

// onGameCreated 處理遊戲創建事件
func (s *DealerService) onGameCreated(gameID string) {
	s.logger.Info("遊戲創建", zap.String("gameID", gameID))

	// 廣播遊戲創建事件
	event := map[string]interface{}{
		"type": "game_created",
		"data": map[string]interface{}{
			"game_id":   gameID,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	// 使用 WebSocket 廣播事件
	s.broadcastEvent(event)
}

// onGameCancelled 處理遊戲取消事件
func (s *DealerService) onGameCancelled(gameID string, reason string) {
	s.logger.Info("遊戲取消",
		zap.String("gameID", gameID),
		zap.String("reason", reason))

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
	s.broadcastEvent(event)
}

// onGameCompleted 處理遊戲完成事件
func (s *DealerService) onGameCompleted(gameID string) {
	s.logger.Info("遊戲完成", zap.String("gameID", gameID))

	// 廣播遊戲完成事件
	event := map[string]interface{}{
		"type": "game_completed",
		"data": map[string]interface{}{
			"game_id":   gameID,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	// 使用 WebSocket 廣播事件
	s.broadcastEvent(event)
}

// onBallDrawn 處理球抽取事件
func (s *DealerService) onBallDrawn(gameID string, ball gameflow.Ball) {
	s.logger.Info("球抽取",
		zap.String("gameID", gameID),
		zap.Int("number", ball.Number),
		zap.String("type", string(ball.Type)),
		zap.Bool("isLast", ball.IsLast))

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
	s.broadcastEvent(event)
}

// onExtraBallSideSelected 處理額外球選邊事件
func (s *DealerService) onExtraBallSideSelected(gameID string, side gameflow.ExtraBallSide) {
	s.logger.Info("額外球選邊",
		zap.String("gameID", gameID),
		zap.String("side", string(side)))

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
	s.broadcastEvent(event)
}

// broadcastEvent 廣播事件到所有連接的荷官端
func (s *DealerService) broadcastEvent(event map[string]interface{}) {
	// 通過 dealerServer 廣播事件
	if err := s.dealerServer.BroadcastMessage(event); err != nil {
		s.logger.Error("廣播事件失敗", zap.Error(err))
	}
}

// broadcastNewGameEvent 廣播新遊戲事件
func (s *DealerService) broadcastNewGameEvent(gameID string, game *gameflow.GameData) {
	event := map[string]interface{}{
		"type": "new_round_started",
		"data": map[string]interface{}{
			"game_id":   gameID,
			"stage":     string(game.CurrentStage),
			"timestamp": game.StartTime.Format(time.RFC3339),
		},
	}

	// 使用 WebSocket 廣播事件
	s.broadcastEvent(event)
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
	case gameflow.StageJackpotStart:
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

	// 準備球的切片
	regularBalls := make([]*pb.Ball, len(game.RegularBalls))
	for i, ball := range game.RegularBalls {
		regularBalls[i] = convertBallToPb(ball)
	}

	extraBalls := make([]*pb.Ball, len(game.ExtraBalls))
	for i, ball := range game.ExtraBalls {
		extraBalls[i] = convertBallToPb(ball)
	}

	jackpotBalls := make([]*pb.Ball, len(game.JackpotBalls))
	for i, ball := range game.JackpotBalls {
		jackpotBalls[i] = convertBallToPb(ball)
	}

	luckyBalls := make([]*pb.Ball, len(game.LuckyBalls))
	for i, ball := range game.LuckyBalls {
		luckyBalls[i] = convertBallToPb(ball)
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
		JackpotWinner:  game.JackpotWinner,
		IsCancelled:    game.IsCancelled,
		CancelReason:   game.CancelReason,
		LastUpdateTime: timestamppb.New(game.LastUpdateTime),
	}

	// 處理可選欄位
	if !game.EndTime.IsZero() {
		pbGame.EndTime = timestamppb.New(game.EndTime)
	}

	if !game.CancelTime.IsZero() {
		pbGame.CancelTime = timestamppb.New(game.CancelTime)
	}

	return pbGame
}
