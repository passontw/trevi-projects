# Gin Skeleton Project

A Go microservice skeleton based on the Gin framework, designed to be beginner-friendly with a modular architecture and dependency injection pattern.

## Project Overview

This skeleton project provides a complete starting point for Golang microservices, integrating common infrastructure and best practices:

- **Modular Design** - Clear directory structure with separation of concerns
- **Dependency Injection** - Loose coupling using the fx framework
- **Database Integration** - Relational database operations with GORM
- **Cache Management** - Redis integration and session management
- **API Documentation** - Automatic Swagger documentation generation
- **WebSocket Support** - Real-time communication features
- **Configuration Management** - Environment variables and configuration center support

## Quick Start

### Installation

1. Clone this project:

```bash
git clone https://github.com/yourusername/gin-skeleton.git my-project
cd my-project
```

2. Run the setup script:

```bash
./setup.sh
```

3. Follow the prompts to enter your project name (e.g., `github.com/yourusername/myproject`)

4. The script will automatically:
   - Copy template files to the correct locations
   - Initialize the Go module
   - Replace all import paths
   - Install necessary dependencies

### Running

Run the project directly with Go:

```bash
go run ./main.go
```

Or use Air for hot-reloading (development mode):

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Start the application
air
```

## Directory Structure Explained

```
.
├── main.go                     # Application entry point
├── internal/                   # Internal code, not for external import
│   ├── config/                 # Configuration management
│   │   └── config.go           # Configuration structures and loading logic
│   ├── domain/                 # Domain models
│   │   └── user.go             # User model example
│   ├── handler/                # HTTP request handlers
│   │   ├── router.go           # Route configuration
│   │   └── user_handler.go     # User-related API handlers
│   ├── interfaces/             # Interface definitions
│   │   └── service_interface.go # Service interfaces
│   ├── middleware/             # HTTP middleware
│   │   └── auth_middleware.go  # Authentication middleware
│   └── service/                # Business logic
│       └── user_service.go     # User service implementation
├── pkg/                        # Public packages, can be imported externally
│   ├── core/                   # Core component integration
│   ├── databaseManager/        # Database connection management
│   ├── httpClient/             # HTTP client
│   ├── logger/                 # Logging management
│   ├── nacosManager/           # Nacos configuration center
│   ├── redisManager/           # Redis cache management
│   ├── utils/                  # Utility functions
│   └── websocketManager/       # WebSocket connection management
└── migrations/                 # Database migration scripts
```

## Module Descriptions

### Core Module (pkg/core)

Provides an integration layer for core infrastructure, using fx modules to integrate various foundation components:

```go
// pkg/core/module.go
var Module = fx.Options(
    databaseManager.Module,
    redisManager.Module,
    logger.Module,
    // Other infrastructure...
)
```

### Configuration Module (internal/config)

Handles application configuration, supporting loading from environment variables and files:

```go
// internal/config/config.go
type Config struct {
    Server struct {
        Port string `env:"SERVER_PORT" envDefault:"8080"`
    }
    Database struct {
        URL string `env:"DATABASE_URL" envDefault:"postgres://user:pass@localhost:5432/dbname"`
    }
    // Other configurations...
}
```

### Handler Module (internal/handler)

HTTP request handling and route setup:

```go
// internal/handler/router.go
func NewRouter(userHandler *UserHandler) *gin.Engine {
    router := gin.Default()
    
    api := router.Group("/api/v1")
    {
        api.GET("/users", userHandler.GetUsers)
        api.POST("/users", userHandler.CreateUser)
        // Other routes...
    }
    
    return router
}
```

### Service Module (internal/service)

Implements business logic, interacting with the database or other external services:

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

## How to Extend

### Adding a New API Route

1. Create a handler function:

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

2. Register it in the router:

```go
// internal/handler/router.go
api.GET("/items", exampleHandler.GetItems)
```

### Adding a New Service

1. Create a service interface:

```go
// internal/interfaces/item_service.go
type ItemService interface {
    GetItems() ([]domain.Item, error)
    AddItem(item domain.Item) error
}
```

2. Implement the service:

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

3. Register it with the fx module:

```go
// internal/service/module.go
var Module = fx.Options(
    fx.Provide(
        NewUserService,
        NewItemService, // Newly added service
    ),
)
```

## Using WebSocket

### Server Side

The WebSocket manager is integrated into the skeleton and can be used like this:

```go
// internal/handler/ws_handler.go
func (h *WSHandler) HandleWS(c *gin.Context) {
    wsManager := h.wsManager
    wsManager.ServeWS(c.Writer, c.Request, func(client *websocketManager.Client) {
        // You can save user-specific information to the client here
        client.Data = map[string]interface{}{
            "userID": c.GetString("userID"),
        }
    })
}
```

### Client Side

```javascript
const socket = new WebSocket('ws://localhost:8080/ws');

socket.onopen = function() {
    console.log('Connection established');
    socket.send(JSON.stringify({type: 'auth', token: 'your-jwt-token'}));
};

socket.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Message received:', data);
};
```

## Using Dependency Injection (fx)

The skeleton uses the fx framework for dependency injection, with the main entry point in `main.go`:

```go
func main() {
    app := fx.New(
        core.Module,        // Infrastructure layer
        config.Module,      // Configuration module
        service.Module,     // Service layer
        handler.Module,     // HTTP handling layer
    )
    
    app.Run()
}
```

This approach allows fx to automatically handle object creation and dependency injection, for example:

```go
// Define a service that needs a database connection
func NewUserService(db *gorm.DB) *UserService {
    return &UserService{db: db}
}

// Define a handler that needs a service
func NewUserHandler(service *UserService) *UserHandler {
    return &UserHandler{service: service}
}
```

fx will automatically create these objects and inject the dependencies without manual handling.

## Frequently Asked Questions

**Q: How do I connect to a different database?**  
A: Modify the database connection string in the `.env` file:
```
DATABASE_URL=postgres://user:password@localhost:5432/mydatabase
```

**Q: How do I deploy to production?**  
A: Docker is recommended. The template includes a Dockerfile, which can be used like this:
```bash
docker build -t myapp .
docker run -p 8080:8080 myapp
```

**Q: Do I need to use WebSocket?**  
A: This is optional. If you don't need real-time communication features, you can skip using the WebSocket-related components.

## Contribution Guidelines

Issues and improvement suggestions are welcome! Please fork the project first, then create a pull request.

## License

This project is licensed under the MIT License.