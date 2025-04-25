package services

import (
	"g38_lottery_service/game"

	"go.uber.org/fx"
)

// Module 是服務模組
var Module = fx.Module("services",
	fx.Provide(
		// 提供遊戲服務
		fx.Annotate(
			NewGameService,
			fx.As(new(GameService)),
		),
	),
)

// ProvideGameController 直接提供遊戲控制器
func ProvideGameController(gameService GameService) *game.DataFlowController {
	return gameService.GetGameController()
}
