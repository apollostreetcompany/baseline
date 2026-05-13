## Blunt verdict

The product shape is close, but “full launch implementation” still needs one brutal constraint:

**Baseline v0 is not an eval platform. It is a local known-good drift checker with a cloud receipt.**

The launchable bead is:

```text
baseline check
baseline compare --known-good
baseline scrub preview
baseline serve mcp
baseline sync on
```

Everything else is either support machinery or a paid retention wrapper. The product should feel like **“I run this before I let OpenClaw touch a repo.”** If a feature does not improve that moment, cut it.

Also: keep Go for the CLI/MCP. The official MCP docs currently list Go as a Tier 1 SDK, alongside TypeScript, Python, and C#; that supports the one-binary local-first choice. ([Model Context Protocol][1])

---

## 1. Hard Go package boundaries

Use one Go binary, but do **not** let it become one Go blob. Package boundaries should enforce trust boundaries.

Recommended tree:

```text
cmd/baseline/
  main.go

internal/app/
  commands.go              # orchestration/use cases only
  check_service.go
  compare_service.go
  sync_service.go

internal/domain/
  ids.go
  run.go
  check_result.go
  observation.go
  egress.go
  status.go

internal/cli/
  root.go
  init.go
  check.go
  report.go
  compare.go
  sync.go
  scrub.go
  serve_mcp.go
  doctor.go

internal/config/
  config.go
  redaction.go
  policy.go
  paths.go

internal/store/sqlite/
  db.go
  migrations/
  runs_repo.go
  checks_repo.go
  known_goods_repo.go
  outbox_repo.go

internal/checks/
  registry.go
  runner.go

internal/checks/core/runtime/
internal/checks/core/mcp/
internal/checks/core/repo/
internal/checks/core/speed/
internal/checks/core/memory/
internal/checks/core/safety/

internal/adapters/openclaw/
  detect.go
  register.go
  runner.go
  mcp_bridge_client.go

internal/adapters/mcpclient/
  client.go
  tools_list.go
  schema_hash.go

internal/mcpserver/
  server.go
  tools.go
  schemas.go
  handlers.go

internal/scrub/
  scrubber.go
  rules.go
  entropy.go
  pii.go
  paths.go
  gate.go
  preview.go

internal/diff/
  known_good.go
  observations.go
  scoring.go

internal/report/
  text.go
  json.go

internal/sync/
  client.go
  payload.go
  outbox.go

internal/crypto/
  hash.go
  hmac.go
  token.go

internal/platform/
  exec.go
  fs.go
  git.go
  clock.go
```

Hard rules:

```text
cli        -> app only
mcpserver  -> app only
app        -> domain, checks, store, sync, report
checks     -> domain only; adapters injected
store      -> domain only; never imports cli/mcp/sync
scrub      -> no network, no LLM, no cloud imports
sync       -> only redacted payload structs; no raw artifact access
openclaw   -> only package allowed to shell out to openclaw
mcpclient  -> only package allowed to inspect third-party MCP servers
```

Non-negotiable: **cloud sync must never read raw artifacts directly.** It should only read scrubbed payloads already written into `sync_outbox`.

Do not let `baseline serve mcp` call `os.Stdout` except through the MCP transport. MCP stdio uses stdout for newline-delimited JSON-RPC messages and allows logging on stderr; contaminating stdout will break clients. ([Model Context Protocol][2])

---

## 2. Local SQLite schema

Use SQLite in WAL mode, foreign keys on, short transactions, and migrations. WAL is the right default because SQLite documents that WAL lets readers and writers proceed concurrently and is faster in many scenarios. ([SQLite][3]) Use `STRICT` tables where practical; SQLite’s strict mode enforces declared column types. ([SQLite][4])

Connection setup:

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
```

Minimum local schema:

```sql
CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
) STRICT;

CREATE TABLE workspaces (
  id TEXT PRIMARY KEY,
  local_root_path TEXT NOT NULL,                 -- local only
  repo_root_hash TEXT NOT NULL,
  workspace_fingerprint_hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;

CREATE TABLE agent_profiles (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  agent_kind TEXT NOT NULL,                      -- openclaw | generic-local
  agent_version TEXT,
  model_provider_label TEXT,                     -- local only unless opted in
  model_name_label TEXT,                         -- local only unless opted in
  config_hash TEXT,
  mcp_registry_hash TEXT,
  created_at TEXT NOT NULL
) STRICT;

CREATE TABLE runs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  agent_profile_id TEXT REFERENCES agent_profiles(id) ON DELETE SET NULL,
  trigger TEXT NOT NULL,                         -- cli | mcp | manual
  mode TEXT NOT NULL,                            -- fast | full | pack
  started_at TEXT NOT NULL,
  ended_at TEXT,
  duration_ms INTEGER,
  status TEXT NOT NULL,                          -- ok | warning | critical | failed
  health_score REAL,
  compared_known_good_id TEXT,
  redaction_status TEXT NOT NULL,                -- passed | blocked | unknown
  cloud_sync_state TEXT NOT NULL DEFAULT 'none', -- none | pending | synced | failed
  raw_exported INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
) STRICT;

CREATE INDEX idx_runs_workspace_started
ON runs(workspace_id, started_at DESC);

CREATE TABLE check_results (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  check_id TEXT NOT NULL,
  lane TEXT NOT NULL,                            -- core | configurable | custom
  pack_id TEXT,
  kind TEXT NOT NULL,                            -- runtime | mcp | repo | speed | memory | safety
  runner TEXT NOT NULL,                          -- local | openclaw | mcp-client
  status TEXT NOT NULL,                          -- ok | warning | critical | failed | skipped
  severity INTEGER NOT NULL DEFAULT 0,
  score REAL,
  duration_ms INTEGER,
  input_hash TEXT,
  output_hash TEXT,
  metrics_json TEXT NOT NULL DEFAULT '{}',
  finding_redacted TEXT,
  redaction_json TEXT NOT NULL DEFAULT '{}',
  egress_class INTEGER NOT NULL,                 -- 0 | 1 | 2 | 3
  created_at TEXT NOT NULL,
  CHECK (json_valid(metrics_json)),
  CHECK (json_valid(redaction_json))
) STRICT;

CREATE INDEX idx_check_results_run
ON check_results(run_id);

CREATE INDEX idx_check_results_check
ON check_results(check_id, created_at DESC);

CREATE TABLE observations (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  check_result_id TEXT REFERENCES check_results(id) ON DELETE CASCADE,
  obs_key TEXT NOT NULL,
  value_type TEXT NOT NULL,                      -- bool | number | string_hash | json_hash
  value_hash TEXT,
  numeric_value REAL,
  redacted_display TEXT,
  previous_value_hash TEXT,
  created_at TEXT NOT NULL
) STRICT;

CREATE INDEX idx_observations_run_key
ON observations(run_id, obs_key);

CREATE TABLE known_goods (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  label TEXT,
  note_local TEXT,                               -- never cloud
  created_by TEXT NOT NULL DEFAULT 'local-user',
  created_at TEXT NOT NULL
) STRICT;

CREATE INDEX idx_known_goods_workspace
ON known_goods(workspace_id, created_at DESC);

CREATE TABLE raw_artifacts (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  check_result_id TEXT REFERENCES check_results(id) ON DELETE SET NULL,
  artifact_kind TEXT NOT NULL,                   -- prompt | response | command_output | report
  local_path TEXT NOT NULL,                      -- local only
  sha256 TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  egress_class INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
) STRICT;

CREATE TABLE mcp_server_snapshots (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  server_name_local TEXT,                        -- local only
  server_name_hash TEXT NOT NULL,
  transport TEXT,
  reachable INTEGER NOT NULL,
  tool_count INTEGER NOT NULL DEFAULT 0,
  signature_hash TEXT,                           -- names + input/output schemas
  behavior_hash TEXT,                            -- signature + descriptions + annotations
  error_redacted TEXT,
  created_at TEXT NOT NULL
) STRICT;

CREATE TABLE sync_outbox (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,                      -- run.ingest
  payload_json TEXT NOT NULL,                    -- already scrubbed
  payload_hash TEXT NOT NULL,
  redaction_status TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TEXT,
  last_error_redacted TEXT,
  created_at TEXT NOT NULL,
  CHECK (json_valid(payload_json))
) STRICT;

CREATE INDEX idx_sync_outbox_next
ON sync_outbox(next_attempt_at, attempts);
```

Important schema decisions:

```text
Raw prompts/responses are files, not syncable rows.
sync_outbox contains only already-scrubbed payloads.
known_good note is local-only.
MCP server/tool names are local raw + cloud hash/redacted display.
Use separate signature_hash and behavior_hash.
```

The separate MCP hashes matter. A tool description change can change model behavior even when the JSON schema does not.

---

## 3. MCP protocol risks

Your MCP server must be conservative because MCP tools are model-controlled: the protocol lets models discover and invoke tools automatically, and the spec calls out the need for human approval and clear tool exposure UI. ([Model Context Protocol][5])

Hard MCP constraints:

```text
baseline_latest          read-only, no network
baseline_report          read-only, no raw output by default
baseline_compare         read-only, local unless sync already enabled
baseline_check           local mutation only; default fast
baseline_scrub_preview   local only
baseline_mark_known_good do not commit directly from MCP
baseline_config          get-only in v0, or pending-confirmation only
baseline_sync_status     read-only
```

Be harsher than the current product shape: **do not let an agent mark known-good directly.** That is a footgun. A degraded agent could bless its own bad run and erase the product’s value. Make `baseline_mark_known_good` return a pending confirmation command:

```text
Pending known-good mark created.

Confirm locally:
baseline known-good confirm kg_pending_01HY...
```

Same for config changes. In v0, MCP can read config; CLI changes config.

Specific MCP traps:

1. **A passive MCP server cannot evaluate the host model.** MCP servers expose tools/resources/prompts; clients may offer sampling, but sampling is client-side, capability-negotiated, and requires user control. ([Model Context Protocol][6]) Generic MCP clients get local environment/tool/repo drift only.

2. **`tools/list` is paginated.** The spec shows `tools/list` with a cursor and `nextCursor`; schema hashing must fetch all pages, guard against duplicate cursors, and stop at a max page count. ([Model Context Protocol][5])

3. **`notifications/tools/list_changed` is not enough.** Servers may send it when tools change, but Baseline should poll and hash during every check. Treat notifications as hints, not history. ([Model Context Protocol][5])

4. **Annotations are not safety controls.** MCP’s own blog says `readOnlyHint`, `destructiveHint`, `idempotentHint`, and `openWorldHint` are hints and untrusted unless from a trusted server. ([Model Context Protocol Blog][7]) Use them for display and policy hints, but enforce policy in Baseline code.

5. **Tool names need stable, valid names.** The spec recommends names of 1–128 chars using letters, digits, underscore, hyphen, and dot, with no spaces or special chars. Your proposed names are fine. ([Model Context Protocol][5])

6. **Every MCP response must be scrubbed twice.** Scrub before storing the finding and again before returning it to the MCP client. Tool outputs are fed back into models; the tools spec explicitly requires validation, access control, rate limiting, and output sanitization. ([Model Context Protocol][5])

7. **Use structured output schemas.** The tools spec supports `structuredContent` and output schemas; return both structured JSON and a short text summary so clients can parse safely. ([Model Context Protocol][5])

---

## 4. OpenClaw install and runner path

Do not confuse the two OpenClaw directions:

```text
baseline serve mcp
  Baseline acts as MCP server.
  OpenClaw may call Baseline tools.

openclaw mcp serve
  OpenClaw acts as MCP server.
  Baseline can act as MCP client to inspect/send OpenClaw conversations.
```

OpenClaw docs say `openclaw mcp serve` runs OpenClaw as a stdio MCP server, while `openclaw mcp list/show/set/unset` manage OpenClaw-owned outbound MCP server definitions. ([OpenClaw][8])

`baseline init` should do this:

```bash
baseline init
```

Implementation sequence:

```text
1. Locate baseline absolute path via os.Executable().
2. Locate openclaw via command -v openclaw.
3. Read openclaw version if available.
4. Create ~/.baseline.
5. Create SQLite DB and config files.
6. Run scrubber self-test.
7. Register Baseline as an OpenClaw MCP server.
8. Verify registration by reading OpenClaw config.
9. Run baseline doctor.
10. Do not enable cloud sync.
```

Registration command:

```bash
openclaw mcp set baseline '{"command":"/ABS/PATH/baseline","args":["serve","mcp"]}'
```

Use an absolute path. Do **not** rely on `PATH`; MCP clients often launch subprocesses with a thinner environment.

OpenClaw’s docs show `openclaw mcp set <name> <json>` with stdio `command` and `args`, and they explicitly say these registry commands only read/write config; they do not connect to or validate the target server. ([OpenClaw][8]) Therefore `baseline init` must run its own verification after registration.

`baseline doctor` should test:

```text
baseline binary executable
baseline serve mcp initialize works
OpenClaw binary found
OpenClaw mcp registry contains baseline
OpenClaw registry command is absolute path
SQLite writable
WAL enabled
redaction rules load
sync disabled unless explicitly enabled
```

Runner modes:

```text
generic-local
  runtime + repo + local MCP drift only

openclaw-inspect
  reads OpenClaw conversations/tool state if available

openclaw-prompt-safe
  behavior checks only if OpenClaw can run prompt-only/tool-disabled or equivalent safe mode

openclaw-tool-enabled
  cut from v0
```

Do **not** send behavior prompts into a tool-enabled coding agent unless you can disable tools or guarantee a scratch, non-mutating harness. A prompt that says “do not use tools” is not a security control. For launch, memory/context checks should run only in safe prompt-only mode; otherwise mark them skipped with a clear finding.

OpenClaw’s MCP bridge exposes conversation and message tools such as `conversations_list`, `messages_read`, `events_poll`, `events_wait`, and `messages_send`, but it also notes live queue state only exists while the bridge is connected. ([OpenClaw][8]) So the runner must handle missed events by reading transcript history, not just waiting for live events.

---

## 5. Cloudflare / Neon / Stripe shape

Be blunt: **do not try to run the launch API as a Go service on Cloudflare Workers.** Keep the local binary in Go. For Cloudflare, use TypeScript Workers or Pages Functions. Cloudflare lists JavaScript, TypeScript, Python, and Rust as first-class Worker languages, while Go is possible through Wasm; that is not the launch path you want. ([Cloudflare Docs][9])

Recommended cloud split:

```text
Cloudflare Pages
  landing
  dashboard shell
  pricing page
  docs

Cloudflare Worker API
  auth/session
  token management
  run ingest
  report reads
  compare reads
  Stripe Checkout session creation
  Stripe webhook handling

Neon Postgres
  cloud history
  accounts/workspaces/tokens/runs/checks
  Stripe subscription state

Stripe
  hosted Checkout
  hosted billing portal later
  webhooks as source of truth
```

Cloudflare Workers Free has a 100,000 requests/day limit and a 10 ms CPU limit per invocation; Standard has larger included usage and higher CPU allowances. ([Cloudflare Docs][10]) That means the Worker API should validate, scrub-check metadata, authorize, insert, and return. It should not run scoring, browser checks, LLM judging, or heavy summaries.

Use Workers Secrets for `DATABASE_URL`, Stripe keys, webhook secrets, and HMAC keys. Cloudflare docs distinguish plain environment variables from secrets: text/JSON env vars are not encrypted, while secret values are hidden after definition. ([Cloudflare Docs][11])

Use Neon through Cloudflare Hyperdrive. Cloudflare’s Neon integration docs recommend Hyperdrive or the Neon serverless driver, with Hyperdrive recommended for pooling and lower latency from Workers. ([Cloudflare Docs][12])

Important Neon correction: **your 14-day free history is app-level retention, not Neon retention.** Neon Free currently lists 100 CU-hours/month per project and 0.5 GB storage per project, while Launch is usage-based; Neon restore windows are 6 hours on Free, 7 days on Launch, and 30 days on Scale. ([Neon][13]) Store `expires_at` per run and purge by your own policy.

Minimum Worker routes:

```text
POST /v1/runs/ingest
GET  /v1/runs
GET  /v1/runs/:id
GET  /v1/compare
GET  /v1/sync/status

POST /v1/tokens
DELETE /v1/tokens/:id

POST /v1/billing/checkout
POST /v1/billing/portal       # can be added after launch
POST /v1/stripe/webhook
```

Minimum cloud Postgres tables:

```sql
accounts (
  id,
  email,
  stripe_customer_id,
  plan_key,
  retention_days,
  created_at,
  updated_at
)

workspaces (
  id,
  account_id,
  workspace_fingerprint_hash,
  repo_root_hash,
  display_name_redacted,
  created_at,
  updated_at
)

api_tokens (
  id,
  workspace_id,
  token_prefix,
  token_hash,
  scopes,
  last_seen_at,
  revoked_at,
  created_at
)

runs (
  id,
  workspace_id,
  client_run_id,
  client_version,
  mode,
  trigger,
  started_at,
  duration_ms,
  status,
  health_score,
  redaction_status,
  raw_exported,
  known_good_client_run_id,
  payload_hash,
  expires_at,
  created_at
)

check_results (
  id,
  run_id,
  check_id,
  lane,
  kind,
  runner,
  status,
  severity,
  score,
  duration_ms,
  metrics_jsonb,
  finding_redacted,
  redaction_jsonb,
  egress_class,
  created_at
)

observations (
  id,
  run_id,
  check_id,
  obs_key,
  value_type,
  value_hash,
  numeric_value,
  redacted_display,
  created_at
)

known_goods (
  id,
  workspace_id,
  client_run_id,
  label_redacted,
  created_at
)

stripe_events (
  id,
  stripe_event_id,
  event_type,
  payload_hash,
  processed_at,
  created_at
)

subscriptions (
  id,
  account_id,
  stripe_customer_id,
  stripe_subscription_id,
  status,
  price_id,
  plan_key,
  current_period_end,
  updated_at
)
```

Stripe shape:

```text
One product at launch: Baseline Pro
One monthly price
Maybe one annual price
No team billing before launch
No usage billing before launch
```

Create a Checkout Session server-side. Stripe describes Checkout Sessions as the object representing a customer’s payment or subscription session and recommends creating a new Session for each payment attempt. ([Stripe Docs][14]) For Pro, use `mode=subscription`.

Do not grant paid access from the success redirect. Grant access only after a verified webhook. Stripe recommends verifying webhook signatures with the `Stripe-Signature` header and endpoint secret. ([Stripe Docs][15])

Use Stripe metadata only to carry internal IDs like `account_id` and `workspace_id`; Stripe metadata is for your own structured information and is not used by Stripe for authorization. ([Stripe Docs][16])

---

## 6. API/token/security constraints

API token format:

```text
bl_live_<8-char-prefix>_<32+ bytes random base64url>
```

Store only:

```text
token_prefix
token_hash = HMAC-SHA256(server_secret, full_token)
scopes
workspace_id
last_seen_at
revoked_at
```

Scopes:

```text
ingest:write
reports:read
alerts:write
billing:write
```

Hard reject ingest if:

```text
token missing/invalid
scope missing
payload too large
client_run_id already exists with different payload_hash
redaction_status != passed
raw_exported == true
egress_class > 2
finding text contains high-confidence secret
payload schema version unsupported
```

Payload limit:

```text
target payload: < 64 KB per run
hard reject: > 512 KB per run
```

No raw cloud storage. No repo names. No file names by default. No custom prompt text. No raw tool outputs.

---

## 7. Performance targets

Use these as launch gates, not aspirations.

Local CLI:

```text
baseline --help                 p95 < 150 ms
baseline latest                 p95 < 250 ms
baseline report                 p95 < 750 ms
baseline scrub preview 1 MB     p95 < 300 ms
baseline scrub preview 10 MB    p95 < 2 s
baseline check --fast           p50 < 5 s, p95 < 15 s, hard timeout 30 s
baseline check --full           p50 < 45 s, p95 < 90 s, hard timeout 120 s
```

MCP server:

```text
stdio initialize                p95 < 500 ms
baseline_latest tool            p95 < 300 ms
baseline_report tool            p95 < 1 s
baseline_compare tool           p95 < 2 s local
baseline_check tool             hard cooldown + timeout
```

Memory:

```text
idle MCP server RSS             < 50 MB target
fast check peak RSS             < 100 MB target
full check peak RSS             < 150 MB target
```

Cloud API:

```text
POST /v1/runs/ingest            p95 < 500 ms excluding cold DB wake
GET /v1/runs                    p95 < 500 ms
GET /v1/compare                 p95 < 1.5 s
Stripe webhook handler          p95 < 500 ms
```

Behavioral rule:

```text
Fast check never waits on cloud.
Fast check never runs OpenClaw prompts.
Full check skips unsafe/unavailable runner instead of blocking.
Sync failure never fails local check.
```

Cloudflare Workers’ CPU limits make this split mandatory; the edge API should stay thin and push expensive work back to the local CLI or later workers/queues. ([Cloudflare Docs][17])

---

## 8. Scrubber failure modes

This is the trust-critical system. Assume it will fail unless you design it to fail closed.

Failure modes and required behavior:

| Failure mode                                    | Required behavior                                                        |
| ----------------------------------------------- | ------------------------------------------------------------------------ |
| Unknown redaction status                        | Block cloud upload                                                       |
| High-confidence secret found in finding text    | Redact; if uncertain, replace whole finding with generic blocked finding |
| Secret found in raw prompt/response             | Cloud gets counts only, no derived finding text from that artifact       |
| Binary/invalid UTF-8 data                       | Do not upload content; store local artifact hash only                    |
| Base64/URL-encoded token                        | Detect common encodings; if suspected, block Class 2 text                |
| JWT/API key split across chunks                 | Use overlapping scan windows                                             |
| Absolute path leak                              | Redact username, repo path, home dir, temp dirs                          |
| Repo/client name leak through tool/server names | Hash names for cloud; local report may show raw                          |
| Hash dictionary attack on short names           | Use HMAC, not plain SHA, for cloud identity hashes                       |
| LLM summary reintroduces secret                 | Do not use LLM summarization in v0                                       |
| Custom prompt text leaks through finding        | Store prompt hash; show generic pack/check ID                            |
| `.env`/secret file accidentally read            | Denylist file globs before reading                                       |
| User-defined deny glob not applied              | Treat as scrubber failure, block upload                                  |
| Redaction rule crash                            | Mark `redaction_status=unknown`, block upload                            |
| Oversized payload                               | Block upload and keep local-only                                         |
| PII false positive                              | Prefer over-redaction to leakage                                         |
| PII false negative                              | Never claim PII-free; claim “scrubber passed configured rules”           |

Use GitHub-style provider patterns as inspiration, but do not pretend regex coverage is complete. GitHub’s own secret scanning docs emphasize supported patterns, provider patterns, validation, and push protection rather than universal detection. ([GitHub Docs][18])

Scrubber output should look like this:

```json
{
  "status": "passed",
  "secrets_found": 0,
  "pii_found": 2,
  "path_redactions": 4,
  "rules_version": "2026.05.13",
  "cloud_safe": true,
  "blocked_reason": null
}
```

Never store matched secret values in SQLite. Store only:

```text
rule_id
class
count
confidence
redacted span length
artifact hash
```

---

## 9. Known-good diff constraints

Known-good diff is the product. Treat health score as secondary.

Diff engine should compare observations, not prose:

```text
runtime.agent.kind
runtime.agent.version
runtime.model.hash
runtime.config.hash
mcp.server.<hash>.reachable
mcp.server.<hash>.tool_count
mcp.server.<hash>.signature_hash
mcp.server.<hash>.behavior_hash
repo.branch.hash
repo.head.hash
repo.dirty_count
repo.untracked_count
speed.cli.duration_ms
speed.mcp.tool_list_ms
memory.user.score
memory.project.score
safety.cloud_sync.enabled
safety.scrubber.status
```

Report format:

```text
Changed since known-good:
1. GitHub MCP signature changed: 1 tool missing.
2. OpenClaw response latency is 2.3x your 7-day median.
3. Agent missed project owner in safe memory check.
4. Cloud sync is still off. Raw data remains local.
```

Do not overfit the score. A user trusts concrete deltas.

---

## 10. What to cut before launch

Cut these now:

```text
browser checks
workflow execution
cloud LLM-as-judge scoring
public benchmarks
benchmark contribution UI
team RBAC
Slack/GitHub/email alert delivery
scheduled cloud summaries
OpenTelemetry exporter
raw transcript cloud storage
auto-known-good
multi-agent adapters
language-specific plugins
full custom eval builder
custom pack cloud sync
test command execution by default
tool-enabled OpenClaw behavior probes
MCP config mutation
MCP direct known-good mutation
Stripe Team/Agency plans
usage billing
complex dashboard filters
```

Keep only:

```text
Go CLI/MCP
SQLite local history
OpenClaw registration + doctor
OpenClaw safe runner fallback
generic local fallback
scrubber + preview
known-good mark/list/compare
redacted sync outbox
Cloudflare landing/dashboard/API
Neon redacted history
Stripe Checkout Pro
14-day free cloud retention by app policy
```

The uncomfortable cut: **alerts should be local/report-only before launch.** Paid “alerts” can be a pricing promise after users ask for them, but external alert delivery before the core drift report is trusted will create noise and support burden.

---

## 11. Build order

Implementation order should be:

```text
1. SQLite migrations + run/check/observation storage
2. scrubber + scrub preview + fail-closed gate
3. baseline check --fast
4. known-good mark/list/compare
5. report/latest renderers
6. MCP server with read-heavy tools
7. OpenClaw init/register/doctor
8. OpenClaw safe runner, skipped if unsafe
9. sync outbox
10. Cloudflare ingest API + Neon schema
11. dashboard latest/timeline/compare
12. Stripe Checkout + webhook entitlement update
```

Do not start with the dashboard. Start with a local report that would still be worth using with sync permanently off.

The launch pass/fail test is simple:

```text
Can a user install it, run baseline check, mark known-good,
run compare later, and see one concrete useful change without trusting cloud?
```

If not, cut more.

[1]: https://modelcontextprotocol.io/docs/sdk "SDKs - Model Context Protocol"
[2]: https://modelcontextprotocol.io/specification/2025-11-25/basic/transports?utm_source=chatgpt.com "Transports"
[3]: https://sqlite.org/wal.html?utm_source=chatgpt.com "Write-Ahead Logging"
[4]: https://www.sqlite.org/stricttables.html?utm_source=chatgpt.com "STRICT Tables"
[5]: https://modelcontextprotocol.io/specification/2025-11-25/server/tools "Tools - Model Context Protocol"
[6]: https://modelcontextprotocol.io/specification/2025-11-25 "Specification - Model Context Protocol"
[7]: https://blog.modelcontextprotocol.io/posts/2026-03-16-tool-annotations/ "Tool Annotations as Risk Vocabulary: What Hints Can and Can't Do | Model Context Protocol Blog"
[8]: https://docs.openclaw.ai/cli/mcp "mcp - OpenClaw"
[9]: https://developers.cloudflare.com/workers/languages/?utm_source=chatgpt.com "Languages - Workers"
[10]: https://developers.cloudflare.com/workers/platform/limits/?utm_source=chatgpt.com "Limits · Cloudflare Workers docs"
[11]: https://developers.cloudflare.com/workers/configuration/environment-variables/?utm_source=chatgpt.com "Environment variables · Cloudflare Workers docs"
[12]: https://developers.cloudflare.com/workers/databases/third-party-integrations/neon/ "Neon · Cloudflare Workers docs"
[13]: https://neon.com/pricing "Pricing — Neon"
[14]: https://docs.stripe.com/api/checkout/sessions?utm_source=chatgpt.com "Checkout Sessions | Stripe API Reference"
[15]: https://docs.stripe.com/webhooks/signature?utm_source=chatgpt.com "Resolve webhook signature verification errors"
[16]: https://docs.stripe.com/api/metadata?utm_source=chatgpt.com "Metadata | Stripe API Reference"
[17]: https://developers.cloudflare.com/workers/platform/pricing/?utm_source=chatgpt.com "Pricing · Cloudflare Workers docs"
[18]: https://docs.github.com/en/code-security/reference/secret-security/supported-secret-scanning-patterns?utm_source=chatgpt.com "Supported secret scanning patterns"
