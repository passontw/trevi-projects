# WebSocket 和 gRPC 整合方案

本文檔描述了彩票系統中 WebSocket 和 gRPC 服務的整合實現。

## 設計概述

我們設計了一個完整的系統，使 gRPC 服務可以與 WebSocket 服務無縫整合，實現以下目標：

1. 當 gRPC 服務接收到遊戲狀態變更（如開始新局、抽球、階段變更等）時，能夠通過 WebSocket 廣播給所有連接的客戶端
2. 使用標準化的事件格式，確保消息格式一致性
3. 簡化開發和維護成本

## 系統架構

系統由以下主要組件組成：

1. **WebSocketEngine**：WebSocket 核心引擎，處理客戶端連接、消息解析和廣播
2. **DealerServer**：WebSocket 服務器，封裝 WebSocketEngine 並提供高層次 API
3. **GrpcBroadcaster**：gRPC 到 WebSocket 的橋接器，負責將 gRPC 事件轉換為 WebSocket 消息
4. **DealerService**：gRPC 服務實現，處理遊戲操作並生成事件

## 實現細節

### WebSocketEngine

`WebSocketEngine` 是一個通用的 WebSocket 引擎，提供以下功能：

- 管理客戶端連接（註冊、註銷）
- 處理消息路由和分發
- 提供消息廣播機制
- 支持 ping/pong 保活
- 提供客戶端元數據存儲

關鍵代碼：
```go
// WebSocketEngine 是WebSocket連線管理引擎
type WebSocketEngine struct {
	// 基本配置
	logger           *zap.Logger
	path             string
	clientType       string
	pingInterval     time.Duration
	closeGracePeriod time.Duration
	upgrader         websocket.Upgrader

	// 客戶端管理
	clients       map[string]*WebSocketClient
	register      chan *WebSocketClient
	unregister    chan *WebSocketClient
	broadcastChan chan []byte
	mu            sync.RWMutex

	// 處理器和回調
	handlers     map[string]MessageHandler
	onConnect    func(*WebSocketClient)
	onDisconnect func(*WebSocketClient)
}
```

### GrpcBroadcaster

`GrpcBroadcaster` 是 gRPC 和 WebSocket 之間的橋接器，負責：

- 接收 gRPC 事件
- 將 proto 消息轉換為 JSON 格式
- 通過 WebSocket 廣播事件

關鍵方法：
```go
// BroadcastGameEvent 廣播遊戲事件到所有WebSocket客戶端
func (b *GrpcBroadcaster) BroadcastGameEvent(event *dealer.GameEvent) error
```

### 事件流程

以抽球事件為例，事件處理流程如下：

1. `DealerService` 接收到抽球 RPC 請求
2. `DealerService` 調用 `gameManager.HandleDrawBall()` 處理請求
3. `gameManager` 觸發 `onBallDrawn` 回調
4. `DealerService` 創建 proto 事件消息
5. `DealerService` 調用 `grpcBroadcaster.BroadcastGameEvent()` 廣播事件
6. `GrpcBroadcaster` 將事件轉換為 JSON 格式
7. `GrpcBroadcaster` 通過 `DealerServer` 廣播消息
8. `WebSocketEngine` 將消息發送給所有連接的客戶端

## 消息格式

WebSocket 事件消息格式：

```json
{
  "type": "game_event",
  "payload": {
    "event_type": "BALL_DRAWN",
    "game_id": "game123",
    "timestamp": 1621234567,
    "data": {
      "ball_number": 42,
      "ball_type": "REGULAR",
      "is_last": false
    }
  }
}
```

## 擴展性

該設計具有良好的擴展性：

1. 可輕鬆添加新的事件類型
2. 可通過實現新的廣播器支持其他通信協議
3. 可通過註冊新的消息處理器支持雙向通信

## 部署與配置

gRPC 和 WebSocket 服務部署在同一應用中，共享以下配置：

- WebSocket 端口: 9000（可配置）
- gRPC 端口: 9100（可配置）

## 測試工具

系統附帶測試工具 `dealer_client.go`，可用於測試 gRPC 服務功能，包括：

- 開始新局
- 抽球
- 階段推進
- 設置 JP 狀態
- 取消遊戲

## 使用範例

```go
// 初始化
dealerService := dealer.NewDealerService(logger, gameManager)

// 啟動服務在 Server.go 中配置
``` 