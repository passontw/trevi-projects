package utils

import (
	"fmt"
	"sync"
	"time"

	"g38_lottery_servic/internal/interfaces/types"
)

const (
	workerBits   uint8 = 10
	sequenceBits uint8 = 12
	workerMax    int64 = -1 ^ (-1 << workerBits)
	sequenceMask int64 = -1 ^ (-1 << sequenceBits)
	timeShift    uint8 = workerBits + sequenceBits
	workerShift  uint8 = sequenceBits
	epoch        int64 = 1577808000000
)

type Snowflake struct {
	mu        sync.Mutex
	timestamp int64
	workerId  int64
	sequence  int64
}

var _ types.IDManager = (*Snowflake)(nil)
var (
	snowflakeInstance *Snowflake
	snowflakeOnce     sync.Once
)

func InitSnowflake(workerId int64) error {
	var err error
	snowflakeOnce.Do(func() {
		if workerId < 0 || workerId > workerMax {
			err = fmt.Errorf("worker ID must be between 0 and %d", workerMax)
			return
		}

		snowflakeInstance = &Snowflake{
			workerId: workerId,
		}
	})
	return err
}

func GetNextID() (int64, error) {
	if snowflakeInstance == nil {
		return 0, fmt.Errorf("snowflake not initialized")
	}
	return snowflakeInstance.NextID()
}

func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano() / 1e6

	if now < s.timestamp {
		return 0, fmt.Errorf("clock moved backwards")
	}

	if now == s.timestamp {
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		s.sequence = 0
	}

	s.timestamp = now

	id := ((now - epoch) << timeShift) |
		(s.workerId << workerShift) |
		s.sequence

	return id, nil
}

func (s *Snowflake) ParseID(id int64) map[string]interface{} {
	timestamp := (id >> timeShift) + epoch
	workerId := (id >> workerShift) & workerMax
	sequence := id & sequenceMask

	t := time.Unix(timestamp/1000, (timestamp%1000)*1000000)

	return map[string]interface{}{
		"timestamp": t,
		"worker_id": workerId,
		"sequence":  sequence,
	}
}

func (s *Snowflake) ValidateID(id int64) bool {
	if id <= 0 {
		return false
	}

	parsed := s.ParseID(id)

	timestamp := parsed["timestamp"].(time.Time)
	if timestamp.Before(time.Unix(epoch/1000, 0)) ||
		timestamp.After(time.Now().Add(time.Hour)) {
		return false
	}

	workerId := parsed["worker_id"].(int64)
	if workerId < 0 || workerId > workerMax {
		return false
	}

	sequence := parsed["sequence"].(int64)
	if sequence < 0 || sequence > sequenceMask {
		return false
	}

	return true
}

func ParseID(id int64) map[string]interface{} {
	if snowflakeInstance == nil {
		snowflakeInstance = &Snowflake{}
	}
	return snowflakeInstance.ParseID(id)
}

func ValidateID(id int64) bool {
	if snowflakeInstance == nil {
		snowflakeInstance = &Snowflake{}
	}
	return snowflakeInstance.ValidateID(id)
}
