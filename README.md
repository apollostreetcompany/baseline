# Baseline.ai v0.1

Baseline is a local-first Go/SQLite CLI and MCP server for coding-agent workstation health. It checks local runtime, repo, MCP, redaction, latency, and optional OpenClaw behavior, then compares new runs against accepted Good Baselines.

Live launch surface:

- Landing page: https://baseline-ai.ryan-borker.workers.dev
- Dashboard: https://baseline-ai.ryan-borker.workers.dev/dashboard
- Admin: https://baseline-ai.ryan-borker.workers.dev/admin
- MCP docs: https://baseline-ai.ryan-borker.workers.dev/docs/mcp
- Latest run API: https://baseline-ai.ryan-borker.workers.dev/api/runs/latest
- Timeline API: https://baseline-ai.ryan-borker.workers.dev/api/runs/timeline

## Install

```sh
go install github.com/apollostreetcompany/baseline/cmd/baseline@latest
baseline bootstrap --openclaw
```

Local source build:

```sh
go build -o bin/baseline ./cmd/baseline
```

## Local OpenClaw Dogfood

Run these commands for a local-only OpenClaw bootstrap, first Good Baseline, and drift comparison:

```sh
baseline bootstrap --openclaw
baseline bootstrap preview
baseline bootstrap run
baseline bootstrap accept --label clean-local
baseline compare
```

Use `baseline check --fast` for local runtime/repo/MCP checks when you do not want to send OpenClaw probe messages. Fast mode never runs the agent.
`baseline bootstrap run` defaults to the 14-question Baseline Core pack; use `--packs enabled` or `--packs all` after reviewing the preview.

## Bootstrap Lifecycle

`baseline bootstrap` is idempotent. It creates `~/.baseline/config.json`, `~/.baseline/baseline.db`, report/redaction directories, runs SQLite migrations, detects OpenClaw, registers the Baseline stdio MCP server when `--openclaw` is set, and prints the next command. It does not enable cloud sync or execute agent prompts unless the user supplies explicit sync or runner flags.

Optional sync setup stays explicit:

```sh
baseline bootstrap --openclaw --sync-url https://baseline-ai.ryan-borker.workers.dev --sync-token <token>
```

## Good Baselines

Good Baselines are manually accepted local anchors. v0.1 keeps up to three active Good Baselines per workspace so users can preserve a small set of trusted states without turning every passing run into truth.

```sh
baseline good accept [RUN_ID] --label clean-local
baseline good list
baseline good replace [RUN_ID] --slot 1 --label clean-local
baseline compare
```

`baseline compare` uses the accepted Good Baselines for the current workspace/config. Baseline refuses a fourth active Good Baseline until the user replaces slot 1, 2, or 3.

## Run

```sh
baseline check --fast
baseline check --full --run-agent
baseline latest --json
baseline report
baseline compare
baseline sync status
baseline sync push
baseline schedule install --at 09:00
baseline schedule status
baseline schedule run
```

Fast mode never runs the agent. Full mode sends real OpenClaw probe messages only when `--run-agent` is set. Bootstrap run always uses explicit probe execution because the user asked to establish the first baseline, and defaults to the 14-question Baseline Core pack.

## OpenClaw Timing and Tokens

For OpenClaw behavior checks, Baseline invokes the real handler path:

```sh
openclaw agent --json --session-id <baseline-session-id> --message <probe>
```

For every probe, Baseline records `system_send_at` immediately before sending the message and `baseline_received_at` immediately after receiving the completed response. It then correlates `openclaw sessions --json` by session ID/time window for model/provider and token metadata. If token metadata cannot be correlated, `token_status` is `unavailable`; if OpenClaw reports stale metadata, `token_status` is `stale`. Baseline never estimates tokens from text length.

OpenClaw's MCP bridge is Gateway-backed: live events only exist while the bridge session is connected, and older history should be read from transcript tools. Token and cost visibility follows OpenClaw usage configuration; OAuth-backed sessions may expose tokens without dollar cost.

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
baseline config set agent_command '<command using BASELINE_PROMPT>'
baseline config set monitor_packs.workflow_test.enabled true
baseline config set monitor_packs.self_log_execution.enabled false
baseline config validate --json
```

## MCP Tools

The MCP server exposes seven tools:

- `baseline_check`
- `baseline_bootstrap`
- `baseline_good`
- `baseline_report`
- `baseline_compare`
- `baseline_schedule`
- `baseline_scrub_preview`

Manual MCP smoke test:

```sh
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./bin/baseline serve mcp
printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"baseline_schedule","arguments":{"action":"run"}}}' | ./bin/baseline serve mcp
```

## Daily Schedule

On macOS, Baseline installs a user LaunchAgent at `~/Library/LaunchAgents/ai.baseline.daily.plist`. The scheduled job runs `baseline schedule run`, which performs a fast local check and syncs queued redacted payloads when cloud sync is enabled.

```sh
baseline schedule install --at 09:00
baseline schedule status
baseline schedule run
```

## Safety

Baseline defaults to local SQLite. Cloud sync sends a small redacted payload: run ID, timing, score, mode, agent kind, check metadata, metric numbers, and workspace hash. Raw prompts, raw responses, local paths, and API keys are not exported by the v0 sync path.

Cloud sync is staged through a local SQLite outbox. Failed uploads remain retryable and visible through `baseline sync status`; `baseline sync push` stages unsynced local runs and retries queued uploads.

The deployed ingest API fails closed unless `BASELINE_API_TOKEN` matches. Stripe checkout is implemented but not live until Stripe credentials or payment links are set as Worker secrets.

## Deployed Infrastructure

- Cloudflare Worker: `baseline-ai`
- Worker URL: https://baseline-ai.ryan-borker.workers.dev
- Neon project: `baseline-v0` (`summer-cake-63602849`)
- Neon tables: `baseline_runs`, `baseline_events`, `canonical_question_sets`, `llm_evaluations`

## Admin and Evaluator

The admin page versions canonical question sets in Neon and can evaluate the latest run. Set `BASELINE_ADMIN_TOKEN` as a Worker secret to enable mutations. Set `OPENAI_API_KEY` and optionally `OPENAI_EVALUATOR_MODEL` to use the OpenAI Responses API with structured JSON output; without those secrets the evaluator uses a local heuristic over redacted check metadata.

## Distribution

- Go: `go install github.com/apollostreetcompany/baseline/cmd/baseline@latest`
- npm/pnpm wrapper: `package/` publishes `@baseline-ai/cli`
- OpenClaw bundle: `openclaw-plugin/` installs with `openclaw plugins install ./openclaw-plugin`

See `docs/PUBLISHING.md` for release and verification steps.

## Current Blocker

Payment buttons route to `/api/checkout`, but checkout returns `503` because no Stripe secret, price IDs, or payment links were available in the environment.
