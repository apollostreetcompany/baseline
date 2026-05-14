# @baseline-ai/cli

Small npm/pnpm wrapper for the Baseline Go CLI and MCP server.

This package does not vendor the Go application yet. It finds an installed `baseline`
binary and forwards all arguments to it. If no binary is found, it exits with the
supported install commands.

## Install

```sh
pnpm add -g @baseline-ai/cli
go install github.com/apollostreetcompany/baseline/cmd/baseline@latest
```

For local development from this repository:

```sh
go build -o bin/baseline ./cmd/baseline
pnpm --dir package test
BASELINE_BIN="$PWD/bin/baseline" pnpm --dir package exec baseline check --fast
```

## MCP

```sh
baseline init --register-openclaw
openclaw mcp set baseline '{"command":"baseline","args":["serve","mcp"]}'
baseline schedule install --at 09:00
```

Fast mode never executes the local agent. Full mode only executes an agent when
`--run-agent` or `BASELINE_RUN_AGENT=1` is set.
