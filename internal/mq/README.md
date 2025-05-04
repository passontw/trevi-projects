# RocketMQ 生產者模組

本模組實現了 RocketMQ 生產者功能，用於開獎服務向遊戲端發送消息。

## 功能特點

- 整合到 FX 依賴注入框架
- 自動從配置中讀取 RocketMQ 連接設定
- 提供常用主題常量定義
- 提供封裝好的消息發送方法
- 支援自定義主題和消息格式
- 完整的錯誤處理和日誌記錄
- 優雅的啟動和關閉處理

## 配置說明

在 `config.go` 中已定義 `RocketMQConfig` 結構，必須在配置中提供以下資訊：

```json
{
  "rocketmq": {
    "nameServers": ["127.0.0.1:9876"],
    "accessKey": "",
    "secretKey": "",
    "producerGroup": "lottery-producer-group",
    "consumerGroup": "lottery-consumer-group"
  }
}
```

配置項說明：
- `nameServers`: RocketMQ NameServer 地址列表
- `accessKey`: 如果啟用了身份驗證，設置訪問鑰匙
- `secretKey`: 如果啟用了身份驗證，設置密鑰
- `producerGroup`: 生產者組名稱
- `consumerGroup`: 消費者組名稱（本模組僅實現生產者功能）

## 使用方法

### 發送開獎結果

```go
// 注入 MessageProducer
func SomeService(producer *mq.MessageProducer) {
    // 發送開獎結果
    err := producer.SendLotteryResult("game-001", map[string]interface{}{
        "balls": []map[string]interface{}{
            {"number": 1, "color": "red"},
            {"number": 15, "color": "blue"},
            // ...更多球
        },
        "extra_ball": map[string]interface{}{
            "number": 88,
            "color": "gold",
        },
    })
    
    if err != nil {
        // 處理錯誤
    }
}
```

### 發送開獎狀態更新

```go
// 發送開獎狀態
err := producer.SendLotteryStatus("game-001", "DRAWING_COMPLETE", map[string]interface{}{
    "total_balls": 5,
    "drawn_balls": 5,
    "is_complete": true,
    // ...其他狀態資訊
})
```

### 發送自定義消息

```go
// 發送自定義消息
err := producer.SendMessage("custom-topic", map[string]interface{}{
    "game_id": "game-001",
    "message_type": "custom_event",
    "data": customData,
})
```

## 測試工具

專案提供了一個測試工具用於測試 RocketMQ 生產者功能：

```bash
# 編譯測試工具
go build -o test_mq_producer cmd/tools/test_mq_producer.go

# 發送開獎結果測試消息
./test_mq_producer -game game-test-001 -topic lottery-result-topic

# 發送狀態更新測試消息
./test_mq_producer -game game-test-001 -topic lottery-status-topic

# 發送自定義主題消息
./test_mq_producer -game game-test-001 -topic custom-topic
```

## 消息格式

### 開獎結果消息 (lottery-result-topic)

```json
{
  "game_id": "game-001",
  "result": {
    "balls": [
      {"number": 1, "color": "red"},
      {"number": 15, "color": "blue"},
      {"number": 22, "color": "green"},
      {"number": 33, "color": "yellow"},
      {"number": 45, "color": "purple"}
    ],
    "extra_ball": {
      "number": 88,
      "color": "gold"
    }
  },
  "timestamp": 1628762345,
  "message_type": "lottery_result"
}
```

### 開獎狀態消息 (lottery-status-topic)

```json
{
  "game_id": "game-001",
  "status": "DRAWING_COMPLETE",
  "details": {
    "total_balls": 5,
    "drawn_balls": 5,
    "extra_balls": 1,
    "is_complete": true,
    "has_winners": true,
    "winner_count": 3
  },
  "timestamp": 1628762400,
  "message_type": "lottery_status"
}
```

## 注意事項

1. 確保 RocketMQ 服務已正確設置和啟動
2. 檢查配置中的 NameServer 地址是否正確
3. 如果在 Docker 環境中運行，需要確保網絡連接正確設置
4. 生產者和消費者的主題必須匹配
5. 消息大小不應超過 RocketMQ 的限制 (默認 4MB) 