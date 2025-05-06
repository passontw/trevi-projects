package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"g38_lottery_service/internal/lottery_service/config"
	"net"
	"os"
	"strings"
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

// 自動檢查環境變量並決定是否禁用 RocketMQ
func init() {
	// 如果環境變量已經設置，則不做改變
	if _, exists := os.LookupEnv("DISABLE_ROCKETMQ"); exists {
		return
	}

	// 檢查 RocketMQ 配置是否可能有問題
	hostnames := []string{
		"172.237.27.51",
		"localhost",
		"127.0.0.1",
	}

	for _, hostname := range hostnames {
		// 嘗試建立連接到預設 RocketMQ 端口
		conn, err := net.DialTimeout("tcp", hostname+":9876", time.Second*1)
		if err == nil {
			// 成功建立連接，關閉並返回
			conn.Close()
			return
		}
	}

	// 如果無法連接到任何一個預設地址，則自動禁用 RocketMQ
	fmt.Println("⚠️ 無法連接到任何 RocketMQ 服務器，自動禁用 RocketMQ 功能")
	os.Setenv("DISABLE_ROCKETMQ", "true")
}

// MessageProducer 封裝 RocketMQ 生產者功能
type MessageProducer struct {
	producer rocketmq.Producer
	logger   *zap.Logger
	config   *config.AppConfig
	disabled bool
}

// NewMessageProducer 創建新的消息生產者
func NewMessageProducer(logger *zap.Logger, config *config.AppConfig) (*MessageProducer, error) {
	// 日誌標記
	l := logger.With(zap.String("component", "rocketmq_producer"))

	// 驗證配置
	if len(config.RocketMQ.NameServers) == 0 {
		l.Error("RocketMQ NameServers not configured")
		return nil, fmt.Errorf("RocketMQ NameServers 配置為空")
	}

	if config.RocketMQ.ProducerGroup == "" {
		l.Warn("RocketMQ producer group not configured, using default")
		config.RocketMQ.ProducerGroup = "lottery-producer-group"
	}

	// 檢查並修正 NameServers 地址格式
	validNameServers := make([]string, 0, len(config.RocketMQ.NameServers))
	for _, server := range config.RocketMQ.NameServers {
		// 確保地址包含端口
		if !strings.Contains(server, ":") {
			l.Warn("NameServer 地址缺少端口，添加默認端口 9876", zap.String("original", server))
			server = server + ":9876"
		}

		// 嘗試解析地址
		host, portStr, err := net.SplitHostPort(server)
		if err != nil {
			l.Error("NameServer 地址格式無效", zap.String("server", server), zap.Error(err))
			continue
		}

		// 檢查 IP 地址是否有效
		if net.ParseIP(host) == nil && host != "localhost" {
			// 嘗試通過 DNS 解析主機名
			ips, err := net.LookupIP(host)
			if err != nil || len(ips) == 0 {
				l.Error("無法解析 NameServer 主機名", zap.String("host", host), zap.Error(err))
				continue
			}

			// 使用解析後的 IP 地址替換主機名
			server = ips[0].String() + ":" + portStr
			l.Info("已將主機名解析為 IP 地址", zap.String("host", host), zap.String("ip", ips[0].String()))
		}

		validNameServers = append(validNameServers, server)
	}

	// 檢查是否有有效的 NameServers
	if len(validNameServers) == 0 {
		l.Error("沒有有效的 RocketMQ NameServers 地址")
		return nil, fmt.Errorf("沒有有效的 RocketMQ NameServers 地址")
	}

	// 使用有效的 NameServers 替換原配置
	config.RocketMQ.NameServers = validNameServers

	// 日誌信息
	l.Info("Initializing RocketMQ producer",
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
		l.Error("Failed to create RocketMQ producer", zap.Error(err))
		return nil, fmt.Errorf("創建 RocketMQ 生產者失敗: %w", err)
	}

	// 啟動生產者
	if err := p.Start(); err != nil {
		l.Error("Failed to start RocketMQ producer", zap.Error(err))
		return nil, fmt.Errorf("啟動 RocketMQ 生產者失敗: %w", err)
	}

	l.Info("RocketMQ producer started successfully")

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

	if p.disabled {
		if p.logger != nil {
			p.logger.Debug("RocketMQ producer is disabled, no need to stop")
		}
		return
	}

	if p.producer != nil {
		err := p.producer.Shutdown()
		if err != nil {
			p.logger.Error("Error shutting down RocketMQ producer", zap.Error(err))
		} else {
			p.logger.Info("RocketMQ producer shutdown successfully")
		}
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
	// 如果生產者已禁用，記錄訊息並返回
	if p.disabled || p.producer == nil {
		p.logger.Debug("RocketMQ producer is disabled, message not sent",
			zap.String("topic", topic),
			zap.Any("payload", payload))
		return nil
	}

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

// Module 提供 FX 模塊
var Module = fx.Options(
	// 註冊 MessageProducer
	fx.Provide(
		func(logger *zap.Logger, config *config.AppConfig) (*MessageProducer, error) {
			// 檢查是否啟用了 RocketMQ
			l := logger.With(zap.String("component", "rocketmq_module"))

			// 從環境變量中檢查是否應該禁用 RocketMQ
			disableRocketMQ := false
			if v, exists := os.LookupEnv("DISABLE_ROCKETMQ"); exists {
				disableRocketMQ = strings.ToLower(v) == "true"
			}

			// 如果環境變量中明確禁用，則創建模擬生產者並返回
			if disableRocketMQ {
				l.Warn("DISABLE_ROCKETMQ 環境變量設為 true，RocketMQ 功能已禁用")
				return &MessageProducer{
					producer: nil,
					logger:   l,
					config:   config,
					disabled: true,
				}, nil
			}

			// 嘗試創建 RocketMQ 生產者
			producer, err := NewMessageProducer(logger, config)
			if err != nil {
				// 如果連接失敗，記錄錯誤，但不中斷應用啟動
				l.Error("無法初始化 RocketMQ 生產者，將以禁用狀態運行",
					zap.Error(err))

				// 顯示如何完全禁用 RocketMQ 的提示
				l.Info("若要在開發環境中禁用 RocketMQ，請設置環境變量 DISABLE_ROCKETMQ=true")

				// 返回禁用狀態的生產者
				return &MessageProducer{
					producer: nil,
					logger:   l,
					config:   config,
					disabled: true,
				}, nil
			}

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
