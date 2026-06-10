# Baseline.ai OpenClaw Plugin

This bundle contributes the Baseline MCP server configuration and a short
operator skill for local health checks.

## Install

```sh
openclaw plugins install ./openclaw-plugin
openclaw plugins inspect baseline-ai
```

Restart the OpenClaw Gateway after install so the embedded MCP settings reload.

If the `baseline` command is not on `PATH`, install the CLI first:

```sh
curl -fsSL https://trackbaseline.com/install.sh | sh
baseline --version
baseline doctor
```

`baseline --version` should print `baseline 0.1.0`. `baseline doctor` is a read-only preflight smoke and does not send OpenClaw probe messages. Source installs can still use `go install github.com/apollostreetcompany/baseline/cmd/baseline@latest`.

## Local Dogfood Path

Run exactly four commands for a local OpenClaw Good Baseline:

```sh
baseline --version
baseline doctor
baseline setup
baseline report
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
baseline run
baseline compare
```

Use `baseline rerun <FAILED_RUN_ID>` only after reviewing a failed lifecycle report and stdout/stderr paths.

Then run `baseline run` and `baseline compare` any time you want to check drift against the accepted Good Baselines. Use `baseline doctor` when you only want local preflight and do not want to send OpenClaw probe messages.

`baseline setup` and `baseline run` default to the 14-question Baseline Core pack and write `REPORT.md`, `RESPONSES.md`, `RECEIPT.md`, and `metrics.json` under `~/.baseline/reports/<RUN_ID>/`. For OpenClaw targets, `baseline setup` also snapshots `~/.openclaw/openclaw.json` and ensures Codex app-server request and turn-idle timeouts are at least 900 seconds. Run `baseline run --packs enabled` only after the operator approves the wider pack set.

Through MCP, `baseline_setup`, `baseline_run`, and `baseline_schedule` with `action:"run"` start the eval in the background and return `run_status.run_id`. Poll `baseline_report` with that run id until the report is complete. The advertised MCP surface stays at seven tools: `baseline_setup`, `baseline_run`, `baseline_doctor`, `baseline_report`, `baseline_accept`, `baseline_schedule`, and `baseline_scrub_preview`. To recover a failed lifecycle run after operator approval, call `baseline_run` with `rerun_id` or use CLI `baseline rerun <RUN_ID>`.

Manual MCP registry fallback:

```sh
openclaw mcp set baseline '{"command":"baseline","args":["serve","mcp"]}'
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | baseline serve mcp
```

If that smoke fails with a missing CLI error, install the CLI or adjust OpenClaw's PATH. Do not add a version/preflight MCP tool; use `baseline --version` and `baseline doctor` outside MCP.

## Gateway Session Metrics

OpenClaw behavior checks invoke `openclaw agent --json --session-id <baseline-session-id> --message <probe>`. Baseline captures `system_send_at` before sending and `baseline_received_at` when the response returns, then correlates `openclaw sessions --json` for model and token metadata. Do not infer token usage from transcript length. If usage metadata is missing, report OpenClaw metrics as unavailable.

The OpenClaw MCP bridge keeps live events only while connected; durable history must be read from the Gateway-backed transcript tools.

Timeout diagnosis rule: `idleMs=60007`, `timeoutMs=60000`, or `turn_completion_idle_timeout` points at OpenClaw's Codex app-server idle watchdog. Rerun `baseline setup` or `baseline install openclaw`, then start a fresh eval. `401 Unauthorized` with `__OPENCLAW_REDACTED__` in ACP child Codex streams or memory search is an auth/env failure, not a real timeout; report the exact child path to the operator and do not remove Google/Gemini search configuration.

Lifecycle rule: `baseline report RUN_ID --json` exits `0` when completed, `2` while still running, and `1` for failed lifecycle runs. Failed background runs include stdout/stderr paths and a rerun action; read those logs before retrying.

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
