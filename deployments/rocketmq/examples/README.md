# RocketMQ 示例程序

本目錄包含與 RocketMQ 互動的 Go 語言示例程序。

## 包含的示例

- **生產者示例 (producer)**: 展示如何建立連接並發送消息到 RocketMQ
- **消費者示例 (consumer)**: 展示如何訂閱主題並消費 RocketMQ 消息

## 編譯和運行示例

### 前置需求

1. Go 1.18 或更高版本
2. RocketMQ 服務已啟動 (可使用此目錄上層的 docker-compose 配置啟動)
3. 安裝所需依賴:

```bash
go mod tidy
# 或手動安裝
go get github.com/apache/rocketmq-client-go/v2
```

### 生產者示例

```bash
# 編譯
cd producer
go build -o producer

# 運行 (使用默認設置)
./producer

# 運行 (指定 NameServer 地址和主題)
./producer 127.0.0.1:9876 my-topic
```

### 消費者示例

```bash
# 編譯
cd consumer
go build -o consumer

# 運行 (使用默認設置)
./consumer

# 運行 (指定 NameServer 地址和主題)
./consumer 127.0.0.1:9876 my-topic
```

## 快速測試

要快速進行端到端測試，請按照以下步驟操作：

1. 首先啟動 RocketMQ 服務:
```bash
cd ../../
docker-compose up -d
```

2. 啟動消費者 (在一個終端窗口):
```bash
cd examples/consumer
go run main.go
```

3. 啟動生產者 (在另一個終端窗口):
```bash
cd examples/producer
go run main.go
```

4. 觀察消費者窗口中收到的消息

## 示例代碼說明

### 生產者示例

生產者示例展示了:
- 如何連接到 RocketMQ NameServer
- 如何創建生產者實例
- 如何發送帶有標簽和鍵的消息
- 如何處理發送結果和錯誤

### 消費者示例

消費者示例展示了:
- 如何連接到 RocketMQ NameServer
- 如何創建消費者實例和設置消費模式
- 如何訂閱主題並處理收到的消息
- 如何實現優雅關閉

## 常見問題

### 連接錯誤

如果出現連接錯誤，請確保:
- RocketMQ 服務已正確啟動
- 指定的 NameServer 地址可以訪問
- 沒有網絡或防火墻阻止連接

### 找不到主題

如果消費者報錯找不到主題，可能是因為:
- 主題尚未創建
- 主題名稱拼寫錯誤
- Broker 設置不允許自動創建主題 (檢查 `autoCreateTopicEnable` 配置) 