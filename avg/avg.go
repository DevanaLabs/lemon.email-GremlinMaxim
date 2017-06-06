package avg

import (
	"sync"
	"time"
)


type Avg struct {
	mu sync.Mutex
	value float64 
	num int
} 

func (a *Avg) AddValue(value float64) {
	a.mu.Lock()
	a.num++
	a.value += value
	a.mu.Unlock()
}

func (a *Avg) AddValuePerTime(value float64, start time.Time) {
	elapsed := float64(time.Now().Sub(start)) / float64(time.Second)

	a.mu.Lock()
	a.num++
	a.value += (value / elapsed)
	a.mu.Unlock()
}

func (a *Avg) GetValue() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.value / float64(a.num)
}

func NewAvg() *Avg {
	return new(Avg)
}
