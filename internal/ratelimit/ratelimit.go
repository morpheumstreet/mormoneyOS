package ratelimit

import (
	"context"
	"sync"
	"time"
)

// RateLimiter enforces a minimum interval between requests per key.
// Implementations must be safe for concurrent use.
type RateLimiter interface {
	// Allow returns true if the request is allowed; retryAfterSec is seconds until the next allowed request (0 when allowed).
	Allow(ctx context.Context, key string) (allowed bool, retryAfterSec int)
}

// MemoryRateLimiter is an in-memory implementation of RateLimiter.
// Suitable for single-process deployments.
type MemoryRateLimiter struct {
	mu       sync.Mutex
	lastCall map[string]time.Time
	cooldown time.Duration
}

// NewMemoryRateLimiter creates a rate limiter with the given cooldown in seconds.
// If cooldownSec <= 0, defaults to 120 seconds.
func NewMemoryRateLimiter(cooldownSec int) *MemoryRateLimiter {
	if cooldownSec <= 0 {
		cooldownSec = 120
	}
	return &MemoryRateLimiter{
		lastCall: make(map[string]time.Time),
		cooldown: time.Duration(cooldownSec) * time.Second,
	}
}

// Allow implements RateLimiter.
func (m *MemoryRateLimiter) Allow(ctx context.Context, key string) (bool, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	last, ok := m.lastCall[key]
	if !ok || now.Sub(last) >= m.cooldown {
		m.lastCall[key] = now
		return true, 0
	}
	retryAfter := int(time.Until(last.Add(m.cooldown)).Seconds())
	if retryAfter < 0 {
		retryAfter = 0
	}
	return false, retryAfter
}
