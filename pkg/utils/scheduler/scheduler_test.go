package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduleOnce(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	var executed int32
	var wg sync.WaitGroup
	wg.Add(1)

	jobID := "test_once"

	// 安排一個1秒後執行的任務
	err := s.ScheduleOnce(1*time.Second, jobID, func() {
		atomic.StoreInt32(&executed, 1)
		wg.Done()
	})

	if err != nil {
		t.Fatalf("安排任務失敗: %v", err)
	}

	// 確認任務存在
	if !s.JobExists(jobID) {
		t.Fatalf("預期任務存在，但未找到")
	}

	// 等待任務執行完成
	wg.Wait()

	// 驗證任務已執行
	if atomic.LoadInt32(&executed) != 1 {
		t.Fatalf("任務沒有被執行")
	}

	// 確認任務已從映射中移除（因為是一次性任務）
	if s.JobExists(jobID) {
		t.Fatalf("預期任務執行後被移除，但仍然存在")
	}
}

func TestCancelJob(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	var executed int32
	jobID := "test_cancel"

	// 安排一個3秒後執行的任務
	err := s.ScheduleOnce(3*time.Second, jobID, func() {
		atomic.StoreInt32(&executed, 1)
	})

	if err != nil {
		t.Fatalf("安排任務失敗: %v", err)
	}

	// 確認任務存在
	if !s.JobExists(jobID) {
		t.Fatalf("預期任務存在，但未找到")
	}

	// 取消任務
	if !s.CancelJob(jobID) {
		t.Fatalf("取消任務失敗")
	}

	// 確認任務已被取消
	if s.JobExists(jobID) {
		t.Fatalf("預期任務被取消，但仍然存在")
	}

	// 等待足夠時間，確保任務不會執行
	time.Sleep(4 * time.Second)

	// 確認任務沒有被執行
	if atomic.LoadInt32(&executed) != 0 {
		t.Fatalf("任務應該被取消，但仍然執行了")
	}
}

func TestScheduleRecurring(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	var counter int32
	jobID := "test_recurring"

	// 安排一個每秒執行的週期性任務
	err := s.ScheduleRecurring(1*time.Second, jobID, func() {
		atomic.AddInt32(&counter, 1)
	})

	if err != nil {
		t.Fatalf("安排週期性任務失敗: %v", err)
	}

	// 確認任務存在
	if !s.JobExists(jobID) {
		t.Fatalf("預期任務存在，但未找到")
	}

	// 等待任務執行幾次
	time.Sleep(3500 * time.Millisecond)

	// 取消任務
	if !s.CancelJob(jobID) {
		t.Fatalf("取消任務失敗")
	}

	// 記錄當前計數
	count := atomic.LoadInt32(&counter)

	// 確認任務已被執行至少3次
	if count < 3 {
		t.Fatalf("預期任務執行至少3次，但只執行了%d次", count)
	}

	// 等待一段時間
	time.Sleep(2 * time.Second)

	// 確認計數沒有再增加
	newCount := atomic.LoadInt32(&counter)
	if newCount != count {
		t.Fatalf("任務應該已被取消，但計數從%d增加到了%d", count, newCount)
	}
}

func TestScheduleAtTime(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	var executed int32
	var wg sync.WaitGroup
	wg.Add(1)

	jobID := "test_at_time"

	// 安排一個在2秒後的特定時間執行的任務
	executionTime := time.Now().Add(2 * time.Second).Format("15:04:05")

	err := s.ScheduleAtTime(executionTime, jobID, func() {
		atomic.StoreInt32(&executed, 1)
		wg.Done()
	})

	if err != nil {
		t.Fatalf("安排指定時間任務失敗: %v", err)
	}

	// 確認任務存在
	if !s.JobExists(jobID) {
		t.Fatalf("預期任務存在，但未找到")
	}

	// 等待任務執行完成，設置較長的超時時間
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 任務完成
	case <-time.After(5 * time.Second):
		t.Fatalf("等待任務執行超時")
	}

	// 驗證任務已執行
	if atomic.LoadInt32(&executed) != 1 {
		t.Fatalf("任務沒有被執行")
	}

	// 確認任務已從映射中移除（因為是一次性任務）
	if s.JobExists(jobID) {
		t.Fatalf("預期任務執行後被移除，但仍然存在")
	}
}

func TestDuplicateJob(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	jobID := "test_duplicate"

	// 安排第一個任務
	err := s.ScheduleOnce(10*time.Second, jobID, func() {})
	if err != nil {
		t.Fatalf("安排第一個任務失敗: %v", err)
	}

	// 嘗試用相同的ID安排第二個任務
	err = s.ScheduleOnce(10*time.Second, jobID, func() {})
	if err == nil {
		t.Fatalf("預期重複任務ID會失敗，但成功了")
	}
}
