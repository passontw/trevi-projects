package grpc

import (
	"go.uber.org/fx"
)

// Module 提供 gRPC 服務模塊
var Module = fx.Options(
	RegisterGrpcServer(),
)
