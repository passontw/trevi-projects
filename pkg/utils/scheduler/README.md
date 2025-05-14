# 排程模組 (Scheduler)

這個模組基於 [go-co-op/gocron](https://github.com/go-co-op/gocron) 實現，提供了延遲任務執行的功能，特別適用於開獎服務的倒數計時需求。

## 功能

- 安排一次性任務（延遲執行）
- 按特定時間執行任務
- 安排週期性任務
- 取消已排程的任務
- 查詢任務狀態
- 開獎服務專用的排程功能

## 安裝

確保已經安裝了相關依賴：

```bash
go get github.com/go-co-op/gocron
```

## 基本用法

### 創建和啟動排程器

```go
import (
    "g38_lottery_service/pkg/utils/scheduler"
    "time"
)

// 創建一個新的排程器
s := scheduler.New()

// 啟動排程器
s.Start()
defer s.Stop()
```

### 安排一次性任務

```go
// 3秒後執行函數
err := s.ScheduleOnce(3*time.Second, "job_id", myFunction, param1, param2)
if err != nil {
    // 處理錯誤
}
```

### 安排特定時間執行的任務

```go
// 在指定時間執行函數 (格式: "15:04:05")
err := s.ScheduleAtTime("14:30:00", "time_job_id", myFunction, param1, param2)
if err != nil {
    // 處理錯誤
}
```

### 安排週期性任務

```go
// 每30秒執行一次
err := s.ScheduleRecurring(30*time.Second, "recurring_job_id", myFunction, param1, param2)
if err != nil {
    // 處理錯誤
}
```

### 取消任務

```go
// 取消指定ID的任務
if s.CancelJob("job_id") {
    // 任務已取消
} else {
    // 任務不存在或取消失敗
}
```

## 開獎服務專用功能

對於開獎服務，我們提供了更專業的封裝：

```go
// 創建開獎服務專用排程器
ls := scheduler.NewLotteryScheduler()
defer ls.Stop()

// 安排房間階段推進
err := ls.ScheduleRoomAdvance("room_123", 3*time.Second, advanceRoomStage, StageDrawing)

// 設置倒數計時
err := ls.ScheduleCountdown(5*time.Second, "countdown_id", onCountdownComplete, param1, param2)

// 取消房間的所有推進任務
count := ls.CancelRoomAdvances("room_123")
```

## 範例

查看 `examples` 目錄中的完整範例：

- `basic/basic_example.go`: 基本排程功能演示
- `lottery/lottery_example.go`: 開獎服務倒數功能演示

## 執行範例

```bash
# 基本功能範例
go run pkg/utils/scheduler/examples/basic/basic_example.go

# 開獎服務範例
go run pkg/utils/scheduler/examples/lottery/lottery_example.go
```

## 注意事項

- 所有任務都需要一個唯一的 jobID
- 一次性任務執行後會自動從排程器中移除
- 在應用關閉前調用 `Stop()` 方法以釋放資源
- 任務函數可以是任何函數或方法，參數將通過反射調用傳遞 