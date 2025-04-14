package utils

import (
	"math/rand"
	"sync"
	"time"
)

// RandomGenerator 提供線程安全的隨機數生成
type RandomGenerator struct {
	rng  *rand.Rand
	lock sync.Mutex
}

var (
	defaultGenerator *RandomGenerator
	once             sync.Once
)

// GetRandomGenerator 返回預設的隨機數生成器實例
func GetRandomGenerator() *RandomGenerator {
	once.Do(func() {
		defaultGenerator = &RandomGenerator{
			rng: rand.New(rand.NewSource(time.Now().UnixNano())),
		}
	})
	return defaultGenerator
}

// Intn 生成 [0,n) 範圍內的隨機整數
func (r *RandomGenerator) Intn(n int) int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.rng.Intn(n)
}

// WeightedChoice 根據權重進行隨機選擇
// weights 是權重列表，返回選中的索引
func (r *RandomGenerator) WeightedChoice(weights []int) int {
	r.lock.Lock()
	defer r.lock.Unlock()

	total := 0
	for _, w := range weights {
		total += w
	}

	choice := r.rng.Intn(total)
	for i, w := range weights {
		choice -= w
		if choice < 0 {
			return i
		}
	}
	return len(weights) - 1
}

// Shuffle 打亂切片順序
func (r *RandomGenerator) Shuffle(slice []interface{}) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for i := len(slice) - 1; i > 0; i-- {
		j := r.rng.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// Float64 返回 [0.0,1.0) 範圍內的隨機浮點數
func (r *RandomGenerator) Float64() float64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.rng.Float64()
}

// RandRange 生成指定範圍內的隨機整數 [min,max]
func (r *RandomGenerator) RandRange(min, max int) int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.rng.Intn(max-min+1) + min
}
