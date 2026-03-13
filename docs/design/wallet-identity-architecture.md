# Wallet Identity Architecture (Go Implementation)

**Date:** 2026-03-13  
**Purpose:** Architecture for implementing wallet/identity in Go, aligned with TypeScript reference (`src/identity/`, `ARCHITECTURE.md`). **Multi-chain support is foundational**, not retrofitted.

---

## 1. Executive Summary

The automaton has a **sovereign multi-chain identity**: a wallet (or HD seed) whose keys are the automaton's root of trust. The design is **multi-chain from the foundation**, supporting **EVM, Bitcoin, Tron, XRP, Sui, Polkadot**, and other chain ecosystems via CAIP-2 identifiers. Chain identifiers, per-chain address derivation, default/primary chain, and chain-scoped operations are first-class. The TS implementation uses `viem` for EVM; the Go implementation will support multiple chain types via `internal/identity/` and chain-specific crypto libraries.

| Component        | TS Reference                    | Go Target                          |
| ---------------- | ------------------------------- | ---------------------------------- |
| Wallet storage   | `~/.automaton/wallet.json` 0600 | Same path, same permissions        |
| Key generation   | viem `generatePrivateKey`       | `crypto/ecdsa` + secp256k1         |
| Address derivation | viem `privateKeyToAccount`    | `go-ethereum` `crypto.PubkeyToAddress` |
| SIWE provisioning | `siwe` + Conway `/v1/auth/*`  | `internal/identity/provision.go`   |
| Identity DB     | `identity` table (key-value)    | Already exists in Go schema        |
| Config merge    | `loadApiKeyFromConfig()`        | Load from `~/.automaton/config.json` |
| **Chain model**  | `eip155:8453` (registry, children) | CAIP-2 identifiers, default chain config |

---

## 2. Multi-Chain Foundation (Design Principle)

Multi-chain support is **built into the foundation**, not layered on later. All identity, config, and on-chain operations are chain-aware.

### 2.1 Chain Identifier Format

Use **CAIP-2** (Chain Agnostic Improvement Proposal 2):

```
<namespace>:<chain_id>
```

**Supported chain namespaces:**

| Namespace | Chains | Chain ID format | Example |
| --------- | ------ | ----------------- | ------- |
| `eip155` | Ethereum, Base, Polygon, Arbitrum, Optimism | Integer (EIP-155) | `eip155:1`, `eip155:8453` |
| `bip122` | Bitcoin | Genesis block hash | `bip122:000000000019d6689c085ae165831e93` |
| `cosmos` | Cosmos Hub, Osmosis | Chain ID string | `cosmos:cosmoshub-4` |
| `polkadot` | Polkadot, Kusama | SS58 prefix or chain ID | `polkadot:91b171bb158e2d3848fa23a9f1c25182` |
| `solana` | Solana | Cluster identifier | `solana:5eykt4zzFbh8iLudsaDkvVygepqvV22oZwod8EqB` |
| `sui` | Sui | Chain identifier | `sui:mainnet` |
| `tron` | Tron | Network ID | `tron:728126428` (mainnet) |
| `xrpl` | XRP Ledger | Network ID or hash | `xrpl:0` (mainnet) |

**EVM (eip155):** `eip155:1`, `eip155:8453`, `eip155:42161`, `eip155:137`, `eip155:10`  
**Bitcoin (bip122):** Mainnet `bip122:000000000019d6689c085ae165831e93`, Testnet `bip122:000000000933ea01ad0ee984209779ba`  
**Tron:** `tron:728126428` (mainnet), `tron:2494104990` (shasta testnet)  
**XRP Ledger:** `xrpl:0` (mainnet), `xrpl:1` (testnet)  
**Sui:** `sui:mainnet`, `sui:testnet`  
**Polkadot:** `polkadot:91b171bb158e2d3848fa23a9f1c25182` (mainnet)

### 2.2 Address Model (Multi-Chain)

- **EVM (eip155):** One secp256k1 key → same address on all EVM chains. Address format: `0x` + 40 hex chars.
- **Bitcoin (bip122):** One key → multiple formats (legacy, segwit, bech32). Use BIP-44 path `m/44'/0'/0'/0/0`.
- **Tron:** secp256k1, base58check with `T` prefix (mainnet). Same key type as EVM but different address encoding.
- **XRP (xrpl):** secp256k1, base58 `r`-prefixed (classic) or X-address format.
- **Sui:** Ed25519 or secp256k1, `0x`-prefixed (different from EVM; 32-byte hash).
- **Polkadot:** Ed25519 or Sr25519, SS58 format (chain-specific prefix).

**Identity storage:** Store **address per chain** (or per namespace) in identity table or `addresses` map:

```
identity: address_eip155:8453 → "0x..."
identity: address_bip122:... → "bc1q..."
identity: address_tron:728126428 → "T..."
identity: address_xrpl:0 → "r..."
identity: address_sui:mainnet → "0x..."
identity: address_polkadot:... → "1..."
```

**Default/primary address:** When `defaultChain` is EVM, `address` key holds the EVM address (backward compatible). For non-EVM default, `address` holds the address for `defaultChain`.

### 2.2.1 Key Derivation (Multi-Chain)

| Chain | Key type | Derivation | Library / standard |
| ----- | -------- | ---------- | ------------------ |
| EVM (eip155) | secp256k1 | Direct from private key | go-ethereum/crypto |
| Bitcoin (bip122) | secp256k1 | BIP-44 `m/44'/0'/0'/0/0` | btcutil, btcd |
| Tron | secp256k1 | Tron-specific from privkey | tron lib |
| XRP (xrpl) | secp256k1 | XRP key derivation | ripple/keypair |
| Sui | Ed25519 or secp256k1 | Sui key scheme | sui-go-sdk |
| Polkadot | Ed25519 or Sr25519 | Substrate key derivation | go-substrate-rpc-client |

**HD wallet (optional):** BIP-39 mnemonic + BIP-44 paths per chain type for single-seed multi-chain. Phase 1 can use single EVM key; Phase 3+ adds HD and per-chain derivation.

### 2.3 Default Chain

- **Default chain** — the chain used for provisioning, Conway API (when applicable), and primary operations when no chain is specified.
- **Primary chain** — alias for default; used in config and identity.
- **Supported chains** — list of chains the automaton can operate on: EVM (Base, Ethereum, Polygon, etc.), Bitcoin, Tron, XRP, Sui, Polkadot, and others as added.

### 2.4 Config Extensions (Multi-Chain)

```json
{
  "defaultChain": "eip155:8453",
  "chains": {
    "eip155:8453": {
      "name": "Base",
      "rpcUrl": "https://mainnet.base.org",
      "conwayApiUrl": "https://api.conway.tech"
    },
    "eip155:1": {
      "name": "Ethereum",
      "rpcUrl": "https://eth.llamarpc.com"
    },
    "bip122:000000000019d6689c085ae165831e93": {
      "name": "Bitcoin",
      "rpcUrl": "https://bitcoin.drpc.org"
    },
    "tron:728126428": {
      "name": "Tron",
      "rpcUrl": "https://api.trongrid.io"
    },
    "xrpl:0": {
      "name": "XRP Ledger",
      "rpcUrl": "https://xrplcluster.com"
    },
    "sui:mainnet": {
      "name": "Sui",
      "rpcUrl": "https://fullnode.mainnet.sui.io"
    },
    "polkadot:91b171bb158e2d3848fa23a9f1c25182": {
      "name": "Polkadot",
      "rpcUrl": "wss://rpc.polkadot.io"
    }
  }
}
```

- `defaultChain` — CAIP-2 string; used for provisioning (SIWE when EVM) and unspecified operations.
- `chains` — optional per-chain config (RPC URL, Conway API URL when applicable, etc.).

### 2.5 Identity Table Extensions

| Key | Description |
| --- | ----------- |
| `address` | Primary address for `defaultChain` (backward compatible) |
| `address_<caip2>` | Address per chain, e.g. `address_eip155:8453`, `address_bip122:...`, `address_tron:728126428` |
| `default_chain` | CAIP-2 default chain (e.g. `eip155:8453`) |
| `supported_chains` | Optional JSON array of CAIP-2 chains the automaton can use |
| `name`, `creator`, `sandbox`, `createdAt` | (unchanged) |

### 2.6 SIWE and Provisioning (Multi-Chain)

- **EVM (eip155):** SIWE message includes `chainId` (EIP-155 integer). Provisioning uses `defaultChain` when it is eip155.
- **Non-EVM:** SIWE is EVM-specific. For Bitcoin, Tron, XRP, Sui, Polkadot, provisioning may use chain-specific auth (e.g. message signing, wallet connect) or Conway may accept alternative auth per chain. When `defaultChain` is non-EVM, primary operations use that chain; Conway API key may be provisioned via EVM as fallback or via chain-specific flow when supported.
- Provisioning is **per default chain** when chain supports it; Conway API may support different chains via different endpoints or config.

### 2.7 Schema Alignment (TS Reference)

TS schema already uses chain in:

- `registry.chain` — default `eip155:8453`
- `children` — (via `address`; chain can be added for child sandbox chain)
- `onchain_transactions.chain` — required

Go schema should mirror: add `chain` or `default_chain` to identity keys; ensure registry, children, and transaction tables use CAIP-2 chain identifiers.

### 2.8 Implementation Implications

- **Wallet:** Support HD seed (BIP-39/44) for multi-chain key derivation, or multiple keys per chain type. EVM: secp256k1; Bitcoin: secp256k1 (BIP-44 path); Tron: secp256k1; XRP: secp256k1; Sui: Ed25519 or secp256k1; Polkadot: Ed25519 or Sr25519.
- **Address derivation:** Per-chain derivation: `DeriveAddress(chainCAIP2 string) (string, error)`.
- **Config:** Add `defaultChain`, `chains`; validate CAIP-2 format and namespace.
- **Identity:** Add `default_chain`, `address_<caip2>` keys; persist from config and derivation.
- **Provision:** Use `defaultChain` when eip155 for SIWE; support chain-specific auth for others.
- **API:** All chain-scoped endpoints accept optional `chain` param; default to `defaultChain`.

---

## 2.9 Standards Package Integration (Borrowed from Morpheum Standards)

The [morpheumlabs/standards](https://github.com/morpheumlabs/standards) package provides reusable patterns for multi-chain signing and verification. mormoneyOS borrows the following concepts.

### 2.9.1 Spec-First, Language-Agnostic Design

- **Domain spec:** EIP-712 domain (name, version, chainId) defined per environment. Canonical behavior lives in specs, not only in code.
- **Wire format:** Signatures in hex; addresses in 0x-prefix; chain IDs as EIP-155 integers.
- **Test vectors:** Shared inputs and expected outputs for conformance across implementations.

### 2.9.2 EIP-712 Domain with ChainId

Standards domain structure (from `types/message_standard.go`):

```go
type Domain struct {
    Name    string   `json:"name"`    // Application name
    Version string   `json:"version"` // Application version
    ChainId *big.Int `json:"chainId"` // EIP-155 chain ID
}
```

For mormoneyOS SIWE and future EIP-712 signing:

- **Domain name:** `"Conway"` or `"conway.tech"` (SIWE)
- **Version:** `"1"`
- **ChainId:** Derived from `defaultChain` (e.g. `eip155:8453` → `8453`)

### 2.9.3 Chain-Aware Nonce (Replay Prevention)

Standards nonce design (from `auth/nonce.go`, multichain audit):

- Nonces are **per (owner, chainID)**, not global.
- Prevents cross-chain replay attacks.
- `ChainNonce` struct: `Owner`, `ChainID`, `Nonce`.

For mormoneyOS:

- Identity/signing operations that use nonces should scope by `(address, chainID)`.
- Use `ChainIDFromCAIP2(defaultChain)` when validating nonces.

### 2.9.4 Per-Signature ChainID (Future)

When mormoneyOS supports multi-sign or cross-chain operations:

- **Signature struct** can include optional `ChainID *big.Int` for per-signature chain.
- **ChainType** derived from SigType/address when needed; not stored in structs (standards SigType-Only design).
- See `docs/improvements/multichain_multisign_audit_gaps.md` for full pattern.

### 2.9.5 Address Validation (Chain-Aware)

Standards supports multi-chain address validation (Ethereum, Solana, Bitcoin variants). mormoneyOS extends to all supported chains:

- **EVM (eip155):** `0x` + 40 hex chars, checksum optional. `common.Address` for type safety.
- **Bitcoin (bip122):** Bech32 (`bc1...`), legacy (`1...`), segwit (`3...`).
- **Tron:** Base58check, `T` prefix (mainnet), `27` (testnet).
- **XRP (xrpl):** Classic `r...` or X-address format.
- **Sui:** `0x` + 64 hex chars (32-byte hash).
- **Polkadot:** SS58 format, chain-specific prefix (e.g. 0 for Polkadot, 2 for Kusama).

`ValidateAddressForChain(addr, chainCAIP2) error` — validate address format per chain namespace.

### 2.9.6 Optional Dependency on Standards Package

| Use Case | mormoneyOS Approach | Standards Package |
| -------- | -------------------- | ----------------- |
| SIWE signing | Own implementation (siwe-go) | N/A |
| EIP-712 domain | Domain struct with chainId | `types.Domain` |
| Nonce management | Per (owner, chainID) | `auth.NonceManager` + chain-aware extension |
| Chain ID parsing | `ChainIDFromCAIP2()`, `ChainIDToCAIP2()` | `types.ChainIDTestnet` etc. |
| Multi-sign / cross-type | Future | `types.EIP712Tx`, `Signature`, `SigType` |

mormoneyOS can **optionally** depend on `github.com/morpheum-labs/standards` for:

- Shared domain types and chain constants.
- Nonce manager with chain-aware validation.
- EIP-712 hashing and verification (when Conway/ecosystem operations use EIP-712).

Or implement a minimal subset locally to avoid cross-repo coupling for Phase 1.

---

## 3. TypeScript Reference (Source of Truth)

### 3.1 Directory Layout

```
src/identity/
  wallet.ts     # Generate/load wallet, get address, load account for signing
  provision.ts  # SIWE flow → Conway API key → config.json
```

### 3.2 Wallet Module (`wallet.ts`)

| Function            | Behavior                                                                 |
| ------------------- | ----------------------------------------------------------------------- |
| `getAutomatonDir()` | `~/.automaton` or `$HOME/.automaton`                                    |
| `getWalletPath()`   | `~/.automaton/wallet.json`                                              |
| `getWallet()`       | Create dir 0700 if missing; load or generate key; return `{ account, isNew }` |
| `getWalletAddress()`| Read wallet.json, derive address, return or null                         |
| `loadWalletAccount()` | Load full `PrivateKeyAccount` for signing                             |
| `walletExists()`    | `fs.existsSync(WALLET_FILE)`                                            |

**Storage format** (`wallet.json`):

```json
{
  "privateKey": "0x...",
  "createdAt": "2026-03-13T12:00:00.000Z"
}
```

### 3.3 Provision Module (`provision.ts`)

| Function                 | Behavior                                                                 |
| ------------------------ | ----------------------------------------------------------------------- |
| `loadApiKeyFromConfig()` | Read `~/.automaton/config.json` → `apiKey`                               |
| `provision()`            | SIWE flow: nonce → sign message → verify → create API key → save config |
| `registerParent()`       | POST `/v1/automaton/register-parent` with creator address               |

**SIWE flow:**

1. Load wallet via `getWallet()`
2. POST `{apiUrl}/v1/auth/nonce` → `{ nonce }`
3. Build SIWE message (domain, address, statement, uri, chainId from defaultChain, nonce)
4. Sign with `account.signMessage()`
5. POST `{apiUrl}/v1/auth/verify` with `{ message, signature }` → `{ access_token }`
6. POST `{apiUrl}/v1/auth/api-keys` with Bearer token, body `{ name: "conway-automaton" }` → `{ key, key_prefix }`
7. Save to `~/.automaton/config.json`: `{ apiKey, walletAddress, provisionedAt }`

**Config.json format:**

```json
{
  "apiKey": "cnwy_k_...",
  "walletAddress": "0x...",
  "provisionedAt": "2026-03-13T12:00:00.000Z"
}
```

### 3.4 Identity Table (DB)

```sql
CREATE TABLE identity (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

**Keys used at runtime:**

| Key            | Source                    | Notes                          |
| -------------- | ------------------------- | ------------------------------ |
| `address`      | `account.address`         | EVM address (0x...); same across chains |
| `default_chain`| config.defaultChain       | CAIP-2 (e.g. `eip155:8453`)    |
| `name`         | config.name               | Agent display name             |
| `creator`      | config.creatorAddress     | Creator wallet                 |
| `sandbox`      | config.sandboxId          | Conway sandbox ID              |
| `createdAt`    | First run timestamp       | Never overwrite once set       |

### 3.5 Bootstrap Flow (TS `index.ts`)

```
1. loadConfig() → null triggers setup wizard
2. getWallet() → { account, isNew }
3. loadApiKeyFromConfig() || config.conwayApiKey
4. createDatabase(dbPath)
5. db.setIdentity("createdAt", ...) if not set
6. identity = { name, address, account, creatorAddress, sandboxId, apiKey, createdAt }
7. db.setIdentity("address", account.address)
8. db.setIdentity("name", config.name)
9. ... create Conway, inference, heartbeat, run loop
```

### 3.6 Consumers of Identity

| Consumer        | Uses                                      |
| --------------- | ----------------------------------------- |
| Agent loop      | `identity` (name, address, account for signing) |
| Heartbeat       | `identity.address` for distress/funding   |
| Web /api/status  | `db.getIdentity("address")` or config     |
| Social client   | `account` for `signSendPayload`, `signPollPayload` |
| Conway topup    | `account` for USDC permit / payment       |
| Policy engine   | Path protection blocks wallet.json       |

---

## 4. Go Architecture

### 4.1 Module Layout

```
internal/identity/
  wallet.go      # GetAutomatonDir, GetWalletPath, GetWallet, GetWalletAddress, WalletExists
  provision.go   # LoadAPIKeyFromConfig, Provision, RegisterParent
  chain.go       # ChainIDFromCAIP2, ChainIDToCAIP2, DefaultChainBase; Domain (name, version, chainId)
  nonce.go       # ChainNonce, ValidateChainNonce (optional; borrow from standards auth/nonce)
  types.go       # WalletData, ProvisionResult, ChainConfig (optional, or use types pkg)
```

### 4.2 Dependencies

| Need              | Go Package                          |
| ----------------- | ----------------------------------- |
| secp256k1 keygen  | `crypto/ecdsa` + `crypto/rand`       |
| Address from key  | `github.com/ethereum/go-ethereum/crypto` |
| SIWE message      | `github.com/spruceid/siwe-go` or manual |
| JSON (wallet)     | `encoding/json`                     |

### 4.3 Wallet Interface

```go
// WalletData is the persisted wallet format (TS-aligned for EVM; extended for multi-chain).
type WalletData struct {
    PrivateKey string `json:"privateKey"` // 0x-prefixed hex (EVM/secp256k1)
    Mnemonic   string `json:"mnemonic,omitempty"`   // BIP-39 seed for HD derivation (optional)
    CreatedAt  string `json:"createdAt"`
}

// Account holds address and signing capability (EVM-focused for Phase 1).
type Account interface {
    Address() string
    SignMessage(message []byte) ([]byte, error)
}

// ChainAccount extends Account with chain-specific address.
type ChainAccount interface {
    Account
    Chain() string  // CAIP-2
}

// GetWallet loads or creates the automaton wallet (EVM primary).
func GetWallet() (Account, bool, error)

// GetWalletAddress returns the primary (defaultChain) address, or "" if no wallet.
func GetWalletAddress() string

// DeriveAddress returns the address for the given CAIP-2 chain.
// Supports eip155, bip122, tron, xrpl, sui, polkadot namespaces.
func DeriveAddress(chainCAIP2 string) (string, error)

// WalletExists returns true if wallet.json exists.
func WalletExists() bool
```

### 4.4 Provision Interface

```go
// ProvisionResult is returned by Provision (TS-aligned).
type ProvisionResult struct {
    APIKey        string `json:"apiKey"`
    WalletAddress string `json:"walletAddress"`
    KeyPrefix     string `json:"keyPrefix"`
}

// LoadAPIKeyFromConfig reads ~/.automaton/config.json and returns apiKey or "".
func LoadAPIKeyFromConfig() string

// Provision runs SIWE flow and saves API key to config.json.
// chainID is the EIP-155 chain ID for SIWE (e.g. 8453 for Base); derived from defaultChain.
func Provision(apiURL string, chainID uint64) (*ProvisionResult, error)

// RegisterParent registers creator with Conway (optional, 404-tolerant).
func RegisterParent(creatorAddress, apiURL, apiKey string) error
```

### 4.5 Chain Helpers (Multi-Chain, Standards-Aligned)

```go
// Namespace returns the CAIP-2 namespace (e.g. "eip155", "bip122", "tron").
func Namespace(caip2 string) string

// ChainIDFromCAIP2 parses "eip155:8453" -> 8453. Returns 0 if invalid or non-eip155.
func ChainIDFromCAIP2(caip2 string) uint64

// ChainIDToCAIP2 formats 8453 -> "eip155:8453" (EVM only).
func ChainIDToCAIP2(chainID uint64) string

// DefaultChainBase is the default CAIP-2 chain (Base).
const DefaultChainBase = "eip155:8453"

// IsEVM returns true if the chain uses EIP-155 (Ethereum, Base, Polygon, etc.).
func IsEVM(caip2 string) bool

// Domain holds EIP-712 domain (standards-aligned; EVM only).
type Domain struct {
    Name    string
    Version string
    ChainId *big.Int  // From ChainIDFromCAIP2(defaultChain); only when IsEVM
}
```

### 4.6 Config Merge (Existing)

Go `config.Load()` already reads `walletAddress` from `automaton.json`. The identity layer adds:

- **Fallback:** If `walletAddress` is empty, call `identity.GetWalletAddress()` and use that.
- **API key fallback:** If `conwayApiKey` is empty, call `identity.LoadAPIKeyFromConfig()`.
- **Multi-chain:** Add `defaultChain` (string, CAIP-2); default `"eip155:8453"` if missing. Add optional `chains` map for per-chain config.

### 4.7 Identity Table Usage (Existing)

Go already has:

- `state.Database.GetIdentity(key)` / `SetIdentity(key, value)`
- Web server: `GetIdentity("address")` with fallback to `Cfg.WalletAddress`

**Gap:** On first run, we must persist `address` (and optionally `name`, `creator`, `sandbox`, `createdAt`) from wallet + config into the identity table. TS does this in the run bootstrap; Go should do the same in `cmd/run.go`.

### 4.8 Bootstrap Integration (cmd/run.go)

Proposed flow:

```
1. config.Load()
2. If cfg == nil → run setup (future: moneyclaw setup)
3. If cfg.WalletAddress == "" → identity.GetWalletAddress() or GetWallet(); persist to cfg
4. If cfg.ConwayAPIKey == "" → identity.LoadAPIKeyFromConfig()
5. defaultChain := cfg.DefaultChain; if empty use "eip155:8453"
6. state.Open(dbPath)
7. Persist identity to DB: SetIdentity("address", address), SetIdentity("default_chain", defaultChain), SetIdentity("name", name), etc.
8. ... rest of run (agent, heartbeat, web)
```

### 4.9 CLI Commands (TS-Aligned)

| Command       | TS                    | Go (proposed)              |
| ------------- | --------------------- | -------------------------- |
| `--init`      | getWallet, print addr | `moneyclaw init`           |
| `--provision` | provision(), print    | `moneyclaw provision`      |
| `--setup`     | setup wizard          | `moneyclaw setup` (exists) |

---

## 5. Security Contract

1. **File permissions:** `wallet.json` and `config.json` must be 0600.
2. **Path protection:** Policy engine must block reads of `wallet.json`, `config.json`, `.env`, and any path containing `privateKey` or `apiKey`.
3. **No logging:** Never log private keys, raw API keys, or full wallet content.
4. **Directory:** `~/.automaton` created with 0700.

---

## 6. Implementation Phases

### Phase 1: Wallet Only (No SIWE)

- [ ] `internal/identity/wallet.go`: GetAutomatonDir, GetWalletPath, GetWallet, GetWalletAddress, WalletExists
- [ ] Use `go-ethereum/crypto` for EVM key generation and address derivation
- [ ] `moneyclaw init` command
- [ ] Config merge: fallback WalletAddress from GetWalletAddress()
- [ ] Run bootstrap: persist address to identity table when missing
- [ ] **Multi-chain:** Add `defaultChain` to config; persist `default_chain` to identity table
- [ ] **Chain types:** `internal/identity/chain.go` — Namespace, IsEVM, ChainIDFromCAIP2; validate CAIP-2 for eip155, bip122, tron, xrpl, sui, polkadot

### Phase 2: SIWE Provisioning

- [ ] `internal/identity/provision.go`: LoadAPIKeyFromConfig, Provision(apiURL, chainID)
- [ ] SIWE message construction (EIP-4361); chainID from defaultChain
- [ ] Conway auth endpoints: nonce, verify, api-keys
- [ ] `moneyclaw provision` command
- [ ] Config merge: fallback ConwayAPIKey from LoadAPIKeyFromConfig()
- [ ] **Multi-chain:** `internal/identity/chain.go`: ChainIDFromCAIP2, ChainIDToCAIP2, Domain struct
- [ ] **Standards-aligned:** Domain (name, version, chainId) for SIWE; optional borrow from standards types

### Phase 3: Full Identity Wiring

- [ ] RegisterParent (optional)
- [ ] Ensure heartbeat, web, agent all use identity table or config consistently
- [ ] Setup wizard: create wallet, provision, write config (if not already done)
- [ ] **Multi-chain:** Schema: registry, children, onchain_transactions use CAIP-2 chain; API accepts optional `chain` param
- [ ] **Standards-aligned:** Chain-aware nonce (per owner, chainID) for replay prevention
- [ ] **Non-EVM:** `DeriveAddress(chainCAIP2)` for Bitcoin, Tron, XRP, Sui, Polkadot; `address_<caip2>` in identity table

---

## 7. Data Flow Diagram

```
                    +------------------+
                    |  ~/.automaton/   |
                    |  wallet.json     |
                    |  (private key)   |
                    +--------+---------+
                             |
                             v
                    +------------------+
                    | identity/wallet   |
                    | GetWallet()      |
                    | GetWalletAddress |
                    +--------+---------+
                             |
         +-------------------+-------------------+
         |                   |                   |
         v                   v                   v
  +-------------+    +-------------+    +------------------+
  | config.Load |    | cmd/run     |    | identity/provision|
  | (merge addr)|    | (bootstrap) |    | (SIWE → api key) |
  +-------------+    +-------------+    +------------------+
         |                   |                   |
         v                   v                   v
  +-------------+    +-------------+    +------------------+
  | automaton   |    | identity    |    | config.json      |
  | .json       |    | table       |    | (apiKey, addr)   |
  +-------------+    +-------------+    +------------------+
                             |
         +-------------------+-------------------+
         |                   |                   |
         v                   v                   v
  +-------------+    +-------------+    +------------------+
  | Agent loop  |    | Heartbeat   |    | Web /api/status  |
  | (prompt)    |    | (distress)  |    | (address)        |
  +-------------+    +-------------+    +------------------+
```

---

## 8. References

### mormoneyOS

- [ARCHITECTURE.md](../../ARCHITECTURE.md) — Identity and Wallet section
- [ts-go-alignment.md](./ts-go-alignment.md) — Current Go vs TS alignment
- TS: `src/identity/wallet.ts`, `src/identity/provision.ts`
- TS: `src/index.ts` (bootstrap), `src/config.ts` (loadApiKeyFromConfig)
- TS: `src/state/schema.ts` — registry.chain, onchain_transactions.chain (eip155:8453)

### Standards Package (morpheumlabs/standards)

- `docs/design/STANDARDS_PACKAGE_DESIGN.md` — Spec-first, language-agnostic design
- `docs/improvements/multichain_multisign_audit_gaps.md` — Multi-chain design gaps and patterns
- `docs/improvements/multichain_multisign_summary.md` — Implementation roadmap
- `types/message_standard.go` — Domain, SigType, EIP712Tx
- `auth/nonce.go` — NonceManager (extend for chain-aware)
- `signer/domain.go` — CreateDefaultDomain, CreateTestnetDomain

### Specifications

- EIP-4361: Sign-In With Ethereum
- CAIP-2: Chain ID specification (namespace:reference)
- CAIP-10: Account ID (chain + address)
- BIP-39: Mnemonic seed
- BIP-44: HD wallet paths (e.g. m/44'/0'/0' for Bitcoin, m/44'/60'/0' for Ethereum)
