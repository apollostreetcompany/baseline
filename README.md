# Baseline.ai v0

Baseline is a local-first health check and drift monitor for coding-agent workstations. It stores runs in SQLite, exposes a small MCP server, compares local state against a known-good run, and can sync redacted run summaries to a Cloudflare Worker backed by Neon.

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
baseline init
baseline install openclaw
openclaw mcp list
```

Local source build:

```sh
go build -o bin/baseline ./cmd/baseline
```

This machine is already configured with:

- Baseline MCP registered in OpenClaw as `baseline`
- Cloud sync enabled against the deployed Worker
- API token stored locally in `~/.baseline/config.json`
- A post-MCP known-good run marked as `post-mcp-clean`

## Run

```sh
./bin/baseline check --fast
./bin/baseline check --full
./bin/baseline check --full --run-agent
./bin/baseline latest --json
./bin/baseline report
./bin/baseline compare
./bin/baseline sync status
./bin/baseline sync push
./bin/baseline schedule install --at 09:00
./bin/baseline schedule status
./bin/baseline schedule run
```

Fast mode never runs the agent. Full mode includes the 12-question baseline pack but skips execution until `--run-agent` or `BASELINE_RUN_AGENT=1` is set.

## MCP Tools

The MCP server exposes seven tools:

- `baseline_check`
- `baseline_latest`
- `baseline_report`
- `baseline_compare`
- `baseline_mark_known_good`
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
