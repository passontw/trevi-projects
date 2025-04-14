package gateway

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

// Server API 網關服務器
type Server struct {
	router     *gin.Engine
	redis      *redis.Client
	wsUpgrader websocket.Upgrader
	// 服務客戶端連接
	userClient     *grpc.ClientConn
	productClient  *grpc.ClientConn
	merchantClient *grpc.ClientConn
	orderClient    *grpc.ClientConn
	cartClient     *grpc.ClientConn
	authClient     *grpc.ClientConn
}

// NewServer 創建一個新的 API 網關服務器
func NewServer(router *gin.Engine, redis *redis.Client, wsUpgrader websocket.Upgrader) *Server {
	return &Server{
		router:     router,
		redis:      redis,
		wsUpgrader: wsUpgrader,
	}
}

// SetupRoutes 設置 API 路由
func (s *Server) SetupRoutes() {
	// 健康檢查
	s.router.GET("/health", s.healthCheck)

	// API 版本
	v1 := s.router.Group("/api/v1")

	// 用戶相關路由
	users := v1.Group("/users")
	{
		users.POST("/", s.createUser)
		users.GET("/", s.listUsers)
		users.GET("/:id", s.getUserByID)
		users.PUT("/:id", s.updateUser)
		users.DELETE("/:id", s.deleteUser)
	}

	// 認證相關路由
	auth := v1.Group("/auth")
	{
		auth.POST("/login", s.login)
		auth.POST("/register", s.register)
		auth.POST("/refresh", s.refreshToken)
	}

	// 產品相關路由
	products := v1.Group("/products")
	{
		products.GET("/", s.listProducts)
		products.GET("/:id", s.getProductByID)
		// 其他產品路由
	}

	// 購物車相關路由
	cart := v1.Group("/cart")
	{
		cart.GET("/", s.getCart)
		cart.POST("/items", s.addCartItem)
		cart.PUT("/items/:id", s.updateCartItem)
		cart.DELETE("/items/:id", s.removeCartItem)
	}

	// 訂單相關路由
	orders := v1.Group("/orders")
	{
		orders.POST("/", s.createOrder)
		orders.GET("/", s.listOrders)
		orders.GET("/:id", s.getOrderByID)
	}

	// 商家相關路由
	merchants := v1.Group("/merchants")
	{
		merchants.GET("/", s.listMerchants)
		merchants.GET("/:id", s.getMerchantByID)
		// 其他商家路由
	}

	// WebSocket 相關路由
	ws := v1.Group("/ws")
	{
		ws.GET("/notifications", s.handleNotifications)
	}
}

// 健康檢查處理器
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// 以下是處理各種 API 請求的方法，這裡僅提供簽名
// 實際實現將需要調用相應的 gRPC 服務

// 用戶相關處理器
func (s *Server) createUser(c *gin.Context) {
	// 調用用戶服務創建用戶
	c.JSON(http.StatusOK, gin.H{"message": "Create user endpoint"})
}

func (s *Server) listUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List users endpoint"})
}

func (s *Server) getUserByID(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Get user endpoint", "id": id})
}

func (s *Server) updateUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Update user endpoint", "id": id})
}

func (s *Server) deleteUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Delete user endpoint", "id": id})
}

// 認證相關處理器
func (s *Server) login(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Login endpoint"})
}

func (s *Server) register(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Register endpoint"})
}

func (s *Server) refreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Refresh token endpoint"})
}

// 產品相關處理器
func (s *Server) listProducts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List products endpoint"})
}

func (s *Server) getProductByID(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Get product endpoint", "id": id})
}

// 購物車相關處理器
func (s *Server) getCart(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get cart endpoint"})
}

func (s *Server) addCartItem(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Add cart item endpoint"})
}

func (s *Server) updateCartItem(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Update cart item endpoint", "id": id})
}

func (s *Server) removeCartItem(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Remove cart item endpoint", "id": id})
}

// 訂單相關處理器
func (s *Server) createOrder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create order endpoint"})
}

func (s *Server) listOrders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List orders endpoint"})
}

func (s *Server) getOrderByID(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Get order endpoint", "id": id})
}

// 商家相關處理器
func (s *Server) listMerchants(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List merchants endpoint"})
}

func (s *Server) getMerchantByID(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Get merchant endpoint", "id": id})
}

// WebSocket 相關處理器
func (s *Server) handleNotifications(c *gin.Context) {
	conn, err := s.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// 處理 WebSocket 連接
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// 處理收到的消息
		log.Printf("Received message: %s", message)

		// 發送回複消息
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}
