# Baseline Health

Use this skill when an OpenClaw workspace needs a local coding-agent health
check, Good Baseline acceptance, or drift comparison through Baseline.

## Workflow

1. Setup first with the CLI: `baseline setup` (or `baseline setup --register-openclaw` if the operator wants this agent to register the MCP server).
2. Prefer `baseline_setup` for first run, or CLI `baseline setup`. MCP setup starts the real default target eval in the background and returns `run_status.run_id`.
3. For later MCP runs, call `baseline_run`; it returns quickly with `run_status.run_id` while the eval continues. Poll `baseline_report` for that run id until it returns the completed report/responses.
4. Show the operator the markdown report and responses before asking for accept/reject/defer.
5. Accept a Good Baseline only after the user explicitly approves the run: `baseline_accept` with `confirm: "accept <RUN_ID>"`, or CLI `baseline accept <RUN_ID> --confirm "accept <RUN_ID>"`.
6. Keep at most three active Good Baselines. If the user wants a fourth, ask which slot to replace.
7. Later, call `baseline_report` to inspect drift from the latest accepted Good Baseline.
8. If a lifecycle run failed before writing a result row, read its stdout/stderr paths, run `baseline repair openclaw` for OpenClaw targets, then ask before rerunning with CLI `baseline rerun <RUN_ID>` or MCP `baseline_run` with `rerun_id`.

## Daily Self-Check

1. Run `baseline_schedule` with `action: "status"` to verify daily self-checks.
2. If the user asks to install the daily check, run `baseline_schedule` with `action: "install"` and an `at` time like `09:00`.
3. If the user asks to trigger the scheduled check now, run `baseline_schedule` with `action: "run"`. Through MCP this starts the configured default eval in the background, not a fake local-only probe.

Safety notes:

- `baseline_doctor` is read-only preflight and does not create a Good Baseline candidate.
- `baseline_setup` and `baseline install openclaw` ensure OpenClaw Codex app-server request and turn-idle timeouts are at least 900 seconds. If logs show `idleMs=60007`, `timeoutMs=60000`, or `turn_completion_idle_timeout`, report that as the OpenClaw Codex idle watchdog and start a fresh run after setup.
- `401 Unauthorized` with `__OPENCLAW_REDACTED__` in ACP child Codex streams or memory search is an auth/env config failure, not a true timeout. Do not remove Google/Gemini search or background API configuration to fix it.
- `baseline report RUN_ID --json` exits `0` for completed, `2` for still running, and `1` for failed lifecycle runs. Do not treat JSON output alone as success.
- `baseline_run` should use real OpenClaw session metadata when available; do not invent token or cost numbers.
- Daily schedule runs the configured default target eval.
- Do not call retired `baseline_mark_known_good`. Use `baseline_accept` only after explicit user approval.
- Use `baseline_scrub_preview` before enabling sync or sharing text externally.
