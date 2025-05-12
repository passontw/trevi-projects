package dealer

import (
	"context"

	"go.uber.org/zap"

	"g38_lottery_service/internal/lottery_service/gameflow"
	oldpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"
)

// DealerServiceWrapper 是將新的 API 轉換到舊的 API 的包裝器
type DealerServiceWrapper struct {
	logger         *zap.Logger
	gameManager    *gameflow.GameManager
	originalDealer *DealerService
	roomID         string // 硬編碼的房間 ID
}

// NewDealerServiceWrapper 創建一個新的包裝器
func NewDealerServiceWrapper(
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
) *DealerServiceWrapper {
	return &DealerServiceWrapper{
		logger:         logger.Named("dealer_service_wrapper"),
		gameManager:    gameManager,
		originalDealer: NewDealerService(logger, gameManager),
		roomID:         "SG01", // 硬編碼房間 ID
	}
}

// StartNewRound 處理開始新遊戲回合的請求
func (w *DealerServiceWrapper) StartNewRound(ctx context.Context, req *oldpb.StartNewRoundRequest) (*oldpb.StartNewRoundResponse, error) {
	w.logger.Info("收到開始新遊戲回合請求")
	return w.originalDealer.StartNewRound(ctx, req)
}

// DrawBall 處理抽球請求
func (w *DealerServiceWrapper) DrawBall(ctx context.Context, req *oldpb.DrawBallRequest) (*oldpb.DrawBallResponse, error) {
	w.logger.Info("收到抽球請求")
	return w.originalDealer.DrawBall(ctx, req)
}

// DrawExtraBall 處理抽取額外球的請求
func (w *DealerServiceWrapper) DrawExtraBall(ctx context.Context, req *oldpb.DrawExtraBallRequest) (*oldpb.DrawExtraBallResponse, error) {
	w.logger.Info("收到抽取額外球請求")
	return w.originalDealer.DrawExtraBall(ctx, req)
}

// DrawJackpotBall 處理抽取頭獎球的請求
func (w *DealerServiceWrapper) DrawJackpotBall(ctx context.Context, req *oldpb.DrawJackpotBallRequest) (*oldpb.DrawJackpotBallResponse, error) {
	w.logger.Info("收到抽取頭獎球請求")
	return w.originalDealer.DrawJackpotBall(ctx, req)
}

// DrawLuckyBall 處理抽取幸運球的請求
func (w *DealerServiceWrapper) DrawLuckyBall(ctx context.Context, req *oldpb.DrawLuckyBallRequest) (*oldpb.DrawLuckyBallResponse, error) {
	w.logger.Info("收到抽取幸運球請求")
	return w.originalDealer.DrawLuckyBall(ctx, req)
}

// CancelGame 處理取消遊戲的請求
func (w *DealerServiceWrapper) CancelGame(ctx context.Context, req *oldpb.CancelGameRequest) (*oldpb.GameData, error) {
	w.logger.Info("收到取消遊戲請求")
	return w.originalDealer.CancelGame(ctx, req)
}

// GetGameStatus 處理獲取遊戲狀態的請求
func (w *DealerServiceWrapper) GetGameStatus(ctx context.Context, req *oldpb.GetGameStatusRequest) (*oldpb.GetGameStatusResponse, error) {
	w.logger.Info("收到獲取遊戲狀態請求")
	return w.originalDealer.GetGameStatus(ctx, req)
}

// StartJackpotRound 處理開始頭獎回合的請求
func (w *DealerServiceWrapper) StartJackpotRound(ctx context.Context, req *oldpb.StartJackpotRoundRequest) (*oldpb.StartJackpotRoundResponse, error) {
	w.logger.Info("收到開始頭獎回合請求")
	return w.originalDealer.StartJackpotRound(ctx, req)
}

// SubscribeGameEvents 處理訂閱遊戲事件的請求
func (w *DealerServiceWrapper) SubscribeGameEvents(req *oldpb.SubscribeGameEventsRequest, stream oldpb.DealerService_SubscribeGameEventsServer) error {
	w.logger.Info("收到訂閱遊戲事件請求")
	return w.originalDealer.SubscribeGameEvents(req, stream)
}
