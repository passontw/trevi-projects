package mq

import (
	"strconv"
	"time"

	"g38_lottery_service/internal/generated/api/v1/dealer"
	mqpb "g38_lottery_service/internal/generated/api/v1/mq"
	"g38_lottery_service/internal/generated/common"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// 轉換 lottery result 到 proto 消息格式
func ConvertToLotteryResultMessage(gameID string, result map[string]interface{}) *mqpb.LotteryResultMessage {
	baseMessage := createBaseMessage(gameID, mqpb.MessageType_MESSAGE_TYPE_LOTTERY_RESULT)

	// 創建結果消息
	resultMessage := &mqpb.LotteryResultMessage{
		Base: baseMessage,
	}

	// 轉換抽出的球（若存在）
	if drawnBalls, ok := result["drawn_balls"].([]interface{}); ok {
		resultMessage.DrawnBalls = make([]*dealer.Ball, 0, len(drawnBalls))
		for _, ball := range drawnBalls {
			if ballMap, ok := ball.(map[string]interface{}); ok {
				resultMessage.DrawnBalls = append(resultMessage.DrawnBalls, convertMapToBall(ballMap))
			}
		}
	}

	// 轉換額外球（若存在）
	if extraBalls, ok := result["extra_balls"].(map[string]interface{}); ok {
		resultMessage.ExtraBalls = make(map[string]*dealer.Ball)
		for key, ball := range extraBalls {
			if ballMap, ok := ball.(map[string]interface{}); ok {
				resultMessage.ExtraBalls[key] = convertMapToBall(ballMap)
			}
		}
	}

	// 轉換頭獎球（若存在）
	if jackpotBall, ok := result["jackpot_ball"].(map[string]interface{}); ok {
		resultMessage.JackpotBall = convertMapToBall(jackpotBall)
	}

	// 轉換幸運球（若存在）
	if luckyBalls, ok := result["lucky_balls"].([]interface{}); ok {
		resultMessage.LuckyBalls = make([]*dealer.Ball, 0, len(luckyBalls))
		for _, ball := range luckyBalls {
			if ballMap, ok := ball.(map[string]interface{}); ok {
				resultMessage.LuckyBalls = append(resultMessage.LuckyBalls, convertMapToBall(ballMap))
			}
		}
	}

	// 轉換獎項信息（若存在）
	if prizes, ok := result["prizes"].(map[string]interface{}); ok {
		resultMessage.Prizes = make(map[string]*mqpb.PrizeInfo)
		for key, prize := range prizes {
			if prizeMap, ok := prize.(map[string]interface{}); ok {
				resultMessage.Prizes[key] = convertMapToPrizeInfo(prizeMap)
			}
		}
	}

	return resultMessage
}

// 轉換 lottery status 到 proto 消息格式
func ConvertToLotteryStatusMessage(gameID string, status string, details map[string]interface{}) *mqpb.LotteryStatusMessage {
	baseMessage := createBaseMessage(gameID, mqpb.MessageType_MESSAGE_TYPE_LOTTERY_STATUS)

	// 創建狀態消息
	statusMessage := &mqpb.LotteryStatusMessage{
		Base:    baseMessage,
		Status:  status,
		Details: make(map[string]string),
	}

	// 提取階段信息（若存在）
	if stageStr, ok := details["stage"].(string); ok {
		statusMessage.Stage = convertStringToGameStage(stageStr)
	}

	// 提取狀態訊息（若存在）
	if msg, ok := details["message"].(string); ok {
		statusMessage.Message = msg
	}

	// 提取其他詳情，並轉換為字符串形式
	for key, value := range details {
		if key != "stage" && key != "message" {
			switch v := value.(type) {
			case string:
				statusMessage.Details[key] = v
			case int, int32, int64, float32, float64, bool:
				statusMessage.Details[key] = toString(v)
			}
		}
	}

	return statusMessage
}

// 轉換 game snapshot 到 proto 消息格式
func ConvertToGameSnapshotMessage(gameID string, snapshot map[string]interface{}) *mqpb.GameSnapshotMessage {
	baseMessage := createBaseMessage(gameID, mqpb.MessageType_MESSAGE_TYPE_STAGE_CHANGE)

	// 創建快照消息
	snapshotMessage := &mqpb.GameSnapshotMessage{
		Base:      baseMessage,
		ExtraData: make(map[string]string),
	}

	// 提取事件類型
	if eventType, ok := snapshot["message_type"].(string); ok {
		snapshotMessage.EventType = eventType
	}

	// 提取遊戲數據（若存在）
	if gameData, ok := snapshot["snapshot"].(map[string]interface{}); ok {
		snapshotMessage.GameData = convertMapToGameData(gameData)
	}

	// 提取其他額外數據
	for key, value := range snapshot {
		if key != "message_type" && key != "snapshot" && key != "game_id" && key != "timestamp" {
			switch v := value.(type) {
			case string:
				snapshotMessage.ExtraData[key] = v
			case int, int32, int64, float32, float64, bool:
				snapshotMessage.ExtraData[key] = toString(v)
			}
		}
	}

	return snapshotMessage
}

// 創建基本消息
func createBaseMessage(gameID string, messageType mqpb.MessageType) *mqpb.BaseMessage {
	return &mqpb.BaseMessage{
		MessageType: messageType,
		Timestamp:   time.Now().Unix(),
		GameId:      gameID,
	}
}

// 將 map 轉換為 Ball 對象
func convertMapToBall(ballMap map[string]interface{}) *dealer.Ball {
	ball := &dealer.Ball{}

	// 提取字段
	if number, ok := getIntValue(ballMap, "number"); ok {
		ball.Number = number
	}

	// 處理球的類型
	if typeStr, ok := ballMap["type"].(string); ok {
		switch typeStr {
		case "REGULAR":
			ball.Type = dealer.BallType_BALL_TYPE_REGULAR
		case "EXTRA":
			ball.Type = dealer.BallType_BALL_TYPE_EXTRA
		case "JACKPOT":
			ball.Type = dealer.BallType_BALL_TYPE_JACKPOT
		case "LUCKY":
			ball.Type = dealer.BallType_BALL_TYPE_LUCKY
		default:
			ball.Type = dealer.BallType_BALL_TYPE_UNSPECIFIED
		}
	}

	// 處理是否為最後一個球
	if isLast, ok := ballMap["is_last"].(bool); ok {
		ball.IsLast = isLast
	}

	// 處理時間戳
	if ts, ok := ballMap["timestamp"].(time.Time); ok {
		ball.Timestamp = &timestamppb.Timestamp{
			Seconds: ts.Unix(),
			Nanos:   int32(ts.Nanosecond()),
		}
	} else if tsInt, ok := getInt64Value(ballMap, "timestamp"); ok {
		ball.Timestamp = &timestamppb.Timestamp{
			Seconds: tsInt,
			Nanos:   0,
		}
	}

	return ball
}

// 將 map 轉換為 PrizeInfo 對象
func convertMapToPrizeInfo(prizeMap map[string]interface{}) *mqpb.PrizeInfo {
	prize := &mqpb.PrizeInfo{}

	// 提取字段
	if name, ok := prizeMap["name"].(string); ok {
		prize.Name = name
	}

	if level, ok := getIntValue(prizeMap, "level"); ok {
		prize.Level = level
	}

	if amount, ok := getInt64Value(prizeMap, "amount"); ok {
		prize.Amount = amount
	}

	if winnersCount, ok := getIntValue(prizeMap, "winners_count"); ok {
		prize.WinnersCount = winnersCount
	}

	return prize
}

// 將 map 轉換為 GameData 對象
func convertMapToGameData(dataMap map[string]interface{}) *dealer.GameData {
	gameData := &dealer.GameData{}

	// 提取字段
	if id, ok := dataMap["id"].(string); ok {
		gameData.Id = id
	}

	if roomId, ok := dataMap["room_id"].(string); ok {
		gameData.RoomId = roomId
	}

	if stageStr, ok := dataMap["stage"].(string); ok {
		gameData.Stage = convertStringToGameStage(stageStr)
	}

	if statusInt, ok := getIntValue(dataMap, "status"); ok {
		gameData.Status = dealer.GameStatus(statusInt)
	}

	// 其他字段如 drawn_balls, extra_balls 等需要更複雜的轉換，根據需要進行實現

	return gameData
}

// 轉換字符串為 GameStage
func convertStringToGameStage(stage string) common.GameStage {
	switch stage {
	case "PREPARATION":
		return common.GameStage_GAME_STAGE_PREPARATION
	case "NEW_ROUND":
		return common.GameStage_GAME_STAGE_NEW_ROUND
	case "CARD_PURCHASE_OPEN":
		return common.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN
	case "CARD_PURCHASE_CLOSE":
		return common.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE
	case "DRAWING_START":
		return common.GameStage_GAME_STAGE_DRAWING_START
	case "DRAWING_CLOSE":
		return common.GameStage_GAME_STAGE_DRAWING_CLOSE
	case "PAYOUT_SETTLEMENT":
		return common.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT
	case "JACKPOT_START":
		return common.GameStage_GAME_STAGE_JACKPOT_START
	case "GAME_OVER":
		return common.GameStage_GAME_STAGE_GAME_OVER
	default:
		return common.GameStage_GAME_STAGE_UNSPECIFIED
	}
}

// 獲取 int32 值
func getIntValue(m map[string]interface{}, key string) (int32, bool) {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return int32(val), true
		case int32:
			return val, true
		case int64:
			return int32(val), true
		case float32:
			return int32(val), true
		case float64:
			return int32(val), true
		}
	}
	return 0, false
}

// 獲取 int64 值
func getInt64Value(m map[string]interface{}, key string) (int64, bool) {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return int64(val), true
		case int32:
			return int64(val), true
		case int64:
			return val, true
		case float32:
			return int64(val), true
		case float64:
			return int64(val), true
		}
	}
	return 0, false
}

// 將值轉換為字符串
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return itoa(int64(val))
	case int32:
		return itoa(int64(val))
	case int64:
		return itoa(val)
	case float32:
		return ftoa(float64(val))
	case float64:
		return ftoa(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// 更新 itoa 函數使用標準庫實現
func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// 更新 ftoa 函數使用標準庫實現
func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
