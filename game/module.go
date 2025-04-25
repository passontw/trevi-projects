package game

import "go.uber.org/fx"

// Module 是遊戲模組
var Module = fx.Module("game",
	fx.Provide(
		// 提供遊戲流程控制器
		NewDataFlowController,
	),
)
