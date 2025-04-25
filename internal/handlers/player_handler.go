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
