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
- `KLAVIYO_PRIVATE_API_KEY`: checkout-start lifecycle events are skipped until configured.
- `KLAVIYO_REVISION`: defaults to `2026-04-15` when Klaviyo is configured.
- `BASELINE_MASTER_EMAIL`: optional owner notification destination for checkout-start events.

## 2026-05-19 Brand Landing Assets

- Worker static assets are now configured through `web/wrangler.jsonc` with `assets.directory = "./public"`.
- Current image assets live under `web/public/assets/` and are uploaded by Wrangler with the Worker.
- Deployed Worker version: `5cc879a3-983d-4e59-a620-e8abd8d70a99`
- Deployed URL: https://baseline-ai.ryan-borker.workers.dev
- Implementation commit deployed: `257c17f`
- Local verification path:

```sh
cd web
npm ci
npm run typecheck
npm run dev -- --port 8787
```

Visual routes checked locally:

- `/`
- `/blog`
- `/docs/mcp`
- `/#pro-monitoring`
- `/checkout/success`
- `/checkout/cancel`

Checkout behavior:

- `GET /api/checkout?plan=pro|team` still redirects to payment links when configured, otherwise creates a Stripe Checkout Session when Stripe secrets/price IDs are present.
- `POST /api/checkout` accepts `{ email, plan, successUrl, cancelUrl }` and returns `{ ok, url }` for the landing-page email form.
- Klaviyo checkout-start events are best-effort through `ctx.waitUntil`; they must not grant entitlement or block checkout.

Live deployment verification on 2026-05-19:

```sh
cd web
npm run deploy
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
curl -fsS https://baseline-ai.ryan-borker.workers.dev/ | rg -n "Baseline.ai|Keep coding agents|Pro monitoring"
curl -fsS https://baseline-ai.ryan-borker.workers.dev/blog | rg -n "Blog stub|Pro Account Architecture|field notes"
curl -I -fsS https://baseline-ai.ryan-borker.workers.dev/assets/baseline-court-robot.png
curl -sS -X POST https://baseline-ai.ryan-borker.workers.dev/api/checkout -H 'content-type: application/json' --data '{"email":"pilot@example.com","plan":"pro"}'
```

Results:

- Live health returned `{"ok":true,"db":true,"stripe":false,"token_required":true,"lifecycle_email":false}`.
- Landing page and blog stub served the new brand/documentation content.
- Uploaded hero image returned `HTTP/2 200`.
- Checkout fallback returned `{"ok":false,"error":"Stripe is not configured. Set STRIPE_SECRET_KEY and STRIPE_PRICE_ID_PRO/TEAM or payment links."}`.

## Live Smoke Test

Commands run:

```sh
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
./bin/baseline doctor
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
