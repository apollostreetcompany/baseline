# Baseline.ai OpenClaw Plugin

This bundle contributes the Baseline MCP server configuration and a short
operator skill for local health checks.

## Install

```sh
openclaw plugins install ./openclaw-plugin
openclaw plugins inspect baseline-ai
```

Restart the OpenClaw gateway after install so the embedded MCP settings reload.

If the `baseline` command is not on `PATH`, install the Go binary first:

```sh
go install github.com/apollostreetcompany/baseline/cmd/baseline@latest
```

Manual MCP registry fallback:

```sh
openclaw mcp set baseline '{"command":"baseline","args":["serve","mcp"]}'
```

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
