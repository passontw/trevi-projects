package main

// @title          G38 Lottery Service API
// @version        1.0
// @description    樂透遊戲服務 API 文檔
// @termsOfService http://swagger.io/terms/

// @contact.name  API Support
// @contact.url   http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url  http://www.apache.org/licenses/LICENSE-2.0.html

// @host     localhost:3001
// @BasePath /

// @tag.name 系統
// @tag.description 系統相關API

// @tag.name 遊戲
// @tag.description 遊戲相關API

// @tag.name WebSocket
// @tag.description WebSocket相關API

// @tag.name 認證
// @tag.description 認證相關API

// @Summary 健康檢查
// @Description 檢查服務是否正常運行
// @Tags 系統
// @Produce json
// @Success 200 {object} map[string]interface{} "服務狀態"
// @Router /health [get]
func swaggerHealthCheckHandler() {}

// @Summary 版本資訊
// @Description 獲取服務版本信息
// @Tags 系統
// @Produce json
// @Success 200 {object} map[string]interface{} "版本信息"
// @Router /version [get]
func swaggerVersionHandler() {}

// @Summary 獲取遊戲狀態
// @Description 獲取當前遊戲的詳細狀態信息
// @Tags 遊戲
// @Produce json
// @Success 200 {object} map[string]interface{} "遊戲狀態詳細資訊"
// @Router /game/status [get]
func swaggerGameStatusHandler() {}

// @Summary WebSocket連接
// @Description 建立WebSocket連接
// @Tags WebSocket
// @Accept json
// @Produce json
// @Success 101 {string} string "WebSocket連接成功"
// @Router /ws [get]
func swaggerWsHandler() {}

// @Summary 認證請求
// @Description 處理用戶認證請求
// @Tags 認證
// @Accept json
// @Produce json
// @Param token header string true "認證令牌"
// @Success 200 {object} map[string]interface{} "認證成功"
// @Failure 400 {object} map[string]interface{} "錯誤的請求"
// @Failure 401 {object} map[string]interface{} "認證失敗"
// @Router /auth [post]
func swaggerAuthHandler() {}
