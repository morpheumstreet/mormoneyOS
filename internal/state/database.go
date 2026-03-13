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

// DeleteKV removes a key from kv.
func (d *Database) DeleteKV(key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("DELETE FROM kv WHERE key = ?", key)
	return err
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
