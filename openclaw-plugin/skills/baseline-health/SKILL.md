# Baseline Health

Use this skill when an OpenClaw workspace needs a local coding-agent health
check, Good Baseline acceptance, or drift comparison through Baseline.

## Workflow

1. Setup first with the CLI: `baseline setup` (or `baseline setup --register-openclaw` if the operator wants this agent to register the MCP server).
2. Prefer `baseline_setup` for first run, or CLI `baseline setup`. It writes Baseline-owned setup files, runs the real default target eval, and returns report paths.
3. For later runs, call `baseline_run`. Baseline sends real OpenClaw messages, records send/receive timestamps, and writes `REPORT.md` plus `RESPONSES.md`.
4. Call `baseline_report` and show the operator the markdown report and responses before asking for accept/reject/defer.
5. Accept a Good Baseline only after the user explicitly approves the run: `baseline_accept` with `confirm: "accept <RUN_ID>"`, or CLI `baseline accept <RUN_ID> --confirm "accept <RUN_ID>"`.
6. Keep at most three active Good Baselines. If the user wants a fourth, ask which slot to replace.
7. Later, call `baseline_report` to inspect drift from the latest accepted Good Baseline.

## Daily Self-Check

1. Run `baseline_schedule` with `action: "status"` to verify daily self-checks.
2. If the user asks to install the daily check, run `baseline_schedule` with `action: "install"` and an `at` time like `09:00`.
3. If the user asks to trigger the scheduled check now, run `baseline_schedule` with `action: "run"`. This runs the configured default eval, not a fake local-only probe.

Safety notes:

- `baseline_doctor` is read-only preflight and does not create a Good Baseline candidate.
- `baseline_run` should use real OpenClaw session metadata when available; do not invent token or cost numbers.
- Daily schedule runs the configured default target eval.
- Do not call retired `baseline_mark_known_good`. Use `baseline_accept` only after explicit user approval.
- Use `baseline_scrub_preview` before enabling sync or sharing text externally.
