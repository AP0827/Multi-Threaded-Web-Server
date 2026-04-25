package ratelimiter

import (
	"sync"
	"time"
)

// tokens: current available capacity (can be fractional)
// lastRefill: last time we updated tokens
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64
	capacity float64
}

func New(rate float64, capacity float64) *Limiter {
	return &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		capacity: capacity,
	}
}

func (l *Limiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]

	//New bucket
	if !ok {
		l.buckets[key] = &bucket{
			tokens:     l.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	//Refill
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * l.rate
	b.lastRefill = now

	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}
