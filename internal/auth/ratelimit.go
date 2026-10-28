package auth

import (
	"sync"
	"time"
)

// RateLimiter bounds failed authentication attempts to protect against brute-force attacks.
type RateLimiter interface {
	Allow(key string) bool
	RecordFailure(key string)
	Reset(key string)
}

type attemptInfo struct {
	failures  []time.Time
	blockedTo time.Time
}

// MemoryRateLimiter implements an in-memory sliding window rate limiter.
type MemoryRateLimiter struct {
	mu            sync.Mutex
	maxFailures   int
	window        time.Duration
	blockDuration time.Duration
	attempts      map[string]*attemptInfo
}

// NewMemoryRateLimiter returns an initialized rate limiter.
func NewMemoryRateLimiter(maxFailures int, window, blockDuration time.Duration) *MemoryRateLimiter {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if window <= 0 {
		window = 1 * time.Minute
	}
	if blockDuration <= 0 {
		blockDuration = 1 * time.Minute
	}

	return &MemoryRateLimiter{
		maxFailures:   maxFailures,
		window:        window,
		blockDuration: blockDuration,
		attempts:      make(map[string]*attemptInfo),
	}
}

// Allow returns true if the key is permitted to attempt authentication.
func (rl *MemoryRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	info, exists := rl.attempts[key]
	if !exists {
		return true
	}

	now := time.Now().UTC()
	if !info.blockedTo.IsZero() && info.blockedTo.After(now) {
		return false
	}

	return true
}

// RecordFailure notes a failed authentication attempt and blocks the key if limit is reached.
func (rl *MemoryRateLimiter) RecordFailure(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UTC()
	cutoff := now.Add(-rl.window)

	info, exists := rl.attempts[key]
	if !exists {
		info = &attemptInfo{}
		rl.attempts[key] = info
	}

	// Filter out failures outside the sliding window
	var validFailures []time.Time
	for _, t := range info.failures {
		if t.After(cutoff) {
			validFailures = append(validFailures, t)
		}
	}
	validFailures = append(validFailures, now)
	info.failures = validFailures

	if len(info.failures) >= rl.maxFailures {
		info.blockedTo = now.Add(rl.blockDuration)
	}
}

// Reset clears recorded failures for a key (e.g. upon successful login).
func (rl *MemoryRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, key)
}
