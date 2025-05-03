# 遊戲事件訂閱客戶端

這是一個用於訂閱和監聽彩票服務遊戲事件的命令行工具，可以接收和顯示遊戲的實時事件流。

## 編譯方法

在專案根目錄下執行以下命令來編譯工具：

```bash
go build -o dealer_event_client cmd/dealer_client_event/main.go
```

編譯完成後，將在項目根目錄生成一個名為 `dealer_event_client` 的可執行文件。

## 使用方法

```bash
# 訂閱所有事件類型
./dealer_event_client

# 訂閱特定事件類型
./dealer_event_client <事件類型1> [事件類型2] [事件類型3]...
```

## 可用事件類型

可以使用以下簡寫指定要訂閱的事件類型：

- `STAGE` 或 `STAGE_CHANGED` - 遊戲階段變更事件
- `NEW` 或 `CREATE` 或 `GAME_CREATED` - 遊戲創建事件
- `CANCEL` 或 `GAME_CANCELLED` - 遊戲取消事件
- `COMPLETE` 或 `GAME_COMPLETED` - 遊戲完成事件
- `BALL` 或 `BALL_DRAWN` - 球抽取事件
- `SIDE` 或 `EXTRA_BALL_SIDE_SELECTED` - 額外球選邊事件
- `ALL` - 訂閱所有事件類型（與不提供任何參數效果相同）

## 使用示例

```bash
# 訂閱所有事件
./dealer_event_client

# 只訂閱球抽取事件
./dealer_event_client BALL

# 訂閱遊戲階段變更、創建和完成事件
./dealer_event_client STAGE CREATE COMPLETE
```

## 顯示格式

事件會以JSON格式顯示，包含事件類型和相關數據。例如：

```
接收到事件: GAME_EVENT_TYPE_STAGE_CHANGED
{
  "old_stage": "GAME_STAGE_PREPARATION",
  "new_stage": "GAME_STAGE_NEW_ROUND"
}
```

## 注意事項

1. 確保彩票服務已啟動並在預設的gRPC端口（9100）上監聽
2. 工具會持續運行直到收到中斷信號（Ctrl+C）
3. 如果連接中斷，工具會自動退出
4. 事件數據的格式和字段取決於事件類型

## 疑難排解

如果遇到問題，請檢查：

1. 彩票服務是否正在運行
2. gRPC端口是否正確
3. 指定的事件類型是否有效
4. 網絡連接是否正常 