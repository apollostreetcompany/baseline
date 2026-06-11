# Baseline Deployment Notes

## Current Production

- Worker: `baseline-ai`
- URL: https://trackbaseline.com
- Fallback Worker URL: https://baseline-ai.ryan-borker.workers.dev
- Current Version ID: `d313f92f-bb02-47b0-81ec-8d571dc61ed7`
- Current Deployment ID: `56391404-4f21-4b3f-b2fb-04a74aa29696`
- Current production source branch: `origin/main` at `cb54ea1a7c04194ab41f2744765a97fbd4b1ac67`
- Current production source worktree: `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-prod-cf-deploy`
- Notes: current production combines Bead 32 Codex plugin docs, Bead 33 SEO/lead-magnet acquisition routes, Bead 34 commercial-viability checkout/pilot/admin paths, the Bead 34 public website clarity pass, the Claude Fable 5 anti-slop copy polish, and Bead 35 `cf` deploy-tooling correction.

## Cloudflare CLI Policy

Current rule: use `cf` for Cloudflare operations. Do not run new production deploys through `wrangler` or `npm run deploy`.

Required operator checks before Cloudflare create/update/delete operations:

```sh
set -a
source /Users/kikimac/.hermes/.env
set +a
cf auth whoami
cf agent-context workers
cf workers deployments list --script-name baseline-ai
```

For Worker version/deployment changes, use the `cf workers versions` and `cf workers deployments` commands with an explicit reviewed request body and `--dry-run` first. Do not run body-less `cf workers scripts update` or body-less `cf workers versions create`; the dry-run only validates the API endpoint and is not a Worker bundle deploy.

`web/wrangler.jsonc` remains the project configuration file name/schema used by the Worker toolchain. Mentions of Wrangler in older sections below are historical receipts from the commands actually run at that time, not the current deploy procedure.

For changed Worker code, the Cloudflare REST upload path requires a `multipart/form-data` Worker upload with `metadata.main_module`, compatibility date, bindings, and code parts. Current `cf workers versions create --body ...` dry-runs serialize the body as JSON and cannot express the multipart upload by themselves. If no first-class `cf` multipart wrapper is available, use a reviewed Cloudflare API version upload with a bundled module, `keep_assets: true`, inherited secret bindings, and then use `cf workers deployments create` for the traffic switch/readback. Do not use `wrangler deploy` as the production mutation path.

## 2026-06-11 Checkout Router Implementation

Bead 37 adds a commercially usable checkout router while preserving the email-first, webhook-authoritative billing model.

Implementation result:

- Added `/checkout` as the dedicated rules page and linked it from nav, footer, pricing cards, and email-required checkout fallbacks.
- Added founder coupon input handling to pricing/checkout forms without printing the live code publicly.
- `POST /api/checkout` accepts only the configured founder code, applies `STRIPE_FOUNDER_PROMOTION_CODE_ID`, uses `payment_method_collection=if_required`, and returns canonical coupon metadata only after server validation.
- Stripe webhook completion tags founder-code entitlements as `stripe_founder`, includes coupon metadata in lifecycle/audit properties, and reprocesses failed event rows on Stripe retry.
- Klaviyo receives redacted buyer lifecycle events and master webhook notifications with event type, object id, plan, coupon presence, account id when known, and email-presence booleans.
- Datafa.st receives checkout start, coupon-applied, redirect, success-return, and cancel-return goals; the Worker carries the `datafast_visitor_id` cookie into Stripe metadata when present.
- Local `cf dev` requires `@cloudflare/wrangler-bundler`; the dependency is declared so local Worker smokes work in fresh worktrees.

Pre-deploy validation:

```sh
subreview --reviewers claude --output /tmp/baseline-subreview-checkout-implementation --intent "Commercial checkout implementation review..."
npm --prefix web run typecheck
make verify
git diff --check
npm --prefix web audit --audit-level=high
curl -fsS http://127.0.0.1:60702/checkout
curl -sS -X POST http://127.0.0.1:60702/api/checkout -H 'content-type: application/json' --data '{"email":"codex-smoke+bead37@example.com","plan":"pro","couponCode":"WrongCode"}'
curl -sS -X POST http://127.0.0.1:60702/api/checkout -H 'content-type: application/json' --data '{"email":"codex-smoke+bead37@example.com","plan":"pro","couponCode":"FounderBaseline"}'
```

Results so far:

- Claude Fable 5 `subreview` completed with 1 reviewer and 0 failed reviewers at `/tmp/baseline-subreview-checkout-implementation/claude.md`; acted-on findings are recorded in the Bead 37 plan.
- Typecheck, `make verify`, `git diff --check`, high-severity audit, local route smokes, and Playwright desktop/mobile layout checks passed.
- Stripe promotion code readback verified the founder code is active, capped, 100% off, and `duration=forever`.
- Worker secrets readback confirmed `STRIPE_FOUNDER_PROMOTION_CODE_ID`, `BASELINE_MASTER_EMAIL`, Stripe, and Klaviyo secret names are present without printing values.

Production deployment and live checkout-session smoke remain pending.

## 2026-06-10 `cf` CLI Correction

Operator correction: future Cloudflare deploy and readback work uses `cf`, not `wrangler`. A follow-up readback with `cf workers deployments list --script-name baseline-ai` confirmed the active deployment is still Worker version `d313f92f-bb02-47b0-81ec-8d571dc61ed7` at 100% traffic.

Current active deployment readback:

- Deployment ID: `56391404-4f21-4b3f-b2fb-04a74aa29696`
- Worker version: `d313f92f-bb02-47b0-81ec-8d571dc61ed7`
- Traffic: 100%
- Source: `api`
- Created: `2026-06-10T08:29:54.585318Z`

## 2026-06-10 `cf` Production Redeploy After PR #10

After PR #10 (`https://github.com/apollostreetcompany/baseline/pull/10`) merged the `cf` deploy-tooling correction to `main`, the exact `origin/main` commit `cb54ea1a7c04194ab41f2744765a97fbd4b1ac67` was redeployed through `cf`. The merged PR did not change `web/src`, `web/public`, or `web/wrangler.jsonc`, so the safe production action was to create a fresh deployment that routes 100% traffic to the already-live Worker version `d313f92f-bb02-47b0-81ec-8d571dc61ed7`.

Deployment result:

- Deployment ID: `56391404-4f21-4b3f-b2fb-04a74aa29696`
- Source: `api`
- Worker version: `d313f92f-bb02-47b0-81ec-8d571dc61ed7`
- Traffic: 100%
- Created: `2026-06-10T08:29:54.585318Z`
- Source branch: `origin/main`.
- Source commit: `cb54ea1a7c04194ab41f2744765a97fbd4b1ac67`.

Validation:

```sh
gh pr view 10 --json number,state,mergedAt,mergeCommit,url,headRefName,baseRefName
git diff --name-only cda91d1b3d3a8244cd8a11424ea39f963d8dc14b..HEAD -- web/src web/public web/wrangler.jsonc web/package.json Makefile docs/DEPLOYMENT.md CONTINUITY.md HANDOFF.md MISTAKES.md handoff/beads.jsonl
make verify-all
git diff --check
npm --prefix web audit --audit-level=high
npm --prefix web run deploy
cf auth whoami
cf workers deployments list --script-name baseline-ai
cf workers deployments create --script-name baseline-ai --dry-run --body '{"strategy":"percentage","versions":[{"version_id":"d313f92f-bb02-47b0-81ec-8d571dc61ed7","percentage":100}]}'
cf workers deployments create --script-name baseline-ai --body '{"strategy":"percentage","versions":[{"version_id":"d313f92f-bb02-47b0-81ec-8d571dc61ed7","percentage":100}]}'
cf workers deployments get 56391404-4f21-4b3f-b2fb-04a74aa29696 --script-name baseline-ai
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://www.trackbaseline.com/api/health
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health
curl -fsS https://trackbaseline.com/ | rg -o "Know when your coding agent quietly changed|Copy install command|Sample data"
curl -fsS https://trackbaseline.com/docs/mcp | rg -o "Remote MCP sequence|baseline serve mcp|trackbaseline.com/mcp"
curl -sS -i -X POST https://trackbaseline.com/mcp -H 'content-type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
curl -sS -i https://trackbaseline.com/api/admin/leads
```

Results:

- PR #10 was merged into `main` at `cb54ea1a7c04194ab41f2744765a97fbd4b1ac67`.
- Diff since the previously deployed app commit touched deployment docs/process files only; `web/src`, `web/public`, and `web/wrangler.jsonc` were unchanged.
- `make verify-all`, `git diff --check`, and the high-severity audit gate passed. The audit still reports the known moderate Wrangler/Miniflare `ws` chain.
- `npm --prefix web run deploy` intentionally exited with the guard instructing operators to use the documented `cf` workflow.
- `cf workers deployments list` and `cf workers deployments get` confirmed deployment `56391404-4f21-4b3f-b2fb-04a74aa29696`, source `api`, Worker version `d313f92f-bb02-47b0-81ec-8d571dc61ed7`, 100% traffic.
- Live health returned `db:true`, `stripe:true`, `lifecycle_email:true`, `pro_auth:true`, `pro_tokens:true`, and `stripe_webhook:true` on apex, `www`, and workers.dev fallback.
- Live homepage and MCP docs markers passed.
- Live unauthenticated `POST /mcp` returned HTTP `401` with `WWW-Authenticate` and `authentication_required`.
- Live unauthenticated `/api/admin/leads` returned HTTP `401` with `invalid admin token`.

Rollback:

- Preferred code rollback target remains previous Fable copy polish Worker version `4966bc91-0e4a-4657-8589-96a14e78d2c1`.
- Deployment-level rollback uses `cf workers deployments create --script-name baseline-ai --dry-run --body 'REVIEWED_ROLLBACK_BODY'`, then repeats without `--dry-run` after verifying the body.

## 2026-06-10 Current Main Production Redeploy

After PR #7 and PR #8 were already merged to `main`, the exact `origin/main` commit `cda91d1b3d3a8244cd8a11424ea39f963d8dc14b` was redeployed from a clean release worktree so production reflects the merged website clarity and Claude Fable 5 copy polish.

Deployment result:

- Worker version `d313f92f-bb02-47b0-81ec-8d571dc61ed7` deployed to `https://trackbaseline.com`, `https://www.trackbaseline.com`, and the workers.dev fallback.
- Source branch: `origin/main`.
- Source commit: `cda91d1b3d3a8244cd8a11424ea39f963d8dc14b`.
- Historical note: this Worker version was uploaded before the `cf` CLI correction. Future Cloudflare changes use the `cf` policy above.

Validation:

```sh
make verify-all
git diff --check
npm --prefix web audit --audit-level=high
cf auth whoami
cf workers deployments list --script-name baseline-ai
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://trackbaseline.com/ | rg -n "Know when your coding agent quietly changed|copy install command|Sample data"
curl -fsS https://trackbaseline.com/blog | rg -n "Field notes for agent operators|How to accept a Good Baseline|MCP drift"
curl -fsS https://trackbaseline.com/docs/mcp | rg -n "Install Baseline, run a check|baseline run --mode fast|Universal MCP smoke"
curl -fsS https://trackbaseline.com/robots.txt | rg -n "Sitemap: https://trackbaseline.com/sitemap.xml|Disallow: /dashboard|Disallow: /checkout|Disallow: /api/"
curl -fsS https://trackbaseline.com/sitemap.xml | rg -n "https://trackbaseline.com/</loc>|https://trackbaseline.com/docs/mcp</loc>|https://trackbaseline.com/blog</loc>"
curl -fsS https://www.trackbaseline.com/api/health | rg -n '"ok":true'
curl -fsS https://baseline-ai.ryan-borker.workers.dev/api/health | rg -n '"ok":true'
curl -i -sS https://trackbaseline.com/mcp -H 'content-type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Results:

- `make verify-all`, `git diff --check`, `cf` deployment readback, and the high-severity audit gate passed. The audit still reports the known moderate Wrangler/Miniflare `ws` chain.
- Live health returned `db:true`, `stripe:true`, `lifecycle_email:true`, `pro_auth:true`, `pro_tokens:true`, and `stripe_webhook:true`.
- Live homepage, blog, MCP docs, robots, sitemap, `www`, and workers.dev fallback smokes passed.
- Live unauthenticated `/mcp` returned HTTP `401` with `WWW-Authenticate` and `authentication_required`.

Rollback:

- Preferred rollback target: previous Fable copy polish Worker version `4966bc91-0e4a-4657-8589-96a14e78d2c1`.

## 2026-06-10 Claude Fable 5 Copy Polish Deploy

`subreview` was rerun on the latest squashed `main` integration with Claude Fable 5 only. It reviewed `HEAD^...HEAD` against the public website/copy scope and wrote the completed manifest to `/tmp/baseline-subreview-fable-copy-20260610T0315Z/manifest.json` with `model: "claude-fable-5"`, `Completed reviewers: 1`, and `Failed reviewers: 0`.

Applied copy fixes:

- Removed the unsupported public "Claude Code" hero claim while keeping the approved local runner language.
- Replaced public funnel labels like "SEO/AEO", "lead resource", "lead magnet", and "pilot prompt" with visitor-facing guide/resource/pilot-invite language.
- Rewrote `/checkout/success` so the buyer sees a concrete magic-link/session-token/workspace-token/sync sequence, not pseudo-HTTP.
- Fixed dashboard install-to-value order to `setup -> run -> report -> accept -> compare` and made the dashboard failure path reveal the example-data banner.
- Made checkout success/cancel copy plan-neutral for Pro and Team buyers.
- Clarified pricing bullets, the 7-day pilot expectation, privacy boundaries, example data labels, JSON-LD offers, and staging-safe install commands.

Deployment result:

- Worker version `4966bc91-0e4a-4657-8589-96a14e78d2c1` deployed to `https://trackbaseline.com`, `https://www.trackbaseline.com`, and the workers.dev fallback.
- Source branch: `codex/feat/bead-34-fable-copy-polish`.

Validation:

```sh
SUBREVIEW_REVIEWERS=claude SUBREVIEW_CLAUDE_MODEL=claude-fable-5 subreview --reviewers claude --base HEAD^ HEAD --output /tmp/baseline-subreview-fable-copy-20260610T0315Z --intent "Copy and messaging review only..."
npm --prefix web run typecheck
make verify
git diff --check
npm --prefix web audit --audit-level=high
curl -fsS http://localhost:8789/ | rg -n "Baseline probes OpenClaw, Codex, Hermes, or any approved local runner|Example fast run|7-day setup pilot|Request pilot invite"
curl -fsS http://localhost:8789/checkout/success | rg -n "Checkout received|session_token|Workspace token setup|baseline sync on"
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://trackbaseline.com/ | rg -n "Baseline probes OpenClaw, Codex, Hermes, or any approved local runner|Example fast run|7-day setup pilot|Request pilot invite"
curl -fsS https://trackbaseline.com/checkout/success | rg -n "Checkout received|session_token|Workspace token setup|baseline sync on"
for route in / /blog /resources/coding-agent-health-checklist /checkout/success /dashboard /privacy; do curl -fsS "https://trackbaseline.com$route" | rg -n "Claude Code|SEO/AEO|Lead magnet|pilot prompt|Lifecycle email|support handoffs|Pro checkout received|No Pro subscription|sample \\." && exit 1 || true; done
```

Results:

- `npm --prefix web run typecheck`, `make verify`, `git diff --check`, and the high-severity audit gate passed. The audit still reports the known moderate Wrangler/Miniflare `ws` chain.
- Local and live route smokes passed for homepage, blog, resource page, checkout success, dashboard, and privacy copy.
- Live `/api/health` returned `db:true`, `stripe:true`, `lifecycle_email:true`, `pro_auth:true`, `pro_tokens:true`, and `stripe_webhook:true`.
- Live negative copy sweep found no `Claude Code`, `SEO/AEO`, `Lead magnet`, `pilot prompt`, `Lifecycle email`, `support handoffs`, plan-wrong checkout wording, or duplicate `sample .` labels.

Rollback:

- Preferred rollback target: previous website integration Worker version `214cec6e-a79d-4360-8aa3-a19e2eb42939`.

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
git diff --check
cf auth whoami
cf agent-context workers
cf workers versions create --script-name baseline-ai --bindings-inherit strict --dry-run --body 'REVIEWED_VERSION_UPLOAD_BODY'
cf workers deployments create --script-name baseline-ai --dry-run --body 'REVIEWED_DEPLOYMENT_BODY'
curl -fsS https://trackbaseline.com/api/health
curl -fsS -X POST https://trackbaseline.com/mcp -H 'content-type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Expected unauthenticated MCP smoke result:

- HTTP `401`
- `WWW-Authenticate` header present
- JSON body includes `authentication_required`

Rollback:

- Cloudflare Workers rollback through `cf workers deployments create --script-name baseline-ai --dry-run --body 'REVIEWED_ROLLBACK_BODY'`, then repeat without `--dry-run` after verifying the body.
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

## 2026-06-10 Website Clarity + Commercial Viability Integration Deploy

The standalone website-clarity branch was based before Bead 33/34 commercial work, so it was not deployed directly. The integration branch starts from deployed commit `8cdbae6` and ports the website clarity commit `9b0e90e944520346c494585169f8131d32b3e111` onto the current lead-magnet, pilot, checkout, admin, and account-private run surface.

Deployment result:

- Worker version `214cec6e-a79d-4360-8aa3-a19e2eb42939` deployed to `https://trackbaseline.com`, `https://www.trackbaseline.com`, and the workers.dev fallback.
- Source branch: `codex/feat/bead-34-website-production-integration`.
- Source commit before deploy receipt: `bab5eed`.

Validation:

```sh
make verify
git diff --check
npm --prefix web audit --audit-level=high
curl -fsS https://trackbaseline.com/api/health
curl -fsS https://trackbaseline.com/ | rg -n "Know when your coding agent quietly changed|First local baseline|pilot-request"
curl -fsS https://trackbaseline.com/docs/mcp | rg -n "Setup -&gt; run -&gt; accept -&gt; compare|First Good Baseline"
curl -fsS https://trackbaseline.com/blog | rg -n "Field notes for agent operators|Lead resources"
curl -fsS https://trackbaseline.com/dashboard | rg -n "Example data|Install-to-value path"
curl -fsS https://trackbaseline.com/checkout/success | rg -n "send magic link|baseline sync on"
curl -fsS https://trackbaseline.com/admin | rg -n "Invite pilot|view-leads"
curl -fsS https://trackbaseline.com/robots.txt | rg -n "Disallow: /admin|Disallow: /api/"
curl -fsS https://trackbaseline.com/sitemap.xml | rg -n "/guides/coding-agent-health-check|/resources/agent-drift-scorecard"
curl -i -sS https://trackbaseline.com/mcp \
  -H 'content-type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Results:

- `make verify`, `git diff --check`, and the high-severity audit gate passed. The audit still reports the known moderate Wrangler/Miniflare `ws` chain.
- Live `/api/health` returned `db:true`, `stripe:true`, `lifecycle_email:true`, `pro_auth:true`, `pro_tokens:true`, and `stripe_webhook:true`.
- Live homepage plainly explains Baseline as a local CLI/MCP checker and shows the setup -> run -> accept -> compare loop.
- Live docs include copyable command blocks; live blog contains field-note sections plus the guide/resource index.
- Live dashboard labels example data, and live admin/checkout success preserve the pilot invite and paid onboarding paths.
- Live `/mcp` unauthenticated smoke returned `401` with `authentication_required`.

Rollback:

- Preferred rollback target: previous commercial-viability Worker version `7940fc3a-f89e-4972-9352-e77424b541a6`.

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

- Preferred code rollback: use `cf workers deployments create --script-name baseline-ai --dry-run --body 'REVIEWED_ROLLBACK_BODY'` to route 100% traffic to Worker version `4f1b94a0-543a-4cb2-8207-62825fb29594`, then repeat without `--dry-run` after review.
- If the domain attachment itself is wrong, remove the `routes` entries from `web/wrangler.jsonc`, redeploy, and verify the fallback workers.dev route.

## 2026-05-19 Brand Landing Assets

- Worker static assets are now configured through `web/wrangler.jsonc` with `assets.directory = "./public"`.
- Current image assets live under `web/public/assets/` and are uploaded with the Worker deployment.
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

Historical live deployment verification on 2026-05-19:

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

Secrets are configured through the Cloudflare API/CLI and must never be printed in logs. Use `cf` going forward; this 2026-05-19 setup was completed before the `cf` correction.

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

Historical deploy command used at the time:

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

- Preferred rollback: use `cf workers deployments create --script-name baseline-ai --dry-run --body 'REVIEWED_ROLLBACK_BODY'` to return 100% traffic to Worker version `b4f73e11-7540-4e97-8112-7698467b0484`, then repeat without `--dry-run` after review.
