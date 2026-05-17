# Baseline.ai OpenClaw Plugin

This bundle contributes the Baseline MCP server configuration and a short
operator skill for local health checks.

## Install

```sh
openclaw plugins install ./openclaw-plugin
openclaw plugins inspect baseline-ai
```

Restart the OpenClaw Gateway after install so the embedded MCP settings reload.

If the `baseline` command is not on `PATH`, install the Go binary first:

```sh
go install github.com/apollostreetcompany/baseline/cmd/baseline@latest
```

## Local Dogfood Path

Run exactly four commands for a local OpenClaw Good Baseline:

```sh
baseline setup
baseline report
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
baseline compare
```

Then run `baseline run` and `baseline compare` any time you want to check drift against the accepted Good Baselines. Use `baseline doctor` when you only want local preflight and do not want to send OpenClaw probe messages.

`baseline setup` and `baseline run` default to the 14-question Baseline Core pack and write `REPORT.md`, `RESPONSES.md`, `RECEIPT.md`, and `metrics.json` under `~/.baseline/reports/<RUN_ID>/`. Run `baseline run --packs enabled` only after the operator approves the wider pack set.

Through MCP, `baseline_setup`, `baseline_run`, and `baseline_schedule` with `action:"run"` start the eval in the background and return `run_status.run_id`. Poll `baseline_report` with that run id until the report is complete.

Manual MCP registry fallback:

```sh
openclaw mcp set baseline '{"command":"baseline","args":["serve","mcp"]}'
```

## Gateway Session Metrics

OpenClaw behavior checks invoke `openclaw agent --json --session-id <baseline-session-id> --message <probe>`. Baseline captures `system_send_at` before sending and `baseline_received_at` when the response returns, then correlates `openclaw sessions --json` for model and token metadata. Do not infer token usage from transcript length. If usage metadata is missing, report OpenClaw metrics as unavailable.

The OpenClaw MCP bridge keeps live events only while connected; durable history must be read from the Gateway-backed transcript tools.

## Daily Self-Check

Scheduled checks run the configured default eval target and write report artifacts. They are not local-only plumbing checks.

From OpenClaw, ask the Baseline plugin to call:

```json
{"name":"baseline_schedule","arguments":{"action":"install","at":"09:00"}}
```

Then verify:

```json
{"name":"baseline_schedule","arguments":{"action":"status"}}
```

Manual CLI equivalent:

```sh
baseline schedule install --at 09:00
baseline schedule status
baseline schedule run
```
