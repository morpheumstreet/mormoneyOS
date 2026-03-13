package state

import (
	"path/filepath"
	"testing"
)

func TestGetHeartbeatSchedule_Empty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	rows, err := db.GetHeartbeatSchedule()
	if err != nil {
		t.Fatalf("GetHeartbeatSchedule() err = %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("GetHeartbeatSchedule() len = %d, want 0", len(rows))
	}
}

func TestUpsertHeartbeatSchedule_GetHeartbeatSchedule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	err = db.UpsertHeartbeatSchedule(HeartbeatScheduleRow{
		Name: "heartbeat_ping", Schedule: "*/15 * * * *", Task: "heartbeat_ping",
		Enabled: 1, TierMinimum: "dead",
	})
	if err != nil {
		t.Fatalf("UpsertHeartbeatSchedule() err = %v", err)
	}

	rows, err := db.GetHeartbeatSchedule()
	if err != nil {
		t.Fatalf("GetHeartbeatSchedule() err = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("GetHeartbeatSchedule() len = %d, want 1", len(rows))
	}
	if rows[0].Name != "heartbeat_ping" || rows[0].Schedule != "*/15 * * * *" {
		t.Errorf("row = %+v", rows[0])
	}
}

func TestAcquireTaskLease_ReleaseTaskLease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() err = %v", err)
	}
	defer db.Close()

	_ = db.UpsertHeartbeatSchedule(HeartbeatScheduleRow{
		Name: "test_task", Schedule: "* * * * *", Task: "test_task", Enabled: 1, TierMinimum: "dead",
	})

	ok, err := db.AcquireTaskLease("test_task", "owner1", 60000)
	if err != nil {
		t.Fatalf("AcquireTaskLease() err = %v", err)
	}
	if !ok {
		t.Error("AcquireTaskLease() = false, want true")
	}

	ok2, _ := db.AcquireTaskLease("test_task", "owner2", 60000)
	if ok2 {
		t.Error("second AcquireTaskLease() = true, want false (lease held)")
	}

	_ = db.ReleaseTaskLease("test_task", "owner1")
	ok3, _ := db.AcquireTaskLease("test_task", "owner2", 60000)
	if !ok3 {
		t.Error("AcquireTaskLease() after release = false, want true")
	}
}
