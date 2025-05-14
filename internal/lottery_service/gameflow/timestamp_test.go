package gameflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// 模擬的遊戲存儲庫
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveGame(ctx context.Context, game *GameData) error {
	args := m.Called(ctx, game)
	return args.Error(0)
}

func (m *MockRepository) GetCurrentGame(ctx context.Context) (*GameData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GameData), args.Error(1)
}

func (m *MockRepository) DeleteCurrentGame(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) GetCurrentGameByRoom(ctx context.Context, roomID string) (*GameData, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GameData), args.Error(1)
}

func (m *MockRepository) DeleteCurrentGameByRoom(ctx context.Context, roomID string) error {
	args := m.Called(ctx, roomID)
	return args.Error(0)
}

func (m *MockRepository) GetLuckyBalls(ctx context.Context) ([]Ball, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Ball), args.Error(1)
}

func (m *MockRepository) SaveLuckyBalls(ctx context.Context, balls []Ball) error {
	args := m.Called(ctx, balls)
	return args.Error(0)
}

func (m *MockRepository) GetLuckyBallsByRoom(ctx context.Context, roomID string) ([]Ball, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).([]Ball), args.Error(1)
}

func (m *MockRepository) SaveLuckyBallsToRoom(ctx context.Context, roomID string, balls []Ball) error {
	args := m.Called(ctx, roomID, balls)
	return args.Error(0)
}

func (m *MockRepository) SaveGameHistory(ctx context.Context, game *GameData) error {
	args := m.Called(ctx, game)
	return args.Error(0)
}

func (m *MockRepository) GetRecentGameHistories(ctx context.Context, limit int) ([]*GameData, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*GameData), args.Error(1)
}

func (m *MockRepository) GetRecentGameHistoriesByRoom(ctx context.Context, roomID string, limit int) ([]*GameData, error) {
	args := m.Called(ctx, roomID, limit)
	return args.Get(0).([]*GameData), args.Error(1)
}

func (m *MockRepository) GetTotalGamesCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) GetCancelledGamesCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) GetTotalGamesCountByRoom(ctx context.Context, roomID string) (int64, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).(int64), args.Error(1)
}

// TestTimestampTimer 測試基於時間戳的計時器
func TestTimestampTimer(t *testing.T) {
	// 創建測試環境
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	mockRepo := new(MockRepository)

	// 創建遊戲管理器
	manager := NewGameManager(mockRepo, logger, nil)

	// 創建測試遊戲數據
	game := NewGameData("test_game_1", "test_room_1")
	game.CurrentStage = StageCardPurchaseOpen

	// 設置 mock 預期
	mockRepo.On("SaveGame", mock.Anything, mock.Anything).Return(nil)

	// 設置短時間計時器 (500ms)
	duration := 500 * time.Millisecond
	manager.setupTimerForGame(ctx, game, "test_room_1", duration)

	// 驗證時間戳已設置
	assert.False(t, game.StageExpireTime.IsZero(), "StageExpireTime 應該已設置")

	// 驗證時間戳在正確的範圍內
	expectedTime := time.Now().Add(duration)
	timeDiff := game.StageExpireTime.Sub(expectedTime)
	assert.True(t, timeDiff > -100*time.Millisecond && timeDiff < 100*time.Millisecond,
		"StageExpireTime 應該接近當前時間加上持續時間")

	// 驗證 BuildGameStatusResponse 是否正確包含時間戳
	response := BuildGameStatusResponse(game)
	assert.NotEmpty(t, response.Game.Timeline.StageExpireTime, "回應中應該包含階段過期時間")

	// 驗證計算的剩餘時間
	assert.True(t, response.Game.Timeline.RemainingTime <= int(duration.Seconds()),
		"剩餘時間應該不超過設置的持續時間")

	// 驗證最大超時時間
	config := GetStageConfig(game.CurrentStage)
	expectedMaxTimeout := int(config.Timeout.Seconds())
	assert.Equal(t, expectedMaxTimeout, response.Game.Timeline.MaxTimeout,
		"最大超時時間應該等於階段配置中的值")

	// 驗證 mock 被調用
	mockRepo.AssertExpectations(t)
}
