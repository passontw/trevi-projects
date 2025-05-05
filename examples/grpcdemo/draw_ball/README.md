# DrawBall gRPC 客戶端示例

這個示例演示了如何使用 gRPC 客戶端呼叫 DealerService 的 DrawBall RPC 方法。

## 前置條件

- Go 1.23 或更高版本
- 運行中的 DealerService gRPC 服務器

## 設置

首先，確保依賴項已下載：

```bash
# 在 draw_ball 目錄中執行
go mod tidy
```

## 使用方法

### 使用默認設置運行

默認情況下，客戶端會嘗試連接到 `localhost:9100`：

```bash
go run main.go
```

### 指定自定義服務器地址

可以使用 `-server` 參數指定不同的服務器地址：

```bash
go run main.go -server=dealer-service:9100
```

## 功能說明

這個客戶端執行以下操作：

1. 連接到 DealerService gRPC 服務器
2. 創建一個包含 30 個常規球的請求，最後一個球標記為 `is_last=true`
3. 調用 DrawBall RPC 方法
4. 顯示服務器的回應，包括：
   - 回應中的球列表
   - 如果提供了遊戲狀態更新，則顯示狀態信息
   - 以 JSON 格式顯示完整回應

## 注意事項

- 確保 DealerService gRPC 服務器正在運行並可訪問
- 如果連接失敗，請檢查服務器地址和網絡設置
- 在執行客戶端之前，可能需要先啟動遊戲（使用 StartNewRound RPC） 