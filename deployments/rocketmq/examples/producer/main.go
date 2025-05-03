package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

// RocketMQ 生產者示例
// 使用前請確保 RocketMQ 服務已啟動
func main() {
	// 設置 NameServer 地址
	nameserver := []string{"127.0.0.1:9876"}
	if len(os.Args) > 1 {
		nameserver = []string{os.Args[1]}
	}

	// 準備發送的 Topic
	topic := "test-topic"
	if len(os.Args) > 2 {
		topic = os.Args[2]
	}

	fmt.Printf("連接到 NameServer: %s\n", nameserver[0])
	fmt.Printf("發送到 Topic: %s\n", topic)

	// 創建生產者實例
	p, err := rocketmq.NewProducer(
		producer.WithNameServer(nameserver),
		producer.WithRetry(2),
		producer.WithGroupName("test-producer-group"),
		// 增加發送超時設置
		producer.WithSendMsgTimeout(time.Second*3),
	)
	if err != nil {
		fmt.Printf("初始化生產者失敗: %s\n", err.Error())
		os.Exit(1)
	}

	// 啟動生產者
	err = p.Start()
	if err != nil {
		fmt.Printf("啟動生產者失敗: %s\n", err.Error())
		os.Exit(1)
	}
	defer p.Shutdown()

	fmt.Println("生產者已啟動，開始發送消息...")
	fmt.Printf("目標 Topic: %s\n", topic)

	// 提示信息
	fmt.Println("提示：如果遇到 'lookup broker: no such host' 錯誤:")
	fmt.Println("1. 確保在 broker.conf 中正確設置了 brokerIP1 = host.docker.internal")
	fmt.Println("2. 確保已重新啟動 RocketMQ 容器使配置生效")
	fmt.Println("3. 確保已在 /etc/hosts 中添加 host.docker.internal 指向 127.0.0.1")
	fmt.Println()

	// 發送 5 條消息
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("這是測試消息 #%d，時間戳: %v", i, time.Now().Format(time.RFC3339))

		// 構建消息
		message := &primitive.Message{
			Topic: topic,
			Body:  []byte(msg),
		}

		// 設置消息標簽和鍵
		message.WithTag("tag1")
		message.WithKeys([]string{fmt.Sprintf("key-%d", i)})

		fmt.Printf("正在發送第 %d 條消息: %s\n", i+1, msg)

		// 同步發送消息
		result, err := p.SendSync(context.Background(), message)
		if err != nil {
			fmt.Printf("發送消息失敗: %s\n", err.Error())
			continue
		}

		fmt.Printf("消息發送成功，消息ID: %s, 存儲位置: %s\n",
			result.MsgID,
			result.MessageQueue.String())
		time.Sleep(time.Second) // 等待 1 秒再發送下一條
	}

	fmt.Println("所有消息已發送完成")
}
