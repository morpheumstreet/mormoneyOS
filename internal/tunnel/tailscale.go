package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// TailscaleProvider exposes localhost via Tailscale Funnel.
// Requires authKey for headless join; runs tailscale up then tailscale funnel.
func TailscaleProvider(pc types.TunnelProviderConfig) TunnelProvider {
	if pc.AuthKey == "" {
		return nil
	}
	return &tailscaleProvider{
		authKey:  pc.AuthKey,
		hostname: pc.Hostname,
		funnel:   pc.Funnel,
	}
}

type tailscaleProvider struct {
	authKey  string
	hostname string
	funnel   bool
	mu       sync.Mutex
	cancel   map[int]context.CancelFunc
}

func (t *tailscaleProvider) init() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel == nil {
		t.cancel = make(map[int]context.CancelFunc)
	}
}

func (t *tailscaleProvider) Name() string { return "tailscale" }

func (t *tailscaleProvider) IsAvailable() bool {
	_, err := exec.LookPath("tailscale")
	return err == nil
}

func (t *tailscaleProvider) Start(ctx context.Context, host string, port int) (string, error) {
	t.init()
	// Run: tailscale up --auth-key=$TS_AUTHKEY (joins tailnet), then tailscale funnel/serve {port}
	exposeCmd := "tailscale serve"
	if t.funnel {
		exposeCmd = "tailscale funnel"
	}
	shellCmd := fmt.Sprintf("tailscale up --auth-key=$TS_AUTHKEY && %s %d", exposeCmd, port)
	if t.hostname != "" {
		shellCmd = fmt.Sprintf("tailscale up --auth-key=$TS_AUTHKEY --hostname=%s && %s %d", t.hostname, exposeCmd, port)
	}
	bgCtx, cancel := context.WithCancel(context.Background())
	t.mu.Lock()
	t.cancel[port] = cancel
	t.mu.Unlock()

	cmd := exec.CommandContext(bgCtx, "sh", "-c", shellCmd)
	cmd.Env = append(os.Environ(), "TS_AUTHKEY="+t.authKey)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return "", err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return "", err
	}

	publicURL := fmt.Sprintf("http://%s:%d", host, port)
	urlCh := make(chan string, 1)
	readPipe := func(r *bufio.Scanner) {
		for r.Scan() {
			if u := extractURL(r.Text(), "https://"); u != "" {
				select {
				case urlCh <- u:
				default:
				}
				return
			}
		}
	}
	go readPipe(bufio.NewScanner(stdout))
	go readPipe(bufio.NewScanner(stderr))

	select {
	case u := <-urlCh:
		publicURL = u
	case <-time.After(15 * time.Second):
	}
	return publicURL, nil
}

func (t *tailscaleProvider) Stop(port int) error {
	t.init()
	t.mu.Lock()
	cancel, ok := t.cancel[port]
	delete(t.cancel, port)
	t.mu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
	return nil
}

// compile-time check
var _ TunnelProvider = (*tailscaleProvider)(nil)
