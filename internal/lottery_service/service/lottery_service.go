package service

import (
	"g38_lottery_service/internal/lottery_service/dealerWebsocket"
	"g38_lottery_service/internal/lottery_service/mq"
	"g38_lottery_service/internal/lottery_service/websocket"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LotteryService 代表開獎服務
type LotteryService struct {
	logger          *zap.Logger
	dealerServer    *dealerWebsocket.DealerServer
	messageProducer *mq.MessageProducer // 添加 RocketMQ 生產者
}

// NewLotteryService 創建新的開獎服務
func NewLotteryService(
	logger *zap.Logger,
	dealerServer *dealerWebsocket.DealerServer,
	messageProducer *mq.MessageProducer, // 注入 RocketMQ 生產者
) *LotteryService {
	return &LotteryService{
		logger:          logger.With(zap.String("component", "lottery_service")),
		dealerServer:    dealerServer,
		messageProducer: messageProducer,
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

		// 通過 RocketMQ 將開獎結果發送到遊戲端
		err := s.messageProducer.SendLotteryResult(gameID, result)
		if err != nil {
			s.logger.Error("Failed to send lottery result to game service",
				zap.String("gameID", gameID),
				zap.Error(err))
		} else {
			s.logger.Info("Lottery result sent to game service successfully",
				zap.String("gameID", gameID))
		}

		return nil
	}

	// 使用DealerServer提供的方法註冊外部處理函數
	s.dealerServer.RegisterExternalHandler("draw_lottery_external", dealerLotteryHandler)

	s.logger.Info("Registered dealer lottery result handler")
}

// SendLotteryStatus 發送開獎狀態更新（提供給其他服務調用）
func (s *LotteryService) SendLotteryStatus(gameID string, status string, details interface{}) error {
	s.logger.Info("Sending lottery status update",
		zap.String("gameID", gameID),
		zap.String("status", status))

	return s.messageProducer.SendLotteryStatus(gameID, status, details)
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
