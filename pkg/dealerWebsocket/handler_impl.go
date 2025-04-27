package dealerWebsocket

import (
	"encoding/json"
	"g38_lottery_service/internal/service"
	"log"
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

	// 創建簡單的成功回應 {"success": true}
	simpleResponse := struct {
		Success bool `json:"success"`
	}{
		Success: true,
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
}

// handleShowLuckyNumbers 處理顯示幸運號碼命令
func (h *DealerMessageHandler) handleShowLuckyNumbers(client *Client, data interface{}) {
	log.Println("處理 SHOW_LUCKY_NUMBERS 命令")
	// 這個方法的實現將在後續加入
	response := NewErrorResponse("SHOW_LUCKY_NUMBERS 功能尚未實現")
	responseJSON, _ := json.Marshal(response)
	client.Send <- responseJSON
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
