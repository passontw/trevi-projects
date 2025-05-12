package dealer

import (
	"context"
	"math/rand"
	"time"

	newpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"
	oldpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"go.uber.org/zap"
)

// DealerServiceAdapter 是將舊的 API 轉換到新的 API 的適配器
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
	_, err := a.originalDealer.StartNewRound(ctx, oldReq)
	if err != nil {
		a.logger.Error("開始新遊戲回合失敗", zap.Error(err))
		return nil, err
	}

	// 獲取遊戲狀態以構建完整的 GameData
	statusReq := &oldpb.GetGameStatusRequest{
		RoomId: "SG01",
	}
	statusResp, err := a.originalDealer.GetGameStatus(ctx, statusReq)
	if err != nil {
		a.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.StartNewRoundResponse{
		GameData: convertGameDataFromOldPb(statusResp.GameData),
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
	_, err := a.originalDealer.DrawBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽球失敗", zap.Error(err))
		return nil, err
	}

	// 獲取遊戲狀態以構建完整的 GameData
	statusReq := &oldpb.GetGameStatusRequest{
		RoomId: "SG01",
	}
	statusResp, err := a.originalDealer.GetGameStatus(ctx, statusReq)
	if err != nil {
		a.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawBallResponse{
		GameData: convertGameDataFromOldPb(statusResp.GameData),
	}

	return newResp, nil
}

// DrawExtraBall 處理抽取額外球的請求
func (a *DealerServiceAdapter) DrawExtraBall(ctx context.Context, req *newpb.DrawExtraBallRequest) (*newpb.DrawExtraBallResponse, error) {
	a.logger.Info("收到抽取額外球請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.DrawExtraBallRequest{
		RoomId: "SG01", // 固定使用 SG01
	}

	// 從新 API 中獲取 Side 值，轉換為舊版的 ExtraBallSide
	if req.Side != commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED {
		oldSide := oldpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
		switch req.Side {
		case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
			oldSide = oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
		case commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
			oldSide = oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
		}
		a.logger.Info("設置額外球側", zap.Int32("side", int32(oldSide)))
	}

	// 調用原始服務
	_, err := a.originalDealer.DrawExtraBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽取額外球失敗", zap.Error(err))
		return nil, err
	}

	// 獲取遊戲狀態以構建完整的 GameData
	statusReq := &oldpb.GetGameStatusRequest{
		RoomId: "SG01",
	}
	statusResp, err := a.originalDealer.GetGameStatus(ctx, statusReq)
	if err != nil {
		a.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawExtraBallResponse{
		GameData: convertGameDataFromOldPb(statusResp.GameData),
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
	_, err := a.originalDealer.DrawJackpotBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽取頭獎球失敗", zap.Error(err))
		return nil, err
	}

	// 獲取遊戲狀態以構建完整的 GameData
	statusReq := &oldpb.GetGameStatusRequest{
		RoomId: "SG01",
	}
	statusResp, err := a.originalDealer.GetGameStatus(ctx, statusReq)
	if err != nil {
		a.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawJackpotBallResponse{
		GameData: convertGameDataFromOldPb(statusResp.GameData),
	}

	return newResp, nil
}

// DrawLuckyBall 處理抽取幸運球的請求
func (a *DealerServiceAdapter) DrawLuckyBall(ctx context.Context, req *newpb.DrawLuckyBallRequest) (*newpb.DrawLuckyBallResponse, error) {
	a.logger.Info("收到抽取幸運球請求 (新 API)")

	// 轉換請求
	oldReq := &oldpb.DrawLuckyBallRequest{
		RoomId: "SG01", // 固定使用 SG01 作為房間 ID
	}

	// 設置數量（如果有提供）
	if req.Count > 0 {
		a.logger.Info("設置幸運球數量", zap.Int32("count", req.Count))
	}

	// 調用原始服務
	_, err := a.originalDealer.DrawLuckyBall(ctx, oldReq)
	if err != nil {
		a.logger.Error("抽取幸運球失敗", zap.Error(err))
		return nil, err
	}

	// 獲取遊戲狀態以構建完整的 GameData
	statusReq := &oldpb.GetGameStatusRequest{
		RoomId: "SG01",
	}
	statusResp, err := a.originalDealer.GetGameStatus(ctx, statusReq)
	if err != nil {
		a.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.DrawLuckyBallResponse{
		GameData: convertGameDataFromOldPb(statusResp.GameData),
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
		GameData: convertGameDataFromOldPb(oldGameData),
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
		GameData: convertGameDataFromOldPb(oldResp.GameData),
	}

	return newResp, nil
}

// StartJackpotRound 處理開始頭獎回合的請求
func (a *DealerServiceAdapter) StartJackpotRound(ctx context.Context, req *newpb.StartJackpotRoundRequest) (*newpb.StartJackpotRoundResponse, error) {
	a.logger.Info("收到開始頭獎回合請求 (新 API)")

	// 獲取房間ID（如果提供）
	roomId := "SG01" // 默認使用 SG01
	if req.RoomId != "" {
		roomId = req.RoomId
		a.logger.Info("使用自定義房間ID", zap.String("roomId", roomId))
	}

	// 轉換請求 - 注意：舊版 API 的 StartJackpotRoundRequest 似乎沒有 RoomId 字段
	oldReq := &oldpb.StartJackpotRoundRequest{}

	// 調用原始服務
	_, err := a.originalDealer.StartJackpotRound(ctx, oldReq)
	if err != nil {
		a.logger.Error("開始頭獎回合失敗", zap.Error(err))
		return nil, err
	}

	// 獲取遊戲狀態以構建完整的 GameData
	statusReq := &oldpb.GetGameStatusRequest{
		RoomId: roomId,
	}
	statusResp, err := a.originalDealer.GetGameStatus(ctx, statusReq)
	if err != nil {
		a.logger.Error("獲取遊戲狀態失敗", zap.Error(err))
		return nil, err
	}

	// 轉換回應
	newResp := &newpb.StartJackpotRoundResponse{
		GameData: convertGameDataFromOldPb(statusResp.GameData),
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

func (w *gameEventStreamWrapper) Context() context.Context {
	return w.ctx
}

func (w *gameEventStreamWrapper) Send(oldEvent *oldpb.GameEvent) error {
	w.logger.Debug("轉換並發送遊戲事件")

	// 轉換事件
	newEvent := convertGameEventFromOldPb(oldEvent)

	// 發送轉換後的事件
	return w.newStream.Send(newEvent)
}

// convertGameDataFromOldPb 將舊版 GameData 轉換為新版 GameData
func convertGameDataFromOldPb(oldData *oldpb.GameData) *newpb.GameData {
	if oldData == nil {
		return nil
	}

	// 建立遊戲數據
	newData := &newpb.GameData{
		Id:        oldData.GameId,
		RoomId:    "SG01", // 使用默認房間ID
		Stage:     convertGameStageFromOldPb(oldData.CurrentStage),
		Status:    newpb.GameStatus_GAME_STATUS_RUNNING, // 根據階段設置狀態
		DealerId:  "system",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 設置創建和更新時間
	if oldData.StartTime != nil {
		newData.CreatedAt = oldData.StartTime.AsTime().Unix()
	}

	if oldData.LastUpdateTime != nil {
		newData.UpdatedAt = oldData.LastUpdateTime.AsTime().Unix()
	}

	// 根據階段更精確地設置狀態
	switch oldData.CurrentStage {
	case oldpb.GameStage_GAME_STAGE_PREPARATION:
		newData.Status = newpb.GameStatus_GAME_STATUS_NOT_STARTED
	case oldpb.GameStage_GAME_STAGE_GAME_OVER:
		newData.Status = newpb.GameStatus_GAME_STATUS_COMPLETED
	}

	// 處理已抽出的常規球
	if len(oldData.RegularBalls) > 0 {
		newData.DrawnBalls = make([]*newpb.Ball, 0, len(oldData.RegularBalls))
		for _, ball := range oldData.RegularBalls {
			newBall := convertBallFromOldPb(ball)
			if newBall != nil {
				newData.DrawnBalls = append(newData.DrawnBalls, newBall)
			}
		}
	}

	// 處理額外球
	newData.ExtraBalls = make(map[string]*newpb.Ball)
	if len(oldData.ExtraBalls) > 0 {
		// 將第一個額外球設定為左側額外球
		newData.ExtraBalls["left"] = convertBallFromOldPb(oldData.ExtraBalls[0])
	}

	// 處理頭獎球
	if len(oldData.JackpotBalls) > 0 {
		newData.JackpotBall = convertBallFromOldPb(oldData.JackpotBalls[0])
	}

	// 處理幸運球
	if len(oldData.LuckyBalls) > 0 {
		newData.LuckyBalls = make([]*newpb.Ball, 0, len(oldData.LuckyBalls))
		for _, ball := range oldData.LuckyBalls {
			newBall := convertBallFromOldPb(ball)
			if newBall != nil {
				newData.LuckyBalls = append(newData.LuckyBalls, newBall)
			}
		}
	}

	// 設置其他字段（hasJackpot 不在新的 API 中，所以跳過）

	return newData
}

// convertGameStageFromOldPb 將舊版 GameStage 轉換為新版 GameStage
func convertGameStageFromOldPb(oldStage oldpb.GameStage) commonpb.GameStage {
	switch oldStage {
	case oldpb.GameStage_GAME_STAGE_UNSPECIFIED:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	case oldpb.GameStage_GAME_STAGE_PREPARATION:
		return commonpb.GameStage_GAME_STAGE_PREPARATION
	case oldpb.GameStage_GAME_STAGE_NEW_ROUND:
		return commonpb.GameStage_GAME_STAGE_NEW_ROUND
	case oldpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case oldpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE:
		return commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case oldpb.GameStage_GAME_STAGE_DRAWING_START:
		return commonpb.GameStage_GAME_STAGE_DRAWING_START
	case oldpb.GameStage_GAME_STAGE_DRAWING_CLOSE:
		return commonpb.GameStage_GAME_STAGE_DRAWING_CLOSE
	case oldpb.GameStage_GAME_STAGE_EXTRA_BALL_PREPARE:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_PREPARE
	case oldpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_START:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_START
	case oldpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED
	case oldpb.GameStage_GAME_STAGE_EXTRA_BALL_WAIT_CLAIM:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_WAIT_CLAIM
	case oldpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START
	case oldpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_CLOSE:
		return commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_CLOSE
	case oldpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		return commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	case oldpb.GameStage_GAME_STAGE_JACKPOT_START:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_START
	case oldpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START
	case oldpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_CLOSED:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_CLOSED
	case oldpb.GameStage_GAME_STAGE_JACKPOT_SETTLEMENT:
		return commonpb.GameStage_GAME_STAGE_JACKPOT_SETTLEMENT
	case oldpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START:
		return commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START
	case oldpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_CLOSED:
		return commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_CLOSED
	case oldpb.GameStage_GAME_STAGE_GAME_OVER:
		return commonpb.GameStage_GAME_STAGE_GAME_OVER
	default:
		return commonpb.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// convertExtraBallSideFromOldPb 將舊版 ExtraBallSide 轉換為新版 ExtraBallSide
func convertExtraBallSideFromOldPb(oldSide oldpb.ExtraBallSide) commonpb.ExtraBallSide {
	switch oldSide {
	case oldpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	case oldpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	case oldpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	default:
		return commonpb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED
	}
}

// convertBallFromOldPb 將舊版Ball轉換為新版Ball
func convertBallFromOldPb(oldBall *oldpb.Ball) *newpb.Ball {
	if oldBall == nil {
		return nil
	}

	// 創建新Ball對象
	newBall := &newpb.Ball{
		Id:      generateRandomString(8),
		Number:  oldBall.Number,
		Color:   getColorFromNumber(oldBall.Number),
		IsOdd:   oldBall.Number%2 != 0,
		IsSmall: oldBall.Number <= 40, // 假設1-40為小，41-80為大
	}

	return newBall
}

// getColorFromNumber 根據球號獲取顏色
func getColorFromNumber(number int32) string {
	// 根據號碼確定顏色（簡單實現，可根據實際規則調整）
	switch {
	case number >= 1 && number <= 10:
		return "red"
	case number >= 11 && number <= 20:
		return "green"
	case number >= 21 && number <= 30:
		return "blue"
	case number >= 31 && number <= 40:
		return "yellow"
	case number >= 41 && number <= 50:
		return "purple"
	case number >= 51 && number <= 60:
		return "orange"
	case number >= 61 && number <= 70:
		return "pink"
	case number >= 71 && number <= 80:
		return "cyan"
	default:
		return "white"
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

// convertGameEventFromOldPb 將舊版GameEvent轉換為新版GameEvent
func convertGameEventFromOldPb(oldEvent *oldpb.GameEvent) *newpb.GameEvent {
	if oldEvent == nil {
		return nil
	}

	// 建立基本的GameEvent結構
	newEvent := &newpb.GameEvent{
		Type:      convertGameEventTypeToCommonPb(oldEvent.EventType),
		Timestamp: time.Now().Unix(),
	}

	// 設置事件相關數據
	if oldEvent.GameId != "" {
		switch oldEvent.EventType {
		case oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT:
			if heartbeat, ok := oldEvent.EventData.(*oldpb.GameEvent_Heartbeat); ok && heartbeat.Heartbeat != nil {
				// 處理心跳事件
				newEvent.EventData = &newpb.GameEvent_GameData{
					GameData: &newpb.GameData{
						Id:     oldEvent.GameId,
						RoomId: "SG01", // 預設房間ID
						Status: newpb.GameStatus_GAME_STATUS_RUNNING,
					},
				}
			}
		case oldpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION:
			if notification, ok := oldEvent.EventData.(*oldpb.GameEvent_Notification); ok && notification.Notification != nil {
				// 處理通知事件
				gameData := convertGameDataFromOldPb(notification.Notification.GameData)
				if gameData == nil {
					gameData = &newpb.GameData{
						Id:     oldEvent.GameId,
						RoomId: "SG01", // 預設房間ID
					}
				}

				newEvent.EventData = &newpb.GameEvent_GameData{
					GameData: gameData,
				}
			}
		}
	}

	return newEvent
}

// convertGameEventTypeToCommonPb 將舊版GameEventType轉換為新版common.GameEventType
func convertGameEventTypeToCommonPb(oldType oldpb.GameEventType) commonpb.GameEventType {
	switch oldType {
	case oldpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT:
		return commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT
	case oldpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION:
		return commonpb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION
	default:
		return commonpb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	}
}
