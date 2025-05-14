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
		"localhost",
		"127.0.0.1",
		"172.19.0.3",  // Docker 容器 IP
		"rmq-namesrv", // Docker 服務名稱
		"rmq-broker",  // Docker 服務名稱
	}

	// 嘗試連接 NameServer (9876 端口) 和 Broker (10911 端口)
	ports := []string{"9876", "10911"}

	for _, hostname := range hostnames {
		for _, port := range ports {
			// 嘗試建立連接到 RocketMQ 端口
			address := hostname + ":" + port
			fmt.Printf("嘗試連接到 RocketMQ: %s\n", address)
			conn, err := net.DialTimeout("tcp", address, time.Second*2)
			if err == nil {
				// 成功建立連接，關閉並返回
				conn.Close()
				fmt.Printf("成功連接到 RocketMQ: %s\n", address)
				return
			}
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

	// 日誌輸出當前配置信息
	if config.RocketMQ.Enabled {
		l.Info("RocketMQ 已啟用",
			zap.Strings("nameServers", config.RocketMQ.NameServers),
			zap.String("producerGroup", config.RocketMQ.ProducerGroup))
	} else {
		l.Warn("RocketMQ 未啟用，將以禁用狀態創建生產者")
		return &MessageProducer{
			producer: nil,
			logger:   l,
			config:   config,
			disabled: true,
		}, nil
	}

	// 測試 NameServers 的連接性
	if len(config.RocketMQ.NameServers) == 0 {
		l.Error("RocketMQ NameServers 配置為空")
		return nil, fmt.Errorf("RocketMQ NameServers 配置為空")
	}

	// 檢查並測試每個 NameServer 的連接
	workingNameServers := make([]string, 0)
	for _, server := range config.RocketMQ.NameServers {
		// 確保地址包含端口
		if !strings.Contains(server, ":") {
			l.Warn("NameServer 地址缺少端口，添加默認端口 9876", zap.String("original", server))
			server = server + ":9876"
		}

		// 嘗試連接
		l.Debug("測試連接到 NameServer", zap.String("address", server))
		conn, err := net.DialTimeout("tcp", server, time.Second*3)
		if err == nil {
			conn.Close()
			l.Info("成功連接到 NameServer", zap.String("address", server))
			workingNameServers = append(workingNameServers, server)
		} else {
			l.Warn("無法連接到 NameServer", zap.String("server", server), zap.Error(err))

			// 嘗試解析主機名
			host, portStr, err := net.SplitHostPort(server)
			if err != nil {
				l.Error("NameServer 地址格式無效", zap.String("server", server), zap.Error(err))
				continue
			}

			// 嘗試使用 IP 而不是主機名
			if net.ParseIP(host) == nil && host != "localhost" {
				// 嘗試通過 DNS 解析主機名
				ips, err := net.LookupIP(host)
				if err != nil || len(ips) == 0 {
					l.Error("無法解析 NameServer 主機名", zap.String("host", host), zap.Error(err))

					// 嘗試一些可能的主機替代方案
					alternativeHosts := []string{"localhost", "127.0.0.1", "rmq-namesrv", "rmq-broker"}
					for _, altHost := range alternativeHosts {
						altServer := altHost + ":" + portStr
						l.Debug("嘗試連接到替代 NameServer", zap.String("address", altServer))
						conn, err := net.DialTimeout("tcp", altServer, time.Second*2)
						if err == nil {
							conn.Close()
							l.Info("成功連接到替代 NameServer", zap.String("address", altServer))
							workingNameServers = append(workingNameServers, altServer)
							break
						}
					}
					continue
				}

				// 使用解析後的 IP 地址替換主機名
				ipServer := ips[0].String() + ":" + portStr
				l.Info("已將主機名解析為 IP 地址", zap.String("host", host), zap.String("ip", ips[0].String()))

				// 測試 IP 連接
				conn, err := net.DialTimeout("tcp", ipServer, time.Second*2)
				if err == nil {
					conn.Close()
					l.Info("成功連接到解析後的 NameServer IP", zap.String("address", ipServer))
					workingNameServers = append(workingNameServers, ipServer)
				} else {
					l.Warn("無法連接到解析後的 NameServer IP", zap.String("server", ipServer), zap.Error(err))
				}
			}
		}
	}

	// 如果沒有找到可用的 NameServers，嘗試使用本地 localhost:9876
	if len(workingNameServers) == 0 {
		localServer := "localhost:9876"
		l.Debug("嘗試連接到本地 NameServer", zap.String("address", localServer))
		conn, err := net.DialTimeout("tcp", localServer, time.Second*2)
		if err == nil {
			conn.Close()
			l.Info("成功連接到本地 NameServer", zap.String("address", localServer))
			workingNameServers = append(workingNameServers, localServer)
		}
	}

	// 檢查是否有有效的 NameServers
	if len(workingNameServers) == 0 {
		l.Error("沒有可用的 RocketMQ NameServers 地址")
		return nil, fmt.Errorf("沒有可用的 RocketMQ NameServers 地址")
	}

	// 使用有效的 NameServers 替換原配置
	config.RocketMQ.NameServers = workingNameServers
	l.Info("使用以下有效的 NameServers", zap.Strings("nameServers", config.RocketMQ.NameServers))

	// 驗證 ProducerGroup 配置
	if config.RocketMQ.ProducerGroup == "" {
		l.Warn("RocketMQ producer group not configured, using default")
		config.RocketMQ.ProducerGroup = "lottery-producer-group"
	}

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

// IsEnabled 檢查生產者是否啟用
func (p *MessageProducer) IsEnabled() bool {
	if p == nil {
		return false
	}
	return !p.disabled && p.producer != nil
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
				l.Info("從環境變量檢測到 RocketMQ 設置", zap.Bool("DISABLE_ROCKETMQ", disableRocketMQ))
			}

			// 環境變量設置優先於配置文件，如果環境變量明確禁用，則忽略配置
			if disableRocketMQ {
				l.Warn("DISABLE_ROCKETMQ 環境變量設為 true，RocketMQ 功能已禁用")
				return &MessageProducer{
					producer: nil,
					logger:   l,
					config:   config,
					disabled: true,
				}, nil
			}

			// 檢查配置中是否啟用 RocketMQ
			if !config.RocketMQ.Enabled {
				l.Warn("配置中 RocketMQ 未啟用", zap.Bool("enabled", config.RocketMQ.Enabled))
				return &MessageProducer{
					producer: nil,
					logger:   l,
					config:   config,
					disabled: true,
				}, nil
			}

			// 檢查 NameServers 配置是否有效
			if len(config.RocketMQ.NameServers) == 0 {
				l.Error("RocketMQ NameServers 未配置，將禁用 RocketMQ")
				return &MessageProducer{
					producer: nil,
					logger:   l,
					config:   config,
					disabled: true,
				}, nil
			}

			// 嘗試檢測可用的 namesrv 和 broker 地址
			testAddresses := []struct {
				Type    string
				Address string
			}{
				{"namesrv", "localhost:9876"},
				{"broker", "localhost:10911"},
				{"namesrv", "rmq-namesrv:9876"},
				{"broker", "rmq-broker:10911"},
				{"namesrv", "172.19.0.2:9876"}, // docker network namesrv
				{"broker", "172.19.0.3:10911"}, // docker network broker
			}

			// 添加配置中的 NameServers
			for _, server := range config.RocketMQ.NameServers {
				if !strings.Contains(server, ":") {
					server = server + ":9876" // 添加默認端口
				}
				testAddresses = append(testAddresses, struct {
					Type    string
					Address string
				}{"namesrv", server})
			}

			// 測試連接
			var namesrvOK, brokerOK bool
			var workingNamesrv, workingBroker string

			for _, test := range testAddresses {
				l.Debug("嘗試連接到 "+test.Type, zap.String("address", test.Address))

				conn, err := net.DialTimeout("tcp", test.Address, time.Second*3)
				if err == nil {
					conn.Close()
					l.Info("成功連接到 "+test.Type, zap.String("address", test.Address))

					if test.Type == "namesrv" && !namesrvOK {
						namesrvOK = true
						workingNamesrv = test.Address
					} else if test.Type == "broker" && !brokerOK {
						brokerOK = true
						workingBroker = test.Address
					}
				} else {
					l.Debug("無法連接到 "+test.Type,
						zap.String("address", test.Address),
						zap.Error(err))
				}
			}

			// 檢查連接結果
			if !namesrvOK {
				l.Error("無法連接到任何 RocketMQ NameServer，將禁用 RocketMQ")
				return &MessageProducer{
					producer: nil,
					logger:   l,
					config:   config,
					disabled: true,
				}, nil
			}

			if !brokerOK {
				l.Warn("成功連接到 NameServer，但無法連接到 Broker，RocketMQ 功能可能受限")
				l.Info("請確保 broker.conf 中的 brokerIP1 設置正確")
			}

			// 更新 NameServers 配置
			config.RocketMQ.NameServers = []string{workingNamesrv}
			l.Info("將使用以下 NameServer", zap.String("address", workingNamesrv))

			if brokerOK {
				l.Info("Broker 連接正常", zap.String("address", workingBroker))
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
