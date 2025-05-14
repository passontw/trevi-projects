package scheduler

import (
	"fmt"
	"time"
)

// LotteryScheduler 是排程器的擴展，提供專為開獎服務設計的方法
type LotteryScheduler struct {
	*Scheduler
}

// NewLotteryScheduler 創建並啟動一個新的開獎服務排程器
func NewLotteryScheduler() *LotteryScheduler {
	s := New()
	s.Start()
	return &LotteryScheduler{Scheduler: s}
}

// ScheduleRoomAdvance 安排房間階段推進
// roomID: 房間ID
// delay: 延遲時間，例如 3*time.Second
// stageFunc: 房間推進階段的處理函數，簽名應為 func(roomID string, stageInfo ...interface{})
// stageInfo: 傳遞給處理函數的額外參數
func (ls *LotteryScheduler) ScheduleRoomAdvance(
	roomID string,
	delay time.Duration,
	stageFunc interface{},
	stageInfo ...interface{},
) error {
	jobID := fmt.Sprintf("room_advance_%s_%d", roomID, time.Now().UnixNano())
	return ls.ScheduleOnce(delay, jobID, stageFunc, append([]interface{}{roomID}, stageInfo...)...)
}

// CancelRoomAdvances 取消指定房間的所有推進任務
// 這是一個示例方法，實際生產環境中可能需要更精確的任務跟踪方法
func (ls *LotteryScheduler) CancelRoomAdvances(roomID string) int {
	allJobs := ls.GetAllJobs()
	count := 0

	for _, jobID := range allJobs {
		// 簡單匹配，實際應用中可能需要更精確的規則
		if jobPrefix := fmt.Sprintf("room_advance_%s_", roomID); len(jobID) > len(jobPrefix) && jobID[:len(jobPrefix)] == jobPrefix {
			if ls.CancelJob(jobID) {
				count++
			}
		}
	}

	return count
}

// ScheduleCountdown 設置倒數計時並在結束後執行指定函數
// duration: 倒數時間
// onComplete: 倒數完成後要執行的函數
// params: 傳遞給完成函數的參數
func (ls *LotteryScheduler) ScheduleCountdown(
	duration time.Duration,
	countdownID string,
	onComplete interface{},
	params ...interface{},
) error {
	jobID := fmt.Sprintf("countdown_%s", countdownID)
	return ls.ScheduleOnce(duration, jobID, onComplete, params...)
}

// SchedulePeriodicTask 設置定期執行的任務
// 適用於定期檢查或處理的場景
func (ls *LotteryScheduler) SchedulePeriodicTask(
	interval time.Duration,
	taskID string,
	taskFunc interface{},
	params ...interface{},
) error {
	jobID := fmt.Sprintf("periodic_%s", taskID)
	return ls.ScheduleRecurring(interval, jobID, taskFunc, params...)
}

// ScheduleBatchProcess 安排批次處理任務，在特定時間執行
func (ls *LotteryScheduler) ScheduleBatchProcess(
	executionTime string, // 格式: "15:04:05"
	batchID string,
	processFunc interface{},
	params ...interface{},
) error {
	jobID := fmt.Sprintf("batch_%s", batchID)
	return ls.ScheduleAtTime(executionTime, jobID, processFunc, params...)
}
