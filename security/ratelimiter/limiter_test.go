package ratelimiter

import (
	"testing"
	"time"
)

func TestLimiterAllowsWithinCapacityAndRejectsExcess(t *testing.T) {
	limiter := New(1, 2)

	if !limiter.Allow("client") {
		t.Fatal("expected first request to be allowed")
	}
	if !limiter.Allow("client") {
		t.Fatal("expected second request to be allowed")
	}
	if limiter.Allow("client") {
		t.Fatal("expected third immediate request to be rejected")
	}
}

func TestLimiterExpiresIdleBuckets(t *testing.T) {
	limiter := NewWithTTL(1, 1, time.Millisecond)
	if !limiter.Allow("client") {
		t.Fatal("expected request to be allowed")
	}

	limiter.mu.Lock()
	limiter.buckets["client"].lastRefill = time.Now().Add(-time.Minute)
	limiter.lastCleanup = time.Now().Add(-time.Minute)
	limiter.mu.Unlock()

	if !limiter.Allow("another-client") {
		t.Fatal("expected second client to be allowed")
	}

	limiter.mu.Lock()
	_, exists := limiter.buckets["client"]
	limiter.mu.Unlock()
	if exists {
		t.Fatal("expected idle bucket to be pruned")
	}
}
