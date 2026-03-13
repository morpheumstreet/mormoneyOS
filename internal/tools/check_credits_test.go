package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

type mockConwayClient struct {
	balance int64
	err     error
}

func (m *mockConwayClient) GetCreditsBalance(ctx context.Context) (int64, error) {
	return m.balance, m.err
}

func (m *mockConwayClient) GetCreditsPricing(ctx context.Context) (map[string]int64, error) {
	return nil, nil
}

func (m *mockConwayClient) ListSandboxes(ctx context.Context) ([]conway.Sandbox, error) {
	return nil, nil
}

func (m *mockConwayClient) ListModels(ctx context.Context) ([]conway.Model, error) {
	return nil, nil
}

func (m *mockConwayClient) TransferCredits(ctx context.Context, toAddress string, amountCents int64, note string) (conway.CreditTransferResult, error) {
	return conway.CreditTransferResult{}, nil
}

func (m *mockConwayClient) CreateSandbox(ctx context.Context, opts conway.CreateSandboxOptions) (conway.SandboxInfo, error) {
	return conway.SandboxInfo{}, nil
}

func (m *mockConwayClient) DeleteSandbox(ctx context.Context, sandboxID string) error {
	return nil
}

func (m *mockConwayClient) ExecInSandbox(ctx context.Context, sandboxID, command string, timeoutMs int) (conway.ExecResult, error) {
	return conway.ExecResult{}, nil
}

func (m *mockConwayClient) WriteFileInSandbox(ctx context.Context, sandboxID, path, content string) error {
	return nil
}

func (m *mockConwayClient) ReadFileInSandbox(ctx context.Context, sandboxID, path string) (string, error) {
	return "", nil
}

var _ conway.Client = (*mockConwayClient)(nil)

func TestCheckCreditsTool_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("returns balance when Conway configured", func(t *testing.T) {
		mock := &mockConwayClient{balance: 1234}
		tool := &CheckCreditsTool{Conway: mock}
		result, err := tool.Execute(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "Credit balance: $12.34 (1234 cents)" {
			t.Errorf("got %q", result)
		}
	})

	t.Run("returns error when Conway nil", func(t *testing.T) {
		tool := &CheckCreditsTool{Conway: nil}
		_, err := tool.Execute(ctx, nil)
		if err != ErrConwayNotConfigured {
			t.Errorf("want ErrConwayNotConfigured, got %v", err)
		}
	})

	t.Run("returns error when API fails", func(t *testing.T) {
		mock := &mockConwayClient{err: errors.New("api error")}
		tool := &CheckCreditsTool{Conway: mock}
		_, err := tool.Execute(ctx, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
