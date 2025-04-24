package main

// @Summary 健康檢查
// @Description 檢查服務是否正常運行
// @Tags 系統
// @Produce json
// @Success 200 {object} map[string]interface{} "服務狀態"
// @Router /health [get]
func healthCheckHandler() {}

// @Summary 版本資訊
// @Description 獲取服務版本信息
// @Tags 系統
// @Produce json
// @Success 200 {object} map[string]interface{} "版本信息"
// @Router /version [get]
func versionHandler() {}

// @Summary 獲取遊戲狀態
// @Description 獲取當前遊戲的詳細狀態信息
// @Tags 遊戲
// @Produce json
// @Success 200 {object} game.GameStatusResponse "遊戲狀態詳細資訊"
// @Router /game/status [get]
func gameStatusHandler() {}

// @Summary WebSocket連接
// @Description 建立WebSocket連接
// @Tags WebSocket
// @Accept json
// @Produce json
// @Success 101 {string} string "WebSocket連接成功"
// @Router /ws [get]
func wsHandler() {}

// @Summary 認證請求
// @Description 處理用戶認證請求
// @Tags 認證
// @Accept json
// @Produce json
// @Param data body map[string]interface{} true "認證信息"
// @Success 200 {object} map[string]interface{} "認證成功"
// @Failure 400 {object} map[string]interface{} "錯誤的請求"
// @Failure 401 {object} map[string]interface{} "認證失敗"
// @Router /auth [post]
func authHandler() {}
