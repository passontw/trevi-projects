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
		// 移除重复提供的 RedisManager 实例
		// 它已经在 redisManager 包中提供了
		fx.Annotate(
			NewAuthService,
			fx.As(new(AuthService)),
		),
	),
)
