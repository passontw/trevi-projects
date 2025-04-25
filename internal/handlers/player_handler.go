// 玩家處理程序
package handlers

import (
	"fmt"
	"g38_lottery_service/internal/services"
	"g38_lottery_service/pkg/websocketManager"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// PlayerHandler 玩家處理程序
type PlayerHandler struct {
	wsHandler     *websocketManager.DualWebSocketHandler
	gameService   services.GameService
	serverHost    string
	serverPort    int
	serverVersion string
	authFunc      func(string) (uint, error)
}

// NewPlayerHandler 創建新的玩家處理程序
func NewPlayerHandler(
	wsHandler *websocketManager.DualWebSocketHandler,
	gameService services.GameService,
	serverHost string,
	serverPort int,
	serverVersion string,
	authFunc func(string) (uint, error),
) *PlayerHandler {
	return &PlayerHandler{
		wsHandler:     wsHandler,
		gameService:   gameService,
		serverHost:    serverHost,
		serverPort:    serverPort,
		serverVersion: serverVersion,
		authFunc:      authFunc,
	}
}

// RegisterRoutes 註冊玩家路由
func (h *PlayerHandler) RegisterRoutes(router *gin.Engine) {
	// 健康檢查
	router.GET("/health", h.HandleHealthCheck)

	// 版本資訊
	router.GET("/version", h.HandleVersionInfo)

	// 遊戲狀態API
	router.GET("/game/status", h.HandleGameStatus)

	// WebSocket 連接端點
	router.GET("/ws", h.wsHandler.HandlePlayerConnection)

	// 認證端點
	router.POST("/auth", h.HandleAuth)

	// 提供Swagger靜態文件
	router.StaticFile("/swagger.json", "./docs/swagger/swagger.json")
	router.StaticFile("/swagger.yaml", "./docs/swagger/swagger.yaml")

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL(fmt.Sprintf("http://%s:%d/swagger.json", h.serverHost, h.serverPort)),
		ginSwagger.DefaultModelsExpandDepth(-1)))
}

// HandleHealthCheck 處理健康檢查請求
func (h *PlayerHandler) HandleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"type":   "player",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// HandleVersionInfo 處理版本信息請求
func (h *PlayerHandler) HandleVersionInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": h.serverVersion,
	})
}

// HandleGameStatus 處理遊戲狀態請求
func (h *PlayerHandler) HandleGameStatus(c *gin.Context) {
	gameController := h.gameService.GetGameController()
	response := gameController.GetGameStatus()
	c.JSON(http.StatusOK, response)
}

// HandleAuth 處理認證請求
func (h *PlayerHandler) HandleAuth(c *gin.Context) {
	token := c.GetHeader("Authorization")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "缺少認證令牌",
		})
		return
	}

	// 使用與初始化DualWebSocketService相同的認證函數
	userID, err := h.authFunc(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "玩家認證失敗: " + err.Error(),
		})
		return
	}

	// 返回 WebSocket 連接 URL 和用戶信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "玩家認證成功",
		"data": gin.H{
			"wsURL":  "/ws",
			"userID": userID,
			"token":  token,
		},
	})
}
