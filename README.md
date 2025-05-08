# 賓果抽球遊戲服務技術與規則報告

## 一、系統概述

賓果抽球遊戲服務是一個基於 Go 語言實現的實時開獎服務，支持賓果球遊戲的完整流程管理。系統通過 WebSocket 與荷官端和遊戲端建立雙向通訊，提供高效、穩定的開獎體驗。服務使用 Uber FX 進行依賴注入，Redis 進行遊戲狀態持久化，TiDB 進行歷史資料儲存。

**核心功能:**
- 管理遊戲流程狀態轉換
- 處理實時球抽取與記錄
- 支持自動選邊機制
- 支持額外球和Jackpot抽取
- 支持遊戲取消與恢復
- 管理幸運號碼

## 二、系統架構

### 1. 整體架構

```
┌──────────────────────────────────────────────────────────┐
│                    賓果抽球遊戲服務                         │
│                                                          │
│  ┌─────────────┐         ┌────────────────┐              │
│  │             │         │                │              │
│  │ Rocket MQ   │◄───────►│                │              │
│  │ (遊戲端通訊)  │         │                │              │
│  └─────────────┘         │   遊戲流程管理   │              │
│                          │                │              │
│  ┌─────────────┐         │                │              │
│  │             │         │                │              │
│  │  荷官端WS模組  │◄───────►│                │              │
│  │             │         └───────┬────────┘              │
│  └─────────────┘                 │                       │
│                                  │                       │
│                          ┌───────▼────────┐              │
│                          │                │              │
│                          │  賓果球狀態管理  │              │
│                          │                │              │
│                          └───────┬────────┘              │
│                                  │                       │
│                          ┌───────▼────────┐              │
│                          │                │              │
│                          │  資料持久化模組  │              │
│                          │ (Redis/TiDB)  │              │
│                          └────────────────┘              │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### 2. 主要模組

- **遊戲流程管理模組**：控制遊戲階段流轉，處理狀態轉換，是系統核心模組，與荷官端WS和Rocket MQ直接連接
- **賓果球狀態管理模組**：管理賓果球抽取、記錄和驗證，由遊戲流程管理模組調用
- **荷官端WebSocket通訊模組**：處理荷官端的即時通訊，直接與遊戲流程管理模組交互
- **Rocket MQ通訊模組**：處理與遊戲端的消息傳遞，替代原WebSocket模組，提供解耦和擴展性
- **資料持久化模組**：處理Redis資料緩存和TiDB永久儲存
- **依賴注入模組**：使用fx管理模組間依賴關係

### 3. 技術棧

- **程式語言**：Go 1.22+
- **Web通訊**：WebSocket (gorilla/websocket) 用於荷官端，Rocket MQ用於遊戲端
- **依賴注入**：Uber FX
- **資料儲存**：Redis (遊戲狀態), TiDB (歷史資料)
- **並發控制**：標準庫sync
- **設定管理**：Nacos

## 三、遊戲階段定義

遊戲分為四個主要階段，每個階段又細分為多個子階段：

### 1. 主遊戲流程階段

```go
// 主遊戲流程
StagePreparation        // 遊戲準備階段
StageNewRound           // 新局開始
StageCardPurchaseOpen   // 開始購卡
StageCardPurchaseClose  // 購卡結束
StageDrawingStart       // 開始抽球
StageDrawingClose       // 抽球結束
```

### 2. 額外球流程階段

```go
// 額外球流程
StageExtraBallPrepare                  // 額外球準備
StageExtraBallSideSelectBettingStart   // 額外球選邊開始
StageExtraBallSideSelectBettingClosed  // 額外球選邊結束
StageExtraBallWaitClaim                // 等待額外球兌獎
StageExtraBallDrawingStart             // 額外球抽球開始
StageExtraBallDrawingClose             // 額外球抽球結束
```

### 3. 派彩與JP流程階段

```go
// 派彩與JP流程
StagePayoutSettlement         // 派彩結算
StageJackpotStart             // JP遊戲開始
StageJackpotDrawingStart      // JP抽球開始
StageJackpotDrawingClosed     // JP抽球結束
StageJackpotSettlement        // JP結算
StageDrawingLuckyBallsStart   // 幸運號碼抽球開始
StageDrawingLuckyBallsClosed  // 幸運號碼抽球結束
```

### 4. 結束階段

```go
// 結束階段
StageGameOver  // 遊戲結束
```

## 四、遊戲流程詳解

### 1. 服務啟動初始化

服務啟動時執行以下初始化流程：
1. 檢查資料庫是否有幸運號碼，沒有則自動產生7個幸運號碼並儲存
2. 檢查Redis是否有上一局未完成的遊戲資料：
   - 如有，恢復該遊戲狀態
   - 如無，設定遊戲狀態為`StagePreparation`準備階段

### 2. 標準遊戲流程

#### 2.1 遊戲準備階段
- 狀態：`StagePreparation`
- 描述：等待荷官端發送遊戲開始的命令
- 觸發條件：荷官發送`START_NEW_ROUND`命令
- 超時：無限(-1)，需荷官手動觸發
- 下一階段：`StageNewRound`

#### 2.2 新局開始階段
- 狀態：`StageNewRound`
- 描述：建立新遊戲並儲存到Redis，通知遊戲端遊戲開始
- 超時：2秒自動進入下一階段
- 下一階段：`StageCardPurchaseOpen`

#### 2.3 購卡開始階段
- 狀態：`StageCardPurchaseOpen`
- 描述：開始購卡，通知荷官端和遊戲端
- 超時：12秒自動進入下一階段
- 下一階段：`StageCardPurchaseClose`

#### 2.4 購卡結束階段
- 狀態：`StageCardPurchaseClose`
- 描述：結束購卡，通知遊戲端
- 超時：1秒自動進入下一階段
- 下一階段：`StageDrawingStart`

#### 2.5 開始抽球階段
- 狀態：`StageDrawingStart`
- 描述：開始抽球，通知荷官端和遊戲端
- 處理邏輯：荷官端發送抽到的球的資訊，系統同步給遊戲端
- 超時：無限(-1)，由荷官控制進度
- 下一階段：當收到最後一顆球時進入`StageDrawingClose`

#### 2.6 抽球結束階段
- 狀態：`StageDrawingClose`
- 描述：結束抽球，通知遊戲端
- 超時：1秒自動進入下一階段
- 下一階段：`StageExtraBallPrepare`

### 3. 額外球流程

#### 3.1 額外球準備階段
- 狀態：`StageExtraBallPrepare`
- 描述：準備進入額外球環節
- 超時：1秒自動進入下一階段
- 下一階段：`StageExtraBallSideSelectBettingStart`

#### 3.2 額外球選邊開始階段
- 狀態：`StageExtraBallSideSelectBettingStart`
- 描述：開始額外球選邊，系統自動選擇左邊或右邊
- 處理邏輯：系統隨機選擇左側或右側，並廣播結果
- 超時：1秒自動進入下一階段
- 下一階段：`StageExtraBallSideSelectBettingClosed`

#### 3.3 額外球選邊結束階段
- 狀態：`StageExtraBallSideSelectBettingClosed`
- 描述：額外球選邊結束
- 超時：1秒自動進入下一階段
- 下一階段：`StageExtraBallDrawingStart`

#### 3.4 額外球抽球開始階段
- 狀態：`StageExtraBallDrawingStart`
- 描述：開始抽取額外球
- 處理邏輯：荷官端發送抽到的額外球資訊，系統同步給遊戲端
- 超時：無限(-1)，由荷官控制進度
- 下一階段：當收到最後一顆額外球(1-3顆)時進入`StageExtraBallDrawingClose`

#### 3.5 額外球抽球結束階段
- 狀態：`StageExtraBallDrawingClose`
- 描述：額外球抽取結束
- 超時：1秒自動進入下一階段
- 下一階段：`StagePayoutSettlement`

### 4. 派彩與JP流程

#### 4.1 派彩結算階段
- 狀態：`StagePayoutSettlement`
- 描述：進行派彩結算
- 超時：3秒自動進入下一階段
- 下一階段：如有JP則進入`StageJackpotStart`，否則進入`StageGameOver`

#### 4.2 JP開始階段
- 狀態：`StageJackpotStart`
- 描述：開始JP遊戲
- 超時：3秒自動進入下一階段
- 下一階段：`StageJackpotDrawingStart`

#### 4.3 JP抽球開始階段
- 狀態：`StageJackpotDrawingStart`
- 描述：開始抽取JP球
- 處理邏輯：荷官端發送抽到的JP球資訊，系統同步給遊戲端
- 特殊處理：遊戲端可隨時通知有人中獎，此時停止JP
- 超時：無限(-1)，由荷官控制或直到有人中獎
- 下一階段：如有人中獎或抽完球則進入`StageJackpotDrawingClosed`

#### 4.4 JP抽球結束階段
- 狀態：`StageJackpotDrawingClosed`
- 描述：JP抽球結束
- 超時：1秒自動進入下一階段
- 下一階段：`StageJackpotSettlement`

#### 4.5 JP結算階段
- 狀態：`StageJackpotSettlement`
- 描述：進行JP派彩結算
- 超時：3秒自動進入下一階段
- 下一階段：`StageDrawingLuckyBallsStart`

#### 4.6 幸運號碼抽球開始階段
- 狀態：`StageDrawingLuckyBallsStart`
- 描述：開始抽取幸運號碼球
- 處理邏輯：荷官端發送抽到的幸運球資訊，系統同步給遊戲端
- 超時：無限(-1)，由荷官控制進度
- 下一階段：抽完7顆幸運球或收到最後一顆球標記時進入`StageDrawingLuckyBallsClosed`

#### 4.7 幸運號碼抽球結束階段
- 狀態：`StageDrawingLuckyBallsClosed`
- 描述：幸運號碼抽球結束
- 超時：1秒自動進入下一階段
- 下一階段：`StageGameOver`

#### 4.8 遊戲結束階段
- 狀態：`StageGameOver`
- 描述：遊戲完全結束
- 處理邏輯：清除Redis中的遊戲數據，將遊戲歷程資料寫入TiDB
- 超時：1秒自動進入下一階段
- 下一階段：回到`StagePreparation`準備下一局

## 五、系統功能與特性

### 1. 階段管理

系統對遊戲階段的管理有以下特點：

#### 1.1 階段定義與轉換

- **標準轉換邏輯**：使用`naturalStageTransition`映射定義標準階段轉換路徑
- **特殊轉換邏輯**：針對特殊情況(如有無JP)實現條件判斷式轉換
- **自動超時轉換**：每個階段可設定超時時間，到期後自動進入下一階段
- **手動轉換**：荷官可發送`FORCE_ADVANCE_STAGE`命令強制進入下一階段

#### 1.2 階段配置

每個階段都配置了以下參數：
- **超時時間**：自動轉入下一階段的時間，-1表示無限
- **荷官要求**：是否需要荷官確認才能進入下一階段
- **遊戲要求**：是否需要遊戲端確認才能進入下一階段
- **允許抽球**：該階段是否允許抽球操作
- **最大球數**：該階段最多可抽球數量

### 2. 球處理邏輯

系統支持四種不同類型的球處理：

#### 2.1 常規球處理
- 數量：最多75顆
- 處理方法：`handleDrawBall`
- 驗證規則：
  - 球號必須在1-75之間
  - 不能重複抽取同一顆球
- 儲存位置：`RegularBalls`陣列
- 結束條件：收到標記為最後一顆的球

#### 2.2 額外球處理
- 數量：1-3顆
- 處理方法：`handleDrawExtraBall`
- 驗證規則：
  - 球號必須在1-75之間
  - 不能與已抽取的常規球或額外球重複
- 儲存位置：`ExtraBalls`陣列
- 相關特性：系統會自動選擇左側或右側

#### 2.3 JP球處理
- 數量：不固定，最多到75顆
- 處理方法：`handleDrawJackpotBall`
- 驗證規則：
  - 球號必須在1-75之間
  - 不能重複抽取同一顆JP球
- 儲存位置：`Jackpot.DrawnBalls`陣列
- 結束條件：收到標記為最後一顆的球或遊戲端通知有人中獎

#### 2.4 幸運號碼球處理
- 數量：7顆
- 處理方法：`handleDrawLuckyBall`
- 驗證規則：
  - 球號必須在1-75之間
  - 不能重複抽取同一顆幸運球
- 儲存位置：`Jackpot.LuckyBalls`陣列
- 特殊處理：服務初始化時自動生成

### 3. WebSocket通訊

系統使用WebSocket作為荷官端實時通訊方式，使用Rocket MQ作為遊戲端通訊方式：

#### 3.1 荷官端WebSocket (`dealerWebsocket`)
- 端口：由配置設定
- 主要功能：
  - 接收荷官命令
  - 發送遊戲狀態
  - 處理抽球命令
  - 處理遊戲控制命令
- 直接與遊戲流程管理模組交互，不直接與賓果球狀態管理模組連接

#### 3.2 遊戲端Rocket MQ通訊
- 主題設計：
  - `bingo.game.status`：遊戲狀態更新
  - `bingo.game.{gameId}`：特定遊戲的相關通知
  - `bingo.game.draw`：抽球結果通知
  - `bingo.jp.win`：JP中獎通知
- 主要優勢：
  - 解耦遊戲端與服務端，提高系統彈性
  - 支持水平擴展，應對高併發連接
  - 消息持久化，確保狀態更新不丟失
  - 負載均衡，避免單點壓力
  - 支持主題和標籤過濾，精確推送

#### 3.3 通訊協議

所有通訊遵循統一格式：
```json
{
  "type": "訊息類型",
  "stage": "遊戲階段",
  "event": "事件類型",
  "data": {},
  "timestamp": 1653123456
}
```

### 4. 取消局功能

系統支持在遊戲進行中取消當前局：

#### 4.1 取消流程
1. 荷官發送`CANCEL_GAME`命令，附帶取消原因
2. 系統將遊戲標記為已取消，並記錄取消原因和時間
3. 保存遊戲數據到TiDB，包含取消標記和原因
4. 清除Redis中的遊戲數據
5. 廣播遊戲取消事件
6. 重置系統狀態為準備階段

#### 4.2 取消事件通知
系統會向所有連接的遊戲端和荷官端發送取消通知：
```json
{
  "type": "event",
  "event": "GAME_CANCELLED",
  "data": {
    "game_id": "遊戲ID",
    "reason": "取消原因"
  },
  "timestamp": 1653123456
}
```

### 5. 資料持久化

系統使用雙層持久化策略：

#### 5.1 Redis緩存
- 用途：存儲當前進行中的遊戲狀態
- 存儲鍵：`bingo:current_game`
- 內容：完整的遊戲數據，包括階段、抽球記錄等
- 存活時間：24小時
- 作用：
  - 提供高速訪問
  - 支持服務重啟後的狀態恢復
  - 減輕數據庫負擔

#### 5.2 TiDB永久儲存
- 用途：存儲已完成或已取消的遊戲歷史記錄
- 存儲內容：
  - 遊戲基本信息
  - 所有抽球記錄
  - 額外球選邊
  - JP中獎情況
  - 幸運號碼
  - 取消狀態和原因
- 寫入時機：
  - 遊戲正常結束時
  - 遊戲被取消時
  - 系統重置時

#### 5.3 資料流程
- 賓果球狀態管理模組只與Redis緩存交互
- 遊戲流程管理模組負責在適當時機將資料從Redis遷移到TiDB
- Redis作為TiDB的前置快取，遊戲進行中的所有資料只存在於Redis

## 六、系統安全性與穩定性

### 1. 並發控制

系統使用多種機制確保在高並發場景下的穩定性：

#### 1.1 互斥鎖保護
- 使用`sync.RWMutex`保護共享數據，區分讀寫操作提高效率
- 關鍵操作如階段轉換、遊戲狀態更新等都有鎖保護
- 使用延遲解鎖(`defer mu.Unlock()`)確保鎖總是被釋放

#### 1.2 原子操作
- 使用通道(`chan`)進行安全的並發通訊
- 階段轉換具有原子性，不會出現中間狀態
- 計時器處理中確保不同階段的計時器不會互相干擾

### 2. 錯誤處理

系統實現了完善的錯誤處理機制：

#### 2.1 主要錯誤處理策略
- **輸入驗證**：所有荷官端和遊戲端輸入都進行嚴格驗證
- **階段檢查**：所有操作都會檢查當前遊戲階段是否允許該操作
- **錯誤傳播**：所有錯誤都記錄日誌並適當傳播給客戶端
- **優雅降級**：當部分功能出錯時，系統可繼續運行核心功能

#### 2.2 常見錯誤處理
- **無效球號**：拒絕處理並返回錯誤
- **重複球號**：檢測並拒絕重複球號
- **非法階段操作**：檢查是否允許在當前階段進行操作
- **WebSocket斷連**：自動清理斷開的連接，並支持重連
- **Redis連接錯誤**：記錄錯誤並嘗試重連

### 3. 容錯與恢復

系統設計了多層容錯與恢復機制：

#### 3.1 服務恢復機制
- 服務啟動時自動從Redis恢復未完成的遊戲狀態
- 啟動時自動檢查並確保幸運號碼存在
- 使用優雅啟動和關閉機制，確保服務狀態一致性

#### 3.2 計時器管理
- 計時器創建時綁定當前階段，防止舊計時器觸發不必要的轉換
- 每次階段轉換時停止並清除舊計時器
- 使用`time.AfterFunc`確保計時器回調在獨立的goroutine中執行

## 七、編譯

### 編譯 Linux 64位元版本

```

$ GOOS=linux GOARCH=amd64 go build -o build/lottery_service ./cmd/lottery_service/main.go

```

### 編譯 Windows 64位元版本

```
$ GOOS=windows GOARCH=amd64 go build -o build/lottery_service.exe ./cmd/lottery_service/main.go
```

### 編譯 macOS 64位元版本

```
$ GOOS=darwin GOARCH=amd64 go build -o build/lottery_service ./cmd/lottery_service/main.go
```

## 八、總結

賓果抽球遊戲開獎服務是一個基於Golang開發的高效實時開獎系統，通過WebSocket提供雙向通訊能力，使用Redis進行狀態持久化，使用TiDB進行歷史數據儲存。系統支持完整的賓果球遊戲流程，包括常規球抽取、額外球選邊、JP抽球和幸運號碼管理。

### 核心優勢
1. **狀態管理清晰**：使用明確定義的遊戲階段和轉換規則
2. **實時性高**：荷官端使用WebSocket，遊戲端使用Rocket MQ確保通訊延遲極低
3. **可靠性強**：使用Redis持久化和TiDB備份，支持服務恢復，Rocket MQ消息持久化確保遊戲端訊息不丟失
4. **可擴展性好**：模組化設計，使用Uber FX進行依賴注入，Rocket MQ支持水平擴展處理大量遊戲端連接
5. **安全性高**：完善的並發控制和錯誤處理
6. **架構靈活**：通過Rocket MQ解耦遊戲端與服務端，提高系統彈性和穩定性

### 後續優化方向
1. **效能監控**：增加系統指標監控，如響應時間、連接數、消息吞吐量等
2. **負載均衡**：支持多實例部署，實現水平擴展，特別是針對Rocket MQ消費者
3. **連接安全**：增強WebSocket和Rocket MQ連接的安全性，如添加身份驗證
4. **管理介面**：開發後台管理介面，方便操作和監控系統狀態
5. **自動化測試**：增加自動化測試覆蓋，確保系統穩定性 
6. **消息序列化優化**：優化Rocket MQ消息格式和序列化方式，降低網路傳輸開銷 