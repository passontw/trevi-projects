package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultRoomID        = "SG01"
	defaultServerAddress = "localhost:9100"
	defaultConfigFile    = "config.json"
)

// Config 配置結構
type Config struct {
	Game struct {
		RegularBalls struct {
			Count    int `json:"count"`
			MaxValue int `json:"max_value"`
		} `json:"regular_balls"`
		ExtraBalls struct {
			Count    int `json:"count"`
			MaxValue int `json:"max_value"`
		} `json:"extra_balls"`
		JackpotBalls struct {
			Count    int `json:"count"`
			MaxValue int `json:"max_value"`
		} `json:"jackpot_balls"`
		LuckyBalls struct {
			Count    int `json:"count"`
			MaxValue int `json:"max_value"`
		} `json:"lucky_balls"`
	} `json:"game"`
	Timing struct {
		RegularBallIntervalMs   int `json:"regular_ball_interval_ms"`
		ExtraBallIntervalMs     int `json:"extra_ball_interval_ms"`
		JackpotBallIntervalMs   int `json:"jackpot_ball_interval_ms"`
		LuckyBallIntervalMs     int `json:"lucky_ball_interval_ms"`
		CardPurchaseDurationSec int `json:"card_purchase_duration_sec"`
		GameOverWaitSec         int `json:"game_over_wait_sec"`
	} `json:"timing"`
}

// AutoDealer 是一個自動荷官，可以自動處理遊戲流程
type AutoDealer struct {
	client        dealerpb.DealerServiceClient
	roomID        string
	ctx           context.Context
	cancel        context.CancelFunc
	state         *GameState
	stateMutex    sync.Mutex
	extraBallSide commonpb.ExtraBallSide
	config        *Config
}

// GameState 保存遊戲的當前狀態
type GameState struct {
	gameID            string
	currentStage      commonpb.GameStage
	drawnRegularBalls []int32
	drawnExtraBalls   []int32
	drawnJPBalls      []int32
	drawnLuckyBalls   []int32
	preparingNewGame  bool
	currentGameData   *dealerpb.GameData
}

// 載入配置文件
func loadConfig(configFile string) (*Config, error) {
	// 預設配置
	config := &Config{}
	config.Game.RegularBalls.Count = 30
	config.Game.RegularBalls.MaxValue = 80
	config.Game.ExtraBalls.Count = 3
	config.Game.ExtraBalls.MaxValue = 80
	config.Game.JackpotBalls.Count = 1
	config.Game.JackpotBalls.MaxValue = 80
	config.Game.LuckyBalls.Count = 7
	config.Game.LuckyBalls.MaxValue = 80
	config.Timing.RegularBallIntervalMs = 500
	config.Timing.ExtraBallIntervalMs = 1000
	config.Timing.JackpotBallIntervalMs = 1000
	config.Timing.LuckyBallIntervalMs = 700
	config.Timing.CardPurchaseDurationSec = 5
	config.Timing.GameOverWaitSec = 5

	// 檢查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("配置文件 %s 不存在，使用預設配置", configFile)
		return config, nil
	}

	// 讀取配置文件
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("讀取配置文件失敗: %v", err)
	}

	// 解析配置
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失敗: %v", err)
	}

	log.Printf("成功載入配置文件 %s", configFile)
	return config, nil
}

// NewAutoDealer 創建一個新的自動荷官
func NewAutoDealer(serverAddr, roomID string, config *Config) (*AutoDealer, error) {
	// 設置連接選項
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// 建立連接
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("無法連接到服務器: %v", err)
	}

	// 創建客戶端
	client := dealerpb.NewDealerServiceClient(conn)

	// 創建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化遊戲狀態
	state := &GameState{
		currentStage:      commonpb.GameStage_GAME_STAGE_UNSPECIFIED,
		drawnRegularBalls: make([]int32, 0, config.Game.RegularBalls.Count),
		drawnExtraBalls:   make([]int32, 0, config.Game.ExtraBalls.Count),
		drawnJPBalls:      make([]int32, 0, config.Game.JackpotBalls.Count),
		drawnLuckyBalls:   make([]int32, 0, config.Game.LuckyBalls.Count),
	}

	// 隨機選擇額外球側邊
	extraBallSide := commonpb.ExtraBallSide_EXTRA_BALL_SIDE_LEFT
	if rand.Intn(2) == 1 {
		extraBallSide = commonpb.ExtraBallSide_EXTRA_BALL_SIDE_RIGHT
	}

	return &AutoDealer{
		client:        client,
		roomID:        roomID,
		ctx:           ctx,
		cancel:        cancel,
		state:         state,
		stateMutex:    sync.Mutex{},
		extraBallSide: extraBallSide,
		config:        config,
	}, nil
}

// Start 啟動自動荷官
func (d *AutoDealer) Start() error {
	log.Println("自動荷官準備啟動，房間ID:", d.roomID)
	log.Println("正在訂閱遊戲事件流...")

	// 先訂閱遊戲事件，優先建立事件監聽
	err := d.SubscribeGameEvents()
	if err != nil {
		return fmt.Errorf("訂閱遊戲事件失敗: %v", err)
	}

	log.Println("遊戲事件訂閱成功，自動荷官已啟動")
	log.Println("等待遊戲事件...")

	// 等待自動荷官結束
	<-d.ctx.Done()
	return nil
}

// Stop 停止自動荷官
func (d *AutoDealer) Stop() {
	d.cancel()
}

// SubscribeGameEvents 訂閱遊戲事件
func (d *AutoDealer) SubscribeGameEvents() error {
	log.Println("訂閱遊戲事件...")

	// 創建訂閱請求
	req := &dealerpb.SubscribeGameEventsRequest{
		RoomId: d.roomID,
	}

	// 嘗試建立訂閱
	maxAttempts := 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 判斷服務是否已經關閉
		select {
		case <-d.ctx.Done():
			log.Println("自動莊家服務已關閉，停止嘗試訂閱")
			return nil
		default:
			// 繼續執行
		}

		log.Printf("嘗試建立訂閱連接 (嘗試 %d/%d)...", attempt, maxAttempts)

		// 建立串流
		stream, err := d.client.SubscribeGameEvents(d.ctx, req)
		if err != nil {
			log.Printf("建立事件訂閱失敗: %v, 等待重試...", err)
			time.Sleep(time.Duration(d.getBackoffTime(attempt)) * time.Second)
			continue
		}

		log.Println("成功連接到遊戲事件串流，開始接收事件...")

		// 持續處理接收的事件
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Printf("接收事件時發生錯誤: %v, 重新連接...", err)
				break
			}

			// 處理接收到的事件
			d.processGameEvent(event)
		}

		// 如果執行到這裡，表示串流已中斷，等待後重試
		time.Sleep(2 * time.Second)
	}

	log.Printf("已達到最大重試次數 (%d)，無法建立訂閱連接", maxAttempts)
	return nil
}

func (d *AutoDealer) getBackoffTime(attempt int) int {
	// 使用指數退避算法: 2^attempt，最大30秒
	backoff := int(math.Pow(2, float64(attempt-1)))
	if backoff > 30 {
		return 30
	}
	return backoff
}

// processGameEvent 處理接收到的遊戲事件
func (d *AutoDealer) processGameEvent(event *dealerpb.GameEvent) {
	if event == nil {
		log.Println("收到空事件")
		return
	}

	// 記錄事件類型
	log.Printf("收到事件類型: %s", event.Type.String())

	// 檢查遊戲數據事件
	if gameData := event.GetGameData(); gameData != nil {
		log.Printf("遊戲數據事件 - ID: %s, 階段: %s",
			gameData.GameId,
			gameData.Stage.String())

		// 更新遊戲狀態
		d.updateGameState(gameData)

		// 處理階段邏輯
		d.processStage()
		return
	}

	// 檢查階段變更事件
	if stageChange := event.GetStageChanged(); stageChange != nil {
		log.Printf("階段變更事件 - 舊: %s, 新: %s",
			stageChange.OldStage.String(),
			stageChange.NewStage.String())

		// 更新階段
		d.stateMutex.Lock()
		d.state.currentStage = stageChange.NewStage
		d.stateMutex.Unlock()

		// 處理階段邏輯
		d.processStage()
		return
	}

	// 檢查球抽取事件
	if ballDrawn := event.GetBallDrawn(); ballDrawn != nil {
		log.Printf("球抽取事件 - 號碼: %d", ballDrawn.Ball.Number)
		d.handleBallDrawEvent(ballDrawn.Ball)
		return
	}

	// 檢查額外球抽取事件
	if extraBall := event.GetExtraBallDrawn(); extraBall != nil {
		log.Printf("額外球抽取事件 - 號碼: %d", extraBall.Ball.Number)
		return
	}

	// 檢查JP球抽取事件
	if jpBall := event.GetJackpotBallDrawn(); jpBall != nil {
		log.Printf("JP球抽取事件 - 號碼: %d", jpBall.Ball.Number)
		return
	}

	// 檢查幸運球抽取事件
	if luckyBall := event.GetLuckyBallDrawn(); luckyBall != nil && len(luckyBall.Balls) > 0 {
		log.Printf("幸運球抽取事件 - 已抽取: %d個", len(luckyBall.Balls))
		return
	}

	// 其他事件類型
	log.Printf("接收到其他類型事件")
}

// updateGameState 更新自動莊家的遊戲狀態
func (d *AutoDealer) updateGameState(gameData *dealerpb.GameData) {
	d.stateMutex.Lock()
	defer d.stateMutex.Unlock()

	// 更新遊戲階段和ID
	d.state.currentStage = gameData.Stage
	d.state.gameID = gameData.GameId

	// 存儲完整的遊戲數據
	d.state.currentGameData = gameData

	log.Printf("更新遊戲狀態：遊戲階段=%s, 遊戲ID=%s", gameData.Stage.String(), gameData.GameId)
}

// processStage 根據遊戲階段執行相應操作
func (d *AutoDealer) processStage() {
	log.Printf("處理遊戲階段: %s, 遊戲ID: %s", d.state.currentStage, d.state.gameID)

	switch d.state.currentStage {
	case commonpb.GameStage_GAME_STAGE_PREPARATION:
		log.Printf("遊戲處於準備階段，將自動開始新遊戲...")

		// 避免多次啟動新遊戲
		if !d.state.preparingNewGame {
			d.state.preparingNewGame = true

			// 啟動一個goroutine來處理新遊戲的開始
			go func() {
				// 等待15秒（增加等待時間），給系統緩衝時間
				log.Printf("等待15秒後開始新遊戲...")
				select {
				case <-d.ctx.Done():
					return
				case <-time.After(15 * time.Second):
					log.Printf("等待結束，現在開始新遊戲...")
					// 繼續處理
				}

				// 重置標誌
				d.state.preparingNewGame = false

				// 開始新遊戲
				d.startNewGame()
			}()
		} else {
			log.Printf("已有正在進行的新遊戲準備，跳過此次操作...")
		}

	case commonpb.GameStage_GAME_STAGE_NEW_ROUND:
		log.Println("遊戲開始新回合，2秒後進入購卡階段...")
		// 播放開始主遊戲動效（停留2秒）
		go func() {
			time.Sleep(2 * time.Second)
			log.Println("播放開始主遊戲動效結束，繼續等待進入下一階段...")
			// 系統會自動切換到卡片購買階段
		}()

	case commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		log.Println("開放購買卡片階段...")
		// 等待一段時間後自動抽球
		go func() {
			// 等待配置的卡片購買時間（21秒）
			log.Printf("等待 21 秒卡片購買時間...")
			time.Sleep(21 * time.Second)
			log.Println("卡片購買時間結束，準備開始抽球...")
			d.startDrawing()
		}()

	case commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_CLOSE:
		log.Println("卡片購買已關閉階段，等待進入抽球階段...")
		// 這個階段是中間過渡階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_DRAWING_START:
		log.Println("開始抽取常規球...")
		// 開始抽取常規球
		go d.drawRegularBalls()

	case commonpb.GameStage_GAME_STAGE_DRAWING_CLOSE:
		log.Println("常規球抽取結束階段，等待3秒查看結果...")
		// 看出球結果（停留3秒）
		go func() {
			time.Sleep(3 * time.Second)
			log.Println("查看常規球結果結束，等待進入下一階段...")
			// 系統會自動進入下一階段
		}()

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_PREPARE:
		log.Println("額外球準備階段，播放額外球動效3秒...")
		// 播放額外球動效（停留3秒）
		go func() {
			time.Sleep(3 * time.Second)
			log.Println("額外球動效播放結束，等待進入下一階段...")
			// 系統會自動切換到額外球選邊階段
		}()

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_START:
		log.Println("額外球選邊開始階段...")
		// 額外球選邊倒數計時與LED RNG表演（停留5秒）
		go func() {
			time.Sleep(5 * time.Second)
			log.Println("額外球選邊倒數時間結束，等待進入下一階段...")
			// 系統會自動切換到選邊結束階段
		}()

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED:
		log.Println("額外球選邊結束階段...")
		// 看額外球RNG結果（停留2秒）
		go func() {
			time.Sleep(2 * time.Second)
			log.Println("查看額外球RNG結果結束，等待進入下一階段...")
			// 系統會自動切換到額外球抽取階段
		}()

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START:
		log.Println("開始抽取額外球...")
		// 開始抽取額外球
		go d.drawExtraBalls()

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_CLOSE:
		log.Println("額外球抽取結束階段...")
		// 看出球結果（停留3秒）
		go func() {
			time.Sleep(3 * time.Second)
			log.Println("查看額外球結果結束，等待進入下一階段...")
			// 系統會自動進入下一階段
		}()

	case commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		log.Println("派彩結算階段...")
		// 看LED結算（停留 10秒）含客戶端表演
		go func() {
			time.Sleep(10 * time.Second)
			log.Println("派彩結算顯示結束，等待進入下一階段...")
			// 系統會自動進入下一階段
		}()

	case commonpb.GameStage_GAME_STAGE_JACKPOT_START:
		log.Println("JP準備階段...")
		// 播放開始JP遊戲動效（停留3秒）+ 看JP卡倒數計時（停留5秒）
		go func() {
			// 播放JP遊戲動效
			log.Println("播放開始JP遊戲動效...")
			time.Sleep(3 * time.Second)

			// JP卡倒數計時
			log.Println("JP卡倒數計時...")
			time.Sleep(5 * time.Second)

			log.Println("JP準備階段結束，等待進入JP抽球階段...")
			// 系統會自動進入JP抽球階段
		}()

	case commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START:
		log.Println("開始抽取JP球...")
		// 開始抽取JP球
		go d.drawJPBalls()

	case commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_CLOSED:
		log.Println("JP球抽取結束階段...")
		// 看出球結果（停留3秒）
		go func() {
			time.Sleep(3 * time.Second)
			log.Println("查看JP球結果結束，等待進入下一階段...")
			// 系統會自動進入下一階段
		}()

	case commonpb.GameStage_GAME_STAGE_JACKPOT_SETTLEMENT:
		log.Println("JP結算階段...")
		// 看LED結算（停留 10秒）含客戶端表演
		go func() {
			time.Sleep(10 * time.Second)
			log.Println("JP結算顯示結束，觸發幸運球準備階段...")
			// JP結算階段結束後，自動觸發幸運球抽取開始
			d.startDrawLuckyBalls()
		}()

	case commonpb.GameStage_GAME_STAGE_LUCKY_PREPARATION:
		log.Println("幸運球準備階段，等待3秒...")
		// 這個階段是中間過渡階段，等待3秒後主動觸發幸運球抽取階段
		go func() {
			time.Sleep(3 * time.Second)
			log.Println("幸運球準備階段結束，主動觸發幸運球抽取階段...")
			// 主動調用API來進入幸運球抽取階段
			d.startDrawLuckyBalls()
		}()

	case commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START:
		log.Println("開始抽取幸運號碼球...")
		// 開始抽取幸運號碼球
		go d.drawLuckyBalls()

	case commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_CLOSED:
		log.Println("幸運號碼球抽取結束階段...")
		// 看出球結果（停留3秒）
		go func() {
			time.Sleep(3 * time.Second)
			log.Println("查看幸運球結果結束，等待進入遊戲結束階段...")
			// 系統會自動進入遊戲結束階段
		}()

	case commonpb.GameStage_GAME_STAGE_GAME_OVER:
		log.Println("遊戲結束階段...")
		log.Println("遊戲完成，等待10秒後重新開始...")

		// 等待10秒再開始新一輪遊戲
		go func() {
			select {
			case <-d.ctx.Done():
				return
			case <-time.After(10 * time.Second):
				log.Println("等待結束，開始檢查是否可以開始新遊戲...")
				// 不需要主動開始新遊戲，因為服務器會自動創建新遊戲
				// 我們會透過事件訂閱收到新遊戲的通知
			}
		}()
	}
}

// handleBallDrawEvent 處理抽球事件
func (d *AutoDealer) handleBallDrawEvent(ball *dealerpb.Ball) {
	if ball == nil {
		return
	}

	log.Printf("抽出球: 號碼=%d, 類型=%s, 是否為最後一顆=%v",
		ball.Number, ball.Type, ball.IsLast)

	// 根據球的類型更新狀態
	switch ball.Type {
	case dealerpb.BallType_BALL_TYPE_REGULAR:
		d.state.drawnRegularBalls = append(d.state.drawnRegularBalls, ball.Number)
	case dealerpb.BallType_BALL_TYPE_EXTRA:
		d.state.drawnExtraBalls = append(d.state.drawnExtraBalls, ball.Number)
	case dealerpb.BallType_BALL_TYPE_JACKPOT:
		d.state.drawnJPBalls = append(d.state.drawnJPBalls, ball.Number)
	case dealerpb.BallType_BALL_TYPE_LUCKY:
		d.state.drawnLuckyBalls = append(d.state.drawnLuckyBalls, ball.Number)
	}
}

// startNewGame 開始一個新的遊戲
func (d *AutoDealer) startNewGame() {
	log.Println("開始新遊戲...")

	// 使用StartNewRound接口開始新遊戲
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &dealerpb.StartNewRoundRequest{
		RoomId: d.roomID,
	}

	resp, err := d.client.StartNewRound(ctx, req)
	if err != nil {
		log.Printf("開始新遊戲失敗: %v", err)
		return
	}

	log.Printf("遊戲創建成功，遊戲ID: %s, 階段: %s", resp.GameData.GameId, resp.GameData.Stage.String())

	// 更新自動莊家的狀態
	d.updateGameState(resp.GameData)
}

// startDrawing 開始抽球
func (d *AutoDealer) startDrawing() {
	log.Println("啟動抽球階段...")

	// 直接由系統自動推進到抽球階段，無需請求
}

// drawRegularBalls 抽取常規球
func (d *AutoDealer) drawRegularBalls() {
	log.Println("開始抽取常規球...")

	// 清空已抽球列表，確保每次開始新的抽球流程時使用空列表
	d.stateMutex.Lock()
	d.state.drawnRegularBalls = []int32{}
	d.stateMutex.Unlock()

	// 累積的球列表
	drawnBalls := []*dealerpb.Ball{}
	// 已成功抽取的球數
	successCount := 0

	// 抽取常規球
	for successCount < d.config.Game.RegularBalls.Count {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 檢查遊戲階段是否仍然是抽球階段
		d.stateMutex.Lock()
		currentStage := d.state.currentStage
		d.stateMutex.Unlock()

		if currentStage != commonpb.GameStage_GAME_STAGE_DRAWING_START {
			log.Printf("遊戲階段已變更為 %s，停止抽取常規球", currentStage.String())
			return
		}

		// 生成一個隨機球號 (1-75)
		ballNumber := d.generateUniqueBallNumber(d.state.drawnRegularBalls, d.config.Game.RegularBalls.MaxValue)

		// 檢查是否生成成功
		if ballNumber == -1 {
			log.Printf("無法生成不重複的常規球號，可能已達到最大數量或系統出錯，中止抽球")

			// 如果已經抽出至少一個球，將最後一個設置為最後一顆球
			if len(drawnBalls) > 0 {
				log.Printf("將已抽出的 %d 個常規球中的最後一個設置為最後一顆", len(drawnBalls))
				drawnBalls[len(drawnBalls)-1].IsLast = true

				// 發送更新請求
				req := &dealerpb.DrawBallRequest{
					RoomId: d.roomID,
					Balls:  drawnBalls,
				}

				if _, err := d.client.DrawBall(d.ctx, req); err != nil {
					log.Printf("更新最後一個常規球狀態失敗: %v", err)
				} else {
					log.Printf("已成功將常規球 %d 標記為最後一顆", drawnBalls[len(drawnBalls)-1].Number)
				}
			}

			return
		}

		// 判斷是否為最後一顆球
		isLast := successCount == d.config.Game.RegularBalls.Count-1

		// 創建新球
		ball := &dealerpb.Ball{
			Number:    ballNumber,
			Type:      dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast:    isLast,
			Timestamp: timestamppb.Now(),
		}

		// 嘗試添加到累積球列表並發送
		tempBalls := append([]*dealerpb.Ball{}, drawnBalls...)
		tempBalls = append(tempBalls, ball)

		// 創建抽球請求（包含所有已抽出的球）
		req := &dealerpb.DrawBallRequest{
			RoomId: d.roomID,
			Balls:  tempBalls,
		}

		// 發送請求
		resp, err := d.client.DrawBall(d.ctx, req)
		if err != nil {
			// 檢查是否是重複球號錯誤
			if containsString(err.Error(), "請求中包含重複的球號") {
				log.Printf("抽取常規球出現重複號碼，略過此號碼: %v", err)
				// 不增加successCount，重新嘗試
				continue
			} else if containsString(err.Error(), "當前階段") && containsString(err.Error(), "不允許替換球") {
				// 階段已經變更，停止抽球
				log.Printf("遊戲階段已變更，無法繼續抽球: %v", err)
				return
			} else {
				// 其他錯誤則終止
				log.Printf("抽取常規球失敗: %v", err)
				return
			}
		}

		// 請求成功，更新正式球列表
		drawnBalls = tempBalls
		successCount++

		// 更新遊戲狀態
		if resp.GameData != nil {
			d.stateMutex.Lock()
			d.state.currentStage = resp.GameData.Stage
			d.stateMutex.Unlock()

			// 檢查是否階段已變更
			if resp.GameData.Stage != commonpb.GameStage_GAME_STAGE_DRAWING_START {
				log.Printf("抽球後遊戲階段已變更為 %s，停止抽取常規球", resp.GameData.Stage.String())
				return
			}
		}

		// 添加到已抽球列表
		d.stateMutex.Lock()
		d.state.drawnRegularBalls = append(d.state.drawnRegularBalls, ballNumber)
		d.stateMutex.Unlock()

		log.Printf("抽取常規球成功，號碼: %d, 是否為最後一顆: %v, 累計已抽出 %d 顆常規球", ballNumber, isLast, len(drawnBalls))

		// 如果這是最後一顆球，則不再抽取更多球
		if isLast {
			log.Println("已抽取完最後一顆常規球，完成抽球流程")
			return
		}

		// 暫停一下，讓球抽取看起來更自然
		time.Sleep(time.Duration(d.config.Timing.RegularBallIntervalMs) * time.Millisecond)
	}
}

// drawExtraBalls 抽取額外球
func (d *AutoDealer) drawExtraBalls() {
	log.Println("開始抽取額外球...")

	// 清空已抽球列表
	d.stateMutex.Lock()
	d.state.drawnExtraBalls = []int32{}
	d.stateMutex.Unlock()

	// 累積的球列表
	drawnBalls := []*dealerpb.Ball{}
	// 已成功抽取的球數
	successCount := 0

	// 抽取額外球
	for successCount < d.config.Game.ExtraBalls.Count {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 檢查遊戲階段是否仍然是抽球階段
		d.stateMutex.Lock()
		currentStage := d.state.currentStage
		d.stateMutex.Unlock()

		if currentStage != commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START {
			log.Printf("遊戲階段已變更為 %s，停止抽取額外球", currentStage.String())
			return
		}

		// 生成一個隨機球號 (1-75)，確保和一般球不重複
		combinedBalls := append([]int32{}, d.state.drawnRegularBalls...)
		combinedBalls = append(combinedBalls, d.state.drawnExtraBalls...)
		ballNumber := d.generateUniqueBallNumber(combinedBalls, d.config.Game.ExtraBalls.MaxValue)

		// 檢查是否生成成功
		if ballNumber == -1 {
			log.Printf("無法生成不重複的額外球號，可能已達到最大數量或系統出錯，中止抽球")

			// 如果已經抽出至少一個球，將最後一個設置為最後一顆球
			if len(drawnBalls) > 0 {
				log.Printf("將已抽出的 %d 個額外球中的最後一個設置為最後一顆", len(drawnBalls))
				drawnBalls[len(drawnBalls)-1].IsLast = true

				// 發送更新請求
				req := &dealerpb.DrawExtraBallRequest{
					RoomId: d.roomID,
					Side:   d.extraBallSide,
					Balls:  drawnBalls,
				}

				if _, err := d.client.DrawExtraBall(d.ctx, req); err != nil {
					log.Printf("更新最後一個額外球狀態失敗: %v", err)
				} else {
					log.Printf("已成功將額外球 %d 標記為最後一顆", drawnBalls[len(drawnBalls)-1].Number)
				}
			}

			return
		}

		// 判斷是否為最後一顆球
		isLast := successCount == d.config.Game.ExtraBalls.Count-1

		// 創建新球
		ball := &dealerpb.Ball{
			Number:    ballNumber,
			Type:      dealerpb.BallType_BALL_TYPE_EXTRA,
			IsLast:    isLast,
			Timestamp: timestamppb.Now(),
		}

		// 嘗試添加到累積球列表並發送
		tempBalls := append([]*dealerpb.Ball{}, drawnBalls...)
		tempBalls = append(tempBalls, ball)

		// 創建抽球請求（包含所有已抽出的球）
		req := &dealerpb.DrawExtraBallRequest{
			RoomId: d.roomID,
			Side:   d.extraBallSide,
			Balls:  tempBalls,
		}

		// 發送請求
		resp, err := d.client.DrawExtraBall(d.ctx, req)
		if err != nil {
			// 檢查是否是重複球號錯誤
			if containsString(err.Error(), "請求中包含重複的球號") {
				log.Printf("抽取額外球出現重複號碼，略過此號碼: %v", err)
				// 不增加successCount，重新嘗試
				continue
			} else if containsString(err.Error(), "當前階段") && containsString(err.Error(), "不允許替換球") {
				// 階段已經變更，停止抽球
				log.Printf("遊戲階段已變更，無法繼續抽球: %v", err)
				return
			} else {
				// 其他錯誤則終止
				log.Printf("抽取額外球失敗: %v", err)
				return
			}
		}

		// 請求成功，更新正式球列表
		drawnBalls = tempBalls
		successCount++

		// 更新遊戲狀態
		if resp.GameData != nil {
			d.stateMutex.Lock()
			d.state.currentStage = resp.GameData.Stage
			d.stateMutex.Unlock()

			// 檢查是否階段已變更
			if resp.GameData.Stage != commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START {
				log.Printf("抽球後遊戲階段已變更為 %s，停止抽取額外球", resp.GameData.Stage.String())
				return
			}
		}

		// 添加到已抽球列表
		d.stateMutex.Lock()
		d.state.drawnExtraBalls = append(d.state.drawnExtraBalls, ballNumber)
		d.stateMutex.Unlock()

		log.Printf("抽取額外球成功，號碼: %d, 是否為最後一顆: %v, 累計已抽出 %d 顆額外球", ballNumber, isLast, len(drawnBalls))

		// 如果這是最後一顆球，則不再抽取更多球
		if isLast {
			log.Println("已抽取完最後一顆額外球，完成抽球流程")
			return
		}

		// 暫停一下，讓球抽取看起來更自然
		time.Sleep(time.Duration(d.config.Timing.ExtraBallIntervalMs) * time.Millisecond)
	}
}

// drawLuckyBalls 抽取幸運球
func (d *AutoDealer) drawLuckyBalls() {
	log.Println("開始抽取幸運球...")

	// 檢查遊戲階段是否適合抽取幸運球
	d.stateMutex.Lock()
	currentStage := d.state.currentStage
	d.stateMutex.Unlock()

	if currentStage != commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START {
		log.Printf("當前階段不是抽取幸運球階段: %s", currentStage.String())
		return
	}

	// 獲取當前已抽球數量
	d.stateMutex.Lock()
	currentDrawnCount := len(d.state.drawnLuckyBalls)
	hasLastBall := false

	// 檢查是否已有最後一顆球
	if d.state.currentGameData != nil && d.state.currentGameData.Jackpot != nil {
		for _, ball := range d.state.currentGameData.Jackpot.LuckyBalls {
			if ball.IsLast {
				hasLastBall = true
				break
			}
		}
	}
	d.stateMutex.Unlock()

	// 如果已經完成所有抽球或已有最後一顆球，則直接返回
	if currentDrawnCount >= d.config.Game.LuckyBalls.Count || hasLastBall {
		log.Printf("幸運球已抽取完成：當前數量=%d，已有最後一球=%v", currentDrawnCount, hasLastBall)
		return
	}

	log.Printf("當前已抽取 %d 個幸運球，目標數量: %d", currentDrawnCount, d.config.Game.LuckyBalls.Count)

	// 創建一個映射來追蹤所有已嘗試和已存在的球號
	attemptedNumbers := make(map[int32]bool)
	existingNumbers := make(map[int32]bool)

	// 記錄已存在的幸運球號碼
	d.stateMutex.Lock()
	for _, ballNum := range d.state.drawnLuckyBalls {
		existingNumbers[ballNum] = true
	}
	d.stateMutex.Unlock()

	// 計算還需要抽取的球數量
	remainingBalls := d.config.Game.LuckyBalls.Count - currentDrawnCount
	log.Printf("準備抽取剩餘 %d 個幸運球", remainingBalls)

	// 抽取剩餘的幸運球
	for i := 0; i < remainingBalls; i++ {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 再次檢查遊戲階段是否仍然是抽球階段
		d.stateMutex.Lock()
		currentStage := d.state.currentStage
		d.stateMutex.Unlock()

		if currentStage != commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START {
			log.Printf("遊戲階段已變更為 %s，停止抽取幸運球", currentStage.String())
			return
		}

		// 確定是否為最後一個球
		isLast := (i == remainingBalls-1)

		// 生成一個隨機球號，確保不與已有的球號重複
		var ballNumber int32
		maxAttempts := d.config.Game.LuckyBalls.MaxValue // 最多嘗試所有可能的球號

		// 嘗試生成不重複的球號
		for attempts := 0; attempts < maxAttempts; attempts++ {
			// 生成1至MaxValue的隨機數
			newBallNumber := int32(rand.Intn(d.config.Game.LuckyBalls.MaxValue) + 1)

			// 確保不與已有球號重複且未嘗試過
			if !existingNumbers[newBallNumber] && !attemptedNumbers[newBallNumber] {
				ballNumber = newBallNumber
				break
			}

			// 標記該球號已被嘗試
			attemptedNumbers[newBallNumber] = true

			// 如果已嘗試了所有可能的球號
			if len(attemptedNumbers) >= d.config.Game.LuckyBalls.MaxValue {
				log.Printf("警告：已嘗試所有可能的球號，無法找到不重複的幸運球號")
				return
			}
		}

		// 如果無法生成有效的球號
		if ballNumber == 0 {
			log.Printf("無法生成不重複的幸運球號，請檢查邏輯")
			return
		}

		// 創建請求
		req := &dealerpb.DrawLuckyBallRequest{
			RoomId: d.roomID,
			Balls: []*dealerpb.Ball{
				{
					Number:    ballNumber,
					Type:      dealerpb.BallType_BALL_TYPE_LUCKY,
					IsLast:    isLast,
					Timestamp: timestamppb.Now(),
				},
			},
		}

		log.Printf("抽取幸運球: %d, isLast=%v", ballNumber, isLast)

		// 發送請求
		resp, err := d.client.DrawLuckyBall(d.ctx, req)
		if err != nil {
			// 檢查是否是球號重複的錯誤
			if strings.Contains(err.Error(), "重複") {
				log.Printf("球號 %d 重複，嘗試另一個球號: %v", ballNumber, err)
				// 標記這個球號為已存在，避免再次嘗試
				existingNumbers[ballNumber] = true
				// 減少計數器，以便重新嘗試這個位置
				i--
				continue
			} else if strings.Contains(err.Error(), "階段") || strings.Contains(err.Error(), "狀態") {
				// 階段已經變更，停止抽球
				log.Printf("遊戲階段已變更，無法繼續抽球: %v", err)
				return
			} else {
				// 其他錯誤則終止
				log.Printf("抽取幸運球失敗: %v", err)
				return
			}
		}

		// 更新遊戲狀態
		if resp != nil && resp.GameData != nil {
			d.updateGameState(resp.GameData)
			log.Printf("成功抽取幸運球 %d，當前已有 %d 個幸運球",
				ballNumber, len(resp.GameData.Jackpot.LuckyBalls))

			// 標記這個球號為已存在
			existingNumbers[ballNumber] = true

			// 將球號添加到已抽取列表
			d.stateMutex.Lock()
			d.state.drawnLuckyBalls = append(d.state.drawnLuckyBalls, ballNumber)
			d.stateMutex.Unlock()

			// 如果階段已經改變，結束抽取
			if resp.GameData.Stage != commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START {
				log.Printf("遊戲階段已改變為 %s，停止抽取幸運球", resp.GameData.Stage.String())
				return
			}

			// 如果是最後一個球，結束抽取
			if isLast {
				log.Printf("最後一個幸運球已抽取完成")
				return
			}
		}

		// 抽球間隔
		time.Sleep(time.Duration(d.config.Timing.LuckyBallIntervalMs) * time.Millisecond)
	}
}

// drawJPBalls 抽取JP球
func (d *AutoDealer) drawJPBalls() {
	log.Println("開始抽取JP球...")

	// 清空已抽球列表
	d.stateMutex.Lock()
	d.state.drawnJPBalls = []int32{}
	d.stateMutex.Unlock()

	// 累積的球列表
	drawnBalls := []*dealerpb.Ball{}
	// 已成功抽取的球數
	successCount := 0

	// 抽取JP球
	for successCount < d.config.Game.JackpotBalls.Count {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 檢查遊戲階段是否仍然是抽球階段
		d.stateMutex.Lock()
		currentStage := d.state.currentStage
		d.stateMutex.Unlock()

		if currentStage != commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START {
			log.Printf("遊戲階段已變更為 %s，停止抽取JP球", currentStage.String())
			return
		}

		// 生成一個隨機球號 (1-75)，與其他類型球無關
		ballNumber := d.generateUniqueBallNumber(d.state.drawnJPBalls, d.config.Game.JackpotBalls.MaxValue)

		// 檢查是否生成成功
		if ballNumber == -1 {
			log.Printf("無法生成不重複的JP球號，可能已達到最大數量或系統出錯，中止抽球")

			// 如果已經抽出至少一個球，將最後一個設置為最後一顆球
			if len(drawnBalls) > 0 {
				log.Printf("將已抽出的 %d 個JP球中的最後一個設置為最後一顆", len(drawnBalls))
				drawnBalls[len(drawnBalls)-1].IsLast = true

				// 發送更新請求
				req := &dealerpb.DrawJackpotBallRequest{
					RoomId: d.roomID,
					Balls:  drawnBalls,
				}

				if _, err := d.client.DrawJackpotBall(d.ctx, req); err != nil {
					log.Printf("更新最後一個JP球狀態失敗: %v", err)
				} else {
					log.Printf("已成功將JP球 %d 標記為最後一顆", drawnBalls[len(drawnBalls)-1].Number)
				}
			}

			return
		}

		// 判斷是否為最後一顆球
		isLast := successCount == d.config.Game.JackpotBalls.Count-1

		// 創建新球
		ball := &dealerpb.Ball{
			Number:    ballNumber,
			Type:      dealerpb.BallType_BALL_TYPE_JACKPOT,
			IsLast:    isLast,
			Timestamp: timestamppb.Now(),
		}

		// 嘗試添加到累積球列表並發送
		tempBalls := append([]*dealerpb.Ball{}, drawnBalls...)
		tempBalls = append(tempBalls, ball)

		// 創建抽球請求（包含所有已抽出的球）
		req := &dealerpb.DrawJackpotBallRequest{
			RoomId: d.roomID,
			Balls:  tempBalls,
		}

		// 發送請求
		resp, err := d.client.DrawJackpotBall(d.ctx, req)
		if err != nil {
			// 檢查是否是重複球號錯誤
			if containsString(err.Error(), "請求中包含重複的球號") {
				log.Printf("抽取JP球出現重複號碼，略過此號碼: %v", err)
				// 不增加successCount，重新嘗試
				continue
			} else if containsString(err.Error(), "當前階段") && containsString(err.Error(), "不允許替換球") {
				// 階段已經變更，停止抽球
				log.Printf("遊戲階段已變更，無法繼續抽球: %v", err)
				return
			} else {
				// 其他錯誤則終止
				log.Printf("抽取JP球失敗: %v", err)
				return
			}
		}

		// 請求成功，更新正式球列表
		drawnBalls = tempBalls
		successCount++

		// 更新遊戲狀態
		if resp.GameData != nil {
			d.stateMutex.Lock()
			d.state.currentStage = resp.GameData.Stage
			d.stateMutex.Unlock()

			// 檢查是否階段已變更
			if resp.GameData.Stage != commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START {
				log.Printf("抽球後遊戲階段已變更為 %s，停止抽取JP球", resp.GameData.Stage.String())
				return
			}
		}

		// 添加到已抽球列表
		d.stateMutex.Lock()
		d.state.drawnJPBalls = append(d.state.drawnJPBalls, ballNumber)
		d.stateMutex.Unlock()

		log.Printf("抽取JP球成功，號碼: %d, 是否為最後一顆: %v, 累計已抽出 %d 顆JP球", ballNumber, isLast, len(drawnBalls))

		// 如果這是最後一顆球，則不再抽取更多球
		if isLast {
			log.Println("已抽取完最後一顆JP球，完成抽球流程")
			return
		}

		// 暫停一下，讓球抽取看起來更自然
		time.Sleep(time.Duration(d.config.Timing.JackpotBallIntervalMs) * time.Millisecond)
	}
}

// generateUniqueBallNumber 生成一個不重複的球號
func (d *AutoDealer) generateUniqueBallNumber(drawnBalls []int32, maxValue int) int32 {
	// 檢查是否所有數字都已經被抽出（無法生成新的不重複數字）
	if len(drawnBalls) >= maxValue {
		log.Printf("警告：所有可用的球號 (1-%d) 已經被抽完，無法生成新的不重複球號", maxValue)
		return -1 // 返回一個特殊值表示無法生成
	}

	// 創建一個已抽出球號的映射，用於快速檢查
	drawnMap := make(map[int32]bool)
	for _, ball := range drawnBalls {
		drawnMap[ball] = true
	}

	// 最多嘗試次數，避免無限循環
	maxAttempts := maxValue * 2
	attempts := 0

	for attempts < maxAttempts {
		// 生成一個1到maxValue的隨機數
		n := int32(rand.Intn(maxValue) + 1)

		// 使用映射檢查是否已抽過，比循環檢查更高效
		if !drawnMap[n] {
			return n
		}

		attempts++
	}

	// 如果多次嘗試後仍未找到不重複的數字，改用順序查找方法
	log.Printf("警告：隨機生成不重複球號失敗，嘗試順序查找可用球號...")
	for i := int32(1); i <= int32(maxValue); i++ {
		if !drawnMap[i] {
			return i
		}
	}

	// 理論上不會執行到這裡，因為之前已經檢查過是否所有數字都被抽出
	log.Printf("嚴重錯誤：無法生成不重複的球號（1-%d），可能存在邏輯錯誤", maxValue)
	return -1
}

// containsString 檢查字符串是否包含特定子字符串
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// 更新 getConfig 函數來正確返回三個值
func getConfig() (string, string, string) {
	// 從環境變數獲取伺服器地址，默認為localhost:9100
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:9100"
	}

	// 從環境變數獲取房間ID，默認為SG01
	roomID := os.Getenv("ROOM_ID")
	if roomID == "" {
		roomID = "SG01"
	}

	// 從環境變數獲取配置文件路徑，默認為config.json
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.json"
	}

	return serverAddr, roomID, configFile
}

// startDrawLuckyBalls 開始幸運球階段
func (d *AutoDealer) startDrawLuckyBalls() {
	log.Println("開始幸運球階段...")

	// 使用StartDrawLuckyBall接口開始幸運球階段
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &dealerpb.StartDrawLuckyBallRequest{
		RoomId: d.roomID,
	}

	resp, err := d.client.StartDrawLuckyBall(ctx, req)
	if err != nil {
		log.Printf("開始幸運球階段失敗: %v", err)
		return
	}

	log.Printf("幸運球階段開始成功，當前階段: %s", resp.GameData.Stage.String())

	// 更新自動莊家的狀態
	d.updateGameState(resp.GameData)
}

func main() {
	log.Println("樂透自動荷官程式啟動")

	// 獲取配置
	serverAddr, roomID, configFile := getConfig()
	log.Printf("使用配置 - 服務器地址: %s, 房間ID: %s, 配置文件: %s", serverAddr, roomID, configFile)

	// 載入配置文件
	config, err := loadConfig(configFile)
	if err != nil {
		log.Printf("警告: 載入配置文件失敗: %v, 將使用默認配置", err)
		config = &Config{} // 使用默認配置
	}

	// 創建自動荷官
	dealer, err := NewAutoDealer(serverAddr, roomID, config)
	if err != nil {
		log.Fatalf("創建自動荷官失敗: %v", err)
	}

	// 設置信號處理，優雅關閉
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 啟動自動荷官
	go func() {
		if err := dealer.Start(); err != nil {
			log.Printf("自動荷官運行失敗: %v", err)
		}
	}()

	// 等待中斷信號
	<-sigCh
	log.Println("收到中斷信號，正在關閉自動荷官...")

	// 停止自動荷官
	dealer.Stop()
	log.Println("自動荷官已關閉，程序退出")
}
