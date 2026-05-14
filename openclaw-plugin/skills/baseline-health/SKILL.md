# Baseline Health

Use this skill when an OpenClaw workspace needs a local coding-agent health
check, known-good marker, or drift comparison through Baseline.

## Workflow

1. Run `baseline_check` in `fast` mode for a local-only health check.
2. Run `baseline_schedule` with `action: "status"` to verify daily self-checks.
3. If the user asks to install the daily check, run `baseline_schedule` with `action: "install"` and an `at` time like `09:00`.
4. If the user asks to trigger the scheduled check now, run `baseline_schedule` with `action: "run"`.
5. If the run is acceptable, run `baseline_mark_known_good` with a clear label.
6. Later, run `baseline_compare` to inspect drift from the known-good run.

Safety notes:

- Fast mode never executes the agent.
- Full mode should only set `run_agent: true` after the user explicitly opts in.
- Daily schedule runs fast mode only.
- Use `baseline_scrub_preview` before enabling sync or sharing text externally.
