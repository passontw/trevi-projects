# 樂透遊戲荷官測試客戶端

這是一個簡單的WebSocket測試客戶端，用於模擬荷官端向服務器發送命令。

## 功能

- 連接到樂透遊戲WebSocket服務器
- 在連接建立後5秒發送一次「開始遊戲」命令
- 每15秒發送心跳命令
- 顯示從服務器接收到的所有消息
- 提供詳細的連接錯誤診斷

## 安裝

確保您已安裝Go 1.22或更高版本，然後執行以下命令：

```bash
cd examples/dealer
go mod tidy
```

## 使用方法

```bash
go run client.go [options]
```

### 可選參數

- `-url string`：WebSocket服務器地址（預設：`ws://localhost:3002`）
- `-client string`：客戶端ID（預設：`dealer-001`）
- `-debug`：啟用調試模式，提供更詳細的日誌輸出
- `-http-test`：在WebSocket連接失敗時嘗試進行HTTP連接測試

### 連接問題排查

如果遇到連接問題（如`bad handshake`錯誤），可以嘗試以下方法：

1. 確認服務器是否已啟動並運行在正確的端口
2. 使用`-http-test`參數檢查基本的HTTP連接
   ```bash
   go run client.go -http-test
   ```
3. 嘗試不同的WebSocket路徑：
   ```bash
   go run client.go -url ws://localhost:3002/ws
   ```
   或
   ```bash
   go run client.go -url ws://localhost:8080/lottery/ws
   ```
4. 開啟調試模式獲取更詳細的日誌：
   ```bash
   go run client.go -debug
   ```

### 範例

使用預設參數連接到本地服務器：

```bash
go run client.go
```

連接到自訂服務器地址：

```bash
go run client.go -url ws://example.com:3002 -client dealer-002
```

帶調試信息的連接：

```bash
go run client.go -debug -http-test
```

## 中斷程式

按下 `Ctrl+C` 可以正常關閉連接並退出程式。 