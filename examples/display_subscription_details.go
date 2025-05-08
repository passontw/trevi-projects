package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	pb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"
)

func main() {
	// 連接到gRPC服務器
	conn, err := grpc.Dial("127.0.0.1:9100", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("無法連接到gRPC服務器: %v", err)
	}
	defer conn.Close()

	// 創建DealerService客戶端
	client := pb.NewDealerServiceClient(conn)

	// 訂閱遊戲事件
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.SubscribeGameEvents(ctx, &pb.SubscribeGameEventsRequest{})
	if err != nil {
		log.Fatalf("訂閱遊戲事件失敗: %v", err)
	}

	fmt.Println("已成功訂閱遊戲事件流，等待通知...")
	fmt.Println("使用 Ctrl+C 停止監聽")
	fmt.Println("===================================")

	// 持續接收事件
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("接收事件時發生錯誤: %v", err)
		}

		// 只處理通知事件
		if event.EventType == pb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION {
			notification := event.GetNotification()
			if notification != nil && notification.GameData != nil {
				gameData := notification.GameData

				// 格式化並輸出遊戲資訊
				timestamp := time.Now().Format("15:04:05")

				// 顯示選擇的額外球一側
				selectedSide := gameData.SelectedSide
				selectedSideStr := "未選擇"

				if selectedSide == pb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT {
					selectedSideStr = "左側 (LEFT)"
				} else if selectedSide == pb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT {
					selectedSideStr = "右側 (RIGHT)"
				}

				// 嘗試通過JSON轉換獲取完整字段
				var extraData map[string]interface{}
				marshaler := protojson.MarshalOptions{
					EmitUnpopulated: true,
					UseProtoNames:   true,
				}

				// 將proto消息轉為JSON
				jsonBytes, _ := marshaler.Marshal(gameData)

				// 解析JSON以檢查是否有extra_ball_count字段
				json.Unmarshal(jsonBytes, &extraData)

				// 額外球設置數量可能的獲取方式
				extraBallCount := 0

				// 檢查JSON中是否有extra_ball_count字段
				if val, ok := extraData["extra_ball_count"]; ok {
					if numVal, ok := val.(float64); ok {
						extraBallCount = int(numVal)
					}
				}

				// 打印信息
				fmt.Printf("\n[%s] 收到遊戲通知\n", timestamp)
				fmt.Printf("遊戲ID: %s\n", gameData.GameId)
				fmt.Printf("遊戲階段: %s\n", gameData.CurrentStage)

				if extraBallCount > 0 {
					fmt.Printf("設定額外球數量: %d 個\n", extraBallCount)
				}

				fmt.Printf("已抽出額外球數量: %d 個\n", len(gameData.ExtraBalls))
				fmt.Printf("選擇的額外球一側: %s\n", selectedSideStr)

				// 打印所有額外球資訊
				if len(gameData.ExtraBalls) > 0 {
					fmt.Println("\n額外球資訊:")
					for i, ball := range gameData.ExtraBalls {
						fmt.Printf("  球 #%d: 號碼 %d\n", i+1, ball.Number)
					}
				}

				// 將完整的解析後數據轉成JSON並打印 (可選)
				if false { // 設為true時才會打印完整JSON
					jsonData, _ := json.MarshalIndent(extraData, "", "  ")
					fmt.Printf("\nJSON詳情: %s\n", string(jsonData))
				}

				fmt.Println("\n===================================")
			}
		} else if event.EventType == pb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT {
			// 可選的心跳事件處理
			timestamp := time.Now().Format("15:04:05")
			fmt.Printf("[%s] 收到心跳信號\n", timestamp)
		}
	}
}
