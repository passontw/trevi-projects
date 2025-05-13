package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
	commonpb "g38_lottery_service/internal/generated/common"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultRoomID        = "SG01"
	defaultServerAddress = "localhost:8080"
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
	// 實現帶重試機制的訂閱
	go d.subscribeWithRetry()
	return nil
}

// subscribeWithRetry 實現帶重試機制的事件訂閱
func (d *AutoDealer) subscribeWithRetry() {
	var (
		backoff     = 1 * time.Second
		maxBackoff  = 30 * time.Second
		maxAttempts = 10
		attempts    = 0
	)

	for {
		select {
		case <-d.ctx.Done():
			log.Println("訂閱服務已停止")
			return
		default:
			// 繼續嘗試連接
		}

		attempts++
		if attempts > maxAttempts {
			log.Printf("嘗試重連 %d 次後放棄", maxAttempts)
			return
		}

		log.Printf("建立訂閱連接 (嘗試 %d/%d)...", attempts, maxAttempts)
		err := d.establishSubscription()
		if err != nil {
			log.Printf("訂閱失敗: %v, %d 秒後重試...", err, backoff/time.Second)
			time.Sleep(backoff)
			// 增加退避時間，但不超過最大值
			backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
			continue
		}

		// 成功建立訂閱後，重置重試次數和退避時間
		attempts = 0
		backoff = 1 * time.Second

		// 給系統一些時間冷卻，避免立即重連
		time.Sleep(2 * time.Second)
	}
}

// establishSubscription 建立單次訂閱
func (d *AutoDealer) establishSubscription() error {
	// 創建訂閱請求
	req := &dealerpb.SubscribeGameEventsRequest{
		RoomId: d.roomID,
	}

	log.Printf("正在連接服務器並訂閱房間 %s 的遊戲事件...", d.roomID)

	// 設置上下文，加上10分鐘超時
	ctx, cancel := context.WithTimeout(d.ctx, 10*time.Minute)
	defer cancel()

	// 訂閱遊戲事件
	stream, err := d.client.SubscribeGameEvents(ctx, req)
	if err != nil {
		return fmt.Errorf("訂閱遊戲事件失敗: %v", err)
	}

	log.Printf("遊戲事件流已成功建立")

	// 記錄最後接收心跳的時間
	lastHeartbeat := time.Now()

	// 創建用於檢查心跳超時的計時器
	heartbeatTimeout := time.NewTimer(30 * time.Second)

	// 創建初始事件定時器
	initialEventTimer := time.NewTimer(5 * time.Second)
	initialEventReceived := false

	// 監聽事件
	for {
		// 設置接收超時
		recvCtx, recvCancel := context.WithTimeout(ctx, 20*time.Second)

		// 在單獨的goroutine中接收事件
		eventCh := make(chan *dealerpb.GameEvent, 1)
		errCh := make(chan error, 1)

		go func() {
			// 使用 recvCtx 來控制接收超時
			select {
			case <-recvCtx.Done():
				// 超時或被取消
				errCh <- recvCtx.Err()
			default:
				event, err := stream.Recv()
				if err != nil {
					errCh <- err
					return
				}
				eventCh <- event
			}
		}()

		// 等待事件、錯誤、心跳超時或上下文取消
		select {
		case <-d.ctx.Done():
			log.Println("客戶端請求終止訂閱")
			recvCancel()
			return nil

		case <-ctx.Done():
			log.Println("訂閱上下文已取消")
			recvCancel()
			return ctx.Err()

		case <-heartbeatTimeout.C:
			elapsed := time.Since(lastHeartbeat)
			log.Printf("心跳超時 (%s)，重新建立連接", elapsed)
			recvCancel()
			return fmt.Errorf("心跳超時")

		case <-initialEventTimer.C:
			if !initialEventReceived {
				log.Println("未收到初始事件，主動開始新遊戲...")
				go d.startNewGame()
			}

		case event := <-eventCh:
			// 重置心跳超時計時器
			if !heartbeatTimeout.Stop() {
				<-heartbeatTimeout.C
			}
			heartbeatTimeout.Reset(30 * time.Second)

			initialEventReceived = true

			// 使用正確的心跳事件類型常量
			if event.GetType() == commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT {
				lastHeartbeat = time.Now()
				log.Printf("收到心跳事件，時間戳: %d", event.Timestamp)
				continue
			}

			// 處理其他事件
			d.handleGameEvent(event)

		case err := <-errCh:
			recvCancel()
			if err == io.EOF {
				log.Println("服務器關閉了連接 (EOF)")
			} else if strings.Contains(err.Error(), "max_age") {
				log.Println("服務器因最大存活時間限制關閉了連接")
			} else {
				log.Printf("事件接收錯誤: %v", err)
			}
			return err
		}
	}
}

// handleGameEvent 處理遊戲事件
func (d *AutoDealer) handleGameEvent(event *dealerpb.GameEvent) {
	if event == nil {
		log.Println("收到空事件")
		return
	}

	// 記錄事件類型，跳過心跳事件的詳細日誌
	if event.GetType() != commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT {
		log.Printf("收到遊戲事件 - 類型: %s, 時間戳: %d", event.Type, event.Timestamp)
	}

	// 檢查事件類型
	if gameData := event.GetGameData(); gameData != nil {
		d.state.gameID = gameData.GameId
		d.state.currentStage = gameData.Stage

		log.Printf("【遊戲數據事件】遊戲ID: %s, 階段: %s, 狀態: %s",
			gameData.GameId, gameData.Stage, gameData.Status)

		// 輸出已抽出的球
		if len(gameData.RegularBalls) > 0 {
			log.Printf("  已抽出的常規球: %d 個", len(gameData.RegularBalls))
			for i, ball := range gameData.RegularBalls {
				if i < 5 || i >= len(gameData.RegularBalls)-5 {
					log.Printf("    球 #%d: 號碼=%d, 類型=%s", i+1, ball.Number, ball.Type)
				} else if i == 5 {
					log.Printf("    ... (省略中間球) ...")
				}
			}
		}

		// 根據遊戲階段進行相應的操作
		d.processStage()

	} else if stageChanged := event.GetStageChanged(); stageChanged != nil {
		oldStage := stageChanged.OldStage
		newStage := stageChanged.NewStage
		d.state.currentStage = newStage

		log.Printf("【階段變更事件】舊階段: %s, 新階段: %s", oldStage, newStage)

		// 根據遊戲階段進行相應的操作
		d.processStage()

	} else if ballDrawn := event.GetBallDrawn(); ballDrawn != nil {
		// 處理抽球事件
		log.Printf("【抽球事件】球號: %d, 類型: %s, 位置: %d",
			ballDrawn.Ball.Number, ballDrawn.Ball.Type, ballDrawn.Position)
		d.handleBallDrawEvent(ballDrawn.Ball)

	} else if gameCreated := event.GetGameCreated(); gameCreated != nil {
		log.Printf("【遊戲創建事件】遊戲ID: %s", gameCreated.GameData.GameId)
		d.state.gameID = gameCreated.GameData.GameId
		d.state.currentStage = gameCreated.GameData.Stage

		// 根據遊戲階段進行相應的操作
		d.processStage()

	} else if gameCancelled := event.GetGameCancelled(); gameCancelled != nil {
		log.Printf("【遊戲取消事件】原因: %s", gameCancelled.Reason)

	} else if jpBallDrawn := event.GetJackpotBallDrawn(); jpBallDrawn != nil {
		log.Printf("【JP球抽取事件】球號: %d", jpBallDrawn.Ball.Number)

	} else if extraBallDrawn := event.GetExtraBallDrawn(); extraBallDrawn != nil {
		log.Printf("【額外球抽取事件】球號: %d, 側: %s",
			extraBallDrawn.Ball.Number, extraBallDrawn.Side)

	} else if luckyBallDrawn := event.GetLuckyBallDrawn(); luckyBallDrawn != nil {
		log.Printf("【幸運球抽取事件】")
		for i, ball := range luckyBallDrawn.Balls {
			log.Printf("  幸運球 #%d: 號碼=%d", i+1, ball.Number)
		}
	} else if event.GetType() == commonpb.GameEventType_GAME_EVENT_TYPE_HEARTBEAT {
		// 心跳事件已在上層處理，這裡不需要特別動作
	} else {
		log.Printf("收到未知類型的事件")
	}
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
				// 等待5秒，給系統緩衝時間
				select {
				case <-d.ctx.Done():
					return
				case <-time.After(5 * time.Second):
					// 繼續處理
				}

				// 重置標誌
				d.state.preparingNewGame = false

				// 開始新遊戲
				d.startNewGame()
			}()
		}

	case commonpb.GameStage_GAME_STAGE_NEW_ROUND:
		log.Println("遊戲開始新回合...")
		// 在這個階段不需要特別操作，等待進入購買卡片階段

	case commonpb.GameStage_GAME_STAGE_CARD_PURCHASE_OPEN:
		log.Println("開放購買卡片階段...")
		// 等待一段時間後自動抽球
		go func() {
			// 等待配置的卡片購買時間
			log.Printf("等待 %d 秒卡片購買時間...", d.config.Timing.CardPurchaseDurationSec)
			time.Sleep(time.Duration(d.config.Timing.CardPurchaseDurationSec) * time.Second)
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
		log.Println("常規球抽取已結束，等待進入額外球階段...")
		// 這個階段是常規球抽取結束後的過渡階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_PREPARE:
		log.Println("準備額外球階段，等待進入額外球側邊選擇階段...")
		// 這個階段是額外球準備階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_START:
		log.Println("開始選擇額外球側邊...")
		// 等待一段時間後選擇額外球側邊
		go func() {
			// 等待2秒，模擬選擇時間
			time.Sleep(2 * time.Second)
			log.Printf("選擇額外球側邊: %s", d.extraBallSide)
			d.startDrawingExtraBalls()
		}()

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_SIDE_SELECT_BETTING_CLOSED:
		log.Println("額外球側邊選擇已關閉，等待進入額外球抽取階段...")
		// 這個階段是額外球側邊選擇結束後的過渡階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_WAIT_CLAIM:
		log.Println("等待額外球聲明階段，等待進入額外球抽取階段...")
		// 這個階段是額外球聲明階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_START:
		log.Println("開始抽取額外球...")
		// 開始抽取額外球
		go d.drawExtraBalls()

	case commonpb.GameStage_GAME_STAGE_EXTRA_BALL_DRAWING_CLOSE:
		log.Println("額外球抽取已結束，等待進入結算階段...")
		// 這個階段是額外球抽取結束後的過渡階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_PAYOUT_SETTLEMENT:
		log.Println("進入支付結算階段，等待進入幸運球或頭獎階段...")
		// 這個階段是支付結算階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_START:
		log.Println("開始抽取幸運球...")
		// 開始抽取幸運球
		go d.drawLuckyBalls()

	case commonpb.GameStage_GAME_STAGE_DRAWING_LUCKY_BALLS_CLOSED:
		log.Println("幸運球抽取已結束，等待進入頭獎階段...")
		// 這個階段是幸運球抽取結束後的過渡階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_JACKPOT_START:
		log.Println("頭獎階段開始，等待進入頭獎抽取階段...")
		// 這個階段是頭獎階段開始，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_START:
		log.Println("開始抽取JP球...")
		// 開始抽取JP球
		go d.drawJPBalls()

	case commonpb.GameStage_GAME_STAGE_JACKPOT_DRAWING_CLOSED:
		log.Println("JP球抽取已結束，等待進入頭獎結算階段...")
		// 這個階段是JP球抽取結束後的過渡階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_JACKPOT_SETTLEMENT:
		log.Println("進入頭獎結算階段，等待進入遊戲結束階段...")
		// 這個階段是頭獎結算階段，無需特別操作，只需記錄日誌

	case commonpb.GameStage_GAME_STAGE_GAME_OVER:
		log.Printf("遊戲已結束，等待 %d 秒後開始新遊戲...", d.config.Timing.GameOverWaitSec)

		// 避免多次啟動新遊戲
		if !d.state.preparingNewGame {
			d.state.preparingNewGame = true

			// 啟動一個goroutine來處理新遊戲的開始
			go func() {
				// 等待指定的秒數
				select {
				case <-d.ctx.Done():
					return
				case <-time.After(time.Duration(d.config.Timing.GameOverWaitSec) * time.Second):
					// 繼續處理
				}

				// 重置標誌
				d.state.preparingNewGame = false

				// 開始新遊戲
				d.startNewGame()
			}()
		}

	default:
		log.Printf("未知的遊戲階段: %s", d.state.currentStage)
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
	log.Println("嘗試開始新遊戲...")

	// 重置遊戲狀態
	d.state.drawnRegularBalls = d.state.drawnRegularBalls[:0]
	d.state.drawnExtraBalls = d.state.drawnExtraBalls[:0]
	d.state.drawnJPBalls = d.state.drawnJPBalls[:0]
	d.state.drawnLuckyBalls = d.state.drawnLuckyBalls[:0]

	// 創建請求
	req := &dealerpb.StartNewRoundRequest{
		RoomId: d.roomID,
	}

	// 設置超時上下文
	ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
	defer cancel()

	// 嘗試最多3次
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		select {
		case <-d.ctx.Done():
			log.Println("操作被取消，停止嘗試開始新遊戲")
			return
		default:
			// 繼續嘗試
		}

		log.Printf("開始新遊戲 (嘗試 %d/3)...", attempt)

		// 發送請求
		resp, err := d.client.StartNewRound(ctx, req)
		if err != nil {
			lastErr = err
			log.Printf("開始新遊戲失敗: %v, 等待重試...", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// 成功啟動新遊戲
		if resp.GameData != nil {
			log.Printf("成功啟動新遊戲，遊戲ID: %s", resp.GameData.GameId)
		} else {
			log.Println("新遊戲已開始，但未返回遊戲數據")
		}
		return
	}

	// 所有嘗試都失敗
	log.Printf("經過多次嘗試後無法開始新遊戲: %v", lastErr)
}

// startDrawing 開始抽球
func (d *AutoDealer) startDrawing() {
	log.Println("啟動抽球階段...")

	// 直接由系統自動推進到抽球階段，無需請求
}

// drawRegularBalls 抽取常規球
func (d *AutoDealer) drawRegularBalls() {
	log.Println("開始抽取常規球...")

	// 抽取常規球
	for i := 0; i < d.config.Game.RegularBalls.Count; i++ {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 生成一個隨機球號
		ballNumber := d.generateUniqueBallNumber(d.state.drawnRegularBalls, d.config.Game.RegularBalls.MaxValue)

		// 判斷是否為最後一顆球
		isLast := i == d.config.Game.RegularBalls.Count-1

		// 創建抽球請求
		ball := &dealerpb.Ball{
			Number: ballNumber,
			Type:   dealerpb.BallType_BALL_TYPE_REGULAR,
			IsLast: isLast,
		}

		req := &dealerpb.DrawBallRequest{
			RoomId: d.roomID,
			Balls:  []*dealerpb.Ball{ball},
		}

		// 發送請求
		resp, err := d.client.DrawBall(d.ctx, req)
		if err != nil {
			log.Printf("抽取常規球失敗: %v", err)
			return
		}

		// 更新遊戲狀態
		if resp.GameData != nil {
			d.state.currentStage = resp.GameData.Stage
		}

		// 添加到已抽球列表
		d.state.drawnRegularBalls = append(d.state.drawnRegularBalls, ballNumber)

		log.Printf("抽取常規球成功，號碼: %d, 是否為最後一顆: %v", ballNumber, isLast)

		// 暫停一下，讓球抽取看起來更自然
		time.Sleep(time.Duration(d.config.Timing.RegularBallIntervalMs) * time.Millisecond)
	}
}

// startDrawingExtraBalls 開始抽取額外球
func (d *AutoDealer) startDrawingExtraBalls() {
	log.Println("準備抽取額外球...")

	// 抽取額外球無需特別請求，直接等待系統推進到抽額外球階段
}

// drawExtraBalls 抽取額外球
func (d *AutoDealer) drawExtraBalls() {
	log.Println("開始抽取額外球...")

	// 抽取額外球
	for i := 0; i < d.config.Game.ExtraBalls.Count; i++ {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 生成一個隨機球號
		ballNumber := d.generateUniqueBallNumber(d.state.drawnExtraBalls, d.config.Game.ExtraBalls.MaxValue)

		// 判斷是否為最後一顆球
		isLast := i == d.config.Game.ExtraBalls.Count-1

		// 創建抽球請求
		req := &dealerpb.DrawExtraBallRequest{
			RoomId: d.roomID,
			Side:   d.extraBallSide,
			Balls: []*dealerpb.Ball{
				{
					Number: ballNumber,
					Type:   dealerpb.BallType_BALL_TYPE_EXTRA,
					IsLast: isLast,
				},
			},
		}

		// 發送請求
		resp, err := d.client.DrawExtraBall(d.ctx, req)
		if err != nil {
			log.Printf("抽取額外球失敗: %v", err)
			return
		}

		// 更新遊戲狀態
		if resp.GameData != nil {
			d.state.currentStage = resp.GameData.Stage
		}

		// 添加到已抽球列表
		d.state.drawnExtraBalls = append(d.state.drawnExtraBalls, ballNumber)

		log.Printf("抽取額外球成功，號碼: %d, 是否為最後一顆: %v", ballNumber, isLast)

		// 暫停一下，讓球抽取看起來更自然
		time.Sleep(time.Duration(d.config.Timing.ExtraBallIntervalMs) * time.Millisecond)
	}
}

// drawLuckyBalls 抽取幸運球
func (d *AutoDealer) drawLuckyBalls() {
	log.Println("開始抽取幸運球...")

	// 抽取幸運球
	for i := 0; i < d.config.Game.LuckyBalls.Count; i++ {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 生成一個隨機球號
		ballNumber := d.generateUniqueBallNumber(d.state.drawnLuckyBalls, d.config.Game.LuckyBalls.MaxValue)

		// 判斷是否為最後一顆球
		isLast := i == d.config.Game.LuckyBalls.Count-1

		// 創建抽球請求
		req := &dealerpb.DrawLuckyBallRequest{
			RoomId: d.roomID,
			Balls: []*dealerpb.Ball{
				{
					Number: ballNumber,
					Type:   dealerpb.BallType_BALL_TYPE_LUCKY,
					IsLast: isLast,
				},
			},
		}

		// 發送請求
		resp, err := d.client.DrawLuckyBall(d.ctx, req)
		if err != nil {
			log.Printf("抽取幸運球失敗: %v", err)
			return
		}

		// 更新遊戲狀態
		if resp.GameData != nil {
			d.state.currentStage = resp.GameData.Stage
		}

		// 添加到已抽球列表
		d.state.drawnLuckyBalls = append(d.state.drawnLuckyBalls, ballNumber)

		log.Printf("抽取幸運球成功，號碼: %d, 是否為最後一顆: %v", ballNumber, isLast)

		// 暫停一下，讓球抽取看起來更自然
		time.Sleep(time.Duration(d.config.Timing.LuckyBallIntervalMs) * time.Millisecond)
	}
}

// drawJPBalls 抽取JP球
func (d *AutoDealer) drawJPBalls() {
	log.Println("開始抽取JP球...")

	// 抽取JP球
	for i := 0; i < d.config.Game.JackpotBalls.Count; i++ {
		// 檢查是否需要停止
		select {
		case <-d.ctx.Done():
			return
		default:
			// 繼續執行
		}

		// 生成一個隨機球號
		ballNumber := d.generateUniqueBallNumber(d.state.drawnJPBalls, d.config.Game.JackpotBalls.MaxValue)

		// 判斷是否為最後一顆球
		isLast := i == d.config.Game.JackpotBalls.Count-1

		// 創建抽球請求
		req := &dealerpb.DrawJackpotBallRequest{
			RoomId: d.roomID,
			Balls: []*dealerpb.Ball{
				{
					Number: ballNumber,
					Type:   dealerpb.BallType_BALL_TYPE_JACKPOT,
					IsLast: isLast,
				},
			},
		}

		// 發送請求
		resp, err := d.client.DrawJackpotBall(d.ctx, req)
		if err != nil {
			log.Printf("抽取JP球失敗: %v", err)
			return
		}

		// 更新遊戲狀態
		if resp.GameData != nil {
			d.state.currentStage = resp.GameData.Stage
		}

		// 添加到已抽球列表
		d.state.drawnJPBalls = append(d.state.drawnJPBalls, ballNumber)

		log.Printf("抽取JP球成功，號碼: %d, 是否為最後一顆: %v", ballNumber, isLast)

		// 暫停一下，讓球抽取看起來更自然
		time.Sleep(time.Duration(d.config.Timing.JackpotBallIntervalMs) * time.Millisecond)
	}
}

// generateUniqueBallNumber 生成一個不重複的球號
func (d *AutoDealer) generateUniqueBallNumber(drawnBalls []int32, maxValue int) int32 {
	for {
		// 生成一個1到maxValue的隨機數
		n := int32(rand.Intn(maxValue) + 1)

		// 檢查是否已經抽過
		exists := false
		for _, ball := range drawnBalls {
			if ball == n {
				exists = true
				break
			}
		}

		// 如果不重複，返回這個數
		if !exists {
			return n
		}
	}
}

// 更新 getConfig 函數來正確返回三個值
func getConfig() (string, string, string) {
	// 從環境變數獲取伺服器地址，默認為localhost:8080
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:8080"
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
