package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "g38_lottery_service/internal/lottery_service/proto/generated/dealer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// 顯示時間格式化函數
func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "無時間戳"
	}
	return ts.AsTime().Format("2006-01-02 15:04:05.000")
}

// 格式化球資訊
func formatBall(ball *pb.Ball) string {
	if ball == nil {
		return "無效球"
	}

	ballType := "未知"
	switch ball.Type {
	case pb.BallType_BALL_TYPE_REGULAR:
		ballType = "常規球"
	case pb.BallType_BALL_TYPE_EXTRA:
		ballType = "額外球"
	case pb.BallType_BALL_TYPE_JACKPOT:
		ballType = "JP球"
	case pb.BallType_BALL_TYPE_LUCKY:
		ballType = "幸運球"
	}

	return fmt.Sprintf("球號: %d, 類型: %s, 是否最後一顆: %v, 時間: %s",
		ball.Number, ballType, ball.IsLast, formatTimestamp(ball.Timestamp))
}

// 格式化遊戲階段
func formatGameStage(stage pb.GameStage) string {
	switch stage {
	case pb.GameStage_GAME_STAGE_UNSPECIFIED:
		return "未指定"
	case pb.GameStage_GAME_STAGE_PREPARATION:
		return "準備階段"
	case pb.GameStage_GAME_STAGE_NEW_ROUND:
		return "新回合"
	case pb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		return "卡片購買開放"
	case pb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE:
		return "卡片購買關閉"
	case pb.GameStage_GAME_STAGE_DRAWING_START:
		return "開始抽獎"
	case pb.GameStage_GAME_STAGE_DRAWING_CLOSE:
		return "結束抽獎"
	case pb.GameStage_GAME_STAGE_EXTRA_BALL_PREPARE:
		return "額外球準備"
	case pb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_START:
		return "額外球選邊投注開始"
	case pb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED:
		return "額外球選邊投注關閉"
	case pb.GameStage_GAME_STAGE_EXTRA_BALL_WAIT_CLAIM:
		return "等待額外球領獎"
	case pb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START:
		return "額外球抽獎開始"
	case pb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_CLOSE:
		return "額外球抽獎結束"
	case pb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		return "派彩結算"
	case pb.GameStage_GAME_STAGE_JACKPOT_START:
		return "JP開始"
	case pb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START:
		return "JP抽獎開始"
	case pb.GameStage_GAME_STAGE_JACKPOT_DRAWING_CLOSED:
		return "JP抽獎結束"
	case pb.GameStage_GAME_STAGE_JACKPOT_SETTLEMENT:
		return "JP結算"
	case pb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START:
		return "幸運球抽獎開始"
	case pb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_CLOSED:
		return "幸運球抽獎結束"
	case pb.GameStage_GAME_STAGE_GAME_OVER:
		return "遊戲結束"
	default:
		return fmt.Sprintf("未知階段(%d)", stage)
	}
}

// 格式化額外球選邊
func formatExtraBallSide(side pb.ExtraBallSide) string {
	switch side {
	case pb.ExtraBallSide_EXTRA_BALL_SIDE_UNSPECIFIED:
		return "未指定"
	case pb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT:
		return "左側"
	case pb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT:
		return "右側"
	default:
		return fmt.Sprintf("未知側(%d)", side)
	}
}

// 球資訊管理（球號快取）
var (
	ballsMutex sync.Mutex
	drawnBalls []*pb.Ball // 已抽出的球清單
)

// 自動抽球控制
var (
	drawingMutex      sync.Mutex
	isDrawingActive   bool
	drawingCancelFunc context.CancelFunc
)

// 創建有序列的球號列表，前兩個球固定為11和12，其餘隨機生成
func createSequentialBallNumbers() []int32 {
	// 初始化隨機數生成器
	rand.Seed(time.Now().UnixNano())

	// 創建固定的前兩顆球號
	ballNumbers := []int32{11, 12}

	// 已使用的球號集合（用於避免重複）
	usedNumbers := map[int32]bool{
		11: true,
		12: true,
	}

	// 生成剩餘的28顆球號（總共30顆）
	for len(ballNumbers) < 30 {
		// 生成 1-75 之間的隨機數
		ballNumber := int32(rand.Intn(75) + 1)

		// 確保球號不重複
		if !usedNumbers[ballNumber] {
			ballNumbers = append(ballNumbers, ballNumber)
			usedNumbers[ballNumber] = true
		}
	}

	return ballNumbers
}

// 創建一個球對象
func createBall(number int32, isLast bool) *pb.Ball {
	return &pb.Ball{
		Number:    number,
		Type:      pb.BallType_BALL_TYPE_REGULAR,
		IsLast:    isLast,
		Timestamp: timestamppb.Now(),
	}
}

// 模擬荷官抽球函數
func dealerDrawBall(ctx context.Context, client pb.DealerServiceClient, roomId string, ballNumber int32, isLast bool, previousBalls []*pb.Ball) (*pb.Ball, error) {
	// 創建新球
	newBall := createBall(ballNumber, isLast)
	fmt.Printf("荷官抽出球號: %d\n", newBall.Number)

	// 構建完整的球列表（包含之前的球和新球）
	balls := append(previousBalls, newBall)

	// 呼叫 DrawBall 方法，通知 gRPC 服務器
	resp, err := client.DrawBall(ctx, &pb.DrawBallRequest{
		RoomId: roomId,
		Balls:  balls,
	})

	if err != nil {
		return nil, fmt.Errorf("荷官抽球失敗: %v", err)
	}

	if len(resp.Balls) == 0 {
		return nil, fmt.Errorf("收到空的球資訊")
	}

	// 返回服務器返回的最後一顆球
	return resp.Balls[len(resp.Balls)-1], nil
}

// 開始自動抽球流程
func startAutoDraw(ctx context.Context, client pb.DealerServiceClient, roomId string) {
	drawingMutex.Lock()
	if isDrawingActive {
		drawingMutex.Unlock()
		fmt.Println("自動抽球流程已在進行中")
		return
	}

	// 建立可取消的 context
	drawCtx, cancel := context.WithCancel(ctx)
	drawingCancelFunc = cancel
	isDrawingActive = true
	drawingMutex.Unlock()

	fmt.Println("===================================")
	fmt.Println("開始模擬荷官抽球流程！將在 3 秒後開始抽第一顆球...")
	fmt.Println("===================================")

	// 等待 3 秒再開始抽球
	select {
	case <-drawCtx.Done():
		stopAutoDrawing()
		return
	case <-time.After(3 * time.Second):
		// 繼續執行
	}

	// 創建球號序列
	ballNumbers := createSequentialBallNumbers()
	fmt.Printf("預設球號序列已生成，共 %d 顆球\n", len(ballNumbers))

	// 球列表，用於保存抽出的球
	var drawnBalls []*pb.Ball

	// 開始抽球循環
	for i, ballNumber := range ballNumbers {
		// 檢查是否已取消
		select {
		case <-drawCtx.Done():
			stopAutoDrawing()
			return
		default:
			// 繼續執行
		}

		// 抽一顆球
		fmt.Printf("正在抽取第 %d 顆球...\n", i+1)

		// 判斷是否是最後一顆球
		isLast := (i == len(ballNumbers)-1)

		// 模擬荷官抽球
		ball, err := dealerDrawBall(drawCtx, client, roomId, ballNumber, isLast, drawnBalls)
		if err != nil {
			fmt.Printf("抽球過程中發生錯誤: %v\n", err)
			stopAutoDrawing()
			return
		}

		// 更新已抽出的球列表
		drawnBalls = append(drawnBalls, ball)

		// 顯示抽出的球
		fmt.Printf("第 %d 顆球: %s\n", i+1, formatBall(ball))
		fmt.Println("-----------------------------------")

		// 如果是最後一顆球，結束抽球
		if isLast {
			fmt.Println("已抽出最後一顆球，抽球環節完成")
			stopAutoDrawing()
			return
		}

		// 如果不是最後一顆球，等待3秒後抽下一顆
		if i < len(ballNumbers)-1 {
			select {
			case <-drawCtx.Done():
				stopAutoDrawing()
				return
			case <-time.After(3 * time.Second):
				// 繼續下一次抽球
			}
		}
	}

	fmt.Println("===================================")
	fmt.Println("已完成所有球的抽取！")
	fmt.Println("===================================")

	stopAutoDrawing()
}

// 停止自動抽球
func stopAutoDrawing() {
	drawingMutex.Lock()
	defer drawingMutex.Unlock()

	if isDrawingActive {
		if drawingCancelFunc != nil {
			drawingCancelFunc()
			drawingCancelFunc = nil
		}
		isDrawingActive = false
	}
}

// 檢查是否要開始自動抽球
func checkForDrawingStart(event *pb.GameEvent, client pb.DealerServiceClient, roomId string) {
	// 階段變更事件，檢查是否進入抽球階段
	if stageChanged := event.GetStageChanged(); stageChanged != nil {
		if stageChanged.NewStage == pb.GameStage_GAME_STAGE_DRAWING_START {
			fmt.Println("===================================")
			fmt.Println("檢測到遊戲階段變更為 DRAWING_START！")

			// 在新的 goroutine 中啟動抽球流程
			go func() {
				// 使用背景上下文，這樣即使主程序上下文被取消，抽球也能繼續
				ctx := context.Background()
				startAutoDraw(ctx, client, roomId)
			}()
		}
	}

	// 也可以檢查通知事件中的遊戲階段
	if notification := event.GetNotification(); notification != nil {
		if gameData := notification.GameData; gameData != nil {
			if gameData.CurrentStage == pb.GameStage_GAME_STAGE_DRAWING_START {
				drawingMutex.Lock()
				alreadyDrawing := isDrawingActive
				drawingMutex.Unlock()

				if !alreadyDrawing {
					fmt.Println("===================================")
					fmt.Println("檢測到遊戲處於 DRAWING_START 階段！")

					// 在新的 goroutine 中啟動抽球流程
					go func() {
						ctx := context.Background()
						startAutoDraw(ctx, client, roomId)
					}()
				}
			}
		}
	}
}

func handleGameEvent(event *pb.GameEvent, showDetailedJson bool) {
	timestamp := time.Now().Format("15:04:05.000")

	// 基本事件資訊
	fmt.Printf("\n[%s] 事件 ID: %s, 類型: ", timestamp, event.GameId)

	switch event.EventType {
	case pb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT:
		fmt.Println("心跳")
		if heartbeat := event.GetHeartbeat(); heartbeat != nil {
			fmt.Printf("  訊息: %s\n", heartbeat.Message)
		}

	case pb.GameEventType_GAME_EVENT_TYPE_NOTIFICATION:
		fmt.Println("通知")
		if notification := event.GetNotification(); notification != nil {
			fmt.Printf("  訊息: %s\n", notification.Message)

			if gameData := notification.GameData; gameData != nil {
				fmt.Printf("  遊戲ID: %s\n", gameData.GameId)
				fmt.Printf("  當前階段: %s\n", formatGameStage(gameData.CurrentStage))
				fmt.Printf("  開始時間: %s\n", formatTimestamp(gameData.StartTime))
				fmt.Printf("  是否有JP: %v\n", gameData.HasJackpot)
				fmt.Printf("  額外球數量: %d\n", gameData.ExtraBallCount)
				fmt.Printf("  選擇的額外球一側: %s\n", formatExtraBallSide(gameData.SelectedSide))

				// 顯示球信息
				fmt.Printf("  常規球數量: %d\n", len(gameData.RegularBalls))
				fmt.Printf("  額外球數量: %d\n", len(gameData.ExtraBalls))
				fmt.Printf("  JP球數量: %d\n", len(gameData.JackpotBalls))
				fmt.Printf("  幸運球數量: %d\n", len(gameData.LuckyBalls))

				// 顯示最後一顆球的詳細信息（如果有）
				if len(gameData.RegularBalls) > 0 {
					fmt.Printf("  最後一顆常規球: %s\n", formatBall(gameData.RegularBalls[len(gameData.RegularBalls)-1]))
				}
				if len(gameData.ExtraBalls) > 0 {
					fmt.Printf("  最後一顆額外球: %s\n", formatBall(gameData.ExtraBalls[len(gameData.ExtraBalls)-1]))
				}
			}
		}

	case pb.GameEventType_GAME_EVENT_TYPE_UNSPECIFIED:
		fmt.Println("未指定")

	default:
		fmt.Printf("其他類型(%d)\n", event.EventType)

		// 根據 oneof 欄位處理不同類型的事件
		if stageChanged := event.GetStageChanged(); stageChanged != nil {
			fmt.Printf("  階段變更: %s -> %s\n",
				formatGameStage(stageChanged.OldStage),
				formatGameStage(stageChanged.NewStage))
		} else if ballDrawn := event.GetBallDrawn(); ballDrawn != nil {
			fmt.Printf("  球抽取: %s\n", formatBall(ballDrawn.Ball))
		} else if gameCreated := event.GetGameCreated(); gameCreated != nil {
			fmt.Printf("  遊戲創建: ID=%s\n", gameCreated.InitialState.GameId)
		} else if gameCancelled := event.GetGameCancelled(); gameCancelled != nil {
			fmt.Printf("  遊戲取消: 原因=%s, 時間=%s\n",
				gameCancelled.Reason,
				formatTimestamp(gameCancelled.CancelTime))
		} else if gameCompleted := event.GetGameCompleted(); gameCompleted != nil {
			fmt.Printf("  遊戲完成: ID=%s\n", gameCompleted.FinalState.GameId)
		} else if extraBallSideSelected := event.GetExtraBallSideSelected(); extraBallSideSelected != nil {
			fmt.Printf("  額外球選邊: %s\n", formatExtraBallSide(extraBallSideSelected.SelectedSide))
		}
	}

	// 顯示詳細的JSON格式（可選）
	if showDetailedJson {
		marshaler := protojson.MarshalOptions{
			EmitUnpopulated: true,
			UseProtoNames:   true,
		}

		jsonBytes, err := marshaler.Marshal(event)
		if err == nil {
			var jsonObj map[string]interface{}
			if err = json.Unmarshal(jsonBytes, &jsonObj); err == nil {
				jsonData, _ := json.MarshalIndent(jsonObj, "", "  ")
				fmt.Printf("\n完整JSON:\n%s\n", string(jsonData))
			}
		}
	}

	fmt.Println("===================================")
}

// 啟動新遊戲
func startNewGame(ctx context.Context, client pb.DealerServiceClient, roomId string) error {
	fmt.Printf("將在 10 秒後開始新遊戲 (房間: %s)...\n", roomId)

	// 等待 10 秒
	for i := 10; i > 0; i-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Printf("倒數 %d 秒...\n", i)
			time.Sleep(1 * time.Second)
		}
	}

	// 呼叫 StartNewRound 方法
	fmt.Println("開始新遊戲！")
	resp, err := client.StartNewRound(ctx, &pb.StartNewRoundRequest{
		RoomId: roomId,
	})

	if err != nil {
		return fmt.Errorf("啟動新遊戲失敗: %v", err)
	}

	fmt.Printf("遊戲啟動成功!\n")
	fmt.Printf("遊戲 ID: %s\n", resp.GameId)
	fmt.Printf("開始時間: %s\n", formatTimestamp(resp.StartTime))
	fmt.Printf("當前階段: %s\n", formatGameStage(resp.CurrentStage))
	fmt.Println("===================================")

	return nil
}

func main() {
	// 使用 flag 包處理命令行參數
	roomIdPtr := flag.String("roomid", "SG01", "房間ID，例如：SG01, SG02 等")
	showJsonPtr := flag.Bool("json", false, "是否顯示完整的JSON數據")
	serverPtr := flag.String("server", "127.0.0.1:9100", "gRPC 服務器地址")
	startGamePtr := flag.Bool("startgame", false, "是否在訂閱 10 秒後自動啟動新遊戲")
	autoDrawPtr := flag.Bool("autodraw", true, "是否在遊戲進入抽球階段後自動抽球")

	// 解析命令行參數
	flag.Parse()

	// 獲取參數值
	roomId := *roomIdPtr
	showDetailedJson := *showJsonPtr
	serverAddr := *serverPtr
	shouldStartGame := *startGamePtr
	enableAutoDraw := *autoDrawPtr

	// 顯示配置信息
	fmt.Printf("配置信息:\n")
	fmt.Printf("- 房間 ID: %s\n", roomId)
	fmt.Printf("- 服務器地址: %s\n", serverAddr)
	fmt.Printf("- 顯示詳細JSON: %v\n", showDetailedJson)
	fmt.Printf("- 自動啟動遊戲: %v\n", shouldStartGame)
	fmt.Printf("- 自動抽球: %v\n\n", enableAutoDraw)

	// 連接設定
	var kacp = keepalive.ClientParameters{
		Time:                15 * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}

	// 連接到 gRPC 服務器
	fmt.Printf("連接到 gRPC 服務器 (%s)...\n", serverAddr)
	conn, err := grpc.Dial(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kacp),
	)
	if err != nil {
		log.Fatalf("無法連接到 gRPC 服務器: %v", err)
	}
	defer conn.Close()

	// 創建 DealerService 客戶端
	client := pb.NewDealerServiceClient(conn)

	// 設置上下文並處理中斷信號
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設置中斷處理
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("\n接收到中斷信號，正在關閉連接...")
		stopAutoDrawing() // 確保停止自動抽球
		cancel()
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	// 訂閱遊戲事件
	fmt.Printf("訂閱房間 [%s] 的遊戲事件...\n", roomId)
	stream, err := client.SubscribeGameEvents(ctx, &pb.SubscribeGameEventsRequest{
		RoomId: roomId,
	})
	if err != nil {
		log.Fatalf("訂閱遊戲事件失敗: %v", err)
	}

	fmt.Println("訂閱成功! 等待遊戲事件...")
	fmt.Println("使用 Ctrl+C 停止監聽")
	fmt.Println("===================================")

	// 如果需要啟動遊戲，在背景執行
	if shouldStartGame {
		go func() {
			err := startNewGame(ctx, client, roomId)
			if err != nil {
				fmt.Printf("遊戲啟動失敗: %v\n", err)
			}
		}()
	}

	// 持續接收事件
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("服務器關閉了事件流")
			break
		}
		if err != nil {
			fmt.Printf("接收事件時發生錯誤: %v\n", err)
			break
		}

		// 處理收到的事件
		handleGameEvent(event, showDetailedJson)

		// 如果啟用了自動抽球，檢查是否需要開始抽球
		if enableAutoDraw {
			checkForDrawingStart(event, client, roomId)
		}
	}
}
