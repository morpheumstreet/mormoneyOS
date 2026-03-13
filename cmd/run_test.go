package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestRunRun_NoConfig(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	err := runRun(runCmd, nil)
	if err == nil {
		t.Error("runRun() err = nil, want error")
	}
	if !strings.Contains(err.Error(), "no config") && !strings.Contains(err.Error(), "setup") {
		t.Errorf("runRun() err = %q, want 'no config' or 'setup'", err.Error())
	}
}
