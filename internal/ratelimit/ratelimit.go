package ratelimit

import (
	"sync"
	"time"
)

type Bucket struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
	mu sync.Mutex
}

func New(capacity,refillRate float64) *Bucket {
	tokens := capacity
	lastRefill := time.Now()
	return &Bucket{tokens: tokens,lastRefill: lastRefill,capacity: capacity,refillRate: refillRate}
}

func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	elapsed :=time.Since(b.lastRefill)
	b.tokens = min(b.tokens+elapsed.Seconds()*b.refillRate,b.capacity)
	b.lastRefill=time.Now()
	if b.tokens >= 1 {
		b.tokens--
		return true
	}else {
		return false
	}
}