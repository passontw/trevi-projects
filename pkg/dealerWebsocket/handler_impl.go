package dealerWebsocket

import (
	"encoding/json"
	"g38_lottery_service/game"
	"g38_lottery_service/internal/service"
	"log"
	"time"
)

// DealerMessageHandler 實現 MessageHandler 接口，專門處理荷官端消息
type DealerMessageHandler struct {
	gameService service.GameService
}

// NewDealerMessageHandler 創建一個新的荷官消息處理器
func NewDealerMessageHandler(gameService service.GameService) MessageHandler {
	return &DealerMessageHandler{
		gameService: gameService,
	}
}

// HandleMessage 處理接收到的消息
func (h *DealerMessageHandler) HandleMessage(client *Client, messageType string, data interface{}) {
	log.Printf("收到荷官端消息，類型: %s", messageType)
	log.Printf("收到荷官端消息，MessageTypeStartExtraBetting 類型: %s", MessageTypeStartExtraBetting)

	switch messageType {
	case MessageTypeGameStart:
		h.handleGameStart(client)
	case MessageTypeShowLuckyNumbers:
		h.handleShowLuckyNumbers(client, data)
	case MessageTypeDrawBall:
		h.handleDrawBall(client)
	case MessageTypeDrawExtraBall:
		h.handleDrawExtraBall(client)
	case MessageTypeChooseExtraBall:
		h.handleChooseExtraBall(client)
	case MessageTypeDrawJPBall:
		h.handleDrawJPBall(client)
	case MessageTypeBettingStarted:
		h.handleBettingStarted(client, data)
	case MessageTypeBettingClosed:
		h.handleBettingClosed(client, data)
	case MessageTypeDrawResult:
		h.handleDrawResult(client, data)
	case MessageTypeStartExtraBetting:
		h.handleStartExtraBetting(client, data)
	case MessageTypeFinishExtraBetting:
		h.handleFinishExtraBetting(client, data)
	case MessageTypeStartResult:
		h.handleStartResult(client)
	case MessageTypeStartJPStandby:
		h.handleStartJPStandby(client)
	case MessageTypeStartJPBetting:
		h.handleStartJPBetting(client)
	case MessageTypeStartJPDrawing:
		h.handleStartJPDrawing(client)
	case MessageTypeStopJPDrawing:
		h.handleStopJPDrawing(client)
	case MessageTypeStartJPShowBalls:
		h.handleStartJPShowBalls(client)
	case MessageTypeStartCompleted:
		h.handleStartCompleted(client)
	default:
		log.Printf("未知消息類型: %s", messageType)
		response := NewErrorResponse("不支持的消息類型")
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
	}
}

// HandleConnect 處理客戶端連接成功事件
func (h *DealerMessageHandler) HandleConnect(client *Client) {
	log.Printf("荷官端連接成功，ID: %s", client.ID)

	// 發送歡迎消息
	welcomeMsg := NewMessage(MessageTypeSystemNotice, map[string]string{
		"message": "歡迎連接到荷官端WebSocket服務",
	})

	msgJSON, _ := welcomeMsg.ToJSON()
	client.Send <- msgJSON
}

// HandleDisconnect 處理客戶端斷開連接事件
func (h *DealerMessageHandler) HandleDisconnect(client *Client) {
	log.Printf("荷官端斷開連接，ID: %s", client.ID)
}

// handleGameStart 處理遊戲開始命令
func (h *DealerMessageHandler) handleGameStart(client *Client) {
	log.Println("處理 GAME_START 命令")

	// 呼叫遊戲服務創建新遊戲
	newGame, err := h.gameService.CreateGame()
	if err != nil {
		log.Printf("創建遊戲失敗: %v", err)
		response := NewErrorResponse("創建遊戲失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 創建簡單的成功回應
	simpleResponse := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		GameID  string `json:"game_id"`
	}{
		Success: true,
		Message: "遊戲創建成功，即將設置幸運號碼",
		GameID:  newGame.ID,
	}

	// 將簡單回應轉換為JSON
	responseJSON, err := json.Marshal(simpleResponse)
	if err != nil {
		log.Printf("序列化回應失敗: %v", err)
		return
	}

	// 發送簡單回應給客戶端
	client.Send <- responseJSON

	// 向所有連接的客戶端廣播遊戲創建消息
	broadcastMsg := NewMessage(
		"GAME_CREATED",
		map[string]interface{}{
			"game_id": newGame.ID,
			"state":   newGame.State,
		},
	)

	broadcastJSON, _ := broadcastMsg.ToJSON()
	client.manager.broadcast <- broadcastJSON

	log.Printf("遊戲 %s 創建成功，狀態: %s, has_jackpot: %v",
		newGame.ID, newGame.State, newGame.HasJackpot)

	// 自動執行幸運號碼設置邏輯
	// 因為CreateGame方法內部已經調用了SetLuckyNumbers，所以不需要再次調用
	// 我們只需要獲取當前狀態，並組裝通知發送出去
	status := h.gameService.GetGameStatus()

	// 創建通知結構 - 嚴格遵照 EventLuckyNumbersSet 格式
	notification := map[string]interface{}{
		"type": "LUCKY_NUMBERS_SET",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":             status.Game.ID,
				"state":          status.Game.State,
				"startTime":      status.Game.StartTime,
				"endTime":        status.Game.EndTime,
				"hasJackpot":     status.Game.HasJackpot,
				"extraBallCount": status.Game.ExtraBallCount,
			},
			"luckyNumbers": status.LuckyNumbers,
			"drawnBalls":   []interface{}{},
			"extraBalls":   []interface{}{},
			"jackpot": map[string]interface{}{
				"active":     false,
				"gameId":     nil,
				"amount":     nil,
				"startTime":  nil,
				"endTime":    nil,
				"drawnBalls": []interface{}{},
				"winner":     nil,
			},
			"topPlayers":     []interface{}{},
			"totalWinAmount": nil,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 將通知序列化為JSON
	notificationJSON, _ := json.Marshal(notification)

	// 廣播通知給所有連接的客戶端
	client.manager.broadcast <- notificationJSON

	log.Printf("已設置並通知幸運號碼: %v，遊戲ID: %s", status.LuckyNumbers, status.Game.ID)
}

// handleShowLuckyNumbers 處理顯示幸運號碼命令
func (h *DealerMessageHandler) handleShowLuckyNumbers(client *Client, data interface{}) {
	log.Println("處理 SHOW_LUCKY_NUMBERS 命令")

	// 抽取七個幸運號碼
	luckyNumbers := h.gameService.SetLuckyNumbers()

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()

	// 創建通知結構 - 嚴格遵照 EventLuckyNumbersSet 格式
	notification := map[string]interface{}{
		"type": "LUCKY_NUMBERS_SET",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":             status.Game.ID,
				"state":          status.Game.State,
				"startTime":      status.Game.StartTime,
				"endTime":        status.Game.EndTime,
				"hasJackpot":     status.Game.HasJackpot,
				"extraBallCount": status.Game.ExtraBallCount,
			},
			"luckyNumbers": luckyNumbers,
			"drawnBalls":   []interface{}{},
			"extraBalls":   []interface{}{},
			"jackpot": map[string]interface{}{
				"active":     false,
				"gameId":     nil,
				"amount":     nil,
				"startTime":  nil,
				"endTime":    nil,
				"drawnBalls": []interface{}{},
				"winner":     nil,
			},
			"topPlayers":     []interface{}{},
			"totalWinAmount": nil,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 將通知序列化為JSON
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		log.Printf("序列化幸運號碼通知失敗: %v", err)
		response := NewErrorResponse("序列化幸運號碼通知失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 向當前客戶端發送回應
	client.Send <- notificationJSON

	// 廣播通知給所有連接的客戶端
	client.manager.broadcast <- notificationJSON

	log.Printf("已設置並通知幸運號碼: %v，遊戲ID: %s", luckyNumbers, status.Game.ID)

	// 記錄當前狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("顯示幸運號碼後的當前遊戲狀態: %s", currentState)

	// 如果不是在投注階段，則自動開始投注階段
	if currentState != game.StateBetting {
		// 使用專用方法開始投注階段
		if err := h.gameService.StartBetting(); err != nil {
			log.Printf("開始投注階段失敗: %v", err)
			return
		}

		// 驗證狀態是否已更改
		newState := h.gameService.GetCurrentState()
		log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

		// 獲取最新的遊戲狀態
		updatedStatus := h.gameService.GetGameStatus()

		// 廣播遊戲狀態變更事件給所有客戶端
		broadcastMsg := map[string]interface{}{
			"type": "GAME_STATE_CHANGED",
			"data": map[string]interface{}{
				"game": map[string]interface{}{
					"id":         updatedStatus.Game.ID,
					"state":      updatedStatus.Game.State,
					"hasJackpot": updatedStatus.Game.HasJackpot,
					"startTime":  time.Now().Format(time.RFC3339),
				},
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}

		broadcastJSON, _ := json.Marshal(broadcastMsg)
		client.manager.broadcast <- broadcastJSON

		log.Printf("已自動開始投注階段並廣播狀態變更")
	}
}

// handleDrawBall 處理抽球命令
func (h *DealerMessageHandler) handleDrawBall(client *Client) {
	log.Println("處理 DRAW_BALL 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("抽球前的遊戲狀態: %s", currentState)

	// 檢查當前狀態是否允許抽球
	if currentState != game.StateDrawing {
		log.Printf("當前狀態 %s 不允許抽球", currentState)
		response := NewErrorResponse("當前遊戲狀態不允許抽球，目前狀態: " + string(currentState))
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 嘗試抽球
	result, err := h.gameService.DrawBall()
	if err != nil {
		log.Printf("抽球失敗: %v", err)
		response := NewErrorResponse("抽球失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	log.Printf("成功抽出球: %d (順序: %d)", result.BallNumber, result.OrderIndex)

	// 獲取當前遊戲狀態和已抽出的球
	status := h.gameService.GetGameStatus()
	drawnBalls := h.gameService.GetDrawnBalls()

	// 創建抽球結果通知
	notification := map[string]interface{}{
		"type": "BALL_DRAWN",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":             status.Game.ID,
				"state":          status.Game.State,
				"hasJackpot":     status.Game.HasJackpot,
				"extraBallCount": status.Game.ExtraBallCount,
			},
			"ball": map[string]interface{}{
				"number":   result.BallNumber,
				"drawTime": result.DrawTime.Format(time.RFC3339),
				"sequence": result.OrderIndex,
			},
			"totalDrawn": len(drawnBalls),
			"drawnBalls": convertDrawResultsToBallInfo(drawnBalls),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 將通知序列化為JSON
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		log.Printf("序列化抽球結果通知失敗: %v", err)
		response := NewErrorResponse("序列化抽球結果通知失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 向當前客戶端發送響應
	client.Send <- notificationJSON

	// 廣播通知給所有連接的客戶端
	client.manager.broadcast <- notificationJSON

	log.Printf("已通知抽球結果: 球號 %d, 遊戲ID: %s, 總計已抽出 %d 個球",
		result.BallNumber, status.Game.ID, len(drawnBalls))

	// 創建成功回應
	response := NewSuccessResponse("DRAW_BALL_RESPONSE", "抽球成功", map[string]interface{}{
		"ball_number": result.BallNumber,
		"sequence":    result.OrderIndex,
		"draw_time":   result.DrawTime.Format(time.RFC3339),
		"total_drawn": len(drawnBalls),
	})

	// 將回應序列化為JSON
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON
}

// convertDrawResultsToBallInfo 將DrawResult數組轉換為前端格式的球信息
func convertDrawResultsToBallInfo(results []game.DrawResult) []map[string]interface{} {
	balls := make([]map[string]interface{}, len(results))
	for i, result := range results {
		balls[i] = map[string]interface{}{
			"number":   result.BallNumber,
			"drawTime": result.DrawTime.Format(time.RFC3339),
			"sequence": result.OrderIndex,
		}
	}
	return balls
}

// handleDrawExtraBall 處理抽額外球命令
func (h *DealerMessageHandler) handleDrawExtraBall(client *Client) {
	log.Println("處理 DRAW_EXTRA_BALL 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("抽額外球前的遊戲狀態: %s", currentState)

	// 檢查當前狀態是否允許抽額外球
	if currentState != game.StateExtraDraw && currentState != game.StateChooseExtraBall {
		log.Printf("當前狀態 %s 不允許抽額外球", currentState)
		response := NewErrorResponse("當前遊戲狀態不允許抽額外球，目前狀態: " + string(currentState))
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 嘗試抽額外球
	result, err := h.gameService.DrawExtraBall()
	if err != nil {
		log.Printf("抽額外球失敗: %v", err)
		response := NewErrorResponse("抽額外球失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	log.Printf("成功抽出額外球: %d (順序: %d)", result.BallNumber, result.OrderIndex)

	// 獲取當前遊戲狀態和已抽出的額外球
	status := h.gameService.GetGameStatus()
	extraBalls := h.gameService.GetExtraBalls()

	// 確定側邊位置 - 依據順序
	side := "LEFT"
	if result.OrderIndex%2 == 0 {
		side = "RIGHT"
	}

	// 創建抽球結果通知
	notification := map[string]interface{}{
		"type": "EXTRA_BALL_DRAWN",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":             status.Game.ID,
				"state":          status.Game.State,
				"hasJackpot":     status.Game.HasJackpot,
				"extraBallCount": status.Game.ExtraBallCount,
			},
			"ball": map[string]interface{}{
				"number":   result.BallNumber,
				"drawTime": result.DrawTime.Format(time.RFC3339),
				"sequence": result.OrderIndex,
				"side":     side,
			},
			"totalDrawn": len(extraBalls),
			"extraBalls": convertDrawResultsToExtraBallInfo(extraBalls),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 將通知序列化為JSON
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		log.Printf("序列化額外球結果通知失敗: %v", err)
		response := NewErrorResponse("序列化額外球結果通知失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 向當前客戶端發送響應
	client.Send <- notificationJSON

	// 廣播通知給所有連接的客戶端
	client.manager.broadcast <- notificationJSON

	log.Printf("已通知額外球結果: 球號 %d, 遊戲ID: %s, 總計已抽出 %d 個額外球",
		result.BallNumber, status.Game.ID, len(extraBalls))

	// 創建成功回應
	response := NewSuccessResponse("DRAW_EXTRA_BALL_RESPONSE", "抽額外球成功", map[string]interface{}{
		"ball_number": result.BallNumber,
		"sequence":    result.OrderIndex,
		"draw_time":   result.DrawTime.Format(time.RFC3339),
		"side":        side,
		"total_drawn": len(extraBalls),
	})

	// 將回應序列化為JSON
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	// 如果已經抽完所有額外球，自動進入結算階段
	if len(extraBalls) >= status.Game.ExtraBallCount {
		log.Printf("已抽完所有 %d 個額外球，準備進入結算階段", status.Game.ExtraBallCount)

		// 嘗試變更狀態為結算階段
		if err := h.gameService.ChangeState(game.StateResult); err != nil {
			log.Printf("更改為結算階段失敗: %v", err)
		} else {
			// 獲取最新遊戲狀態
			updatedStatus := h.gameService.GetGameStatus()

			// 廣播遊戲狀態變更事件
			broadcastMsg := map[string]interface{}{
				"type": "GAME_STATE_CHANGED",
				"data": map[string]interface{}{
					"game": map[string]interface{}{
						"id":         updatedStatus.Game.ID,
						"state":      updatedStatus.Game.State,
						"hasJackpot": updatedStatus.Game.HasJackpot,
						"startTime":  time.Now().Format(time.RFC3339),
					},
				},
				"timestamp": time.Now().Format(time.RFC3339),
			}

			broadcastJSON, _ := json.Marshal(broadcastMsg)
			client.manager.broadcast <- broadcastJSON

			log.Printf("已自動進入結算階段並廣播狀態變更")
		}
	}
}

// convertDrawResultsToExtraBallInfo 將DrawResult數組轉換為前端格式的額外球信息
func convertDrawResultsToExtraBallInfo(results []game.DrawResult) []map[string]interface{} {
	balls := make([]map[string]interface{}, len(results))
	for i, result := range results {
		// 確定側邊位置 - 依據順序
		side := "LEFT"
		if result.OrderIndex%2 == 0 {
			side = "RIGHT"
		}

		balls[i] = map[string]interface{}{
			"number":   result.BallNumber,
			"drawTime": result.DrawTime.Format(time.RFC3339),
			"sequence": result.OrderIndex,
			"side":     side,
		}
	}
	return balls
}

// handleChooseExtraBall 處理選擇額外球命令
func (h *DealerMessageHandler) handleChooseExtraBall(client *Client) {
	log.Println("處理 STATE_CHOOSE_EXTRA_BALL 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("選擇額外球前的遊戲狀態: %s", currentState)

	// 嘗試切換到選擇額外球狀態
	err := h.gameService.ChooseExtraBall()
	if err != nil {
		log.Printf("切換到選擇額外球狀態失敗: %v", err)
		response := NewErrorResponse("切換到選擇額外球狀態失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("STATE_CHOOSE_EXTRA_BALL_RESPONSE", "已進入選擇額外球狀態", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送STATE_CHOOSE_EXTRA_BALL_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleDrawJPBall 處理抽JP球命令
func (h *DealerMessageHandler) handleDrawJPBall(client *Client) {
	log.Println("處理 DRAW_JP_BALL 命令")
	// 這個方法的實現將在後續加入
	response := NewErrorResponse("DRAW_JP_BALL 功能尚未實現")
	responseJSON, _ := json.Marshal(response)
	client.Send <- responseJSON
}

// handleBettingStarted 處理投注開始命令
func (h *DealerMessageHandler) handleBettingStarted(client *Client, data interface{}) {
	log.Println("處理 BETTING_STARTED 命令")

	// 輸出收到的資料，幫助調試
	jsonData, _ := json.Marshal(data)
	log.Printf("收到BETTING_STARTED命令資料: %s", string(jsonData))

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("處理BETTING_STARTED前的遊戲狀態: %s", currentState)

	// 使用專用方法開始投注階段
	err := h.gameService.StartBetting()
	if err != nil {
		log.Printf("開始投注階段失敗: %v", err)
		response := NewErrorResponse("開始投注階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("BETTING_STARTED_RESPONSE", "投注階段已開始", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送BETTING_STARTED_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleBettingClosed 處理投注關閉命令
func (h *DealerMessageHandler) handleBettingClosed(client *Client, data interface{}) {
	log.Println("處理 BETTING_CLOSED 命令")

	// 輸出收到的資料，幫助調試
	jsonData, _ := json.Marshal(data)
	log.Printf("收到BETTING_CLOSED命令資料: %s", string(jsonData))

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("處理BETTING_CLOSED前的遊戲狀態: %s", currentState)

	// 使用專用方法關閉投注並進入抽球階段
	err := h.gameService.CloseBetting()
	if err != nil {
		log.Printf("關閉投注階段失敗: %v", err)
		response := NewErrorResponse("關閉投注階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("BETTING_CLOSED_RESPONSE", "投注階段已關閉，進入開球階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送BETTING_CLOSED_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleDrawResult 處理開獎結果命令
func (h *DealerMessageHandler) handleDrawResult(client *Client, data interface{}) {
	log.Println("處理 DRAW_RESULT 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("處理DRAW_RESULT前的遊戲狀態: %s", currentState)

	// 檢查當前狀態是否允許處理開獎結果
	if currentState != game.StateResult {
		log.Printf("當前狀態 %s 不允許處理開獎結果", currentState)
		response := NewErrorResponse("當前遊戲狀態不允許處理開獎結果，目前狀態: " + string(currentState))
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	drawnBalls := h.gameService.GetDrawnBalls()
	extraBalls := h.gameService.GetExtraBalls()
	luckyNumbers := h.gameService.GetLuckyNumbers()

	// 創建開獎結果通知
	notification := map[string]interface{}{
		"type": "DRAW_RESULT",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":             status.Game.ID,
				"state":          status.Game.State,
				"hasJackpot":     status.Game.HasJackpot,
				"extraBallCount": status.Game.ExtraBallCount,
				"endTime":        time.Now().Format(time.RFC3339),
			},
			"luckyNumbers":   luckyNumbers,
			"drawnBalls":     convertDrawResultsToBallInfo(drawnBalls),
			"extraBalls":     convertDrawResultsToExtraBallInfo(extraBalls),
			"topPlayers":     []interface{}{}, // 這裡可以添加獲取頂級玩家的邏輯
			"totalWinAmount": nil,             // 這裡可以添加計算總獲勝金額的邏輯
			"jackpot": map[string]interface{}{
				"active":     status.Game.HasJackpot,
				"gameId":     status.Game.ID,
				"amount":     nil, // 添加相應的JP數據
				"startTime":  nil,
				"endTime":    nil,
				"drawnBalls": []interface{}{},
				"winner":     nil,
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 將通知序列化為JSON
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		log.Printf("序列化開獎結果通知失敗: %v", err)
		response := NewErrorResponse("序列化開獎結果通知失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 向當前客戶端發送響應
	client.Send <- notificationJSON

	// 廣播通知給所有連接的客戶端
	client.manager.broadcast <- notificationJSON

	log.Printf("已通知開獎結果: 遊戲ID: %s, 狀態: %s",
		status.Game.ID, status.Game.State)

	// 創建成功回應
	response := NewSuccessResponse("DRAW_RESULT_RESPONSE", "開獎結果已發送", map[string]interface{}{
		"game_id":       status.Game.ID,
		"state":         status.Game.State,
		"lucky_numbers": luckyNumbers,
		"drawn_balls":   len(drawnBalls),
		"extra_balls":   len(extraBalls),
	})

	// 將回應序列化為JSON
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	// 完成遊戲後，準備下一場遊戲
	if err := h.gameService.ChangeState(game.StateStandby); err != nil {
		log.Printf("更改為空閒狀態失敗: %v", err)
	} else {
		log.Printf("遊戲已結束，進入空閒狀態等待下一場遊戲開始")
	}
}

// handleStartExtraBetting 處理開始額外球投注命令
func (h *DealerMessageHandler) handleStartExtraBetting(client *Client, data interface{}) {
	log.Println("處理 START_EXTRA_BETTING 命令")

	// 輸出收到的資料，幫助調試
	jsonData, _ := json.Marshal(data)
	log.Printf("收到START_EXTRA_BETTING命令資料: %s", string(jsonData))

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("處理START_EXTRA_BETTING前的遊戲狀態: %s", currentState)

	// 使用專用方法開始額外球投注階段
	err := h.gameService.StartExtraBetting()
	if err != nil {
		log.Printf("開始額外球投注階段失敗: %v", err)
		response := NewErrorResponse("開始額外球投注階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("START_EXTRA_BETTING_RESPONSE", "額外球投注階段已開始", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送START_EXTRA_BETTING_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleFinishExtraBetting 處理結束額外球投注命令
func (h *DealerMessageHandler) handleFinishExtraBetting(client *Client, data interface{}) {
	log.Println("處理 FINISH_EXTRA_BETTING 命令")

	// 輸出收到的資料，幫助調試
	jsonData, _ := json.Marshal(data)
	log.Printf("收到FINISH_EXTRA_BETTING命令資料: %s", string(jsonData))

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("處理FINISH_EXTRA_BETTING前的遊戲狀態: %s", currentState)

	// 使用專用方法結束額外球投注階段
	err := h.gameService.FinishExtraBetting()
	if err != nil {
		log.Printf("結束額外球投注階段失敗: %v", err)
		response := NewErrorResponse("結束額外球投注階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("FINISH_EXTRA_BETTING_RESPONSE", "額外球投注階段已結束，進入額外球抽取階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送FINISH_EXTRA_BETTING_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleStartResult 處理進入結算階段命令
func (h *DealerMessageHandler) handleStartResult(client *Client) {
	log.Println("處理 START_RESULT 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("進入結算階段前的遊戲狀態: %s", currentState)

	// 嘗試進入結算階段
	err := h.gameService.StartResult()
	if err != nil {
		log.Printf("進入結算階段失敗: %v", err)
		response := NewErrorResponse("進入結算階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("START_RESULT_RESPONSE", "已進入結算階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送START_RESULT_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleStartJPStandby 處理進入JP待機階段命令
func (h *DealerMessageHandler) handleStartJPStandby(client *Client) {
	log.Println("處理 START_JP_STANDBY 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("進入JP待機階段前的遊戲狀態: %s", currentState)

	// 嘗試進入JP待機階段
	err := h.gameService.StartJPStandby()
	if err != nil {
		log.Printf("進入JP待機階段失敗: %v", err)
		response := NewErrorResponse("進入JP待機階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("START_JP_STANDBY_RESPONSE", "已進入JP待機階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送START_JP_STANDBY_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleStartJPBetting 處理進入JP投注階段命令
func (h *DealerMessageHandler) handleStartJPBetting(client *Client) {
	log.Println("處理 START_JP_BETTING 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("進入JP投注階段前的遊戲狀態: %s", currentState)

	// 嘗試進入JP投注階段
	err := h.gameService.StartJPBetting()
	if err != nil {
		log.Printf("進入JP投注階段失敗: %v", err)
		response := NewErrorResponse("進入JP投注階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("START_JP_BETTING_RESPONSE", "已進入JP投注階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送START_JP_BETTING_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleStartJPDrawing 處理進入JP抽球階段命令
func (h *DealerMessageHandler) handleStartJPDrawing(client *Client) {
	log.Println("處理 START_JP_DRAWING 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("進入JP抽球階段前的遊戲狀態: %s", currentState)

	// 嘗試進入JP抽球階段
	err := h.gameService.StartJPDrawing()
	if err != nil {
		log.Printf("進入JP抽球階段失敗: %v", err)
		response := NewErrorResponse("進入JP抽球階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("START_JP_DRAWING_RESPONSE", "已進入JP抽球階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送START_JP_DRAWING_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleStopJPDrawing 處理結束JP抽球並進入JP結果階段命令
func (h *DealerMessageHandler) handleStopJPDrawing(client *Client) {
	log.Println("處理 STOP_JP_DRAWING 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("進入JP結果階段前的遊戲狀態: %s", currentState)

	// 嘗試進入JP結果階段
	err := h.gameService.StopJPDrawing()
	if err != nil {
		log.Printf("進入JP結果階段失敗: %v", err)
		response := NewErrorResponse("進入JP結果階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("STOP_JP_DRAWING_RESPONSE", "已進入JP結果階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送STOP_JP_DRAWING_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleStartJPShowBalls 處理進入JP開獎階段命令
func (h *DealerMessageHandler) handleStartJPShowBalls(client *Client) {
	log.Println("處理 START_JP_SHOW_BALLS 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("進入JP開獎階段前的遊戲狀態: %s", currentState)

	// 嘗試進入JP開獎階段
	err := h.gameService.StartJPShowBalls()
	if err != nil {
		log.Printf("進入JP開獎階段失敗: %v", err)
		response := NewErrorResponse("進入JP開獎階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("START_JP_SHOW_BALLS_RESPONSE", "已進入JP開獎階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送START_JP_SHOW_BALLS_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}

// handleStartCompleted 處理進入遊戲完成階段命令
func (h *DealerMessageHandler) handleStartCompleted(client *Client) {
	log.Println("處理 START_COMPLETED 命令")

	// 獲取當前遊戲狀態
	currentState := h.gameService.GetCurrentState()
	log.Printf("進入遊戲完成階段前的遊戲狀態: %s", currentState)

	// 嘗試進入遊戲完成階段
	err := h.gameService.StartCompleted()
	if err != nil {
		log.Printf("進入遊戲完成階段失敗: %v", err)
		response := NewErrorResponse("進入遊戲完成階段失敗: " + err.Error())
		responseJSON, _ := json.Marshal(response)
		client.Send <- responseJSON
		return
	}

	// 驗證狀態是否已更改
	newState := h.gameService.GetCurrentState()
	log.Printf("遊戲狀態已更改: %s -> %s", currentState, newState)

	// 獲取當前遊戲狀態
	status := h.gameService.GetGameStatus()
	log.Printf("當前遊戲狀態: ID=%s, State=%s, HasJackpot=%v",
		status.Game.ID, status.Game.State, status.Game.HasJackpot)

	// 創建成功回應
	response := NewSuccessResponse("START_COMPLETED_RESPONSE", "已進入遊戲完成階段", map[string]interface{}{
		"game_id":    status.Game.ID,
		"state":      status.Game.State,
		"hasJackpot": status.Game.HasJackpot,
	})

	// 序列化回應
	responseJSON, _ := json.Marshal(response)

	// 發送回應給客戶端
	client.Send <- responseJSON

	log.Printf("已發送START_COMPLETED_RESPONSE回應")

	// 廣播遊戲狀態變更事件給所有客戶端
	broadcastMsg := map[string]interface{}{
		"type": "GAME_STATE_CHANGED",
		"data": map[string]interface{}{
			"game": map[string]interface{}{
				"id":         status.Game.ID,
				"state":      status.Game.State,
				"hasJackpot": status.Game.HasJackpot,
				"startTime":  time.Now().Format(time.RFC3339),
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	broadcastJSON, _ := json.Marshal(broadcastMsg)
	client.manager.broadcast <- broadcastJSON

	log.Printf("已廣播遊戲狀態變更事件")
}
