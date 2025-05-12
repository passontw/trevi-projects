package grpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	newdealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
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
	dealerAdapter  *dealer.DealerServiceAdapter
	lotteryService *lottery.LotteryService
	lotteryWrapper *lottery.LotteryServiceWrapper
	gameManager    *gameflow.GameManager
	listener       net.Listener
	started        bool
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

	// 創建 DealerServiceAdapter 實例
	dealerAdapter := dealer.NewDealerServiceAdapter(logger, dealerSvc)

	// 創建 LotteryService 實例
	lotteryService := lottery.NewLotteryService(logger, gameManager, dealerSvc)

	// 創建 LotteryServiceWrapper 實例
	lotteryWrapper := lottery.NewLotteryServiceWrapper(logger, lotteryService)

	// 註冊 DealerService (舊版 API)
	oldpb.RegisterDealerServiceServer(s, dealerSvc)

	// 註冊新的 API 服務
	newdealerpb.RegisterDealerServiceServer(s, dealerAdapter)
	newlotterypb.RegisterLotteryServiceServer(s, lotteryWrapper)

	// 啟用 gRPC 反射，用於服務發現
	reflection.Register(s)

	return &GrpcServer{
		config:         config,
		logger:         logger.Named("grpc_server"),
		server:         s,
		dealerSvc:      dealerSvc,
		dealerAdapter:  dealerAdapter,
		lotteryService: lotteryService,
		lotteryWrapper: lotteryWrapper,
		gameManager:    gameManager,
		started:        false,
	}
}

// Start 啟動 gRPC 伺服器
func (gs *GrpcServer) Start() error {
	if gs.started {
		gs.logger.Warn("gRPC 伺服器已經在運行，無需重複啟動")
		return nil
	}

	addr := fmt.Sprintf("%s:%d", gs.config.Server.Host, gs.config.Server.GrpcPort)
	gs.logger.Info("嘗試啟動 gRPC 伺服器",
		zap.String("host", gs.config.Server.Host),
		zap.Int("port", gs.config.Server.GrpcPort),
		zap.String("address", addr))

	// 驗證地址
	if gs.config.Server.Host == "" {
		gs.logger.Warn("主機地址為空，使用默認地址 '0.0.0.0'")
		gs.config.Server.Host = "0.0.0.0"
		addr = fmt.Sprintf("%s:%d", gs.config.Server.Host, gs.config.Server.GrpcPort)
	}

	// 驗證端口
	if gs.config.Server.GrpcPort <= 0 {
		gs.logger.Error("無效的 gRPC 端口號", zap.Int("port", gs.config.Server.GrpcPort))
		return fmt.Errorf("無效的 gRPC 端口號: %d", gs.config.Server.GrpcPort)
	}

	// 嘗試聆聽端口
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		gs.logger.Error("無法啟動 gRPC 伺服器",
			zap.Error(err),
			zap.String("address", addr))
		return err
	}

	gs.listener = listener
	gs.started = true
	gs.logger.Info("gRPC 伺服器已啟動監聽", zap.String("address", addr))

	// 在新的 goroutine 中啟動伺服器
	go func() {
		gs.logger.Info("gRPC 伺服器開始接受請求")
		if err := gs.server.Serve(listener); err != nil {
			if err != grpc.ErrServerStopped {
				gs.logger.Error("gRPC 伺服器錯誤", zap.Error(err))
			} else {
				gs.logger.Info("gRPC 伺服器已正常停止")
			}
		}
	}()

	return nil
}

// Stop 停止 gRPC 伺服器
func (gs *GrpcServer) Stop() {
	if !gs.started {
		gs.logger.Warn("gRPC 伺服器未啟動，無需停止")
		return
	}

	gs.logger.Info("正在停止 gRPC 伺服器")
	gs.server.GracefulStop()
	gs.started = false
	gs.logger.Info("gRPC 伺服器已停止")
}

// ProvideGrpcServer 為 fx 框架提供 gRPC 伺服器
func ProvideGrpcServer(lc fx.Lifecycle, config *config.AppConfig, logger *zap.Logger, gameManager *gameflow.GameManager) *GrpcServer {
	logger.Info("初始化 gRPC 伺服器")

	// 確保配置正確
	if config.Server.GrpcPort <= 0 {
		// 設置默認端口
		defaultPort := 9100
		logger.Warn("gRPC 端口未設置或無效，使用默認端口", zap.Int("defaultPort", defaultPort))
		config.Server.GrpcPort = defaultPort
	}

	// 確保主機地址正確
	if config.Server.Host == "" {
		logger.Warn("主機地址為空，使用默認地址 '0.0.0.0'")
		config.Server.Host = "0.0.0.0"
	}

	gs := NewGrpcServer(config, logger, gameManager)

	// 檢查環境變量
	logger.Info("檢查 gRPC 相關環境變量")
	grpcPortEnv := os.Getenv("GRPC_PORT")
	if grpcPortEnv != "" {
		// 嘗試將環境變量轉換為整數
		if port, err := strconv.Atoi(grpcPortEnv); err == nil && port > 0 {
			logger.Info("使用環境變量設置的 gRPC 端口",
				zap.String("GRPC_PORT", grpcPortEnv),
				zap.Int("port", port))
			config.Server.GrpcPort = port
		} else {
			logger.Warn("環境變量 GRPC_PORT 無效，使用配置中的端口",
				zap.String("GRPC_PORT", grpcPortEnv),
				zap.Error(err),
				zap.Int("configPort", config.Server.GrpcPort))
		}
	} else {
		logger.Info("未設置 GRPC_PORT 環境變量，使用配置中的端口",
			zap.Int("port", config.Server.GrpcPort))
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("開始啟動 gRPC 伺服器",
				zap.String("host", config.Server.Host),
				zap.Int("port", config.Server.GrpcPort))

			// 顯示詳細配置
			logger.Info("gRPC 伺服器詳細配置",
				zap.String("host", config.Server.Host),
				zap.Int("grpcPort", config.Server.GrpcPort),
				zap.String("address", fmt.Sprintf("%s:%d", config.Server.Host, config.Server.GrpcPort)))

			// 啟動服務器
			if err := gs.Start(); err != nil {
				logger.Error("gRPC 伺服器啟動失敗", zap.Error(err))
				return err
			}

			// 檢查伺服器是否確實運行
			if !gs.started {
				err := fmt.Errorf("gRPC 伺服器未能正確啟動")
				logger.Error("gRPC 伺服器啟動檢查失敗", zap.Error(err))
				return err
			}

			logger.Info("gRPC 伺服器啟動成功",
				zap.String("address", fmt.Sprintf("%s:%d", config.Server.Host, config.Server.GrpcPort)))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("開始停止 gRPC 伺服器")
			gs.Stop()
			logger.Info("gRPC 伺服器停止完成")
			return nil
		},
	})

	return gs
}

// RegisterGrpcServer 注册 gRPC 伺服器
func RegisterGrpcServer() fx.Option {
	return fx.Provide(ProvideGrpcServer)
}
