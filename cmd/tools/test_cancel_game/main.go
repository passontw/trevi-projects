package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 解析命令行參數
	serverAddr := flag.String("server", "127.0.0.1:9100", "gRPC 服務器地址")
	roomID := flag.String("room", "SG01", "房間 ID")
	timeout := flag.Int("timeout", 60, "超時時間（秒）")
	flag.Parse()

	// 創建 gRPC 連接
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("無法連接到服務器: %v", err)
	}
	defer conn.Close()

	fmt.Printf("已連接到服務器: %s\n", *serverAddr)

	// 創建 DealerService 客戶端
	client := dealerpb.NewDealerServiceClient(conn)

	// 創建上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	// 發送 CancelGame 請求
	fmt.Printf("正在發送 CancelGame 請求 (房間ID: %s)...\n", *roomID)
	fmt.Printf("超時設置為: %d 秒\n", *timeout)

	response, err := client.CancelGame(ctx, &dealerpb.CancelGameRequest{
		RoomId: *roomID,
	})

	// 輸出結果
	if err != nil {
		log.Fatalf("CancelGame 呼叫失敗: %v", err)
	}

	// 格式化輸出響應
	fmt.Println("收到響應:")
	fmt.Printf("  GameID: %s\n", response.GameData.GameId)
	fmt.Printf("  RoomID: %s\n", response.GameData.RoomId)
	fmt.Printf("  Stage: %s\n", response.GameData.Stage.String())
	fmt.Printf("  Status: %s\n", response.GameData.Status.String())
	fmt.Printf("  Regular Balls: %d\n", len(response.GameData.RegularBalls))
	fmt.Printf("  Extra Balls: %d\n", len(response.GameData.ExtraBalls))

	fmt.Println("CancelGame 測試完成")
}
