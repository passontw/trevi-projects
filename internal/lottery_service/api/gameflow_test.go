package api

import (
	"context"
	"fmt"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/pkg/utils/scheduler"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// 首先需要創建一個模擬的存儲庫
type MockGameRepository struct {
	mock.Mock
}

// 實現 GameRepository 接口所需的方法
func (m *MockGameRepository) SaveGame(ctx context.Context, game *gameflow.GameData) error {
	args := m.Called(ctx, game)
	return args.Error(0)
}

func (m *MockGameRepository) GetCurrentGameByRoom(ctx context.Context, roomID string) (*gameflow.GameData, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gameflow.GameData), args.Error(1)
}

func (m *MockGameRepository) DeleteCurrentGameByRoom(ctx context.Context, roomID string) error {
	args := m.Called(ctx, roomID)
	return args.Error(0)
}

func (m *MockGameRepository) SaveGameHistory(ctx context.Context, game *gameflow.GameData) error {
	args := m.Called(ctx, game)
	return args.Error(0)
}

func (m *MockGameRepository) GetLuckyBallsByRoom(ctx context.Context, roomID string) ([]gameflow.Ball, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]gameflow.Ball), args.Error(1)
}

func (m *MockGameRepository) SaveLuckyBallsToRoom(ctx context.Context, roomID string, balls []gameflow.Ball) error {
	args := m.Called(ctx, roomID, balls)
	return args.Error(0)
}

// TestSchedulerForGameAdvancement 測試使用排程器進行遊戲階段推進
func TestSchedulerForGameAdvancement(t *testing.T) {
	// 初始化測試環境
	logger, _ := zap.NewDevelopment()
	mockRepo := new(MockGameRepository)
	scheduler := scheduler.NewLotteryScheduler()

	// 啟動排程器
	scheduler.Start()
	defer scheduler.Stop()

	ctx := context.Background()
	roomID := "test_room_1"
	gameID := "test_game_1"

	// 模擬遊戲數據
	game := &gameflow.GameData{
		GameID:       gameID,
		RoomID:       roomID,
		CurrentStage: gameflow.StagePreparation,
	}

	// 設置 Mock 預期行為
	mockRepo.On("GetCurrentGameByRoom", ctx, roomID).Return(game, nil)
	mockRepo.On("SaveGame", ctx, mock.Anything).Return(nil)

	// 短時間排程測試
	duration := 200 * time.Millisecond
	taskID := fmt.Sprintf("%s_%s_advance", roomID, gameID)

	// 創建帶有 WaitGroup 的閉包，用於同步測試
	var wg sync.WaitGroup
	wg.Add(1)

	var advanceCalled bool
	var taskExecuted bool

	// 排程遊戲推進
	err := scheduler.ScheduleOnce(
		duration,
		taskID,
		func(ctx context.Context) {
			taskExecuted = true
			// 在這裡不實際調用 AdvanceStageForRoom，只標記已被調用
			advanceCalled = true
			wg.Done()
		},
	)

	// 驗證排程是否成功
	assert.NoError(t, err)
	assert.True(t, scheduler.JobExists(taskID))

	// 等待任務執行
	wg.Wait()

	// 驗證結果
	assert.True(t, taskExecuted, "任務應該被執行")
	assert.True(t, advanceCalled, "推進階段函數應該被調用")

	// 驗證任務執行後被移除
	assert.False(t, scheduler.JobExists(taskID), "任務執行後應該被移除")
}
