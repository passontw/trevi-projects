package handlers

import (
	"go.uber.org/fx"
)

// AuthenticationFunc 是認證函數類型
type AuthenticationFunc func(token string) (uint, error)

// Module 是處理程序模組
var Module = fx.Module("handlers",
	fx.Provide(
		// 玩家處理程序
		fx.Annotate(
			NewPlayerHandler,
			fx.ParamTags(``, ``, `name:"serverHost"`, `name:"playerPort"`, `name:"serverVersion"`, ``),
		),

		// 荷官處理程序
		fx.Annotate(
			NewDealerHandler,
			fx.ParamTags(``, ``, `name:"serverHost"`, `name:"dealerPort"`, `name:"serverVersion"`),
		),
	),
)
