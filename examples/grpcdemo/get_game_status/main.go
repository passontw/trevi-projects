package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	pb "g38_lottery_service/internal/proto/generated/dealer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

func main() {
	// 解析命令行參數，獲取服務器地址
	serverAddr := "localhost:9100"
	if len(os.Args) > 1 {
		serverAddr = os.Args[1]
	}

	// 檢查服務器是否可達
	fmt.Printf("請求:\n{}\n\n")
	fmt.Printf("檢查服務器 %s 是否可連接...\n", serverAddr)
	conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
	if err != nil {
		fmt.Printf("錯誤: 無法連接到服務器 %s: %v\n", serverAddr, err)
		os.Exit(1)
	}
	conn.Close()
	fmt.Println("TCP 連接成功，服務器可達\n")

	// 設置 gRPC 連接選項
	var kacp = keepalive.ClientParameters{
		Time:                15 * time.Second, // 每 15 秒發送 ping (增加間隔時間)
		Timeout:             5 * time.Second,  // 等待 ping ack 的時間 (增加超時時間)
		PermitWithoutStream: true,             // 即使沒有 RPC，也允許 ping
	}

	// 設置 gRPC 連接
	fmt.Printf("連接 gRPC 服務器 %s...\n", serverAddr)
	startTime := time.Now()
	grpcConn, err := grpc.Dial(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kacp),
		grpc.WithBlock(),
		grpc.WithTimeout(15*time.Second),
	)
	if err != nil {
		fmt.Printf("錯誤: 無法建立 gRPC 連接: %v\n", err)
		st, ok := status.FromError(err)
		if ok {
			fmt.Printf("gRPC 錯誤碼: %s\n", st.Code())
			fmt.Printf("gRPC 錯誤信息: %s\n", st.Message())
		}
		os.Exit(1)
	}
	connTime := time.Since(startTime)
	fmt.Printf("gRPC 連接成功! (連接時間: %v)\n", connTime)
	defer grpcConn.Close()

	// 創建 gRPC 客戶端
	client := pb.NewDealerServiceClient(grpcConn)

	// 發送請求
	fmt.Println("發送請求 GetGameStatus...")
	reqStartTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 呼叫 GetGameStatus 方法
	resp, err := client.GetGameStatus(ctx, &pb.GetGameStatusRequest{})
	if err != nil {
		fmt.Printf("錯誤: 請求失敗: %v\n", err)
		st, ok := status.FromError(err)
		if ok {
			fmt.Printf("gRPC 錯誤碼: %s\n", st.Code())
			fmt.Printf("gRPC 錯誤信息: %s\n", st.Message())
		}
		os.Exit(1)
	}
	respTime := time.Since(reqStartTime)
	fmt.Printf("\n請求成功! (回應時間: %v)\n\n", respTime)

	// 獲取遊戲數據
	gameData := resp.GetGameData()
	if gameData == nil {
		fmt.Println("錯誤: 服務器返回的遊戲數據為空")
		os.Exit(1)
	}

	// 顯示基本遊戲信息
	fmt.Printf("遊戲 ID: %s\n", gameData.GetGameId())
	fmt.Printf("遊戲當前階段: %s\n", gameData.GetCurrentStage())

	// 顯示時間信息
	if startTime := gameData.GetStartTime(); startTime != nil {
		fmt.Printf("遊戲開始時間: %s\n", startTime.AsTime().UTC().Format(time.RFC3339))
	}
	if endTime := gameData.GetEndTime(); endTime != nil && endTime.AsTime().Unix() > 0 {
		fmt.Printf("遊戲結束時間: %s\n", endTime.AsTime().UTC().Format(time.RFC3339))
	} else {
		fmt.Printf("遊戲結束時間: %s\n", time.Unix(0, 0).Format(time.RFC3339))
	}

	// 顯示遊戲狀態
	fmt.Printf("是否有JP: %v\n", gameData.GetHasJackpot())
	fmt.Printf("是否已取消: %v\n", gameData.GetIsCancelled())
	if lastUpdateTime := gameData.GetLastUpdateTime(); lastUpdateTime != nil {
		fmt.Printf("最後更新時間: %s\n", lastUpdateTime.AsTime().UTC().Format(time.RFC3339))
	}
	fmt.Println()

	// 顯示球信息
	displayBalls("常規球", gameData.GetRegularBalls())
	displayBalls("額外球", gameData.GetExtraBalls())
	displayBalls("JP球", gameData.GetJackpotBalls())
	displayBalls("幸運號碼球", gameData.GetLuckyBalls())

	// 顯示完整的 JSON 格式
	fmt.Println("完整回應 JSON 格式:")
	jsonData, err := json.MarshalIndent(gameData, "", "  ")
	if err != nil {
		fmt.Printf("錯誤: 無法轉換為 JSON: %v\n", err)
	} else {
		fmt.Println(string(jsonData))
	}
}

// 顯示球列表
func displayBalls(title string, balls []*pb.Ball) {
	fmt.Printf("%s (總數: %d):\n", title, len(balls))
	for i, ball := range balls {
		fmt.Printf("  %d. 球號: %d", i+1, ball.GetNumber())
		if ball.GetTimestamp() != nil {
			fmt.Printf(", 時間: %s", ball.GetTimestamp().AsTime().UTC().Format(time.RFC3339))
		}
		fmt.Printf(", 是否最後一個: %v", ball.GetIsLast())
		fmt.Println()
	}
	fmt.Println()
}
