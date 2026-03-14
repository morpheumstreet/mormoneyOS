package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CommandTunnelProvider is the base for providers that run a CLI command.
// Subclasses supply: CommandTemplate, URLPattern, Binary.
// Replacements: optional map for {key} placeholders (e.g. {"token": "xxx"}).
type CommandTunnelProvider struct {
	ProviderName    string
	CommandTemplate string            // "bore local {port} --to bore.pub"
	URLPattern      string            // "https://" or substring to find URL in stdout
	Binary          string            // "bore" — for IsAvailable() check
	Env             []string          // optional env vars (merged with os.Environ())
	Replacements    map[string]string // optional {key} → value for command template
	mu              sync.Mutex
	portToCancel    map[int]context.CancelFunc
}

// NewCommandTunnelProvider creates a command-based provider.
func NewCommandTunnelProvider(name, cmdTemplate, urlPattern, binary string, env []string) *CommandTunnelProvider {
	return &CommandTunnelProvider{
		ProviderName:    name,
		CommandTemplate: cmdTemplate,
		URLPattern:      urlPattern,
		Binary:          binary,
		Env:             env,
		portToCancel:    make(map[int]context.CancelFunc),
	}
}

// NewCommandTunnelProviderWithReplacements creates a provider with {key} replacements in the command.
func NewCommandTunnelProviderWithReplacements(name, cmdTemplate, urlPattern, binary string, env []string, replacements map[string]string) *CommandTunnelProvider {
	c := NewCommandTunnelProvider(name, cmdTemplate, urlPattern, binary, env)
	c.Replacements = replacements
	return c
}

// Name returns the provider name.
func (c *CommandTunnelProvider) Name() string {
	return c.ProviderName
}

// IsAvailable returns true if the binary is in PATH.
func (c *CommandTunnelProvider) IsAvailable() bool {
	_, err := exec.LookPath(c.Binary)
	return err == nil
}

// Start runs the command and parses stdout for the public URL.
func (c *CommandTunnelProvider) Start(ctx context.Context, host string, port int) (string, error) {
	cmdStr := strings.ReplaceAll(c.CommandTemplate, "{port}", fmt.Sprintf("%d", port))
	cmdStr = strings.ReplaceAll(cmdStr, "{host}", host)
	for k, v := range c.Replacements {
		cmdStr = strings.ReplaceAll(cmdStr, "{"+k+"}", v)
	}
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command for provider %s", c.ProviderName)
	}

	// Use background context so process survives after Start returns.
	// Cancel is stored for Stop() to terminate the process.
	bgCtx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	c.portToCancel[port] = cancel
	c.mu.Unlock()

	cmd := exec.CommandContext(bgCtx, parts[0], parts[1:]...)
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
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

	// Default: local URL
	publicURL := fmt.Sprintf("http://%s:%d", host, port)
	urlCh := make(chan string, 1)

	readPipe := func(r *bufio.Scanner) {
		for r.Scan() {
			if u := extractURL(r.Text(), c.URLPattern); u != "" {
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
		// Keep default local URL
	}

	return publicURL, nil
}

// Stop cancels the context for the given port, terminating the subprocess.
func (c *CommandTunnelProvider) Stop(port int) error {
	c.mu.Lock()
	cancel, ok := c.portToCancel[port]
	delete(c.portToCancel, port)
	c.mu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
	return nil
}

func extractURL(line, pattern string) string {
	if pattern == "" {
		pattern = "https://"
	}
	idx := strings.Index(line, pattern)
	if idx < 0 {
		idx = strings.Index(line, "http://")
	}
	if idx < 0 {
		return ""
	}
	urlPart := line[idx:]
	// Trim to end of URL (whitespace or end)
	end := 0
	for i, r := range urlPart {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '\'' {
			end = i
			break
		}
		end = i + 1
	}
	url := strings.TrimSpace(urlPart[:end])
	// Basic validation
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return ""
}

// compile-time check: *CommandTunnelProvider implements TunnelProvider
var _ TunnelProvider = (*CommandTunnelProvider)(nil)
