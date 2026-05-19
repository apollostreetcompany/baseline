# @baseline-ai/cli

Small npm/pnpm wrapper for the Baseline v0.1 Go CLI and MCP server.

The wrapper finds an installed `baseline` binary and forwards all arguments to it.
If no binary is present, it downloads the matching macOS/Linux release tarball
from GitHub Releases, verifies `checksums.txt`, caches it under
`~/.cache/baseline-ai`, and then runs it.

## Install

```sh
pnpm add -g @baseline-ai/cli
baseline setup
```

Direct install without npm:

```sh
curl -fsSL https://trackbaseline.com/install.sh | sh
```

## Local OpenClaw Dogfood

Run exactly:

```sh
baseline setup
baseline report
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean-local
baseline compare
```

Use `baseline doctor` when you only want read-only runtime/repo/MCP/config preflight and do not want to send OpenClaw probe messages. `baseline run` and `baseline setup` send real probe messages to the configured default target, write markdown artifacts, and require operator confirmation before accepting. For OpenClaw targets, setup snapshots `~/.openclaw/openclaw.json` and ensures Codex app-server request and turn-idle timeouts are at least 900 seconds.

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
baseline rerun RUN_ID
baseline repair openclaw
baseline accept RUN_ID --confirm "accept RUN_ID" --label <label>
baseline good list
baseline config show
baseline config set api_token <token>
baseline schedule install --at 09:00
```

`baseline run` captures Baseline send/receive timestamps, stores local `RESPONSES.md`, and uses OpenClaw session metadata for tokens when available. Legacy `baseline check --fast|--full` remains for scripted compatibility. If OpenClaw logs show `turn_completion_idle_timeout` around 60 seconds, rerun `baseline setup` or `baseline install openclaw`; if logs show `__OPENCLAW_REDACTED__` with `401 Unauthorized`, treat it as child env/auth configuration rather than a timeout. `baseline report RUN_ID --json` exits `0` for completed, `2` while running, and `1` for failed lifecycle runs; use `baseline rerun RUN_ID` only after reviewing the logged failure.
