package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"g38_lottery_service/internal/lottery_service/config"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 定義常用的主題
const (
	// LotteryResultTopic 開獎結果主題
	LotteryResultTopic = "lottery-result-topic"
	// LotteryStatusTopic 開獎狀態主題
	LotteryStatusTopic = "lottery-status-topic"
)

// MessageProducer 封裝 RocketMQ 生產者功能
type MessageProducer struct {
	producer rocketmq.Producer
	logger   *zap.Logger
	config   *config.AppConfig
}

// NewMessageProducer 創建新的消息生產者
func NewMessageProducer(logger *zap.Logger, config *config.AppConfig) (*MessageProducer, error) {
	// 日誌標記
	l := logger.With(zap.String("component", "rocketmq_producer"))

	// 從 Nacos 驗證並檢查配置
	if !config.RocketMQ.Enabled {
		l.Error("RocketMQ 未啟用，但 RocketMQ 連線是必須的")
		return nil, fmt.Errorf("RocketMQ 連線是必須的，請在 Nacos 配置中啟用 RocketMQ")
	}

	// 驗證 NameServers 配置
	if len(config.RocketMQ.NameServers) == 0 {
		l.Error("Nacos 配置中 RocketMQ NameServers 為空")
		return nil, fmt.Errorf("Nacos 配置中 RocketMQ NameServers 為空，請在 Nacos 配置中設置有效的 NameServers")
	}

	// 驗證 ProducerGroup 配置
	if config.RocketMQ.ProducerGroup == "" {
		l.Error("Nacos 配置中 RocketMQ ProducerGroup 為空")
		return nil, fmt.Errorf("Nacos 配置中 RocketMQ ProducerGroup 為空，請在 Nacos 配置中設置有效的 ProducerGroup")
	}

	// 日誌輸出當前配置信息
	l.Info("使用 Nacos 配置初始化 RocketMQ",
		zap.Strings("nameServers", config.RocketMQ.NameServers),
		zap.String("producerGroup", config.RocketMQ.ProducerGroup))

	// 創建生產者選項
	opts := []producer.Option{
		producer.WithNameServer(config.RocketMQ.NameServers),
		producer.WithGroupName(config.RocketMQ.ProducerGroup),
		producer.WithRetry(2),
		producer.WithSendMsgTimeout(time.Second * 3),
	}

	// 如果配置了驗證信息，添加相應選項
	if config.RocketMQ.AccessKey != "" && config.RocketMQ.SecretKey != "" {
		opts = append(opts, producer.WithCredentials(primitive.Credentials{
			AccessKey: config.RocketMQ.AccessKey,
			SecretKey: config.RocketMQ.SecretKey,
		}))
	}

	// 創建生產者實例
	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		l.Error("創建 RocketMQ 生產者失敗", zap.Error(err),
			zap.Strings("nameServers", config.RocketMQ.NameServers),
			zap.String("producerGroup", config.RocketMQ.ProducerGroup))
		return nil, fmt.Errorf("創建 RocketMQ 生產者失敗: %w，請檢查 Nacos 中的 RocketMQ 配置", err)
	}

	// 啟動生產者
	if err := p.Start(); err != nil {
		l.Error("啟動 RocketMQ 生產者失敗", zap.Error(err),
			zap.Strings("nameServers", config.RocketMQ.NameServers),
			zap.String("producerGroup", config.RocketMQ.ProducerGroup))
		return nil, fmt.Errorf("啟動 RocketMQ 生產者失敗: %w，請檢查 RocketMQ 服務是否正常運行", err)
	}

	l.Info("RocketMQ 生產者啟動成功")

	// 返回封裝的生產者
	return &MessageProducer{
		producer: p,
		logger:   l,
		config:   config,
	}, nil
}

// Stop 停止生產者
func (p *MessageProducer) Stop() {
	if p == nil {
		return
	}

	if p.producer == nil {
		p.logger.Debug("RocketMQ producer is not initialized, no need to stop")
		return
	}

	err := p.producer.Shutdown()
	if err != nil {
		p.logger.Error("Error shutting down RocketMQ producer", zap.Error(err))
	} else {
		p.logger.Info("RocketMQ producer shutdown successfully")
	}
}

// SendLotteryResult 發送開獎結果
func (p *MessageProducer) SendLotteryResult(gameID string, result interface{}) error {
	return p.sendMessage(LotteryResultTopic, map[string]interface{}{
		"game_id":      gameID,
		"result":       result,
		"timestamp":    time.Now().Unix(),
		"message_type": "lottery_result",
	})
}

// SendLotteryStatus 發送開獎狀態更新
func (p *MessageProducer) SendLotteryStatus(gameID string, status string, details interface{}) error {
	return p.sendMessage(LotteryStatusTopic, map[string]interface{}{
		"game_id":      gameID,
		"status":       status,
		"details":      details,
		"timestamp":    time.Now().Unix(),
		"message_type": "lottery_status",
	})
}

// SendMessage 發送自定義消息
func (p *MessageProducer) SendMessage(topic string, payload map[string]interface{}) error {
	return p.sendMessage(topic, payload)
}

// sendMessage 內部方法，發送消息到指定主題
func (p *MessageProducer) sendMessage(topic string, payload map[string]interface{}) error {
	// 序列化消息體
	jsonData, err := json.Marshal(payload)
	if err != nil {
		p.logger.Error("Failed to marshal message payload",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("序列化消息失敗: %w", err)
	}

	// 創建消息
	msg := primitive.NewMessage(topic, jsonData)

	// 設置消息標簽和鍵
	messageType, ok := payload["message_type"].(string)
	if ok && messageType != "" {
		msg.WithTag(messageType)
	}

	gameID, ok := payload["game_id"].(string)
	if ok && gameID != "" {
		msg.WithKeys([]string{gameID})
	}

	// 發送消息
	p.logger.Debug("Sending message to RocketMQ",
		zap.String("topic", topic),
		zap.Any("payload", payload))

	// 同步發送
	res, err := p.producer.SendSync(context.Background(), msg)
	if err != nil {
		p.logger.Error("Failed to send message",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("發送消息失敗: %w", err)
	}

	p.logger.Info("Message sent successfully",
		zap.String("topic", topic),
		zap.String("messageID", res.MsgID),
		zap.String("queue", res.MessageQueue.String()))

	return nil
}

// StructToMap 將 struct 轉為 map[string]interface{}
func StructToMap(obj interface{}) (map[string]interface{}, error) {
	var m map[string]interface{}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &m)
	return m, err
}

// SendGameSnapshot 發送遊戲狀態快照到 game_events topic
func (p *MessageProducer) SendGameSnapshot(gameID string, snapshot map[string]interface{}) error {
	return p.sendMessage("game_events", map[string]interface{}{
		"game_id":      gameID,
		"message_type": "stage_change",
		"snapshot":     snapshot,
		"timestamp":    time.Now().Unix(),
	})
}

// IsEnabled 檢查生產者是否啟用
func (p *MessageProducer) IsEnabled() bool {
	if p == nil {
		return false
	}
	return p.producer != nil
}

// Module 提供 FX 模塊
var Module = fx.Options(
	// 註冊 MessageProducer
	fx.Provide(
		func(logger *zap.Logger, config *config.AppConfig) (*MessageProducer, error) {
			// 檢查是否啟用了 RocketMQ
			l := logger.With(zap.String("component", "rocketmq_module"))

			// 從 Nacos 檢查配置
			if !config.RocketMQ.Enabled {
				l.Error("Nacos 配置中 RocketMQ 未啟用，但 RocketMQ 連線是必須的")
				return nil, fmt.Errorf("RocketMQ 連線是必須的，請在 Nacos 配置中啟用 RocketMQ")
			}

			// 檢查 NameServers 配置
			if len(config.RocketMQ.NameServers) == 0 {
				l.Error("Nacos 配置中 RocketMQ NameServers 為空")
				return nil, fmt.Errorf("Nacos 配置中 RocketMQ NameServers 為空，請在 Nacos 配置中設置有效的 NameServers")
			}

			// 直接嘗試創建 RocketMQ 生產者
			producer, err := NewMessageProducer(logger, config)
			if err != nil {
				l.Error("初始化 RocketMQ 生產者失敗", zap.Error(err))
				return nil, fmt.Errorf("初始化 RocketMQ 生產者失敗: %w", err)
			}

			l.Info("RocketMQ 生產者初始化成功")
			return producer, nil
		},
	),

	// 確保應用關閉時停止生產者
	fx.Invoke(func(lc fx.Lifecycle, producer *MessageProducer) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				producer.Stop()
				return nil
			},
		})
	}),
)

// contains 檢查字符串切片是否包含特定字符串
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
