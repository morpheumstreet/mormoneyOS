package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryRateLimiter_Allow(t *testing.T) {
	ctx := context.Background()
	rl := NewMemoryRateLimiter(2) // 2 second cooldown

	// First call allowed
	allowed, retry := rl.Allow(ctx, "user1")
	if !allowed || retry != 0 {
		t.Errorf("first call: allowed=%v retry=%d, want allowed=true retry=0", allowed, retry)
	}

	// Immediate second call denied
	allowed, retry = rl.Allow(ctx, "user1")
	if allowed || retry <= 0 {
		t.Errorf("immediate second call: allowed=%v retry=%d, want allowed=false retry>0", allowed, retry)
	}

	// Different key allowed
	allowed, retry = rl.Allow(ctx, "user2")
	if !allowed || retry != 0 {
		t.Errorf("different key: allowed=%v retry=%d, want allowed=true retry=0", allowed, retry)
	}

	// After cooldown, allowed again
	time.Sleep(2100 * time.Millisecond)
	allowed, retry = rl.Allow(ctx, "user1")
	if !allowed || retry != 0 {
		t.Errorf("after cooldown: allowed=%v retry=%d, want allowed=true retry=0", allowed, retry)
	}
}

func TestMemoryRateLimiter_DefaultCooldown(t *testing.T) {
	rl := NewMemoryRateLimiter(0)
	if rl == nil {
		t.Fatal("NewMemoryRateLimiter(0) returned nil")
	}
	allowed, _ := rl.Allow(context.Background(), "key")
	if !allowed {
		t.Error("first call with cooldown=0 should be allowed (defaults to 120)")
	}
}
