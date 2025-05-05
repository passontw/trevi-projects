# StartNewRound gRPC 客戶端示例

這是一個展示如何使用 gRPC 客戶端調用 DealerService 的 StartNewRound 方法的示例應用程式。

## 用法

基本用法:
```bash
go run main.go
```

### 命令行選項

| 選項 | 說明 |
|------|------|
| `-cancel` | 在開始新遊戲前嘗試取消當前遊戲 |
| `-reason="原因"` | 指定取消遊戲的原因 (與 -cancel 一起使用) |
| `-skip-status` | 跳過獲取遊戲狀態的步驟 |
| `-direct-start` | 直接開始新遊戲，跳過其他檢查步驟 |
| `-skip-http` | 跳過 HTTP 請求測試 |

### 示例

直接開始新遊戲，跳過 HTTP 測試:
```bash
go run main.go -skip-http -direct-start
```

先取消當前遊戲再開始新遊戲:
```bash
go run main.go -skip-http -cancel -reason="測試取消"
```

## 請求和回應

### StartNewRound 請求

```json
{}
```

### StartNewRound 回應

```json
{
  "gameId": "遊戲ID",
  "startTime": "遊戲開始時間",
  "currentStage": "GAME_STAGE_NEW_ROUND"
}
```

## 故障排除

如果遇到 `context deadline exceeded` 錯誤，請檢查:

1. gRPC 服務器是否正在運行並監聽指定端口
2. 服務器是否實現了 StartNewRound 方法
3. 是否存在網絡連接問題或防火牆限制
4. 服務器端是否有 CPU 或 I/O 瓶頸導致響應緩慢
5. 當前遊戲狀態是否允許開始新遊戲 