package RABCOUNT

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type RABCOUNT struct {
	counter    Counter
	slots      [60]int64
	index      int
	mu         sync.Mutex
	ticker     *time.Ticker
	countCPS   int64
	currentCPM int64
	maxCPM     int64
	second     int

	running    int32
	onStop     func(r *RABCOUNT)
	onStopLock sync.RWMutex
}

func NewRABCOUNT() *RABCOUNT {
	r := &RABCOUNT{}
	go r.startTicker()
	return r
}

func (r *RABCOUNT) Incr(val int64) {
	r.counter.Incr(val)
	r.mu.Lock()
	r.slots[r.index] += val

	sum := int64(0)
	for _, v := range r.slots {
		sum += v
	}
	r.currentCPM = sum
	if sum > r.maxCPM {
		r.maxCPM = sum
	}
	r.mu.Unlock()
}

func (r *RABCOUNT) Rate() int64 {
	return r.counter.Value()
}

func (r *RABCOUNT) CPM() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentCPM
}

func (r *RABCOUNT) MaxCPM() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxCPM
}

func (r *RABCOUNT) CPS() int64 {
	return atomic.LoadInt64(&r.countCPS)
}

func (r *RABCOUNT) String() string {
	return strconv.FormatInt(r.counter.Value(), 10)
}

func (r *RABCOUNT) OnStop(f func(*RABCOUNT)) {
	r.onStopLock.Lock()
	r.onStop = f
	r.onStopLock.Unlock()
}

func (r *RABCOUNT) startTicker() {
	r.ticker = time.NewTicker(1 * time.Second)
	for range r.ticker.C {
		r.mu.Lock()

		prevIndex := (r.index + 59) % 60
		r.countCPS = r.slots[prevIndex]
		r.index = (r.index + 1) % 60
		r.slots[r.index] = 0

		r.second++
		if r.second >= 60 {
			r.second = 0
			if r.onStop != nil {
				r.onStop(r)
			}
		}

		r.mu.Unlock()
	}
}

// --- Counter ---
type Counter int64

func (c *Counter) Incr(val int64) {
	atomic.AddInt64((*int64)(c), val)
}

func (c *Counter) Reset() {
	atomic.StoreInt64((*int64)(c), 0)
}

func (c *Counter) Value() int64 {
	return atomic.LoadInt64((*int64)(c))
}
