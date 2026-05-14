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
baseline bootstrap --openclaw
baseline bootstrap preview
baseline bootstrap run
baseline bootstrap accept --label clean-local
baseline compare
```

Use `baseline check --fast` when you only want local runtime/repo/MCP checks and do not want to send OpenClaw probe messages.
`baseline bootstrap run` requires a recent preview receipt, defaults to the 14-question Baseline Core pack, and accepts `--preview-id <id>` from the preview output when you want an exact receipt match. Use `--packs enabled` or `--packs all` only after previewing the wider packs.

For local development from this repository:

```sh
go build -o bin/baseline ./cmd/baseline
pnpm --dir package test
BASELINE_BIN="$PWD/bin/baseline" pnpm --dir package exec baseline check --fast
```

## CLI Shape

```sh
baseline bootstrap --openclaw
baseline good accept [RUN_ID] --label <label>
baseline good list
baseline config show
baseline config set api_token <token>
baseline schedule install --at 09:00
```

Fast mode never executes the local agent. Bootstrap run and `check --full --run-agent` send real OpenClaw messages, capture Baseline send/receive timestamps, and use OpenClaw session metadata for tokens when available.
