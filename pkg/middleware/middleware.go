package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 記錄請求信息的中間件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 開始時間
		startTime := time.Now()

		// 處理請求
		c.Next()

		// 結束時間
		endTime := time.Now()

		// 執行時間
		latency := endTime.Sub(startTime)

		// 請求方法
		reqMethod := c.Request.Method

		// 請求路由
		reqUri := c.Request.RequestURI

		// 狀態碼
		statusCode := c.Writer.Status()

		// 請求IP
		clientIP := c.ClientIP()

		// 日誌格式
		log.Printf("| %3d | %13v | %15s | %s | %s |",
			statusCode,
			latency,
			clientIP,
			reqMethod,
			reqUri,
		)
	}
}

// Cors 處理跨域請求
func Cors() gin.HandlerFunc {
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

// Recovery 從 panic 恢復的中間件
func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}
