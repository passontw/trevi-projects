package dealerWebsocket

import (
	"fmt"
	"net/http"
	"time"

	"g38_lottery_service/pkg/core/service"

	"go.uber.org/zap"
)

// WebSocketHandler 處理 WebSocket 連接請求
type WebSocketHandler struct {
	manager        *Manager
	tokenValidator TokenValidator
	logger         *zap.Logger
}

// NewWebSocketHandler 創建新的 WebSocket 處理程序
func NewWebSocketHandler(manager *Manager, tokenValidator TokenValidator) *WebSocketHandler {
	return &WebSocketHandler{
		manager:        manager,
		tokenValidator: tokenValidator,
		logger:         zap.L().Named("websocket_handler"),
	}
}

// HandleDealerConnection 處理荷官連接
func (h *WebSocketHandler) HandleDealerConnection(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Handling dealer connection request")
	h.manager.HandleConnection(w, r, true)
}

// HandlePlayerConnection 處理玩家連接
func (h *WebSocketHandler) HandlePlayerConnection(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Handling player connection request")
	h.manager.HandleConnection(w, r, false)
}

// DealerMessageHandler 實現荷官消息處理
type DealerMessageHandler struct {
	gameService service.GameService
	logger      *zap.Logger
}

// NewDealerMessageHandler 創建新的荷官消息處理器
func NewDealerMessageHandler(gameService service.GameService) *DealerMessageHandler {
	return &DealerMessageHandler{
		gameService: gameService,
		logger:      zap.L().Named("dealer_message_handler"),
	}
}

// Handle 處理荷官端發送的消息
func (h *DealerMessageHandler) Handle(client *Client, message Message) error {
	// 記錄收到的消息
	h.logger.Info("Received dealer message",
		zap.String("clientID", client.ID),
		zap.String("type", message.Type))

	// 根據消息類型處理
	switch message.Type {
	case "START_GAME":
		return h.handleStartGame(client, message)
	case "END_GAME":
		return h.handleEndGame(client, message)
	case "DRAW_BALL":
		return h.handleDrawBall(client, message)
	case "SELECT_SIDE":
		return h.handleSelectSide(client, message)
	default:
		// 返回未知消息類型的錯誤
		return client.SendJSON(Response{
			Type: "error",
			Payload: map[string]interface{}{
				"message":   fmt.Sprintf("未知消息類型: %s", message.Type),
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
	}
}

// handleStartGame 處理開始游戲請求
func (h *DealerMessageHandler) handleStartGame(client *Client, message Message) error {
	h.logger.Info("Handling START_GAME request", zap.String("clientID", client.ID))

	// 這裡會調用 gameService 的方法來開始遊戲
	// 實際實現可能需要從 message.Payload 中提取參數

	// 模擬開始遊戲
	gameID := fmt.Sprintf("game-%d", time.Now().Unix())

	return client.SendJSON(Response{
		Type: "response",
		Payload: map[string]interface{}{
			"type":      "START_GAME",
			"success":   true,
			"game_id":   gameID,
			"message":   "遊戲已開始",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}

// handleEndGame 處理結束遊戲請求
func (h *DealerMessageHandler) handleEndGame(client *Client, message Message) error {
	h.logger.Info("Handling END_GAME request", zap.String("clientID", client.ID))

	// 從消息中獲取遊戲ID
	gameID, ok := message.Payload["game_id"].(string)
	if !ok {
		return client.SendJSON(Response{
			Type: "response",
			Payload: map[string]interface{}{
				"type":      "END_GAME",
				"success":   false,
				"message":   "缺少遊戲ID",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
	}

	// 這裡會調用 gameService 的方法來結束遊戲

	return client.SendJSON(Response{
		Type: "response",
		Payload: map[string]interface{}{
			"type":      "END_GAME",
			"success":   true,
			"game_id":   gameID,
			"message":   "遊戲已結束",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}

// handleDrawBall 處理抽球請求
func (h *DealerMessageHandler) handleDrawBall(client *Client, message Message) error {
	h.logger.Info("Handling DRAW_BALL request", zap.String("clientID", client.ID))

	// 從消息中獲取遊戲ID
	gameID, ok := message.Payload["game_id"].(string)
	if !ok {
		return client.SendJSON(Response{
			Type: "response",
			Payload: map[string]interface{}{
				"type":      "DRAW_BALL",
				"success":   false,
				"message":   "缺少遊戲ID",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
	}

	// 這裡會調用 gameService 的方法來抽球
	// 模擬抽球結果
	ballNumber := fmt.Sprintf("%d", time.Now().Unix()%80+1)

	return client.SendJSON(Response{
		Type: "response",
		Payload: map[string]interface{}{
			"type":      "DRAW_BALL",
			"success":   true,
			"game_id":   gameID,
			"ball":      ballNumber,
			"ball_type": "regular",
			"draw_time": time.Now().Format(time.RFC3339),
			"message":   fmt.Sprintf("成功抽取球號: %s", ballNumber),
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}

// handleSelectSide 處理選邊請求
func (h *DealerMessageHandler) handleSelectSide(client *Client, message Message) error {
	h.logger.Info("Handling SELECT_SIDE request", zap.String("clientID", client.ID))

	// 從消息中獲取遊戲ID
	gameID, ok := message.Payload["game_id"].(string)
	if !ok {
		return client.SendJSON(Response{
			Type: "response",
			Payload: map[string]interface{}{
				"type":      "SELECT_SIDE",
				"success":   false,
				"message":   "缺少遊戲ID",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
	}

	// 從消息中獲取選擇的邊
	side, ok := message.Payload["side"].(string)
	if !ok {
		return client.SendJSON(Response{
			Type: "response",
			Payload: map[string]interface{}{
				"type":      "SELECT_SIDE",
				"success":   false,
				"message":   "缺少選邊信息",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
	}

	// 這裡會調用 gameService 的方法來處理選邊

	return client.SendJSON(Response{
		Type: "response",
		Payload: map[string]interface{}{
			"type":        "SELECT_SIDE",
			"success":     true,
			"game_id":     gameID,
			"side":        side,
			"select_time": time.Now().Format(time.RFC3339),
			"message":     fmt.Sprintf("成功選擇: %s", side),
			"timestamp":   time.Now().Format(time.RFC3339),
		},
	})
}
