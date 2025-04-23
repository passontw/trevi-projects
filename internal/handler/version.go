package handler

import (
	"fmt"
	"net/http"

	"g38_lottery_service/internal/config"

	"github.com/gin-gonic/gin"
)

// RegisterVersionEndpoint 註冊一個 API 端點以獲取版本信息
func RegisterVersionEndpoint(router *gin.Engine) {
	apiVersion := config.GetAPIVersion()

	// 使用當前 API 版本的路徑
	// @Summary      取得應用版本資訊
	// @Description  返回應用程式的詳細版本資訊，包括構建日期、Git提交等
	// @Tags         系統資訊
	// @Accept       json
	// @Produce      json
	// @Success      200  {object}  config.Version
	// @Router       /api/{api_version}/version [get]
	router.GET(fmt.Sprintf("/api/%s/version", apiVersion), func(c *gin.Context) {
		c.JSON(http.StatusOK, config.GetVersion())
	})

	// 保留簡潔版本端點作為向後兼容
	// @Summary      取得應用版本資訊（向後兼容）
	// @Description  返回應用程式的詳細版本資訊，包括構建日期、Git提交等
	// @Tags         系統資訊
	// @Accept       json
	// @Produce      json
	// @Success      200  {object}  config.Version
	// @Router       /api/version [get]
	router.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, config.GetVersion())
	})

	// 添加簡潔版本端點
	// @Summary      取得應用版本資訊（簡潔）
	// @Description  返回應用程式的詳細版本資訊，包括構建日期、Git提交等
	// @Tags         系統資訊
	// @Accept       json
	// @Produce      json
	// @Success      200  {object}  config.Version
	// @Router       /version [get]
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, config.GetVersion())
	})
}
