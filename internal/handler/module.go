package handler

import (
	"g38_lottery_service/internal/config"
	"g38_lottery_service/pkg/dealerWebsocket"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		NewGameHandler,
		NewRouter,
	),
	fx.Invoke(func(handler *GameHandler, wsHandler *dealerWebsocket.WebSocketHandler) {
		// 這裡不需要做任何事情，只是告訴 fx 我們需要這些依賴
	}),
	fx.Invoke(
		fx.Annotate(
			func(cfg *config.Config, router *gin.Engine, wsHandler *dealerWebsocket.WebSocketHandler, playerWSHandler *dealerWebsocket.WebSocketHandler, gameHandler *GameHandler) {
				StartServer(cfg, router, wsHandler, playerWSHandler, gameHandler)
			},
			fx.ParamTags(``, ``, ``, `name:"playerWSHandler"`, ``),
		),
	),
)
