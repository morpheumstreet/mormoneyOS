# Agent Card

The agent card is a JSON-LD document served at `/.well-known/agent-card.json` for [ERC-8004](https://ethereum-magicians.org/t/erc-8004-autonomous-agent-identity/22268) agent discovery. It enables other agents and systems to discover an automaton's identity, capabilities, and contact information.

## Endpoint

| Method | Path | Content-Type | Cache |
|--------|------|--------------|-------|
| GET | `/.well-known/agent-card.json` | `application/ld+json; charset=utf-8` | `public, max-age=300` (5 min) |

- **POST** and other methods return `405 Method Not Allowed`.
- The route is registered in `internal/web/server.go` and must be defined before the catch-all `/` handler.

## JSON-LD Structure

The agent card is a `SoftwareApplication` with schema.org and agent vocabularies:

```json
{
  "@context": ["https://schema.org", "https://w3id.org/agent"],
  "@type": "SoftwareApplication",
  "applicationCategory": "AI Agent",
  "url": "https://example.com/.well-known/agent-card.json",
  "name": "MoneyClaw",
  "description": "...",
  "identifier": [{ "@type": "PropertyValue", "name": "ethereum:address", "value": "0x...", "description": "eip155:8453" }],
  "capabilities": ["Budget tracking", "Portfolio analysis"],
  "creator": { "@type": "Person", "identifier": [...] }
}
```

## Data Sources and Resolution Order

### 1. Base URL

Derived from the HTTP request so it works behind tunnels and proxies:

| Source | Priority |
|--------|----------|
| `X-Forwarded-Proto` | If present, used as scheme |
| `r.TLS != nil` | If true → `https`, else `http` |
| `X-Forwarded-Host` | If present, used as host |
| `r.Host` | Fallback host |

`url` = `{scheme}://{host}/.well-known/agent-card.json`

### 2. Name

| Source | Priority |
|--------|----------|
| `ServerConfig.Name` | If set |
| `"MoneyClaw"` | Default |

### 3. Description

| Source | Priority |
|--------|----------|
| Soul `## Core Purpose` section | If `soul_content` exists and has this section |
| Soul `## Mission` section | Fallback if Core Purpose missing |
| First 300 chars of soul + `"..."` | If soul > 300 chars and no Core Purpose/Mission |
| Full soul content | If soul ≤ 300 chars |
| `"Autonomous AI agent (" + Name + ")"` | If config has Name but no soul |
| `"Autonomous AI agent powered by mormoneyOS"` | Final fallback |

### 4. Ethereum Address (identifier)

Used for ERC-8004 on-chain identity. Chain is `ServerConfig.DefaultChain` or `eip155:8453` (Base).

| Source | Priority |
|--------|----------|
| `DB.GetIdentity("address_eip155:8453")` | Per-chain key from `identity.AddressKeyForChain(chain)` |
| `DB.GetIdentity("address")` | Legacy key |
| `ServerConfig.WalletAddress` | Config fallback |
| `identity.DeriveAddress(chain)` | Derived from wallet mnemonic if no stored address |

If an address is found, it is emitted as:

```json
"identifier": [{
  "@type": "PropertyValue",
  "name": "ethereum:address",
  "value": "0x...",
  "description": "eip155:8453"
}]
```

### 5. Capabilities

Parsed from the soul document's `## Capabilities` section:

- Soul content is split by `## Section Name` headers.
- Section names are lowercased for lookup (e.g. `"Capabilities"` → `"capabilities"`).
- Each line in the section is trimmed, `-` prefixes removed, and non-empty lines become capability strings.

Example soul:

```markdown
## Capabilities
- Budget tracking
- Portfolio analysis
```

Produces:

```json
"capabilities": ["Budget tracking", "Portfolio analysis"]
```

### 6. Creator (optional)

If `ServerConfig.CreatorAddress` is set:

```json
"creator": {
  "@type": "Person",
  "identifier": [{ "@type": "PropertyValue", "name": "ethereum:address", "value": "0xcreator..." }]
}
```

## Soul Section Parsing

`parseSoulSections(body string)` extracts markdown sections from `soul_content`:

- **Regex**: `(?m)^##\s+(.+)$` — matches `## Section Name` at line start.
- **Section content**: Text from the header line to the next `##` or end of document.
- **Truncation**: Section content is capped at 500 characters; excess is replaced with `"..."`.
- **Keys**: Section names are lowercased and trimmed (e.g. `"Core Purpose"` → `"core purpose"`).

Used sections:

- `core purpose` — description
- `mission` — description fallback
- `capabilities` — capability list

## Implementation Flow

```
GET /.well-known/agent-card.json
    │
    ▼
handleWellKnownAgentCard
    │
    ├─ Method != GET → 405 Method Not Allowed
    │
    ▼
buildAgentCard(r)
    │
    ├─ Base URL from r (scheme, host, X-Forwarded-*)
    ├─ Name from Cfg.Name or "MoneyClaw"
    ├─ Description from soul_content (Core Purpose / Mission / truncate) or fallbacks
    ├─ Address from DB (address_<chain>, address) or Cfg.WalletAddress or DeriveAddress
    ├─ Capabilities from soul_content ## Capabilities
    └─ Creator from Cfg.CreatorAddress
    │
    ▼
JSON encode → application/ld+json, Cache-Control: max-age=300
```

## Database Dependencies

The agent card uses these KV/identity keys:

| Key | Purpose |
|-----|---------|
| `soul_content` | Soul markdown (description, capabilities) |
| `address` | Legacy Ethereum address |
| `address_eip155:8453` | Per-chain address (Base) |
| `address_<caip2>` | Per-chain address for other chains |

`identity.AddressKeyForChain(caip2)` returns `"address_" + caip2` (e.g. `address_eip155:8453`).

## Related Components

- **`update_agent_card` tool** (stub): Intended to generate and persist the agent card; not yet implemented.
- **`register_erc8004` tool** (stub): Registers the agent on-chain and publishes the agent card URI.
- **`discover_agents` tool** (stub): Fetches agent cards from the ERC-8004 registry.

## Testing

`internal/web/agent_card_test.go` covers:

- Successful GET with name, description, identifier, Content-Type, and URL
- POST returns 405
- Mock DB with `soul_content`, `address`, and `address_eip155:8453`
