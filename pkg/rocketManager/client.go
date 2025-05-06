package rocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"g38_lottery_service/internal/lottery_service/config"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

// 常量定義
const (
	// 主題定義
	TopicGameStatus = "bingo.game.status" // 遊戲狀態更新
	TopicGameDraw   = "bingo.game.draw"   // 抽球結果通知
	TopicJPWin      = "bingo.jp.win"      // JP中獎通知
	TopicGamePrefix = "bingo.game."       // 特定遊戲前綴，後跟遊戲ID

	// 標籤定義
	TagStageChange   = "STAGE_CHANGED"  // 階段變更事件
	TagBallDrawn     = "BALL_DRAWN"     // 抽球事件
	TagGameCreated   = "GAME_CREATED"   // 遊戲創建事件
	TagGameCancelled = "GAME_CANCELLED" // 遊戲取消事件
	TagGameFinished  = "GAME_FINISHED"  // 遊戲結束事件
)

// RocketConfig 存儲 RocketMQ 連接的配置項
type RocketConfig struct {
	NameServers   []string // NameServer地址列表
	AccessKey     string   // 身份驗證AccessKey
	SecretKey     string   // 身份驗證SecretKey
	ProducerGroup string   // 生產者組名
	ConsumerGroup string   // 消費者組名
}

// 消息結構
type Message struct {
	Type      string      `json:"type"`      // 消息類型
	Stage     string      `json:"stage"`     // 遊戲階段
	Event     string      `json:"event"`     // 事件類型
	GameID    string      `json:"game_id"`   // 遊戲ID
	Data      interface{} `json:"data"`      // 數據
	Timestamp int64       `json:"timestamp"` // 時間戳
}

// MessageHandler 處理接收到的消息
type MessageHandler func(ctx context.Context, msg *primitive.MessageExt) error

// RocketManager 提供 RocketMQ 操作的介面
type RocketManager interface {
	// 消息生產
	SendMessage(ctx context.Context, topic, tag string, msg interface{}, keys ...string) error
	SendMessageToGame(ctx context.Context, gameID, event string, data interface{}) error
	SendGameStatus(ctx context.Context, gameID, stage, event string, data interface{}) error
	BroadcastBallDrawn(ctx context.Context, gameID string, ballInfo interface{}) error

	// 消息訂閱
	Subscribe(topic string, tag string, handler MessageHandler) error
	SubscribeGame(gameID string, handler MessageHandler) error

	// 連接管理
	Start(ctx context.Context) error
	Shutdown() error
}

// rocketManagerImpl 是 RocketManager 介面的實作
type rocketManagerImpl struct {
	config        *RocketConfig
	producer      rocketmq.Producer
	consumer      rocketmq.PushConsumer
	logger        *logrus.Logger
	subscriptions map[string]MessageHandler // 保存訂閱關係
}

// ProvideRocketConfig 從配置中提取RocketMQ配置
func ProvideRocketConfig(cfg *config.Config) *RocketConfig {
	return &RocketConfig{
		NameServers:   cfg.RocketMQ.NameServers,
		AccessKey:     cfg.RocketMQ.AccessKey,
		SecretKey:     cfg.RocketMQ.SecretKey,
		ProducerGroup: cfg.RocketMQ.ProducerGroup,
		ConsumerGroup: cfg.RocketMQ.ConsumerGroup,
	}
}

// ProvideRocketManager 提供 RocketManager 實例，用於 fx
func ProvideRocketManager(lc fx.Lifecycle, config *RocketConfig, logger *logrus.Logger) (RocketManager, error) {
	// 創建生產者
	p, err := rocketmq.NewProducer(
		producer.WithNameServer(config.NameServers),
		producer.WithGroupName(config.ProducerGroup),
		producer.WithCredentials(primitive.Credentials{
			AccessKey: config.AccessKey,
			SecretKey: config.SecretKey,
		}),
		producer.WithRetry(2),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	// 創建消費者
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer(config.NameServers),
		consumer.WithGroupName(config.ConsumerGroup),
		consumer.WithCredentials(primitive.Credentials{
			AccessKey: config.AccessKey,
			SecretKey: config.SecretKey,
		}),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	manager := &rocketManagerImpl{
		config:        config,
		producer:      p,
		consumer:      c,
		logger:        logger,
		subscriptions: make(map[string]MessageHandler),
	}

	// 使用fx生命週期鉤子管理啟動和關閉
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := manager.producer.Start(); err != nil {
				return fmt.Errorf("failed to start producer: %w", err)
			}
			logger.Info("RocketMQ producer started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down RocketMQ producer...")
			if err := manager.producer.Shutdown(); err != nil {
				logger.Errorf("Error shutting down producer: %v", err)
				return err
			}

			logger.Info("Shutting down RocketMQ consumer...")
			if err := manager.consumer.Shutdown(); err != nil {
				logger.Errorf("Error shutting down consumer: %v", err)
				return err
			}

			return nil
		},
	})

	return manager, nil
}

// 創建fx模組，包含所有RocketMQ相關組件
var Module = fx.Module("rocketmq",
	fx.Provide(
		ProvideRocketConfig,
		ProvideRocketManager,
	),
)

// 實作 RocketManager 介面

// SendMessage 發送消息到指定主題
func (r *rocketManagerImpl) SendMessage(ctx context.Context, topic, tag string, msg interface{}, keys ...string) error {
	// 將消息轉換為JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 創建消息
	message := &primitive.Message{
		Topic: topic,
		Body:  data,
	}

	// 設置標籤
	if tag != "" {
		message.WithTag(tag)
	}

	// 設置Keys (用於消息查詢和去重)
	if len(keys) > 0 {
		message.WithKeys(keys)
	}

	// 發送消息
	res, err := r.producer.SendSync(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"topic":     topic,
		"tag":       tag,
		"messageID": res.MsgID,
		"offset":    res.QueueOffset,
	}).Debug("Message sent successfully")

	return nil
}

// SendMessageToGame 發送消息到特定遊戲的主題
func (r *rocketManagerImpl) SendMessageToGame(ctx context.Context, gameID, event string, data interface{}) error {
	// 構建消息
	msg := Message{
		Type:      "event",
		Event:     event,
		GameID:    gameID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	// 發送到特定遊戲主題
	topic := TopicGamePrefix + gameID
	return r.SendMessage(ctx, topic, event, msg, gameID)
}

// SendGameStatus 發送遊戲狀態更新
func (r *rocketManagerImpl) SendGameStatus(ctx context.Context, gameID, stage, event string, data interface{}) error {
	// 構建消息
	msg := Message{
		Type:      "event",
		Stage:     stage,
		Event:     event,
		GameID:    gameID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	// 發送到遊戲狀態主題
	if err := r.SendMessage(ctx, TopicGameStatus, event, msg, gameID); err != nil {
		return err
	}

	// 同時發送到特定遊戲主題
	return r.SendMessage(ctx, TopicGamePrefix+gameID, event, msg, gameID)
}

// BroadcastBallDrawn 廣播球抽取事件
func (r *rocketManagerImpl) BroadcastBallDrawn(ctx context.Context, gameID string, ballInfo interface{}) error {
	// 構建消息
	msg := Message{
		Type:      "event",
		Event:     TagBallDrawn,
		GameID:    gameID,
		Data:      ballInfo,
		Timestamp: time.Now().Unix(),
	}

	// 發送到抽球主題
	if err := r.SendMessage(ctx, TopicGameDraw, TagBallDrawn, msg, gameID); err != nil {
		return err
	}

	// 同時發送到特定遊戲主題
	return r.SendMessage(ctx, TopicGamePrefix+gameID, TagBallDrawn, msg, gameID)
}

// Subscribe 訂閱指定主題和標籤的消息
func (r *rocketManagerImpl) Subscribe(topic string, tag string, handler MessageHandler) error {
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: tag,
	}

	// 如果標籤為空，訂閱所有標籤
	if tag == "" {
		selector = consumer.MessageSelector{}
	}

	// 保存訂閱關係，用於日誌輸出
	subKey := fmt.Sprintf("%s:%s", topic, tag)
	r.subscriptions[subKey] = handler

	err := r.consumer.Subscribe(topic, selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for i := range msgs {
			r.logger.WithFields(logrus.Fields{
				"topic":     msgs[i].Topic,
				"tag":       msgs[i].GetTags(),
				"messageID": msgs[i].MsgId,
				"keys":      msgs[i].GetKeys(),
			}).Debug("Received message")

			err := handler(ctx, msgs[i])
			if err != nil {
				r.logger.WithError(err).Error("Failed to process message")
				// 可以根據錯誤類型決定是否重試
				return consumer.ConsumeRetryLater, err
			}
		}
		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to topic %s with tag %s: %w", topic, tag, err)
	}

	r.logger.WithFields(logrus.Fields{
		"topic": topic,
		"tag":   tag,
	}).Info("Successfully subscribed to topic")
	return nil
}

// SubscribeGame 訂閱特定遊戲的所有事件
func (r *rocketManagerImpl) SubscribeGame(gameID string, handler MessageHandler) error {
	topic := TopicGamePrefix + gameID
	return r.Subscribe(topic, "", handler)
}

// Start 啟動消費者
func (r *rocketManagerImpl) Start(ctx context.Context) error {
	if err := r.consumer.Start(); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}
	r.logger.Info("RocketMQ consumer started successfully")

	// 阻塞直到上下文取消
	<-ctx.Done()

	return nil
}

// Shutdown 關閉連接
func (r *rocketManagerImpl) Shutdown() error {
	if err := r.producer.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown producer: %w", err)
	}

	if err := r.consumer.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown consumer: %w", err)
	}

	return nil
}
