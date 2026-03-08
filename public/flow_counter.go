package public

import (
	"fmt"
	"sync"
	"time"
)

type FlowCounter struct {
	ServiceName string
	TotalCount  int64
	QPS         int64
	mu          sync.Mutex
	lastTime    time.Time
	lastCount   int64
}

type FlowCounterHandlerStruct struct {
	counters map[string]*FlowCounter
	mu       sync.Mutex
}

var FlowCounterHandler = &FlowCounterHandlerStruct{
	counters: make(map[string]*FlowCounter),
}

func (f *FlowCounterHandlerStruct) GetCounter(serviceName string) (*FlowCounter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if counter, exists := f.counters[serviceName]; exists {
		return counter, nil
	}

	counter := &FlowCounter{
		ServiceName: serviceName,
		TotalCount:  0,
		QPS:         0,
		lastTime:    time.Now(),
		lastCount:   0,
	}
	f.counters[serviceName] = counter
	return counter, nil
}

func (fc *FlowCounter) Increase() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.TotalCount++

	// 计算QPS
	now := time.Now()
	duration := now.Sub(fc.lastTime).Seconds()
	if duration >= 1 {
		fc.QPS = int64(float64(fc.TotalCount-fc.lastCount) / duration)
		fc.lastCount = fc.TotalCount
		fc.lastTime = now
	}
	fmt.Println(fc.QPS)
}
