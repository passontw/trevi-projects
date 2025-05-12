package mq

import (
	"encoding/json"
	"fmt"
	"strconv"

	mqpb "g38_lottery_service/internal/generated/api/v1/mq"

	"go.uber.org/zap"
)

// ProducerWrapper 包裝舊的 MessageProducer 實現，使用新的 proto 消息格式
type ProducerWrapper struct {
	producer *MessageProducer
	logger   *zap.Logger
}

// NewProducerWrapper 創建一個新的生產者包裝器
func NewProducerWrapper(producer *MessageProducer, logger *zap.Logger) *ProducerWrapper {
	return &ProducerWrapper{
		producer: producer,
		logger:   logger.Named("producer_wrapper"),
	}
}

// SendLotteryResultMessage 發送開獎結果消息
func (w *ProducerWrapper) SendLotteryResultMessage(message *mqpb.LotteryResultMessage) error {
	w.logger.Debug("準備發送開獎結果消息", zap.String("game_id", message.Base.GameId))

	// 將 proto 消息轉換為 map 格式
	resultMap, err := protoMessageToMap(message)
	if err != nil {
		w.logger.Error("轉換開獎結果消息失敗", zap.Error(err))
		return fmt.Errorf("轉換開獎結果消息失敗: %w", err)
	}

	// 使用底層 producer 發送消息
	return w.producer.SendLotteryResult(message.Base.GameId, resultMap)
}

// SendLotteryStatusMessage 發送開獎狀態消息
func (w *ProducerWrapper) SendLotteryStatusMessage(message *mqpb.LotteryStatusMessage) error {
	w.logger.Debug("準備發送開獎狀態消息", zap.String("game_id", message.Base.GameId), zap.String("status", message.Status))

	// 將 proto 消息轉換為 map 格式
	statusMap, err := protoMessageToMap(message)
	if err != nil {
		w.logger.Error("轉換開獎狀態消息失敗", zap.Error(err))
		return fmt.Errorf("轉換開獎狀態消息失敗: %w", err)
	}

	// 使用底層 producer 發送消息
	return w.producer.SendLotteryStatus(message.Base.GameId, message.Status, statusMap)
}

// SendGameSnapshotMessage 發送遊戲快照消息
func (w *ProducerWrapper) SendGameSnapshotMessage(message *mqpb.GameSnapshotMessage) error {
	w.logger.Debug("準備發送遊戲快照消息", zap.String("game_id", message.Base.GameId), zap.String("event_type", message.EventType))

	// 將 proto 消息轉換為 map 格式
	snapshotMap, err := protoMessageToMap(message)
	if err != nil {
		w.logger.Error("轉換遊戲快照消息失敗", zap.Error(err))
		return fmt.Errorf("轉換遊戲快照消息失敗: %w", err)
	}

	// 使用底層 producer 發送消息
	return w.producer.SendGameSnapshot(message.Base.GameId, snapshotMap)
}

// SendCustomMessage 發送自定義消息
func (w *ProducerWrapper) SendCustomMessage(topic string, message interface{}) error {
	w.logger.Debug("準備發送自定義消息", zap.String("topic", topic))

	// 將消息轉換為 map 格式
	var messageMap map[string]interface{}
	var err error

	switch msg := message.(type) {
	case map[string]interface{}:
		messageMap = msg
	case *mqpb.BaseMessage, *mqpb.LotteryResultMessage, *mqpb.LotteryStatusMessage, *mqpb.GameSnapshotMessage:
		messageMap, err = protoMessageToMap(msg)
		if err != nil {
			w.logger.Error("轉換自定義消息失敗", zap.Error(err))
			return fmt.Errorf("轉換自定義消息失敗: %w", err)
		}
	default:
		messageMap, err = StructToMap(msg)
		if err != nil {
			w.logger.Error("轉換自定義消息失敗", zap.Error(err))
			return fmt.Errorf("轉換自定義消息失敗: %w", err)
		}
	}

	// 使用底層 producer 發送消息
	return w.producer.SendMessage(topic, messageMap)
}

// Stop 停止生產者
func (w *ProducerWrapper) Stop() {
	if w.producer != nil {
		w.producer.Stop()
	}
}

// protoMessageToMap 將 proto 消息轉換為 map[string]interface{}
func protoMessageToMap(message interface{}) (map[string]interface{}, error) {
	// 先將消息轉換為 JSON
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("序列化消息失敗: %w", err)
	}

	// 再將 JSON 轉換為 map
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("反序列化消息失敗: %w", err)
	}

	// 處理特殊字段（如枚舉類型）
	processMessageMap(result)

	return result, nil
}

// processMessageMap 處理消息中的特殊字段
func processMessageMap(message map[string]interface{}) {
	// 處理 MessageType 枚舉
	if msgType, ok := message["message_type"].(float64); ok {
		// 轉換為字符串表示
		switch int(msgType) {
		case int(mqpb.MessageType_MESSAGE_TYPE_LOTTERY_RESULT):
			message["message_type"] = "lottery_result"
		case int(mqpb.MessageType_MESSAGE_TYPE_LOTTERY_STATUS):
			message["message_type"] = "lottery_status"
		case int(mqpb.MessageType_MESSAGE_TYPE_STAGE_CHANGE):
			message["message_type"] = "stage_change"
		case int(mqpb.MessageType_MESSAGE_TYPE_GAME_EVENT):
			message["message_type"] = "game_event"
		}
	}

	// 處理嵌套 map
	for _, value := range message {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			processMessageMap(nestedMap)
		} else if nestedMaps, ok := value.([]interface{}); ok {
			for _, item := range nestedMaps {
				if itemMap, ok := item.(map[string]interface{}); ok {
					processMessageMap(itemMap)
				}
			}
		}
	}
}

// 初始化數值轉字符串函數
func init() {
	// 使用標準庫函數替換适配器中的佔位實現
	if _, err := strconv.ParseInt("0", 10, 64); err == nil {
		// 如果標準庫可用，則我們這裡不需要做任何事
		// adapter.go 中的 itoa 和 ftoa 函數將在代碼生成後被替換
	}
}
