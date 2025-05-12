package gameflow

import (
	"fmt"
	"net/http"
)

// HealthChecker 定義健康檢查接口
type HealthChecker interface {
	// Name 返回檢查器的名稱
	Name() string

	// Check 執行健康檢查，如果健康返回 nil，否則返回錯誤
	Check(r *http.Request) error
}

// GameManagerChecker 檢查遊戲管理器健康狀態
type GameManagerChecker struct {
	Name_       string
	GameManager *GameManager
}

// Name 返回檢查器的名稱
func (g *GameManagerChecker) Name() string {
	if g.Name_ != "" {
		return g.Name_
	}
	return "game-manager"
}

// Check 檢查遊戲管理器狀態
func (g *GameManagerChecker) Check(req *http.Request) error {
	if g.GameManager == nil {
		return fmt.Errorf("game manager not initialized")
	}

	// 獲取當前遊戲和支持的房間
	currentGame := g.GameManager.GetCurrentGame()
	supportedRooms := g.GameManager.GetSupportedRooms()

	// 檢查是否有支持的房間
	if len(supportedRooms) == 0 {
		return fmt.Errorf("no supported rooms")
	}

	// 檢查默認房間的遊戲是否存在
	if currentGame == nil {
		return fmt.Errorf("no current game in default room")
	}

	return nil
}
