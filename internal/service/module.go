package service

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		ProvideGormDB,
		fx.Annotate(
			NewUserService,
			fx.As(new(UserService)),
		),
		NewAuthService,
	),
)
