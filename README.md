# Baseline.ai v0.1

Baseline is a local-first Go/SQLite CLI and MCP server for coding-agent workstation health. It checks local runtime, repo, MCP, redaction, latency, and optional OpenClaw behavior, then compares new runs against accepted Good Baselines.

Live launch surface:

- Landing page: https://trackbaseline.com
- Dashboard: https://trackbaseline.com/dashboard
- Admin: https://trackbaseline.com/admin
- MCP docs: https://trackbaseline.com/docs/mcp
- Blog stub: https://trackbaseline.com/blog
- Remote MCP: https://trackbaseline.com/mcp
- Latest run API: https://trackbaseline.com/api/runs/latest
- Timeline API: https://trackbaseline.com/api/runs/timeline

## Install

```sh
curl -fsSL https://trackbaseline.com/install.sh | sh
baseline setup
```

Local source build:

```sh
go build -o bin/baseline ./cmd/baseline
```

## Local OpenClaw Dogfood

Run these commands for a local-only OpenClaw setup, first Good Baseline, and drift comparison:

```sh
baseline setup
baseline report
baseline rerun <FAILED_RUN_ID>
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
baseline compare
```

`baseline setup` writes `~/.baseline/config.json`, `~/.baseline/BOOTSTRAP.md`, the local database/redaction files, ensures OpenClaw Codex app-server request/turn-idle timeouts are at least 900 seconds, then runs the real default target eval. It prints the report and response artifact paths so the operator can review before accepting. Use `baseline doctor` for read-only preflight when you do not want to send agent probe messages.

The default target is OpenClaw `agent:main` with `model_policy: follow_current`, which means Baseline evaluates the agent as the operator currently configured it. Pinning a different model is an advanced config choice. The default run uses the 14-question Baseline Core pack; use `--packs enabled` or `--packs all` only when the operator wants a wider eval. The full v0.1 pack list is in [docs/QUESTION_SET.md](docs/QUESTION_SET.md).

## Setup Lifecycle

`baseline setup` is idempotent for Baseline-owned files. It creates `~/.baseline/config.json`, `~/.baseline/BOOTSTRAP.md`, `~/.baseline/baseline.db`, report/redaction directories, runs SQLite migrations, detects OpenClaw, runs the default eval, and prints the next command. For OpenClaw targets, setup also snapshots `~/.openclaw/openclaw.json` and patches `plugins.entries.codex.config.appServer.requestTimeoutMs` plus `turnCompletionIdleTimeoutMs` to at least `900000` ms so long Baseline runs do not inherit OpenClaw's 60s Codex app-server idle watchdog. It does not enable cloud sync. OpenClaw MCP registration stays explicit through `baseline install openclaw` or `baseline setup --register-openclaw`.

Optional sync setup stays explicit:

```sh
baseline sync on --url https://trackbaseline.com --token <token>
```

For Pro, that token should now be a workspace token created from an invited account session. The old global dogfood token still works as a temporary fallback, but it is not the customer path.

## Pro Cloud Account Path

Baseline Pro is Cloudflare Worker + Neon first. The Worker stores users, accounts, sessions, workspaces, HMAC-hashed workspace tokens, Stripe subscriptions, entitlements, audit events, lifecycle outbox rows, self-history runs, and aggregate-safe comparison fields.

Primary account routes:

- `POST /api/admin/invites`: admin-only invite and optional pilot grant.
- `POST /api/auth/magic-link`: request a magic link for an invited email.
- `POST /api/auth/consume`: exchange a magic-link token for an account session.
- `GET /api/account/status`: account and entitlement status.
- `GET|POST /api/workspaces`: list or create account workspaces.
- `POST /api/tokens`: create a one-time visible workspace ingest token.
- `POST /api/tokens/revoke`: revoke with `confirm: "revoke <token_id>"`.
- `GET /api/history`, `/api/hotspots`, `/api/compare`: self-history and hotspot APIs.
- `POST /api/stripe/webhook`: raw Stripe signature verification and idempotent entitlement updates.
- `POST /mcp`: Streamable-HTTP-style JSON-RPC remote MCP adapter over account, workspace, history, hotspot, compare, subscription, and owner support tools.

Billing management uses Stripe Checkout and the Stripe Billing Portal. Baseline does not cancel subscriptions directly through MCP.

## macOS Hotspot App

A SwiftUI macOS client lives in `macos/BaselineHotspots`. It signs in with a magic link, connects to the remote MCP, shows the run timeline and hotspots, and generates an operator insight from cloud redacted summaries. It tries local agent provider bridges first (`hermes`/`openclaw` where available), then falls back to the user's OpenRouter API key and model stored in macOS Keychain.

Build:

```sh
make mac-build
```

## Good Baselines

Good Baselines are manually accepted local anchors. v0.1 keeps up to three active Good Baselines per workspace so users can preserve a small set of trusted states without turning every passing run into truth.

```sh
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
baseline good list
baseline good replace <RUN_ID> --slot 1 --confirm "replace <RUN_ID> slot 1" --label clean-local
baseline compare
```

`baseline compare` uses the accepted Good Baselines for the current workspace/config. Baseline refuses a fourth active Good Baseline until the user replaces slot 1, 2, or 3.

## Run

```sh
baseline setup
baseline run
baseline report
baseline repair openclaw
baseline rerun <FAILED_RUN_ID>
baseline accept <RUN_ID> --confirm "accept <RUN_ID>"
baseline doctor
baseline latest --json
baseline compare
baseline sync status
baseline sync push
baseline schedule install --at 09:00
baseline schedule status
baseline schedule run
```

`baseline run` sends real probe messages to the configured target, records latency/quality, and writes `REPORT.md`, `RESPONSES.md`, `RECEIPT.md`, and `metrics.json` under `~/.baseline/reports/<RUN_ID>/`. Long non-interactive runs are detached into their own process session and write per-question progress into `~/.baseline/runs/<RUN_ID>.stdout.log`; `baseline report RUN_ID --json` exits `0` for completed, `2` for still running, and `1` for failed lifecycle runs. `baseline rerun RUN_ID` applies known guardrails and starts a new run with the same pack selection. `baseline doctor` is read-only preflight and never becomes a Good Baseline candidate. Legacy `baseline check --fast|--full` remains available for scripted compatibility.

## OpenClaw Timing and Tokens

For OpenClaw behavior checks, Baseline invokes the real handler path:

```sh
openclaw agent --json --session-id <baseline-session-id> --message <probe>
```

For every probe, Baseline records `system_send_at` immediately before sending the message and `baseline_received_at` immediately after receiving the completed response. It then correlates `openclaw sessions --json` by session ID/time window for model/provider and token metadata. If token metadata cannot be correlated, `token_status` is `unavailable`; if OpenClaw reports stale metadata, `token_status` is `stale`. Baseline never estimates tokens from text length.

OpenClaw's MCP bridge is Gateway-backed: live events only exist while the bridge session is connected, and older history should be read from transcript tools. Token and cost visibility follows OpenClaw usage configuration; OAuth-backed sessions may expose tokens without dollar cost.

If logs show `idleMs=60007`, `timeoutMs=60000`, `lastActivityReason=notification:item/completed`, or `turn_completion_idle_timeout`, the failing layer is OpenClaw's Codex app-server idle watchdog. Run `baseline setup` or `baseline install openclaw`, then start a fresh eval. If ACP child Codex runs or memory search show `401 Unauthorized` with `__OPENCLAW_REDACTED__`, treat that as a child environment/auth configuration failure, not a model timeout. Do not remove Google/Gemini search or background API configuration to fix that class of failure.

References: [OpenClaw MCP bridge](https://docs.openclaw.ai/cli/mcp), [OpenClaw token usage](https://openclawlab.com/en/docs/reference/token-use/).

## Config CLI Shape

Config is local JSON under `~/.baseline/config.json` with file mode `0600`. Token reads should display only `token_set:true`.

```sh
baseline config show
baseline config file
baseline config get cloud_sync --json
baseline config set cloud_sync true
baseline config set api_base_url <url>
baseline config set api_token <token>
baseline config set target.model_policy pinned
baseline config set target.pinned_model openai/gpt-5.5
baseline config set target.runtime hermes
baseline config set target.runtime custom
baseline config set agent_command '<advanced command using BASELINE_PROMPT>'
baseline config set monitor_packs.workflow_test.enabled true
baseline config set monitor_packs.self_log_execution.enabled false
baseline config validate --json
```

Hermes native mode (`target.runtime hermes`) shells directly to `hermes chat -Q -q <prompt>` with the current Hermes model. Use `target.model_policy pinned` plus `target.pinned_model` when a run should pass `-m` to Hermes. Custom mode remains available for arbitrary commands; Baseline sets `BASELINE_PROMPT` before invoking `agent_command`.

## MCP Tools

The MCP server exposes seven tools:

- `baseline_setup`
- `baseline_run`
- `baseline_doctor`
- `baseline_report`
- `baseline_accept`
- `baseline_schedule`
- `baseline_scrub_preview`

Manual MCP smoke test:

```sh
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./bin/baseline serve mcp
printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"baseline_schedule","arguments":{"action":"run"}}}' | ./bin/baseline serve mcp
```

MCP errors return structured recovery hints and `next_actions`. `baseline_setup`, `baseline_run`, and `baseline_schedule` with `action:"run"` start the eval in the background and return a `run_status` with a `run_id`; agents should poll `baseline_report` with that run id until it returns the completed report/responses. Only call `baseline_accept` after showing the operator the markdown report plus local responses and receiving explicit confirmation.

## Daily Schedule

On macOS, Baseline installs a user LaunchAgent at `~/Library/LaunchAgents/ai.baseline.daily.plist`. The scheduled job runs `baseline schedule run`, which evaluates the configured default target and syncs queued redacted payloads when cloud sync is enabled.

```sh
baseline schedule install --at 09:00
baseline schedule status
baseline schedule run
```

## Safety

Baseline defaults to local SQLite. Cloud sync sends a small redacted payload: run ID, timing, score, mode, agent kind, check metadata, metric numbers, and workspace hash. Raw prompts, raw responses, local paths, and API keys are not exported by the v0 sync path.

Cloud sync is staged through a local SQLite outbox. Failed uploads remain retryable and visible through `baseline sync status`; `baseline sync push` stages unsynced local runs and retries queued uploads.

The deployed ingest API accepts either the temporary global dogfood token or a Pro workspace token. Pro tokens are stored in Neon as prefix plus HMAC hash only. Stripe checkout and webhooks are implemented but not live until Stripe credentials, price IDs, webhook secret, magic-link secret, token HMAC secret, and Klaviyo credentials are set as Worker secrets.

## Deployed Infrastructure

- Cloudflare Worker: `baseline-ai`
- Worker URL: https://trackbaseline.com
- Neon project: `baseline-v0` (`summer-cake-63602849`)
- Neon tables: `baseline_runs`, `baseline_events`, `canonical_question_sets`, `llm_evaluations`, plus Pro account/billing tables in `web/schema.sql`.

## Admin and Evaluator

The admin page versions canonical question sets in Neon and can evaluate the latest run. Set `BASELINE_ADMIN_TOKEN` as a Worker secret to enable mutations. Set `OPENAI_API_KEY` and optionally `OPENAI_EVALUATOR_MODEL` to use the OpenAI Responses API with structured JSON output; without those secrets the evaluator uses a local heuristic over redacted check metadata.

## Distribution

- Install script: `https://trackbaseline.com/install.sh` downloads release binaries from GitHub Releases, verifies `checksums.txt`, and installs to `~/.local/bin`.
- GitHub Release assets: macOS arm64/x86_64 and Linux arm64/x86_64 tarballs built by `.github/workflows/release.yml`.
- npm/pnpm wrapper: `package/` publishes `@baseline-ai/cli`, which auto-downloads the matching release binary when no local `baseline` is present.
- OpenClaw bundle: release artifact `baseline-openclaw-plugin.tgz`, also installable from `openclaw-plugin/` during development.

See `docs/PUBLISHING.md` for release and verification steps.

## Production Pro Status

Production Worker secrets are configured for Stripe Checkout, Stripe webhooks, Klaviyo lifecycle email, magic-link auth, and HMAC workspace tokens. The remaining paid-pilot check is an end-to-end live checkout/magic-link/workspace-token/sync smoke with a real invited account.
