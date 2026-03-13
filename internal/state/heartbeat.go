package state

import (
	"database/sql"
	"time"
)

// HeartbeatScheduleRow represents a row in heartbeat_schedule (TS-aligned).
type HeartbeatScheduleRow struct {
	Name        string
	Schedule    string // cron expression
	Task        string
	Enabled     int
	TierMinimum string
	LastRun     string
	NextRun     string
	LeaseUntil  string
	LeaseOwner  string
}

// HeartbeatScheduleStore provides DB access for the scheduler.
type HeartbeatScheduleStore interface {
	GetHeartbeatSchedule() ([]HeartbeatScheduleRow, error)
	UpsertHeartbeatSchedule(row HeartbeatScheduleRow) error
	UpdateHeartbeatSchedule(name, lastRun, leaseUntil, leaseOwner string) error
	AcquireTaskLease(name, owner string, ttlMs int) (bool, error)
	ReleaseTaskLease(name, owner string) error
	ClearExpiredLeases() (int, error)
	InsertHeartbeatHistory(id, taskName, startedAt, finishedAt, success int, result string, shouldWake int) error
}

// GetHeartbeatSchedule returns all schedule rows (TS getHeartbeatSchedule-aligned).
func (d *Database) GetHeartbeatSchedule() ([]HeartbeatScheduleRow, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		`SELECT name, schedule, task, enabled, COALESCE(tier_minimum,'dead'), last_run, next_run, lease_until, COALESCE(lease_owner,'')
		 FROM heartbeat_schedule ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HeartbeatScheduleRow
	for rows.Next() {
		var r HeartbeatScheduleRow
		var lastRun, nextRun, leaseUntil, leaseOwner sql.NullString
		if err := rows.Scan(&r.Name, &r.Schedule, &r.Task, &r.Enabled, &r.TierMinimum, &lastRun, &nextRun, &leaseUntil, &leaseOwner); err != nil {
			return nil, err
		}
		r.LastRun = lastRun.String
		r.NextRun = nextRun.String
		r.LeaseUntil = leaseUntil.String
		r.LeaseOwner = leaseOwner.String
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpsertHeartbeatSchedule inserts or replaces a schedule row (TS upsertHeartbeatSchedule-aligned).
func (d *Database) UpsertHeartbeatSchedule(row HeartbeatScheduleRow) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	tierMin := row.TierMinimum
	if tierMin == "" {
		tierMin = "dead"
	}
	_, err := d.db.Exec(
		`INSERT OR REPLACE INTO heartbeat_schedule (name, schedule, task, enabled, tier_minimum, last_run, next_run, lease_until, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		row.Name, row.Schedule, row.Task, row.Enabled, tierMin, nullStr(row.LastRun), nullStr(row.NextRun), nullStr(row.LeaseUntil),
	)
	return err
}

// UpdateHeartbeatSchedule updates last_run and optionally lease fields.
func (d *Database) UpdateHeartbeatSchedule(name, lastRun, leaseUntil, leaseOwner string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		`UPDATE heartbeat_schedule SET last_run=?, next_run=NULL, lease_until=?, lease_owner=?, updated_at=datetime('now') WHERE name=?`,
		nullStr(lastRun), nullStr(leaseUntil), nullStr(leaseOwner), name,
	)
	return err
}

// AcquireTaskLease acquires a lease for a task (TS acquireTaskLease-aligned).
// Returns true if acquired.
func (d *Database) AcquireTaskLease(name, owner string, ttlMs int) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	expires := time.Now().Add(time.Duration(ttlMs) * time.Millisecond).UTC().Format(time.RFC3339)
	res, err := d.db.Exec(
		`UPDATE heartbeat_schedule SET lease_owner=?, lease_until=?, updated_at=datetime('now')
		 WHERE name=? AND (lease_owner='' OR lease_until < datetime('now'))`,
		owner, expires, name,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ReleaseTaskLease releases a lease (TS releaseTaskLease-aligned).
func (d *Database) ReleaseTaskLease(name, owner string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		`UPDATE heartbeat_schedule SET lease_owner='', lease_until=NULL, updated_at=datetime('now')
		 WHERE name=? AND lease_owner=?`,
		name, owner,
	)
	return err
}

// ClearExpiredLeases clears all expired leases (TS clearExpiredLeases-aligned).
func (d *Database) ClearExpiredLeases() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.db.Exec(
		`UPDATE heartbeat_schedule SET lease_owner='', lease_until=NULL, updated_at=datetime('now')
		 WHERE lease_until IS NOT NULL AND lease_until != '' AND lease_until < datetime('now')`,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// InsertHeartbeatHistory inserts a history record (TS insertHeartbeatHistory-aligned).
func (d *Database) InsertHeartbeatHistory(id, taskName, startedAt, finishedAt string, success int, result string, shouldWake int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		`INSERT INTO heartbeat_history (id, task_name, started_at, finished_at, success, result, should_wake)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, taskName, startedAt, finishedAt, success, result, shouldWake,
	)
	return err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// migrateHeartbeatScheduleLease adds lease_owner if missing.
func (d *Database) migrateHeartbeatScheduleLease() error {
	var has int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('heartbeat_schedule') WHERE name='lease_owner'",
	).Scan(&has)
	if err != nil || has > 0 {
		return nil
	}
	_, err = d.db.Exec("ALTER TABLE heartbeat_schedule ADD COLUMN lease_owner TEXT DEFAULT ''")
	return err
}
