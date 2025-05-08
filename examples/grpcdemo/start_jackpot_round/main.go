package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	pb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

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
		Time:                15 * time.Second, // 每 15 秒發送 ping
		Timeout:             5 * time.Second,  // 等待 ping ack 的時間
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
	fmt.Println("發送 StartJackpotRound 請求...")
	reqStartTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 呼叫 StartJackpotRound 方法
	resp, err := client.StartJackpotRound(ctx, &pb.StartJackpotRoundRequest{})
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

	// 顯示回應結果
	fmt.Printf("操作結果: %v\n", resp.GetSuccess())
	fmt.Printf("遊戲 ID: %s\n", resp.GetGameId())
	fmt.Printf("舊階段: %s\n", resp.GetOldStage())
	fmt.Printf("新階段: %s\n", resp.GetNewStage())
}
