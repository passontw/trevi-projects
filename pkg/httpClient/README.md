# HTTP客戶端模組

這個模組提供了一個統一的HTTP客戶端，用於在微服務間進行通信，並支持追蹤ID的傳遞。

## 特點

- 標準化的請求和響應格式
- 自動追蹤ID傳遞和管理
- 支持依賴注入 (基於uber/fx)
- 配置化的服務端點管理
- HTTP中間件支持

## 使用方法

### 基本用法

```go
import (
    "context"
    "github.com/your-org/your-repo/pkg/httpClient"
)

func main() {
    // 創建客戶端
    client := httpClient.NewClient(
        httpClient.WithServiceName("my-service"),
        httpClient.WithTimeout(30 * time.Second),
    )
    
    // 設置服務端點
    client.SetServiceEndpoint(httpClient.ServiceEndpoint{
        Name: "user-service",
        BaseURL: "http://localhost:8081/api",
    })
    
    // 發送GET請求
    ctx := context.Background()
    resp, err := client.Get(ctx, "user-service", "/users", map[string]string{"page": "1"}, nil)
    if err != nil {
        // 處理錯誤
    }
    
    // 使用響應數據
    fmt.Printf("狀態碼: %d\n", resp.StatusCode)
    fmt.Printf("追蹤ID: %s\n", resp.TraceID)
}
```

### 使用追蹤ID

```go
import (
    "context"
    "github.com/your-org/your-repo/pkg/httpClient"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 從請求中獲取追蹤ID
    traceID := httpClient.TraceIDFromRequest(r)
    
    // 創建帶有追蹤ID的上下文
    ctx := httpClient.WithTraceID(context.Background(), traceID)
    
    // 使用此上下文調用其他服務
    client := getHTTPClient() // 假設這個函數返回您的客戶端實例
    resp, err := client.Get(ctx, "another-service", "/data", nil, nil)
    
    // 響應中將自動包含相同的追蹤ID
}
```

### 使用中間件

```go
import (
    "net/http"
    "github.com/your-org/your-repo/pkg/httpClient"
)

func setupRouter() http.Handler {
    mux := http.NewServeMux()
    
    // 添加路由
    mux.HandleFunc("/api/data", handleData)
    
    // 應用追蹤中間件
    return httpClient.TraceMiddleware(mux)
}
```

### 使用依賴注入 (uber/fx)

```go
import (
    "go.uber.org/fx"
    "github.com/your-org/your-repo/pkg/httpClient"
)

func main() {
    app := fx.New(
        // 引入HTTP客戶端模組
        httpClient.Module,
        
        // 提供配置
        fx.Provide(func() *httpClient.Config {
            config, _ := httpClient.LoadConfig("config/services.json")
            return config
        }),
        
        // 提供服務名稱
        fx.Provide(func() string {
            return "my-service"
        }),
        
        // 引入您的服務模組
        fx.Provide(NewMyService),
    )
    
    app.Run()
}

// 在服務中使用HTTP客戶端
type MyService struct {
    client httpClient.HTTPClient
}

func NewMyService(client httpClient.HTTPClient) *MyService {
    return &MyService{client: client}
}
```

## 配置文件示例

```json
{
  "serviceEndpoints": {
    "user-service": {
      "name": "user-service",
      "baseUrl": "http://localhost:8081/api"
    },
    "payment-service": {
      "name": "payment-service",
      "baseUrl": "http://localhost:8082/api"
    }
  },
  "defaultTimeout": 30
}
``` 