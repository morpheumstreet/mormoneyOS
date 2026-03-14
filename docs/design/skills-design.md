# Skills System Design for mormoneyOS

**Date:** 2026-03-13  
**Purpose:** Optimal skills design for mormoneyOS — packageable. References mormclaw (ZeroClaw) and [OpenClaw](https://github.com/openclaw/openclaw).

---

## 1. Executive Summary

mormoneyOS currently has a **DB-only skills model**: `skills` table, `install_skill` / `create_skill` / `remove_skill` / `list_skills` tools, and `/api/strategies` merging skills + children. Skills are **not** injected into the system prompt. This design proposes a layered, extensible skills architecture that:

- Keeps DB as the source of truth for **enabled** skills (strategy selection)
- Adds **file-based skill packages** (SKILL.md / SKILL.toml) for rich content
- Introduces the **subconscious** — the runtime merge layer where Subconscious synthesizes file + DB into a unified skill representation
- Unifies **skills** and **strategies** under one contract
- Aligns with mormclaw (ZeroClaw) and OpenClaw patterns without over-engineering

---

## 2. Reference Models

### 2.1 mormclaw (ZeroClaw)

| Aspect | Design |
|--------|--------|
| **Location** | `~/.zeroclaw/workspace/skills/<name>/` |
| **Format** | `SKILL.toml` (preferred) or `SKILL.md` (frontmatter) |
| **Skill struct** | `name`, `description`, `version`, `author`, `tags`, `tools[]`, `prompts[]`, `location`, `always` |
| **SkillTool** | `name`, `description`, `kind` (shell/http/script), `command`, `args` |
| **Loading** | Workspace skills + open-skills repo + `trusted_skill_roots` |
| **Tool bridge** | `SkillToolHandler` — parses `{placeholder}` in commands, generates JSON schema, executes shell |
| **SkillForge** | Scout (GitHub, ClawHub) → Evaluate → Integrate; writes SKILL.toml + SKILL.md |
| **Templates** | Rust, TypeScript, Go, Python templates for `create_skill` |
| **Security** | `audit_skill_directory_with_options`, symlink trust, `allow_scripts` |

### 2.2 OpenClaw

| Aspect | Design |
|--------|--------|
| **Location** | `/skills` (workspace) > `~/.openclaw/skills` > bundled |
| **Format** | AgentSkills-compatible `SKILL.md` with YAML frontmatter |
| **Precedence** | Workspace > managed > bundled |
| **ClawHub** | Registry: sync, install, update skills |
| **Gating** | `metadata.openclaw.requires.bins`, `requires.env`, `requires.config` |
| **Config** | `skills.entries.<name>.enabled`, `apiKey`, `env` |
| **Plugins** | `openclaw.plugin.json` lists skill dirs |

---

## 3. Current mormoneyOS State

### 3.1 Schema (skills table)

```sql
CREATE TABLE skills (
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
```

### 3.2 Tools

- **list_skills** — reads `GetSkills()` from Store
- **install_skill** — `InsertSkill(name, desc, "", "installed", path, true)` — path **required** (file-based); validate dir + SKILL.md/SKILL.toml + trusted roots
- **create_skill** — `InsertSkill(name, desc, instructions, "builtin", "", true)` — DB-only (builtin)
- **remove_skill** — `DeleteSkill(name)` (hard delete)

### 3.3 Gaps

1. **No prompt injection** — `instructions` and skill content never reach the agent
2. **install_skill path unused** — path is stored but no file loading
3. **No file-based packages** — no SKILL.md / SKILL.toml support
4. **No gating** — `requires` column exists but unused
5. **CLI strategies** — placeholder; does not use DB
6. **No discovery** — no registry, no SkillForge-style pipeline

---

## 4. Proposed Architecture (Clean, DRY, SOLID)

### 4.1 Layered Model

```
┌─────────────────────────────────────────────────────────────────┐
│  Layer 1: Strategy Selection (DB)                               │
│  skills table: name, description, enabled, path, instructions     │
│  Source of truth for "what is active"                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Layer 2: Subconscious (file + DB merge)                       │
│  Runtime merge: file content + DB row → unified Skill            │
│  - If path set: load SKILL.md/SKILL.toml from path, merge with DB│
│  - Else: use DB instructions only                               │
│  Single LoadSkill(dbRow) → *Skill                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Layer 3: Skill Consumer                                        │
│  - Prompt builder: inject skill instructions into system prompt  │
│  - API/CLI: list strategies (skills + children)                 │
│  - Tools: list_skills, install_skill, create_skill, remove_skill│
└─────────────────────────────────────────────────────────────────┘
```

**Subconscious:** The merge layer that runs at runtime. It reads from both the filesystem (when `path` is set) and the DB, synthesizing them into a single `Skill` before the agent consumes it. File content can evolve independently; the subconscious always serves the freshest merged result.

### 4.2 Single Responsibility

| Component | Responsibility |
|-----------|----------------|
| **SkillStore** | CRUD for skills table (Insert, Delete, GetSkills) |
| **Subconscious** | Runtime merge: load from path + DB; synthesize into `Skill` |
| **SkillFormatter** | Format skills for prompt injection |
| **SkillTools** | Agent tools (install, create, remove, list) |

### 4.3 Interface Segregation

```go
// SkillStore — DB contract
type SkillStore interface {
    InsertSkill(name, description, instructions, source, path string, enabled bool) error
    DeleteSkill(name string) error
    GetSkills() ([]map[string]any, bool)   // metadata only
    GetSkillRows() ([]SkillRow, error)     // full rows for subconscious
}

// SkillRow — full row for loader
type SkillRow struct {
    Name, Description, Instructions, Source, Path string
    Enabled      bool
    AutoActivate int  // for sort-before-truncation (higher = more important)
}

// Skill — unified in-memory representation
type Skill struct {
    Name         string
    Description  string
    Instructions string  // from file or DB
    Source       string  // "builtin" | "installed" | "registry"
    Path         string  // filesystem path if file-based
    Enabled      bool
}

// Subconscious — single place for loading logic
type Subconscious interface {
    Load(s SkillRow) (*Skill, error)
}
```

### 4.4 DRY: One Load Path (Subconscious)

- **install_skill** with path → **requires** path; validate, upsert DB (path = skill directory only)
- **create_skill** → DB-only; no path (use for builtin skills)
- **GetSkills** → metadata only (list_skills, /api/strategies) — no file I/O
- **GetSkillRows** → full rows for subconscious
- **Subconscious** — invoked **only by prompt builder**; for each row, if path set → load from file and merge with DB; else use DB. Single merge point.

No duplicate "load from path" logic; the subconscious is the only place file + DB are merged. list_skills and /api/strategies use raw `GetSkills()` — no file I/O on hot paths.

### 4.5 Package Layout

```
internal/
  skills/
    loader.go      # Subconscious, Load(s) → *Skill
    format.go      # FormatForPrompt(skills []*Skill) string
    store.go       # SkillStore adapter over *state.Database
  tools/
    skills.go      # InstallSkillTool, CreateSkillTool, RemoveSkillTool (unchanged interface)
    list_skills.go  # ListSkillsTool (unchanged)
  agent/
    prompt.go      # BuildSystemPrompt receives skills []*Skill, injects via FormatForPrompt
```

---

## 5. Skill Package Format (Minimal, Compatible)

### 5.1 SKILL.md (AgentSkills / OpenClaw compatible)

```markdown
---
name: crypto-dca
description: DCA into crypto — buy fixed amounts on schedule
version: 0.1.0
---

## Instructions

When the user asks to DCA, use the following procedure:
1. Check USDC balance
2. ...
```

- **Frontmatter:** `name`, `description`, `version` (required)
- **Body:** instructions (injected into prompt when skill enabled)

### 5.2 SKILL.toml (mormclaw compatible)

```toml
[skill]
name = "crypto-dca"
description = "DCA into crypto — buy fixed amounts on schedule"
version = "0.1.0"

# Optional: embedded instructions (fallback if instructions.md missing)
[skill.instructions]
text = """
When the user asks to DCA, use the following procedure:
1. Check USDC balance
2. ...
"""

# Phase 2: [[tools]] for shell-based tools (SkillToolHandler pattern)
```

- **Phase 1:** metadata from TOML; instructions load order:
  1. `instructions.md` (sibling file) if exists
  2. `[skill.instructions]` multiline string in TOML if present
  3. `""`
  Max ZeroClaw compatibility without converting files.
- **Phase 2:** `[[tools]]` for shell-based tools (SkillToolHandler pattern)

### 5.3 Precedence (Subconscious Merge)

**File precedence:** SKILL.toml first (structured), then SKILL.md. Respects ZeroClaw (TOML) and OpenClaw (MD).

```go
candidates := []string{"SKILL.toml", "SKILL.md"}
for _, f := range candidates {
    if exists(filepath.Join(dir, f)) {
        return loadThatFile()
    }
}
```

**Merge rules:** DB `name` is always canonical (PK). File `description` wins when file is loaded. File content wins for instructions; DB for enabled/source.

---

## 6. Prompt Injection

### 6.1 When to Inject

- **BuildSystemPrompt** receives `skills []*Skill` from `Subconscious.LoadAll(store.GetSkillRows())`
- Append block after genesis prompt:

```
--- ENABLED SKILLS ---
<skill_name>: <description>
<instructions>

<skill_name>: <description>
<instructions>
--- END SKILLS ---
```

### 6.2 Token Budget

- Per skill: ~50 (name+desc) + len(instructions)
- Cap total: e.g. 2000 chars for skills block (configurable)
- Truncate instructions with `...` when over limit
- **Sort before truncation:** `auto_activate DESC`, then `name` — most important skills get full content first. (Future: optional `priority` int in Skill/DB.)

---

## 7. Gating (Optional Phase 2)

Use `requires` JSON column:

```json
{"bins": ["python3"], "env": ["COINBASE_API_KEY"]}
```

- **Subconscious** parses `requires` **early** in `Load()` — before any file reads. If gating fails, return skip immediately (no I/O).
- If not satisfied: exclude from prompt, mark "unavailable" in list_skills
- Aligns with OpenClaw `metadata.openclaw.requires`

---

## 8. Discovery & Registry (Future)

| Feature | Priority | Notes |
|---------|----------|-------|
| **ClawHub-style registry** | P2 | HTTP API to list/install skills; `install_skill` from URL |
| **SkillForge pipeline** | P3 | Scout → Evaluate → Integrate; GitHub search |
| **Bundled skills** | P2 | Ship default skills in binary/embed |

---

## 9. Migration Path

### Phase 1 (Minimal, Low Risk)

1. Add `SkillRow` struct + `GetSkillRows() ([]SkillRow, error)` — full columns
2. Add `internal/skills/loader.go` — Subconscious: `Load(row) (*Skill, error)` with merge, validation, security, fault-tolerant handling
3. Add `internal/skills/format.go` — `FormatForPrompt(skills []*Skill) string`
4. Wire `BuildSystemPrompt` to call `loader.LoadAll(GetSkillRows())` → `FormatForPrompt`
5. Harden **install_skill**: path required; validate dir + SKILL.md/SKILL.toml; trusted roots; store directory only
6. Update SkillStore interface minimally
7. Write `TestSubconscious_MergeFileAndDB` — cover: MD only, TOML+instructions.md, TOML+[skill.instructions], bad path, permission denied, symlink escape attempt

list_skills and /api/strategies keep using `GetSkills()` (metadata only). No file I/O on hot paths.

### Phase 2

1. Gating: `requires` check in Loader
2. SKILL.toml `[[tools]]` → register dynamic tools (SkillToolHandler pattern)
3. CLI `strategies` → use DB + children (replace placeholder)

### Phase 3

1. Registry integration
2. SkillForge-style discovery

---

## 10. Alignment Summary

| Aspect | mormclaw | OpenClaw | mormoneyOS (proposed) |
|--------|----------|----------|------------------------|
| **Format** | SKILL.toml, SKILL.md | SKILL.md (YAML) | SKILL.md, SKILL.toml |
| **Location** | workspace/skills | workspace > ~/.openclaw | DB + optional path |
| **Source of truth** | Files | Files + config | DB (enabled), files (content) |
| **Prompt injection** | Full/compact | formatSkillsForPrompt | FormatForPrompt |
| **Tools from skills** | SkillToolHandler | Plugin tools | Phase 2 |
| **Gating** | trusted_skill_roots | requires.bins/env | requires (Phase 2) |
| **Registry** | SkillForge | ClawHub | Future |

---

## 11. Design Review: Logical Flaws & Approved Fixes

### Overall Design Review (Judgement Call)

This is a **strong, production-ready proposal** — clean, DRY, SOLID, and perfectly scoped for Phase 1. The **Subconscious** as the single runtime merge point is the smartest part: it solves the exact pain Claude-style agents have with stateless skills while keeping the DB as the enabled/strategy source of truth. It mirrors real-world patterns in **ZeroClaw** (mormclaw) and **OpenClaw** almost 1:1 (SKILL.md YAML frontmatter + `metadata.openclaw.requires`, workspace precedence, ClawHub gating, compact prompt injection all match).

No over-engineering. Token budget cap and fault-tolerant loading are thoughtful. Migration is low-risk.

The 10 flaws below are real blockers, but **every one has a simple, low-effort fix**. All fixes are adopted with minor refinements where they make the code cleaner or more future-proof.

---

### 11.1 GetSkills Returns Insufficient Columns (Blocker)

**Adopted fix.** Add `GetSkillRows() ([]SkillRow, error)` that returns the full struct (name, description, instructions, source, path, enabled). Keep existing `GetSkills()` for metadata-only callers (list_skills, /api/strategies). One extra SQL SELECT — negligible cost.

---

### 11.2 Path Semantics Ambiguous

**Adopted + strict.** Store **only the directory** in DB `path` column. Subconscious always does `filepath.Join(row.Path, "SKILL.*")`.

**Implementation (install_skill):**
```go
dir := path
if strings.HasSuffix(path, "SKILL.md") || strings.HasSuffix(path, "SKILL.toml") {
    dir = filepath.Dir(path)
}
if !isValidSkillDir(dir) { return error }
```
Document: "path = skill directory".

---

### 11.3 SKILL.toml Phase 1 Instructions Source Missing

**Adopted Option B with tweak.** Phase 1: Support **SKILL.md** for instructions (body after frontmatter). SKILL.toml is metadata-only in Phase 1.

**Subconscious rule (SKILL.toml):**
```go
if toml exists {
    meta = parseTOML()
    instructions = readFile("instructions.md")  // 1. sibling file
    if instructions == "" && meta.Instructions != "" {
        instructions = meta.Instructions       // 2. [skill.instructions]
    }
} else if md exists {
    instructions = parseMDbody()
}
```

---

### 11.4 SKILL.md vs SKILL.toml Precedence

**Adopted: TOML wins** (structured > markdown).

**Implementation:**
```go
candidates := []string{"SKILL.toml", "SKILL.md"}
for _, f := range candidates {
    if exists(filepath.Join(dir, f)) {
        return loadThatFile()
    }
}
```
ZeroClaw prefers TOML, OpenClaw uses MD — this respects both ecosystems.

---

### 11.5 install_skill Path Validation

**Adopted 100% — do not store bad paths.**

**Implementation (install_skill):**
1. Resolve to absolute path
2. Check exists && is dir
3. Contains SKILL.md **or** SKILL.toml
4. (Phase 1 security) Under `config.skills.trustedRoots` (default: `~/.mormoney/skills` + workspace)

Return clear error: "Skill directory must contain SKILL.md or SKILL.toml".

---

### 11.6 Subconscious Error Handling

**Adopted exactly.** Graceful degradation is critical for real agents.

**Implementation:**
```go
skills := []*Skill{}
for _, row := range rows {
    s, err := loader.Load(row)
    if err != nil {
        log.Warn("skill load failed", row.Name, err)
        continue  // never fail prompt build
    }
    skills = append(skills, s)
}
```

---

### 11.7 Name/Description Merge Rules

**Adopted + refinement.** DB `name` is **always** canonical (PK). File `description` wins (richer).

**Refinement:** On first successful file load, `UPDATE skills SET description = fileDesc WHERE name = ?` (one-time sync). Prevents stale DB descs forever. Very low cost; commit in Phase 1.

---

### 11.8 Security: Path Traversal

**Adopted immediately (even Phase 1).** Non-negotiable.

**Config (automaton.json):** Standardize on `~/.mormoney/skills` for project branding.
```json
{
  "skills": {
    "trustedRoots": ["~/.mormoney/skills", "{{workspace}}/skills"]
  }
}
```

**Implementation (loader + install_skill):**
```go
abs := filepath.Clean(path)
resolved, err := filepath.EvalSymlinks(abs)
if err != nil { reject }
allowed := false
for _, root := range trustedRoots {
    if strings.HasPrefix(resolved, root) { allowed = true; break }
}
if !allowed { reject }
```
Symlink hardening: if trusted root = `~/mormoney/skills` and user symlinks `malicious -> /etc`, reject — resolved path must stay under a trusted root. Matches ZeroClaw `trusted_skill_roots` and OpenClaw sandbox rules.

---

### 11.9 Which Consumers Need Subconscious?

**Adopted the split.**

- `list_skills` + `/api/strategies` → use raw `GetSkills()` (fast, no I/O)
- `BuildSystemPrompt` → `loader.LoadAll(GetSkillRows())` (full merged instructions)

Document in code comment so nobody accidentally adds file I/O to hot paths.

---

### 11.10 install_skill Without Path

**Adopted (b) + clarify naming.**

- `install_skill(name, path)` → **requires** path (file-based skill)
- `create_skill(name, description, instructions)` → DB-only (builtin)
- If someone calls install_skill with no path, error: "Use create_skill for DB-only skills"

This removes ambiguity and makes the two tools semantically distinct.

---

### 11.11 Recommended Phase 1 Rollout (Minimal Changes, Zero Downtime)

1. Add `SkillRow` struct + `GetSkillRows()`
2. Add `internal/skills/loader.go` (Subconscious) with all merge + validation + security rules above
3. Add `format.go` (the exact prompt block)
4. Update `BuildSystemPrompt` to call loader
5. Harden `install_skill` (validation + trusted roots)
6. Update SkillStore interface minimally
7. Write one test: `TestSubconscious_MergeFileAndDB`

Everything else (gating, tools from skills, registry) stays Phase 2.

This fixes **all** flaws with <200 lines of new Go code. The Subconscious layer becomes the exact "Claude subconscious" — background, automatic, persistent skill/memory management that just works across sessions.

---

### 11.12 Final Verdict & Next Steps

**Score: 9.2 / 10.** Ready to implement as Phase 1. Architecture feels "battle-tested inspired" rather than speculative.

**Recommended immediate actions (in order):**
1. Fix path consistency → `~/.mormoney/skills` everywhere (done in this doc)
2. Add TOML `[skill.instructions]` fallback for max ZeroClaw compatibility (done)
3. Write `TestSubconscious_MergeFileAndDB` (cover: MD only, TOML+instructions.md, TOML+[skill.instructions], bad path, permission denied, symlink escape)
4. Implement & land Phase 1 items in §9 — 1–2 focused days
5. Dogfood: create 2–3 test skills (one MD, one TOML), install via tool, edit file on disk → verify agent sees change without restart

---

## 12. References

- [ts-go-alignment.md](./ts-go-alignment.md) — Skills row in readiness table; design doc index
- [tool-system.md](./tool-system.md) — Tool registry, ServiceProvider pattern
- mormclaw: `src/skills/`, `src/skillforge/`
- OpenClaw: [docs.openclaw.ai/skills](https://docs.openclaw.ai/skills), [ClawHub](https://clawhub.com/)
