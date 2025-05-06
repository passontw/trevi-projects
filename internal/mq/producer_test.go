package mq

import (
	"g38_lottery_service/internal/config"
	"testing"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 模擬 RocketMQ 生產者
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProducer) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProducer) SendSync(ctx interface{}, msg ...*primitive.Message) (*primitive.SendResult, error) {
	args := m.Called(ctx, msg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*primitive.SendResult), args.Error(1)
}

func (m *MockProducer) SendAsync(ctx interface{}, f func(context interface{}, result *primitive.SendResult, err error), msg ...*primitive.Message) error {
	args := m.Called(ctx, f, msg)
	return args.Error(0)
}

func (m *MockProducer) SendOneWay(ctx interface{}, msg ...*primitive.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

// 原始 NewProducer 函數的替代品
var originalNewProducer = rocketmq.NewProducer
var mockProducerInstance *MockProducer

// 測試用的配置
func createTestConfig() *config.AppConfig {
	return &config.AppConfig{
		RocketMQ: config.RocketMQConfig{
			NameServers:   []string{"127.0.0.1:9876"},
			ProducerGroup: "test-producer-group",
		},
	}
}

// 測試前的設置
func setupTest() (*MockProducer, *zap.Logger) {
	mockProducer := new(MockProducer)
	mockProducerInstance = mockProducer

	// 替換原始函數
	rocketmq.NewProducer = func(opts ...interface{}) (rocketmq.Producer, error) {
		return mockProducer, nil
	}

	logger := zaptest.NewLogger(nil)
	return mockProducer, logger
}

// 測試後的清理
func teardownTest() {
	// 恢復原始函數
	rocketmq.NewProducer = originalNewProducer
	mockProducerInstance = nil
}

// TestNewMessageProducer 測試創建生產者
func TestNewMessageProducer(t *testing.T) {
	mockProducer, logger := setupTest()
	defer teardownTest()

	// 設置預期行為
	mockProducer.On("Start").Return(nil)

	// 創建生產者
	producer, err := NewMessageProducer(logger, createTestConfig())

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, producer)
	mockProducer.AssertExpectations(t)
}

// TestSendLotteryResult 測試發送開獎結果
func TestSendLotteryResult(t *testing.T) {
	mockProducer, logger := setupTest()
	defer teardownTest()

	// 設置預期行為
	mockProducer.On("Start").Return(nil)
	mockProducer.On("SendSync", mock.Anything, mock.Anything).Return(
		&primitive.SendResult{
			Status:       primitive.SendOK,
			MsgID:        "test-msg-id",
			MessageQueue: primitive.MessageQueue{},
		}, nil,
	)

	// 創建生產者
	producer, err := NewMessageProducer(logger, createTestConfig())
	assert.NoError(t, err)

	// 發送開獎結果
	result := map[string]interface{}{
		"balls": []map[string]interface{}{
			{"number": 1, "color": "red"},
			{"number": 15, "color": "blue"},
		},
	}

	err = producer.SendLotteryResult("game-001", result)

	// 驗證結果
	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

// TestSendLotteryStatus 測試發送開獎狀態
func TestSendLotteryStatus(t *testing.T) {
	mockProducer, logger := setupTest()
	defer teardownTest()

	// 設置預期行為
	mockProducer.On("Start").Return(nil)
	mockProducer.On("SendSync", mock.Anything, mock.Anything).Return(
		&primitive.SendResult{
			Status:       primitive.SendOK,
			MsgID:        "test-msg-id",
			MessageQueue: primitive.MessageQueue{},
		}, nil,
	)

	// 創建生產者
	producer, err := NewMessageProducer(logger, createTestConfig())
	assert.NoError(t, err)

	// 發送開獎狀態
	details := map[string]interface{}{
		"total_balls": 5,
		"drawn_balls": 3,
		"is_complete": false,
	}

	err = producer.SendLotteryStatus("game-001", "DRAWING_IN_PROGRESS", details)

	// 驗證結果
	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

// TestSendMessage 測試發送自定義消息
func TestSendMessage(t *testing.T) {
	mockProducer, logger := setupTest()
	defer teardownTest()

	// 設置預期行為
	mockProducer.On("Start").Return(nil)
	mockProducer.On("SendSync", mock.Anything, mock.Anything).Return(
		&primitive.SendResult{
			Status:       primitive.SendOK,
			MsgID:        "test-msg-id",
			MessageQueue: primitive.MessageQueue{},
		}, nil,
	)

	// 創建生產者
	producer, err := NewMessageProducer(logger, createTestConfig())
	assert.NoError(t, err)

	// 發送自定義消息
	payload := map[string]interface{}{
		"game_id":      "game-001",
		"message_type": "test_message",
		"data":         "這是一條測試消息",
	}

	err = producer.SendMessage("custom-topic", payload)

	// 驗證結果
	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

// TestSendError 測試發送失敗的情況
func TestSendError(t *testing.T) {
	mockProducer, logger := setupTest()
	defer teardownTest()

	// 設置預期行為
	mockProducer.On("Start").Return(nil)
	mockProducer.On("SendSync", mock.Anything, mock.Anything).Return(
		nil, primitive.NewRemotingErr("測試發送失敗"),
	)

	// 創建生產者
	producer, err := NewMessageProducer(logger, createTestConfig())
	assert.NoError(t, err)

	// 發送自定義消息
	payload := map[string]interface{}{
		"game_id": "game-001",
		"data":    "測試數據",
	}

	err = producer.SendMessage("custom-topic", payload)

	// 驗證結果
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "發送消息失敗")
	mockProducer.AssertExpectations(t)
}

// TestProducerStop 測試停止生產者
func TestProducerStop(t *testing.T) {
	mockProducer, logger := setupTest()
	defer teardownTest()

	// 設置預期行為
	mockProducer.On("Start").Return(nil)
	mockProducer.On("Shutdown").Return(nil)

	// 創建生產者
	producer, err := NewMessageProducer(logger, createTestConfig())
	assert.NoError(t, err)

	// 停止生產者
	producer.Stop()

	// 驗證結果
	mockProducer.AssertExpectations(t)
}
