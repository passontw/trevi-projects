package dealer

import (
	"context"

	newpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"
	oldpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"go.uber.org/zap"
)

// DealerServiceAdapter 是將新的 API 轉換到舊的 API 的適配器
type DealerServiceAdapter struct {
	logger         *zap.Logger
	originalDealer oldpb.DealerServiceServer
}

// NewDealerServiceAdapter 創建一個新的 dealer 服務適配器
func NewDealerServiceAdapter(
	logger *zap.Logger,
	originalDealer oldpb.DealerServiceServer,
) *DealerServiceAdapter {
	return &DealerServiceAdapter{
		logger:         logger.Named("dealer_service_adapter"),
		originalDealer: originalDealer,
	}
}

// StartNewRound 處理開始新遊戲回合的請求
func (a *DealerServiceAdapter) StartNewRound(ctx context.Context, req *newpb.StartNewRoundRequest) (*newpb.StartNewRoundResponse, error) {
	a.logger.Info("收到開始新遊戲回合請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.StartNewRoundRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
	}

	// 調用原始服務
	oldResp, err := a.originalDealer.StartNewRound(ctx, oldReq)
	if err != nil {
		a.logger.Error("開始新遊戲回合失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.StartNewRoundResponse{
		GameData: ConvertGameDataFromOldPb(oldResp.GetGameData()),
	}

	return newResp, nil
}

// DrawBall 處理抽球請求
func (a *DealerServiceAdapter) DrawBall(ctx context.Context, req *newpb.DrawBallRequest) (*newpb.DrawBallResponse, error) {
	a.logger.Info("收到抽球請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.DrawBallRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
	}

	// 調用原始服務
	oldResp, err := a.originalDealer.DrawBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawBallResponse{
		GameData: ConvertGameDataFromOldPb(oldResp.GetGameData()),
	}

	return newResp, nil
}

// DrawExtraBall 處理抽取額外球的請求
func (a *DealerServiceAdapter) DrawExtraBall(ctx context.Context, req *newpb.DrawExtraBallRequest) (*newpb.DrawExtraBallResponse, error) {
	a.logger.Info("收到抽取額外球請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.DrawExtraBallRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
		Side:   ConvertExtraBallSideToOldPb(req.GetSide()),
	}

	// 調用原始服務
	oldResp, err := a.originalDealer.DrawExtraBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽取額外球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawExtraBallResponse{
		GameData: ConvertGameDataFromOldPb(oldResp.GetGameData()),
	}

	return newResp, nil
}

// DrawJackpotBall 處理抽取頭獎球的請求
func (a *DealerServiceAdapter) DrawJackpotBall(ctx context.Context, req *newpb.DrawJackpotBallRequest) (*newpb.DrawJackpotBallResponse, error) {
	a.logger.Info("收到抽取頭獎球請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.DrawJackpotBallRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
	}

	// 調用原始服務
	oldResp, err := a.originalDealer.DrawJackpotBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽取頭獎球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawJackpotBallResponse{
		GameData: ConvertGameDataFromOldPb(oldResp.GetGameData()),
	}

	return newResp, nil
}

// DrawLuckyBall 處理抽取幸運球的請求
func (a *DealerServiceAdapter) DrawLuckyBall(ctx context.Context, req *newpb.DrawLuckyBallRequest) (*newpb.DrawLuckyBallResponse, error) {
	a.logger.Info("收到抽取幸運球請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.DrawLuckyBallRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
		Count:  req.GetCount(),
	}

	// 調用原始服務
	oldResp, err := a.originalDealer.DrawLuckyBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽取幸運球失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawLuckyBallResponse{
		GameData: ConvertGameDataFromOldPb(oldResp.GetGameData()),
	}

	return newResp, nil
}

// CancelGame 處理取消遊戲的請求
func (a *DealerServiceAdapter) CancelGame(ctx context.Context, req *newpb.CancelGameRequest) (*newpb.CancelGameResponse, error) {
	a.logger.Info("收到取消遊戲請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.CancelGameRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
	}

	// 調用原始服務
	oldGameData, err := a.originalDealer.CancelGame(ctx, oldReq)
	if err != nil {
		a.logger.Error("取消遊戲失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.CancelGameResponse{
		GameData: ConvertGameDataFromOldPb(oldGameData),
	}

	return newResp, nil
}

// GetGameStatus 處理獲取遊戲狀態的請求
func (a *DealerServiceAdapter) GetGameStatus(ctx context.Context, req *newpb.GetGameStatusRequest) (*newpb.GetGameStatusResponse, error) {
	a.logger.Info("收到獲取遊戲狀態請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.GetGameStatusRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
	}

	// 調用原始服務
	oldResp, err := a.originalDealer.GetGameStatus(ctx, oldReq)
	if err != nil {
		a.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.GetGameStatusResponse{
		GameData: ConvertGameDataFromOldPb(oldResp.GetGameData()),
	}

	return newResp, nil
}

// StartJackpotRound 處理開始頭獎回合的請求
func (a *DealerServiceAdapter) StartJackpotRound(ctx context.Context, req *newpb.StartJackpotRoundRequest) (*newpb.StartJackpotRoundResponse, error) {
	a.logger.Info("收到開始頭獎回合請求 (新 API)")

	// 使用提供的 room_id 或預設值
	roomId := req.GetRoomId()
	if roomId == "" {
		roomId = "SG01" // 預設使用 SG01 作為房間 ID
	}

	// 轉換請求
	oldReq := &oldpb.StartJackpotRoundRequest{
		RoomId: roomId,
	}

	// 調用原始服務
	oldResp, err := a.originalDealer.StartJackpotRound(ctx, oldReq)
	if err != nil {
		a.logger.Error("開始頭獎回合失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.StartJackpotRoundResponse{
		GameData: ConvertGameDataFromOldPb(oldResp.GetGameData()),
	}

	return newResp, nil
}

// SubscribeGameEvents 處理訂閱遊戲事件的請求
func (a *DealerServiceAdapter) SubscribeGameEvents(req *newpb.SubscribeGameEventsRequest, stream newpb.DealerService_SubscribeGameEventsServer) error {
	a.logger.Info("收到訂閱遊戲事件請求 (新 API)")

	// 創建舊 API 的請求
	oldReq := &oldpb.SubscribeGameEventsRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
	}

	// 創建一個包裝器來處理舊 API 的串流
	streamWrapper := &gameEventStreamWrapper{
		ctx:       stream.Context(),
		newStream: stream,
		logger:    a.logger,
	}

	// 調用原始服務
	return a.originalDealer.SubscribeGameEvents(oldReq, streamWrapper)
}

// gameEventStreamWrapper 是一個包裝器，用於將舊 API 的事件流轉換為新 API 的事件流
type gameEventStreamWrapper struct {
	oldpb.DealerService_SubscribeGameEventsServer
	ctx       context.Context
	newStream newpb.DealerService_SubscribeGameEventsServer
	logger    *zap.Logger
}

// Context 返回上下文
func (w *gameEventStreamWrapper) Context() context.Context {
	return w.ctx
}

// Send 發送事件
func (w *gameEventStreamWrapper) Send(oldEvent *oldpb.GameEvent) error {
	// 轉換事件
	newEvent := ConvertGameEventFromOldPb(oldEvent)

	// 發送到新的流
	err := w.newStream.Send(newEvent)
	if err != nil {
		w.logger.Error("發送遊戲事件失敗", zap.Error(err))
	}

	return err
}

// 轉換函數

// ConvertGameDataFromOldPb 將舊版 GameData 轉換為新版 GameData
func ConvertGameDataFromOldPb(oldData *oldpb.GameData) *newpb.GameData {
	if oldData == nil {
		return nil
	}

	newData := &newpb.GameData{
		Id:          oldData.GetId(),
		RoomId:      oldData.GetRoomId(),
		Stage:       ConvertGameStageFromOldPb(oldData.GetStage()),
		Status:      ConvertGameStatusFromOldPb(oldData.GetStatus()),
		DrawnBalls:  ConvertBallsFromOldPb(oldData.GetDrawnBalls()),
		ExtraBalls:  make(map[string]*newpb.Ball),
		JackpotBall: ConvertBallFromOldPb(oldData.GetJackpotBall()),
		LuckyBalls:  ConvertBallsFromOldPb(oldData.GetLuckyBalls()),
		CreatedAt:   oldData.GetCreatedAt(),
		UpdatedAt:   oldData.GetUpdatedAt(),
		DealerId:    oldData.GetDealerId(),
	}

	// 轉換額外球 map
	for k, v := range oldData.GetExtraBalls() {
		newData.ExtraBalls[k] = ConvertBallFromOldPb(v)
	}

	return newData
}

// ConvertGameStageFromOldPb 將舊版 GameStage 轉換為新版 GameStage
func ConvertGameStageFromOldPb(oldStage oldpb.GameStage) commonpb.GameStage {
	switch oldStage {
	case oldpb.GameStage_GAME_STAGE_NOT_STARTED:
		return commonpb.GameStage_GAME_STAGE_NEW_ROUND
	case oldpb.GameStage_GAME_STAGE_BALL_DRAWING:
		return commonpb.GameStage_GAME_STAGE_DRAWING_START
	case oldpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START
	case oldpb.GameStage_GAME_STAGE_JACKPOT_ROUND:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START
	case oldpb.GameStage_GAME_STAGE_LUCKY_DRAW:
		return commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START
	case oldpb.GameStage_GAME_STAGE_COMPLETED:
		return commonpb.GameStage_GAME_STAGE_GAME_OVER
	default:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// ConvertGameStageToOldPb 將新版 GameStage 轉換為舊版 GameStage
func ConvertGameStageToOldPb(newStage commonpb.GameStage) oldpb.GameStage {
	switch newStage {
	case commonpb.GameStage_GAME_STAGE_NEW_ROUND:
		return oldpb.GameStage_GAME_STAGE_NOT_STARTED
	case commonpb.GameStage_GAME_STAGE_DRAWING_START:
		return oldpb.GameStage_GAME_STAGE_BALL_DRAWING
	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START:
		return oldpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING
	case commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START:
		return oldpb.GameStage_GAME_STAGE_JACKPOT_ROUND
	case commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START:
		return oldpb.GameStage_GAME_STAGE_LUCKY_DRAW
	case commonpb.GameStage_GAME_STAGE_GAME_OVER:
		return oldpb.GameStage_GAME_STAGE_COMPLETED
	default:
		return oldpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// ConvertGameStatusFromOldPb 將舊版 GameStatus 轉換為新版 GameStatus
func ConvertGameStatusFromOldPb(oldStatus oldpb.GameStatus) newpb.GameStatus {
	switch oldStatus {
	case oldpb.GameStatus_GAME_STATUS_NOT_STARTED:
		return newpb.GameStatus_GAME_STATUS_NOT_STARTED
	case oldpb.GameStatus_GAME_STATUS_RUNNING:
		return newpb.GameStatus_GAME_STATUS_RUNNING
	case oldpb.GameStatus_GAME_STATUS_COMPLETED:
		return newpb.GameStatus_GAME_STATUS_COMPLETED
	case oldpb.GameStatus_GAME_STATUS_CANCELLED:
		return newpb.GameStatus_GAME_STATUS_CANCELLED
	case oldpb.GameStatus_GAME_STATUS_ERROR:
		return newpb.GameStatus_GAME_STATUS_ERROR
	default:
		return newpb.GameStatus_GAME_STATUS_UNSPECIFIED
	}
}

// ConvertGameStatusToOldPb 將新版 GameStatus 轉換為舊版 GameStatus
func ConvertGameStatusToOldPb(newStatus newpb.GameStatus) oldpb.GameStatus {
	switch newStatus {
	case newpb.GameStatus_GAME_STATUS_NOT_STARTED:
		return oldpb.GameStatus_GAME_STATUS_NOT_STARTED
	case newpb.GameStatus_GAME_STATUS_RUNNING:
		return oldpb.GameStatus_GAME_STATUS_RUNNING
	case newpb.GameStatus_GAME_STATUS_COMPLETED:
		return oldpb.GameStatus_GAME_STATUS_COMPLETED
	case newpb.GameStatus_GAME_STATUS_CANCELLED:
		return oldpb.GameStatus_GAME_STATUS_CANCELLED
	case newpb.GameStatus_GAME_STATUS_ERROR:
		return oldpb.GameStatus_GAME_STATUS_ERROR
	default:
		return oldpb.GameStatus_GAME_STATUS_UNSPECIFIED
	}
}

// ConvertExtraBallSideFromOldPb 將舊版 ExtraBallSide 轉換為新版 ExtraBallSide
func ConvertExtraBallSideFromOldPb(oldSide oldpb.ExtraBallSide) commonpb.ExtraBallSide {
	switch oldSide {
	case oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// ConvertExtraBallSideToOldPb 將新版 ExtraBallSide 轉換為舊版 ExtraBallSide
func ConvertExtraBallSideToOldPb(newSide commonpb.ExtraBallSide) oldpb.ExtraBallSide {
	switch newSide {
	case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return oldpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// ConvertBallFromOldPb 將舊版 Ball 轉換為新版 Ball
func ConvertBallFromOldPb(oldBall *oldpb.Ball) *newpb.Ball {
	if oldBall == nil {
		return nil
	}

	return &newpb.Ball{
		Id:      oldBall.GetId(),
		Number:  oldBall.GetNumber(),
		Color:   oldBall.GetColor(),
		IsOdd:   oldBall.GetIsOdd(),
		IsSmall: oldBall.GetIsSmall(),
	}
}

// ConvertBallToOldPb 將新版 Ball 轉換為舊版 Ball
func ConvertBallToOldPb(newBall *newpb.Ball) *oldpb.Ball {
	if newBall == nil {
		return nil
	}

	return &oldpb.Ball{
		Id:      newBall.GetId(),
		Number:  newBall.GetNumber(),
		Color:   newBall.GetColor(),
		IsOdd:   newBall.GetIsOdd(),
		IsSmall: newBall.GetIsSmall(),
	}
}

// ConvertBallsFromOldPb 將舊版 Ball 列表轉換為新版 Ball 列表
func ConvertBallsFromOldPb(oldBalls []*oldpb.Ball) []*newpb.Ball {
	if oldBalls == nil {
		return nil
	}

	newBalls := make([]*newpb.Ball, 0, len(oldBalls))
	for _, oldBall := range oldBalls {
		newBalls = append(newBalls, ConvertBallFromOldPb(oldBall))
	}

	return newBalls
}

// ConvertBallsToOldPb 將新版 Ball 列表轉換為舊版 Ball 列表
func ConvertBallsToOldPb(newBalls []*newpb.Ball) []*oldpb.Ball {
	if newBalls == nil {
		return nil
	}

	oldBalls := make([]*oldpb.Ball, 0, len(newBalls))
	for _, newBall := range newBalls {
		oldBalls = append(oldBalls, ConvertBallToOldPb(newBall))
	}

	return oldBalls
}

// ConvertGameEventFromOldPb 將舊版 GameEvent 轉換為新版 GameEvent
func ConvertGameEventFromOldPb(oldEvent *oldpb.GameEvent) *newpb.GameEvent {
	if oldEvent == nil {
		return nil
	}

	newEvent := &newpb.GameEvent{
		Type:      ConvertGameEventTypeFromOldPb(oldEvent.GetType()),
		Timestamp: oldEvent.GetTimestamp(),
	}

	// 根據事件類型轉換事件數據
	switch oldEvent.GetType() {
	case oldpb.GameEventType_GAME_EVENT_TYPE_STAGE_CHANGED:
		if oldEvent.GetStageChanged() != nil {
			newEvent.EventData = &newpb.GameEvent_StageChanged{
				StageChanged: &newpb.StageChangedEvent{
					GameId:   oldEvent.GetStageChanged().GetGameId(),
					OldStage: ConvertGameStageFromOldPb(oldEvent.GetStageChanged().GetOldStage()),
					NewStage: ConvertGameStageFromOldPb(oldEvent.GetStageChanged().GetNewStage()),
				},
			}
		}
	case oldpb.GameEventType_GAME_EVENT_TYPE_BALL_DRAWN:
		if oldEvent.GetBallDrawn() != nil {
			newEvent.EventData = &newpb.GameEvent_BallDrawn{
				BallDrawn: &newpb.BallDrawnEvent{
					GameId:   oldEvent.GetBallDrawn().GetGameId(),
					Ball:     ConvertBallFromOldPb(oldEvent.GetBallDrawn().GetBall()),
					Position: oldEvent.GetBallDrawn().GetPosition(),
				},
			}
		}
	case oldpb.GameEventType_GAME_EVENT_TYPE_GAME_CREATED:
		if oldEvent.GetGameCreated() != nil {
			newEvent.EventData = &newpb.GameEvent_GameCreated{
				GameCreated: &newpb.GameCreatedEvent{
					GameData: ConvertGameDataFromOldPb(oldEvent.GetGameCreated().GetGameData()),
				},
			}
		}
	case oldpb.GameEventType_GAME_EVENT_TYPE_GAME_CANCELLED:
		if oldEvent.GetGameCancelled() != nil {
			newEvent.EventData = &newpb.GameEvent_GameCancelled{
				GameCancelled: &newpb.GameCancelledEvent{
					GameId: oldEvent.GetGameCancelled().GetGameId(),
					Reason: oldEvent.GetGameCancelled().GetReason(),
				},
			}
		}
	case oldpb.GameEventType_GAME_EVENT_TYPE_JACKPOT_BALL_DRAWN:
		if oldEvent.GetJackpotBallDrawn() != nil {
			newEvent.EventData = &newpb.GameEvent_JackpotBallDrawn{
				JackpotBallDrawn: &newpb.JackpotBallDrawnEvent{
					GameId: oldEvent.GetJackpotBallDrawn().GetGameId(),
					Ball:   ConvertBallFromOldPb(oldEvent.GetJackpotBallDrawn().GetBall()),
				},
			}
		}
	case oldpb.GameEventType_GAME_EVENT_TYPE_EXTRA_BALL_DRAWN:
		if oldEvent.GetExtraBallDrawn() != nil {
			newEvent.EventData = &newpb.GameEvent_ExtraBallDrawn{
				ExtraBallDrawn: &newpb.ExtraBallDrawnEvent{
					GameId: oldEvent.GetExtraBallDrawn().GetGameId(),
					Ball:   ConvertBallFromOldPb(oldEvent.GetExtraBallDrawn().GetBall()),
					Side:   ConvertExtraBallSideFromOldPb(oldEvent.GetExtraBallDrawn().GetSide()),
				},
			}
		}
	case oldpb.GameEventType_GAME_EVENT_TYPE_LUCKY_BALL_DRAWN:
		if oldEvent.GetLuckyBallDrawn() != nil {
			newEvent.EventData = &newpb.GameEvent_LuckyBallDrawn{
				LuckyBallDrawn: &newpb.LuckyBallDrawnEvent{
					GameId: oldEvent.GetLuckyBallDrawn().GetGameId(),
					Balls:  ConvertBallsFromOldPb(oldEvent.GetLuckyBallDrawn().GetBalls()),
				},
			}
		}
	case oldpb.GameEventType_GAME_EVENT_TYPE_GAME_DATA:
		if oldEvent.GetGameData() != nil {
			newEvent.EventData = &newpb.GameEvent_GameData{
				GameData: ConvertGameDataFromOldPb(oldEvent.GetGameData()),
			}
		}
	}

	return newEvent
}

// ConvertGameEventTypeFromOldPb 將舊版 GameEventType 轉換為新版 GameEventType
func ConvertGameEventTypeFromOldPb(oldType oldpb.GameEventType) commonpb.GameEventType {
	switch oldType {
	case oldpb.GameEventType_GAME_EVENT_TYPE_STAGE_CHANGED:
		return commonpb.GameEventType_GAME_EVENT_TYPE_STAGE_CHANGED
	case oldpb.GameEventType_GAME_EVENT_TYPE_BALL_DRAWN:
		return commonpb.GameEventType_GAME_EVENT_TYPE_BALL_DRAWN
	case oldpb.GameEventType_GAME_EVENT_TYPE_GAME_CREATED:
		return commonpb.GameEventType_GAME_EVENT_TYPE_GAME_CREATED
	case oldpb.GameEventType_GAME_EVENT_TYPE_GAME_CANCELLED:
		return commonpb.GameEventType_GAME_EVENT_TYPE_GAME_CANCELLED
	case oldpb.GameEventType_GAME_EVENT_TYPE_JACKPOT_BALL_DRAWN:
		return commonpb.GameEventType_GAME_EVENT_TYPE_JACKPOT_BALL_DRAWN
	case oldpb.GameEventType_GAME_EVENT_TYPE_EXTRA_BALL_DRAWN:
		return commonpb.GameEventType_GAME_EVENT_TYPE_EXTRA_BALL_DRAWN
	case oldpb.GameEventType_GAME_EVENT_TYPE_LUCKY_BALL_DRAWN:
		return commonpb.GameEventType_GAME_EVENT_TYPE_LUCKY_BALL_DRAWN
	case oldpb.GameEventType_GAME_EVENT_TYPE_GAME_DATA:
		return commonpb.GameEventType_GAME_EVENT_TYPE_GAME_DATA
	default:
		return commonpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	}
}
