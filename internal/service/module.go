package service

import (
	"g38_lottery_service/game"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewGameService,
			fx.As(new(GameService)),
		),
	),
	game.Module,
)
