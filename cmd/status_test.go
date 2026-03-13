package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestRunStatus_NoConfig(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	r, w, _ := os.Pipe()
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	err := runStatus(statusCmd, nil)
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("runStatus() err = %v", err)
	}
	if !strings.Contains(buf.String(), "No config") {
		t.Errorf("runStatus() stderr = %q, want 'No config'", buf.String())
	}
}

func TestRunStatus_WithConfig(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	cfg := &types.AutomatonConfig{
		Name:           "test-status",
		ConwayAPIURL:   "https://api.conway.tech",
		DBPath:         filepath.Join(dir, "state.db"),
		TreasuryPolicy: func() *types.TreasuryPolicy { tp := types.DefaultTreasuryPolicy(); return &tp }(),
	}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	err := runStatus(statusCmd, nil)
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("runStatus() err = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "test-status") {
		t.Errorf("runStatus() output = %q, want 'test-status'", out)
	}
	if !strings.Contains(out, "Config:") {
		t.Errorf("runStatus() output = %q, want 'Config:'", out)
	}
}
