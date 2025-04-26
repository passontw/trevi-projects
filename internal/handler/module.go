package handler

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		NewGameHandler,
		NewRouter,
	),
	fx.Invoke(StartServer),
)
