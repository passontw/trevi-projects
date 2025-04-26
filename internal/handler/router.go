package handler

import (
	"fmt"
	"log"
	"net/http"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/pkg/dealerWebsocket"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type SuccessResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewRouter(
	cfg *config.Config,
	gameHandler *GameHandler,
	wsHandler *dealerWebsocket.WebSocketHandler,
) *gin.Engine {
	r := gin.Default()
	r.Use(configureCORS())
	r.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, SuccessResponse{Message: "Service is healthy"})
	})

	r.GET("/ws", func(c *gin.Context) {
		wsHandler.HandleWebSocket(c.Writer, c.Request)
	})

	api := r.Group("/api/v1")
	{
		configurePublicRoutes(api, gameHandler)
		configureAuthenticatedRoutes(api, gameHandler)
	}

	return r
}

func configureCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func configurePublicRoutes(api *gin.RouterGroup, gameHandler *GameHandler) {
	api.GET("/game/status", gameHandler.GetGameStatus)
	api.GET("/game/state", gameHandler.GetGameState)
}

func configureAuthenticatedRoutes(api *gin.RouterGroup, gameHandler *GameHandler) {
	authorized := api.Group("/")

	authorized.POST("/game/state", gameHandler.ChangeGameState)
}
func StartServer(cfg *config.Config, router *gin.Engine) {
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("正在使用端口 %d 啟動 API 服務器...\n", cfg.Server.Port)

	// 將 Gin 服務器的啟動放在單獨的 goroutine 中，避免阻塞 FX 生命週期
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("無法啟動 API 服務器: %v", err)
		}
	}()
}
