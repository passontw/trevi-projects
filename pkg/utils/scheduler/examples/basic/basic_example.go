// Package main 展示如何使用排程器的基本功能
package main

import (
	"fmt"
	"g38_lottery_service/pkg/utils/scheduler"
	"time"
)

// 模擬的房間推進函數
func advanceRoomStage(roomID string, stage int) {
	fmt.Printf("房間 %s 已推進到階段 %d\n", roomID, stage)
}

func main() {
	// 創建一個新的排程器
	s := scheduler.New()

	// 啟動排程器
	s.Start()
	defer s.Stop()

	// 案例1: 安排一個一次性任務，3秒後執行
	roomID := "room123"
	jobID := fmt.Sprintf("advance_room_%s", roomID)

	err := s.ScheduleOnce(3*time.Second, jobID, advanceRoomStage, roomID, 2)
	if err != nil {
		fmt.Printf("排程錯誤: %v\n", err)
	} else {
		fmt.Printf("任務已排程: %s, 3秒後將執行\n", jobID)
	}

	// 案例2: 排程在特定時間執行
	futureTime := time.Now().Add(5 * time.Second).Format("15:04:05")
	jobID2 := fmt.Sprintf("advance_room_%s_specific_time", roomID)

	err = s.ScheduleAtTime(futureTime, jobID2, advanceRoomStage, roomID, 3)
	if err != nil {
		fmt.Printf("排程錯誤: %v\n", err)
	} else {
		fmt.Printf("任務已排程: %s, 將在 %s 執行\n", jobID2, futureTime)
	}

	// 案例3: 排程週期性任務，每2秒執行一次
	recurringJobID := "recurring_job"
	counter := 0

	// 這個匿名函數會追蹤執行次數，執行5次後取消任務
	err = s.ScheduleRecurring(2*time.Second, recurringJobID, func() {
		counter++
		fmt.Printf("週期性任務執行，計數: %d\n", counter)

		if counter >= 5 {
			fmt.Println("週期性任務已達到執行次數上限，取消任務")
			s.CancelJob(recurringJobID)
		}
	})

	if err != nil {
		fmt.Printf("排程週期性任務錯誤: %v\n", err)
	} else {
		fmt.Printf("週期性任務已排程: %s, 每2秒執行一次\n", recurringJobID)
	}

	// 檢查和取消任務的示例
	time.Sleep(1 * time.Second)

	if s.JobExists(jobID) {
		fmt.Printf("任務 %s 存在\n", jobID)
	}

	// 提前取消第一個任務
	if s.CancelJob(jobID) {
		fmt.Printf("任務 %s 已取消\n", jobID)
	}

	// 保持主程序運行，以便觀察任務執行
	time.Sleep(15 * time.Second)
}
