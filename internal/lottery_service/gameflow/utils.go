package gameflow

import (
	"strings"
)

// GetRoomIDFromGameID 從遊戲ID中提取房間ID
// 遊戲ID格式假設為 "room_SG01_game_uuid" 或 "game_uuid"
func GetRoomIDFromGameID(gameID string) string {
	// 如果包含 "room_" 前綴，則嘗試提取房間ID
	if len(gameID) > 5 && gameID[:5] == "room_" {
		parts := strings.Split(gameID, "_")
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	// 默認返回 SG01
	return "SG01"
}
