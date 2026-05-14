# Baseline Health

Use this skill when an OpenClaw workspace needs a local coding-agent health
check, Good Baseline acceptance, or drift comparison through Baseline.

## Workflow

1. Bootstrap first with the CLI: `baseline bootstrap --openclaw`.
2. Run `baseline_check` in `fast` mode for a local-only health check.
3. For real OpenClaw behavior metrics, run `baseline bootstrap preview` before `baseline bootstrap run`, or run `baseline check --full --run-agent --packs baseline`; Baseline sends real OpenClaw messages and records send/receive timestamps. The default bootstrap run is the 14-question Baseline Core pack.
4. Accept a Good Baseline only after the user explicitly approves the run: `baseline bootstrap accept [RUN_ID] --label <label>`.
5. Keep at most three active Good Baselines. If the user wants a fourth, ask which slot to replace.
6. Later, run `baseline_compare` to inspect drift from the latest accepted Good Baseline.

## Daily Self-Check

1. Run `baseline_schedule` with `action: "status"` to verify daily self-checks.
2. If the user asks to install the daily check, run `baseline_schedule` with `action: "install"` and an `at` time like `09:00`.
3. If the user asks to trigger the scheduled check now, run `baseline_schedule` with `action: "run"`.

Safety notes:

- Fast mode never executes the agent.
- Full mode should use real OpenClaw session metadata when available; do not invent token or cost numbers.
- Daily schedule runs fast mode only.
- Do not call retired `baseline_mark_known_good`. Use `baseline_bootstrap` with `action: "accept"` only after explicit user approval, and prefer the `baseline bootstrap accept` CLI flow for v0.1.
- Use `baseline_scrub_preview` before enabling sync or sharing text externally.
