package state

import (
	"path/filepath"
	"testing"
)

func TestOpen_CreatesDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()
}

func TestOpen_WALMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()
	var mode string
	err = db.db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestInsertWakeEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	err = db.InsertWakeEvent("heartbeat", "credits")
	if err != nil {
		t.Fatalf("InsertWakeEvent() err = %v", err)
	}
}

func TestHasUnconsumedWakeEvents_Empty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	has, err := db.HasUnconsumedWakeEvents()
	if err != nil {
		t.Fatalf("HasUnconsumedWakeEvents() err = %v", err)
	}
	if has {
		t.Error("HasUnconsumedWakeEvents() = true, want false")
	}
}

func TestHasUnconsumedWakeEvents_WithEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	_ = db.InsertWakeEvent("heartbeat", "credits")
	has, err := db.HasUnconsumedWakeEvents()
	if err != nil {
		t.Fatalf("HasUnconsumedWakeEvents() err = %v", err)
	}
	if !has {
		t.Error("HasUnconsumedWakeEvents() = false, want true")
	}
}

func TestConsumeWakeEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	_ = db.InsertWakeEvent("a", "r1")
	_ = db.InsertWakeEvent("b", "r2")
	count, err := db.ConsumeWakeEvents()
	if err != nil {
		t.Fatalf("ConsumeWakeEvents() err = %v", err)
	}
	if count != 2 {
		t.Errorf("ConsumeWakeEvents() count = %d, want 2", count)
	}
}

func TestConsumeWakeEvents_AfterConsume(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	_, _ = db.ConsumeWakeEvents()
	count, err := db.ConsumeWakeEvents()
	if err != nil {
		t.Fatalf("ConsumeWakeEvents() err = %v", err)
	}
	if count != 0 {
		t.Errorf("ConsumeWakeEvents() count = %d, want 0", count)
	}
}

func TestSetKV_GetKV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	err = db.SetKV("x", "y")
	if err != nil {
		t.Fatalf("SetKV() err = %v", err)
	}
	val, ok, err := db.GetKV("x")
	if err != nil {
		t.Fatalf("GetKV() err = %v", err)
	}
	if !ok || val != "y" {
		t.Errorf("GetKV(x) = %q, %v, want y, true", val, ok)
	}
}

func TestGetKV_Missing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	val, ok, err := db.GetKV("nonexistent")
	if err != nil {
		t.Fatalf("GetKV() err = %v", err)
	}
	if ok || val != "" {
		t.Errorf("GetKV(nonexistent) = %q, %v, want '', false", val, ok)
	}
}

func TestClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() err = %v", err)
	}
}

func TestSchemaTablesExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	tables := []string{"turns", "kv", "wake_events", "policy_decisions", "schema_version", "installed_tools"}
	for _, tbl := range tables {
		var n int
		err := db.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&n)
		if err != nil {
			t.Fatalf("check table %s: %v", tbl, err)
		}
		if n != 1 {
			t.Errorf("table %s not found", tbl)
		}
	}
}

func TestInstallTool_GetInstalledTools_RemoveTool(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	tools, ok := db.GetInstalledTools()
	if !ok {
		t.Fatal("GetInstalledTools() ok = false")
	}
	if len(tools) != 0 {
		t.Errorf("GetInstalledTools() = %d tools, want 0", len(tools))
	}

	err = db.InstallTool(InstalledTool{
		ID:         "test-1",
		Name:       "my_tool",
		Type:       "custom",
		Config:     `{"command":"echo"}`,
		InstalledAt: "",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("InstallTool() err = %v", err)
	}

	tools, ok = db.GetInstalledTools()
	if !ok {
		t.Fatal("GetInstalledTools() ok = false")
	}
	if len(tools) != 1 {
		t.Errorf("GetInstalledTools() = %d tools, want 1", len(tools))
	}
	if tools[0].Name != "my_tool" || tools[0].Type != "custom" {
		t.Errorf("GetInstalledTools()[0] = %+v", tools[0])
	}

	err = db.RemoveTool("test-1")
	if err != nil {
		t.Fatalf("RemoveTool() err = %v", err)
	}

	tools, ok = db.GetInstalledTools()
	if !ok {
		t.Fatal("GetInstalledTools() ok = false")
	}
	if len(tools) != 0 {
		t.Errorf("GetInstalledTools() after RemoveTool = %d tools, want 0", len(tools))
	}
}
