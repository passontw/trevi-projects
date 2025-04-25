package handler

import (
	"net/http"

	"g38_lottery_service/game"
	"g38_lottery_service/internal/service"

	"github.com/gin-gonic/gin"
)

// GameHandler 處理遊戲相關請求
type GameHandler struct {
	gameService service.GameService
}

// NewGameHandler 創建一個新的遊戲處理器
func NewGameHandler(gameService service.GameService) *GameHandler {
	return &GameHandler{
		gameService: gameService,
	}
}

// GetGameStatus 獲取遊戲狀態
// @Summary 獲取遊戲狀態
// @Description 返回當前遊戲的狀態信息
// @Tags game
// @Accept json
// @Produce json
// @Success 200 {object} game.GameStatusResponse "遊戲狀態"
// @Failure 500 {object} ErrorResponse "服務器錯誤"
// @Router /api/v1/game/status [get]
func (h *GameHandler) GetGameStatus(c *gin.Context) {
	status := h.gameService.GetGameStatus()
	c.JSON(http.StatusOK, status)
}

// GetGameState 獲取遊戲狀態
// @Summary 獲取遊戲狀態字符串
// @Description 返回當前遊戲的狀態字符串
// @Tags game
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "遊戲狀態"
// @Failure 500 {object} ErrorResponse "服務器錯誤"
// @Router /api/v1/game/state [get]
func (h *GameHandler) GetGameState(c *gin.Context) {
	state := h.gameService.GetCurrentState()
	c.JSON(http.StatusOK, gin.H{"state": string(state)})
}

// ChangeGameState 更改遊戲狀態
// @Summary 更改遊戲狀態
// @Description 更改當前遊戲狀態
// @Tags game
// @Accept json
// @Produce json
// @Param data body map[string]string true "狀態信息"
// @Success 200 {object} SuccessResponse "狀態更改成功"
// @Failure 400 {object} ErrorResponse "請求錯誤"
// @Failure 500 {object} ErrorResponse "服務器錯誤"
// @Router /api/v1/game/state [post]
func (h *GameHandler) ChangeGameState(c *gin.Context) {
	var req struct {
		State string `json:"state" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.gameService.ChangeState(game.GameState(req.State))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "遊戲狀態已更改"})
}
