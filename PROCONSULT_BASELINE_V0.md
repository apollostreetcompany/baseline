## Blunt verdict

The direction is good, but the current shape is still too broad. It is flirting with “LLM observability,” “eval framework,” “benchmark network,” “agent memory test,” “dashboard,” and “workflow runner” all at once. That will dilute trust and slow shipping.

The wedge should be narrower:

> **Baseline is `git status` + smoke test + known-good diff for OpenClaw/coding-agent setups.**

Not “evals.” Not “personality.” Not “observability.” Not “AI monitoring.” The user should feel: **“Before I let this Claw touch my repo, I want Baseline green.”**

One hard correction: a passive MCP server cannot magically evaluate an agent’s answers unless the host supports a model-calling path, sampling, or an OpenClaw runner path. Baseline v0 must be honest about this. The agent-behavior checks should be **OpenClaw-runner-first**, and generic MCP clients should fall back to local environment/tool drift only. OpenClaw already has an MCP bridge and client registry shape, including `openclaw mcp serve`, `openclaw mcp set`, and conversation/message tools, so use that instead of pretending generic MCP is enough. ([OpenClaw][1])

Also: Go is the right default. MCP lists Go as a Tier 1 official SDK, and the official Go SDK supports MCP client/server construction, stdio transport, and the current MCP spec family. ([Model Context Protocol][2])

---

# 1) Smallest irresistible v0

## Product promise

**“Tell me if my Claw is normal today, compared with a known-good run.”**

That is the whole v0.

The smallest irresistible v0 is a single Go binary:

```bash
baseline init
baseline check
baseline report
baseline compare
baseline known-good mark
baseline sync on
baseline serve mcp
```

The user’s first useful moment should look like this:

```text
Baseline: WARNING · health 73/100

Changed since known-good:
1. GitHub MCP schema changed: pull_request_review disappeared.
2. Claw latency is 2.4x slower than your 7-day median.
3. Current Claw no longer recalls project owner: expected “Future”, got “unknown”.
4. Repo awareness degraded: missed dirty file src/agent/runner.go.

Suggested next step:
Run: baseline compare --known-good --explain
```

## Required v0 checks

Cut to **six core capsules**.

### A. Runtime capsule

Detect:

```text
agent/runtime
model/provider if visible
OpenClaw version
agent config hash
permission/sandbox mode if visible
active workspace fingerprint
```

This catches silent model/runtime/config changes.

### B. MCP/tool capsule

Detect:

```text
configured MCP servers
server reachability
tool list hash
tool schema hash
missing/new tools
auth failure
transport type
```

Important: tool/schema listing must handle pagination. MCP `tools/list` supports `nextCursor`, so a one-page schema hash is not defensible. ([Model Context Protocol][3])

### C. Repo capsule

Detect:

```text
git branch
HEAD commit hash
dirty/untracked count
repo language/package markers
test command candidates
recent test result if known
```

Do **not** run arbitrary tests by default. Detect only. Running tests becomes a toggle.

### D. Speed capsule

Measure:

```text
baseline CLI check duration
MCP echo/tool-list latency
OpenClaw prompt-response latency if runner available
tool retry/timeouts if observable
```

Do not overfit to provider latency. Users care about **end-to-end waiting time**.

### E. Memory/context capsule

Ask only four questions through the OpenClaw runner path:

```text
Who is the user?
What project is this?
What is the active task?
What constraints should you not violate?
```

Score against known-good facts and current profile. This is where the product becomes lovable.

### F. Safety/egress capsule

Report:

```text
scrubber enabled?
cloud sync enabled?
raw output local-only?
secrets detected?
PII detected?
benchmark contribution enabled?
```

Trust is a feature. The safety capsule should be visible every run.

## Configuration model

You are right that configuration should be toggles only.

Use exactly three lanes:

```toml
[core]
enabled = true # immutable

[configurable.fact_checks]
enabled = true
frequency = "daily"      # every_run | daily | manual
sensitivity = "normal"   # low | normal | high

[configurable.style_checks]
enabled = false
frequency = "manual"
sensitivity = "normal"

[configurable.repo_awareness]
enabled = true
frequency = "every_run"
sensitivity = "normal"

[custom]
enabled = false
packs = []
```

No arbitrary threshold editing in v0. No UI builder. No deeply nested config.

## What makes it irresistible

The killer feature is **known-good diff**, not scoring.

A raw score is weak:

```text
health_score = 73
```

A diff is useful:

```text
Since Monday’s known-good:
- Model changed from claude-sonnet-* to unknown.
- GitHub MCP has 4 fewer tools.
- Claw forgot your “do not auto-push” rule.
- Agent response is 2.4x slower.
```

That is what users will show friends.

---

# 2) Exact MCP/CLI surface

## CLI surface

Ship this and nothing else.

```bash
baseline init
baseline check
baseline report
baseline latest
baseline compare
baseline known-good mark
baseline known-good list
baseline sync on
baseline sync off
baseline sync status
baseline serve mcp
baseline doctor
baseline scrub preview
```

### `baseline init`

Purpose: create local profile, detect OpenClaw, register MCP server, create SQLite DB.

```bash
baseline init
baseline init --no-openclaw
baseline init --sync-token bl_live_...
```

What it does:

```text
1. Detect OpenClaw.
2. Register Baseline as an OpenClaw MCP server if possible.
3. Detect whether OpenClaw runner checks are possible.
4. Create ~/.baseline/baseline.db.
5. Create ~/.baseline/config.toml.
6. Create ~/.baseline/redaction.toml.
7. Run scrubber self-test.
8. Print exact cloud egress preview before sync.
```

For OpenClaw registration, the command should write something equivalent to:

```bash
openclaw mcp set baseline '{"command":"baseline","args":["serve","mcp"]}'
```

OpenClaw’s docs describe `mcp set` as a way to store MCP server definitions in OpenClaw config, and show stdio definitions with `command` and `args`. ([OpenClaw][1])

### `baseline check`

```bash
baseline check
baseline check --fast
baseline check --full
baseline check --pack fact_checks
baseline check --no-sync
baseline check --json
```

Modes:

```text
--fast  = runtime + MCP + repo + scrubber + latest known-good diff
--full  = fast + memory/context + enabled configurable packs
--pack  = run one configurable/custom pack
```

Default should be `--fast`.

### `baseline report`

```bash
baseline report
baseline report --run run_01HY...
baseline report --local
baseline report --json
```

Returns the latest human-readable report.

### `baseline latest`

```bash
baseline latest
baseline latest --json
```

Tiny one-screen summary for agents and terminal.

### `baseline compare`

```bash
baseline compare
baseline compare --known-good
baseline compare --days 7
baseline compare --days 14
baseline compare --from run_a --to run_b
baseline compare --dimension tools
baseline compare --dimension memory
baseline compare --json
```

Free: local history and 14-day cloud window.

Paid: cloud windows beyond 14 days, team/workspace compare, time summaries.

### `baseline known-good`

```bash
baseline known-good mark
baseline known-good mark --run run_01HY... --label "pre-agent-rollout"
baseline known-good list
```

Do not auto-mark known-good. Manual marking matters.

### `baseline sync`

```bash
baseline sync on --token bl_live_...
baseline sync off
baseline sync status
baseline sync flush
```

Cloud sync should be opt-in. Do not auto-sync during `init`. The dashboard is not worth a trust hit.

### `baseline scrub preview`

```bash
baseline scrub preview
baseline scrub preview --file ./tmp/agent-output.txt
baseline scrub preview --stdin
```

This is not optional. Users should be able to see exactly what would leave the machine.

---

## MCP tools

Keep the MCP surface small and mostly read-heavy.

| Tool                        |        Tier |    Mutates state? | Purpose                                              |
| --------------------------- | ----------: | ----------------: | ---------------------------------------------------- |
| `baseline_check`            |        Free |     Yes, local DB | Run a local check and return redacted summary        |
| `baseline_latest`           |        Free |                No | Return latest score + top findings                   |
| `baseline_report`           |        Free |                No | Return latest or selected run report                 |
| `baseline_compare`          |    Free/Pro |                No | Compare latest vs known-good or time window          |
| `baseline_mark_known_good`  |        Free |     Yes, local DB | Mark run as anchor                                   |
| `baseline_config`           |        Free | Yes, local config | Get/set lane toggles only                            |
| `baseline_sync_status`      |        Free |                No | Show sync/egress state                               |
| `baseline_alerts`           |         Pro | Yes, cloud config | Preview/configure alerts                             |
| `baseline_workflow_compare` |    Pro/Team |    Yes if execute | Run/dry-run prose workflow package and compare drift |
| `baseline_benchmarks`       |       Team+ |                No | Compare against anonymized aggregates                |
| `baseline_browser_check`    | Higher tier |      Yes/external | Run browser/tool workflow probes                     |

### Exact tool schemas

#### `baseline_check`

```json
{
  "mode": "fast|full|pack",
  "packs": ["fact_checks"],
  "sync": false,
  "return_format": "summary|json"
}
```

Return:

```json
{
  "run_id": "run_01HY...",
  "status": "ok|warning|critical",
  "health_score": 73,
  "top_findings": [],
  "cloud_synced": false,
  "redaction": {
    "scrubbed": true,
    "raw_exported": false,
    "secrets_found": 0,
    "pii_found": 1
  }
}
```

#### `baseline_latest`

```json
{
  "workspace": "current",
  "include_findings": true
}
```

#### `baseline_report`

```json
{
  "run_id": "latest",
  "format": "summary|json",
  "include_raw": false
}
```

`include_raw` should always return `false` unless local-only policy explicitly permits it.

#### `baseline_compare`

```json
{
  "from": "known_good|7d|14d|30d|run_id",
  "to": "latest",
  "dimensions": ["runtime", "tools", "repo", "speed", "memory", "style", "safety"]
}
```

#### `baseline_mark_known_good`

```json
{
  "run_id": "latest",
  "label": "optional-human-label",
  "note": "optional-local-only-note"
}
```

#### `baseline_config`

```json
{
  "action": "get|set_toggles",
  "toggles": {
    "fact_checks": true,
    "style_checks": false,
    "repo_awareness": true,
    "browser_checks": false,
    "custom": false
  }
}
```

No arbitrary prompt editing through MCP in v0.

#### `baseline_workflow_compare`

```json
{
  "pack_id": "custom.prose.offer_smoke",
  "mode": "dry_run|execute",
  "variables": {
    "workflow": "./recipes/03-offer-smoke-test.prose.md",
    "task": "compare expected actions to agent response"
  },
  "compare_against": [
    "expected_steps",
    "missing_actions",
    "invented_actions",
    "repo_changes",
    "agent_response"
  ]
}
```

Default must be `dry_run`.

### MCP annotations

Use annotations, but never depend on them for safety. The MCP spec explicitly says tool annotations are hints, not guarantees, and clients should not make security decisions based on untrusted annotations. ([Model Context Protocol][3])

Suggested annotations:

| Tool                        | `readOnlyHint` | `destructiveHint` | `idempotentHint` |            `openWorldHint` |
| --------------------------- | -------------: | ----------------: | ---------------: | -------------------------: |
| `baseline_latest`           |           true |             false |             true |                      false |
| `baseline_report`           |           true |             false |             true |                      false |
| `baseline_compare`          |           true |             false |             true |            cloud-dependent |
| `baseline_check`            |          false |             false |            false | false unless cloud/browser |
| `baseline_mark_known_good`  |          false |             false |            false |                      false |
| `baseline_config`           |          false |             false |            false |                      false |
| `baseline_alerts`           |          false |             false |            false |                       true |
| `baseline_benchmarks`       |           true |             false |             true |                       true |
| `baseline_browser_check`    |          false |          true-ish |            false |                       true |
| `baseline_workflow_compare` |          false |   depends on mode |            false |            depends on mode |

The MCP blog’s guidance is also aligned: use annotations for UX, but keep actual safety guarantees in deterministic controls. ([Model Context Protocol Blog][4])

---

# 3) Minimum infra

## Local infra

Minimum local stack:

```text
single Go binary
SQLite database
stdio MCP server
OpenClaw runner adapter
generic local adapter
redaction engine
config files
local report renderer
```

Filesystem:

```text
~/.baseline/
  baseline.db
  config.toml
  redaction.toml
  tokens.toml          # chmod 0600
  reports/
    run_*.json
  raw/
    run_*/             # local-only, optional
```

Repo-local optional file:

```text
.baseline/profile.toml
```

But keep it tiny. No generated junk.

### Local database

Use SQLite with WAL mode.

```text
runs
checks
observations
known_goods
packs
pack_runs
sync_outbox
```

### Local runner adapters

Ship only two in v0:

```text
openclaw
generic-local
```

Do not ship Claude Code, Cursor, Windsurf, Aider, Goose, OpenHands adapters in v0. Detect them, report them, but do not deeply integrate them yet.

OpenClaw is special because it already exposes MCP conversation/message tools and an MCP server registry. ([OpenClaw][1])

## Cloud infra

Minimum cloud stack:

```text
Go API service
Neon Postgres
static or server-rendered dashboard
API token auth
daily retention purge job
```

No queue in v0. No workers unless alerts force it. The CLI can batch-upload redacted run summaries through a simple outbox.

API endpoints:

```http
POST /v1/runs
GET  /v1/workspaces
GET  /v1/workspaces/:id/runs
GET  /v1/runs/:id
GET  /v1/compare
GET  /v1/sync/status
POST /v1/alerts/preview
POST /v1/alerts/rules
GET  /v1/benchmarks
```

Dashboard views:

```text
1. Latest health
2. Timeline
3. Compare to known-good
4. Token/settings
```

Nothing else.

## Neon reality check

Use Neon, but do not confuse your product’s 14-day free retention with Neon’s own plan features. As of the current Neon pricing page, the Free plan lists 100 projects, 100 CU-hours monthly per project, and 0.5 GB storage per project; paid Launch lists usage-based compute and storage, and Scale lists 14-day metrics/logs in the Neon UI. ([Neon][5])

For dogfood, Neon Free is fine. For external beta, use paid Neon Launch or Scale with hard retention caps. “14 days free history” should be enforced by your app:

```sql
DELETE FROM runs
WHERE plan = 'free'
AND started_at < now() - interval '14 days';
```

---

# 4) Eval data model

The data model should make **core checks, configurable packs, custom prompt packs, and paid workflow comparisons all look like the same thing**.

## Core entities

### `workspace`

```json
{
  "workspace_id": "ws_01HY...",
  "local_fingerprint_hash": "sha256:...",
  "repo_root_hash": "sha256:...",
  "created_at": "2026-05-13T00:00:00Z",
  "cloud_retention_days": 14,
  "benchmark_contribution": false
}
```

Do not upload raw repo path or repo name by default.

### `agent_profile`

```json
{
  "agent_profile_id": "ap_01HY...",
  "workspace_id": "ws_01HY...",
  "agent_kind": "openclaw",
  "agent_version": "detected-or-unknown",
  "model_provider_hash": "sha256:...",
  "model_name_hash": "sha256:...",
  "config_hash": "sha256:...",
  "mcp_registry_hash": "sha256:..."
}
```

Cloud can store provider/model as redacted labels only if user opts in.

### `run`

```json
{
  "run_id": "run_01HY...",
  "workspace_id": "ws_01HY...",
  "agent_profile_id": "ap_01HY...",
  "trigger": "cli|mcp|scheduled|manual",
  "mode": "fast|full|pack",
  "started_at": "2026-05-13T12:00:00Z",
  "duration_ms": 18420,
  "status": "ok|warning|critical|failed",
  "health_score": 73,
  "known_good_compared_to": "kg_01HY...",
  "redaction_status": "passed|blocked|unknown",
  "cloud_synced": true,
  "raw_exported": false
}
```

### `check_result`

```json
{
  "check_result_id": "cr_01HY...",
  "run_id": "run_01HY...",
  "check_id": "core.memory.project",
  "lane": "core|configurable|custom",
  "pack_id": "openclaw.default",
  "kind": "runtime|mcp|repo|speed|memory|style|fact|workflow|safety",
  "runner": "local|openclaw|browser|workflow",
  "status": "ok|warning|critical|failed|skipped",
  "severity": 2,
  "score": 0.72,
  "duration_ms": 1420,
  "input_hash": "sha256:...",
  "output_hash": "sha256:...",
  "metrics": {
    "latency_ratio": 2.4,
    "fact_recall": 0.7,
    "invented_fact_penalty": 0.1
  },
  "finding": "Agent identified the project but missed the owner.",
  "redaction": {
    "secrets_found": 0,
    "pii_found": 1,
    "cloud_safe": true
  },
  "egress_class": "summary_only"
}
```

### `observation`

Use this for diffable facts.

```json
{
  "observation_id": "obs_01HY...",
  "check_result_id": "cr_01HY...",
  "key": "mcp.github.tool.pull_request_review.present",
  "value_type": "bool|number|string_hash|json_hash",
  "value_hash": "sha256:...",
  "numeric_value": null,
  "redacted_display": "tool disappeared",
  "previous_value_hash": "sha256:..."
}
```

### `known_good`

```json
{
  "known_good_id": "kg_01HY...",
  "workspace_id": "ws_01HY...",
  "run_id": "run_01HY...",
  "label": "before-agent-rollout",
  "created_at": "2026-05-13T12:30:00Z",
  "created_by": "local-user"
}
```

### `pack`

```json
{
  "pack_id": "custom.prose.offer_smoke",
  "lane": "custom",
  "version": "0.1.0",
  "manifest_hash": "sha256:...",
  "source": "local|synced|baseline",
  "safety_class": "prompt_only|dry_run|exec_required|browser_required"
}
```

### `pack_run`

```json
{
  "pack_run_id": "pr_01HY...",
  "run_id": "run_01HY...",
  "pack_id": "custom.prose.offer_smoke",
  "mode": "dry_run|execute",
  "variables_hash": "sha256:...",
  "workflow_hash": "sha256:...",
  "repo_diff_hash": "sha256:...",
  "agent_response_hash": "sha256:...",
  "contract_score": 0.81
}
```

## Scoring rule

Do not expose a fake-scientific personality score.

Expose:

```text
Health score
Severity
Diff from known-good
Dimension-specific findings
```

Dimension weights for v0:

```text
runtime/tool health      25%
repo awareness           15%
speed                    15%
memory/context           25%
safety/egress            10%
configurable/custom      10%
```

Style checks should be secondary. Users will like them, but they are noisy and easy to overclaim.

## OpenTelemetry shape

Keep the schema OTel-shaped, but do not ship an exporter in v0. OpenTelemetry’s GenAI semantic conventions are still marked “Development,” so use the concepts without binding the product to an unstable exporter surface yet. ([OpenTelemetry][6])

Mapping:

```text
run          = trace
check        = span
observation  = span attribute or event
finding      = event
score        = metric
latency      = metric
```

Cut the actual OTel exporter until users ask for it.

---

# 5) Safety/egress model

This product lives or dies on trust.

## Default policy

```toml
mode = "local"
cloud_sync = false
raw_outputs = "local_only"
benchmark_contribution = false
browser = false
workflow_execution = false
external_alerts = false
```

Cloud sync only after:

```bash
baseline sync on --token bl_live_...
```

Before enabling sync, show:

```text
Will upload:
- run IDs
- timestamps
- scores
- durations
- check IDs
- redacted findings
- hashes
- scrubber counts
- runtime labels if allowed

Will not upload:
- raw prompts
- raw responses
- code
- file contents
- repo names
- file names
- absolute paths
- usernames
- secrets
- exact custom prompt text
```

## Egress classes

Use four classes.

### Class 0: Local raw

```text
raw prompts
raw responses
raw workflow text
raw command output
raw file paths
repo-local notes
```

Never cloud by default.

### Class 1: Cloud-safe metrics

```text
duration
score
status
check ID
tool count
dirty file count
latency ratio
hashes
```

Allowed after sync.

### Class 2: Redacted findings

```text
“GitHub MCP lost 1 tool”
“Agent missed project owner”
“Latency 2.4x slower”
```

Allowed after scrubber pass.

### Class 3: External execution

```text
browser checks
workflow execution
external alerts
benchmark contribution
LLM-as-judge scoring
```

Explicit opt-in only.

## Scrubber requirements

Minimum scrubber:

```text
known secret regexes
high-entropy token detection
.env denylist
SSH/API key patterns
cloud credential patterns
email/phone/basic PII
absolute path redaction
username redaction
repo name redaction
branch name optional redaction
custom deny globs
```

Cloud upload must fail closed:

```text
redaction_status = unknown  -> block
secrets_found > 0           -> block unless redacted and class <= 2
raw_exported = true         -> block
```

## Benchmark contribution

Benchmark contribution should be opt-in and aggregate only.

Rules:

```text
no raw prompts
no raw responses
no custom prompt text
no file names
no repo names
no user names
no exact workspace labels
no single-user slices
k-anonymity threshold before display
```

Do not launch a public benchmark marketplace in v0. The OpenClaw ecosystem has real security sensitivity around skills/add-ons and broad local permissions, so “we safely scrub and aggregate” must be proven before benchmarks become a headline feature. ([The Verge][7])

---

# 6) Paid packaging

Do not charge for local history. That would damage the local-first promise.

Charge for:

```text
cloud retention
compare over longer windows
alerts
team/agency views
synced custom packs
workflow comparisons
benchmark access
browser checks
```

## Free

```text
local CLI/MCP
local SQLite history forever
14-day cloud history
dashboard
API token
Baseline Core
Baseline Configurable fact/repo/style packs
baseline_compare within local + 14-day cloud window
manual known-good anchors
```

## Pro

Target buyer: founder, solo CTO, heavy OpenClaw user.

```text
90-day or 180-day cloud retention
long-window baseline_compare
alerts
scheduled summaries
custom prompt packs synced across machines
more workspaces
API export
```

Paid MCP additions:

```text
baseline_compare --days 30/90/180
baseline_alerts
baseline_workflow_compare dry_run
```

## Team / Agency

Target buyer: agency owner, CTO managing agent workstations.

```text
team workspaces
client/project dashboards
alert routing
workspace-level known-good anchors
custom packs shared across team
longer retention
audit trail
benchmark access
```

Paid MCP additions:

```text
baseline_workflow_compare execute
baseline_benchmarks
team compare summaries
```

## Higher tier

```text
browser workflow probes
private benchmark pools
benchmark segmentation
hosted runners
long retention
compliance/export controls
```

Do not put browser checks in Pro by default. They are expensive, flaky, and risky.

## Packaging warning

“Public anonymized benchmarks” are appealing but dangerous. They should be a **higher-tier add-on**, not the free-tier hook. The free-tier hook is:

```text
14 days of history + dashboard + known-good drift compare
```

That is enough.

---

# 7) Validation plan and kill criteria

## Dogfood plan

Use it on your own Claws first.

Seven-day protocol:

```text
1. Run baseline check before each work session.
2. Run baseline check after each long agent session.
3. Mark one known-good manually.
4. Record whether every warning was useful, noisy, or wrong.
5. Track whether Baseline caught anything before you noticed it manually.
```

Success after 7 days:

```text
install < 5 minutes
fast check < 60 seconds
at least 5 useful findings
at least 1 finding you would have missed manually
false/noisy alert rate < 20%
you voluntarily run baseline report or compare without forcing yourself
```

Harder success criterion:

```text
You trust it enough to run before letting a Claw touch an important repo.
```

If that is not true, the product is not ready.

## External smoke test

Run a paid pilot before building the SaaS deeply.

Target:

```text
10 OpenClaw/coding-agent power users
5 installs
3 day-two returns
2 users willing to pay for alerts or retention
1 agency/CTO asks for team/workspace compare
```

Offer:

```text
$29–$99 for a 14-day agent health pilot
```

Deliverable:

```text
local CLI/MCP
14-day dashboard
weekly drift report
manual onboarding
```

Do not build team dashboards before someone pays or strongly commits.

## Kill criteria

Kill or radically narrow if:

```text
setup takes > 5 minutes for most users
fast check takes > 60 seconds
agent-behavior checks cannot run reliably through OpenClaw
findings are interesting but not actionable
users do not mark known-good manually
users do not come back on day two
alerts are noisy > 25%
users distrust cloud sync even after scrub preview
nobody asks for compare, retention, or alerts
workflow packs become the only thing users care about
```

The most important kill criterion:

> If users do not say, “I want to run this before my agent starts work,” cut scope until they do.

---

# 8) Technical risks and what to cut

## Biggest technical risks

### 1. MCP server cannot evaluate the host by itself

A local MCP server mostly waits for calls. It does not automatically get to interrogate the model. You need an OpenClaw runner, MCP sampling support, or another adapter path.

Mitigation:

```text
v0 = OpenClaw runner + local environment checks
generic MCP = local/tool drift only
```

Do not overpromise generic agent evals.

### 2. OpenClaw integration churn

OpenClaw’s MCP bridge and registry are the wedge, but they may move quickly. Keep the adapter small and tested.

Mitigation:

```text
adapter/openclaw only
contract tests against openclaw mcp serve/list/set
clear fallback to generic-local
```

### 3. Drift scoring can become nonsense

Behavioral drift is useful, but noisy.

Mitigation:

```text
compare concrete facts first
score only after diff
show evidence
avoid “personality” language
prefer “behavior changed”
```

### 4. Redaction false negatives

One leaked key kills trust.

Mitigation:

```text
cloud sync off by default
scrub preview
fail-closed upload
no raw cloud storage
deny env/secrets files
hash names/paths
```

### 5. Workflow execution can mutate repos

Running prose workflows as packages is powerful and dangerous.

Mitigation:

```text
v0 custom = prompt/contract comparison only
paid workflow = dry_run first
execute requires explicit policy + clean git check
capture repo diff hash
never auto-push
```

### 6. Browser checks will eat the product

Browser probes are valuable but slow, flaky, expensive, and permission-heavy.

Mitigation:

```text
higher tier only
not in v0
not in default check
```

### 7. Dashboard creep

The dashboard can turn into fake observability.

Mitigation:

```text
latest
timeline
compare
token/settings
```

Nothing else.

### 8. OTel exporter distraction

OTel shape is useful; exporter implementation is not v0. The GenAI semantic conventions are useful but still marked development. ([OpenTelemetry][6])

Mitigation:

```text
schema compatible now
exporter later
```

## Cut from v0

Cut aggressively:

```text
Rust implementation
multi-agent framework adapters
full custom eval builder
arbitrary workflow execution
browser checks
public benchmark marketplace
team RBAC
Slack/GitHub/email alert delivery
OTel exporter
raw transcript cloud storage
language-specific plugins
hosted eval runners
complex dashboard
auto-known-good
LLM-as-judge cloud scoring
```

## Keep

Keep only:

```text
one Go binary
one local SQLite DB
one OpenClaw adapter
one generic local adapter
one MCP server
one scrubber
one known-good diff
one hosted ingest API
one tiny dashboard
one 14-day free cloud retention policy
```

## Final product shape

The defensible v0 is:

> **Baseline Core: required local health and drift checks.**
> **Baseline Configurable: toggle packs for fact/style/repo/tool checks.**
> **Baseline Custom: local prompt packs first, workflow package comparison later.**
> **Cloud: 14-day redacted history, dashboard, API token.**
> **Paid: longer compare windows, alerts, team views, workflow comparison, benchmarks, browser probes.**

The uncomfortable cut: **do not build a general eval framework yet.**

The irresistible product is much smaller:

```text
baseline check
baseline compare --known-good
baseline serve mcp
```

That is the bead.

[1]: https://docs.openclaw.ai/ja-JP/cli/mcp "mcp - OpenClaw"
[2]: https://modelcontextprotocol.io/docs/sdk "SDKs - Model Context Protocol"
[3]: https://modelcontextprotocol.io/specification/2025-11-25/schema "Schema Reference - Model Context Protocol"
[4]: https://blog.modelcontextprotocol.io/posts/2026-03-16-tool-annotations/ "Tool Annotations as Risk Vocabulary: What Hints Can and Can't Do | Model Context Protocol Blog"
[5]: https://neon.com/pricing "Pricing — Neon"
[6]: https://opentelemetry.io/docs/specs/semconv/gen-ai/ "Semantic conventions for generative AI systems | OpenTelemetry"
[7]: https://www.theverge.com/news/874011/openclaw-ai-skill-clawhub-extensions-security-nightmare "OpenClaw’s AI ‘skill’ extensions are a security nightmare | The Verge"
