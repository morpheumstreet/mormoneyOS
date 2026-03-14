package state

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Database wraps SQLite for automaton state.
type Database struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// Open opens or creates the database and applies migrations.
func Open(path string) (*Database, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	d := &Database{db: db, path: path}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

func (d *Database) migrate() error {
	if _, err := d.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("set WAL: %w", err)
	}
	if _, err := d.db.Exec(SchemaV1); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	if err := d.migrateWakeEventsIfNeeded(); err != nil {
		return fmt.Errorf("migrate wake_events: %w", err)
	}
	if err := d.migrateHeartbeatScheduleLease(); err != nil {
		return fmt.Errorf("migrate heartbeat_schedule: %w", err)
	}
	if err := d.migrateChildrenChain(); err != nil {
		return fmt.Errorf("migrate children chain: %w", err)
	}
	if err := d.migrateOnchainTransactions(); err != nil {
		return fmt.Errorf("migrate onchain_transactions: %w", err)
	}
	if err := d.migrateRegistry(); err != nil {
		return fmt.Errorf("migrate registry: %w", err)
	}
	if err := d.migrateChildLifecycleEvents(); err != nil {
		return fmt.Errorf("migrate child_lifecycle_events: %w", err)
	}
	if err := d.migrateTransactions(); err != nil {
		return fmt.Errorf("migrate transactions: %w", err)
	}
	if err := d.migrateInboxMessages(); err != nil {
		return fmt.Errorf("migrate inbox_messages: %w", err)
	}
	if err := d.migrateMetricSnapshots(); err != nil {
		return fmt.Errorf("migrate metric_snapshots: %w", err)
	}
	if err := d.migrateMemory5Tier(); err != nil {
		return fmt.Errorf("migrate memory_5tier: %w", err)
	}
	_, err := d.db.Exec("INSERT OR IGNORE INTO schema_version (version) VALUES (?)", schemaVersion)
	return err
}

// migrateWakeEventsIfNeeded migrates wake_events from old schema (id TEXT, consumed INT)
// to TS-aligned schema (id AUTOINCREMENT, consumed_at TEXT, payload).
func (d *Database) migrateWakeEventsIfNeeded() error {
	var hasConsumed int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('wake_events') WHERE name='consumed'",
	).Scan(&hasConsumed)
	if err != nil || hasConsumed == 0 {
		return nil // already new schema or table empty
	}
	_, err = d.db.Exec(`
		CREATE TABLE wake_events_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			reason TEXT NOT NULL,
			payload TEXT DEFAULT '{}',
			consumed_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		INSERT INTO wake_events_new (source, reason, payload, consumed_at, created_at)
		SELECT source, reason, '{}',
			CASE WHEN consumed = 1 THEN created_at ELSE NULL END,
			created_at
		FROM wake_events;
		DROP TABLE wake_events;
		ALTER TABLE wake_events_new RENAME TO wake_events;
		CREATE INDEX IF NOT EXISTS idx_wake_events_unconsumed ON wake_events(created_at) WHERE consumed_at IS NULL;
	`)
	return err
}

// migrateChildrenChain adds chain column to children if missing (multi-chain).
func (d *Database) migrateChildrenChain() error {
	var has int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('children') WHERE name='chain'",
	).Scan(&has)
	if err != nil || has > 0 {
		return nil
	}
	_, err = d.db.Exec("ALTER TABLE children ADD COLUMN chain TEXT NOT NULL DEFAULT 'eip155:8453'")
	return err
}

// migrateOnchainTransactions creates onchain_transactions table if missing.
func (d *Database) migrateOnchainTransactions() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS onchain_transactions (
			id TEXT PRIMARY KEY,
			chain TEXT NOT NULL,
			tx_hash TEXT,
			from_address TEXT NOT NULL,
			to_address TEXT,
			amount_cents INTEGER,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_onchain_transactions_chain ON onchain_transactions(chain);
		CREATE INDEX IF NOT EXISTS idx_onchain_transactions_created ON onchain_transactions(created_at);
	`)
	return err
}

// migrateRegistry creates registry table if missing.
func (d *Database) migrateRegistry() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS registry (
			id TEXT PRIMARY KEY,
			chain TEXT NOT NULL DEFAULT 'eip155:8453',
			address TEXT NOT NULL,
			sandbox_id TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	return err
}

// migrateChildLifecycleEvents creates child_lifecycle_events table (TS-aligned, replication audit).
func (d *Database) migrateChildLifecycleEvents() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS child_lifecycle_events (
			id TEXT PRIMARY KEY,
			child_id TEXT NOT NULL,
			from_state TEXT NOT NULL,
			to_state TEXT NOT NULL,
			reason TEXT,
			metadata TEXT DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_child_events_child_created ON child_lifecycle_events(child_id, created_at);
	`)
	return err
}

// migrateTransactions creates transactions table (TS-aligned, application-level financial log).
// Types: transfer_out, transfer_in, credit_purchase, topup, x402_payment, inference.
func (d *Database) migrateTransactions() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			amount_cents INTEGER,
			balance_after_cents INTEGER,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
		CREATE INDEX IF NOT EXISTS idx_transactions_created ON transactions(created_at);
	`)
	return err
}

// migrateInboxMessages creates inbox_messages table (TS-aligned, social inbox for soul reflection).
func (d *Database) migrateInboxMessages() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS inbox_messages (
			id TEXT PRIMARY KEY,
			from_address TEXT NOT NULL,
			content TEXT NOT NULL,
			received_at TEXT NOT NULL DEFAULT (datetime('now')),
			processed_at TEXT,
			reply_to TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_inbox_unprocessed ON inbox_messages(received_at) WHERE processed_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_inbox_received ON inbox_messages(received_at);
	`)
	return err
}

// migrateMetricSnapshots creates metric_snapshots table (TS-aligned, report_metrics).
func (d *Database) migrateMetricSnapshots() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS metric_snapshots (
			id TEXT PRIMARY KEY,
			snapshot_at TEXT NOT NULL,
			metrics_json TEXT NOT NULL DEFAULT '{}',
			alerts_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_metric_snapshots_at ON metric_snapshots(snapshot_at);
	`)
	return err
}

// migrateMemory5Tier creates 5-tier memory tables (TS-aligned, memory-system-5-tier.md).
func (d *Database) migrateMemory5Tier() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS working_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			content TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'note',
			priority INTEGER NOT NULL DEFAULT 0,
			token_count INTEGER NOT NULL DEFAULT 0,
			expires_at TEXT,
			source_turn TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_working_memory_session ON working_memory(session_id);
		CREATE INDEX IF NOT EXISTS idx_working_memory_expires ON working_memory(expires_at) WHERE expires_at IS NOT NULL;

		CREATE TABLE IF NOT EXISTS episodic_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			event_type TEXT NOT NULL DEFAULT 'event',
			summary TEXT NOT NULL,
			detail TEXT,
			outcome TEXT,
			importance REAL NOT NULL DEFAULT 0,
			embedding_key TEXT,
			classification TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_episodic_memory_session ON episodic_memory(session_id);
		CREATE INDEX IF NOT EXISTS idx_episodic_memory_importance ON episodic_memory(importance DESC);

		CREATE TABLE IF NOT EXISTS semantic_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			confidence REAL NOT NULL DEFAULT 1,
			source TEXT,
			embedding_key TEXT,
			last_verified_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(category, key)
		);
		CREATE INDEX IF NOT EXISTS idx_semantic_memory_category ON semantic_memory(category);

		CREATE TABLE IF NOT EXISTS procedural_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			steps TEXT NOT NULL,
			success_count INTEGER NOT NULL DEFAULT 0,
			failure_count INTEGER NOT NULL DEFAULT 0,
			last_used_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS relationship_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_address TEXT NOT NULL UNIQUE,
			entity_name TEXT,
			relationship_type TEXT,
			trust_score REAL NOT NULL DEFAULT 0.5,
			interaction_count INTEGER NOT NULL DEFAULT 0,
			last_interaction_at TEXT,
			notes TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_relationship_trust ON relationship_memory(trust_score DESC);
	`)
	return err
}

// MetricsInsertSnapshot inserts a metrics snapshot (TS metricsInsertSnapshot-aligned).
func (d *Database) MetricsInsertSnapshot(id, snapshotAt, metricsJSON, alertsJSON string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if metricsJSON == "" {
		metricsJSON = "{}"
	}
	if alertsJSON == "" {
		alertsJSON = "[]"
	}
	_, err := d.db.Exec(
		`INSERT INTO metric_snapshots (id, snapshot_at, metrics_json, alerts_json, created_at)
		 VALUES (?, ?, ?, ?, datetime('now'))`,
		id, snapshotAt, metricsJSON, alertsJSON,
	)
	return err
}

// MetricsGetRecent returns the most recent metric snapshots (for reports).
func (d *Database) MetricsGetRecent(limit int) ([]struct {
	ID          string
	SnapshotAt  string
	MetricsJSON string
	AlertsJSON  string
}, error) {
	if limit <= 0 {
		limit = 20
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.db.Query(
		"SELECT id, snapshot_at, metrics_json, alerts_json FROM metric_snapshots ORDER BY snapshot_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID          string
		SnapshotAt  string
		MetricsJSON string
		AlertsJSON  string
	}
	for rows.Next() {
		var r struct {
			ID          string
			SnapshotAt  string
			MetricsJSON string
			AlertsJSON  string
		}
		if err := rows.Scan(&r.ID, &r.SnapshotAt, &r.MetricsJSON, &r.AlertsJSON); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MetricsPruneOld deletes metric_snapshots older than n days (TS metricsPruneOld-aligned).
func (d *Database) MetricsPruneOld(days int) (int64, error) {
	if days <= 0 {
		days = 7
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.db.Exec(
		"DELETE FROM metric_snapshots WHERE snapshot_at < datetime('now', ?)",
		fmt.Sprintf("-%d days", days),
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// Close closes the database.
func (d *Database) Close() error {
	return d.db.Close()
}

// InsertWakeEvent inserts a wake event (TS-aligned: id AUTOINCREMENT, consumed_at NULL).
func (d *Database) InsertWakeEvent(source, reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		"INSERT INTO wake_events (source, reason, payload) VALUES (?, ?, '{}')",
		source, reason,
	)
	return err
}

// InsertWakeEventWithPayload inserts a wake event with optional JSON payload.
func (d *Database) InsertWakeEventWithPayload(source, reason, payload string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if payload == "" {
		payload = "{}"
	}
	_, err := d.db.Exec(
		"INSERT INTO wake_events (source, reason, payload) VALUES (?, ?, ?)",
		source, reason, payload,
	)
	return err
}

// ConsumeWakeEvents marks all unconsumed wake events as consumed (TS-aligned).
func (d *Database) ConsumeWakeEvents() (count int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.db.Exec("UPDATE wake_events SET consumed_at = datetime('now') WHERE consumed_at IS NULL")
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// HasUnconsumedWakeEvents returns true if any unconsumed wake events exist.
func (d *Database) HasUnconsumedWakeEvents() (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var n int
	err := d.db.QueryRow("SELECT COUNT(*) FROM wake_events WHERE consumed_at IS NULL").Scan(&n)
	return n > 0, err
}

// SetKV sets a key-value pair.
func (d *Database) SetKV(key, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		"INSERT OR REPLACE INTO kv (key, value, updated_at) VALUES (?, ?, datetime('now'))",
		key, value,
	)
	return err
}

// GetKV returns a value by key.
func (d *Database) GetKV(key string) (string, bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var value string
	err := d.db.QueryRow("SELECT value FROM kv WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return value, err == nil, err
}

// ListKeysWithPrefix returns keys matching the prefix (e.g. "procedure:").
// Used for procedure enumeration in memory retrieval.
func (d *Database) ListKeysWithPrefix(prefix string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	pattern := prefix + "%"
	rows, err := d.db.Query("SELECT key FROM kv WHERE key LIKE ?", pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// DeleteKV removes a key from kv.
func (d *Database) DeleteKV(key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("DELETE FROM kv WHERE key = ?", key)
	return err
}

// GetRecentToolNames returns distinct tool names from recent tool_calls (soul reflection evidence).
func (d *Database) GetRecentToolNames() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		"SELECT DISTINCT name FROM tool_calls ORDER BY created_at DESC LIMIT 50",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// InboxMessage represents a claimed inbox message (TS claimInboxMessages-aligned).
type InboxMessage struct {
	ID          string
	FromAddress string
	Content     string
}

// ClaimInboxMessages selects up to limit unprocessed inbox messages for agent consumption.
// TS-aligned: when no pendingInput, agent claims messages and uses them as pendingInput.
// Uses processed_at IS NULL for simplicity (single-process); no status/retry columns needed.
func (d *Database) ClaimInboxMessages(limit int) ([]InboxMessage, error) {
	if limit <= 0 {
		limit = 10
	}
	d.mu.RLock()
	rows, err := d.db.Query(
		"SELECT id, from_address, content FROM inbox_messages WHERE processed_at IS NULL ORDER BY received_at ASC LIMIT ?",
		limit,
	)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InboxMessage
	for rows.Next() {
		var m InboxMessage
		if err := rows.Scan(&m.ID, &m.FromAddress, &m.Content); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkInboxProcessed marks inbox messages as processed (TS markInboxProcessed-aligned).
// Call after turn persistence when claimed messages were used as pendingInput.
func (d *Database) MarkInboxProcessed(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, id := range ids {
		_, err := d.db.Exec(
			"UPDATE inbox_messages SET processed_at = datetime('now') WHERE id = ?",
			id,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertInboxMessage inserts a message into inbox_messages (TS insertInboxMessage-aligned).
// Used by check_social_inbox when polling channels. INSERT OR IGNORE for deduplication.
func (d *Database) InsertInboxMessage(id, fromAddress, content, receivedAt string) error {
	if receivedAt == "" {
		receivedAt = time.Now().UTC().Format(time.RFC3339)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		`INSERT OR IGNORE INTO inbox_messages (id, from_address, content, received_at) VALUES (?, ?, ?, ?)`,
		id, fromAddress, content, receivedAt,
	)
	return err
}

// GetRecentInboxAddresses returns from_address from recent inbox_messages (soul reflection evidence).
func (d *Database) GetRecentInboxAddresses() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		"SELECT from_address FROM inbox_messages ORDER BY received_at DESC LIMIT 20",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := make(map[string]bool)
	var out []string
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, err
		}
		if !seen[addr] {
			seen[addr] = true
			out = append(out, addr)
		}
	}
	return out, rows.Err()
}

// GetRecentTransactionDescriptions returns "type: description" from recent transactions (soul reflection evidence).
func (d *Database) GetRecentTransactionDescriptions() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		"SELECT type, description FROM transactions ORDER BY created_at DESC LIMIT 20",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var typ, desc string
		if err := rows.Scan(&typ, &desc); err != nil {
			return nil, err
		}
		out = append(out, typ+": "+desc)
	}
	return out, rows.Err()
}

// GetIdentity returns a value from the identity table.
func (d *Database) GetIdentity(key string) (string, bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var value string
	err := d.db.QueryRow("SELECT value FROM identity WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return value, err == nil, err
}

// SetIdentity sets a key-value in the identity table.
func (d *Database) SetIdentity(key, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		"INSERT OR REPLACE INTO identity (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}

// GetAgentState returns agent_state from kv (TS-aligned).
func (d *Database) GetAgentState() (string, bool, error) {
	return d.GetKV("agent_state")
}

// SetAgentState sets agent_state in kv.
func (d *Database) SetAgentState(state string) error {
	return d.SetKV("agent_state", state)
}

// InsertTurn inserts a turn record (TS-aligned).
func (d *Database) InsertTurn(id, timestamp, state, input, inputSource, thinking, toolCalls, tokenUsage string, costCents int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		`INSERT INTO turns (id, timestamp, state, input, input_source, thinking, tool_calls, token_usage, cost_cents)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, timestamp, state, input, inputSource, thinking, toolCalls, tokenUsage, costCents,
	)
	return err
}

// Turn represents a persisted agent turn (TS AgentTurn-aligned).
type Turn struct {
	ID         string
	Timestamp  string
	State      string
	Input      string
	InputSource string
	Thinking   string
	ToolCalls  string // JSON array
	TokenUsage string // JSON object
	CostCents  int
}

// GetRecentTurns returns the most recent turns for context (TS-aligned).
func (d *Database) GetRecentTurns(limit int) ([]Turn, error) {
	if limit <= 0 {
		limit = 20
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		`SELECT id, timestamp, state, input, input_source, thinking, tool_calls, token_usage, cost_cents
		 FROM turns ORDER BY timestamp DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Turn
	for rows.Next() {
		var t Turn
		err := rows.Scan(&t.ID, &t.Timestamp, &t.State, &t.Input, &t.InputSource, &t.Thinking, &t.ToolCalls, &t.TokenUsage, &t.CostCents)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	// Reverse so oldest first (chronological for context)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// InsertPolicyDecision inserts a policy decision for audit and rate-limit tracking (TS-aligned).
func (d *Database) InsertPolicyDecision(id, turnID, toolName, argsHash, riskLevel, decision, reason, source string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if source == "" {
		source = "self"
	}
	_, err := d.db.Exec(
		`INSERT INTO policy_decisions (id, turn_id, tool_name, tool_args_hash, risk_level, decision, reason, source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, turnID, toolName, argsHash, riskLevel, decision, reason, source,
	)
	return err
}

// CountRecentPolicyDecisions returns the count of allow decisions for a tool within the window (TS rate-limits aligned).
func (d *Database) CountRecentPolicyDecisions(toolName string, windowMs int64) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	cutoff := time.Now().Add(-time.Duration(windowMs) * time.Millisecond)
	cutoffStr := cutoff.Format("2006-01-02 15:04:05")
	var n int
	err := d.db.QueryRow(
		`SELECT COUNT(*) FROM policy_decisions WHERE tool_name = ? AND decision = 'allow' AND created_at >= ?`,
		toolName, cutoffStr,
	).Scan(&n)
	return n, err
}

// InsertToolCall inserts a tool call result (TS-aligned).
func (d *Database) InsertToolCall(turnID, id, name, args, result string, durationMs int, errStr string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		`INSERT INTO tool_calls (id, turn_id, name, arguments, result, duration_ms, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, turnID, name, args, result, durationMs, errStr,
	)
	return err
}

// GetTurnCount returns the number of turns (TS-aligned).
func (d *Database) GetTurnCount() (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var n int64
	err := d.db.QueryRow("SELECT COUNT(*) FROM turns").Scan(&n)
	return n, err
}

// Child represents a spawned child automaton (TS-aligned).
type Child struct {
	ID                 string
	Name               string
	Address            string
	Chain              string // CAIP-2, e.g. eip155:8453
	SandboxID          string
	GenesisPrompt      string
	CreatorMessage     string
	FundedAmountCents  int64
	Status             string
	CreatedAt          string
	LastChecked        string
}

// GetChildren returns children from children table if it exists (TS-aligned).
// Filters out dead children for strategies display.
func (d *Database) GetChildren() ([]map[string]any, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='children'").Scan(&exists); err != nil || exists == 0 {
		return nil, false
	}
	rows, err := d.db.Query("SELECT id, name, address, COALESCE(chain,'eip155:8453'), sandbox_id, genesis_prompt, creator_message, funded_amount_cents, status, created_at, last_checked FROM children WHERE status != 'dead' ORDER BY created_at DESC")
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var c Child
		var creatorMsg, lastChecked sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.Chain, &c.SandboxID, &c.GenesisPrompt, &creatorMsg, &c.FundedAmountCents, &c.Status, &c.CreatedAt, &lastChecked); err != nil {
			continue
		}
		enabled := c.Status == "healthy" || c.Status == "running"
		out = append(out, map[string]any{
			"name":        c.Name,
			"description": "Child automaton (" + c.Status + ")",
			"enabled":     enabled,
			"risk_level":  "medium",
		})
	}
	return out, true
}

// GetAllChildren returns all children from children table if it exists (TS-aligned).
// Includes dead children; used by heartbeat tasks for health check and prune.
func (d *Database) GetAllChildren() ([]Child, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='children'").Scan(&exists); err != nil || exists == 0 {
		return nil, false
	}
	rows, err := d.db.Query("SELECT id, name, address, COALESCE(chain,'eip155:8453'), sandbox_id, genesis_prompt, creator_message, funded_amount_cents, status, created_at, last_checked FROM children ORDER BY created_at DESC")
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []Child
	for rows.Next() {
		var c Child
		var creatorMsg, lastChecked sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.Chain, &c.SandboxID, &c.GenesisPrompt, &creatorMsg, &c.FundedAmountCents, &c.Status, &c.CreatedAt, &lastChecked); err != nil {
			continue
		}
		if lastChecked.Valid {
			c.LastChecked = lastChecked.String
		}
		out = append(out, c)
	}
	return out, true
}

// UpdateChildStatus updates a child's status and last_checked (TS-aligned).
func (d *Database) UpdateChildStatus(id, status string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		"UPDATE children SET status = ?, last_checked = datetime('now') WHERE id = ?",
		status, id,
	)
	return err
}

// UpdateChildSandboxID updates a child's sandbox_id (used during spawn).
func (d *Database) UpdateChildSandboxID(id, sandboxID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("UPDATE children SET sandbox_id = ? WHERE id = ?", sandboxID, id)
	return err
}

// UpdateChildAddress updates a child's address (used during spawn after wallet init).
func (d *Database) UpdateChildAddress(id, address string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("UPDATE children SET address = ? WHERE id = ?", address, id)
	return err
}

// GetChildByID returns a child by ID, or nil if not found.
func (d *Database) GetChildByID(id string) (*Child, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='children'").Scan(&exists); err != nil || exists == 0 {
		return nil, false
	}
	var c Child
	var creatorMsg, lastChecked sql.NullString
	err := d.db.QueryRow(
		"SELECT id, name, address, COALESCE(chain,'eip155:8453'), sandbox_id, genesis_prompt, creator_message, funded_amount_cents, status, created_at, last_checked FROM children WHERE id = ?",
		id,
	).Scan(&c.ID, &c.Name, &c.Address, &c.Chain, &c.SandboxID, &c.GenesisPrompt, &creatorMsg, &c.FundedAmountCents, &c.Status, &c.CreatedAt, &lastChecked)
	if err != nil {
		return nil, false
	}
	if lastChecked.Valid {
		c.LastChecked = lastChecked.String
	}
	return &c, true
}

// AddChildFundedAmount adds to a child's funded_amount_cents (TS-aligned).
func (d *Database) AddChildFundedAmount(id string, amount int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		"UPDATE children SET funded_amount_cents = funded_amount_cents + ? WHERE id = ?",
		amount, id,
	)
	return err
}

// InsertSkill inserts or replaces a skill (TS-aligned).
func (d *Database) InsertSkill(name, description, instructions, source, path string, enabled bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	en := 0
	if enabled {
		en = 1
	}
	if description == "" {
		description = ""
	}
	if instructions == "" {
		instructions = ""
	}
	if source == "" {
		source = "builtin"
	}
	if path == "" {
		path = ""
	}
	_, err := d.db.Exec(
		`INSERT OR REPLACE INTO skills (name, description, auto_activate, requires, instructions, source, path, enabled, installed_at)
		 VALUES (?, ?, 1, '{}', ?, ?, ?, ?, datetime('now'))`,
		name, description, instructions, source, path, en,
	)
	return err
}

// DeleteSkill removes a skill by name (soft: sets enabled=0, or hard delete).
func (d *Database) DeleteSkill(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("DELETE FROM skills WHERE name = ?", name)
	return err
}

// InsertChild inserts a child automaton (TS-aligned).
func (d *Database) InsertChild(c Child) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	chain := c.Chain
	if chain == "" {
		chain = "eip155:8453"
	}
	_, err := d.db.Exec(
		`INSERT INTO children (id, name, address, chain, sandbox_id, genesis_prompt, creator_message, funded_amount_cents, status, created_at, last_checked)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?)`,
		c.ID, c.Name, c.Address, chain, c.SandboxID, c.GenesisPrompt, nullStr(c.CreatorMessage), c.FundedAmountCents, c.Status, nullStr(c.LastChecked),
	)
	return err
}

// InsertChildLifecycleEvent inserts a lifecycle event (TS-aligned, replication audit).
func (d *Database) InsertChildLifecycleEvent(childID, fromState, toState, reason, metadata string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	eventID := uuid.New().String()
	if metadata == "" {
		metadata = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO child_lifecycle_events (id, child_id, from_state, to_state, reason, metadata, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		eventID, childID, fromState, toState, reason, metadata,
	)
	return err
}

// GetLatestChildState returns the latest lifecycle state for a child (from child_lifecycle_events).
// Returns empty string if no events exist.
func (d *Database) GetLatestChildState(childID string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='child_lifecycle_events'").Scan(&exists); err != nil || exists == 0 {
		return "", nil
	}
	var toState string
	err := d.db.QueryRow(
		"SELECT to_state FROM child_lifecycle_events WHERE child_id = ? ORDER BY created_at DESC LIMIT 1",
		childID,
	).Scan(&toState)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return toState, err
}

// DeleteChild removes a child from DB (children + child_lifecycle_events). Used by cleanup.
func (d *Database) DeleteChild(childID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.db.Exec("DELETE FROM child_lifecycle_events WHERE child_id = ?", childID); err != nil {
		return err
	}
	_, err := d.db.Exec("DELETE FROM children WHERE id = ?", childID)
	return err
}

// SkillRow is a full row from the skills table (for subconscious loader).
type SkillRow struct {
	Name         string
	Description  string
	Instructions string
	Source       string
	Path         string
	Enabled      bool
	AutoActivate int
}

// GetSkillRows returns full skill rows for enabled skills (for subconscious prompt injection).
// Used by SkillLoader; list_skills and /api/strategies use GetSkills() instead (metadata only).
func (d *Database) GetSkillRows() ([]SkillRow, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='skills'").Scan(&exists); err != nil || exists == 0 {
		return nil, nil
	}
	rows, err := d.db.Query("SELECT name, description, instructions, source, path, enabled, auto_activate FROM skills WHERE enabled = 1 ORDER BY auto_activate DESC, name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SkillRow
	for rows.Next() {
		var r SkillRow
		var enabled int
		if err := rows.Scan(&r.Name, &r.Description, &r.Instructions, &r.Source, &r.Path, &enabled, &r.AutoActivate); err != nil {
			continue
		}
		r.Enabled = enabled == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateSkillDescription updates description for a skill (one-time sync when file loads richer desc).
func (d *Database) UpdateSkillDescription(name, description string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("UPDATE skills SET description = ? WHERE name = ?", description, name)
	return err
}

// GetSkills returns enabled skills from skills table if it exists.
func (d *Database) GetSkills() ([]map[string]any, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='skills'").Scan(&exists); err != nil || exists == 0 {
		return nil, false
	}
	rows, err := d.db.Query("SELECT name, description, enabled FROM skills WHERE enabled = 1")
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var name, desc string
		var enabled int
		if err := rows.Scan(&name, &desc, &enabled); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"name": name, "description": desc, "risk_level": "low", "enabled": enabled == 1,
		})
	}
	return out, true
}

// GetAllSkills returns all skills (enabled and disabled) with full columns.
func (d *Database) GetAllSkills() ([]SkillRow, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='skills'").Scan(&exists); err != nil || exists == 0 {
		return nil, nil
	}
	rows, err := d.db.Query("SELECT name, description, instructions, source, path, enabled, auto_activate FROM skills ORDER BY auto_activate DESC, name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SkillRow
	for rows.Next() {
		var r SkillRow
		var enabled int
		if err := rows.Scan(&r.Name, &r.Description, &r.Instructions, &r.Source, &r.Path, &enabled, &r.AutoActivate); err != nil {
			continue
		}
		r.Enabled = enabled == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetSkillByName returns a single skill by name, or nil if not found.
func (d *Database) GetSkillByName(name string) (*SkillRow, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='skills'").Scan(&exists); err != nil || exists == 0 {
		return nil, nil
	}
	var r SkillRow
	var enabled int
	err := d.db.QueryRow(
		"SELECT name, description, instructions, source, path, enabled, auto_activate FROM skills WHERE name = ?",
		name,
	).Scan(&r.Name, &r.Description, &r.Instructions, &r.Source, &r.Path, &enabled, &r.AutoActivate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = enabled == 1
	return &r, nil
}

// UpdateSkillEnabled sets the enabled flag for a skill.
func (d *Database) UpdateSkillEnabled(name string, enabled bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	en := 0
	if enabled {
		en = 1
	}
	_, err := d.db.Exec("UPDATE skills SET enabled = ? WHERE name = ?", en, name)
	return err
}

// UpdateSkill updates description and instructions for a skill.
func (d *Database) UpdateSkill(name, description, instructions string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("UPDATE skills SET description = ?, instructions = ? WHERE name = ?", description, instructions, name)
	return err
}

// InstalledTool represents a tool from installed_tools table (TS-aligned).
type InstalledTool struct {
	ID          string
	Name        string
	Type        string // "builtin", "mcp", "custom"
	Config      string // JSON
	InstalledAt string
	Enabled     bool
}

// GetInstalledTools returns enabled tools from installed_tools table (TS-aligned).
func (d *Database) GetInstalledTools() ([]InstalledTool, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='installed_tools'").Scan(&exists); err != nil || exists == 0 {
		return nil, false
	}
	rows, err := d.db.Query("SELECT id, name, type, config, installed_at, enabled FROM installed_tools WHERE enabled = 1")
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []InstalledTool
	for rows.Next() {
		var t InstalledTool
		var enabled int
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.Config, &t.InstalledAt, &enabled); err != nil {
			continue
		}
		t.Enabled = enabled == 1
		out = append(out, t)
	}
	return out, true
}

// InstallTool inserts or replaces a tool in installed_tools (TS-aligned).
func (d *Database) InstallTool(t InstalledTool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t.Config == "" {
		t.Config = "{}"
	}
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err := d.db.Exec(
		`INSERT OR REPLACE INTO installed_tools (id, name, type, config, installed_at, enabled)
		 VALUES (?, ?, ?, ?, COALESCE(?, datetime('now')), ?)`,
		t.ID, t.Name, t.Type, t.Config, t.InstalledAt, enabled,
	)
	return err
}

// RemoveTool disables a tool by id (TS-aligned: soft disable).
func (d *Database) RemoveTool(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("UPDATE installed_tools SET enabled = 0 WHERE id = ?", id)
	return err
}

// Memory5TierRow types for 5-tier memory retrieval (memory-system-5-tier.md).
type WorkingMemoryRow struct {
	ID          int64
	SessionID   string
	Content     string
	ContentType string
	Priority    int
	TokenCount  int
	ExpiresAt   string
	SourceTurn  string
}

type EpisodicMemoryRow struct {
	ID            int64
	SessionID     string
	EventType     string
	Summary       string
	Detail        string
	Outcome       string
	Importance    float64
	EmbeddingKey  string
	Classification string
}

type SemanticMemoryRow struct {
	ID             int64
	Category       string
	Key            string
	Value          string
	Confidence     float64
	Source         string
	EmbeddingKey   string
	LastVerifiedAt string
}

type ProceduralMemoryRow struct {
	ID            int64
	Name          string
	Description   string
	Steps         string
	SuccessCount  int
	FailureCount  int
	LastUsedAt    string
}

type RelationshipMemoryRow struct {
	ID                 int64
	EntityAddress      string
	EntityName         string
	RelationshipType   string
	TrustScore         float64
	InteractionCount   int
	LastInteractionAt  string
	Notes              string
}

// GetWorkingMemory returns working memory entries for session, ordered by priority desc, limit.
func (d *Database) GetWorkingMemory(sessionID string, limit int) ([]WorkingMemoryRow, error) {
	if limit <= 0 {
		limit = 20
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		`SELECT id, session_id, content, content_type, priority, token_count, expires_at, source_turn
		 FROM working_memory WHERE session_id = ? AND (expires_at IS NULL OR expires_at > datetime('now'))
		 ORDER BY priority DESC, id DESC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkingMemoryRow
	for rows.Next() {
		var r WorkingMemoryRow
		var expiresAt, sourceTurn sql.NullString
		if err := rows.Scan(&r.ID, &r.SessionID, &r.Content, &r.ContentType, &r.Priority, &r.TokenCount, &expiresAt, &sourceTurn); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			r.ExpiresAt = expiresAt.String
		}
		if sourceTurn.Valid {
			r.SourceTurn = sourceTurn.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetEpisodicMemory returns episodic memory entries for session, ordered by importance desc, limit.
func (d *Database) GetEpisodicMemory(sessionID string, limit int) ([]EpisodicMemoryRow, error) {
	if limit <= 0 {
		limit = 20
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		`SELECT id, session_id, event_type, summary, detail, outcome, importance, embedding_key, classification
		 FROM episodic_memory WHERE session_id = ? ORDER BY importance DESC, id DESC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EpisodicMemoryRow
	for rows.Next() {
		var r EpisodicMemoryRow
		var detail, outcome, emb, class sql.NullString
		if err := rows.Scan(&r.ID, &r.SessionID, &r.EventType, &r.Summary, &detail, &outcome, &r.Importance, &emb, &class); err != nil {
			return nil, err
		}
		if detail.Valid {
			r.Detail = detail.String
		}
		if outcome.Valid {
			r.Outcome = outcome.String
		}
		if emb.Valid {
			r.EmbeddingKey = emb.String
		}
		if class.Valid {
			r.Classification = class.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetSemanticMemory returns semantic memory entries, limit.
func (d *Database) GetSemanticMemory(limit int) ([]SemanticMemoryRow, error) {
	if limit <= 0 {
		limit = 50
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		`SELECT id, category, key, value, confidence, source, embedding_key, last_verified_at
		 FROM semantic_memory ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SemanticMemoryRow
	for rows.Next() {
		var r SemanticMemoryRow
		var source, emb, lastVer sql.NullString
		if err := rows.Scan(&r.ID, &r.Category, &r.Key, &r.Value, &r.Confidence, &source, &emb, &lastVer); err != nil {
			return nil, err
		}
		if source.Valid {
			r.Source = source.String
		}
		if emb.Valid {
			r.EmbeddingKey = emb.String
		}
		if lastVer.Valid {
			r.LastVerifiedAt = lastVer.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetProceduralMemory returns procedural memory entries, limit.
func (d *Database) GetProceduralMemory(limit int) ([]ProceduralMemoryRow, error) {
	if limit <= 0 {
		limit = 20
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		`SELECT id, name, description, steps, success_count, failure_count, last_used_at
		 FROM procedural_memory ORDER BY last_used_at DESC, id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProceduralMemoryRow
	for rows.Next() {
		var r ProceduralMemoryRow
		var desc, lastUsed sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &desc, &r.Steps, &r.SuccessCount, &r.FailureCount, &lastUsed); err != nil {
			return nil, err
		}
		if desc.Valid {
			r.Description = desc.String
		}
		if lastUsed.Valid {
			r.LastUsedAt = lastUsed.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRelationshipMemory returns relationship memory entries, limit.
func (d *Database) GetRelationshipMemory(limit int) ([]RelationshipMemoryRow, error) {
	if limit <= 0 {
		limit = 20
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.db.Query(
		`SELECT id, entity_address, entity_name, relationship_type, trust_score, interaction_count, last_interaction_at, notes
		 FROM relationship_memory ORDER BY trust_score DESC, interaction_count DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RelationshipMemoryRow
	for rows.Next() {
		var r RelationshipMemoryRow
		var name, relType, lastInt, notes sql.NullString
		if err := rows.Scan(&r.ID, &r.EntityAddress, &name, &relType, &r.TrustScore, &r.InteractionCount, &lastInt, &notes); err != nil {
			return nil, err
		}
		if name.Valid {
			r.EntityName = name.String
		}
		if relType.Valid {
			r.RelationshipType = relType.String
		}
		if lastInt.Valid {
			r.LastInteractionAt = lastInt.String
		}
		if notes.Valid {
			r.Notes = notes.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Has5TierMemoryTables returns true if all 5-tier memory tables exist.
func (d *Database) Has5TierMemoryTables() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var n int
	err := d.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('working_memory','episodic_memory','semantic_memory','procedural_memory','relationship_memory')`,
	).Scan(&n)
	return err == nil && n == 5
}

// GetInferenceCostSummary returns today_cost, today_calls, total_cost from inference_costs if table exists.
func (d *Database) GetInferenceCostSummary() (todayCost, todayCalls float64, totalCost float64, ok bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var exists int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='inference_costs'").Scan(&exists); err != nil || exists == 0 {
		return 0, 0, 0, false
	}
	today := time.Now().Format("2006-01-02")
	var todaySum sql.NullInt64
	var todayCnt sql.NullInt64
	_ = d.db.QueryRow(
		"SELECT COALESCE(SUM(cost_cents),0), COUNT(*) FROM inference_costs WHERE date(created_at) = ?",
		today,
	).Scan(&todaySum, &todayCnt)
	var totalSum sql.NullInt64
	_ = d.db.QueryRow("SELECT COALESCE(SUM(cost_cents),0) FROM inference_costs").Scan(&totalSum)
	tc := float64(0)
	if todaySum.Valid {
		tc = float64(todaySum.Int64) / 100
	}
	tot := float64(0)
	if totalSum.Valid {
		tot = float64(totalSum.Int64) / 100
	}
	cnt := float64(0)
	if todayCnt.Valid {
		cnt = float64(todayCnt.Int64)
	}
	return tc, cnt, tot, true
}
