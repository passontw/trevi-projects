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

	switch messageType {
	case MessageTypeGameStart:
		h.handleGameStart(client)
	case MessageTypeShowLuckyNumbers:
		h.handleShowLuckyNumbers(client, data)
	case MessageTypeDrawBall:
		h.handleDrawBall(client)
	case MessageTypeDrawExtraBall:
		h.handleDrawExtraBall(client)
	case MessageTypeStartJPGame:
		h.handleStartJPGame(client, data)
	case MessageTypeDrawJPBall:
		h.handleDrawJPBall(client)
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

	// 更改遊戲狀態為投注階段
	if err := h.gameService.ChangeState(game.StateBetting); err != nil {
		log.Printf("更改遊戲狀態失敗: %v", err)
		return
	}

	log.Printf("遊戲狀態已更改為投注階段: %s", game.StateBetting)
}

// handleDrawBall 處理抽球命令
func (h *DealerMessageHandler) handleDrawBall(client *Client) {
	log.Println("處理 DRAW_BALL 命令")
	// 這個方法的實現將在後續加入
	response := NewErrorResponse("DRAW_BALL 功能尚未實現")
	responseJSON, _ := json.Marshal(response)
	client.Send <- responseJSON
}

// handleDrawExtraBall 處理抽額外球命令
func (h *DealerMessageHandler) handleDrawExtraBall(client *Client) {
	log.Println("處理 DRAW_EXTRA_BALL 命令")
	// 這個方法的實現將在後續加入
	response := NewErrorResponse("DRAW_EXTRA_BALL 功能尚未實現")
	responseJSON, _ := json.Marshal(response)
	client.Send <- responseJSON
}

// handleStartJPGame 處理開始JP遊戲命令
func (h *DealerMessageHandler) handleStartJPGame(client *Client, data interface{}) {
	log.Println("處理 START_JP_GAME 命令")
	// 這個方法的實現將在後續加入
	response := NewErrorResponse("START_JP_GAME 功能尚未實現")
	responseJSON, _ := json.Marshal(response)
	client.Send <- responseJSON
}

// handleDrawJPBall 處理抽JP球命令
func (h *DealerMessageHandler) handleDrawJPBall(client *Client) {
	log.Println("處理 DRAW_JP_BALL 命令")
	// 這個方法的實現將在後續加入
	response := NewErrorResponse("DRAW_JP_BALL 功能尚未實現")
	responseJSON, _ := json.Marshal(response)
	client.Send <- responseJSON
}
