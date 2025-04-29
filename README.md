# 開獎服務 (Lottery Service)

這是一個使用Go語言開發的開獎服務，包含荷官端(Dealer)和客戶端(Player)的WebSocket連接功能。

## 功能特點

- 使用Go 1.22新的ServeMux路由功能
- 支持荷官端開獎操作並廣播結果
- 支持玩家端訂閱特定遊戲的開獎結果
- 實時推送開獎數據
- 支持並發連接處理
- 基於事件的消息處理架構

## 系統架構

```
+----------------+      +-------------------+
|                |      |                   |
|  荷官端         |      |     開獎服務       |
|  WebSocket     |<---->|     連接器         |
|  (Port: 3002)  |      |                   |
|                |      |                   |
+----------------+      |                   |      +----------------+
                        |                   |      |                |
                        |                   |<---->|   玩家端        |
                        |                   |      |   WebSocket    |
                        |                   |      |   (Port: 3001) |
                        +-------------------+      |                |
                                                   +----------------+
```

## 服務端口

- 荷官端WebSocket服務: `3002`
- 玩家端WebSocket服務: `3001`

## 環境變量

設置在`.env`檔案中：

```
DEALER_WS_PORT=3002
PLAYER_WS_PORT=3001
```

## 消息格式

### 荷官端API

1. **Ping/Pong測試**
   - 發送: `{"type":"ping","payload":{}}`
   - 回應: `{"type":"pong","payload":{"time":"2023-05-20T12:34:56Z"}}`

2. **開獎操作**
   - 發送: `{"type":"draw_lottery","payload":{"game_id":"game1","result":[1,2,3,4,5]}}`
   - 回應: `{"type":"lottery_result","payload":{"game_id":"game1","result":[1,2,3,4,5],"time":"2023-05-20T12:34:56Z"}}`

### 玩家端API

1. **Ping/Pong測試**
   - 發送: `{"type":"ping","payload":{}}`
   - 回應: `{"type":"pong","payload":{"time":"2023-05-20T12:34:56Z"}}`

2. **訂閱開獎結果**
   - 發送: `{"type":"subscribe","payload":{"game_id":"game1"}}`
   - 回應: `{"type":"subscribe_success","payload":{"game_id":"game1","message":"Successfully subscribed to game updates"}}`

3. **取消訂閱**
   - 發送: `{"type":"unsubscribe","payload":{}}`
   - 回應: `{"type":"unsubscribe_success","payload":{"message":"Successfully unsubscribed from game updates"}}`

4. **接收開獎結果**
   - 接收: `{"type":"lottery_result","payload":{"game_id":"game1","result":[1,2,3,4,5],"time":"2023-05-20T12:34:56Z"}}`

## 測試頁面

提供了一個HTML測試頁面 `examples/websocket_test.html`，可以用來測試荷官端和玩家端的WebSocket連接和功能。

## 啟動服務

```bash
go run main.go
```

## 開發測試

1. 啟動服務
2. 打開 `examples/websocket_test.html` 文件
3. 點擊"連接"按鈕連接到荷官端和玩家端
4. 在玩家端訂閱遊戲ID
5. 在荷官端執行開獎操作
6. 觀察數據流動 