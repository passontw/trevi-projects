# 健康檢查系統設計與實現指南

本文檔提供對 `g38_lottery_service` 中健康檢查系統的詳細說明，包括設計理念、實現細節、使用方法和最佳實踐。

## 目錄

- [概述](#概述)
- [技術選擇](#技術選擇)
- [核心組件](#核心組件)
- [健康檢查類型](#健康檢查類型)
- [實際應用範例](#實際應用範例)
- [優缺點分析](#優缺點分析)
- [擴展指南](#擴展指南)
- [Kubernetes 部署示例](#kubernetes-部署示例)
- [故障排除與常見問題](#故障排除與常見問題)

## 概述

健康檢查系統是微服務架構中的關鍵組件，特別是在 Kubernetes 等容器編排平台中，它們負責監控服務的狀態並影響自動擴展、負載均衡和故障恢復等決策。

我們的健康檢查系統採用了模組化設計，基於以下核心理念構建：

1. **簡單性**：API 設計簡潔，容易理解和使用
2. **可擴展性**：支持添加自定義檢查器
3. **高可靠性**：穩定運作，不引入額外故障點
4. **低資源消耗**：檢查過程輕量高效

## 技術選擇

我們的健康檢查系統基於 Go 標準庫實現，主要技術選擇包括：

1. **標準 `net/http` 包**：使用標準 HTTP 伺服器處理健康檢查請求
2. **Go 1.22 新版 `ServeMux`**：利用新版路由器支持的 HTTP 方法匹配
3. **原子操作**：使用 `sync/atomic` 確保狀態更新的線程安全
4. **上下文管理**：使用 `context` 包進行超時控制
5. **結構化日誌**：通過 `zap` 實現高效日誌記錄
6. **依賴注入**：通過 `fx` 實現組件的依賴管理

## 核心組件

### 健康檢查管理器 (Manager)

`Manager` 是健康檢查系統的核心組件，負責管理檢查器和處理健康檢查請求。

```go
// 創建健康檢查管理器
health := healthcheck.New(healthcheck.Config{
    Logger: logger,
})
```

主要功能：

- 管理不同類型的檢查器（存活檢查、就緒檢查等）
- 提供 HTTP 端點處理健康檢查請求
- 維護服務的就緒狀態（ready/not ready）
- 支持自定義健康檢查路徑

### 檢查器接口 (Checker)

所有健康檢查器必須實現 `Checker` 接口：

```go
type Checker interface {
    // 返回檢查器的名稱
    Name() string
    
    // 執行健康檢查，健康時返回 nil，否則返回錯誤
    Check(r *http.Request) error
}
```

### 通用檢查器

系統提供了多種通用檢查器：

1. **PingChecker**：基本的 ping 檢查，始終返回成功
2. **DatabaseChecker**：檢查數據庫連接是否正常
3. **RedisChecker**：檢查 Redis 連接是否正常
4. **HTTPChecker**：檢查外部 HTTP 服務是否可訪問
5. **CompositeChecker**：組合多個檢查器
6. **CustomChecker**：使用自定義邏輯的檢查器

### 優雅關閉服務器 (Server)

提供支持優雅啟動和關閉的 HTTP 服務器包裝：

```go
server := healthcheck.NewServer(healthcheck.ServerConfig{
    Addr:          ":8080",
    Handler:       mux,
    HealthManager: health,
    Logger:        logger,
})
```

主要功能：

- 啟動 HTTP 伺服器並執行初始化流程
- 處理操作系統信號（如 SIGTERM）
- 實現優雅關閉流程，避免中斷在途請求
- 在關閉過程中自動標記服務為未就緒

## 健康檢查類型

我們的健康檢查系統支持以下類型的檢查：

### 存活檢查 (Liveness)

確認服務是否運行中。如果存活檢查失敗，Kubernetes 會重啟容器。

端點：`/liveness` 或 `/livez`

### 就緒檢查 (Readiness)

確認服務是否已準備好處理請求。如果就緒檢查失敗，Kubernetes 會暫停將流量導向該服務實例。

端點：`/readiness` 或 `/readyz`

### 綜合健康檢查

結合存活和就緒檢查的結果。

端點：`/healthz`

## 實際應用範例

### 基本使用

```go
// 創建健康檢查管理器
health := healthcheck.New(healthcheck.Config{
    Logger: logger,
})

// 創建 HTTP 路由
mux := http.NewServeMux()

// 安裝健康檢查路由
health.InstallHandlers(mux)

// 添加數據庫健康檢查
health.AddReadinessCheck(&healthcheck.DatabaseChecker{
    Name_: "main-db",
    DB:    db,
})

// 設置服務為就緒
health.SetReady(true)
```

### 自定義檢查器

以遊戲管理器檢查器為例：

```go
// GameManagerChecker 檢查遊戲管理器健康狀態
type GameManagerChecker struct {
    Name_       string
    GameManager *GameManager
}

func (g *GameManagerChecker) Name() string {
    if g.Name_ != "" {
        return g.Name_
    }
    return "game-manager"
}

func (g *GameManagerChecker) Check(req *http.Request) error {
    if g.GameManager == nil {
        return fmt.Errorf("game manager not initialized")
    }
    
    // 獲取當前遊戲和支持的房間
    currentGame := g.GameManager.GetCurrentGame()
    supportedRooms := g.GameManager.GetSupportedRooms()
    
    // 檢查是否有支持的房間
    if len(supportedRooms) == 0 {
        return fmt.Errorf("no supported rooms")
    }
    
    // 檢查默認房間的遊戲是否存在
    if currentGame == nil {
        return fmt.Errorf("no current game in default room")
    }
    
    return nil
}
```

### 與 fx 框架整合

```go
app := fx.New(
    // 提供日誌記錄器
    fx.Provide(func() (*zap.Logger, error) {
        return zap.NewProduction()
    }),
    
    // 引入健康檢查模組
    healthcheck.Module,
    
    // 添加數據庫健康檢查
    fx.Invoke(func(health *healthcheck.Manager, db *sql.DB) {
        health.AddReadinessCheck(healthcheck.SQLDBChecker("main-db", db))
    }),
)
```

### 自定義健康檢查路徑

```go
// 創建帶有自定義路由的健康檢查管理器
health := healthcheck.New(healthcheck.Config{
    Logger:       logger,
    CustomRoutes: true,
})

// 自定義健康檢查路徑
customRoutes := map[string]string{
    "/api/system/ping":   "liveness",
    "/api/system/ready":  "readiness",
    "/api/system/health": "health",
}

// 安裝自定義健康檢查路由
health.InstallCustomHandlers(mux, customRoutes)
```

## 優缺點分析

### 優點

1. **模組化設計**：各組件之間松耦合，便於替換和升級
2. **標準庫為主**：最小化外部依賴，減少兼容性問題
3. **靈活性高**：支持多種檢查器和自定義邏輯
4. **易於整合**：與 fx 框架無縫集成，支持依賴注入
5. **優雅關閉**：支持平滑的服務啟動和關閉流程
6. **支持 Kubernetes**：符合 Kubernetes 健康檢查規範
7. **詳細日誌**：提供完整日誌記錄，便於問題診斷

### 缺點

1. **非通用協議**：不支持 gRPC 等其他協議的健康檢查
2. **檢查頻率固定**：不支持動態調整檢查頻率
3. **無狀態歷史**：不保存歷史健康檢查結果
4. **依賴 zap**：對 zap 日誌庫有依賴
5. **無分佈式追蹤**：不支持與分佈式追蹤系統集成

## 擴展指南

### 添加新的檢查器

創建一個實現 `Checker` 接口的結構：

```go
type MyServiceChecker struct {
    Name_ string
    Service *MyService
}

func (c *MyServiceChecker) Name() string {
    if c.Name_ != "" {
        return c.Name_
    }
    return "my-service"
}

func (c *MyServiceChecker) Check(r *http.Request) error {
    if c.Service == nil {
        return fmt.Errorf("service not initialized")
    }
    
    // 執行服務特定的檢查邏輯
    if !c.Service.IsRunning() {
        return fmt.Errorf("service is not running")
    }
    
    return nil
}
```

註冊檢查器：

```go
health.AddReadinessCheck(&MyServiceChecker{
    Name_: "my-critical-service",
    Service: myService,
})
```

### 整合消息隊列檢查

以 Kafka 為例：

```go
type KafkaChecker struct {
    Name_ string
    Client *kafka.Client
    Topics []string
}

func (c *KafkaChecker) Name() string {
    if c.Name_ != "" {
        return c.Name_
    }
    return "kafka"
}

func (c *KafkaChecker) Check(r *http.Request) error {
    if c.Client == nil {
        return fmt.Errorf("kafka client not initialized")
    }
    
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()
    
    // 檢查連接
    if err := c.Client.Ping(ctx); err != nil {
        return fmt.Errorf("failed to connect to kafka: %w", err)
    }
    
    // 檢查主題
    for _, topic := range c.Topics {
        if exists, err := c.Client.TopicExists(ctx, topic); err != nil {
            return fmt.Errorf("failed to check topic %s: %w", topic, err)
        } else if !exists {
            return fmt.Errorf("required topic %s does not exist", topic)
        }
    }
    
    return nil
}
```

### 基於 OAuth 的安全健康檢查

添加安全中間件：

```go
func SecureHealthCheck(next http.Handler, validator func(*http.Request) bool) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !validator(r) {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// 使用
secureHandler := SecureHealthCheck(health.GetHandler("health"), func(r *http.Request) bool {
    token := r.Header.Get("Authorization")
    return validateToken(token)
})

mux.Handle("GET /secure/health", secureHandler)
```

## Kubernetes 部署示例

在 Kubernetes 環境中配置健康檢查是發揮其全部價值的關鍵。以下是一個完整的 Kubernetes 部署示例，展示如何配置存活檢查和就緒檢查。

### Deployment 配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: lottery-service
  namespace: games
  labels:
    app: lottery-service
    component: backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: lottery-service
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: lottery-service
    spec:
      containers:
      - name: lottery-service
        image: g38/lottery-service:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: DB_HOST
          value: "postgres.database"
        - name: REDIS_HOST
          value: "redis.cache"
        - name: LOG_LEVEL
          value: "info"
        - name: SERVER_PORT
          value: "8080"
        - name: SHUTDOWN_TIMEOUT
          value: "30"
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /liveness
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readiness
            port: http
          initialDelaySeconds: 15
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
        startupProbe:
          httpGet:
            path: /readiness
            port: http
          failureThreshold: 30
          periodSeconds: 10
        lifecycle:
          preStop:
            exec:
              command: ["sh", "-c", "sleep 10"]
```

### Service 配置

```yaml
apiVersion: v1
kind: Service
metadata:
  name: lottery-service
  namespace: games
  labels:
    app: lottery-service
spec:
  selector:
    app: lottery-service
  ports:
  - port: 80
    targetPort: 8080
    name: http
  type: ClusterIP
```

### 配置說明

1. **存活檢查 (Liveness Probe)**：
   - 使用 `/liveness` 端點檢查服務是否存活
   - 初始延遲 10 秒，給服務啟動時間
   - 每 10 秒檢查一次
   - 超時 5 秒，如果響應時間超過 5 秒則視為失敗
   - 連續 3 次失敗後，Kubernetes 將重啟容器

2. **就緒檢查 (Readiness Probe)**：
   - 使用 `/readiness` 端點檢查服務是否就緒
   - 初始延遲 15 秒，留出時間完成初始化
   - 每 5 秒檢查一次，頻率高於存活檢查
   - 超時 3 秒，較存活檢查更嚴格
   - 連續 2 次失敗後，Kubernetes 將停止向該 Pod 發送流量，但不會重啟容器

3. **啟動檢查 (Startup Probe)**：
   - 使用 `/readiness` 端點確認服務已完成啟動
   - 允許 30 次失敗（共 300 秒），適用於啟動較慢的服務
   - 在啟動檢查成功前，存活檢查和就緒檢查不會開始執行

4. **優雅關閉**：
   - 通過 `preStop` 鉤子設置 10 秒的優雅關閉時間
   - 允許服務完成處理中的請求並進行清理

### HorizontalPodAutoscaler 配置

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: lottery-service
  namespace: games
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: lottery-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

這個 HPA 配置會基於 CPU 使用率自動擴展服務實例數量，確保系統在負載增加時能及時擴展。

### 故障恢復策略

Kubernetes 會根據健康檢查結果採取以下操作：

1. **服務不可用時**：
   - 如果存活檢查失敗，Pod 將被重新啟動
   - 如果就緒檢查失敗，Pod 將被標記為未就緒，流量將不再路由到該 Pod

2. **滾動更新時**：
   - 基於就緒檢查確保只有健康的 Pod 接收流量
   - 通過優雅關閉確保請求被完整處理
   - `maxSurge` 和 `maxUnavailable` 設置確保更新過程中服務容量不會顯著下降

3. **系統故障時**：
   - 節點故障後，健康檢查會幫助 Kubernetes 快速識別不健康的 Pod
   - Kubernetes 會在其他節點上調度新的 Pod 替代不健康的實例

## 故障排除與常見問題

### 服務無法通過健康檢查

可能原因：

1. **依賴服務不可用**：檢查數據庫、Redis 等依賴服務是否正常
2. **超時設置過短**：如果檢查邏輯需要較長時間，考慮延長超時設置
3. **主機或端口綁定問題**：確認服務正確綁定到了配置的 IP 和端口

解決方案：

- 檢查日誌了解具體失敗原因
- 臨時禁用特定檢查器進行隔離測試
- 使用 curl 直接調用健康檢查端點進行測試

### 優雅關閉沒有按預期工作

可能原因：

1. **關閉超時過短**：服務可能需要更長時間完成現有請求
2. **信號處理不正確**：確保正確捕獲了 SIGTERM 和 SIGINT 信號
3. **阻塞操作**：某些操作可能阻塞了關閉過程

解決方案：

- 增加關閉超時時間
- 確保阻塞操作有適當的超時設置
- 添加更詳細的日誌來追蹤關閉流程

### 健康檢查影響性能

可能原因：

1. **檢查頻率過高**：Kubernetes 默認每秒進行多次檢查
2. **檢查邏輯過重**：複雜的檢查邏輯可能消耗過多資源
3. **外部依賴延遲**：數據庫等外部服務響應慢

解決方案：

- 簡化檢查邏輯，僅檢查關鍵組件
- 優化查詢，例如使用 `SELECT 1` 代替複雜查詢
- 考慮實現結果緩存，避免頻繁執行相同檢查

---

通過本指南，您應該能夠理解我們健康檢查系統的設計理念和實現細節，以及如何在您的服務中使用和擴展它。如有更多問題或需求，請參考代碼文檔或聯繫開發團隊。 