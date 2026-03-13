# Child Runtime / Spawn Protocol

**Date:** 2026-03-13  
**Purpose:** Protocol and API to deploy, start, message, and verify child agents. Unblocks spawn_child, start_child, message_child, verify_child_constitution.

---

## 1. Overview

The child runtime protocol enables a parent automaton to spawn and manage child agents in Conway sandboxes. It requires:

1. **Conway create_sandbox** (or equivalent) for compute
2. **Sandbox-scoped operations** (exec, writeFile, readFile) for child provisioning
3. **Child runtime** that supports start, message, and constitution verification

---

## 2. Conway Client Extensions

The Conway HTTP client now supports sandbox-scoped operations:

| Method | API | Purpose |
|--------|-----|---------|
| `ExecInSandbox(ctx, sandboxID, command, timeoutMs)` | POST /v1/sandboxes/{id}/exec | Run commands in a child sandbox |
| `WriteFileInSandbox(ctx, sandboxID, path, content)` | POST /v1/sandboxes/{id}/files/upload/json | Write files in a child sandbox |
| `ReadFileInSandbox(ctx, sandboxID, path)` | GET /v1/sandboxes/{id}/files/read | Read files from a child sandbox |

These are used by spawn_child (provisioning), start_child (exec), and verify_child_constitution (readFile).

---

## 3. Tools Implemented

### 3.1 spawn_child

- **Requires:** Conway, ChildSpawnStore, ParentAddress, GenesisPrompt
- **Flow:** Create sandbox → install runtime (apt, npm) → write genesis.json → init wallet → update child record
- **Output:** Child in `wallet_verified` status, ready for funding and start

### 3.2 start_child

- **Requires:** Conway, Store with GetChildByID, UpdateChildStatus
- **Flow:** Transition to `starting` → exec `automaton --init && automaton --provision && systemctl start automaton` → transition to `healthy`

### 3.3 message_child

- **Requires:** SocialClient (optional), Store with GetChildByID
- **When SocialClient nil:** Returns "Social relay not configured. Set socialRelayUrl in config."
- **Flow:** Look up child address → send via SocialClient.Send(childAddress, payload)

### 3.4 verify_child_constitution

- **Requires:** Conway, Store with GetChildByID, GetKV
- **Flow:** Get stored hash from KV (`constitution_hash:{sandboxID}`) → read constitution from child sandbox → SHA-256 compare

---

## 4. Database Extensions

- `UpdateChildSandboxID(id, sandboxID)` — set sandbox_id after CreateSandbox
- `UpdateChildAddress(id, address)` — set address after wallet init

---

## 5. RegistryOptions Extensions

- `ParentAddress` — wallet address for spawn_child genesis
- `GenesisPrompt` — parent's genesis prompt for child lineage
- `SocialClient` — optional; when nil, message_child returns stub message

---

## 6. Dependencies

| Tool | Conway | Store | SocialClient |
|------|--------|-------|--------------|
| spawn_child | ✅ | ChildSpawnStore | — |
| start_child | ✅ | GetChildByID, UpdateChildStatus | — |
| message_child | — | GetChildByID | optional |
| verify_child_constitution | ✅ | GetChildByID, GetKV | — |

---

## 7. Constitution Propagation

Constitution propagation (propagateConstitution) is stubbed: the tool context does not have access to the local `~/.automaton/constitution.md` path. When a config-driven constitution path is added, propagateConstitution can be implemented to:

1. Read local constitution
2. Compute SHA-256 hash
3. Write to child sandbox
4. Store hash in KV for verify_child_constitution

---

## 8. Replication (Lifecycle, Health, Cleanup)

Go implements TS-aligned replication in `internal/replication/`:

| Component | Purpose |
|-----------|---------|
| `child_lifecycle_events` | Audit table for state transitions (migration in schema v10) |
| `ChildLifecycle` | State machine with ValidTransitions; InsertChildLifecycleEvent, GetLatestChildState |
| `ChildHealthMonitor` | Conway exec `automaton --status`; JSON health parsing; concurrency limit |
| `SandboxCleanup` | Conway DeleteSandbox + DeleteChild for dead/failed/cleaned_up |
| `GetLineageSummary` | Short text for system prompt (Children: name (status); ...) |

**Heartbeat wiring:** When Conway is configured, `runCheckChildHealth` uses `ChildHealthMonitor.Check()`; `runPruneDeadChildren` uses `PruneDeadChildren` (mark stale dead) + `SandboxCleanup.PruneDead` (delete sandboxes, remove from DB).

**System prompt:** `BuildSystemPrompt` accepts optional `lineageSummary`; LoopOptions.LineageStore provides `GetLineageSummary(store)`.

---

## 9. References

- [ts-go-alignment.md](./ts-go-alignment.md) — tool parity matrix
- TS: `src/replication/spawn.ts`, `lifecycle.ts`, `constitution.ts`, `messaging.ts`, `health.ts`, `cleanup.ts`
- Go: `internal/tools/child_runtime.go`, `internal/conway/http.go`, `internal/replication/`
