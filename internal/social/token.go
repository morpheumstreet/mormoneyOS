package social

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// TokenManager is the reusable, thread-safe refresh engine for social channel tokens.
// Lazy refresh: only refreshes on next GetAuthToken when token is empty or expired.
type TokenManager struct {
	mu          sync.RWMutex
	token       string
	expiresAt   time.Time
	refreshFunc func(ctx context.Context) (string, time.Time, error)
	logger      *slog.Logger
}

// NewTokenManager creates a TokenManager with the given refresh function.
// If logger is nil, slog.Default() is used.
func NewTokenManager(refreshFunc func(ctx context.Context) (string, time.Time, error), logger *slog.Logger) *TokenManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &TokenManager{
		refreshFunc: refreshFunc,
		logger:      logger,
	}
}

// GetAuthToken returns a valid token, refreshing if necessary.
func (m *TokenManager) GetAuthToken(ctx context.Context) (string, error) {
	m.mu.RLock()
	if m.token != "" && (m.expiresAt.IsZero() || time.Now().Before(m.expiresAt)) {
		tok := m.token
		m.mu.RUnlock()
		return tok, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if m.token != "" && (m.expiresAt.IsZero() || time.Now().Before(m.expiresAt)) {
		return m.token, nil
	}

	token, expiry, err := m.refreshFunc(ctx)
	if err != nil {
		m.logger.Error("token refresh failed", "err", err)
		return "", err
	}

	m.token = token
	m.expiresAt = expiry
	if !expiry.IsZero() {
		m.logger.Debug("token refreshed", "expires_in", time.Until(expiry))
	}
	return token, nil
}

// Invalidate clears the cached token so the next GetAuthToken will refresh.
// Call this on 401/403 to force re-fetch on retry.
func (m *TokenManager) Invalidate() {
	m.mu.Lock()
	m.token = ""
	m.expiresAt = time.Time{}
	m.mu.Unlock()
}
