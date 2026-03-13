package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunTestAPI_NoConfig(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	err := runTestAPI(testAPICmd, nil)
	if err == nil {
		t.Error("runTestAPI() err = nil, want error")
	}
	if !strings.Contains(err.Error(), "no config") {
		t.Errorf("runTestAPI() err = %q, want 'no config'", err.Error())
	}
}

func TestRunTestAPI_ChatJimmyOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","backend":"healthy"}`))
		case "/api/models":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":[{"id":"llama3.1-8B"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	os.Setenv("AUTOMATON_DIR", dir)
	defer os.Unsetenv("AUTOMATON_DIR")

	cfgPath := filepath.Join(dir, "automaton.json")
	cfg := map[string]any{
		"provider":         "chatjimmy",
		"chatjimmyApiUrl":  server.URL,
		"inferenceModel":   "llama3.1-8B",
		"maxTokensPerTurn": 4096,
	}
	b, _ := json.Marshal(cfg)
	if err := os.WriteFile(cfgPath, b, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := runTestAPI(testAPICmd, nil)
	if err != nil {
		t.Errorf("runTestAPI() err = %v", err)
	}
}
