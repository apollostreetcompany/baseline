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
baseline bootstrap --openclaw
baseline bootstrap preview
baseline bootstrap run
baseline bootstrap accept --label clean-local
```

Then run `baseline compare` any time you want to check drift against the accepted Good Baselines. Use `baseline check --fast` when you only want local checks and do not want to send OpenClaw probe messages.
`baseline bootstrap run` requires a recent preview receipt, defaults to the 14-question Baseline Core pack, and accepts `--preview-id <id>` from the preview output when you want an exact receipt match. Run `baseline bootstrap run --packs enabled` only after reviewing the wider pack preview.

Manual MCP registry fallback:

```sh
openclaw mcp set baseline '{"command":"baseline","args":["serve","mcp"]}'
```

## Gateway Session Metrics

OpenClaw behavior checks invoke `openclaw agent --json --session-id <baseline-session-id> --message <probe>`. Baseline captures `system_send_at` before sending and `baseline_received_at` when the response returns, then correlates `openclaw sessions --json` for model and token metadata. Do not infer token usage from transcript length. If usage metadata is missing, report OpenClaw metrics as unavailable.

The OpenClaw MCP bridge keeps live events only while connected; durable history must be read from the Gateway-backed transcript tools.

## Daily Self-Check

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
