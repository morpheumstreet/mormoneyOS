package state

const schemaVersion = 14

// SchemaV1 creates core tables per mormoneyOS design.
const SchemaV1 = `
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS identity (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS turns (
  id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,
  state TEXT NOT NULL,
  input TEXT,
  input_source TEXT,
  thinking TEXT NOT NULL,
  tool_calls TEXT NOT NULL DEFAULT '[]',
  token_usage TEXT NOT NULL DEFAULT '{}',
  cost_cents INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS tool_calls (
  id TEXT PRIMARY KEY,
  turn_id TEXT NOT NULL REFERENCES turns(id),
  name TEXT NOT NULL,
  arguments TEXT NOT NULL DEFAULT '{}',
  result TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0,
  error TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS kv (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS policy_decisions (
  id TEXT PRIMARY KEY,
  turn_id TEXT,
  tool_name TEXT NOT NULL,
  tool_args_hash TEXT NOT NULL,
  risk_level TEXT NOT NULL CHECK(risk_level IN ('safe','caution','dangerous','forbidden')),
  decision TEXT NOT NULL CHECK(decision IN ('allow','deny','quarantine')),
  reason TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT 'self',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS spend_tracking (
  id TEXT PRIMARY KEY,
  category TEXT NOT NULL,
  amount_cents INTEGER NOT NULL,
  window_start TEXT NOT NULL,
  window_end TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS heartbeat_schedule (
  name TEXT PRIMARY KEY,
  schedule TEXT NOT NULL,
  task TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  tier_minimum TEXT,
  last_run TEXT,
  next_run TEXT,
  lease_until TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS heartbeat_history (
  id TEXT PRIMARY KEY,
  task_name TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL,
  success INTEGER NOT NULL,
  result TEXT,
  should_wake INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS wake_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  reason TEXT NOT NULL,
  payload TEXT DEFAULT '{}',
  consumed_at TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS heartbeat_dedup (
  dedup_key TEXT PRIMARY KEY,
  task_name TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS inference_costs (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  turn_id TEXT,
  model TEXT NOT NULL,
  provider TEXT NOT NULL,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cost_cents INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  tier TEXT NOT NULL,
  task_type TEXT NOT NULL CHECK(task_type IN ('agent_turn','heartbeat_triage','safety_check','summarization','planning')),
  cache_hit INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS skills (
  name TEXT PRIMARY KEY,
  description TEXT NOT NULL DEFAULT '',
  auto_activate INTEGER NOT NULL DEFAULT 1,
  requires TEXT DEFAULT '{}',
  instructions TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT 'builtin',
  path TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  installed_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS children (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  address TEXT NOT NULL,
  chain TEXT NOT NULL DEFAULT 'eip155:8453',
  sandbox_id TEXT NOT NULL,
  genesis_prompt TEXT NOT NULL,
  creator_message TEXT,
  funded_amount_cents INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'spawning',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  last_checked TEXT
);

CREATE INDEX IF NOT EXISTS idx_policy_decisions_tool ON policy_decisions(tool_name);
CREATE INDEX IF NOT EXISTS idx_policy_decisions_created ON policy_decisions(created_at);

CREATE INDEX IF NOT EXISTS idx_turns_timestamp ON turns(timestamp);
CREATE INDEX IF NOT EXISTS idx_turns_state ON turns(state);
CREATE INDEX IF NOT EXISTS idx_tool_calls_turn ON tool_calls(turn_id);
CREATE INDEX IF NOT EXISTS idx_wake_events_unconsumed ON wake_events(created_at) WHERE consumed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_dedup_expires ON heartbeat_dedup(expires_at);
CREATE INDEX IF NOT EXISTS idx_inf_created ON inference_costs(created_at);
CREATE INDEX IF NOT EXISTS idx_children_status ON children(status);

-- Onchain transactions (TS-aligned): chain required for multi-chain.
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

-- Registry (TS-aligned): parent automaton registry with chain.
CREATE TABLE IF NOT EXISTS registry (
  id TEXT PRIMARY KEY,
  chain TEXT NOT NULL DEFAULT 'eip155:8453',
  address TEXT NOT NULL,
  sandbox_id TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Installed tools (TS-aligned): id, name, type, config, installed_at, enabled
CREATE TABLE IF NOT EXISTS installed_tools (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  config TEXT DEFAULT '{}',
  installed_at TEXT NOT NULL DEFAULT (datetime('now')),
  enabled INTEGER NOT NULL DEFAULT 1
);
`
