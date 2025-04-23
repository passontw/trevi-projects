package handler

import (
	"fmt"
	"net/http"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/interfaces"
	"g38_lottery_service/internal/middleware"
	"g38_lottery_service/internal/service"
	"g38_lottery_service/pkg/websocketManager"

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

type UserResponse = interfaces.User

func NewRouter(
	cfg *config.Config,
	authHandler *AuthHandler,
	userHandler *UserHandler,
	authService service.AuthService,
	wsHandler *websocketManager.WebSocketHandler,
) *gin.Engine {
	r := gin.Default()
	r.Use(configureCORS())
	r.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// @Summary      健康檢查
	// @Description  返回服務健康狀態
	// @Tags         系統資訊
	// @Accept       json
	// @Produce      json
	// @Success      200  {object}  SuccessResponse
	// @Router       /health [get]
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, SuccessResponse{Message: "Service is healthy"})
	})

	// @Summary      取得應用版本資訊
	// @Description  返回應用程式的詳細版本資訊，包括構建日期、Git提交等
	// @Tags         系統資訊
	// @Accept       json
	// @Produce      json
	// @Success      200  {object}  config.Version
	// @Router       /version [get]
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, config.GetVersion())
	})

	// @Summary      WebSocket 連接
	// @Description  建立 WebSocket 連接以接收即時通知
	// @Tags         WebSocket
	// @Accept       json
	// @Produce      json
	// @Router       /ws [get]
	r.GET("/ws", wsHandler.HandleConnection)

	apiVersion := config.GetAPIVersion()

	api := r.Group(fmt.Sprintf("/api/%s", apiVersion))
	{
		configurePublicRoutes(api, authHandler, userHandler)
		configureAuthenticatedRoutes(api, authHandler, userHandler, authService)
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

func configurePublicRoutes(api *gin.RouterGroup, authHandler *AuthHandler, userHandler *UserHandler) {
	api.POST("/auth", authHandler.UserLogin)
	api.POST("/users", userHandler.CreateUser)
}

func configureAuthenticatedRoutes(api *gin.RouterGroup, authHandler *AuthHandler, userHandler *UserHandler, authService service.AuthService) {
	authorized := api.Group("/")
	authorized.Use(middleware.AuthMiddleware(authService))

	authorized.POST("/auth/logout", authHandler.UserLogout)
	authorized.POST("/auth/token", authHandler.ValidateToken)
	authorized.GET("/users", userHandler.GetUsers)
}

func StartServer(cfg *config.Config, router *gin.Engine) {
	vInfo := config.GetVersion()
	fmt.Printf("\n======================================\n")
	fmt.Printf("  %s v%s\n", vInfo.AppName, vInfo.Version)
	fmt.Printf("  環境: %s\n", vInfo.BuildEnv)
	fmt.Printf("  API 版本: %s\n", config.GetAPIVersion())
	fmt.Printf("======================================\n\n")

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("服務啟動在 http://localhost%s\n", addr)
	fmt.Printf("API 文檔: http://localhost%s/api-docs/index.html\n", addr)
	fmt.Printf("健康檢查: http://localhost%s/health\n", addr)
	fmt.Printf("版本信息: http://localhost%s/version\n\n", addr)

	router.Run(addr)
}
