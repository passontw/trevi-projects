package handler

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		NewUserHandler,
		NewAuthHandler,
		NewRouter,
	),
	fx.Invoke(StartServer),
)
