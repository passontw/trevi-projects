# Speed Bingo 開獎服務設計與實現報告

## 1. 專案概述

Speed Bingo 是一款快速出球的傳統5*5賓果遊戲，加上額外球機制組成主玩法，並在主遊戲中有機會觸發必爆JP玩法。本報告記錄了 Speed Bingo 開獎服務的設計與實現討論過程。

### 1.1 遊戲流程摘要

**主玩法**：
- 雙軸球機抽出30球 + 局末1~3顆額外球
- 依照達成連線的球數獲得對應獎金

**必爆JP玩法**：
- 當主遊戲抽出的球號完全匹配7個幸運號時觸發
- 單軸球機抽球，直到有玩家達到全盤獲得JP獎池所有獎金

## 2. 服務架構設計

### 2.1 核心服務組件

專案設計為四個主要服務：

1. **主持端**：
   - 顯示遊戲狀態和資訊給現場主持人
   - 提供遊戲進度、玩家參與情況、JP獎池等資訊

2. **荷官端**：
   - 控制遊戲進程
   - 開始新局(Game Start)
   - 監控球的抽出和識別(與RFID結合)

3. **開獎服務端**：
   - 處理開獎流程與自動開獎
   - 提供遊戲狀態API給其他服務
   - RFID球號識別處理
   - WebSocket實時通訊

4. **其餘遊戲功能端**：
   - 玩家註冊和身份驗證
   - 購卡和投注處理
   - 社交功能
   - 禮物系統

### 2.2 服務間的互動模式

- **解耦合**：各服務之間通過API交互，減少直接依賴
- **可擴展性**：開獎服務可以獨立擴展以應對高並發
- **職責清晰**：每個服務專注於自己的核心功能

## 3. 核心功能討論結果

### 3.1 開獎服務設計重點

開獎服務的主要職責：
1. 處理開獎流程
2. 執行自動開獎機制
3. 提供API讓其他服務讀取遊戲狀態

### 3.2 WebSocket 通訊

採用WebSocket實現實時遊戲數據推送，更新事件包括：
- 遊戲狀態變更
- 球號抽出通知
- 倒計時更新
- 遊戲結果

### 3.3 REST API 設計

提供統一的API端點格式，用於：
- 遊戲狀態查詢
- 歷史數據查詢
- 健康檢查
- 統計數據

## 4. 遊戲參數配置設計

提取遊戲流程的時間參數為可配置項：

```yaml
# 遊戲時間設定 (單位: 秒)
timings:
  # 主遊戲階段
  main_game:
    standby_time: 2           # 待命階段時間
    bet_time: 12              # 投注倒數時間
    drawing_interval: 0.5     # 每球開獎間隔
    extra_bet_time: 4         # 額外球押注時間
    extra_drawing_interval: 1 # 額外球開獎間隔
    result_time: 3            # 結算顯示時間
    jackpot_trigger_delay: 5  # 觸發JP延遲時間
  
  # JP遊戲階段
  jackpot_game:
    standby_time: 1           # JP待命階段時間
    bet_time: 30              # JP購卡時間
    drawing_interval: 0.3     # JP每球開獎間隔
    result_time: 30           # JP結算時間
```

## 5. WebSocket 事件類型設計

根據遊戲流程分析，設計了以下WebSocket事件類型：

### 5.1 主要事件類型

```go
// 事件類型常量
const (
    // 遊戲狀態事件
    EventGameStateChanged = "GAME_STATE_CHANGED"  // 遊戲階段變更
    EventGameReady = "GAME_READY"                // 遊戲準備就緒，伺服器已準備好開始新局
    
    // 主遊戲事件
    EventGameStart = "GAME_START"                 // 新局開始
    EventBetCountdown = "BET_COUNTDOWN"           // 投注倒計時
    EventBallDrawn = "BALL_DRAWN"                 // 新球抽出
    EventLuckyNumberMatch = "LUCKY_NUMBER_MATCH"  // 幸運號碼匹配
    EventLuckyNumbersSet = "LUCKY_NUMBERS_SET"    // 幸運號碼設定
    EventExtraBetStart = "EXTRA_BET_START"        // 額外球投注開始
    EventExtraBetStats = "EXTRA_BET_STATS"        // 額外球押注統計
    EventExtraBallDrawn = "EXTRA_BALL_DRAWN"      // 額外球抽出
    EventGameResult = "GAME_RESULT"               // 遊戲結果
    EventGameCancelled = "GAME_CANCELLED"         // 遊戲取消
    
    // JP遊戲事件
    EventJPGameStart = "JP_GAME_START"            // JP遊戲開始
    EventJPBetCountdown = "JP_BET_COUNTDOWN"      // JP投注倒計時
    EventJPCardStats = "JP_CARD_STATS"            // JP卡購買統計
    EventJPBallDrawn = "JP_BALL_DRAWN"            // JP球抽出
    EventJPTopPlayers = "JP_TOP_PLAYERS"          // JP前幾名玩家
    EventJPGameResult = "JP_GAME_RESULT"          // JP遊戲結果
    
    // 通用事件
    EventCountdownUpdate = "COUNTDOWN_UPDATE"     // 倒計時更新，以秒為單位
    EventOperationResult = "OPERATION_RESULT"     // 操作結果反饋
    EventError = "ERROR"                           // 錯誤消息
    EventServerShutdown = "SERVER_SHUTDOWN"        // 服務器關閉通知
)
```

### 5.2 事件和遊戲階段對應關係

| 遊戲階段 | 對應WebSocket事件 |
|---------|-----------------|
| 待命階段 (Stand by) | GAME_READY, GAME_STATE_CHANGED |
| 投注倒數階段 (Bet Time) | GAME_START, BET_COUNTDOWN, GAME_STATE_CHANGED |
| 開獎階段 (Drawing Balls) | BALL_DRAWN, LUCKY_NUMBER_MATCH, GAME_STATE_CHANGED |
| 額外球押注階段 (Buying L or R) | EXTRA_BET_START, EXTRA_BET_STATS, GAME_STATE_CHANGED |
| 額外球開獎階段 (Extra Drawing) | EXTRA_BALL_DRAWN, GAME_STATE_CHANGED |
| 結算階段 (Main Game Result) | GAME_RESULT, GAME_STATE_CHANGED |
| JP待命階段 (JP Stand by) | JP_GAME_START, GAME_STATE_CHANGED |
| JP購卡階段 (JP Buying cards) | JP_BET_COUNTDOWN, JP_CARD_STATS, GAME_STATE_CHANGED |
| JP開獎階段 (JP Drawing balls) | JP_BALL_DRAWN, JP_TOP_PLAYERS, GAME_STATE_CHANGED |
| JP結算階段 (JP Game Result) | JP_GAME_RESULT, GAME_STATE_CHANGED |

## 6. REST API 結構設計

### 6.1 API 端點設計

```
/api/health                   # 健康檢查
/api/health/liveness          # 存活檢查
/api/health/readiness         # 準備就緒檢查

/api/game/status              # 獲取當前遊戲狀態
/api/game/history/{gameId}    # 獲取特定局遊戲歷史
/api/game/lucky-numbers       # 獲取當前幸運號碼

/api/game/admin/start         # 開始新局(需認證)
/api/game/admin/force-draw    # 強制抽球(測試用)(需認證)
/api/game/admin/reset         # 重置遊戲(需認證)

/api/jackpot/info             # 獲取JP獎池信息
/api/jackpot/history          # 獲取JP歷史記錄

/api/stats                    # 獲取遊戲統計數據
/api/stats/players            # 獲取玩家統計
```

### 6.2 統一 API 響應格式

```json
{
  "status": {
    "code": 200,
    "message": "Success"
  },
  "data": {
    "game": {
      "id": "G12345",
      "phase": "DRAWING",
      "countdown": 10,
      "timestamp": 1650123456789
    },
    "drawing": {
      "mainBalls": [1, 5, 12, 24, 32],
      "extraBalls": [45],
      "drawnCount": 18,
      "totalCount": 30,
      "extraSide": "LEFT",
      "extraCount": 1
    },
    "lucky": {
      "numbers": [3, 14, 27, 35, 42, 56, 69],
      "matched": [true, false, true, false, false, true, false]
    },
    "jackpot": {
      "isActive": false,
      "pool": 125000,
      "drawnCount": 0,
      "totalCards": 0,
      "maxMatched": 0
    },
    "statistics": {
      "onlinePlayers": 1250,
      "totalBets": 5400,
      "leftBets": 1200,
      "rightBets": 1800
    }
  }
}
```

## 7. Docker 與 Kubernetes 部署設計

### 7.1 Dockerfile

```dockerfile
FROM golang:1.18-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lottery-service ./main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/lottery-service .
COPY configs/ ./configs/

# 健康檢查設定
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

EXPOSE 8080
EXPOSE 8081

ENTRYPOINT ["./lottery-service"]
```

### 7.2 Kubernetes 部署

**deployment.yaml 核心部分**：

```yaml
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
      - name: drawing-service
        image: your-registry/drawing-service:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8081
          name: websocket
        livenessProbe:
          httpGet:
            path: /api/health/liveness
            port: http
          initialDelaySeconds: 30
          periodSeconds: 15
        readinessProbe:
          httpGet:
            path: /api/health/readiness
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
        lifecycle:
          preStop:
            exec:
              command: ["sh", "-c", "sleep 10"]
        terminationGracePeriodSeconds: 60
```

## 8. 專案目錄結構

建議的專案目錄結構：

```
g38_lottery_service/
├── cmd/                           # 應用程式入口點
│   └── server/                    # 主要服務啟動
│       └── main.go                # 主程式
├── internal/                      # 內部套件，不對外暴露
│   ├── api/                       # API層
│   ├── config/                    # 設定管理
│   ├── domain/                    # 領域模型(業務邏輯)
│   ├── service/                   # 業務服務
│   ├── websocket/                 # WebSocket 服務
│   └── utils/                     # 工具函數
├── pkg/                           # 可能被外部使用的包
├── configs/                       # 設定檔
├── deployments/                   # 部署資源
└── test/                          # 測試
```

## 9. 優雅關閉設計

為了確保在Kubernetes環境中的優雅關閉，設計了以下機制：

1. **捕獲信號**：監聽SIGTERM和SIGINT信號
2. **停止接受新請求**：設置標誌禁止新連接
3. **WebSocket通知**：向所有連接的客戶端發送關閉通知
4. **等待遊戲階段完成**：在適當的遊戲狀態(STANDBY或RESULT)時關閉
5. **資源釋放**：關閉資料庫連接、Redis連接等外部資源

## 10. 總結

Speed Bingo 開獎服務設計基於Go語言，採用了模塊化的架構設計，將遊戲參數抽離為配置檔，支援WebSocket實時通訊和REST API狀態查詢。服務設計考慮了Kubernetes環境的健康檢查和優雅關閉，確保了整個系統的可靠性和擴展性。

完整的WebSocket事件類型設計覆蓋了遊戲所有階段的數據推送需求，統一的API響應格式提供了良好的開發體驗，整體架構符合微服務設計原則和最佳實踐。

## 11. 後續工作與建議

### 11.1 安全性考量

在實際部署前，建議進一步加強以下安全措施：

1. **WebSocket認證機制**：確保只有合法客戶端可以建立WebSocket連接
2. **API權限控制**：實施更細粒度的API權限控制，特別是管理API
3. **RFID數據驗證**：加強RFID數據的驗證機制，防止數據篡改
4. **日誌審計**：實施完整的操作日誌記錄，便於問題追蹤和審計

### 11.2 擴展性優化

隨著用戶量增長，以下方面可能需要進一步優化：

1. **服務水平擴展**：實現服務之間的無狀態設計，便於水平擴展
2. **緩存策略**：引入更多的緩存層，減少數據庫訪問
3. **消息隊列**：使用消息隊列處理高峰期的事件推送
4. **數據分片**：考慮遊戲歷史數據的分片存儲策略

### 11.3 監控與可觀測性

為了確保系統的穩定運行，建議實施以下監控措施：

1. **Prometheus指標**：暴露關鍵業務指標給Prometheus
2. **分布式追蹤**：實現OpenTelemetry追蹤，深入了解服務間調用關係
3. **告警機制**：設置多級告警閾值，及時發現問題
4. **故障演練**：定期進行故障演練，測試系統恢復能力

### 11.4 開發流程優化

為了提高開發效率，建議考慮以下改進：

1. **CI/CD優化**：建立完整的CI/CD流程，加速迭代
2. **測試自動化**：增加單元測試和集成測試覆蓋率
3. **代碼質量工具**：引入更多代碼質量檢查工具
4. **文檔自動生成**：實現API文檔的自動生成和更新

通過不斷改進這些方面，Speed Bingo開獎服務將能夠提供更穩定、安全和高效的遊戲體驗。
