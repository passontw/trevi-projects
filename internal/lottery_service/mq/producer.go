package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"g38_lottery_service/internal/lottery_service/config"
	"sync"
	"time"

	"git.trevi.cc/server/go_gamecommon/msgqueue"
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
	// DNS解析間隔（秒）
	dnsResolvePeriod = 60
)

// MessageProducer 封裝 RocketMQ 生產者功能
type MessageProducer struct {
	producer    rocketmq.Producer
	dnsResolver *msgqueue.DnsResolver
	logger      *zap.Logger
	config      *config.AppConfig
	lock        sync.Mutex
	initialized bool
	cancelFunc  context.CancelFunc
}

// Message 代表一個 RocketMQ 消息
// 實際使用時應替換為 msgqueue.Message
type Message struct {
	Topic string
	Tag   string
	Keys  []string
	Body  []byte
}

// MessageResult 代表消息發送結果
// 實際使用時應替換為 msgqueue.SendResult
type MessageResult struct {
	MsgID string
}

// Producer 接口定義
// 實際使用時應匹配 msgqueue.Producer 接口
type Producer interface {
	Start() error
	Shutdown() error
	SendSync(ctx context.Context, msg *Message) (*MessageResult, error)
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

	// 轉換 NameServers 配置為 msgqueue.NamesrvNode 格式
	namesrvs := make([]msgqueue.NamesrvNode, 0, len(config.RocketMQ.NameServers))
	for _, addr := range config.RocketMQ.NameServers {
		// 解析地址格式 "host:port"
		host, portStr, found := parseHostPort(addr)
		if !found {
			l.Warn("無法解析 NameServer 地址格式", zap.String("address", addr))
			continue
		}

		// 轉換端口
		port, err := parsePort(portStr)
		if err != nil {
			l.Warn("無法解析 NameServer 端口", zap.String("port", portStr), zap.Error(err))
			continue
		}

		namesrvs = append(namesrvs, msgqueue.NamesrvNode{
			Host: host,
			Port: port,
		})
	}

	if len(namesrvs) == 0 {
		l.Error("無法解析任何有效的 NameServer 地址")
		return nil, fmt.Errorf("無法解析任何有效的 NameServer 地址")
	}

	// 創建 DNS 解析器
	l.Info("創建 RocketMQ DNS 解析器",
		zap.Any("nameServers", namesrvs))
	dnsResolver := msgqueue.NewDnsResolver(namesrvs)

	// 解析得到的地址
	resolvedNameServers := dnsResolver.Resolve()
	if len(resolvedNameServers) == 0 {
		l.Error("DNS 解析後無有效的 NameServer 地址")
		return nil, fmt.Errorf("DNS 解析後無有效的 NameServer 地址")
	}

	l.Info("解析 NameServer 地址成功",
		zap.Strings("resolvedNameServers", resolvedNameServers))

	// 創建生產者選項
	opts := []producer.Option{
		producer.WithNameServer(resolvedNameServers),
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
			zap.Strings("nameServers", resolvedNameServers),
			zap.String("producerGroup", config.RocketMQ.ProducerGroup))
		return nil, fmt.Errorf("創建 RocketMQ 生產者失敗: %w", err)
	}

	// 啟動生產者
	if err := p.Start(); err != nil {
		l.Error("啟動 RocketMQ 生產者失敗", zap.Error(err),
			zap.Strings("nameServers", resolvedNameServers),
			zap.String("producerGroup", config.RocketMQ.ProducerGroup))
		return nil, fmt.Errorf("啟動 RocketMQ 生產者失敗: %w", err)
	}

	l.Info("RocketMQ 生產者啟動成功")

	// 創建上下文，用於控制定時解析線程
	ctx, cancel := context.WithCancel(context.Background())

	// 創建並返回生產者實例
	msgProducer := &MessageProducer{
		producer:    p,
		dnsResolver: dnsResolver,
		logger:      l,
		config:      config,
		initialized: true,
		cancelFunc:  cancel,
	}

	// 啟動定時 DNS 解析
	go msgProducer.startDnsResolver(ctx, namesrvs)

	l.Info("啟動 NameServer DNS 定時解析器")

	return msgProducer, nil
}

// startDnsResolver 定時解析 NameServer 地址
func (p *MessageProducer) startDnsResolver(ctx context.Context, namesrvs []msgqueue.NamesrvNode) {
	ticker := time.NewTicker(dnsResolvePeriod * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("停止 NameServer DNS 定時解析")
			return
		case <-ticker.C:
			p.logger.Debug("執行 NameServer DNS 定時解析")

			// 解析地址
			resolvedAddrs := p.dnsResolver.Resolve()

			if len(resolvedAddrs) == 0 {
				p.logger.Warn("DNS 定時解析未返回有效地址")
				continue
			}

			// 由於 Apache RocketMQ 客戶端不支持動態更新 NameServer 地址，
			// 這裡僅記錄解析結果，實際生產環境中可能需要重新創建生產者
			p.logger.Debug("NameServer DNS 定時解析完成",
				zap.Strings("resolvedNameServers", resolvedAddrs))
		}
	}
}

// 解析 host:port 格式的地址
func parseHostPort(address string) (host string, port string, ok bool) {
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] == ':' {
			return address[:i], address[i+1:], true
		}
	}
	return "", "", false
}

// 解析端口字符串為整數
func parsePort(portStr string) (int, error) {
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0, err
	}
	return port, nil
}

// Stop 停止生產者
func (p *MessageProducer) Stop() {
	if p == nil {
		return
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	// 取消 DNS 解析器
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.logger.Info("已取消 DNS 解析任務")
	}

	if !p.initialized || p.producer == nil {
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
	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.initialized || p.producer == nil {
		p.logger.Error("RocketMQ producer is not initialized")
		return fmt.Errorf("RocketMQ producer is not initialized")
	}

	// 序列化消息體
	jsonData, err := json.Marshal(payload)
	if err != nil {
		p.logger.Error("Failed to marshal message payload",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("序列化消息失敗: %w", err)
	}

	// 日誌輸出：準備傳送的消息資訊
	p.logger.Debug("準備發送消息到 RocketMQ",
		zap.String("topic", topic),
		zap.Any("payload", payload))

	// 獲取標籤和鍵
	var tag string
	var keys []string

	if messageType, ok := payload["message_type"].(string); ok && messageType != "" {
		tag = messageType
	}

	if gameID, ok := payload["game_id"].(string); ok && gameID != "" {
		keys = append(keys, gameID)
	}

	// 構造消息
	msg := primitive.NewMessage(topic, jsonData)

	// 設置標籤和鍵
	if tag != "" {
		msg.WithTag(tag)
	}

	if len(keys) > 0 {
		msg.WithKeys(keys)
	}

	// 發送消息
	result, err := p.producer.SendSync(context.Background(), msg)
	if err != nil {
		p.logger.Error("Failed to send message",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("發送消息失敗: %w", err)
	}

	p.logger.Info("Message sent successfully",
		zap.String("topic", topic),
		zap.String("messageID", result.MsgID),
		zap.String("queue", result.MessageQueue.String()))

	return nil
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

// IsEnabled 檢查生產者是否啟用
func (p *MessageProducer) IsEnabled() bool {
	if p == nil {
		return false
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	return p.initialized && p.producer != nil
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

// 重要說明：
// 此文件提供了使用 git.trevi.cc/server/go_gamecommon/msgqueue 包的接口兼容實現
// 實際使用時，請確保正確引入該包並取消相關註釋
//
// 根據範例 livesvr/main.go，實際使用步驟應如下：
// 1. 在 go.mod 中添加 git.trevi.cc/server/go_gamecommon 依賴
// 2. 在 import 中添加 "git.trevi.cc/server/go_gamecommon/msgqueue"
// 3. 從 Nacos 獲取 rocketmq.xml 配置並使用 msgqueue.LoadConfigFromXML 解析
// 4. 使用 msgqueue.NewDnsResolver 創建 DNS 解析器
// 5. 使用 msgqueue.NewProducer 創建生產者
// 6. 使用 producer.SendSync 發送消息
//
// 例如：
// rocketmqconfigs, err := msgqueue.LoadConfigFromXML([]byte(configContent))
// if err != nil {
//     panic(err)
// }
// dnsresolver = msgqueue.NewDnsResolver(rocketmqconfigs.Namesrvs)
