# Mnemonic-Only Multi-Chain Wallet Design (Refined)

**Date:** 2026-03-16  
**Purpose:** Store **only** BIP-39 mnemonic in `wallet.json`; derive keys & addresses on-demand via `github.com/morpheum-labs/standards` `MultiChainKeyManager`. Optimized for long-lived, automated agents (mormoneyOS / automaton).

---

## 1. Core Design Principles

- **Single source of truth** = mnemonic only (no persisted private keys)
- **On-demand derivation** (ephemeral keys in memory)
- **Maximize reuse** of standards package → minimal custom crypto
- **Immutable by default** (no accidental overwrite of mnemonic)
- **Agent-first mindset**: favor operational safety & linkability reduction over maximum theoretical security when trade-offs exist

---

## 2. Wallet Storage Format

**Recommended format** (minimal + extensible):

```json
{
  "mnemonic":       "abandon abandon ... about",
  "createdAt":      "2026-03-15T12:00:00Z",
  "hdAccountIndex": 0,
  "wordCount":      12
}
```

| Field | Required | Description |
|-------|----------|-------------|
| mnemonic | Yes | BIP-39 phrase; never logged |
| createdAt | Yes | ISO 8601 |
| hdAccountIndex | Yes | Default 0; stored here or in identity DB |
| wordCount | No | Recommended (12\|15\|18\|21\|24) for validation |

**Do NOT store passphrase here** — see §5.

**File rules** (enforced):

- 0600 permissions
- Blocked by policy engine (same as before)
- Never logged / never exposed via API / agent tools

---

## 3. CAIP-2 → ChainType Mapping

Only standards-supported chains get full guarantees:

| CAIP-2 | ChainType | Format | Status |
|--------|-----------|--------|--------|
| eip155:* (≠10900) | ChainTypeEthereum | 0x... | Supported |
| eip155:10900 (Morpheum) | ChainTypeMorpheum | mr4m1... | Supported |
| bip122:* mainnet | ChainTypeBitcoinSegwit | bc1q... | Supported |
| solana:* | ChainTypeSolana | Base58 | Supported |
| tron:*, xrpl:*, sui:*, polkadot:* | — | — | Unsupported |

**Decision**: Unsupported chains → either drop support or maintain separate local derivation (lower assurance, clearly documented).

---

## 4. Derivation & Index Strategy

**Current problem**: fixed index 0 → permanent address reuse → high linkability + poor forward security.

**Recommended approach** (balanced for agents):

- **Store `hdAccountIndex`** (uint32) — either in `wallet.json` (simpler) **or** in identity DB under key `"hd_account_index"` (preferred for consistency with other identity state)
- **Default**: 0 on first creation
- **Do not auto-rotate** — rotation is explicit & manual (agent continuity matters)
- Provide safe rotation command:

  ```bash
  moneyclaw wallet rotate --to-index 5 --preview
  # Shows new addresses per chain + diff vs current
  # Requires --confirm to actually update index
  ```

- **Starting index recommendation** for new agents: **10** (avoids trivial low-index enumeration if mnemonic partially leaked)

**Why not per-operation / time-bound paths?**  
Too complex for current scope; breaks continuity of identity / reputation / allowances. Reserve for future v2 (session keys).

---

## 5. BIP-39 Passphrase (25th word) — Agent Context

**Verdict**: **Optional advanced feature — not recommended by default for most automated agents**

**Pros** (still valid):

- Turns leaked `wallet.json` into decoy / empty wallet
- Plausible deniability in coercion scenarios

**Cons & agent-specific reality** (outweigh pros in most cases):

- Requires supplying passphrase **at every agent startup** (env var, prompt, secret manager) → operational fragility
- Forget passphrase → permanent loss (even with mnemonic backup)
- Runtime compromise still exposes derived keys (no protection once process is running)
- Most daemon/wallet-in-process designs avoid it precisely because startup becomes non-trivial

**Recommendation hierarchy**:

1. **Default**: no passphrase (simplest, least error-prone)
2. **High-value / high-risk deployments only**:
   - Fetch passphrase from secure store at startup (HashiCorp Vault, AWS Secrets Manager, systemd credentials, etc.)
   - Never store it in `wallet.json` or config
   - Show permanent warning in logs & `/api/status`:  
     `"Wallet uses BIP-39 passphrase — startup requires passphrase secret"`
3. **Never** prompt interactively in non-interactive daemon mode

---

## 6. Key Cache & Memory Discipline

**Problem**: indefinite caching of derived keys → long exposure window.

**Required implementation**:

- Use short-lived cache inside `MultiChainKeyManager` wrapper (or decorate it)
- **TTL options** (pick one):
  - Per-request / per-signing-operation (most secure, ~every call derives fresh)
  - Fixed TTL: 5–15 minutes (good balance)
  - Event-driven: clear after every sensitive action + every 30–60 min + on SIGTERM

- **Mandatory clear points**:
  - Graceful shutdown (SIGTERM)
  - After batch of signings (heartbeat, provisioning, tx submission)
  - Periodic timer (every 10–30 min)
  - Before long idle periods (if detectable)

- **Go implementation tip**:

  ```go
  type SecureKeyCache struct {
      mu    sync.RWMutex
      cache map[cacheKey]*CachedKey
      ttl   time.Duration
  }

  // ... with periodic cleanup goroutine
  ```

- If paranoid → explore `golang.org/x/crypto` zeroing helpers or `runtime.MemclrNoHeapPointers`

---

## 7. API Surface

| Function | Behavior |
|----------|----------|
| `GetWallet()` | Returns EVMAccount; derives from mnemonic. Creates new wallet only when wallet.json does not exist. |
| `GetWalletAddress()` | Primary EVM address; same derivation |
| `DeriveAddress(chainCAIP2)` | Returns address for chain; uses standards for supported chains |
| `DeriveAddress(chainCAIP2, index uint32)` | Extended with explicit index |
| `CurrentIndex() uint32` | Returns current hdAccountIndex |
| `RotateIndex(newIndex uint32) error` | With preview mode; does not sweep funds |
| `WalletExists()` | True if wallet.json exists |

### 7.1 moneyclaw init Behavior

- **No wallet.json:** Creates new mnemonic wallet, prints address.
- **wallet.json exists with mnemonic:** Uses existing mnemonic; does **not** create or overwrite. Prints address.
- **wallet.json exists without mnemonic:** Returns error; user must delete wallet.json and run init again to create a new wallet.

---

## 8. Security & Hardening Checklist

| Priority | Item | Status / Action |
|----------|------|-----------------|
| Critical | Never log mnemonic / derived keys | Enforce |
| Critical | 0600 + policy engine block on `wallet.json` | Done |
| High | Short-lived derived key cache + explicit clear | Implement |
| High | Configurable `hdAccountIndex` + safe rotate command | Implement |
| Medium | Validate mnemonic + wordCount on every load | Add |
| Medium | Optional passphrase support (startup secret only) | Optional |
| Low | xpub/master-pubkey fingerprint in identity DB | Nice-to-have |
| Low | Graceful legacy privateKey migration helper | Phase 2 |

---

## 9. Summary Verdict & Roadmap

**Production-ready with these fixes applied**:

### Must-have (next sprint)

1. `hdAccountIndex` storage + derivation by index
2. Short-lived key cache + clear discipline
3. `moneyclaw wallet rotate` (with preview)

### Should-have (soon)

4. Word count validation
5. Optional passphrase-at-startup support (secret store integration)

### Nice-to-have (later)

6. xpub fingerprint check
7. Unsupported chains fallback strategy

The design is now much tighter for real automated/agentic usage — strong security posture without sacrificing operability.

---

## 10. Implementation

| File | Role |
|------|------|
| `internal/identity/types.go` | WalletData: mnemonic, createdAt, hdAccountIndex, wordCount |
| `internal/identity/wallet.go` | Load/save; GetWallet derives on demand; no create when mnemonic exists |
| `internal/identity/derivation.go` | CAIP2→ChainType mapping; DeriveAddressFromMnemonic; index-aware |
| `internal/identity/bootstrap.go` | Uses DeriveAddress |
| `internal/identity/provision.go` | Uses GetWallet → EVMAccount |
| `cmd/init.go` | Calls GetWallet; prints address. Creates wallet only when none exists. |
| `cmd/wallet.go` | `rotate` subcommand with --preview, --to-index, --confirm |

---

## 11. Standards Package Usage

```go
// Key derivation
keyManager := clitool.NewMultiChainKeyManager()
chainKey, err := keyManager.DeriveKey(mnemonic, types.ChainTypeEthereum, index)
addr, err := chainKey.PublicKey().Address()

// Mnemonic generation
mnemonic, err := clitool.GenerateMnemonicFromLength(12)

// Validation
err := clitool.ValidateMnemonic(mnemonic)
```
