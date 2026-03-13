package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

type mockListSandboxesClient struct {
	sandboxes []conway.Sandbox
	err       error
}

func (m *mockListSandboxesClient) GetCreditsBalance(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockListSandboxesClient) GetCreditsPricing(ctx context.Context) (map[string]int64, error) {
	return nil, nil
}

func (m *mockListSandboxesClient) ListSandboxes(ctx context.Context) ([]conway.Sandbox, error) {
	return m.sandboxes, m.err
}

func (m *mockListSandboxesClient) ListModels(ctx context.Context) ([]conway.Model, error) {
	return nil, nil
}

func (m *mockListSandboxesClient) TransferCredits(ctx context.Context, toAddress string, amountCents int64, note string) (conway.CreditTransferResult, error) {
	return conway.CreditTransferResult{}, nil
}

func (m *mockListSandboxesClient) CreateSandbox(ctx context.Context, opts conway.CreateSandboxOptions) (conway.SandboxInfo, error) {
	return conway.SandboxInfo{}, nil
}

func (m *mockListSandboxesClient) DeleteSandbox(ctx context.Context, sandboxID string) error {
	return nil
}

func (m *mockListSandboxesClient) ExecInSandbox(ctx context.Context, sandboxID, command string, timeoutMs int) (conway.ExecResult, error) {
	return conway.ExecResult{}, nil
}

func (m *mockListSandboxesClient) WriteFileInSandbox(ctx context.Context, sandboxID, path, content string) error {
	return nil
}

func (m *mockListSandboxesClient) ReadFileInSandbox(ctx context.Context, sandboxID, path string) (string, error) {
	return "", nil
}

var _ conway.Client = (*mockListSandboxesClient)(nil)

func TestListSandboxesTool_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("returns no sandboxes when empty", func(t *testing.T) {
		mock := &mockListSandboxesClient{sandboxes: nil}
		tool := &ListSandboxesTool{Conway: mock}
		result, err := tool.Execute(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "No sandboxes found." {
			t.Errorf("got %q", result)
		}
	})

	t.Run("returns sandbox list when populated", func(t *testing.T) {
		mock := &mockListSandboxesClient{
			sandboxes: []conway.Sandbox{
				{ID: "sb-1", Name: "dev"},
				{ID: "sb-2", Name: "prod"},
			},
		}
		tool := &ListSandboxesTool{Conway: mock}
		result, err := tool.Execute(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "dev") || !strings.Contains(result, "sb-1") {
			t.Errorf("got %q", result)
		}
		if !strings.Contains(result, "prod") || !strings.Contains(result, "sb-2") {
			t.Errorf("got %q", result)
		}
	})

	t.Run("returns error when Conway nil", func(t *testing.T) {
		tool := &ListSandboxesTool{Conway: nil}
		_, err := tool.Execute(ctx, nil)
		if err != ErrConwayNotConfigured {
			t.Errorf("want ErrConwayNotConfigured, got %v", err)
		}
	})

	t.Run("returns error when API fails", func(t *testing.T) {
		mock := &mockListSandboxesClient{err: errors.New("api error")}
		tool := &ListSandboxesTool{Conway: mock}
		_, err := tool.Execute(ctx, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
