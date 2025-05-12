package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	newlotterypb "g38_lottery_service/internal/generated/api/v1/lottery"
	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/internal/lottery_service/grpc/dealer"
	"g38_lottery_service/internal/lottery_service/grpc/lottery"
	oldpb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// GrpcServer 代表 gRPC 伺服器
type GrpcServer struct {
	config         *config.AppConfig
	logger         *zap.Logger
	server         *grpc.Server
	dealerSvc      *dealer.DealerService
	dealerWrapper  *dealer.DealerServiceWrapper
	lotteryService *lottery.LotteryService
	lotteryWrapper *lottery.LotteryServiceWrapper
	gameManager    *gameflow.GameManager
	listener       net.Listener
}

// NewGrpcServer 創建一個新的 gRPC 伺服器
func NewGrpcServer(
	config *config.AppConfig,
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
) *GrpcServer {
	// 設置 gRPC 的 keepalive 參數
	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second,
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
	}

	// 創建 gRPC 伺服器
	s := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
	)

	// 創建 DealerService 實例
	dealerSvc := dealer.NewDealerService(logger, gameManager)

	// 創建 DealerServiceWrapper 實例
	dealerWrapper := dealer.NewDealerServiceWrapper(logger, gameManager)

	// 創建 LotteryService 實例
	lotteryService := lottery.NewLotteryService(logger, gameManager, dealerSvc)

	// 創建 LotteryServiceWrapper 實例
	lotteryWrapper := lottery.NewLotteryServiceWrapper(logger, lotteryService)

	// 註冊 DealerService
	oldpb.RegisterDealerServiceServer(s, dealerSvc)

	// 註冊新的 API 服務
	// 暫時停用 DealerServiceAdapter 註冊，等待適配器正確實現
	// newdealerpb.RegisterDealerServiceServer(s, ...)
	newlotterypb.RegisterLotteryServiceServer(s, lotteryWrapper)

	// 啟用 gRPC 反射，用於服務發現
	reflection.Register(s)

	return &GrpcServer{
		config:         config,
		logger:         logger.Named("grpc_server"),
		server:         s,
		dealerSvc:      dealerSvc,
		dealerWrapper:  dealerWrapper,
		lotteryService: lotteryService,
		lotteryWrapper: lotteryWrapper,
		gameManager:    gameManager,
	}
}

// Start 啟動 gRPC 伺服器
func (gs *GrpcServer) Start() error {
	addr := fmt.Sprintf("%s:%d", gs.config.Server.Host, gs.config.Server.GrpcPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		gs.logger.Error("無法啟動 gRPC 伺服器", zap.Error(err))
		return err
	}

	gs.listener = listener
	gs.logger.Info("gRPC 伺服器已啟動", zap.String("address", addr))

	// 在新的 goroutine 中啟動伺服器
	go func() {
		if err := gs.server.Serve(listener); err != nil {
			gs.logger.Error("gRPC 伺服器錯誤", zap.Error(err))
		}
	}()

	return nil
}

// Stop 停止 gRPC 伺服器
func (gs *GrpcServer) Stop() {
	gs.logger.Info("正在停止 gRPC 伺服器")
	gs.server.GracefulStop()
	gs.logger.Info("gRPC 伺服器已停止")
}

// ProvideGrpcServer 為 fx 框架提供 gRPC 伺服器
func ProvideGrpcServer(lc fx.Lifecycle, config *config.AppConfig, logger *zap.Logger, gameManager *gameflow.GameManager) *GrpcServer {
	gs := NewGrpcServer(config, logger, gameManager)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return gs.Start()
		},
		OnStop: func(ctx context.Context) error {
			gs.Stop()
			return nil
		},
	})

	return gs
}

// RegisterGrpcServer 注册 gRPC 伺服器
func RegisterGrpcServer() fx.Option {
	return fx.Provide(ProvideGrpcServer)
}
