package social

import (
	"context"
	"log/slog"
	"sync"
)

// ChannelManager holds social channels and coordinates their lifecycle.
// Channels that implement LifecycleChannel are started with ctx and stopped on Close.
type ChannelManager struct {
	channels map[string]SocialChannel
	started  bool
	mu       sync.Mutex
}

// NewChannelManager creates a manager for the given channels.
func NewChannelManager(channels map[string]SocialChannel) *ChannelManager {
	if channels == nil {
		channels = make(map[string]SocialChannel)
	}
	return &ChannelManager{channels: channels}
}

// Channels returns the channel map for polling and sending. Safe to call after Start.
func (m *ChannelManager) Channels() map[string]SocialChannel {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.channels
}

// Start starts all LifecycleChannel implementations. Call with the process ctx so
// shutdown (ctx.Done()) propagates. Idempotent.
func (m *ChannelManager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	for name, ch := range m.channels {
		if lc, ok := ch.(LifecycleChannel); ok {
			lc.Start(ctx)
			slog.Info("social channel started", "channel", name)
		}
	}
}

// Close stops all LifecycleChannel implementations and waits for them to exit.
// Idempotent. Call before process exit to avoid stuck goroutines.
func (m *ChannelManager) Close() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.started = false
	channels := m.channels
	m.mu.Unlock()

	for name, ch := range channels {
		if lc, ok := ch.(LifecycleChannel); ok {
			lc.Stop()
			slog.Info("social channel stopped", "channel", name)
		}
	}
}
