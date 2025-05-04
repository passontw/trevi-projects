package dealerWebsocket

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// GetGorilla 返回 gorilla websocket 包
func GetGorilla() *websocket.Upgrader {
	return &upgrader
}

// Broadcast 是 Manager 的 Broadcast 方法的包裝函數
func (m *Manager) Broadcast(message interface{}) error {
	// 轉換為 Response 結構
	resp := Response{
		Type:    "broadcast",
		Payload: message,
	}

	data, err := m.marshalJSON(resp)
	if err != nil {
		return err
	}

	m.broadcast <- data
	return nil
}

// marshalJSON 封裝 JSON 序列化邏輯
func (m *Manager) marshalJSON(message interface{}) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		m.logger.Error("Error marshaling message", zap.Error(err))
		return nil, err
	}
	return data, nil
}
