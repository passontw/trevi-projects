package service

import (
	"g38_lottery_service/internal/dealerWebsocket"
	"g38_lottery_service/internal/websocket"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LotteryService 代表開獎服務
type LotteryService struct {
	logger       *zap.Logger
	dealerServer *dealerWebsocket.DealerServer
}

// NewLotteryService 創建新的開獎服務
func NewLotteryService(
	logger *zap.Logger,
	dealerServer *dealerWebsocket.DealerServer,
) *LotteryService {
	return &LotteryService{
		logger:       logger.With(zap.String("component", "lottery_service")),
		dealerServer: dealerServer,
	}
}

// Start 啟動開獎服務
func (s *LotteryService) Start() {
	// 註冊荷官端訊息處理函數
	s.registerDealerHandlers()

	s.logger.Info("Lottery service started")
}

// 註冊荷官端訊息處理函數
func (s *LotteryService) registerDealerHandlers() {
	// 監聽荷官端的開獎事件
	dealerLotteryHandler := func(client *websocket.Client, message websocket.Message) error {
		// 從訊息中提取必要的信息
		gameID, ok := message.Payload["game_id"].(string)
		if !ok {
			s.logger.Error("Missing or invalid game_id in dealer draw_lottery message")
			return nil
		}

		result, ok := message.Payload["result"]
		if !ok {
			s.logger.Error("Missing result in dealer draw_lottery message")
			return nil
		}

		// 記錄開獎結果
		s.logger.Info("Received lottery result from dealer",
			zap.String("gameID", gameID),
			zap.Any("result", result))

		// 這裡不再轉發到玩家端，而是可以進行其他處理，如保存到數據庫等

		return nil
	}

	// 使用DealerServer提供的方法註冊外部處理函數
	s.dealerServer.RegisterExternalHandler("draw_lottery_external", dealerLotteryHandler)

	s.logger.Info("Registered dealer lottery result handler")
}

// Module 提供 FX 模塊
var Module = fx.Options(
	// 註冊服務
	fx.Provide(NewLotteryService),

	// 啟動服務
	fx.Invoke(func(service *LotteryService) {
		service.Start()
	}),
)
