# Social Channel Design — Multi-Provider Architecture

**Date:** 2026-03-13  
**Purpose:** Design a CLEAN, DRY, SOLID social channel layer (Conway, Telegram, Discord, etc.) for mormoneyOS. Unblocks `send_message` and `check_social_inbox`. All channels share one interface; Conway is a first-class channel. Borrows patterns from mormclaw channels and mormoneyOS provider design.

---

## 0. Design Principles (Clean, DRY, SOLID)

### 0.1 Clean

| Principle | Application |
|-----------|-------------|
| **Single responsibility** | Each channel does one thing: send/receive messages on its platform. No business logic, no routing. |
| **Clear boundaries** | Channel = transport + auth. Agent loop = orchestration. Factory = construction. No cross-cutting concerns. |
| **Explicit over implicit** | API keys, webhook URLs, allowed users — all passed via config. No magic defaults. |

### 0.2 DRY

| Principle | Application |
|-----------|-------------|
| **Shared types** | `OutboundMessage`, `InboxMessage`, `ChannelMessage` live in a common package; all channels consume them. |
| **Shared validation** | Message size limits, rate limiting, replay protection — one module, all channels. |
| **Provider descriptors, not code** | New channel = registry entry (key, config keys, constructor). Minimal per-provider code. |

### 0.3 SOLID

| Principle | Application |
|-----------|-------------|
| **S**ingle Responsibility | Channel = send + poll only. Factory = construction. Registry = metadata. |
| **O**pen/Closed | Add channels by registering, not by editing factory switch. |
| **L**iskov Substitution | Any `SocialChannel` implementation works in agent loop; no channel-specific branches. |
| **I**nterface Segregation | `SocialChannel` has minimal surface: `Send`, `Poll`, `HealthCheck`. No fat interface. |
| **D**ependency Inversion | Agent loop depends on `SocialChannel` interface; factory injects concrete implementation. |

---

## 1. Context: Unified Social Channels

All social messaging — Conway relay, Telegram, Discord, Slack — uses the same `SocialChannel` interface. No distinction between "agent-to-agent" and "user-facing"; they are all channels.

| Channel | Recipient format | Auth | Current State |
|---------|------------------|------|---------------|
| **conway** | `0x...` wallet address | Wallet signing + socialRelayUrl | TS: `SocialClientInterface`. Go: stub. |
| **telegram** | Chat/user ID | Bot token | Not implemented. |
| **discord** | Channel ID | Bot token | Not implemented. |
| **slack** | Channel ID | Bot token | Not implemented. |

---

## 2. mormclaw Channel Architecture (Reference)

### 2.1 Core Components

| Component | Path | Role |
|----------|------|------|
| **Channel trait** | `src/channels/traits.rs` | `send`, `listen`, `health_check`, `start_typing`, `stop_typing`, optional draft/reaction methods |
| **ChannelMessage** | `traits.rs` | Inbound: id, sender, reply_target, content, channel, timestamp, thread_ts |
| **SendMessage** | `traits.rs` | Outbound: content, recipient, subject, thread_ts |
| **ChannelConfig** | `config/traits.rs` | Per-channel: `name()`, `desc()` |
| **Config schema** | `config/schema.rs` | `ChannelsConfig` with optional `telegram`, `discord`, `slack`, etc. |
| **Factory** | `channels/mod.rs` | Build channels from config; wire into agent |

### 2.2 Channel Trait (Simplified)

```rust
#[async_trait]
pub trait Channel: Send + Sync {
    fn name(&self) -> &str;
    async fn send(&self, message: &SendMessage) -> anyhow::Result<()>;
    async fn listen(&self, tx: Sender<ChannelMessage>) -> anyhow::Result<()>;
    async fn health_check(&self) -> bool;
    // Optional: typing, draft updates, reactions
}
```

### 2.3 Per-Channel Config

- **Telegram**: `bot_token`, `allowed_users`, optional `workspace_dir`, `ack_reaction`
- **Discord**: `bot_token`, `guild_id`, `allowed_users`, `listen_to_bots`, `mention_only`
- **Slack**: `bot_token`, `allowed_channels`, etc.

---

## 3. mormoneyOS Social Channel Design

### 3.1 Interface (Minimal, ISP)

```go
// internal/social/channel.go

package social

import "context"

// OutboundMessage is the normalized outbound payload.
type OutboundMessage struct {
	Content   string
	Recipient string // Conway: 0x... wallet; Telegram: chat_id; Discord: channel_id
	ThreadID  string // Optional; for threaded replies (Slack thread_ts, Discord thread)
}

// InboxMessage is a normalized inbound message.
type InboxMessage struct {
	ID         string
	Sender     string
	ReplyTarget string // Where to send reply (chat_id, channel_id, 0x...); session routing
	Content    string
	Channel    string // "conway", "telegram", "discord", etc.
	Timestamp  int64
	ThreadID   string
}

// SocialChannel is the minimal interface for all social platforms (Conway, Telegram, Discord, etc.).
type SocialChannel interface {
	Name() string
	Send(ctx context.Context, msg *OutboundMessage) (messageID string, err error)
	Poll(ctx context.Context, cursor string, limit int) ([]InboxMessage, string, error)
	HealthCheck(ctx context.Context) error
}

// Optional interfaces (default no-op when not implemented):
// - TypingChannel: StartTyping(recipient), StopTyping(recipient)
// - ReactionChannel: AddReaction(channelID, messageID, emoji), RemoveReaction(...)
```

**Rationale:** `Poll` instead of `Listen` for mormoneyOS — heartbeat-driven polling fits the existing `check_social_inbox` task. Long-running `Listen` can be added later as an optional interface.

### 3.2 Shared Types and Validation

```go
// internal/social/types.go

const (
	MaxMessageLength   = 4096  // Align with Telegram; truncate for others
	MaxOutboundPerHour = 60
)

// ValidateOutbound checks size and rate limits before send.
func ValidateOutbound(content string) error { ... }
```

### 3.3 Channel Registry (Open/Closed)

```go
// internal/social/registry.go

type ChannelSpec struct {
	Key             string   // "conway", "telegram", "discord", "slack"
	DisplayName     string
	TokenConfigKey  string   // e.g. "TelegramBotToken"; Conway uses wallet, not token
	URIConfigKey    string   // e.g. "SocialRelayURL" for Conway
	Constructor     func(cfg *types.AutomatonConfig) (SocialChannel, error)
}

var registry = []ChannelSpec{
	{"conway", "Conway", "", "SocialRelayURL", NewConwayChannel},
	{"telegram", "Telegram", "TelegramBotToken", "", NewTelegramChannel},
	{"discord", "Discord", "DiscordBotToken", "", NewDiscordChannel},
	// ...
}

func LookupChannel(key string) *ChannelSpec
func NewChannelFromConfig(key string, cfg *types.AutomatonConfig) (SocialChannel, error)
```

### 3.4 Factory

```go
// internal/social/factory.go

// NewChannelsFromConfig builds all enabled channels from config.
// Returns map keyed by channel name for tool/heartbeat use.
func NewChannelsFromConfig(cfg *types.AutomatonConfig) map[string]SocialChannel
```

**Logic:** For each channel key in `cfg.SocialChannels` (e.g. `["conway", "telegram"]`), lookup spec, resolve config (token or URL + wallet per channel), call constructor. Skip channels with missing config.

### 3.5 Config Shape

```yaml
# automaton.json

# Enabled social channels — first is default for send_message
socialChannels: ["conway", "telegram"]

# Conway channel (wallet-signed agent-to-agent)
socialRelayUrl: "https://social.conway.tech"

# Per-channel keys (factory reads based on socialChannels)
telegramBotToken: "..."
discordBotToken: "..."
discordGuildId: "..."   # optional
slackBotToken: "..."

# Per-channel allowlist (inbound filter; empty = deny all, ["*"] = allow all)
telegramAllowedUsers: ["123456789"]
discordAllowedUsers: ["987654321"]
discordMentionOnly: false   # When true, only respond to @-mentions in guilds
discordListenToBots: false
```

**Allowlist semantics (mormclaw-aligned):** `allowed_users` empty = deny all until configured. `["*"]` = allow all. Otherwise exact match on sender ID. Poll implementations filter before returning; or filter at wake_event insertion.

---

## 4. Tool Integration

### 4.1 send_message Tool

Single tool with `channel` and `recipient` — same params for all channels:

```json
{
  "channel": "conway",
  "recipient": "0x1234...",
  "content": "...",
  "reply_to": "msg_123"
}
```

- `channel`: `"conway"` | `"telegram"` | `"discord"` | … (must be enabled in config)
- `recipient`: `0x...` for Conway; chat/channel ID for Telegram/Discord
- Omit `channel` → use first enabled channel as default

### 4.2 check_social_inbox Heartbeat Task

Poll all enabled channels uniformly:

```go
func runCheckSocialInbox(tc *TickContext) {
	for _, ch := range tc.Channels {
		msgs, next, _ := ch.Poll(ctx, cursor, limit)
		// Merge, dedupe, insert wake_events for new messages
	}
}
```

---

## 5. File Layout

```
internal/social/
  channel.go      # SocialChannel interface, OutboundMessage, InboxMessage
  types.go        # Validation, constants
  registry.go     # ChannelSpec, registry, LookupChannel
  factory.go      # NewChannelsFromConfig
  conway.go       # ConwayChannel (wallet-signed relay)
  telegram.go     # TelegramChannel
  discord.go      # DiscordChannel
  stub.go         # StubChannel for tests / no config
```

**TS alignment:** Refactor `src/social/client.ts` into `ConwayChannel` implementing a shared `SocialChannel` interface; add `TelegramChannel`, `DiscordChannel` when porting.

---

## 6. Implementation Order

1. **Scaffold** — Add `internal/social/` with interface, types, registry, stub.
2. **Config** — Add `socialChannels`, `socialRelayUrl`, `telegramBotToken`, etc. to `types.AutomatonConfig` and `config.Load()`.
3. **Conway** — Implement `ConwayChannel` (port TS `SocialClientInterface` logic: send, poll, wallet signing).
4. **Factory** — Wire `NewChannelsFromConfig`; integrate into run path.
5. **send_message** — Replace stub with real impl; channel + recipient params.
6. **check_social_inbox** — Implement real polling for all enabled channels.
7. **Telegram** — Implement `TelegramChannel` (send via Bot API, poll via getUpdates).
8. **Discord** — Add `DiscordChannel` (Gateway or REST poll).

---

## 7. Security and Limits

| Concern | Mitigation |
|---------|------------|
| **API key exposure** | Never log tokens; config keys in allowlist only. |
| **Rate limiting** | Per-channel outbound cap (e.g. 60/hour); shared `ValidateOutbound`. |
| **Allowlist** | Per-channel `allowed_users` / `allowed_channels` (config). |
| **Message size** | Enforce `MaxMessageLength`; truncate or reject. |
| **Replay** | Conway: nonce in signed payload; Telegram/Discord: platform message IDs. |

---

## 8. Remaining Design from mormclaw (To Add or Defer)

Features present in mormclaw channels that are not yet in this design:

| Feature | mormclaw | Purpose | Priority |
|---------|----------|---------|----------|
| **Listen (long-running)** | `listen(tx)` | Real-time inbound via channel sender; vs Poll for heartbeat | Defer — Poll sufficient for MVP |
| **Typing indicators** | `start_typing`, `stop_typing` | UX: show "typing..." while processing | Medium — add to interface with default no-op |
| **Draft / progressive updates** | `send_draft`, `update_draft`, `finalize_draft`, `cancel_draft`, `supports_draft_updates` | Streaming: edit message in-place (Telegram) | Defer — add when streaming UX needed |
| **ACK reactions** | `add_reaction`, `remove_reaction` | Add 👀/⚡️ to incoming msg when processing | Medium — improves UX; optional interface |
| **Approval prompt** | `send_approval_prompt` | Tool approval: send prompt + `/approve-allow`, `/approve-deny` | Defer — mormoneyOS policy engine differs |
| **Subject** | `SendMessage.subject` | Email-like channels (subject line) | Low — add to OutboundMessage when needed |
| **reply_target** | `ChannelMessage.reply_target` | Where to send reply (chat_id, channel_id) | **Add** — InboxMessage should have ReplyTarget for session/reply routing |
| **Per-channel allowlist** | `allowed_users`, `allowed_channels` | Inbound filter: only process from allowed IDs | **Add** — config + Poll filter |
| **Per-channel group/mention** | `mention_only`, `group_reply`, `listen_to_bots` | When to respond in groups | Add for Telegram/Discord |
| **Ack reaction config** | `ack_reaction` per channel | Emojis, rules, sample_rate | Defer — add with reactions |
| **Stream mode** | `stream_mode` (Telegram) | Off / Chunks / Native | Defer — with draft updates |
| **Message timeout** | `message_timeout_secs` | Max time per channel message | Low — add to run context |
| **Bind / pairing** | `/bind` command (Telegram) | User-initiated add to allowlist | Defer — CLI/setup flow |
| **ChannelConfig display** | `name()`, `desc()` | UI/CLI channel list | Low — registry already has DisplayName |

### Recommended Additions (MVP+)

1. **InboxMessage.ReplyTarget** — Where to send the reply (chat_id, channel_id). Aligns with mormclaw `reply_target`; needed for session isolation and correct reply routing.
2. **Per-channel allowlist** — `allowed_users` / `allowed_channels` in config; filter inbound in Poll or before wake_event.
3. **Optional typing interface** — `StartTyping`, `StopTyping` with default no-op; Telegram/Discord can implement.
4. **Optional reaction interface** — `AddReaction`, `RemoveReaction` with default no-op; for ACK reactions later.

### Defer to Later Phases

- Listen (long-running) — Poll is enough for heartbeat-driven mormoneyOS.
- Draft updates, stream mode — When streaming response UX is required.
- Approval prompt — Policy engine integration differs; design separately.
- Bind/pairing — User onboarding flow; separate from channel core.

---

## 9. References

- mormclaw: `src/channels/traits.rs`, `telegram.rs`, `discord.rs`, `ack_reaction.rs`, `config/schema.rs`
- mormoneyOS: `docs/design/mormclaw-provider-borrow.md`, `docs/design/ts-go-alignment.md`
- mormoneyOS TS: `src/social/client.ts` (Conway relay), `src/agent/tools.ts` (send_message)
