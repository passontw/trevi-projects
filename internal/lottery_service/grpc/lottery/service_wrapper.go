package lottery

import (
	"context"

	"go.uber.org/zap"

	newpb "g38_lottery_service/internal/generated/api/v1/lottery"
)

// LotteryServiceWrapper 是將新的 API 轉換到舊的 API 的包裝器
type LotteryServiceWrapper struct {
	logger                 *zap.Logger
	originalLotteryService *LotteryService
}

// NewLotteryServiceWrapper 創建一個新的包裝器
func NewLotteryServiceWrapper(
	logger *zap.Logger,
	lotteryService *LotteryService,
) *LotteryServiceWrapper {
	return &LotteryServiceWrapper{
		logger:                 logger.Named("lottery_service_wrapper"),
		originalLotteryService: lotteryService,
	}
}

// StartNewRound 處理開始新遊戲回合的請求
func (w *LotteryServiceWrapper) StartNewRound(ctx context.Context, req *newpb.StartNewRoundRequest) (*newpb.StartNewRoundResponse, error) {
	w.logger.Info("收到開始新遊戲回合請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.StartNewRound(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DrawBall 處理抽球請求
func (w *LotteryServiceWrapper) DrawBall(ctx context.Context, req *newpb.DrawBallRequest) (*newpb.DrawBallResponse, error) {
	w.logger.Info("收到抽球請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.DrawBall(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DrawExtraBall 處理抽取額外球的請求
func (w *LotteryServiceWrapper) DrawExtraBall(ctx context.Context, req *newpb.DrawExtraBallRequest) (*newpb.DrawExtraBallResponse, error) {
	w.logger.Info("收到抽取額外球請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.DrawExtraBall(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DrawJackpotBall 處理抽取頭獎球的請求
func (w *LotteryServiceWrapper) DrawJackpotBall(ctx context.Context, req *newpb.DrawJackpotBallRequest) (*newpb.DrawJackpotBallResponse, error) {
	w.logger.Info("收到抽取頭獎球請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.DrawJackpotBall(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DrawLuckyBall 處理抽取幸運球的請求
func (w *LotteryServiceWrapper) DrawLuckyBall(ctx context.Context, req *newpb.DrawLuckyBallRequest) (*newpb.DrawLuckyBallResponse, error) {
	w.logger.Info("收到抽取幸運球請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.DrawLuckyBall(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CancelGame 處理取消遊戲的請求
func (w *LotteryServiceWrapper) CancelGame(ctx context.Context, req *newpb.CancelGameRequest) (*newpb.GameData, error) {
	w.logger.Info("收到取消遊戲請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.CancelGame(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetGameStatus 處理獲取遊戲狀態的請求
func (w *LotteryServiceWrapper) GetGameStatus(ctx context.Context, req *newpb.GetGameStatusRequest) (*newpb.GetGameStatusResponse, error) {
	w.logger.Info("收到獲取遊戲狀態請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.GetGameStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// StartJackpotRound 處理開始頭獎回合的請求
func (w *LotteryServiceWrapper) StartJackpotRound(ctx context.Context, req *newpb.StartJackpotRoundRequest) (*newpb.StartJackpotRoundResponse, error) {
	w.logger.Info("收到開始頭獎回合請求")

	// 調用原始服務處理請求
	resp, err := w.originalLotteryService.StartJackpotRound(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SubscribeGameEvents 處理訂閱遊戲事件的請求
func (w *LotteryServiceWrapper) SubscribeGameEvents(req *newpb.SubscribeGameEventsRequest, stream newpb.LotteryService_SubscribeGameEventsServer) error {
	w.logger.Info("收到訂閱遊戲事件請求")

	// 調用原始服務處理請求
	return w.originalLotteryService.SubscribeGameEvents(req, stream)
}
