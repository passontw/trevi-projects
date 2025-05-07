package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "g38_lottery_service/internal/proto/generated/dealer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

func main() {
	// 顯示請求訊息
	req := &pb.StartNewRoundRequest{}
	reqJSON, _ := json.MarshalIndent(req, "", "  ")
	fmt.Printf("請求:\n%s\n\n", reqJSON)

	// 設定 gRPC 連接參數
	kacp := keepalive.ClientParameters{
		Time:                15 * time.Second, // 每15秒發送ping (增加間隔時間)
		Timeout:             5 * time.Second,  // 如果ping 5秒內未收到回應，視為連接斷開
		PermitWithoutStream: true,             // 允許在沒有活躍RPC的情況下發送ping
	}

	// 連接 gRPC 服務器
	serverAddr := "localhost:9100"
	fmt.Printf("連接 gRPC 服務器 %s...\n", serverAddr)

	// 設定連接超時為15秒
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(
		dialCtx,
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kacp),
		grpc.WithBlock(), // 等待直到連接建立
	)
	if err != nil {
		fmt.Printf("無法連接 gRPC 服務器: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("連接成功!\n")

	// 創建 gRPC 客戶端
	client := pb.NewDealerServiceClient(conn)

	// 設定請求超時為60秒
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 發送請求
	fmt.Printf("發送請求 StartNewRound...\n\n")
	resp, err := client.StartNewRound(ctx, req)

	// 錯誤處理
	if err != nil {
		fmt.Printf("請求失敗: %v\n", err)

		// 獲取更詳細的 gRPC 錯誤信息
		if st, ok := status.FromError(err); ok {
			fmt.Printf("錯誤代碼: %s\n", st.Code())
			fmt.Printf("錯誤信息: %s\n", st.Message())

			// 顯示詳細信息
			for _, detail := range st.Details() {
				fmt.Printf("詳細信息: %v\n", detail)
			}
		}
		return
	}

	// 格式化輸出響應
	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("收到回應:\n%s\n", respJSON)
}
