package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	pb "g38_lottery_service/internal/proto/dealer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 命令參數處理
	var eventTypes []pb.GameEventType
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			eventType := parseEventType(arg)
			if eventType != pb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED {
				eventTypes = append(eventTypes, eventType)
			}
		}
	}

	// 如果未指定事件類型，則訂閱所有事件
	if len(eventTypes) == 0 {
		fmt.Println("未指定事件類型，將訂閱所有遊戲事件")
		eventTypes = []pb.GameEventType{
			pb.GameEventType_GAME_EVENT_TYPE_STAGE_CHANGED,
			pb.GameEventType_GAME_EVENT_TYPE_GAME_CREATED,
			pb.GameEventType_GAME_EVENT_TYPE_GAME_CANCELLED,
			pb.GameEventType_GAME_EVENT_TYPE_GAME_COMPLETED,
			pb.GameEventType_GAME_EVENT_TYPE_BALL_DRAWN,
			pb.GameEventType_GAME_EVENT_TYPE_EXTRA_BALL_SIDE_SELECTED,
		}
	}

	// 連接gRPC服務
	conn, err := grpc.Dial("localhost:9100", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("無法連接到gRPC服務: %v", err)
	}
	defer conn.Close()

	// 創建客戶端
	client := pb.NewDealerServiceClient(conn)

	fmt.Println("連接到gRPC服務成功，開始訂閱遊戲事件...")
	fmt.Println("訂閱的事件類型:")
	for _, eventType := range eventTypes {
		fmt.Printf("  - %s\n", eventType)
	}
	fmt.Println("\n等待事件...(按Ctrl+C退出)")

	// 創建訂閱請求
	req := &pb.SubscribeGameEventsRequest{
		EventTypes: eventTypes,
	}

	// 設置上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 訂閱事件流
	stream, err := client.SubscribeGameEvents(ctx, req)
	if err != nil {
		log.Fatalf("訂閱事件流失敗: %v", err)
	}

	// 處理中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 創建一個goroutine來接收事件
	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Printf("接收事件時發生錯誤: %v", err)
				return
			}

			// 將事件轉為JSON格式顯示
			eventJSON, _ := formatEventAsJSON(event)
			fmt.Printf("\n接收到事件: %s\n%s\n", event.EventType, eventJSON)
		}
	}()

	// 等待中斷信號
	<-sigChan
	fmt.Println("\n收到中斷信號，停止訂閱...")
}

// 將事件格式化為JSON字符串
func formatEventAsJSON(event *pb.GameEvent) (string, error) {
	var payload interface{}

	switch event.EventType {
	case pb.GameEventType_GAME_EVENT_TYPE_STAGE_CHANGED:
		if event.GetStageChanged() != nil {
			payload = event.GetStageChanged()
		}
	case pb.GameEventType_GAME_EVENT_TYPE_GAME_CREATED:
		if event.GetGameCreated() != nil {
			payload = event.GetGameCreated()
		}
	case pb.GameEventType_GAME_EVENT_TYPE_GAME_CANCELLED:
		if event.GetGameCancelled() != nil {
			payload = event.GetGameCancelled()
		}
	case pb.GameEventType_GAME_EVENT_TYPE_GAME_COMPLETED:
		if event.GetGameCompleted() != nil {
			payload = event.GetGameCompleted()
		}
	case pb.GameEventType_GAME_EVENT_TYPE_BALL_DRAWN:
		if event.GetBallDrawn() != nil {
			payload = event.GetBallDrawn()
		}
	case pb.GameEventType_GAME_EVENT_TYPE_EXTRA_BALL_SIDE_SELECTED:
		if event.GetExtraBallSideSelected() != nil {
			payload = event.GetExtraBallSideSelected()
		}
	default:
		return fmt.Sprintf("未知事件類型: %s", event.EventType), nil
	}

	// 轉換為JSON
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// 解析事件類型參數
func parseEventType(typeStr string) pb.GameEventType {
	typeStr = strings.ToUpper(typeStr)

	switch typeStr {
	case "STAGE_CHANGED", "STAGE":
		return pb.GameEventType_GAME_EVENT_TYPE_STAGE_CHANGED
	case "GAME_CREATED", "CREATE", "NEW":
		return pb.GameEventType_GAME_EVENT_TYPE_GAME_CREATED
	case "GAME_CANCELLED", "CANCEL":
		return pb.GameEventType_GAME_EVENT_TYPE_GAME_CANCELLED
	case "GAME_COMPLETED", "COMPLETE":
		return pb.GameEventType_GAME_EVENT_TYPE_GAME_COMPLETED
	case "BALL_DRAWN", "BALL":
		return pb.GameEventType_GAME_EVENT_TYPE_BALL_DRAWN
	case "EXTRA_BALL_SIDE_SELECTED", "SIDE":
		return pb.GameEventType_GAME_EVENT_TYPE_EXTRA_BALL_SIDE_SELECTED
	case "ALL":
		// 特殊處理 ALL 在調用函數處理
		return pb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	default:
		fmt.Printf("警告: 未知的事件類型 '%s'，將被忽略\n", typeStr)
		return pb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED
	}
}
