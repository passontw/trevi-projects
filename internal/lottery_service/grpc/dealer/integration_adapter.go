package dealer

import (
	"context"
	"time"

	newlotterypb "g38_lottery_service/internal/generated/api/v1/lottery"
	"g38_lottery_service/internal/generated/common"
	"g38_lottery_service/internal/lottery_service/gameflow"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IntegrationAdapter 實現 LotteryService 接口
// 作為 DealerService 和 LotteryService 的整合層
type IntegrationAdapter struct {
	newlotterypb.UnimplementedLotteryServiceServer
	logger      *zap.Logger
	dealerSvc   *DealerService
	gameManager *gameflow.GameManager
}

// NewIntegrationAdapter 創建一個新的整合適配器
func NewIntegrationAdapter(
	logger *zap.Logger,
	dealerSvc *DealerService,
	gameManager *gameflow.GameManager,
) *IntegrationAdapter {
	return &IntegrationAdapter{
		logger:      logger.Named("integration_adapter"),
		dealerSvc:   dealerSvc,
		gameManager: gameManager,
	}
}

// StartNewRound 實現 LotteryService 的 StartNewRound 方法
func (a *IntegrationAdapter) StartNewRound(ctx context.Context, req *newlotterypb.StartNewRoundRequest) (*newlotterypb.StartNewRoundResponse, error) {
	a.logger.Info("整合適配器: StartNewRound")

	// 使用 GameManager 創建新遊戲
	gameID, err := a.gameManager.CreateNewGameForRoom(ctx, req.RoomId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "無法創建新遊戲: %v", err)
	}

	// 獲取當前遊戲
	game := a.gameManager.GetCurrentGameByRoom(req.RoomId)
	if game == nil {
		return nil, status.Errorf(codes.NotFound, "找不到房間 %s 的遊戲", req.RoomId)
	}

	// 在後台推進階段
	go func() {
		// 創建新的上下文
		newCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := a.gameManager.AdvanceStageForRoom(newCtx, req.RoomId, true); err != nil {
			a.logger.Error("推進階段失敗", zap.Error(err))
		}
	}()

	// 構建回應
	return &newlotterypb.StartNewRoundResponse{
		GameId:       gameID,
		StartTime:    timestamppb.New(game.StartTime),
		CurrentStage: convertGameflowStageToProtoStage(game.CurrentStage),
	}, nil
}

// convertGameflowStageToProtoStage 將 gameflow.GameStage 轉換為 common.GameStage
func convertGameflowStageToProtoStage(stage gameflow.GameStage) common.GameStage {
	switch stage {
	case gameflow.StagePreparation:
		return common.GameStage_GAME_STAGE_PREPARATION
	case gameflow.StageNewRound:
		return common.GameStage_GAME_STAGE_NEW_ROUND
	case gameflow.StageCardPurchaseOpen:
		return common.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case gameflow.StageCardPurchaseClose:
		return common.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case gameflow.StageDrawingStart:
		return common.GameStage_GAME_STAGE_DRAWING_START
	case gameflow.StageDrawingClose:
		return common.GameStage_GAME_STAGE_DRAWING_CLOSE
	default:
		return common.GameStage_GAME_STAGE_UNSPECIFIED
	}
}
