<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus

**A Simple, Beautiful, and Elegant LLM API Aggregation Service for Individuals**

 English | [简体中文](README_zh.md)

</div>


## ✨ Features

- 🔀 **Multi-Channel Aggregation** - Connect multiple LLM provider channels with unified management
- 🔄 **Protocol Conversion** - Seamless conversion between OpenAI Chat / OpenAI Responses / Anthropic API formats
- 💰 **Price Sync** - Automatic model pricing updates
- 🔃 **Model Sync** - Automatic synchronization of available model lists with channels
- 🛡️ **Automatic Failover** - Automatically switches to an available channel when an upstream channel fails
- 🔍 **Real-Time End-to-End Request Visualization** - Watch the complete request path in the frontend from the moment the client sends it
- 🚧 **Upstream Error Shielding** - Intercept all upstream errors to keep agent tasks running without interruption
- 📊 **Analytics** - Comprehensive request statistics, token consumption, and cost tracking
- 🎨 **Elegant UI** - Clean and beautiful web management panel
- 📦 **Lightweight Single-Binary Deployment** - Run as a single binary with no external runtime dependencies
- 🗄️ **Multi-Database Support** - Support for SQLite, MySQL, PostgreSQL
- 🎭 **Client Emulation** - Per-channel Claude Code / Codex client impersonation (headers + request-body shape) for upstreams that validate client identity; probes and the candy test follow the same shape over streaming


## 🚀 Quick Start

### 🐳 Docker

Build this fork's container locally (this release publishes binaries, not a GHCR image):

```bash
git clone https://github.com/1466109846/octopus.git
cd octopus
docker build -f scripts/dockerfile/Dockerfile -t octopus:local .
docker run -d --name octopus -v /path/to/data:/app/data -p 18874:18874 octopus:local
```

Or build via docker compose from this checkout:

```bash
docker compose up -d --build
```


### 📦 Download from Release

Download the binary for your platform from [Releases](https://github.com/1466109846/octopus/releases), then run:

```bash
./octopus start
```

### 🛠️ Build from Source

**Requirements:**
- Go 1.26.4
- Node.js 24+
- pnpm 10+

```bash
# Clone the forked repository
git clone https://github.com/1466109846/octopus.git
cd octopus
# Build frontend
cd web && pnpm install && pnpm run build
# Start the backend service
go run main.go start 
```

> 💡 **Tip**: The frontend build artifacts are embedded into the Go binary, so you must build the frontend before starting the backend.

**Development Mode**

```bash
cd web && pnpm install && pnpm run dev
## Open a new terminal, start the backend service
go run main.go start
## Access the frontend at
http://localhost:5173
```

### 🔐 Default Credentials

After first launch, visit http://localhost:18874 and log in to the management panel with:

- **Username**: `admin`
- **Password**: `admin`

> ⚠️ **Security Notice**: Please change the default password immediately after first login.

### 📝 Configuration File

The configuration file is located at `data/config.json` by default and is automatically generated on first startup.

**Complete Configuration Example:**

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 18874
  },
  "database": {
    "type": "sqlite",
    "path": "data/data.db"
  },
  "log": {
    "level": "info"
  }
}
```

**Configuration Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `server.host` | Listen address | `0.0.0.0` |
| `server.port` | Server port | `18874` |
| `database.type` | Database type | `sqlite` |
| `database.path` | Database connection string | `data/data.db` |
| `log.level` | Log level | `info` |

**Database Configuration:**

Three database types are supported:

| Type | `database.type` | `database.path` Format |
|------|-----------------|-----------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL Configuration Example:**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL Configuration Example:**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **Tip**: MySQL and PostgreSQL require manual database creation. The application will automatically create the table structure.

### 🌐 Environment Variables

All configuration options can be overridden via environment variables using the format `OCTOPUS_` + configuration path (joined with `_`):

| Environment Variable | Configuration Option |
|---------------------|---------------------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_GITHUB_PAT` | For rate limiting when getting the latest version (optional) |

## 📸 Screenshots

### 🖥️ Desktop

<div align="center">
<table>
<tr>
<td align="center"><b>Dashboard</b></td>
<td align="center"><b>Channel Management</b></td>
<td align="center"><b>Group Management</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="Dashboard" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="Channel" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="Group" width="400"></td>
</tr>
<tr>
<td align="center"><b>Price Management</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="Price Management" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="Logs" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="Settings" width="400"></td>
</tr>
</table>
</div>

### 📱 Mobile

<div align="center">
<table>
<tr>
<td align="center"><b>Home</b></td>
<td align="center"><b>Channel</b></td>
<td align="center"><b>Group</b></td>
<td align="center"><b>Price</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="Mobile Home" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="Mobile Channel" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="Mobile Group" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="Mobile Price" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="Mobile Logs" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="Mobile Settings" width="140"></td>
</tr>
</table>
</div>


## 📖 Documentation

### 📡 Channel Management

Channels are the basic configuration units for connecting to LLM providers.

**Base URL Guide:**

The program automatically appends the API version and endpoint path based on the channel type. You only need to provide the service root URL:

| Channel Type | Auto-appended Path | Base URL | Full Request URL Example |
|--------------|-------------------|----------|--------------------------|
| OpenAI Chat | `/v1/chat/completions` | `https://api.openai.com` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/v1/responses` | `https://api.openai.com` | `https://api.openai.com/v1/responses` |
| Anthropic | `/v1/messages` | `https://api.anthropic.com` | `https://api.anthropic.com/v1/messages` |
| Gemini | `/v1beta/models/:model:generateContent` | `https://generativelanguage.googleapis.com` | `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` |

> 💡 **Tip**: The Base URL does not need to include `/v1`, `/v1beta`, or a specific API endpoint path - the program handles them automatically.

---

> 📢 **Note:** This repository is a fork of `bestruirui/octopus`, extending the upstream codebase with the fork-only features documented below and the [Model Monitor](#-model-monitor--scheduled-probe).

### 🧪 Upstream Model Substitution Detection

> **Why**: Upstreams (especially relay/aggregator services) may silently answer with a different model than the one you requested — you ask for `claude-opus-4-8` and pay flagship prices while possibly being served by a cheaper model instead. The client only ever sees the name it sent, so until now there was no local way to spot this "watering down" directly.

The Log page now compares the **requested model** with the model the **upstream response claims to have served** (parsed from the top-level `model` field of the response JSON):

- **Match** → a green check; **mismatch** → a red warning badge showing the model the upstream actually reported, with the full explanation on hover (both on the log card and in the detail dialog)
- Covers all four forwarding paths (same-protocol passthrough / cross-protocol conversion × streaming / non-streaming) — the final bodies all converge on the request-state finalization points, so parsing happens in exactly one place
- Nothing is shown when the upstream omits the field (no false alarms); the comparison ignores case and surrounding whitespace
- **Detection only, no blocking** — for some relays a "model name mapping" is legitimate (the official name differs from the API name); what to do about it is the user's call based on the logs

### 🛡️ Probe Prompt Anti-Ban

> **Why**: The generic `"hi"` probe prompt is easily fingerprinted as probe traffic by upstream sites, risking account bans (some sites already flag probe-like traffic). Accounts are expensive, and probing is a core feature of this program — it must not become unusable.

Probe requests now send a human-sounding radio-check phrase:

> 喂喂喂 通信测试通信测试：请回答1988 收到请回复：是的，1988

The model promptly replies with a short answer, keeping the original minimal-cost path (`max_tokens` stays 16; timeouts and verdict semantics are unchanged). If this phrase ever gets fingerprinted too, only the `probePrompt` constant in `internal/relay/probe.go` needs a change and a rebuild.

### 💓 Keepalive

> **Why**: Some upstreams require you to keep the session active — queue-based or activity-gated services can drop you out of the queue when no traffic arrives for a while. Model Monitor answers "is this channel healthy?", which is not the same question: its failures cool the channel down, and in queue scenarios a rejection (HTTP 429) is normal, so a monitor-style cooldown would punish a perfectly fine channel.

The **Keepalive** page sends a minimal request to each selected channel/credential on an interval (default **5 minutes**) to *attempt* to maintain upstream activity and a queue slot:

- Queue, interval amortization, credential rotation and per-credential rows are shared with Model Monitor (same task table, split by task kind); the two pages never mix.
- Each row shows the latest heartbeat's status chip, **HTTP status code** and latency: `429/503` means the request reached the upstream but was rejected; no status code usually means a timeout, network failure or local configuration problem. **A response code alone does not prove the upstream reserved your queue slot**.
- **Keeps knocking while the upstream is busy**: on a busy rejection (`429` queue full / `500`, `502`, `503`, `504` transient upstream failures) the same round retries with backoff (up to 3 attempts, 2s→4s, whole-round budget 60s) — so the moment a queue slot opens you can slip in, instead of knocking once per cycle (default 5 minutes). The verdict notes "retried N times, still rejected" so you can see the behavior working.
- Retry boundaries: timeouts and network errors (no status code) are never retried — that would turn one hang into three; 4xx other than 429 (wrong key, unknown model) are deterministic and are not retried either.
- **Display only, never routing**: a failed keepalive does not cool the channel down, does not lower its health score and never affects failover — exactly the opposite of what you would want here.
- Keepalive always sends the minimal probe request (never the candy/IQ question), and runs all day (no time window) to avoid prolonged periods of inactivity; it cannot guarantee that an upstream will keep your queue position.
- Manual "keepalive now" works per credential or for the whole task, with per-row verdicts.

### 🎭 Client Emulation (Claude Code / Codex)

> **Why**: Some upstreams (especially Claude Code / Codex relay services) validate the client identity in request headers and reject requests that don't look like they come from the official client. Two rejections reproduced against a real upstream: a relay answered `请启用 1m 上下文后重试` ("enable 1M context and retry") when the 1M-context beta header was missing, and a new-api based codex channel answered `invalid codex request` when `originator` / `OpenAI-Beta` were missing. The gateway's own requests never carry these headers, so both probes and forwarding get rejected — unrelated to the request itself, the protocol or the key: it is purely an identity issue.

The channel form's **Connection** step now has a **Client Emulation** dropdown, choosing which official client the outbound request should impersonate:

- **None (default)** — keeps the gateway's own request shape; behavior is byte-for-byte what it was before
- **Claude Code** — sends the `claude-cli` User-Agent / `X-App: cli`, and **always includes the 1M-context beta header** (`context-1m-2025-08-07`); `anthropic-beta` is additive, so an existing beta value is appended to rather than overwritten
- **Codex** — fills in `originator: codex_cli_rs`, `OpenAI-Beta: responses=experimental`, `Version`, and the Session / Thread / Turn metadata headers that make up the Codex CLI session shape

Key points:

- **Emulation has two layers, neither touching the URL or protocol direction**: the header layer adds identity headers, the body layer (Claude Code + Messages only) adds the request-body identity shape. Which endpoint to use (Chat / Responses / Messages) is still decided by the granted protocol bits; the two are orthogonal
- Covers every outbound path: forwarding (passthrough and conversion), probing (monitor and keepalive), model fetching and image forwarding — so "try fetching in the form" and "forward after saving" present the same client identity
- **Ordering contract**: emulation lays down defaults first, and the channel's **custom headers and parameter overrides take effect last** — override explicitly to use a different UA / beta set / system prompt
- Existing valid shapes (e.g. a client's passthrough User-Agent, or a real Claude Code client's own system and metadata) are preserved byte-for-byte; emulation only fills in what is missing

**Claude Code mode — the two layers in detail** (every rule below is a hard gate measured against a real upstream):

- **Header layer**: the `claude-cli` User-Agent / `X-App: cli`, plus the **1M-context beta header by default** (`context-1m-2025-08-07`; missing it is an immediate 400); `anthropic-beta` is additive, so an existing beta value is appended to rather than overwritten
- **Body layer** (Messages endpoint):
  - `system` is normalized to an array whose first block is an official marker line (`You are Claude Code, Anthropic's official CLI for Claude.` or the Agent SDK line); when the real 2.1.x billing line (`x-anthropic-billing-header:`) comes first, the identity line is inserted right after it
  - `metadata.user_id` is normalized to parseable v2 JSON (`device_id` + a well-formed UUID `session_id`; `account_uuid` may be empty) or the legacy format — randomly generated values are accepted
  - Only applies to Claude Code + Messages; channels without emulation and endpoints that are not Messages stay byte-for-byte unchanged
- **Probes follow the client's streaming shape**: the real Claude Code client always streams to the Messages endpoint, and the upstream's CC supply path returns no content for non-streaming requests (measured: HTTP 520 empty response). So for channels in Claude Code mode, availability probes (monitor / keepalive / model page) and the candy (IQ) test all stream; the candy test collects the full event stream and aggregates it through the inbound transformer before grading

### 📁 Group Management

Groups aggregate multiple channels into a unified external model name.

**Core Concepts:**

- **Group name** is the model name exposed by the program
- When calling the API, set the `model` parameter to the group name

> 💡 **Example**: Create a group named `gpt-4o`, add multiple providers' GPT-4o channels to it, then access all channels via a unified `model: gpt-4o`.
>
> 💡 **Tip**: In the Channel page, click the menu on a channel and choose "Add to Group" to add all of the channel's models to an existing group in one click.

---

### 💰 Price Management

Manage model pricing information in the system.

**Data Sources:**

- The system periodically syncs model pricing data from [models.dev](https://github.com/sst/models.dev)
- When creating a channel, if the channel contains models not in models.dev, the system automatically creates pricing information for those models on this page, so this page displays models that haven't had their prices fetched from upstream, allowing users to set prices manually
- Manual creation of models that exist in models.dev is also supported for custom pricing

**Price Priority:**

| Priority | Source | Description |
|:--------:|--------|-------------|
| 🥇 High | This Page | Prices set by user in price management page |
| 🥈 Low | models.dev | Auto-synced default prices |

> 💡 **Tip**: To override a model's default price, simply set a custom price for it in the price management page.

---

### ⚙️ Settings

Global system configuration.

**Statistics Save Interval (minutes):**

Since the program handles numerous statistics, writing to the database on every request would impact read/write performance. The program uses this strategy:

- Statistics are first stored in **memory**
- Periodically **batch-written** to the database at the configured interval

> ⚠️ **Important**: When exiting the program, use proper shutdown methods (like `Ctrl+C` or sending `SIGTERM` signal) to ensure in-memory statistics are correctly written to the database. **Do NOT use `kill -9` or other forced termination methods**, as this may result in statistics data loss.

---

## 🔌 Client Integration

> 💡 **Tip**: This fork keeps the same API surface as upstream (`OpenAI Chat / Responses / Anthropic`, plus image passthrough at `/v1/images/{generations,edits}`).

### OpenAI SDK

```python
from openai import OpenAI
import os

client = OpenAI(   
    base_url="http://127.0.0.1:18874/v1",
    api_key="sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg", 
)
completion = client.chat.completions.create(
    model="octopus-openai",  # Use the correct group name
    messages = [
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### 🎨 Image Generation / Editing ( Images )

Passthrough for `/v1/images/generations` and `/v1/images/edits`. Currently only OpenAI-protocol channels participate in routing:

```python
from openai import OpenAI

client = OpenAI(base_url="http://127.0.0.1:18874/v1", api_key="sk-octopus-...")
resp = client.images.generate(prompt="a cat wearing sunglasses", n=1, size="1024x1024")
print(resp.data[0].url)
```

### Claude Code

Edit `~/.claude/settings.json`

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:18874",
    "ANTHROPIC_AUTH_TOKEN": "sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg",
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "ANTHROPIC_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_SMALL_FAST_MODEL": "octopus-haiku-4-5",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "octopus-haiku-4-5"
  }
}
```

### Codex

Edit `~/.codex/config.toml`

```toml
model = "gpt-5.6-sol"
model_reasoning_effort = "xhigh"
model_provider = "octopus"
preferred_auth_method = "apikey"

[model_providers.octopus]
base_url = "http://127.0.0.1:18874/v1"
name = "octopus"
supports_websockets = false
requires_openai_auth = true
wire_api = "responses"
experimental_bearer_token = "sk-octopus-"
```

Edit `~/.codex/auth.json`

```json
{
  "OPENAI_API_KEY": ""
}
```

---

## 🤝 Acknowledgments

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - The LLM API adaptation module in this project is directly derived from this repository
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI model database providing model pricing data
- 🇨🇳 [AtomGit](https://atomgit.com/bestruirui/octopus) - China-based code hosting
- 💬 [Linux.do](https://linux.do/)

### 🕵️ Model Monitor / Scheduled Probe

A fork-only feature for periodic health checks: probes each channel/credential in the background on a rotating schedule and surfaces the current latency/status on the **Model Monitor** page.

- **Multi-target with rename** — a single probe task can monitor many channels/models, with a custom group name
- **Evenly distributed intervals** — the interval is split across credentials to avoid thundering probes on the same group
- **Drag-to-reorder rows** — reorder probe targets by dragging
- **Never-stale results** — live-updating badge & latency, with a 5-minute validity window (expired results fade out)
