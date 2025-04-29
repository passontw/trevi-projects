package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

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
// @Description 更改當前遊戲狀態，並觸發相應的業務邏輯
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "請求格式錯誤: " + err.Error()})
		return
	}

	// 獲取當前狀態以記錄狀態變更
	currentState := h.gameService.GetCurrentState()

	// 檢查請求的狀態是否為有效的遊戲狀態
	targetState := game.GameState(req.State)
	if !isValidGameState(targetState) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "無效的遊戲狀態: " + req.State})
		return
	}

	// 嘗試變更狀態
	err := h.gameService.ChangeState(targetState)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "狀態變更失敗: " + err.Error()})
		return
	}

	// 獲取更新後的遊戲狀態
	status := h.gameService.GetGameStatus()

	// 根據不同的目標狀態，可能需要執行額外的業務邏輯
	var message string
	switch targetState {
	case game.StateReady:
		message = "遊戲已準備就緒"
	case game.StateShowLuckyNums:
		message = "已顯示幸運號碼"
	case game.StateBetting:
		message = "投注階段已開始"
	case game.StateDrawing:
		message = "開球階段已開始"
	case game.StateExtraBet:
		message = "額外球投注階段已開始"
	case game.StateExtraDraw:
		message = "額外球抽取階段已開始"
	case game.StateResult:
		message = "結算階段已開始"
	default:
		message = "遊戲狀態已更改"
	}

	// 返回成功響應，包含更詳細的狀態信息
	c.JSON(http.StatusOK, SuccessResponse{
		Message: message,
		Data: map[string]interface{}{
			"previous_state": string(currentState),
			"current_state":  string(targetState),
			"game_id":        status.Game.ID,
			"timestamp":      time.Now().Format(time.RFC3339),
		},
	})
}

// isValidGameState 檢查狀態是否為有效的遊戲狀態
func isValidGameState(state game.GameState) bool {
	validStates := []game.GameState{
		game.StateInitial,
		game.StateReady,
		game.StateShowLuckyNums,
		game.StateBetting,
		game.StateDrawing,
		game.StateShowBalls,
		game.StateExtraBet,
		game.StateExtraDraw,
		game.StateChooseExtraBall,
		game.StateShowExtraBalls,
		game.StateResult,
		game.StateJPStandby,
		game.StateJPBetting,
		game.StateJPDrawing,
		game.StateJPResult,
		game.StateJPShowBalls,
		game.StateCompleted,
	}

	for _, validState := range validStates {
		if state == validState {
			return true
		}
	}

	return false
}

// TestGameFlow 測試完整的遊戲狀態流程
// @Summary 測試遊戲狀態流程
// @Description 按順序執行遊戲完整流程，包括各個狀態轉換，用於調試和壓力測試
// @Tags game
// @Accept json
// @Produce json
// @Param data body map[string]interface{} false "測試參數"
// @Success 200 {object} SuccessResponse "測試完成"
// @Failure 400 {object} ErrorResponse "請求錯誤"
// @Failure 500 {object} ErrorResponse "服務器錯誤"
// @Router /api/v1/game/test-flow [post]
func (h *GameHandler) TestGameFlow(c *gin.Context) {
	// 獲取開始時間
	startTime := time.Now()

	// 可選參數
	var req struct {
		SkipSteps         []string `json:"skip_steps,omitempty"`
		DelayBetweenSteps int      `json:"delay_between_steps,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果無法解析請求體，使用默認值
		req.DelayBetweenSteps = 1 // 默認延遲1秒
		req.SkipSteps = []string{}
	}

	// 如果未設置延遲，使用默認值
	if req.DelayBetweenSteps <= 0 {
		req.DelayBetweenSteps = 1
	}

	// 記錄當前狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("遊戲流程測試開始，當前狀態: %s", currentState)

	// 流程步驟
	steps := []struct {
		Name        string
		State       game.GameState
		Description string
		Action      func() error
	}{
		{
			Name:        "INITIAL",
			State:       game.StateInitial,
			Description: "初始狀態",
			Action: func() error {
				return h.gameService.ChangeState(game.StateInitial)
			},
		},
		{
			Name:        "READY",
			State:       game.StateReady,
			Description: "準備就緒",
			Action: func() error {
				return h.gameService.ChangeState(game.StateReady)
			},
		},
		{
			Name:        "SHOW_LUCKYNUMS",
			State:       game.StateShowLuckyNums,
			Description: "顯示幸運號碼",
			Action: func() error {
				// 創建新遊戲並設置幸運號碼
				_, err := h.gameService.CreateGame()
				return err
			},
		},
		{
			Name:        "BETTING",
			State:       game.StateBetting,
			Description: "投注階段",
			Action: func() error {
				return h.gameService.StartBetting()
			},
		},
		{
			Name:        "DRAWING",
			State:       game.StateDrawing,
			Description: "抽球階段",
			Action: func() error {
				return h.gameService.CloseBetting()
			},
		},
		{
			Name:        "DRAW_BALLS",
			Description: "抽取20個球",
			Action: func() error {
				for i := 0; i < 20; i++ {
					_, err := h.gameService.DrawBall()
					if err != nil {
						return fmt.Errorf("抽取第 %d 個球失敗: %v", i+1, err)
					}
					// 小延遲，模擬真實抽球過程
					time.Sleep(200 * time.Millisecond)
				}
				return nil
			},
		},
		{
			Name:        "EXTRA_BET",
			State:       game.StateExtraBet,
			Description: "額外投注階段",
			Action: func() error {
				return h.gameService.ChangeState(game.StateExtraBet)
			},
		},
		{
			Name:        "EXTRA_DRAW",
			State:       game.StateExtraDraw,
			Description: "額外球抽取階段",
			Action: func() error {
				return h.gameService.ChangeState(game.StateExtraDraw)
			},
		},
		{
			Name:        "DRAW_EXTRA_BALLS",
			Description: "抽取3個額外球",
			Action: func() error {
				for i := 0; i < 3; i++ {
					_, err := h.gameService.DrawExtraBall()
					if err != nil {
						return fmt.Errorf("抽取第 %d 個額外球失敗: %v", i+1, err)
					}
					// 小延遲，模擬真實抽球過程
					time.Sleep(500 * time.Millisecond)
				}
				return nil
			},
		},
		{
			Name:        "RESULT",
			State:       game.StateResult,
			Description: "結算階段",
			Action: func() error {
				return h.gameService.ChangeState(game.StateResult)
			},
		},
		{
			Name:        "COMPLETED",
			State:       game.StateCompleted,
			Description: "遊戲完成",
			Action: func() error {
				return h.gameService.ChangeState(game.StateCompleted)
			},
		},
		{
			Name:        "READY",
			State:       game.StateReady,
			Description: "回到準備狀態",
			Action: func() error {
				return h.gameService.ChangeState(game.StateReady)
			},
		},
	}

	// 記錄步驟結果
	results := make([]map[string]interface{}, 0)

	// 執行每個步驟
	for _, step := range steps {
		// 檢查是否需要跳過此步驟
		skip := false
		for _, skipStep := range req.SkipSteps {
			if skipStep == step.Name {
				skip = true
				break
			}
		}

		if skip {
			log.Printf("跳過步驟 [%s]: %s", step.Name, step.Description)
			results = append(results, map[string]interface{}{
				"step":      step.Name,
				"status":    "SKIPPED",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			continue
		}

		// 執行步驟
		log.Printf("執行步驟 [%s]: %s", step.Name, step.Description)
		stepStart := time.Now()
		err := step.Action()

		// 記錄結果
		status := "SUCCESS"
		var errMsg string
		if err != nil {
			status = "FAILED"
			errMsg = err.Error()
			log.Printf("步驟 [%s] 失敗: %v", step.Name, err)
		} else {
			log.Printf("步驟 [%s] 成功完成", step.Name)
		}

		results = append(results, map[string]interface{}{
			"step":       step.Name,
			"status":     status,
			"error":      errMsg,
			"duration":   time.Since(stepStart).Milliseconds(),
			"timestamp":  time.Now().Format(time.RFC3339),
			"game_state": string(h.gameService.GetCurrentState()),
		})

		// 如果有錯誤，中斷流程
		if err != nil {
			break
		}

		// 步驟間延遲
		time.Sleep(time.Duration(req.DelayBetweenSteps) * time.Second)
	}

	// 計算總耗時
	totalDuration := time.Since(startTime)

	// 返回測試結果
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "遊戲流程測試完成",
		Data: map[string]interface{}{
			"total_steps":     len(steps),
			"completed_steps": len(results),
			"total_duration":  totalDuration.Milliseconds(),
			"start_time":      startTime.Format(time.RFC3339),
			"end_time":        time.Now().Format(time.RFC3339),
			"initial_state":   string(currentState),
			"final_state":     string(h.gameService.GetCurrentState()),
			"steps":           results,
		},
	})
}
