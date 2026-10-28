package auth_test

import (
	"sync"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
)

func TestMemoryRateLimiter_AllowsAndBlocks(t *testing.T) {
	limiter := auth.NewMemoryRateLimiter(3, 100*time.Millisecond, 200*time.Millisecond)

	key := "client-ip-123"

	// Initially allowed
	if !limiter.Allow(key) {
		t.Fatal("expected key to be allowed initially")
	}

	// 2 failures -> still allowed
	limiter.RecordFailure(key)
	limiter.RecordFailure(key)
	if !limiter.Allow(key) {
		t.Fatal("expected key to be allowed after 2 failures")
	}

	// 3rd failure -> blocked
	limiter.RecordFailure(key)
	if limiter.Allow(key) {
		t.Fatal("expected key to be blocked after 3 failures")
	}

	// Different key is unaffected
	if !limiter.Allow("other-ip") {
		t.Fatal("expected different key to be unaffected")
	}

	// Reset key
	limiter.Reset(key)
	if !limiter.Allow(key) {
		t.Fatal("expected key to be allowed after reset")
	}
}

func TestMemoryRateLimiter_Concurrency(t *testing.T) {
	limiter := auth.NewMemoryRateLimiter(10, 1*time.Minute, 1*time.Minute)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "concurrent-ip"
			limiter.RecordFailure(key)
			_ = limiter.Allow(key)
		}(i)
	}

	wg.Wait()
}
