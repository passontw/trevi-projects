package grpc

import (
	"context"
	"fmt"
	"net"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/dealerWebsocket"
	"g38_lottery_service/internal/gameflow"
	"g38_lottery_service/internal/grpc/dealer"
	pb "g38_lottery_service/internal/proto/dealer"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GrpcServer 代表 gRPC 服務器
type GrpcServer struct {
	config      *config.AppConfig
	logger      *zap.Logger
	server      *grpc.Server
	dealerSvc   *dealer.DealerService
	gameManager *gameflow.GameManager
}

// NewGrpcServer 創建新的 gRPC 服務器
func NewGrpcServer(
	config *config.AppConfig,
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
	dealerServer *dealerWebsocket.DealerServer,
) *GrpcServer {
	// 創建 gRPC 服務器
	server := grpc.NewServer()

	// 創建實例
	grpcServer := &GrpcServer{
		config:      config,
		logger:      logger.With(zap.String("component", "grpc_server")),
		server:      server,
		gameManager: gameManager,
	}

	// 創建並註冊 DealerService
	dealerSvc := dealer.NewDealerService(logger, gameManager, dealerServer)
	pb.RegisterDealerServiceServer(server, dealerSvc)
	grpcServer.dealerSvc = dealerSvc

	// 啟用反射服務，支持諸如 grpcurl 之類的工具進行服務發現
	reflection.Register(server)

	return grpcServer
}

// Start 啟動 gRPC 服務器
func (s *GrpcServer) Start(lc fx.Lifecycle) {
	// 使用應用配置中的 gRPC 端口
	serverAddr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.GrpcPort)
	s.logger.Info("Starting gRPC server",
		zap.String("address", serverAddr),
		zap.Int("port", s.config.Server.GrpcPort))

	// 生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 監聽指定端口
			lis, err := net.Listen("tcp", serverAddr)
			if err != nil {
				s.logger.Error("Failed to listen on gRPC port", zap.Error(err))
				return err
			}

			// 啟動 gRPC 服務器
			go func() {
				s.logger.Info("gRPC server listening", zap.String("address", serverAddr))
				if err := s.server.Serve(lis); err != nil {
					s.logger.Error("gRPC server failed", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.logger.Info("Stopping gRPC server")
			s.server.GracefulStop()
			return nil
		},
	})
}

// Module 提供 FX 模塊
var Module = fx.Options(
	fx.Provide(NewGrpcServer),
	fx.Invoke(func(server *GrpcServer, lc fx.Lifecycle) {
		server.Start(lc)
	}),
)
