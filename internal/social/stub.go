package social

import (
	"context"
	"fmt"
)

// notImplementedChannel returns errors for Send/Poll; used for placeholder channels.
type notImplementedChannel struct {
	name string
}

func (n *notImplementedChannel) Name() string { return n.name }

func (n *notImplementedChannel) Send(ctx context.Context, msg *OutboundMessage) (string, error) {
	return "", fmt.Errorf("%s channel not implemented yet", n.name)
}

func (n *notImplementedChannel) Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error) {
	return nil, "", fmt.Errorf("%s channel not implemented yet", n.name)
}

func (n *notImplementedChannel) HealthCheck(ctx context.Context) error {
	return nil
}

// StubChannel is a no-op channel for tests or when no config is available.
type StubChannel struct {
	stubName string
}

// NewStubChannel returns a stub that does nothing.
func NewStubChannel(name string) *StubChannel {
	return &StubChannel{stubName: name}
}

func (s *StubChannel) Name() string {
	if s.stubName != "" {
		return s.stubName
	}
	return "stub"
}

func (s *StubChannel) Send(ctx context.Context, msg *OutboundMessage) (string, error) {
	return "stub-id", nil
}

func (s *StubChannel) Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error) {
	return nil, "", nil
}

func (s *StubChannel) HealthCheck(ctx context.Context) error {
	return nil
}
