package cmd

import (
	"os"
	"testing"
)

func TestRunInit(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	err := runInit(initCmd, nil)
	if err != nil {
		t.Fatalf("runInit() err = %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("init did not create directory")
	}
}

func TestRunInit_Idempotent(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	_ = runInit(initCmd, nil)
	err := runInit(initCmd, nil)
	if err != nil {
		t.Fatalf("runInit() second call err = %v", err)
	}
}
