package service

import (
	"g38_lottery_service/internal/dealerWebsocket"
	"g38_lottery_service/internal/playerWebsocket"
	"g38_lottery_service/internal/websocket"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LotteryService 代表開獎服務，連接荷官端和玩家端
type LotteryService struct {
	logger       *zap.Logger
	dealerServer *dealerWebsocket.DealerServer
	playerServer *playerWebsocket.PlayerServer
}

// NewLotteryService 創建新的開獎服務
func NewLotteryService(
	logger *zap.Logger,
	dealerServer *dealerWebsocket.DealerServer,
	playerServer *playerWebsocket.PlayerServer,
) *LotteryService {
	return &LotteryService{
		logger:       logger.With(zap.String("component", "lottery_service")),
		dealerServer: dealerServer,
		playerServer: playerServer,
	}
}

// Start 啟動開獎服務
func (s *LotteryService) Start() {
	// 註冊荷官端訊息處理函數，將開獎結果轉發到玩家端
	s.registerDealerHandlers()

	s.logger.Info("Lottery service started and connected dealer with player websockets")
}

// 註冊荷官端訊息處理函數
func (s *LotteryService) registerDealerHandlers() {
	// 監聽荷官端的開獎事件，並轉發到玩家端
	// 這裡我們使用一個自定義的事件處理函數

	// 建立一個在荷官端開獎後轉發到玩家端的處理函數
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
		s.logger.Info("Forwarding lottery result from dealer to players",
			zap.String("gameID", gameID),
			zap.Any("result", result))

		// 將開獎結果轉發到玩家端
		err := s.playerServer.BroadcastLotteryResult(gameID, result)
		if err != nil {
			s.logger.Error("Failed to broadcast lottery result to players",
				zap.Error(err),
				zap.String("gameID", gameID))
		}

		return nil
	}

	// 使用DealerServer提供的方法註冊外部處理函數
	s.dealerServer.RegisterExternalHandler("draw_lottery_external", dealerLotteryHandler)

	s.logger.Info("Registered dealer lottery result handler to forward results to players")
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
