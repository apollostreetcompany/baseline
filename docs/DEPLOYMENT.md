# Baseline Deployment Notes

## 2026-05-14 Cloudflare Deploy

- Worker: `baseline-ai`
- URL: https://baseline-ai.ryan-borker.workers.dev
- Version ID: `b143ba10-4546-4d89-8ae5-3c5d920ec326`
- Commit deployed: `73346f7 feat(bead-11): add distribution packages`

## 2026-05-14 MCP Schedule Docs Deploy

- Worker: `baseline-ai`
- URL: https://baseline-ai.ryan-borker.workers.dev
- Version ID: `3999eaaf-d845-487f-a6a7-beaf41027773`
- Change: MCP docs now refer to `baseline_schedule` instead of the hidden legacy config tool.

Configured Worker secrets:

- `DATABASE_URL`
- `BASELINE_API_TOKEN`
- `BASELINE_ADMIN_TOKEN`

Missing optional secrets:

- `OPENAI_API_KEY`: evaluator uses `local-heuristic` mode until set.
- `OPENAI_EVALUATOR_MODEL`: defaults to `gpt-5` when `OPENAI_API_KEY` is present.
- Stripe secrets/payment links: checkout still reports Stripe as unconfigured.

## Live Smoke Test

Commands run:

```sh
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
./bin/baseline check --fast
./bin/baseline sync status
./bin/baseline sync push
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/runs/latest
curl -fsS -X POST https://baseline-ai.ryan-borker.workers.dev/api/admin/evaluate
```

Results:

- Health API returned `db:true`, `stripe:false`, `token_required:true`.
- Local run `run_dii09roqdp20` synced successfully.
- Live latest-run API rendered `run_dii09roqdp20` with score `90`, status `warning`, mode `fast`, and `5` checks.
- Admin question-set API seeded `baseline-core@2026-05-14`.
- Evaluator stored `99161224-a275-48b6-b7a1-489b9f73a916` using `local-heuristic`, score `92`, verdict `pass`.

## Admin Access

For dogfood, `BASELINE_ADMIN_TOKEN` is currently set to the same local token used by `baseline sync on`. This keeps the page usable without introducing another secret file, but it should be split before any external pilot.

## Local Daily Schedule

Baseline is installed on this machine as a launchd user agent:

- Label: `ai.baseline.daily`
- Plist: `~/Library/LaunchAgents/ai.baseline.daily.plist`
- Time: `09:00` local
- Program: `/opt/homebrew/bin/baseline schedule run`

OpenClaw can trigger the same path through MCP:

```json
{"name":"baseline_schedule","arguments":{"action":"run"}}
```

Smoke result:

- Run: `run_dii2iaoed2xk`
- Score: `90`
- Status: `warning`
- Cloud synced: `true`
