# Baseline.ai v0

Baseline is a local-first health check and drift monitor for coding-agent workstations. It stores runs in SQLite, exposes a small MCP server, compares local state against a known-good run, and can sync redacted run summaries to a Cloudflare Worker backed by Neon.

Live launch surface:

- Landing page: https://baseline-ai.ryan-borker.workers.dev
- Dashboard: https://baseline-ai.ryan-borker.workers.dev/dashboard
- MCP docs: https://baseline-ai.ryan-borker.workers.dev/docs/mcp

## Install

```sh
go build -o bin/baseline ./cmd/baseline
./bin/baseline init
./bin/baseline install openclaw
openclaw mcp list
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
```

Fast mode never runs the agent. Full mode includes the 12-question baseline pack but skips execution until `--run-agent` or `BASELINE_RUN_AGENT=1` is set.

## MCP Tools

The MCP server exposes seven tools:

- `baseline_check`
- `baseline_latest`
- `baseline_report`
- `baseline_compare`
- `baseline_mark_known_good`
- `baseline_config`
- `baseline_scrub_preview`

Manual MCP smoke test:

```sh
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./bin/baseline serve mcp
```

## Safety

Baseline defaults to local SQLite. Cloud sync sends a small redacted payload: run ID, timing, score, mode, agent kind, check metadata, metric numbers, and workspace hash. Raw prompts, raw responses, local paths, and API keys are not exported by the v0 sync path.

The deployed ingest API fails closed unless `BASELINE_API_TOKEN` matches. Stripe checkout is implemented but not live until Stripe credentials or payment links are set as Worker secrets.

## Deployed Infrastructure

- Cloudflare Worker: `baseline-ai`
- Worker URL: https://baseline-ai.ryan-borker.workers.dev
- Neon project: `baseline-v0` (`summer-cake-63602849`)
- Neon tables: `baseline_runs`, `baseline_events`

## Current Blocker

Payment buttons route to `/api/checkout`, but checkout returns `503` because no Stripe secret, price IDs, or payment links were available in the environment.
