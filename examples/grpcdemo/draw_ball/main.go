package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"g38_lottery_service/internal/proto/generated/dealer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Ball 表示一個球的數據
type Ball struct {
	Number    int    `json:"number"`
	Type      string `json:"type"`
	IsLast    bool   `json:"isLast"`
	Timestamp string `json:"timestamp"`
}

// DrawBallRequest 抽球請求
type DrawBallRequest struct {
	Balls []Ball `json:"balls"`
}

// GameStatus 遊戲狀態
type GameStatus struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// DrawnBall 抽出的球
type DrawnBall struct {
	Number    int    `json:"number"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

// ExtraBall 額外球
type ExtraBall struct {
	Number    int    `json:"number"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

// GameInfo 遊戲信息
type GameInfo struct {
	Id          string      `json:"id"`
	State       string      `json:"state"`
	StartTime   string      `json:"startTime"`
	DrawTime    string      `json:"drawTime"`
	DrawnBalls  []DrawnBall `json:"drawnBalls,omitempty"`
	ExtraBalls  []ExtraBall `json:"extraBalls,omitempty"`
	LuckyNumber []int       `json:"luckyNumber,omitempty"`
}

// DrawBallResponse 抽球回應
type DrawBallResponse struct {
	Balls      []Ball      `json:"balls,omitempty"`
	GameStatus *GameStatus `json:"gameStatus,omitempty"`
	GameInfo   *GameInfo   `json:"gameInfo,omitempty"`
}

func main() {
	// 設定命令列參數
	serverAddr := flag.String("server", "localhost:9100", "gRPC 服務器地址")
	flag.Parse()

	// 顯示連接訊息
	fmt.Printf("正在連接到 gRPC 服務器: %s\n", *serverAddr)

	// 建立 gRPC 連接
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("無法連接到 gRPC 服務器: %v", err)
	}
	defer conn.Close()

	// 建立 DealerService 客戶端
	client := dealer.NewDealerServiceClient(conn)

	// 準備抽球請求
	request := prepareBallRequest()

	// 請求前顯示訊息
	fmt.Println("\n抽球請求:")
	printRequestAsJSON(request)

	fmt.Println("\n------------------------------------------")
	fmt.Println("開始呼叫 DrawBall RPC...")
	fmt.Println("------------------------------------------\n")

	// 呼叫 DrawBall RPC
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := client.DrawBall(ctx, request)
	if err != nil {
		log.Fatalf("呼叫 DrawBall RPC 失敗: %v", err)
	}

	// 顯示回應結果
	displayResponse(response)
}

// 準備抽球請求
func prepareBallRequest() *dealer.DrawBallRequest {
	// 建立測試用的球列表
	var balls []*dealer.Ball
	now := time.Now().UTC()

	for i := 0; i < 30; i++ {
		isLast := i == 29 // 最後一個球標記為最後一顆
		ball := &dealer.Ball{
			Number:    int32(i + 1),
			Type:      dealer.BallType_BALL_TYPE_REGULAR,
			IsLast:    isLast,
			Timestamp: timestamppb.New(now),
		}
		balls = append(balls, ball)
	}

	return &dealer.DrawBallRequest{
		Balls: balls,
	}
}

// 將請求轉換為 JSON 並顯示
func printRequestAsJSON(req *dealer.DrawBallRequest) {
	// 建立可讀的結構以便轉換為 JSON
	type BallJSON struct {
		Number    int32  `json:"number"`
		Type      string `json:"type"`
		IsLast    bool   `json:"isLast"`
		Timestamp string `json:"timestamp"`
	}

	type RequestJSON struct {
		Balls []BallJSON `json:"balls"`
	}

	jsonReq := RequestJSON{
		Balls: make([]BallJSON, len(req.Balls)),
	}

	for i, ball := range req.Balls {
		jsonReq.Balls[i] = BallJSON{
			Number:    ball.Number,
			Type:      ball.Type.String(),
			IsLast:    ball.IsLast,
			Timestamp: ball.Timestamp.AsTime().Format(time.RFC3339),
		}
	}

	// 轉換為格式化的 JSON
	jsonData, err := json.MarshalIndent(jsonReq, "", "  ")
	if err != nil {
		log.Fatalf("無法轉換為 JSON: %v", err)
	}
	fmt.Println(string(jsonData))
}

// 顯示回應
func displayResponse(resp *dealer.DrawBallResponse) {
	fmt.Println("抽球回應:")

	// 顯示回應中的球
	fmt.Printf("回應中的球數量: %d\n", len(resp.Balls))
	for i, ball := range resp.Balls {
		fmt.Printf("球 #%d: 號碼 = %d, 類型 = %s, 是否最後一球 = %v\n",
			i+1, ball.Number, ball.Type.String(), ball.IsLast)
	}

	// 如果有遊戲狀態，顯示遊戲狀態
	if resp.GameStatus != nil {
		fmt.Println("\n遊戲狀態更新:")
		fmt.Printf("階段: %s\n", resp.GameStatus.Stage.String())
		fmt.Printf("訊息: %s\n", resp.GameStatus.Message)
	}

	// 顯示完整回應（轉換為 JSON 格式）
	fmt.Println("\n完整回應 JSON 格式:")

	// 為了更好的顯示，將 protobuf 消息轉換為自定義結構
	type BallJSON struct {
		Number    int32  `json:"number"`
		Type      string `json:"type"`
		IsLast    bool   `json:"isLast"`
		Timestamp string `json:"timestamp"`
	}

	type GameStatusJSON struct {
		Stage   string `json:"stage"`
		Message string `json:"message"`
	}

	type ResponseJSON struct {
		Balls      []BallJSON      `json:"balls,omitempty"`
		GameStatus *GameStatusJSON `json:"gameStatus,omitempty"`
	}

	jsonResp := ResponseJSON{
		Balls: make([]BallJSON, len(resp.Balls)),
	}

	for i, ball := range resp.Balls {
		jsonResp.Balls[i] = BallJSON{
			Number:    ball.Number,
			Type:      ball.Type.String(),
			IsLast:    ball.IsLast,
			Timestamp: ball.Timestamp.AsTime().Format(time.RFC3339),
		}
	}

	if resp.GameStatus != nil {
		jsonResp.GameStatus = &GameStatusJSON{
			Stage:   resp.GameStatus.Stage.String(),
			Message: resp.GameStatus.Message,
		}
	}

	// 轉換為格式化的 JSON
	jsonData, err := json.MarshalIndent(jsonResp, "", "  ")
	if err != nil {
		log.Fatalf("無法轉換為 JSON: %v", err)
	}
	fmt.Println(string(jsonData))
}
