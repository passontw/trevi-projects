package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/mq"
	"g38_lottery_service/pkg/nacosManager"

	"go.uber.org/zap"
)

func main() {
	// 命令行參數
	gameID := flag.String("game", "test-game-001", "Game ID to use for the test message")
	topic := flag.String("topic", mq.LotteryResultTopic, "Topic to send the message to")
	flag.Parse()

	// 初始化日誌
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("無法初始化日誌: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 初始化 Nacos 客戶端 (如需要的話)
	var nacosClient nacosManager.NacosClient
	// 這裡可以初始化實際的 Nacos 客戶端，或者使用 nil

	// 加載配置
	cfg, err := config.LoadConfig(nacosClient)
	if err != nil {
		logger.Error("無法加載配置", zap.Error(err))
		os.Exit(1)
	}

	// 檢查 RocketMQ 配置
	if len(cfg.RocketMQ.NameServers) == 0 {
		logger.Error("RocketMQ NameServers 未配置")
		// 設置默認值用於測試
		cfg.RocketMQ.NameServers = []string{"127.0.0.1:9876"}
		logger.Info("已設置默認 NameServer 地址", zap.Strings("nameServers", cfg.RocketMQ.NameServers))
	}

	if cfg.RocketMQ.ProducerGroup == "" {
		cfg.RocketMQ.ProducerGroup = "test-producer-group"
		logger.Info("已設置默認生產者組", zap.String("producerGroup", cfg.RocketMQ.ProducerGroup))
	}

	// 創建 RocketMQ 生產者
	producer, err := mq.NewMessageProducer(logger, cfg)
	if err != nil {
		logger.Error("無法創建 RocketMQ 生產者", zap.Error(err))
		os.Exit(1)
	}
	defer producer.Stop()

	// 創建測試消息
	logger.Info("正在發送測試消息...",
		zap.String("topic", *topic),
		zap.String("gameID", *gameID))

	// 根據主題選擇不同的測試消息
	if *topic == mq.LotteryResultTopic {
		// 開獎結果測試消息
		err = producer.SendLotteryResult(*gameID, map[string]interface{}{
			"balls": []map[string]interface{}{
				{"number": 1, "color": "red"},
				{"number": 15, "color": "blue"},
				{"number": 22, "color": "green"},
				{"number": 33, "color": "yellow"},
				{"number": 45, "color": "purple"},
			},
			"extra_ball": map[string]interface{}{
				"number": 88,
				"color":  "gold",
			},
		})
	} else if *topic == mq.LotteryStatusTopic {
		// 狀態更新測試消息
		err = producer.SendLotteryStatus(*gameID, "DRAWING_COMPLETE", map[string]interface{}{
			"total_balls":  5,
			"drawn_balls":  5,
			"extra_balls":  1,
			"is_complete":  true,
			"has_winners":  true,
			"winner_count": 3,
		})
	} else {
		// 自定義主題測試消息
		err = producer.SendMessage(*topic, map[string]interface{}{
			"game_id":      *gameID,
			"message_type": "test_message",
			"data":         "這是一條測試消息",
			"test_value":   12345,
		})
	}

	if err != nil {
		logger.Error("發送消息失敗", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("測試消息發送成功！")

	// 如果需要監聽是否有收到回應，可以在這裡添加代碼
	fmt.Println("按 Ctrl+C 退出...")

	// 等待系統信號
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
