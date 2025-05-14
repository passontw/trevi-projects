// Package scheduler 提供基於 go-co-op/gocron 的排程功能，
// 支持延遲任務執行，主要用於開獎服務的倒數功能和定時任務處理。
package scheduler

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

// Scheduler 是對 gocron 排程器的包裝，提供延遲執行和管理任務的功能
type Scheduler struct {
	scheduler *gocron.Scheduler
	jobs      map[string]*gocron.Job
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// New 創建並返回一個新的排程器實例
func New() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		scheduler: gocron.NewScheduler(time.Local),
		jobs:      make(map[string]*gocron.Job),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 啟動排程器
func (s *Scheduler) Start() {
	s.scheduler.StartAsync()
}

// Stop 停止排程器
func (s *Scheduler) Stop() {
	s.scheduler.Stop()
	s.cancel()
}

// ScheduleOnce 安排一個一次性任務，在指定的延遲後執行
// - delay: 延遲時間，例如 3*time.Second
// - jobID: 任務ID，用於識別和取消任務
// - fn: 要執行的函數
// - params: 函數參數
func (s *Scheduler) ScheduleOnce(delay time.Duration, jobID string, fn interface{}, params ...interface{}) error {
	// 計算執行時間
	executionTime := time.Now().Add(delay)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 檢查是否已存在相同ID的任務
	if _, exists := s.jobs[jobID]; exists {
		return fmt.Errorf("job with ID %s already exists", jobID)
	}

	// 創建一次性任務
	job, err := s.scheduler.At(executionTime.Format("15:04:05")).Do(func() {
		// 執行任務
		s.executeJob(jobID, fn, params...)

		// 任務執行後從映射中移除
		s.mutex.Lock()
		delete(s.jobs, jobID)
		s.mutex.Unlock()
	})

	if err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	// 設置為一次性任務
	job.SingletonMode()

	// 儲存任務引用
	s.jobs[jobID] = job

	return nil
}

// CancelJob 取消指定ID的任務
func (s *Scheduler) CancelJob(jobID string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		s.scheduler.RemoveByReference(job)
		delete(s.jobs, jobID)
		return true
	}

	return false
}

// JobExists 檢查指定ID的任務是否存在
func (s *Scheduler) JobExists(jobID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.jobs[jobID]
	return exists
}

// executeJob 執行任務函數
func (s *Scheduler) executeJob(jobID string, fn interface{}, params ...interface{}) {
	// 嘗試執行反射函數調用
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Error executing job %s: %v\n", jobID, r)
		}
	}()

	// 使用反射機制調用函數
	f := reflect.ValueOf(fn)
	if f.Kind() != reflect.Func {
		fmt.Printf("Error: %s is not a function\n", jobID)
		return
	}

	// 準備參數
	inputs := make([]reflect.Value, len(params))
	for i, param := range params {
		inputs[i] = reflect.ValueOf(param)
	}

	// 調用函數
	f.Call(inputs)
}

// ScheduleAtTime 安排一個在指定時間執行的一次性任務
// - timeStr: 時間字符串，格式為 "15:04:05"
// - jobID: 任務ID
// - fn: 要執行的函數
// - params: 函數參數
func (s *Scheduler) ScheduleAtTime(timeStr string, jobID string, fn interface{}, params ...interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 檢查是否已存在相同ID的任務
	if _, exists := s.jobs[jobID]; exists {
		return fmt.Errorf("job with ID %s already exists", jobID)
	}

	// 創建指定時間的任務
	job, err := s.scheduler.At(timeStr).Do(func() {
		// 執行任務
		s.executeJob(jobID, fn, params...)

		// 任務執行後從映射中移除
		s.mutex.Lock()
		delete(s.jobs, jobID)
		s.mutex.Unlock()
	})

	if err != nil {
		return fmt.Errorf("failed to schedule job at time %s: %w", timeStr, err)
	}

	// 設置為一次性任務
	job.SingletonMode()

	// 儲存任務引用
	s.jobs[jobID] = job

	return nil
}

// ScheduleRecurring 安排一個週期性任務
// - interval: 時間間隔，例如 30*time.Second
// - jobID: 任務ID
// - fn: 要執行的函數
// - params: 函數參數
func (s *Scheduler) ScheduleRecurring(interval time.Duration, jobID string, fn interface{}, params ...interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 檢查是否已存在相同ID的任務
	if _, exists := s.jobs[jobID]; exists {
		return fmt.Errorf("job with ID %s already exists", jobID)
	}

	// 創建週期性任務
	job, err := s.scheduler.Every(interval).Do(func() {
		// 執行任務
		s.executeJob(jobID, fn, params...)
	})

	if err != nil {
		return fmt.Errorf("failed to schedule recurring job: %w", err)
	}

	// 儲存任務引用
	s.jobs[jobID] = job

	return nil
}

// GetAllJobs 返回所有當前活動的任務ID
func (s *Scheduler) GetAllJobs() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	jobIDs := make([]string, 0, len(s.jobs))
	for id := range s.jobs {
		jobIDs = append(jobIDs, id)
	}

	return jobIDs
}
