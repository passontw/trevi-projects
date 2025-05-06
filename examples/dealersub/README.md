# WebSocket 訂閱服務範例

這是一個簡單的 WebSocket 訂閱服務範例，展示了如何使用 Go 語言實現發布/訂閱模式的實時通訊。

## 功能特點

- 實時雙向通訊
- 多主題訂閱支持
- JSON 格式訊息處理
- 發布/訂閱模式
- 定時訊息推送

## 編譯

```bash
# 編譯伺服器
cd examples/dealersub/server
go build

# 編譯客戶端
cd examples/dealersub/client
go build
```

## 使用方法

### 啟動伺服器

直接運行伺服器，會在9000端口監聽WebSocket連接，並每5秒向訂閱 `game_events` 主題的客戶端發送一條訊息：

```bash
go run examples/dealersub/server/main.go
```

### 啟動客戶端

客戶端可以以訂閱者、發布者或兩者皆是的模式運行：

```bash
# 訂閱者模式
go run examples/dealersub/client/main.go -mode subscriber -topic game_events

# 發布者模式
go run examples/dealersub/client/main.go -mode publisher -topic game_events -message "Hello, World!"

# 同時訂閱和發布
go run examples/dealersub/client/main.go -mode both -topic game_events -message "Test Message"
```

#### 客戶端參數

- `-mode`: 運行模式，可選值有 `subscriber`（訂閱者）、`publisher`（發布者）或 `both`（兩者）
- `-topic`: 訂閱或發布的主題
- `-message`: 要發布的訊息內容（僅對 publisher 和 both 模式有效）
- `-addr`: 服務器地址，格式為 `host:port`，默認為 `localhost:9000`

## 協議格式

### 客戶端到伺服器

1. 訂閱請求:
```json
{
  "type": "SUBSCRIBE",
  "data": {
    "topic": "game_events"
  }
}
```

2. 取消訂閱請求:
```json
{
  "type": "UNSUBSCRIBE",
  "data": {
    "topic": "game_events"
  }
}
```

3. 發布訊息:
```json
{
  "type": "PUBLISH",
  "data": {
    "topic": "game_events",
    "data": "Hello, World!"
  }
}
```

### 伺服器到客戶端

1. 訂閱成功:
```json
{
  "type": "SUBSCRIBED",
  "success": true,
  "message": "訂閱成功",
  "data": {
    "topic": "game_events"
  },
  "timestamp": 1620000000
}
```

2. 收到訊息:
```json
{
  "type": "MESSAGE",
  "success": true,
  "message": "收到訊息",
  "data": {
    "topic": "game_events",
    "data": "Hello, World!"
  },
  "timestamp": 1620000000
}
```

## 系統架構

- `examples/dealersub/server/main.go`: 伺服器實現，處理WebSocket連接、訂閱管理和訊息傳遞，每5秒會自動向 `game_events` 主題發送一條訊息
- `examples/dealersub/client/main.go`: 客戶端實現，可以作為訂閱者、發布者或兩者同時運行

## 開發者資訊

這個範例展示了如何使用 Go 的 WebSocket 庫實現一個簡單的 Pub/Sub 系統。它適合用於學習和作為實時通訊系統的起點。 