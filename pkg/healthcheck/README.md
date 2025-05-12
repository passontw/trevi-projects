# 健康檢查套件 (pkg/healthcheck)

這個套件提供了健康檢查管理功能，可以輕鬆整合到服務中以實現 Kubernetes 的健康檢查需求。

## 功能特性

- 提供標準的健康檢查端點：`/liveness`、`/readiness`、`/healthz`
- 支援自定義健康檢查路徑
- 支援不同類型的健康檢查：存活檢查、就緒檢查
- 提供數據庫、Redis、HTTP 等常見服務的檢查器
- 支援 fx 依賴注入框架的整合
- 優雅啟動和關閉功能

## 使用方法

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

// 添加 Redis 健康檢查
health.AddReadinessCheck(&healthcheck.RedisChecker{
    Name_: "redis-cache",
    PingFunc: func(ctx context.Context) error {
        return redisClient.Ping(ctx).Err()
    },
})

// 設置服務為就緒
health.SetReady(true)
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

### 使用內建的伺服器

```go
// 創建並啟動服務器
server := healthcheck.NewServer(healthcheck.ServerConfig{
    Addr:          ":8080",
    Handler:       mux,
    HealthManager: health,
    Logger:        logger,
})

// 啟動伺服器
err := server.Start(func(ctx context.Context) error {
    // 初始化過程，完成後設置服務為就緒
    health.SetReady(true)
    return nil
})
```

## 檢查器類型

套件提供了多種檢查器：

- `PingChecker` - 基本的 ping 檢查
- `DatabaseChecker` - 數據庫連接檢查
- `RedisChecker` - Redis 連接檢查
- `HTTPChecker` - 外部 HTTP 服務檢查
- `CompositeChecker` - 組合多個檢查器
- `CustomChecker` - 自定義檢查邏輯

## 端點說明

默認提供的端點：

- `/liveness` - 存活檢查，確認服務是否運行
- `/readiness` - 就緒檢查，確認服務是否可以處理請求
- `/healthz` - 綜合健康檢查
- `/livez` - 詳細存活檢查（包含更多信息）
- `/readyz` - 詳細就緒檢查（包含更多信息）

## 優雅關閉

使用內建的伺服器可實現優雅關閉：

```go
// 捕獲中斷信號
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

// 等待中斷信號
<-sigCh

// 優雅關閉
server.Shutdown()
```

## 設計理念

本套件設計目標是提供簡單、可靠的健康檢查功能，同時支援與現有服務的輕鬆整合。主要設計原則：

1. **簡單性**：API 設計簡單直觀，易於使用和理解。
2. **靈活性**：支援多種檢查類型和自定義檢查邏輯。
3. **可靠性**：提供穩定的健康檢查機制，適用於生產環境。
4. **可觀測性**：詳細的日誌記錄，便於問題診斷。
5. **可擴展性**：易於添加新的檢查類型和整合到不同的框架中。 