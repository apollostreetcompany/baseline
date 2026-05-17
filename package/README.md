# @baseline-ai/cli

Small npm/pnpm wrapper for the Baseline v0.1 Go CLI and MCP server.

This package does not vendor the Go application yet. It finds an installed `baseline`
binary and forwards all arguments to it. If no binary is found, it exits with the
supported install commands.

## Install

```sh
pnpm add -g @baseline-ai/cli
go install github.com/apollostreetcompany/baseline/cmd/baseline@latest
```

## Local OpenClaw Dogfood

Run exactly:

```sh
baseline setup
baseline report
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
baseline compare
```

Use `baseline doctor` when you only want read-only runtime/repo/MCP/config preflight and do not want to send OpenClaw probe messages. `baseline run` and `baseline setup` send real probe messages to the configured default target, write markdown artifacts, and require operator confirmation before accepting.

For local development from this repository:

```sh
go build -o bin/baseline ./cmd/baseline
pnpm --dir package test
BASELINE_BIN="$PWD/bin/baseline" pnpm --dir package exec baseline doctor
```

## CLI Shape

```sh
baseline setup
baseline run
baseline report [RUN_ID]
baseline accept RUN_ID --confirm "accept RUN_ID" --label <label>
baseline good list
baseline config show
baseline config set api_token <token>
baseline schedule install --at 09:00
```

`baseline run` captures Baseline send/receive timestamps, stores local `RESPONSES.md`, and uses OpenClaw session metadata for tokens when available. Legacy `baseline check --fast|--full` remains for scripted compatibility.
