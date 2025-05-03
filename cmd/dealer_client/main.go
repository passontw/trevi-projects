package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	pb "g38_lottery_service/internal/proto/dealer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 檢查命令行參數
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	// 連接gRPC服務
	conn, err := grpc.Dial("localhost:9100", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("無法連接到gRPC服務: %v", err)
	}
	defer conn.Close()

	// 創建客戶端
	client := pb.NewDealerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 根據命令執行不同的操作
	command := os.Args[1]
	switch command {
	case "start_new_round":
		// 開始新局
		resp, err := client.StartNewRound(ctx, &pb.StartNewRoundRequest{})
		if err != nil {
			log.Fatalf("開始新局失敗: %v", err)
		}
		fmt.Printf("成功開始新局! 遊戲ID: %s, 階段: %s\n", resp.GameId, resp.CurrentStage)

	case "advance_stage":
		// 推進階段
		force := false
		if len(os.Args) > 2 && os.Args[2] == "force" {
			force = true
		}
		resp, err := client.AdvanceStage(ctx, &pb.AdvanceStageRequest{Force: force})
		if err != nil {
			log.Fatalf("推進階段失敗: %v", err)
		}
		fmt.Printf("成功推進階段! 從 %s 到 %s\n", resp.OldStage, resp.NewStage)

	case "get_status":
		// 獲取遊戲狀態
		resp, err := client.GetGameStatus(ctx, &pb.GetGameStatusRequest{})
		if err != nil {
			log.Fatalf("獲取遊戲狀態失敗: %v", err)
		}
		game := resp.GameData
		fmt.Printf("遊戲ID: %s\n", game.GameId)
		fmt.Printf("階段: %s\n", game.CurrentStage)
		fmt.Printf("已抽取的常規球: %d\n", len(game.RegularBalls))
		fmt.Printf("已抽取的額外球: %d\n", len(game.ExtraBalls))
		fmt.Printf("已抽取的JP球: %d\n", len(game.JackpotBalls))
		fmt.Printf("已抽取的幸運球: %d\n", len(game.LuckyBalls))
		fmt.Printf("選擇的額外球一側: %s\n", game.SelectedSide)
		fmt.Printf("是否有JP: %t\n", game.HasJackpot)
		fmt.Printf("JP獲獎者: %s\n", game.JackpotWinner)
		fmt.Printf("是否已取消: %t\n", game.IsCancelled)

	case "draw_ball":
		// 需要檢查參數
		if len(os.Args) < 3 {
			fmt.Println("錯誤: 缺少球號參數")
			fmt.Println("用法: dealer_client draw_ball <球號> [is_last]")
			os.Exit(1)
		}

		// 解析球號
		number, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("無效的球號: %v", err)
		}

		// 是否是最後一個球
		isLast := false
		if len(os.Args) > 3 && os.Args[3] == "last" {
			isLast = true
		}

		// 抽取球
		resp, err := client.DrawBall(ctx, &pb.DrawBallRequest{Number: int32(number), IsLast: isLast})
		if err != nil {
			log.Fatalf("抽取球失敗: %v", err)
		}
		fmt.Printf("成功抽取球! 號碼: %d, 類型: %s, 是否最後一個: %t\n",
			resp.Ball.Number, resp.Ball.Type, resp.Ball.IsLast)

	case "draw_extra_ball":
		// 需要檢查參數
		if len(os.Args) < 3 {
			fmt.Println("錯誤: 缺少球號參數")
			fmt.Println("用法: dealer_client draw_extra_ball <球號> [is_last]")
			os.Exit(1)
		}

		// 解析球號
		number, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("無效的球號: %v", err)
		}

		// 是否是最後一個球
		isLast := false
		if len(os.Args) > 3 && os.Args[3] == "last" {
			isLast = true
		}

		// 抽取額外球
		resp, err := client.DrawExtraBall(ctx, &pb.DrawExtraBallRequest{Number: int32(number), IsLast: isLast})
		if err != nil {
			log.Fatalf("抽取額外球失敗: %v", err)
		}
		fmt.Printf("成功抽取額外球! 號碼: %d, 類型: %s, 是否最後一個: %t\n",
			resp.Ball.Number, resp.Ball.Type, resp.Ball.IsLast)

	case "draw_jackpot_ball":
		// 需要檢查參數
		if len(os.Args) < 3 {
			fmt.Println("錯誤: 缺少球號參數")
			fmt.Println("用法: dealer_client draw_jackpot_ball <球號> [is_last]")
			os.Exit(1)
		}

		// 解析球號
		number, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("無效的球號: %v", err)
		}

		// 是否是最後一個球
		isLast := false
		if len(os.Args) > 3 && os.Args[3] == "last" {
			isLast = true
		}

		// 抽取JP球
		resp, err := client.DrawJackpotBall(ctx, &pb.DrawJackpotBallRequest{Number: int32(number), IsLast: isLast})
		if err != nil {
			log.Fatalf("抽取JP球失敗: %v", err)
		}
		fmt.Printf("成功抽取JP球! 號碼: %d, 類型: %s, 是否最後一個: %t\n",
			resp.Ball.Number, resp.Ball.Type, resp.Ball.IsLast)

	case "draw_lucky_ball":
		// 需要檢查參數
		if len(os.Args) < 3 {
			fmt.Println("錯誤: 缺少球號參數")
			fmt.Println("用法: dealer_client draw_lucky_ball <球號> [is_last]")
			os.Exit(1)
		}

		// 解析球號
		number, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("無效的球號: %v", err)
		}

		// 是否是最後一個球
		isLast := false
		if len(os.Args) > 3 && os.Args[3] == "last" {
			isLast = true
		}

		// 抽取幸運球
		resp, err := client.DrawLuckyBall(ctx, &pb.DrawLuckyBallRequest{Number: int32(number), IsLast: isLast})
		if err != nil {
			log.Fatalf("抽取幸運球失敗: %v", err)
		}
		fmt.Printf("成功抽取幸運球! 號碼: %d, 類型: %s, 是否最後一個: %t\n",
			resp.Ball.Number, resp.Ball.Type, resp.Ball.IsLast)

	case "set_jackpot":
		// 需要檢查參數
		if len(os.Args) < 3 {
			fmt.Println("錯誤: 缺少參數")
			fmt.Println("用法: dealer_client set_jackpot <true|false>")
			os.Exit(1)
		}

		// 解析是否有JP
		hasJackpot := false
		if os.Args[2] == "true" {
			hasJackpot = true
		}

		// 設置JP狀態
		game, err := client.SetHasJackpot(ctx, &pb.SetHasJackpotRequest{HasJackpot: hasJackpot})
		if err != nil {
			log.Fatalf("設置JP狀態失敗: %v", err)
		}
		fmt.Printf("成功設置JP狀態為 %t! 遊戲ID: %s\n", hasJackpot, game.GameId)

	case "notify_jackpot_winner":
		// 需要檢查參數
		if len(os.Args) < 3 {
			fmt.Println("錯誤: 缺少贏家ID參數")
			fmt.Println("用法: dealer_client notify_jackpot_winner <贏家ID>")
			os.Exit(1)
		}

		// 通知JP贏家
		game, err := client.NotifyJackpotWinner(ctx, &pb.NotifyJackpotWinnerRequest{WinnerId: os.Args[2]})
		if err != nil {
			log.Fatalf("通知JP贏家失敗: %v", err)
		}
		fmt.Printf("成功通知JP贏家! 贏家ID: %s, 遊戲ID: %s\n", game.JackpotWinner, game.GameId)

	case "cancel_game":
		// 需要檢查參數
		if len(os.Args) < 3 {
			fmt.Println("錯誤: 缺少取消原因參數")
			fmt.Println("用法: dealer_client cancel_game <取消原因>")
			os.Exit(1)
		}

		// 取消遊戲
		game, err := client.CancelGame(ctx, &pb.CancelGameRequest{Reason: os.Args[2]})
		if err != nil {
			log.Fatalf("取消遊戲失敗: %v", err)
		}
		fmt.Printf("成功取消遊戲! 遊戲ID: %s, 原因: %s\n", game.GameId, game.CancelReason)

	default:
		fmt.Printf("未知命令: %s\n", command)
		showUsage()
		os.Exit(1)
	}
}

// 顯示用法說明
func showUsage() {
	fmt.Println("用法: dealer_client <命令> [參數...]")
	fmt.Println("\n可用命令:")
	fmt.Println("  start_new_round          - 開始新局")
	fmt.Println("  advance_stage [force]    - 推進遊戲階段，可選參數 force 強制推進")
	fmt.Println("  get_status               - 獲取當前遊戲狀態")
	fmt.Println("  draw_ball <球號> [last]   - 抽取常規球，可選參數 last 表示最後一個球")
	fmt.Println("  draw_extra_ball <球號> [last] - 抽取額外球，可選參數 last 表示最後一個球")
	fmt.Println("  draw_jackpot_ball <球號> [last] - 抽取JP球，可選參數 last 表示最後一個球")
	fmt.Println("  draw_lucky_ball <球號> [last] - 抽取幸運球，可選參數 last 表示最後一個球")
	fmt.Println("  set_jackpot <true|false> - 設置是否有JP")
	fmt.Println("  notify_jackpot_winner <贏家ID> - 通知JP贏家")
	fmt.Println("  cancel_game <原因>       - 取消遊戲")
	fmt.Println("\n示例:")
	fmt.Println("  dealer_client start_new_round")
	fmt.Println("  dealer_client draw_ball 42")
	fmt.Println("  dealer_client draw_ball 99 last")
	fmt.Println("  dealer_client set_jackpot true")
	fmt.Println("  dealer_client notify_jackpot_winner player123")
	fmt.Println("  dealer_client cancel_game \"設備故障\"")
}
