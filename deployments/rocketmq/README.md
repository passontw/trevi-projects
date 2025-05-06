# RocketMQ Docker 部署指南

本目錄包含 RocketMQ 消息中間件的 Docker Compose 部署配置。

## 組件

- **NameServer**: RocketMQ 的命名服務，負責維護所有 Broker 的位置信息
- **Broker**: 消息的路由、存儲和轉發
- **Dashboard**: Web 管理控制台，方便監控和管理 RocketMQ

## 目錄結構

```
deployments/rocketmq/
├── conf/                   # 配置文件目錄
│   └── broker.conf         # Broker 配置
├── data/                   # 持久化數據目錄 (自動創建)
│   ├── namesrv/            # NameServer 數據
│   └── broker/             # Broker 數據
├── docker-compose.yaml     # Docker Compose 配置
└── README.md               # 本文檔
```

## 使用方法

### 啟動服務

```bash
cd deployments/rocketmq
docker compose up -d
```

首次啟動會自動創建 `data` 目錄用於存儲 RocketMQ 數據。

### 檢查服務狀態

```bash
docker compose ps
```

### 停止服務

```bash
docker compose down
```

### 查看日誌

```bash
# 查看 NameServer 日誌
docker compose logs namesrv

# 查看 Broker 日誌
docker compose logs broker

# 查看 Dashboard 日誌
docker compose logs dashboard
```

## 訪問管理控制台

啟動服務後，可以通過以下地址訪問 RocketMQ Dashboard 管理控制台：

```
http://localhost:8080
```

## 服務端口

- NameServer: 9876
- Broker: 10909, 10911, 10912
- Dashboard: 8080

## 配置說明

### Broker 配置

`conf/broker.conf` 文件包含 Broker 的核心配置參數：

- `brokerClusterName`: 叢集名稱
- `brokerName`: Broker 名稱
- `brokerId`: 0 表示 Master Broker
- `autoCreateTopicEnable`: 是否允許自動創建 Topic
- `autoCreateSubscriptionGroup`: 是否允許自動創建訂閱組

### 內存配置

各服務的內存配置可在 `docker-compose.yaml` 中調整：

- NameServer: 
  - MAX_HEAP_SIZE=256M
  - HEAP_NEWSIZE=128M

- Broker: 
  - MAX_HEAP_SIZE=512M
  - HEAP_NEWSIZE=256M

對於生產環境，建議增加這些值以提高效能。

### 平台兼容性設置

RocketMQ 官方鏡像目前僅支援 linux/amd64 架構，在 Apple Silicon (ARM64) 設備上需要使用 Rosetta 2 模擬運行。
Docker Compose 配置文件中已設置平台指定為 linux/amd64：

```yaml
platform: linux/amd64  # 強制使用 x86_64 架構
```

此設置能確保在 Apple Silicon Mac 上通過模擬方式正確運行 RocketMQ 容器。

> 注意：目前 RocketMQ 官方尚未提供原生 ARM64 鏡像，但已在 GitHub 上有相關 Issue 討論 ([#116](https://github.com/apache/rocketmq-docker/issues/116))。
> 等待官方支援 ARM64 架構之前，只能通過模擬方式運行。

## 應用連接

要從應用程序連接到 RocketMQ，使用以下連接參數：

- NameServer地址: `localhost:9876`
- 生產者/消費者組: 根據您的應用需求自定義

## 常見問題

### 無法連接到 NameServer

確保端口 9876 未被其他應用占用，並檢查防火牆設置。

### Broker 無法啟動

檢查 broker.conf 配置，特別是 `brokerIP1` 是否正確設置為 `broker`。

### 內存不足

若遇到 OOM 錯誤，調整 `docker-compose.yaml` 中的內存配置。

### 磁盤空間不足

RocketMQ 需要足夠的磁盤空間來存儲消息。您可以調整 `fileReservedTime` 參數減少日誌保留時間。

### Topic 權限問題

若要使用 ACL，請修改 `broker.conf` 中的 `aclEnable` 設置為 `true` 並配置相應的權限文件。

### 平台架構問題

如果您在 Apple Silicon Mac 上運行時看到以下錯誤：

```
Error response from daemon: image with reference apache/rocketmq:5.1.3 was found but its platform (linux/amd64) does not match the specified platform (linux/arm64)
```

解決方案：

1. 使用 `platform: linux/amd64` 設置，讓 Docker 使用 Rosetta 2 模擬執行 x86_64 架構
2. 確保未指定 `platform: linux/arm64`，因為目前沒有官方 ARM64 鏡像

## 生產環境建議

1. 增加 Broker 內存配置
2. 設置適當的 commitLog 和消費隊列存儲策略
3. 啟用多主多從架構提高可用性
4. 啟用監控和告警系統
5. 定期備份 RocketMQ 數據 

---

# RocketMQ 核心概念與詳細設定指南

## 1. 核心架構組件

### 1.1 NameServer
- **功能**：輕量級的服務發現和路由注冊中心
- **特點**：無狀態、可互相獨立部署、彼此間無通信
- **配置項**：
  ```properties
  # NameServer 配置示例
  listenPort=9876                # 監聽端口
  serverWorkerThreads=8          # 工作線程數
  serverCallbackExecutorThreads=0 # 回調線程數
  serverSelectorThreads=3        # 選擇器線程數
  serverOnewaySemaphoreValue=256 # 單向請求信號量
  serverAsyncSemaphoreValue=64   # 異步請求信號量
  ```

### 1.2 Broker
- **角色分類**：
  - **Master**：提供讀寫服務
  - **Slave**：提供讀服務，自動從 Master 同步數據
- **部署模式**：
  - **單 Master**：簡單，有單點風險
  - **多 Master**：無單點風險，但單個 Master 宕機會導致部分消息不可用
  - **多 Master 多 Slave (同步雙寫)**：最高可用性，性能稍低
  - **多 Master 多 Slave (異步複製)**：高可用性與性能的平衡
- **重要配置項**：
  ```properties
  # Broker 核心配置
  brokerClusterName=DefaultCluster  # 集群名
  brokerName=broker-a               # Broker 名
  brokerId=0                        # 0表示Master，大於0表示Slave
  deleteWhen=04                     # 每天刪除時間點
  fileReservedTime=72               # 文件保留小時數
  flushDiskType=ASYNC_FLUSH         # 刷盤方式：SYNC_FLUSH或ASYNC_FLUSH
  ```

## 2. 消息模型詳解

### 2.1 Topic（主題）
- **概念**：消息的邏輯分類，發布/訂閱的基本單位
- **物理結構**：
  - 一個 Topic 分布在多個 Broker 上
  - 在每個 Broker 中，一個 Topic 包含多個 Queue
- **配置選項**：
  ```properties
  # Topic 相關配置
  autoCreateTopicEnable=true     # 是否允許自動創建Topic
  defaultTopicQueueNums=4        # 默認每個Topic的Queue數量
  topicSysFlag=0                 # Topic系統標誌
  ```
- **創建方式**：
  - 自動創建：生產者發送消息時自動創建
  - 手動創建：通過命令行或管理控制台創建
  ```bash
  # 手動創建Topic示例
  sh bin/mqadmin updateTopic -n localhost:9876 -t TestTopic -c DefaultCluster -r 8
  ```

### 2.2 Queue（隊列）
- **概念**：Topic 的物理分區，消息存儲和消費的最小單位
- **特性**：
  - 單個 Queue 內消息嚴格有序
  - 多個 Queue 可實現並行消費提高吞吐量
  - Queue 數量決定並行度
- **數量設置考量**：
  - Queue 數 ≤ 消費者數：每個消費者至少能分配到一個 Queue
  - Queue 數 > 消費者數：部分消費者會分配到多個 Queue
  - 建議設置為消費者數量的 1-2 倍

### 2.3 消息（Message）
- **結構組成**：
  - **Topic**：消息所屬主題
  - **Body**：消息主體內容
  - **Tags**：消息標簽，用於消費端過濾
  - **Keys**：消息鍵，用於查詢和業務標識
  - **Properties**：擴展屬性，支持自定義
- **消息大小**：默認限制 4MB，可通過 `maxMessageSize` 配置調整
- **消息類型**：
  - **普通消息**：最基本的消息類型
  - **順序消息**：保證消息消費順序
  - **延時消息**：指定延遲時間後才能被消費
  - **事務消息**：支持分布式事務處理
  - **批量消息**：一次發送多條消息提高吞吐量

### 2.4 消息示例（Go 客戶端）
```go
// 創建消息
message := &primitive.Message{
    Topic: "TestTopic",
    Body:  []byte("Hello RocketMQ"),
}

// 設置標簽和鍵
message.WithTag("TagA")
message.WithKeys([]string{"your-unique-key"})

// 設置自定義屬性
message.WithProperty("property-name", "property-value")

// 發送消息
result, err := producer.SendSync(context.Background(), message)
```

## 3. 消費模式詳解

### 3.1 集群消費（Clustering）
- **特點**：同一消費組內的多個消費者共同消費一個 Topic 的消息，每條消息只被一個消費者處理
- **負載均衡**：消費者平均分配 Queue，實現負載均衡
- **適用場景**：需要高吞吐、負載均衡的業務處理
- **配置示例**：
  ```go
  consumer, _ := rocketmq.NewPushConsumer(
      consumer.WithGroupName("your-consumer-group"),
      consumer.WithConsumerModel(consumer.Clustering), // 集群模式（默認）
  )
  ```

### 3.2 廣播消費（Broadcasting）
- **特點**：同一消費組內的每個消費者都接收 Topic 的所有消息
- **進度管理**：每個消費者單獨管理自己的消費進度
- **適用場景**：配置更新、全局緩存刷新、系統通知
- **配置示例**：
  ```go
  consumer, _ := rocketmq.NewPushConsumer(
      consumer.WithGroupName("your-consumer-group"),
      consumer.WithConsumerModel(consumer.BroadCasting), // 廣播模式
  )
  ```

### 3.3 消費位點管理
- **概念**：記錄消費者在每個 Queue 上的消費進度
- **存儲方式**：
  - **本地模式**：消費位點存儲在消費者本地
  - **遠程模式**：消費位點存儲在 Broker
- **消費起始位置選擇**：
  - **CONSUME_FROM_LAST_OFFSET**：從最新位置開始消費
  - **CONSUME_FROM_FIRST_OFFSET**：從最早位置開始消費
  - **CONSUME_FROM_TIMESTAMP**：從指定時間點開始消費
  ```go
  consumer, _ := rocketmq.NewPushConsumer(
      consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset), // 從最新位置開始
  )
  ```

## 4. 集群架構與高可用

### 4.1 集群（Cluster）
- **概念**：多個 Broker 組成的邏輯分組
- **命名規則**：通過 `brokerClusterName` 配置項指定
- **優勢**：提高容錯能力和系統吞吐量

### 4.2 Broker 組（Broker Group）
- **概念**：具有相同 `brokerName` 的多個 Broker 實例
- **組成**：一個 Master 和多個 Slave
- **數據同步**：Slave 從 Master 同步消息

### 4.3 高可用部署模式
- **單 Master**：
  ```
  +-------------+
  |  NameServer |
  +-------------+
         |
  +-------------+
  | Master Broker|
  +-------------+
  ```
  簡單但有單點風險

- **多 Master**：
  ```
  +-------------+
  |  NameServer |
  +-------------+
         |
  +-----------+-----------+
  | Master A  | Master B  |
  +-----------+-----------+
  ```
  無單點風險，但單個 Master 故障會導致部分消息不可用

- **多 Master 多 Slave (異步複製)**：
  ```
  +-------------+
  |  NameServer |
  +-------------+
         |
  +-----------+-----------+
  | Master A  | Master B  |
  +-----------+-----------+
  |  Slave A  |  Slave B  |
  +-----------+-----------+
  ```
  Master 故障時，Slave 可提供讀服務，但可能丟失部分消息

- **多 Master 多 Slave (同步雙寫)**：
  ```
  +-------------+
  |  NameServer |
  +-------------+
         |
  +-----------+-----------+
  | Master A  | Master B  |
  +-----------+-----------+
  |  Slave A  |  Slave B  |
  +-----------+-----------+
  ```
  最高可用性，消息不丟失，但性能較低

### 4.4 高可用配置
```properties
# 主從複製模式配置
brokerRole=ASYNC_MASTER          # 異步複製：ASYNC_MASTER，同步雙寫：SYNC_MASTER
flushDiskType=ASYNC_FLUSH        # 刷盤方式
slaveReadEnable=true             # 允許從 Slave 讀取數據
```

## 5. 消息追踪（MessageTrace）

### 5.1 概述
- **功能**：追踪消息從生產到消費的完整鏈路
- **實現方式**：通過插件化方式實現，將追踪數據存儲到指定 Topic
- **追踪內容**：生產時間、存儲時間、消費時間等關鍵節點信息

### 5.2 開啟方式
- **Broker 端配置**：
  ```properties
  # 在 broker.conf 中添加
  traceTopicEnable=true           # 開啟消息追踪
  ```

- **生產者端配置**：
  ```go
  p, _ := rocketmq.NewProducer(
      producer.WithNameServer(nameserver),
      producer.WithGroupName("test-producer-group"),
      producer.WithTrace(), // 開啟消息追踪
  )
  ```

- **消費者端配置**：
  ```go
  c, _ := rocketmq.NewPushConsumer(
      consumer.WithNameServer(nameserver),
      consumer.WithGroupName("test-consumer-group"),
      consumer.WithTrace(), // 開啟消息追踪
  )
  ```

### 5.3 追踪數據查看
- 通過 RocketMQ Dashboard 查看消息追踪信息
- 追踪數據默認存儲在 `TRACE_TOPIC_XXX` 主題

## 6. 高級特性與最佳實踐

### 6.1 消息存儲
- **CommitLog**：
  - 所有消息順序存儲在 CommitLog 文件中
  - 默認大小為 1GB
  - 存儲路徑可通過 `storePathCommitLog` 配置
- **ConsumeQueue**：
  - Topic 的邏輯隊列，存儲指向 CommitLog 的索引
  - 每個 Queue 對應一個 ConsumeQueue 文件
  - 存儲路徑可通過 `storePathConsumeQueue` 配置
- **存儲配置**：
  ```properties
  # 存儲相關配置
  storePathRootDir=/home/rocketmq/store  # 存儲根目錄
  storePathCommitLog=/home/rocketmq/store/commitlog  # CommitLog 路徑
  mapedFileSizeCommitLog=1073741824  # CommitLog 映射文件大小，默認 1GB
  ```

### 6.2 消息過濾
- **TAG 過濾**：基於消息 Tag 屬性進行過濾
  ```go
  // 訂閱指定 Tag 的消息
  consumer.Subscribe("TopicTest", consumer.MessageSelector{
      Type: consumer.TAG,
      Expression: "TagA || TagB",
  })
  ```

- **SQL92 過濾**：基於 SQL 表達式進行更復雜的過濾
  ```go
  // 使用 SQL 表達式過濾消息
  consumer.Subscribe("TopicTest", consumer.MessageSelector{
      Type: consumer.SQL92,
      Expression: "a > 5 AND b = 'abc'",
  })
  ```

### 6.3 性能調優
- **JVM 參數調優**：
  ```
  -server -Xms8g -Xmx8g -Xmn4g -XX:+UseG1GC -XX:G1HeapRegionSize=16m 
  -XX:G1ReservePercent=25 -XX:InitiatingHeapOccupancyPercent=30
  ```

- **Broker 參數調優**：
  ```properties
  sendMessageThreadPoolNums=16        # 發送消息線程池大小
  pullMessageThreadPoolNums=16        # 拉取消息線程池大小
  processReplyMessageThreadPoolNums=16 # 處理回覆消息線程池大小
  flushCommitLogLeastPages=4          # 刷新 CommitLog 最小頁數
  flushConsumeQueueLeastPages=2       # 刷新 ConsumeQueue 最小頁數
  ```

### 6.4 消息重試與死信隊列
- **消息重試機制**：
  - 消費失敗的消息自動進入重試隊列
  - 重試間隔遞增：1s, 5s, 10s, 30s, 1m, 2m...
  - 最大重試次數通過 `maxReconsumeTimes` 配置，默認 16 次
  ```go
  // 設置最大重試次數
  consumer, _ := rocketmq.NewPushConsumer(
      consumer.WithGroupName("your-consumer-group"),
      consumer.WithMaxReconsumeTimes(5), // 設置最大重試次數為 5
  )
  ```

- **死信隊列（DLQ）**：
  - 超過最大重試次數的消息進入死信隊列
  - 命名規則：`%DLQ%your-consumer-group`
  - 需要手動處理死信隊列中的消息

### 6.5 事務消息
- **實現流程**：
  1. 發送半事務消息（Half Message）
  2. 執行本地事務
  3. 根據本地事務結果提交或回滾消息
- **示例代碼**：
  ```go
  // 定義事務生產者監聽器
  type DemoTransactionListener struct {}
  
  // 執行本地事務
  func (l *DemoTransactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
      // 執行本地事務邏輯
      return primitive.CommitMessageState  // 提交消息
      // 或 return primitive.RollbackMessageState  // 回滾消息
      // 或 return primitive.UnknownState  // 事務狀態未知，等待檢查
  }
  
  // 檢查本地事務狀態
  func (l *DemoTransactionListener) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
      // 檢查本地事務狀態邏輯
      return primitive.CommitMessageState  // 提交消息
  }
  
  // 創建事務生產者
  p, _ := rocketmq.NewTransactionProducer(
      &DemoTransactionListener{},
      producer.WithNameServer(nameserver),
      producer.WithGroupName("transaction-producer-group"),
  )
  
  // 發送事務消息
  res, err := p.SendMessageInTransaction(context.Background(), message)
  ```

## 7. 總結與進階資源

### 7.1 常見應用場景
- **解耦微服務**：服務間通過消息隊列進行非同步通信
- **削峰填谷**：處理流量突增，保護下游服務
- **異步處理**：提升系統響應速度
- **順序處理**：保證消息處理順序
- **事務消息**：實現最終一致性

### 7.2 監控與運維
- **RocketMQ Dashboard**：官方 Web 控制台
- **監控指標**：
  - TPS（每秒事務數）
  - 消息積壓量
  - 磁盤使用率
  - 消費延遲
- **告警設置**：對關鍵指標設置告警閾值

### 7.3 進階學習資源
- [RocketMQ 官方文檔](https://rocketmq.apache.org/docs/)
- [RocketMQ GitHub 倉庫](https://github.com/apache/rocketmq)
- [RocketMQ Go 客戶端](https://github.com/apache/rocketmq-client-go) 