package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"g38_lottery_service/game"
	"g38_lottery_service/internal/config"
	"g38_lottery_service/pkg/dealerWebsocket"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type SuccessResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewRouter(
	cfg *config.Config,
	gameHandler *GameHandler,
	wsHandler *dealerWebsocket.WebSocketHandler,
) *gin.Engine {
	r := gin.Default()
	r.Use(configureCORS())
	r.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, SuccessResponse{Message: "Service is healthy"})
	})

	r.GET("/ws", func(c *gin.Context) {
		wsHandler.HandleWebSocket(c.Writer, c.Request)
	})

	api := r.Group("/api/v1")
	{
		configurePublicRoutes(api, gameHandler)
		configureAuthenticatedRoutes(api, gameHandler)
	}

	return r
}

func configureCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func configurePublicRoutes(api *gin.RouterGroup, gameHandler *GameHandler) {
	api.GET("/game/status", gameHandler.GetGameStatus)
	api.GET("/game/state", gameHandler.GetGameState)
}

func configureAuthenticatedRoutes(api *gin.RouterGroup, gameHandler *GameHandler) {
	authorized := api.Group("/")

	authorized.POST("/game/state", gameHandler.ChangeGameState)
}

func StartServer(
	cfg *config.Config,
	router *gin.Engine,
	wsHandler *dealerWebsocket.WebSocketHandler,
	playerWSHandler *dealerWebsocket.WebSocketHandler,
	gameHandler *GameHandler,
) {
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("正在使用端口 %d 啟動 API 服務器...\n", cfg.Server.Port)

	// 創建通道來追蹤 WebSocket 服務器的啟動狀態
	playerWSReady := make(chan bool, 1)
	dealerWSReady := make(chan bool, 1)

	// 將 Gin 服務器的啟動放在單獨的 goroutine 中，避免阻塞 FX 生命週期
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("無法啟動 API 服務器: %v", err)
		}
	}()

	// 啟動專門的荷官 WebSocket 服務器
	if cfg.Server.DealerWSPort > 0 {
		go func() {
			dealerWSAddr := fmt.Sprintf(":%d", cfg.Server.DealerWSPort)
			dealerMux := http.NewServeMux()

			// 健康檢查端點
			dealerMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status": "ok", "service": "dealer_websocket"}`))
			})

			// 註冊 WebSocket 處理程序
			dealerMux.HandleFunc("/ws", wsHandler.HandleWebSocket)

			fmt.Printf("正在使用端口 %d 啟動荷官 WebSocket 服務器...\n", cfg.Server.DealerWSPort)

			if err := http.ListenAndServe(dealerWSAddr, dealerMux); err != nil {
				log.Fatalf("無法啟動荷官 WebSocket 服務器: %v", err)
			}
		}()

		// 使用簡單的延遲來模擬等待服務器啟動完成
		time.Sleep(500 * time.Millisecond)
		log.Println("荷官 WebSocket 服務器啟動完成")
		dealerWSReady <- true
	} else {
		// 如果未配置荷官 WebSocket，視為已準備就緒
		dealerWSReady <- true
	}

	// 啟動專門的玩家 WebSocket 服務器
	if cfg.Server.PlayerWSPort > 0 {
		go func() {
			playerWSAddr := fmt.Sprintf(":%d", cfg.Server.PlayerWSPort)
			playerMux := http.NewServeMux()

			// 健康檢查端點
			playerMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status": "ok", "service": "player_websocket"}`))
			})

			// 註冊 WebSocket 處理程序
			playerMux.HandleFunc("/ws", playerWSHandler.HandleWebSocket) // 使用玩家專用的處理程序

			fmt.Printf("正在使用端口 %d 啟動玩家 WebSocket 服務器...\n", cfg.Server.PlayerWSPort)

			if err := http.ListenAndServe(playerWSAddr, playerMux); err != nil {
				log.Fatalf("無法啟動玩家 WebSocket 服務器: %v", err)
			}
		}()

		// 使用簡單的延遲來模擬等待服務器啟動完成
		time.Sleep(500 * time.Millisecond)
		log.Println("玩家 WebSocket 服務器啟動完成")
		playerWSReady <- true
	} else {
		// 如果未配置玩家 WebSocket，視為已準備就緒
		playerWSReady <- true
	}

	// 啟動一個 goroutine 來等待兩個 WebSocket 服務器都準備就緒
	go func() {
		// 等待兩個通道都收到信號
		<-playerWSReady
		<-dealerWSReady

		// 給服務器一些額外時間來穩定運行
		time.Sleep(1 * time.Second)

		// 兩個 WebSocket 服務器都已啟動，將遊戲狀態改為 READY
		log.Println("玩家和荷官 WebSocket 服務器都已啟動完成，將遊戲狀態更改為 READY")
		err := gameHandler.gameService.ChangeState(game.StateReady)
		if err != nil {
			log.Printf("將遊戲狀態更改為 READY 失敗: %v", err)
		} else {
			log.Println("遊戲狀態已成功更改為 READY")
		}
	}()
}
