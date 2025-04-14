# Gin 骨架專案

一個基於 Gin 框架的 Go 微服務骨架，針對初學者友好設計，採用模組化架構與依賴注入模式。

## 專案概述

這個骨架專案提供了一個完整的 Golang 微服務起點，整合了常用的基礎設施和最佳實踐：

- **模組化設計** - 清晰的目錄結構與職責分離
- **依賴注入** - 使用 fx 框架實現鬆散耦合
- **資料庫整合** - 使用 GORM 操作關聯式資料庫
- **快取管理** - Redis 整合與會話管理
- **API 文檔** - 自動生成 Swagger 文檔
- **WebSocket 支援** - 實時通訊功能
- **配置管理** - 環境變數與配置中心支援

## 快速開始

### 安裝

1. 複製此專案：

```bash
git clone https://github.com/yourusername/gin-skeleton.git my-project
cd my-project
```

2. 執行設定腳本：

```bash
./setup.sh
```

3. 依照提示輸入您的專案名稱 (例如 `github.com/yourusername/myproject`)

4. 腳本將自動：
   - 複製模板文件到正確位置
   - 初始化 Go 模組
   - 替換所有匯入路徑
   - 安裝必要依賴

### 運行

直接使用 Go 運行專案：

```bash
go run ./main.go
```

或使用 Air 熱重載（開發模式）：

```bash
# 安裝 Air
go install github.com/cosmtrek/air@latest

# 啟動應用
air
```

## 目錄結構詳解

```
.
├── main.go                     # 應用程式入口
├── internal/                   # 內部程式碼，不供外部匯入
│   ├── config/                 # 配置管理
│   │   └── config.go           # 配置結構與載入邏輯
│   ├── domain/                 # 領域模型
│   │   └── user.go             # 使用者模型範例
│   ├── handler/                # HTTP 請求處理
│   │   ├── router.go           # 路由配置
│   │   └── user_handler.go     # 使用者相關 API 處理
│   ├── interfaces/             # 介面定義
│   │   └── service_interface.go # 服務接口
│   ├── middleware/             # HTTP 中間件
│   │   └── auth_middleware.go  # 身份驗證中間件
│   └── service/                # 業務邏輯
│       └── user_service.go     # 使用者服務實現
├── pkg/                        # 公共套件，可被外部匯入
│   ├── core/                   # 核心組件整合
│   ├── databaseManager/        # 資料庫連接管理
│   ├── httpClient/             # HTTP 客戶端
│   ├── logger/                 # 日誌管理
│   ├── nacosManager/           # Nacos 配置中心
│   ├── redisManager/           # Redis 快取管理
│   ├── utils/                  # 通用工具函數
│   └── websocketManager/       # WebSocket 連接管理
└── migrations/                 # 資料庫遷移腳本
```

## 模組說明

### 核心模組 (pkg/core)

提供核心基礎設施的整合層，使用 fx 模組整合各種基礎組件：

```go
// pkg/core/module.go
var Module = fx.Options(
    databaseManager.Module,
    redisManager.Module,
    logger.Module,
    // 其他基礎設施...
)
```

### 配置模組 (internal/config)

處理應用程式配置，支援從環境變數和檔案載入：

```go
// internal/config/config.go
type Config struct {
    Server struct {
        Port string `env:"SERVER_PORT" envDefault:"8080"`
    }
    Database struct {
        URL string `env:"DATABASE_URL" envDefault:"postgres://user:pass@localhost:5432/dbname"`
    }
    // 其他配置...
}
```

### 處理器模組 (internal/handler)

HTTP 請求處理與路由設定：

```go
// internal/handler/router.go
func NewRouter(userHandler *UserHandler) *gin.Engine {
    router := gin.Default()
    
    api := router.Group("/api/v1")
    {
        api.GET("/users", userHandler.GetUsers)
        api.POST("/users", userHandler.CreateUser)
        // 其他路由...
    }
    
    return router
}
```

### 服務模組 (internal/service)

實現業務邏輯，與資料庫或其他外部服務交互：

```go
// internal/service/user_service.go
type UserService struct {
    db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
    return &UserService{db: db}
}

func (s *UserService) GetUsers() ([]domain.User, error) {
    var users []domain.User
    result := s.db.Find(&users)
    return users, result.Error
}
```

## 如何擴展

### 添加新的 API 路由

1. 創建處理器函數：

```go
// internal/handler/example_handler.go
func (h *ExampleHandler) GetItems(c *gin.Context) {
    items, err := h.service.GetItems()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, items)
}
```

2. 在路由中註冊：

```go
// internal/handler/router.go
api.GET("/items", exampleHandler.GetItems)
```

### 添加新的服務

1. 創建服務介面：

```go
// internal/interfaces/item_service.go
type ItemService interface {
    GetItems() ([]domain.Item, error)
    AddItem(item domain.Item) error
}
```

2. 實現服務：

```go
// internal/service/item_service.go
type itemService struct {
    db *gorm.DB
}

func NewItemService(db *gorm.DB) interfaces.ItemService {
    return &itemService{db: db}
}

func (s *itemService) GetItems() ([]domain.Item, error) {
    var items []domain.Item
    result := s.db.Find(&items)
    return items, result.Error
}
```

3. 注冊到 fx 模組：

```go
// internal/service/module.go
var Module = fx.Options(
    fx.Provide(
        NewUserService,
        NewItemService, // 新增的服務
    ),
)
```

## 使用 WebSocket

### 服務端

WebSocket 管理器已集成到骨架中，可以這樣使用：

```go
// internal/handler/ws_handler.go
func (h *WSHandler) HandleWS(c *gin.Context) {
    wsManager := h.wsManager
    wsManager.ServeWS(c.Writer, c.Request, func(client *websocketManager.Client) {
        // 可以在這裡保存用戶特定資訊到客戶端
        client.Data = map[string]interface{}{
            "userID": c.GetString("userID"),
        }
    })
}
```

### 客戶端

```javascript
const socket = new WebSocket('ws://localhost:8080/ws');

socket.onopen = function() {
    console.log('連接已建立');
    socket.send(JSON.stringify({type: 'auth', token: 'your-jwt-token'}));
};

socket.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('收到消息:', data);
};
```

## 使用依賴注入 (fx)

骨架使用 fx 框架實現依賴注入，主入口在 `main.go`：

```go
func main() {
    app := fx.New(
        core.Module,        // 基礎設施層
        config.Module,      // 配置模組
        service.Module,     // 服務層
        handler.Module,     // HTTP 處理層
    )
    
    app.Run()
}
```

這種方式讓 fx 自動處理對象創建和依賴注入，例如：

```go
// 定義服務，需要資料庫連接
func NewUserService(db *gorm.DB) *UserService {
    return &UserService{db: db}
}

// 定義處理器，需要服務
func NewUserHandler(service *UserService) *UserHandler {
    return &UserHandler{service: service}
}
```

fx 會自動創建這些對象並注入依賴，無需手動處理。

## 常見問題與解答

**Q: 如何連接到不同的資料庫？**  
A: 在 `.env` 檔案中修改資料庫連接字串：
```
DATABASE_URL=postgres://user:password@localhost:5432/mydatabase
```

**Q: 如何部署到生產環境？**  
A: 建議使用 Docker。模板已包含 Dockerfile，可以這樣使用：
```bash
docker build -t myapp .
docker run -p 8080:8080 myapp
```

**Q: 我需要使用 WebSocket 嗎？**  
A: 這是可選的。如果您不需要實時通訊功能，可以不使用 WebSocket 相關組件。

## 貢獻指南

歡迎提交問題和改進建議！請先 fork 專案，然後創建 pull request。

## 授權

本項目採用 MIT 授權。