package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/dealerWebsocket"
	"g38_lottery_service/internal/gameflow"
	"g38_lottery_service/internal/grpc/dealer"
	pb "g38_lottery_service/internal/proto/generated/dealer"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// GrpcServer 代表 gRPC 服務器
type GrpcServer struct {
	config      *config.AppConfig
	logger      *zap.Logger
	server      *grpc.Server
	dealerSvc   *dealer.DealerService
	gameManager *gameflow.GameManager
	listener    net.Listener
}

// NewGrpcServer 創建新的 gRPC 服務器
func NewGrpcServer(
	config *config.AppConfig,
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
	dealerServer *dealerWebsocket.DealerServer,
) *GrpcServer {
	// 設定 gRPC keepalive 參數，確保連接活躍
	keepAliveOpts := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // 如果連接閒置超過此時間，發送 ping
		MaxConnectionAge:      30 * time.Second, // 連接最大存活時間
		MaxConnectionAgeGrace: 5 * time.Second,  // 強制關閉連接前的寬限期
		Time:                  5 * time.Second,  // 每隔 x 秒發送一次 ping
		Timeout:               2 * time.Second,  // ping 超時後等待的時間
	})

	// 創建 gRPC 服務器
	logger.Info("創建 gRPC 服務器")
	server := grpc.NewServer(
		keepAliveOpts,
		grpc.ConnectionTimeout(10*time.Second), // 連接超時設置
	)

	// 創建實例
	grpcServer := &GrpcServer{
		config:      config,
		logger:      logger.With(zap.String("component", "grpc_server")),
		server:      server,
		gameManager: gameManager,
	}

	// 創建並註冊 DealerService
	logger.Info("註冊 DealerService 到 gRPC 服務器")
	dealerSvc := dealer.NewDealerService(logger, gameManager, dealerServer)
	pb.RegisterDealerServiceServer(server, dealerSvc)
	grpcServer.dealerSvc = dealerSvc

	// 啟用反射服務，支持諸如 grpcurl 之類的工具進行服務發現
	logger.Info("啟用 gRPC 反射服務")
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

	// 嘗試立即監聽，以便快速檢測端口問題
	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		s.logger.Error("Failed to listen on gRPC port, will retry in lifecycle hooks",
			zap.Error(err),
			zap.String("address", serverAddr))
	} else {
		s.listener = lis
		s.logger.Info("Successfully listening on gRPC port",
			zap.String("address", serverAddr))
	}

	// 生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 如果已經成功監聽，直接啟動服務
			if s.listener != nil {
				// 啟動 gRPC 服務器
				go func() {
					s.logger.Info("gRPC server starts serving requests",
						zap.String("address", serverAddr))

					if err := s.server.Serve(s.listener); err != nil {
						s.logger.Error("gRPC server failed", zap.Error(err))
					}
				}()
				return nil
			}

			// 如果之前沒有成功監聽，再次嘗試
			lis, err := net.Listen("tcp", serverAddr)
			if err != nil {
				s.logger.Error("Failed to listen on gRPC port", zap.Error(err))
				return err
			}

			s.listener = lis
			s.logger.Info("Successfully listening on gRPC port in lifecycle hook",
				zap.String("address", serverAddr))

			// 啟動 gRPC 服務器
			go func() {
				s.logger.Info("gRPC server starts serving requests in lifecycle hook",
					zap.String("address", serverAddr))

				if err := s.server.Serve(s.listener); err != nil {
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
