package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestGetAutomatonDir_Default(t *testing.T) {
	os.Unsetenv("AUTOMATON_DIR")
	dir := GetAutomatonDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".automaton")
	if dir != want {
		t.Errorf("GetAutomatonDir() = %q, want %q", dir, want)
	}
}

func TestGetAutomatonDir_Override(t *testing.T) {
	os.Setenv("AUTOMATON_DIR", "/tmp/auto")
	defer os.Unsetenv("AUTOMATON_DIR")
	dir := GetAutomatonDir()
	if dir != "/tmp/auto" {
		t.Errorf("GetAutomatonDir() = %q, want /tmp/auto", dir)
	}
}

func TestGetConfigPath(t *testing.T) {
	os.Setenv("AUTOMATON_DIR", "/tmp/auto")
	defer os.Unsetenv("AUTOMATON_DIR")
	path := GetConfigPath()
	if path != "/tmp/auto/automaton.json" {
		t.Errorf("GetConfigPath() = %q, want /tmp/auto/automaton.json", path)
	}
}

func TestResolvePath_WithTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := ResolvePath("~/foo")
	want := filepath.Join(home, "foo")
	if got != want {
		t.Errorf("ResolvePath(~/foo) = %q, want %q", got, want)
	}
}

func TestResolvePath_WithoutTilde(t *testing.T) {
	got := ResolvePath("/abs/path")
	if got != "/abs/path" {
		t.Errorf("ResolvePath(/abs/path) = %q, want /abs/path", got)
	}
}

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg != nil {
		t.Errorf("Load() cfg = %v, want nil", cfg)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")
	path := GetConfigPath()
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load() err = nil, want error")
	}
}

func TestLoad_MergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")
	path := GetConfigPath()
	raw := map[string]any{"name": "test-agent", "creatorAddress": "0x123"}
	data, _ := json.Marshal(raw)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() cfg = nil")
	}
	if cfg.Name != "test-agent" {
		t.Errorf("Name = %q, want test-agent", cfg.Name)
	}
	if cfg.ConwayAPIURL != "https://api.conway.tech" {
		t.Errorf("ConwayAPIURL = %q, want default", cfg.ConwayAPIURL)
	}
	if cfg.InferenceModel != "gpt-5.2" {
		t.Errorf("InferenceModel = %q, want gpt-5.2", cfg.InferenceModel)
	}
}

func TestLoad_TreasuryMerge(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")
	path := GetConfigPath()
	raw := map[string]any{
		"name": "x",
		"treasuryPolicy": map[string]any{
			"maxSingleTransferCents": 1000.0,
		},
	}
	data, _ := json.Marshal(raw)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg.TreasuryPolicy.MaxSingleTransferCents != 1000 {
		t.Errorf("MaxSingleTransferCents = %d, want 1000", cfg.TreasuryPolicy.MaxSingleTransferCents)
	}
	if cfg.TreasuryPolicy.MaxHourlyTransferCents != 10000 {
		t.Errorf("MaxHourlyTransferCents = %d, want 10000 (default)", cfg.TreasuryPolicy.MaxHourlyTransferCents)
	}
}

func TestSave_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested", "automaton")
	os.Setenv("AUTOMATON_DIR", sub)
	defer os.Unsetenv("AUTOMATON_DIR")

	cfg := &types.AutomatonConfig{Name: "save-test"}
	tp := types.DefaultTreasuryPolicy()
	cfg.TreasuryPolicy = &tp
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() err = %v", err)
	}
	if _, err := os.Stat(GetConfigPath()); os.IsNotExist(err) {
		t.Error("Save() did not create config file")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	orig := &types.AutomatonConfig{
		Name:           "roundtrip",
		GenesisPrompt:  "test",
		CreatorAddress: "0xabc",
	}
	tp := types.DefaultTreasuryPolicy()
	orig.TreasuryPolicy = &tp

	if err := Save(orig); err != nil {
		t.Fatalf("Save() err = %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if loaded.Name != orig.Name || loaded.GenesisPrompt != orig.GenesisPrompt {
		t.Errorf("Load() = %+v, want %+v", loaded, orig)
	}
}

func TestLoadToolsFromFile_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.json")
	data := []byte(`{"tools":[{"name":"echo","description":"Echo","parameters":"{}","type":"shell","command":"echo"}]}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	tools, err := LoadToolsFromFile(path)
	if err != nil {
		t.Fatalf("LoadToolsFromFile() err = %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("LoadToolsFromFile() = %d tools, want 1", len(tools))
	}
	if tools[0].Name != "echo" || tools[0].Command != "echo" {
		t.Errorf("tools[0] = %+v", tools[0])
	}
}

func TestLoadToolsFromFile_YAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	data := []byte("tools:\n  - name: greet\n    description: Greet\n    parameters: '{}'\n    command: echo hello")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	tools, err := LoadToolsFromFile(path)
	if err != nil {
		t.Fatalf("LoadToolsFromFile() err = %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("LoadToolsFromFile() = %d tools, want 1", len(tools))
	}
	if tools[0].Name != "greet" {
		t.Errorf("tools[0].Name = %q, want greet", tools[0].Name)
	}
}

func TestLoad_WithTools(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")
	path := GetConfigPath()
	raw := map[string]any{
		"name":  "tools-test",
		"tools": []any{
			map[string]any{"name": "custom_echo", "description": "Custom echo", "parameters": "{}", "command": "echo"},
		},
	}
	data, _ := json.Marshal(raw)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if len(cfg.Tools) != 1 {
		t.Fatalf("cfg.Tools = %d, want 1", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "custom_echo" {
		t.Errorf("cfg.Tools[0].Name = %q, want custom_echo", cfg.Tools[0].Name)
	}
}
