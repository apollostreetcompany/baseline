# Baseline Deployment Notes

## Current Production

- Worker: `baseline-ai`
- URL: https://baseline-ai.ryan-borker.workers.dev
- Current Version ID: `4f1b94a0-543a-4cb2-8207-62825fb29594`
- Current production source branch: `codex/feat/bead-27-landing-a-brand-os`
- Current production source worktree: `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-landing-a-brand-os`
- Notes: current production combines the Bead 25 cloud account/remote MCP surface with the Bead 27 `landing-a` homepage redesign.

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

## 2026-05-19 Pro Cloud Accounts And Remote MCP

Bead 25 adds the cloud-backed Pro account path on the existing Cloudflare Worker and Neon database. The local runner remains the probe executor; cloud is now the account, billing, history, comparison, and remote MCP surface.

New Worker routes:

- `POST /api/admin/invites`
- `POST /api/auth/magic-link`
- `POST|GET /api/auth/consume`
- `GET /api/account/status`
- `GET|POST /api/workspaces`
- `POST /api/tokens`
- `POST /api/tokens/revoke`
- `GET /api/history`
- `GET /api/hotspots`
- `GET /api/compare`
- `POST /api/billing/portal`
- `POST /api/stripe/webhook`
- `POST /mcp`
- `GET /.well-known/oauth-protected-resource`
- `GET /.well-known/oauth-authorization-server`

Required production secrets before paid pilot:

- `DATABASE_URL`
- `BASELINE_ADMIN_TOKEN`
- `MAGIC_LINK_SECRET`
- `TOKEN_HMAC_SECRET`
- `STRIPE_SECRET_KEY`
- `STRIPE_PRICE_ID_PRO`
- `STRIPE_WEBHOOK_SECRET`

Optional or rollout-dependent secrets:

- `STRIPE_PRICE_ID_TEAM`
- `STRIPE_PAYMENT_LINK_PRO`
- `STRIPE_PAYMENT_LINK_TEAM`
- `KLAVIYO_PRIVATE_API_KEY`
- `KLAVIYO_REVISION`
- `BASELINE_MASTER_EMAIL`
- `OPENAI_API_KEY`
- `OPENAI_EVALUATOR_MODEL`
- `MAGIC_LINK_DEV_ECHO` only for local/staging debug; do not enable in production.

Stripe webhook configuration:

- Endpoint: `https://baseline-ai.ryan-borker.workers.dev/api/stripe/webhook`
- Events: `checkout.session.completed`, `customer.subscription.created`, `customer.subscription.updated`, `customer.subscription.deleted`
- Verification: Worker reads the raw body and validates `Stripe-Signature` against `STRIPE_WEBHOOK_SECRET`.
- Idempotency: `stripe_events.stripe_event_id` is unique; duplicate events return success without reprocessing.

Remote MCP configuration:

- Endpoint: `https://baseline-ai.ryan-borker.workers.dev/mcp`
- Transport: HTTP JSON-RPC endpoint shaped for remote MCP clients.
- Auth: Bearer account session created through magic-link auth. Unauthenticated calls return a `WWW-Authenticate` challenge and protected-resource metadata.
- Tools: `baseline_account`, `baseline_workspaces`, `baseline_history`, `baseline_hotspots`, `baseline_compare`, `baseline_subscription`, `baseline_admin`.
- Guardrails: no direct billing cancellation, no token revocation without confirmation, no destructive raw export path, and every mutation writes `audit_log`.

Pro ingest behavior:

- Temporary dogfood `BASELINE_API_TOKEN` still works.
- Workspace Pro tokens use prefix plus HMAC hash in `api_tokens`; raw tokens are returned only once.
- `/api/runs` resolves `account_id` and `workspace_id` from the server-side token row; callers cannot spoof account/workspace IDs.
- `baseline_runs` keeps legacy dashboard fields and adds nullable `account_id`, `workspace_id`, `expires_at`, `account_private_payload`, and `comparison_scope`.

Mac app companion from the Bead 25 worktree:

- Source: `macos/BaselineHotspots` in the Bead 25 implementation worktree.
- Build: `make mac-build` in that worktree.
- Storage: macOS Keychain for session token and OpenRouter API key.
- Data source: remote MCP first; local SQLite is intentionally not the primary source. The Bead 27 landing worktree only carries the Worker deploy surface needed for the public page.

Preflight before deploy:

```sh
make verify
cd web && npm run deploy
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
curl -fsS -X POST https://baseline-ai.ryan-borker.workers.dev/mcp -H 'content-type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Expected unauthenticated MCP smoke result:

- HTTP `401`
- `WWW-Authenticate` header present
- JSON body includes `authentication_required`

Rollback:

- Cloudflare Workers rollback to the previous deployment version from Wrangler or the Cloudflare dashboard.
- If a schema bug is found, leave additive tables in place and rollback Worker code. Do not drop account/billing tables without explicit data-retention approval.

Deployment result:

- First deploy version `c8adbd91-0139-461a-953c-91b76c9085be` succeeded but uploaded an untracked `.DS_Store` static asset from `web/public`.
- `.DS_Store` was added to `.gitignore`, the stray local metadata files were removed, and the Worker was redeployed.
- Clean deploy version: `46e6414b-d540-4373-b0bf-c140c1f80334`.
- Skill-audit refinement deploy version: `dfc2198f-9151-4a64-8511-4e25d3c2d529`.

Live smoke on 2026-05-19:

```sh
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
curl -i -sS https://baseline-ai.ryan-borker.workers.dev/mcp \
  -H 'content-type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
curl -fsS https://baseline-ai.ryan-borker.workers.dev/.well-known/oauth-protected-resource
curl -fsS https://baseline-ai.ryan-borker.workers.dev/docs/mcp | rg -n "Remote MCP|/mcp|magic-link"
curl -sS -X POST https://baseline-ai.ryan-borker.workers.dev/api/checkout \
  -H 'content-type: application/json' \
  --data '{"email":"pilot@example.com","plan":"pro"}'
```

Results:

- Health returned `db:true`, `stripe:false`, `token_required:true`, `pro_auth:false`, `pro_tokens:false`, `stripe_webhook:false`.
- Unauthenticated `/mcp` returned HTTP `401` with `WWW-Authenticate` and `authentication_required`.
- Protected-resource metadata returned the remote MCP resource and authorization-server metadata URL.
- `/docs/mcp` includes the remote MCP setup section.
- Checkout still fails closed with the expected unconfigured Stripe response until production billing secrets are set.

Skill audit:

- Review artifact: `docs/reviews/2026-05-19-bead-25-skill-audit.md`.
- Applied fixes from the audit: Stripe `invoice.payment_failed` dunning state, lifecycle outbox rows, token scope enforcement, past-due grace without new token creation, and more discoverable MCP tool schemas.

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

## 2026-05-19 Landing A Redesign And BrandOS Repair

Bead 27 replaces the homepage with a Worker-native port of `/Users/kikimac/Downloads/baseline.zip` `landing-a.jsx` and reuses the supplied court robot image assets already present under `web/public/assets/`. The deploy intentionally preserves Bead 25 cloud account, token, webhook, history, comparison, and remote MCP routes.

Deployment result:

- Worker: `baseline-ai`
- URL: https://baseline-ai.ryan-borker.workers.dev
- Version ID: `4f1b94a0-543a-4cb2-8207-62825fb29594`
- Source branch: `codex/feat/bead-27-landing-a-brand-os`
- Source worktree: `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-landing-a-brand-os`

Local validation:

```sh
cd /Users/kikimac/.hermes/repos/apollostreetcompany/baseline-landing-a-brand-os/web
npm run typecheck
npm audit --audit-level=high
npm run dev -- --port 8787
cd ..
go test ./...
```

Local smoke:

- `/` contained `Your agent forgot`, `Three agents`, `Fourteen probes`, and `In the line`.
- `/docs/mcp` continued to serve the remote MCP docs.
- `/assets/baseline-court-serve.png` returned `200`.
- `POST /api/checkout` failed closed with the expected unconfigured Stripe response.
- Playwright layout checks at `1280x900` and `390x844` found no horizontal overflow and no broken images.

Live smoke:

```sh
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
curl -fsS https://baseline-ai.ryan-borker.workers.dev/ | rg -n "Your agent forgot|Three agents|Fourteen probes|In the line"
curl -fsS https://baseline-ai.ryan-borker.workers.dev/docs/mcp | rg -n "Remote MCP|/mcp|magic-link"
curl -I -fsS https://baseline-ai.ryan-borker.workers.dev/assets/baseline-court-serve.png
curl -sS -X POST https://baseline-ai.ryan-borker.workers.dev/api/checkout \
  -H 'content-type: application/json' \
  --data '{"email":"pilot@example.com","plan":"pro"}'
```

Results:

- Health returned `db:true`, `stripe:false`, `token_required:true`, `pro_auth:false`, `pro_tokens:false`, `stripe_webhook:false`.
- Landing page served the `landing-a` copy and layout.
- Remote MCP docs remained live.
- Hero image returned `HTTP/2 200`.
- Checkout still fails closed until Stripe secrets are configured.

BrandOS machine repair:

- Skill path: `/Users/kikimac/.hermes/repos/apollostreetcompany/skills-library/skills/brand-os-studio`
- Repair: scripts now use `python3`, avoid PyYAML, and validate `.prose` workflows with a bundled fallback when no `prose` CLI exists.
- Validation: `scripts/audit_skill_pack.py`, `scripts/validate_prose.py workflows`, `bash scripts/compile_prose.sh`, `scripts/check_stage_gates.py examples/shogun-sauce/workspace`, `python3 -m py_compile`, `bash -n`, `make verify-library`, and `make verify-codex` passed.

Rollback:

- Roll back the Worker to version `dfc2198f-9151-4a64-8511-4e25d3c2d529` to restore the previous Bead 25 production surface before the Landing A homepage redesign.

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
