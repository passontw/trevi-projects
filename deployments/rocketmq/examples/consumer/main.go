package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// RocketMQ 消費者示例
// 使用前請確保 RocketMQ 服務已啟動
func main() {
	// 設置 NameServer 地址
	nameserver := []string{"127.0.0.1:9876"}
	if len(os.Args) > 1 {
		nameserver = []string{os.Args[1]}
	}

	// 設置要訂閱的 Topic
	topic := "game_events"
	if len(os.Args) > 2 {
		topic = os.Args[2]
	}

	fmt.Printf("連接到 NameServer: %s\n", nameserver[0])
	fmt.Printf("訂閱 Topic: %s\n", topic)

	// 創建推送型消費者
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer(nameserver),
		consumer.WithGroupName("test-consumer-group"),
		consumer.WithConsumerModel(consumer.BroadCasting),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
		consumer.WithConsumerOrder(false),
		// 設置重試次數，避免因網絡問題造成的持續重試
		consumer.WithRetry(2),
	)
	if err != nil {
		fmt.Printf("初始化消費者失敗: %s\n", err.Error())
		os.Exit(1)
	}

	// 訂閱 Topic，處理消息
	err = c.Subscribe(topic, consumer.MessageSelector{}, func(ctx context.Context,
		msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {

		for i, msg := range msgs {
			fmt.Printf("收到消息 [%d]: %s\n", i, string(msg.Body))
			fmt.Printf("  消息ID: %s, 存儲時間: %v, 標籤: %s, 鍵: %s\n",
				msg.MsgId,
				time.Unix(0, msg.StoreTimestamp).Format(time.RFC3339),
				msg.GetTags(),
				msg.GetKeys())
		}

		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		fmt.Printf("訂閱 Topic 失敗: %s\n", err.Error())
		os.Exit(1)
	}

	// 啟動消費者
	err = c.Start()
	if err != nil {
		fmt.Printf("啟動消費者失敗: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("消費者已啟動，訂閱 Topic: %s\n", topic)
	fmt.Println("按 Ctrl+C 退出程序...")

	// 如果遇到連接問題，可以嘗試檢查 broker 地址
	fmt.Println("提示：如果遇到 'lookup broker: no such host' 錯誤:")
	fmt.Println("1. 確保在 broker.conf 中正確設置了 brokerIP1 = host.docker.internal")
	fmt.Println("2. 確保已重新啟動 RocketMQ 容器使配置生效")
	fmt.Println("3. 確保已在 /etc/hosts 中添加 host.docker.internal 指向 127.0.0.1")

	// 等待系統信號以優雅停止
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// 使用 WaitGroup 保持主程序運行
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-sig
		fmt.Println("\n接收到停止信號，正在關閉消費者...")
		err = c.Shutdown()
		if err != nil {
			fmt.Printf("關閉消費者時出錯: %s\n", err.Error())
		}
		wg.Done()
	}()

	wg.Wait()
	fmt.Println("消費者已關閉")
}
