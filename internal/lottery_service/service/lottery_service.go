package service

import (
	"g38_lottery_service/internal/lottery_service/mq"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LotteryService 代表開獎服務
type LotteryService struct {
	logger          *zap.Logger
	messageProducer *mq.MessageProducer // 添加 RocketMQ 生產者
}

// NewLotteryService 創建新的開獎服務
func NewLotteryService(
	logger *zap.Logger,
	messageProducer *mq.MessageProducer, // 注入 RocketMQ 生產者
) *LotteryService {
	return &LotteryService{
		logger:          logger.With(zap.String("component", "lottery_service")),
		messageProducer: messageProducer,
	}
}

// Start 啟動開獎服務
func (s *LotteryService) Start() {
	s.logger.Info("Lottery service started")
}

// SendLotteryStatus 發送開獎狀態更新（提供給其他服務調用）
func (s *LotteryService) SendLotteryStatus(gameID string, status string, details interface{}) error {
	s.logger.Info("Sending lottery status update",
		zap.String("gameID", gameID),
		zap.String("status", status))

	return s.messageProducer.SendLotteryStatus(gameID, status, details)
}

// SendLotteryResult 發送開獎結果（提供給其他服務調用）
func (s *LotteryService) SendLotteryResult(gameID string, result interface{}) error {
	s.logger.Info("Sending lottery result",
		zap.String("gameID", gameID),
		zap.Any("result", result))

	return s.messageProducer.SendLotteryResult(gameID, result)
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
