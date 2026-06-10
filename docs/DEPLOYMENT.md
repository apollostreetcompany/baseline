# Baseline Deployment Notes

## Current Production

- Worker: `baseline-ai`
- URL: https://trackbaseline.com
- Fallback Worker URL: https://baseline-ai.ryan-borker.workers.dev
- Current Version ID: `7940fc3a-f89e-4972-9352-e77424b541a6`
- Current production source branch: `codex/feat/bead-34-commercial-viability`
- Current production source worktree: `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability`
- Notes: current production combines the Bead 25 cloud account/remote MCP surface, Bead 33 SEO/lead-magnet acquisition surface, and Bead 34 commercial-viability checkout/pilot path.

## 2026-05-14 Cloudflare Deploy

- Worker: `baseline-ai`
- URL: https://trackbaseline.com
- Version ID: `b143ba10-4546-4d89-8ae5-3c5d920ec326`
- Commit deployed: `73346f7 feat(bead-11): add distribution packages`

## 2026-05-14 MCP Schedule Docs Deploy

- Worker: `baseline-ai`
- URL: https://trackbaseline.com
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

- Endpoint: `https://trackbaseline.com/api/stripe/webhook`
- Events: `checkout.session.completed`, `customer.subscription.created`, `customer.subscription.updated`, `customer.subscription.deleted`
- Verification: Worker reads the raw body and validates `Stripe-Signature` against `STRIPE_WEBHOOK_SECRET`.
- Idempotency: `stripe_events.stripe_event_id` is unique; duplicate events return success without reprocessing.

Remote MCP configuration:

- Endpoint: `https://trackbaseline.com/mcp`
- Transport: HTTP JSON-RPC endpoint shaped for remote MCP clients.
- Auth: Bearer account session created through magic-link auth. Unauthenticated calls return a `WWW-Authenticate` challenge and protected-resource metadata.
- Tools: `baseline_account`, `baseline_workspaces`, `baseline_history`, `baseline_hotspots`, `baseline_compare`, `baseline_subscription`, `baseline_admin`.
- Guardrails: no direct billing cancellation, no token revocation without confirmation, no destructive raw export path, and every mutation writes `audit_log`.

Pro ingest behavior:

- Temporary dogfood `BASELINE_API_TOKEN` still works.
- Workspace Pro tokens use prefix plus HMAC hash in `api_tokens`; raw tokens are returned only once.
- `/api/runs` resolves `account_id` and `workspace_id` from the server-side token row; callers cannot spoof account/workspace IDs.
- `baseline_runs` keeps legacy dashboard fields and adds nullable `account_id`, `workspace_id`, `expires_at`, `account_private_payload`, and `comparison_scope`.

Mac app:

- Source: `macos/BaselineHotspots`
- Build: `make mac-build`
- Storage: macOS Keychain for session token and OpenRouter API key.
- Data source: remote MCP first; local SQLite is intentionally not the primary source.

Preflight before deploy:

```sh
make verify-all
cd web && npm run deploy
curl -fsS https://trackbaseline.com/api/health
curl -fsS -X POST https://trackbaseline.com/mcp -H 'content-type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Expected unauthenticated MCP smoke result:

- HTTP `401`
- `WWW-Authenticate` header present
- JSON body includes `authentication_required`

Rollback:

- Cloudflare Workers rollback to the previous deployment version from Wrangler or the Cloudflare dashboard.
- If a schema bug is found, leave additive tables in place and rollback Worker code. Do not drop account/billing tables without explicit data-retention approval.

## 2026-06-10 Commercial Viability Gate

Bead 34 closes the first-customer gaps found by `subreview`:

- `/checkout/success` is now an onboarding bridge: it checks the returned Stripe session, requests a magic link for the checkout email, and shows the workspace-token / `baseline sync on` path.
- `/api/checkout/session` resolves the actual Stripe session and scopes entitlement hints to that session/account. It no longer returns the latest global entitlement.
- Paid checkout now requires email before Stripe session creation. Payment links are disabled for onboarding because they cannot guarantee account metadata and entitlement provisioning.
- Team checkout uses the same email-first form as Pro.
- Stripe webhook completion falls back from checkout metadata to known Stripe customer row, then to Stripe customer email, before granting entitlement.
- `/admin` includes an "Invite pilot" panel that calls `POST /api/admin/invites` with optional pilot entitlement.
- Public pricing includes a `pilot_request` form; `/api/admin/leads` returns both lead-magnet and pilot requests with pagination.
- Public `/api/runs/latest` and `/api/runs/timeline` exclude account-private Pro runs; dashboard demo rows are labeled as examples.

Required paid-pilot smoke:

```sh
curl -fsS https://trackbaseline.com/api/health
curl -sS -X POST https://trackbaseline.com/api/checkout \
  -H 'content-type: application/json' \
  --data '{"email":"pilot@example.com","plan":"pro"}'
curl -sS -X POST https://trackbaseline.com/api/events \
  -H 'content-type: application/json' \
  --data '{"type":"pilot_request","email":"pilot@example.com","plan":"pro","context":"checkout smoke"}'
curl -i -sS "https://trackbaseline.com/api/admin/leads" \
  -H "authorization: Bearer REDACTED_ADMIN_TOKEN"
```

Klaviyo verification before announcing paid pilot:

- `Baseline Lead Magnet Requested` flow sends the promised resource follow-up or pilot prompt.
- `Baseline Pilot Requested` notifies the operator and/or starts the manual pilot follow-up.
- `Baseline Magic Link` sends account login links for invites and checkout success.
- `Baseline Subscription Started` sends the paid onboarding email with the magic link when configured.

Rollback:

- Previous deployed Bead 33 version: `df4d479d-9fbd-4f8a-af50-b2f3a88253a8`.
- Roll back Worker code from Cloudflare Workers versions if checkout/session behavior blocks production.
- Database changes are additive or query-only; do not drop lead/account/billing tables during rollback.

Bead 34 deployment result:

- Worker version `7940fc3a-f89e-4972-9352-e77424b541a6` deployed to `https://trackbaseline.com`, `https://www.trackbaseline.com`, and the workers.dev fallback.
- Live `/api/health` returned `db:true`, `stripe:true`, `lifecycle_email:true`, `pro_auth:true`, `pro_tokens:true`, and `stripe_webhook:true`.
- Live homepage exposes example scoreboard copy, Team email-first checkout form, and `/#pilot-request`.
- Live `/checkout/success?session_id=cs_test_fake` renders the magic-link and workspace-token onboarding path; `/api/checkout/session` correctly fails closed without a real Stripe session/secret context.
- Live `GET /api/checkout?plan=team` returns a human-readable email-required page instead of creating an unattributed Stripe session.
- Live `POST /api/events` rejects `pilot_request` without a valid email and accepts a synthetic `codex-smoke+bead34@example.com` pilot request with Klaviyo configured.
- Protected `/api/admin/leads` returned `401` from this shell because the local deploy env did not contain `BASELINE_ADMIN_TOKEN`; unauthenticated rejection was verified, but live lead readback remains to be confirmed with the operator/admin token.

Bead 25 deployment result:

- First deploy version `c8adbd91-0139-461a-953c-91b76c9085be` succeeded but uploaded an untracked `.DS_Store` static asset from `web/public`.
- `.DS_Store` was added to `.gitignore`, the stray local metadata files were removed, and the Worker was redeployed.
- Clean deploy version: `46e6414b-d540-4373-b0bf-c140c1f80334`.
- Skill-audit refinement deploy version: `dfc2198f-9151-4a64-8511-4e25d3c2d529`.

Live smoke on 2026-05-19:

```sh
curl -fsS https://trackbaseline.com/api/health
curl -i -sS https://trackbaseline.com/mcp \
  -H 'content-type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
curl -fsS https://trackbaseline.com/.well-known/oauth-protected-resource
curl -fsS https://trackbaseline.com/docs/mcp | rg -n "Remote MCP|/mcp|magic-link"
curl -sS -X POST https://trackbaseline.com/api/checkout \
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

## 2026-05-19 Trackbaseline Custom Domain

Bead 28 attaches the Worker to the public launch domain and makes that domain the canonical application origin.

Cloudflare zone:

- Domain: `trackbaseline.com`
- Nameservers: `bingo.ns.cloudflare.com`, `harlan.ns.cloudflare.com`
- Worker custom domains: `trackbaseline.com`, `www.trackbaseline.com`
- Fallback workers.dev route: `https://baseline-ai.ryan-borker.workers.dev`
- `APP_URL`: `https://trackbaseline.com`
- Preview URLs: disabled explicitly.

Deployment result:

- Worker: `baseline-ai`
- URL: https://trackbaseline.com
- Version ID: `0d0924c3-5c8e-4029-9327-369a73588786`

DNS verification:

```sh
dig +short NS trackbaseline.com
dig +short A trackbaseline.com
dig +short AAAA trackbaseline.com
dig +short A www.trackbaseline.com
dig +short AAAA www.trackbaseline.com
```

Results:

- Nameservers returned `bingo.ns.cloudflare.com` and `harlan.ns.cloudflare.com`.
- Apex and `www` returned Cloudflare A/AAAA records.

Live smoke:

```sh
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://trackbaseline.com/ | rg -n "Your agent forgot|Three agents|Fourteen probes|In the line"
curl -fsS https://trackbaseline.com/docs/mcp | rg -n "Remote MCP|https://trackbaseline.com/mcp|magic-link"
curl -i -sS https://trackbaseline.com/mcp \
  -H 'content-type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
curl -fsS https://trackbaseline.com/.well-known/oauth-protected-resource
curl -I -fsS https://trackbaseline.com/assets/baseline-court-serve.png
curl -sS -X POST https://trackbaseline.com/api/checkout \
  -H 'content-type: application/json' \
  --data '{"email":"pilot@example.com","plan":"pro"}'
curl -fsS https://www.trackbaseline.com/api/health
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
```

Results:

- Apex health returned `db:true`, `stripe:false`, `token_required:true`, `pro_auth:false`, `pro_tokens:false`, `stripe_webhook:false`.
- Landing page served the Landing A copy and imagery.
- Remote MCP docs and OAuth protected-resource metadata use `https://trackbaseline.com`.
- Unauthenticated `/mcp` returned HTTP `401` with a `WWW-Authenticate` challenge pointing at `https://trackbaseline.com/.well-known/oauth-protected-resource`.
- Hero image returned `HTTP/2 200`.
- Checkout still fails closed until Stripe secrets are configured.
- `www.trackbaseline.com` and the fallback workers.dev route both returned healthy Worker responses.

Rollback:

- Preferred code rollback: `cd web && npx wrangler rollback 4f1b94a0-543a-4cb2-8207-62825fb29594`
- If the domain attachment itself is wrong, remove the `routes` entries from `web/wrangler.jsonc`, redeploy, and verify the fallback workers.dev route.

## 2026-05-19 Brand Landing Assets

- Worker static assets are now configured through `web/wrangler.jsonc` with `assets.directory = "./public"`.
- Current image assets live under `web/public/assets/` and are uploaded by Wrangler with the Worker.
- Deployed Worker version: `5cc879a3-983d-4e59-a620-e8abd8d70a99`
- Deployed URL: https://trackbaseline.com
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
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://trackbaseline.com/ | rg -n "Baseline.ai|Keep coding agents|Pro monitoring"
curl -fsS https://trackbaseline.com/blog | rg -n "Blog stub|Pro Account Architecture|field notes"
curl -I -fsS https://trackbaseline.com/assets/baseline-court-robot.png
curl -sS -X POST https://trackbaseline.com/api/checkout -H 'content-type: application/json' --data '{"email":"pilot@example.com","plan":"pro"}'
```

Results:

- Live health returned `{"ok":true,"db":true,"stripe":false,"token_required":true,"lifecycle_email":false}`.
- Landing page and blog stub served the new brand/documentation content.
- Uploaded hero image returned `HTTP/2 200`.
- Checkout fallback returned `{"ok":false,"error":"Stripe is not configured. Set STRIPE_SECRET_KEY and STRIPE_PRICE_ID_PRO/TEAM or payment links."}`.

## 2026-05-19 Distribution Activation

Baseline distribution is intentionally separate from Pro billing:

- Free local runner: `https://trackbaseline.com/install.sh` installs the latest release binary.
- Paid Pro account: Stripe Checkout grants hosted history, workspace tokens, remote MCP account operations, monitoring, and billing-backed retention.
- The binary should not be paywalled for the first launch; paywalling the executable would increase install friction before users see a local drift report.

Release preflight:

```sh
make verify-all
make plugin-validate
bash scripts/build-release.sh
ls dist
```

Release publish:

```sh
git tag v0.1.0
git push origin v0.1.0
gh release view v0.1.0
```

Install smoke after the release exists:

```sh
tmp_home="$(mktemp -d)"
HOME="$tmp_home" curl -fsSL https://trackbaseline.com/install.sh | HOME="$tmp_home" sh
"$tmp_home/.local/bin/baseline" doctor
```

Rollback:

- Delete or mark the bad GitHub Release as prerelease.
- Publish a fixed tag and update public docs to the new pinned `BASELINE_VERSION` if needed.
- Existing installed binaries remain local; Pro sync can be disabled with `baseline sync off` if a cloud issue is discovered.

## 2026-05-19 Production Pro Secret Activation

Secrets are configured through Wrangler and must never be printed in logs. On 2026-05-19 the production Worker was configured with:

- Stripe Checkout secret key from Hermes `.env`.
- Pro monthly price: `price_1TYkoGG0PuTic3wedLLYoZYS` at `$39/mo`.
- Team monthly price: `price_1TYkoHG0PuTic3weoX9qhMFP` at `$129/mo`.
- Stripe webhook endpoint: `we_1TYkp0G0PuTic3wegF1qVMpx` for `https://trackbaseline.com/api/stripe/webhook`.
- Generated magic-link and token HMAC secrets.
- Klaviyo private API key from Hermes `.env`.

Verification:

```sh
curl -fsS https://trackbaseline.com/api/health
curl -sS -X POST https://trackbaseline.com/api/checkout \
  -H 'content-type: application/json' \
  --data '{"email":"pilot@example.com","plan":"pro"}'
curl -i -sS -X POST https://trackbaseline.com/api/stripe/webhook \
  -H 'content-type: application/json' \
  --data '{}'
```

Expected:

- Health returns `stripe:true`, `lifecycle_email:true`, `pro_auth:true`, `pro_tokens:true`, and `stripe_webhook:true`.
- Checkout returns a Stripe Checkout URL without completing payment.
- Unsigned webhook requests fail closed with `400` or `401`.

Deployment result:

- Worker version `e38523fc-d11a-41d9-b05e-6dcef5f4b5f0` serves the updated install docs and `/install.sh` asset on `trackbaseline.com`.

## 2026-05-19 DataFast Funnel Analytics

DataFast is configured for the public launch share loop.

- Website domain: `trackbaseline.com`
- Website id: `6a0c48aa9a21aee7bf04cf6e`
- Tracking id: `dfid_PYprhfTkwwQKhkzRUhVtO`
- Tracking script: loaded in the Worker HTML `<head>` on all pages.

Tracked launch goals:

- `scroll_to_scoreboard`
- `scroll_to_probes`
- `scroll_to_pricing`
- `scroll_to_final_cta`
- `install_click`
- `docs_click`
- `dashboard_click`
- `blog_click`
- `checkout_start`
- `checkout_redirect`
- `checkout_return_success`
- `checkout_return_cancel`

Created DataFast CLI funnels:

- `Baseline install funnel`: `/` pageview -> `scroll_to_pricing` -> `install_click`
- `Baseline Pro funnel`: `/` pageview -> `scroll_to_pricing` -> `checkout_start` -> `checkout_return_success`

CLI reporting:

```sh
export DATAFAST_TOKEN="dft_..."
make analytics-report
DATAFAST_PERIOD=last24h make analytics-report
```

Do not commit or print DataFast tokens. Prefer `DATAFAST_TOKEN` in the shell session, 1Password, Keychain, or CI secret storage.

## 2026-05-25 Robot Favicon

Baseline uses the existing court robot photo as the browser/app icon source. The generated assets live in `web/public/`:

- `favicon.ico`
- `favicon-16x16.png`
- `favicon-32x32.png`
- `apple-touch-icon.png`
- `icon-192.png`
- `icon-512.png`
- `site.webmanifest`

The shared Worker layout links these assets from every HTML page. Local smoke:

```sh
cd web
npm run dev -- --port 8787
curl -fsS http://localhost:8787/ | rg 'rel="icon"|apple-touch-icon|site.webmanifest'
curl -I http://localhost:8787/favicon.ico
```

Live smoke after deploy:

```sh
curl -I https://trackbaseline.com/favicon.ico
curl -fsS https://trackbaseline.com/ | rg 'rel="icon"|apple-touch-icon|site.webmanifest'
curl -fsS https://trackbaseline.com/site.webmanifest
```

Deployment result:

- Worker version `b4f73e11-7540-4e97-8112-7698467b0484` uploaded seven favicon/app-icon assets and serves `/favicon.ico` with `HTTP 200`.

## 2026-05-19 Landing A Redesign And BrandOS Repair

Bead 27 replaces the homepage with a Worker-native port of `/Users/kikimac/Downloads/baseline.zip` `landing-a.jsx` and reuses the supplied court robot image assets already present under `web/public/assets/`. The deploy intentionally preserves Bead 25 cloud account, token, webhook, history, comparison, and remote MCP routes.

Deployment result:

- Worker: `baseline-ai`
- URL: https://trackbaseline.com
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
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://trackbaseline.com/ | rg -n "Your agent forgot|Three agents|Fourteen probes|In the line"
curl -fsS https://trackbaseline.com/docs/mcp | rg -n "Remote MCP|/mcp|magic-link"
curl -I -fsS https://trackbaseline.com/assets/baseline-court-serve.png
curl -sS -X POST https://trackbaseline.com/api/checkout \
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
curl -fsS https://trackbaseline.com/api/health
./bin/baseline doctor
./bin/baseline sync status
./bin/baseline sync push
curl -fsS https://trackbaseline.com/api/runs/latest
curl -fsS -X POST https://trackbaseline.com/api/admin/evaluate
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

## 2026-06-01 Bead 33 SEO and Lead Magnet Deploy

Bead 33 deploys the market-acquisition surface:

- `/blog` is now a content index, not a stub.
- Eight guide routes are live under `/guides/...`.
- Five lead-magnet routes are live under `/resources/...`.
- Lead-magnet requests post to `/api/events`, emit Klaviyo lead/master events when lifecycle email is configured, and are queryable through protected `/api/admin/leads`.
- `/dashboard`, `/admin`, and checkout return pages remain `noindex,follow` and are omitted from `/sitemap.xml`.

Preflight:

```sh
make verify
make plugin-validate
go run ./cmd/baseline --version
go run ./cmd/baseline version
cd web && npm run typecheck
```

Local route smoke:

```sh
cd web
npm run dev -- --port 8787
curl -fsS http://localhost:8787/blog | rg -n "/guides/coding-agent-health-check|/resources/agent-drift-scorecard|Guides and resources"
curl -fsS http://localhost:8787/resources/agent-drift-scorecard | rg -n "Request pilot prompt|lead_magnet_request|canonical|CreativeWork"
curl -fsS http://localhost:8787/sitemap.xml | rg -n "/guides/|/resources/"
curl -fsS -X POST http://localhost:8787/api/events -H 'content-type: application/json' --data '{"type":"lead_magnet_request","path":"/resources/agent-drift-scorecard","resource":"/resources/agent-drift-scorecard","email":"test@example.com","context":"codex mcp drift"}'
```

Deploy:

```sh
cd web
set -a; source /path/to/deploy.env; set +a
wrangler deploy
```

The first deploy attempt with unsourced Wrangler/OAuth auth failed with Cloudflare API `Authentication error [code: 10000]` because the token did not match `web/wrangler.jsonc` account `3a0bfe287d4dfb27f802ee5d7e4b21e1`. Retrying with `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` sourced from the operator env succeeded.

Live deploy:

- Worker version: `df4d479d-9fbd-4f8a-af50-b2f3a88253a8`
- URLs: `https://trackbaseline.com`, `https://www.trackbaseline.com`, `https://baseline-ai.ryan-borker.workers.dev`

Live smoke:

```sh
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://trackbaseline.com/blog | rg -n "/guides/coding-agent-health-check|/resources/agent-drift-scorecard|Guides and resources"
curl -fsS https://trackbaseline.com/resources/agent-drift-scorecard | rg -n "Request pilot prompt|lead_magnet_request|canonical|CreativeWork"
curl -fsS https://trackbaseline.com/sitemap.xml | rg -n "/guides/|/resources/"
curl -fsS -X POST https://trackbaseline.com/api/events -H 'content-type: application/json' --data '{"type":"lead_magnet_request","path":"/resources/agent-drift-scorecard","resource":"/resources/agent-drift-scorecard","email":"smoke+bead33@trackbaseline.com","context":"deploy smoke"}'
curl -sS -i https://trackbaseline.com/api/admin/leads | sed -n '1,8p'
```

Results:

- Health returned `db:true`, `stripe:true`, `token_required:true`, `lifecycle_email:true`, `pro_auth:true`, `pro_tokens:true`, and `stripe_webhook:true`.
- Blog index, resource page, canonical URL, CreativeWork JSON-LD, and lead-magnet request CTA served live.
- Sitemap included `/guides/...` and `/resources/...` and omitted `/dashboard`, `/admin`, and checkout return pages.
- Synthetic lead POST returned `{"ok":true}`.
- Unauthenticated `/api/admin/leads` returned `401`, confirming the lead queue is protected.

Rollback:

- Preferred rollback: `cd web && wrangler rollback b4f73e11-7540-4e97-8112-7698467b0484` to return to the immediately previous pre-Bead-33 Worker version listed by `wrangler deployments list --name baseline-ai`.
