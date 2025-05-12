package grpc

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module 提供 gRPC 服務模塊
var Module = fx.Options(
	RegisterGrpcServer(),
	fx.Invoke(func(gs *GrpcServer, logger *zap.Logger) {
		logger.Info("gRPC 模塊已註冊到 fx 應用程序")
		// 確保 GrpcServer 被正常啟動
		if gs != nil {
			logger.Info("已確認 GrpcServer 實例有效")
		} else {
			logger.Error("GrpcServer 實例為空，無法正常啟動 gRPC 服務")
		}
	}),
)
