# Baseline.ai V0 Product Shape

## Blunt Position

Baseline should start as a high-trust local MCP/CLI that answers:

> Is my Claw or coding agent behaving normally today?

Do not sell "LLM observability." Do not start with dashboards. Do not start with custom eval frameworks. Start with a tiny, legible MCP that dogfoods on real OpenClaw work and produces useful drift reports fast.

The sharper wedge is:

> Baseline is `git status` plus smoke test plus known-good diff for OpenClaw/coding-agent setups.

The user should want to run Baseline before letting a Claw touch an important repo.

Hard constraint: a passive MCP server cannot evaluate a host model by itself. Agent-behavior checks require an OpenClaw runner path, MCP sampling support, or another host adapter. V0 is OpenClaw-runner-first. Generic MCP clients get local environment, repo, tool, and MCP drift checks only.

## Smallest Irresistible V0

Ship one Go binary:

```bash
baseline init
baseline check
baseline report
baseline latest
baseline compare
baseline known-good mark
baseline serve mcp
baseline sync on
baseline scrub preview
```

What the user gets in the first 5 minutes:
- A local MCP server registered with OpenClaw.
- A fast baseline check over the current Claw/workspace.
- A redacted local report explaining whether anything is off.
- A free cloud token for 14 days of redacted run history.
- A dashboard URL showing recent drift, speed, failures, and comparison to known-good.

The magic moment:

```text
Baseline found 3 changes since your known-good Claw:
1. Tool latency is 2.3x slower than your 7-day median.
2. Your agent no longer remembers the project owner.
3. GitHub MCP schema changed: 1 tool disappeared.
```

Known-good diff is the killer feature. A raw score is weak; a concrete "what changed since Monday's known-good run" report is useful.

## Why Go First

Use Go for v0:
- Official MCP Go SDK is Tier 1.
- Single static-ish binary is easier to trust and install than a Node/Python dependency tree.
- Good fit for CLI, SQLite, HTTP API client, concurrency, and cross-platform distribution.
- Supports many languages by inspecting repos and running configured commands instead of embedding language-specific runtimes.

Do not use Rust first unless the first wedge becomes local sandboxing or tamper-resistant agents. Rust is credible, but Go is faster to ship and easier for MCP/server/CLI ergonomics right now.

## Product Lanes

Configuration must be toggles only.

### 1. Baseline Core

Required. Cannot be disabled.

Checks:
- runtime identity: agent, model, provider, versions
- e2e speed per query
- MCP/tool availability and schema hash, including paginated `tools/list` responses
- repo state: branch, dirty files, latest commit, test command detection
- core memory: user, project, active task, current constraints
- safety scrubber status
- local store health

### 2. Baseline Configurable

Toggle packs. No prompt editing required.

Examples:
- fact checks
- style checks
- agency handoff checks
- CTO risk checks
- repo-awareness checks
- MCP reliability checks
- latency variance checks
- browser/tool-use checks

Each pack has:
- `enabled: true|false`
- `frequency: every_run|daily|manual`
- `sensitivity: low|normal|high`

### 3. Baseline Custom

Custom prompts and workflow packs.

Examples:
- "Ask my Claw who I am and what project this is."
- "Run this prose workflow package and compare expected actions to the agent's response."
- "Check whether the agent still follows my review style."

V0 should support custom prompt packs before arbitrary workflow execution. Workflow execution can come next, after the scoring and safety model are proven.

## MCP Surface

Keep the MCP small and read-heavy.

Free local tools:
- `baseline_check`
- `baseline_report`
- `baseline_latest`
- `baseline_mark_known_good`
- `baseline_config`
- `baseline_sync_status`
- `baseline_scrub_preview`

Paid/cloud-backed tools:
- `baseline_compare`
- `baseline_alerts`
- `baseline_benchmarks`
- `baseline_browser_check`

Tool behavior:
- `baseline_check`: runs local checks and returns a redacted summary.
- `baseline_report`: returns latest local report.
- `baseline_latest`: returns current health score and top drift findings.
- `baseline_mark_known_good`: stores current run as local comparison anchor.
- `baseline_scrub_preview`: shows what would leave the machine before sync/export.
- `baseline_compare`: summarizes changes over time; free limited to local/14-day cloud window.
- `baseline_alerts`: configures and previews alerts; delivery requires paid cloud.
- `baseline_benchmarks`: compares anonymized metrics against public aggregates.
- `baseline_browser_check`: runs browser/tool workflow probes; higher tier.

Use MCP annotations:
- read-only for report/latest/compare/benchmarks
- non-destructive for check
- open-world true only for cloud/browser/export tools

Do not rely on annotations for safety. Enforce policy in code.

## Minimum Defensible Infrastructure

Local:
- Go binary
- SQLite database under `~/.baseline/baseline.db`
- config under `~/.baseline/config.toml`
- redaction rules under `~/.baseline/redaction.toml`
- MCP stdio server in the same binary

Cloud:
- Go API service
- Neon Postgres
- static/server-rendered dashboard
- API token auth
- 14-day retention for free workspaces
- paid retention beyond 14 days

No queue needed for v0. The CLI can batch-upload redacted run summaries opportunistically. Add a queue only after sync failures matter.

Cloud sync should not be enabled during `baseline init`. Use an explicit command:

```bash
baseline sync on --token bl_live_...
```

Before enabling sync, print an egress preview that separates uploaded metrics from local-only raw data.

Tables:
- `accounts`
- `workspaces`
- `api_tokens`
- `runs`
- `check_results`
- `alerts`
- `benchmark_rollups`

Store raw prompts and raw responses locally only by default. Cloud gets redacted summaries, scores, hashes, timings, runtime metadata, and sanitized finding text.

## Data Model

Core objects:

```json
{
  "run_id": "run_01",
  "workspace_id": "ws_01",
  "profile": "openclaw-default",
  "runtime": {
    "agent": "openclaw",
    "model": "unknown-or-detected",
    "version": "detected"
  },
  "started_at": "2026-05-13T12:00:00Z",
  "duration_ms": 18342,
  "health_score": 82,
  "egress": {
    "cloud_synced": true,
    "raw_exported": false,
    "scrubbed": true
  },
  "checks": []
}
```

```json
{
  "check_id": "core.memory.user",
  "lane": "core",
  "kind": "memory",
  "status": "warning",
  "score": 0.72,
  "duration_ms": 1420,
  "input_hash": "sha256:...",
  "output_hash": "sha256:...",
  "metrics": {
    "fact_recall": 0.8,
    "invented_fact_penalty": 0.1,
    "latency_ratio": 1.4
  },
  "finding": "Agent identified the project but missed the user preference.",
  "redaction": {
    "secrets_found": 0,
    "pii_found": 1,
    "cloud_safe": true
  }
}
```

OpenTelemetry mapping:
- run = trace
- check = span
- check result = span status + attributes
- latency/score = metrics
- drift finding = event

Export OTel later. Shape the schema now.

## Eval Shape

Every eval follows the same lane model:

```yaml
id: openclaw-default
lanes:
  core:
    required: true
    checks:
      - core.runtime.identity
      - core.mcp.schema
      - core.repo.state
      - core.speed.simple
      - core.memory.user
      - core.memory.project

  configurable:
    packs:
      fact_checks: true
      style_checks: true
      agency_handoff: false
      browser_checks: false

  custom:
    packs:
      - ./baseline/custom/future-style.yaml
      - ./baseline/custom/prose-workflow-smoke.yaml
```

Check shape:

```yaml
id: core.memory.user
prompt: "Who is your user?"
expected_behavior: "Answer from known context; say unknown if not known."
scorers:
  - fact_recall
  - hallucination_penalty
  - honesty_when_unknown
thresholds:
  warning: 0.75
  critical: 0.50
egress:
  cloud: summary_only
```

Custom workflow shape:

```yaml
id: custom.prose.run.compare
variables:
  workflow: "./recipes/03-offer-smoke-test.prose.md"
  task: "Summarize expected actions and compare to agent's actual response."
compare:
  against:
    - expected_steps
    - missing_actions
    - invented_actions
    - style_match
```

For v0, compare the agent response to the workflow contract. Do not execute arbitrary prose workflows automatically until safety and repeatability are proven.

Default check modes:

```bash
baseline check --fast  # runtime + MCP + repo + scrubber + latest known-good diff
baseline check --full  # fast + memory/context + enabled configurable packs
baseline check --pack fact_checks
```

Default is `--fast`.

## Safety And Trust

Trust comes from legibility.

Rules:
- Raw prompts and responses stay local by default.
- Cloud sync uploads summaries, scores, hashes, timings, runtime metadata, and scrubbed findings only.
- User must opt in to benchmark contribution.
- Benchmarks never include raw prompts, raw code, file names, repo names, user names, secrets, or exact custom prompt text.
- API tokens are scoped: `ingest:write`, `reports:read`, `alerts:write`.
- MCP export/browser/cloud tools require explicit enabled policy.

Scrubbing:
- known key patterns
- high-entropy token detection
- `.env` and secret file denylist
- email/phone/basic PII redaction
- repo path and username redaction
- custom deny globs

Policy modes:

```toml
mode = "local"
cloud_sync = false
raw_outputs = "local_only"
benchmark_contribution = false

[allow]
mcp_check = true
mcp_report = true
mcp_compare = true
external_alerts = false
browser = false
```

Egress classes:
- Class 0: local raw data, never cloud by default.
- Class 1: cloud-safe metrics such as duration, score, check ID, status, counts, and hashes.
- Class 2: redacted findings such as "GitHub MCP lost 1 tool" or "Latency 2.4x slower."
- Class 3: external execution such as browser checks, workflow execution, external alerts, and benchmark contribution; explicit opt-in only.

Cloud upload must fail closed if redaction status is unknown, secrets remain, or raw export is requested outside local-only policy.

## Paid Packaging

Free:
- local CLI/MCP
- local SQLite history
- 14 days cloud history
- dashboard
- API token
- core checks
- configurable fact/style packs

Pro:
- cloud retention beyond 14 days
- `baseline_compare` over long windows
- alerts
- scheduled cloud summaries
- team workspaces
- custom packs synced across machines

Team/Agency:
- multiple workspaces/projects
- client/project dashboards
- alert routing
- public anonymized benchmark access
- browser checks
- export API

Higher tier:
- browser workflow probes
- benchmark segmentation
- private benchmark pools
- longer retention

Do not charge for local history. Charge for cloud retention, alerts, team visibility, benchmarks, and expensive browser checks.

## Validation Plan

Dogfood first:
- Install in the user's own OpenClaw setup.
- Run before every work session and after every long agent session.
- Mark known-good runs manually.
- Track whether alerts match felt agent degradation.

7-day success criteria:
- Setup takes under 5 minutes.
- Daily fast check finishes under 60 seconds.
- At least 5 real drift findings are useful.
- Fewer than 20% of alerts are judged noisy.
- User voluntarily runs `baseline report` without prompting.

Harder dogfood criterion:

> You trust Baseline enough to run it before letting a Claw touch an important repo.

Agency/CTO smoke:
- Ask 10 agency owners or CTOs to install on one active project.
- Success requires 5 installs, 3 day-two returns, and 2 requests for alerts or longer history.
- Payment signal: at least 2 users agree to pay for retention/alerts after the 14-day window.

Kill criteria:
- Install takes more than 5 minutes for most users.
- Findings are interesting but not actionable.
- Users do not return after day one.
- Benchmarks require too much trust too early.
- Browser checks become the product before core checks are loved.

## What To Cut

Cut from v0:
- full custom eval builder UI
- arbitrary workflow execution
- browser checks by default
- team permissions
- Slack/GitHub alert delivery before local alerts work
- raw cloud transcript storage
- public benchmark marketplace
- language-specific plugins
- OTel exporter implementation
- auto-known-good marking
- cloud LLM-as-judge scoring
- generic deep adapters for every coding agent

Keep:
- one binary
- one MCP
- one local DB
- one hosted ingest API
- one tiny dashboard
- one comparison tool
- one redaction gate
- one OpenClaw runner adapter
- one generic local fallback adapter

## Proconsult Corrections Incorporated

The successful Proconsult run added these hard corrections:
- Do not describe V0 as generic MCP agent evaluation. Generic MCP is insufficient for model-behavior checks without a runner/sampling path.
- Make OpenClaw the first real behavior runner; generic MCP/local mode should still be useful for environment, repo, and tool drift.
- Treat known-good diff as the product center, not health score.
- Do not auto-enable cloud sync in `init`; use explicit `baseline sync on`.
- Add `baseline scrub preview` as a first-class trust command.
- Hash MCP tool/schema state only after following paginated `tools/list` responses.
- Cut OTel exporter, browser checks, workflow execution, full eval builder, and multi-agent adapters from V0.

## Sources To Recheck During Build

- MCP SDK tiers and official Go SDK: https://modelcontextprotocol.io/docs/sdk
- Go MCP SDK: https://github.com/modelcontextprotocol/go-sdk
- MCP tool annotation guidance: https://blog.modelcontextprotocol.io/posts/2026-03-16-tool-annotations/
- Neon pricing/free plan: https://neon.com/pricing
- OpenTelemetry GenAI conventions: https://opentelemetry.io/docs/specs/semconv/gen-ai/
