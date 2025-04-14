# Gin 骨架项目

一个基于 Gin 框架的 Go 微服务骨架，针对初学者友好设计，采用模块化架构与依赖注入模式。

## 项目概述

这个骨架项目提供了一个完整的 Golang 微服务起点，整合了常用的基础设施和最佳实践：

- **模块化设计** - 清晰的目录结构与职责分离
- **依赖注入** - 使用 fx 框架实现松散耦合
- **数据库整合** - 使用 GORM 操作关系型数据库
- **缓存管理** - Redis 整合与会话管理
- **API 文档** - 自动生成 Swagger 文档
- **WebSocket 支持** - 实时通讯功能
- **配置管理** - 环境变量与配置中心支持

## 快速开始

### 安装

1. 复制此项目：

```bash
git clone https://github.com/yourusername/gin-skeleton.git my-project
cd my-project
```

2. 执行设置脚本：

```bash
./setup.sh
```

3. 按照提示输入您的项目名称 (例如 `github.com/yourusername/myproject`)

4. 脚本将自动：
   - 复制模板文件到正确位置
   - 初始化 Go 模块
   - 替换所有导入路径
   - 安装必要依赖

### 运行

直接使用 Go 运行项目：

```bash
go run ./main.go
```

或使用 Air 热重载（开发模式）：

```bash
# 安装 Air
go install github.com/cosmtrek/air@latest

# 启动应用
air
```

## 目录结构详解

```
.
├── main.go                     # 应用程序入口
├── internal/                   # 内部代码，不供外部导入
│   ├── config/                 # 配置管理
│   │   └── config.go           # 配置结构与加载逻辑
│   ├── domain/                 # 领域模型
│   │   └── user.go             # 用户模型示例
│   ├── handler/                # HTTP 请求处理
│   │   ├── router.go           # 路由配置
│   │   └── user_handler.go     # 用户相关 API 处理
│   ├── interfaces/             # 接口定义
│   │   └── service_interface.go # 服务接口
│   ├── middleware/             # HTTP 中间件
│   │   └── auth_middleware.go  # 身份验证中间件
│   └── service/                # 业务逻辑
│       └── user_service.go     # 用户服务实现
├── pkg/                        # 公共包，可被外部导入
│   ├── core/                   # 核心组件整合
│   ├── databaseManager/        # 数据库连接管理
│   ├── httpClient/             # HTTP 客户端
│   ├── logger/                 # 日志管理
│   ├── nacosManager/           # Nacos 配置中心
│   ├── redisManager/           # Redis 缓存管理
│   ├── utils/                  # 通用工具函数
│   └── websocketManager/       # WebSocket 连接管理
└── migrations/                 # 数据库迁移脚本
```

## 模块说明

### 核心模块 (pkg/core)

提供核心基础设施的整合层，使用 fx 模块整合各种基础组件：

```go
// pkg/core/module.go
var Module = fx.Options(
    databaseManager.Module,
    redisManager.Module,
    logger.Module,
    // 其他基础设施...
)
```

### 配置模块 (internal/config)

处理应用程序配置，支持从环境变量和文件加载：

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

### 处理器模块 (internal/handler)

HTTP 请求处理与路由设置：

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

### 服务模块 (internal/service)

实现业务逻辑，与数据库或其他外部服务交互：

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

## 如何扩展

### 添加新的 API 路由

1. 创建处理器函数：

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

2. 在路由中注册：

```go
// internal/handler/router.go
api.GET("/items", exampleHandler.GetItems)
```

### 添加新的服务

1. 创建服务接口：

```go
// internal/interfaces/item_service.go
type ItemService interface {
    GetItems() ([]domain.Item, error)
    AddItem(item domain.Item) error
}
```

2. 实现服务：

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

3. 注册到 fx 模块：

```go
// internal/service/module.go
var Module = fx.Options(
    fx.Provide(
        NewUserService,
        NewItemService, // 新增的服务
    ),
)
```

## 使用 WebSocket

### 服务端

WebSocket 管理器已集成到骨架中，可以这样使用：

```go
// internal/handler/ws_handler.go
func (h *WSHandler) HandleWS(c *gin.Context) {
    wsManager := h.wsManager
    wsManager.ServeWS(c.Writer, c.Request, func(client *websocketManager.Client) {
        // 可以在这里保存用户特定信息到客户端
        client.Data = map[string]interface{}{
            "userID": c.GetString("userID"),
        }
    })
}
```

### 客户端

```javascript
const socket = new WebSocket('ws://localhost:8080/ws');

socket.onopen = function() {
    console.log('连接已建立');
    socket.send(JSON.stringify({type: 'auth', token: 'your-jwt-token'}));
};

socket.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('收到消息:', data);
};
```

## 使用依赖注入 (fx)

骨架使用 fx 框架实现依赖注入，主入口在 `main.go`：

```go
func main() {
    app := fx.New(
        core.Module,        // 基础设施层
        config.Module,      // 配置模块
        service.Module,     // 服务层
        handler.Module,     // HTTP 处理层
    )
    
    app.Run()
}
```

这种方式让 fx 自动处理对象创建和依赖注入，例如：

```go
// 定义服务，需要数据库连接
func NewUserService(db *gorm.DB) *UserService {
    return &UserService{db: db}
}

// 定义处理器，需要服务
func NewUserHandler(service *UserService) *UserHandler {
    return &UserHandler{service: service}
}
```

fx 会自动创建这些对象并注入依赖，无需手动处理。

## 常见问题与解答

**Q: 如何连接到不同的数据库？**  
A: 在 `.env` 文件中修改数据库连接字符串：
```
DATABASE_URL=postgres://user:password@localhost:5432/mydatabase
```

**Q: 如何部署到生产环境？**  
A: 建议使用 Docker。模板已包含 Dockerfile，可以这样使用：
```bash
docker build -t myapp .
docker run -p 8080:8080 myapp
```

**Q: 我需要使用 WebSocket 吗？**  
A: 这是可选的。如果您不需要实时通讯功能，可以不使用 WebSocket 相关组件。

## 贡献指南

欢迎提交问题和改进建议！请先 fork 项目，然后创建 pull request。

## 授权

本项目采用 MIT 授权。 