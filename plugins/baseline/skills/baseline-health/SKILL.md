---
name: baseline-health
description: Use Baseline's local MCP tools to set up, run, inspect, accept, and schedule coding-agent health checks in Codex.
---

# Baseline Health

Use this skill when a Codex workspace needs a local coding-agent health check,
Good Baseline acceptance, drift comparison, or daily self-check through
Baseline.

## Requirements

- The `baseline` CLI must be installed and available on `PATH`.
- Install the CLI with `curl -fsSL https://trackbaseline.com/install.sh | sh`
  or from the GitHub Release assets before calling MCP tools.
- Verify `baseline --version` prints `baseline 0.1.0`, then run
  `baseline doctor` for read-only preflight before starting an eval.
- Baseline is local-first. Raw prompts, responses, local paths, and secrets
  must not be copied into cloud systems unless the operator explicitly enables
  redacted sync.

## Workflow

1. Start with `baseline --version` and `baseline doctor` if the CLI has not
   been verified in this environment. `baseline doctor` is read-only and does
   not send probe messages.
2. Start the first run with `baseline_setup` or CLI `baseline setup`. This
   configures the local workspace, writes Baseline-owned local state, and starts
   the default target eval in the background when needed.
3. For later checks, call `baseline_run`. It returns quickly with
   `run_status.run_id` while the eval continues.
4. Poll `baseline_report` with that run id until the report is completed.
5. Show the operator the report and responses before asking for an
   accept/reject/defer decision.
6. Accept a Good Baseline only after explicit operator approval:
   `baseline_accept` with `confirm: "accept <RUN_ID>"`, or CLI
   `baseline accept <RUN_ID> --confirm "accept <RUN_ID>"`.
7. Use `baseline_doctor` only for read-only preflight. It must not send agent
   probes or create a Good Baseline candidate.
8. Use `baseline_scrub_preview` before enabling sync or sharing any text
   externally.

## Daily Self-Check

1. Call `baseline_schedule` with `action: "status"` to inspect the configured
   schedule.
2. If the operator asks to install a daily check, call `baseline_schedule` with
   `action: "install"` and an `at` value such as `09:00`.
3. If the operator asks to trigger the scheduled check now, call
   `baseline_schedule` with `action: "run"`.

## Recovery

- If MCP startup fails because `baseline` is missing, install the CLI or point
  Codex at a built binary, then re-run `baseline --version`. Do not add a new
  MCP tool for version or preflight; the advertised local tool count stays seven.
- If a lifecycle run is still running, keep polling `baseline_report`.
- If a lifecycle run failed before writing a result row, inspect the stdout and
  stderr paths in the report, then ask before rerunning with `baseline_run`
  using `rerun_id` or CLI `baseline rerun <RUN_ID>`.
- If OpenClaw logs show `idleMs=60007`, `timeoutMs=60000`, or
  `turn_completion_idle_timeout`, report that as the OpenClaw Codex idle
  watchdog and run setup before starting a fresh eval.
- If `401 Unauthorized` appears with an `__OPENCLAW_REDACTED__` placeholder,
  treat it as an auth/env configuration problem, not a timeout.
