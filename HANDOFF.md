# HANDOFF.md - Baseline.ai

## Current Thread
- Working branch: `codex/feat/bead-34-fable-copy-polish`.
- Current request history:
  - Bead 23B: Pro account architecture doc committed as `96d2e28`.
  - Bead 23A: landing/design/docs/blog/pro checkout stub implementation committed as `257c17f`.
  - Bead 24: refreshed Worker deployed to https://baseline-ai.ryan-borker.workers.dev at version `5cc879a3-983d-4e59-a620-e8abd8d70a99`.
  - Bead 25: cloud accounts, Stripe/Klaviyo entitlement lifecycle, account-scoped tokens, remote MCP, SwiftUI macOS hotspot client, skill audit, and Worker deployment version `dfc2198f-9151-4a64-8511-4e25d3c2d529`.
  - Bead 27: `landing-a` homepage redesign plus local BrandOS runtime repair, deployed to Cloudflare Worker version `4f1b94a0-543a-4cb2-8207-62825fb29594`.
  - Integration: PR #1 (`https://github.com/apollostreetcompany/baseline/pull/1`) combines Bead 25 cloud/Mac app functionality with Bead 27 landing before merge to `main`.
  - Bead 28: Cloudflare custom domain deployment makes `https://trackbaseline.com` the canonical public URL, with `www.trackbaseline.com` and workers.dev fallback triggers.
  - Bead 29: public distribution and Pro activation. Implementation commit `63e8d1bb59663fee502c18aad36141b5bd5fa1dd`; Worker deploy `e38523fc-d11a-41d9-b05e-6dcef5f4b5f0`; GitHub Release `v0.1.0` published.
  - Bead 30: DataFast launch funnel analytics. Implementation commit `6474606e7ed151be888fd924abd6c8f5c3cbe9f2`; Worker deploy `fb899682-a797-4201-9842-4dfb72d5cecd`; DataFast funnels created with CLI.
  - Bead 31: Robot photo favicon/app icons. Worker deploy `b4f73e11-7540-4e97-8112-7698467b0484`; live `/favicon.ico` now returns `200`.
  - Bead 32: Codex plugin readiness/build. `plugins/baseline/` is the v1 Codex plugin, `openclaw-plugin/` remains the legacy/OpenClaw compatibility bundle, and `baseline-codex-plugin.tgz` is now part of release packaging.
  - Bead 33: market-effectiveness pass for first organic customer path. Added eight guide routes, five lead resources, lead request capture/admin queue/Klaviyo events, dashboard/admin clarity, CLI `--version`, docs/package first-run guidance, and Worker deploy `df4d479d-9fbd-4f8a-af50-b2f3a88253a8`.
  - Bead 34: commercial viability pass from fresh `subreview`. Added pilot request capture, admin pilot invite/grant UI, email-first Pro/Team checkout attribution, operational checkout success magic-link/token guidance, safe checkout-session status, public dashboard account-private filtering/demo labeling, account-scoped run upsert guard, and paid-pilot deployment docs.
  - Bead 34 website clarity integration: the standalone `codex/feat/bead-34-website-clarity` branch was based before Bead 33/34 commercial work, so it was not deployed directly. Its public copy, sample labels, copyable commands, field-note blog sections, metadata, robots, and sitemap improvements are integrated onto the commercial-viability branch and deployed as Worker version `214cec6e-a79d-4360-8aa3-a19e2eb42939`.
  - Bead 34 Fable copy polish: `subreview` was rerun on latest squashed `main` with Claude Fable 5 only and confirmed via manifest. Applied traced user-facing copy fixes: concrete checkout-success onboarding, visitor-facing resource labels, corrected dashboard command order, plan-neutral checkout copy, clearer pilot/pricing/privacy copy, and Worker deploy `4966bc91-0e4a-4657-8589-96a14e78d2c1`.

## Key Context
- Existing app is a Cloudflare Worker in `web/src/index.ts`.
- Canonical production URL is now `https://trackbaseline.com`.
- Current production branch preserves Bead 33 content/lead routes, Bead 34 commercial checkout/pilot/admin routes, the Bead 34 website clarity copy pass, and the Claude Fable 5 anti-slop copy polish in a single production-ready Worker.
- Public install command is now `curl -fsSL https://trackbaseline.com/install.sh | sh`, backed by GitHub Release assets and checksum verification.
- Production Pro secrets are active: Stripe Checkout, Stripe webhook verification, Klaviyo lifecycle email, magic-link auth, and HMAC workspace tokens. Do not print secret values.
- DataFast website id is `6a0c48aa9a21aee7bf04cf6e`; tracking id is `dfid_PYprhfTkwwQKhkzRUhVtO`; CLI-created funnels are `baseline-install-funnel` and `baseline-pro-funnel`.
- Favicon source is `web/public/assets/baseline-court-robot.png`; generated icon assets live at the root of `web/public/`.
- Codex plugin v1 source is `plugins/baseline/`, with repo-local marketplace metadata at `.agents/plugins/marketplace.json`. It assumes the `baseline` CLI is installed and available on `PATH`.
- Bead 33 public acquisition routes live under `/blog`, `/guides/...`, and `/resources/...`; `/dashboard`, `/admin`, and checkout return pages remain `noindex,follow` and are omitted from `/sitemap.xml`.
- Lead-magnet requests post to `/api/events`, emit Klaviyo customer/master events when lifecycle email is configured, and are listed through protected `/api/admin/leads`.
- Bead 34 changes checkout policy: paid checkout now requires an email-first Stripe Checkout Session so account metadata can provision entitlement. Stripe payment links are intentionally disabled for paid onboarding because they cannot guarantee account attribution.
- `/checkout/success` now requests the buyer's magic link and shows the workspace-token / `baseline sync on` path; `/api/checkout/session` fetches the exact Stripe session before returning any entitlement hint.
- `/admin` now has an **Invite pilot** panel that calls `POST /api/admin/invites` and can grant pilot entitlement. Public pricing now has `/#pilot-request`, which records `pilot_request` events in the admin lead queue.
- Public `/api/runs/latest` and `/api/runs/timeline` now exclude account-private Pro runs and fall back to labeled example/demo data.
- Existing admin/evaluator endpoints use `BASELINE_ADMIN_TOKEN`, Neon, and optional OpenAI evaluator.
- Bibe Code reference patterns inspected:
  - Benefit-led landing copy and bold editorial composition.
  - Email capture before public checkout.
  - Stripe checkout metadata tied to a created user id.
  - Idempotent Stripe webhook events.
  - Klaviyo event emission for subscription lifecycle and owner notifications.
- MagicPath themes inspected:
  - `Brutalism` for hard-edged editorial structure.
  - `Ramp` for operational SaaS restraint.
- Bead 25 locked Cloudflare Worker + Neon as canonical. Supabase and local-only MCP were walked back.
- REST remains canonical and remote MCP is an authenticated adapter over account, workspace, history, hotspot, comparison, subscription, and owner-support operations.
- Mac app is cloud MCP first; local SQLite can be a later secondary connector.
- LLM insight order is local agent/provider bridge first, then OpenRouter API key fallback stored in Keychain.
- Bead 27 was intentionally based in a new worktree because another agent has the original Baseline worktree dirty.
- The Bead 27 Worker source preserves Bead 25 cloud account/remote-MCP files from the original worktree before applying the `landing-a` homepage redesign.
- BrandOS local repair lives in `/Users/kikimac/.hermes/repos/apollostreetcompany/skills-library/skills/brand-os-studio`: scripts now avoid PyYAML, use `python3`, and fall back to a bundled `.prose` validator when no `prose` CLI is installed.

## Active Beads
- Bead 34 Fable copy polish is deployed from `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-website-production-integration` on `codex/feat/bead-34-fable-copy-polish`.
- Current Worker version `4966bc91-0e4a-4657-8589-96a14e78d2c1` is live on `https://trackbaseline.com`; rollback target is previous website integration version `214cec6e-a79d-4360-8aa3-a19e2eb42939`.
- Bead 32 Codex plugin v1 remains implemented and locally validated; productionizing next means CLI preflight/auto-install, clean Codex environment smoke tests, plugin assets, and CI schema validation.

## Commands To Re-run
- `cd /Users/kikimac/.hermes/repos/apollostreetcompany/baseline`
- `make verify-all`
- `cd web && npm run typecheck`
- `cd web && npm run dev`
- `go test ./...`
- `cd macos/BaselineHotspots && swift build`
- `cd web && npm audit --audit-level=high`
- `bash scripts/build-release.sh`
- `make plugin-validate`
- `go run ./cmd/baseline --version`
- `go run ./cmd/baseline version`
- `bash scripts/validate-codex-plugin.sh openclaw-plugin || true`
- `npm --prefix package pack --dry-run`
- `DATAFAST_TOKEN=... make analytics-report`
- `curl -I https://trackbaseline.com/favicon.ico`
- `cd /Users/kikimac/.hermes/repos/apollostreetcompany/skills-library && make verify-library && make verify-codex`

## Local QA Evidence
- Local Worker was run on `http://localhost:8787`.
- Browser visual checks covered `/`, mobile `390x844`, desktop `1280x720`, `/#pro-monitoring`, `/blog`, and `/docs/mcp`.
- Image assets loaded with natural size `1024x1024`.
- Browser layout check found no horizontal overflow.
- `POST /api/checkout` returns the expected unconfigured Stripe JSON error when no Stripe secret/payment link is present.
- Live deployment smoke checked `https://baseline-ai.ryan-borker.workers.dev/`, `/blog`, `/assets/baseline-court-robot.png`, `/api/health`, and `POST /api/checkout`.
- Live health returned `{"ok":true,"db":true,"stripe":false,"token_required":true,"lifecycle_email":false}`.
- Live checkout fallback returned the expected unconfigured Stripe JSON error while secrets/payment links remain unset.
- Bead 25 validation: `make verify` passed before code edits; `make verify-all` passed after implementation and after skill-audit fixes; `git diff --check` passed; `cd web && npm audit --audit-level=high` found no high-severity advisories; local `/mcp` smoke returned 401 challenge; live deploy version `dfc2198f-9151-4a64-8511-4e25d3c2d529` passed health, MCP challenge, protected-resource metadata, docs, and checkout-fails-closed smoke.
- Bead 27 validation: `cd web && npm run typecheck`, `go test ./...`, and `cd web && npm audit --audit-level=high` passed. Local Worker on `http://localhost:8787` served the `landing-a` markers, `/docs/mcp`, `/api/health`, static hero asset, and checkout-fails-closed JSON.
- Bead 27 screenshots saved to `/tmp/baseline-landing-a-combined-desktop.png`, `/tmp/baseline-landing-a-combined-mobile.png`, `/tmp/baseline-live-landing-a-desktop.png`, and `/tmp/baseline-live-landing-a-mobile.png`.
- Bead 27 live smoke: `https://baseline-ai.ryan-borker.workers.dev/api/health` returned `db:true`, `stripe:false`, `token_required:true`, `pro_auth:false`, `pro_tokens:false`, `stripe_webhook:false`; live `/` contained `Your agent forgot`, `Three agents`, `Fourteen probes`, and `In the line`; `/docs/mcp` still served remote MCP docs; `/assets/baseline-court-serve.png` returned `200`; checkout still fails closed until Stripe secrets are set.
- BrandOS validation: `scripts/audit_skill_pack.py`, `scripts/validate_prose.py workflows`, `bash scripts/compile_prose.sh`, `scripts/check_stage_gates.py examples/shogun-sauce/workspace`, `python3 -m py_compile`, `bash -n`, `make verify-library`, and `make verify-codex` all passed; `make verify-codex` retains optional missing-agent warnings only.
- Integration validation for PR #1: `make verify-all`, `git diff --check`, `git diff --cached --check`, `node` JSONL parse, `cd web && npm audit --audit-level=high`, and local Worker smokes for `/api/health`, `/`, `/docs/mcp`, `/mcp`, `/.well-known/oauth-protected-resource`, `/assets/baseline-court-serve.png`, and `POST /api/checkout` passed.
- CI hardening: GitHub Actions initially caught a Swift 6 strict-concurrency failure in the macOS app (`[String: Any]` MCP payload crossing actor isolation). `BaselineMCPClient` is now `@MainActor`, and `make mac-build` runs `swift build -Xswiftc -strict-concurrency=complete`.
- Bead 28 validation: `make verify-all`, `cd web && npm audit --audit-level=high`, `git diff --check`, Wrangler dry run, Wrangler deploy, DNS checks, apex/`www`/workers.dev health checks, landing markers, MCP docs, MCP unauth challenge, OAuth protected-resource metadata, hero asset, and checkout fail-closed smoke all passed.
- Bead 29 validation: `make verify-all`, `bash scripts/build-release.sh`, `npm --prefix package pack --dry-run`, `npm --prefix web audit --audit-level=high`, `git diff --check`, local Worker `/docs/mcp` + `/install.sh` smoke, Wrangler deploy, live health, live docs/install route smoke, Stripe Checkout URL smoke, unsigned webhook fail-closed smoke, GitHub Release workflow `26091658646`, release asset/checksum inspection, public `install.sh` temp-home install smoke, and npm wrapper temp-home auto-download smoke all passed. `npm whoami` failed with `ENEEDAUTH`, so the npm package is prepared but not published.
- Bead 30 validation: DataFast CLI docs checked; `npx @datafast/cli websites list` confirmed `trackbaseline.com`; `funnels create` created install and Pro funnels; `DATAFAST_PERIOD=last24h bash scripts/datafast-funnel-report.sh` returned overview/goals/pages/referrers/funnels; `npm run typecheck`, `bash -n scripts/datafast-funnel-report.sh`, `make verify-all`, `npm --prefix web audit --audit-level=high`, `git diff --check`, local Worker script/goal smoke, Wrangler deploy, and live script/goal/health smokes passed.
- Bead 31 validation: `npm run typecheck`, manifest JSON parse, `sips` dimension checks, local Worker favicon metadata and all icon asset smokes, `make verify-all`, `npm --prefix web audit --audit-level=high`, `git diff --check`, Wrangler deploy, live `/favicon.ico`, PNG icon, Apple touch icon, manifest, homepage metadata, and health smokes passed.
- Bead 32 validation: `make plugin-validate` passed for `plugins/baseline`; the same validator intentionally fails `openclaw-plugin` on legacy `mcp`/`publisher` fields, missing `author`, missing `interface`, and skill frontmatter. `make test`, `make package-test`, `make web-typecheck`, JSON syntax checks, referenced-path checks, shell syntax checks, `git diff --check`, and a temp `DIST_DIR` release build all passed. The temp release build produced `baseline-codex-plugin.tgz` containing `.codex-plugin/plugin.json`, `.mcp.json`, `README.md`, `assets/`, and `skills/baseline-health/SKILL.md`.
- Bead 33 validation: `make verify`, `make plugin-validate`, `go run ./cmd/baseline --version`, `go run ./cmd/baseline version`, local `/blog`, `/resources/agent-drift-scorecard`, `/sitemap.xml`, `/api/events`, `/admin`, and `/api/admin/leads` smokes all passed. Playwright screenshots are stored at `handoff/bead-33-blog.png`, `handoff/bead-33-dashboard.png`, `handoff/bead-33-lead-resource-final.png`, and `handoff/bead-33-admin-final.png`.
- Bead 33 review: fresh skill-specific RepoPrompt subagents researched X/reddit/SEO/AEO/lead magnets/MCP/native MCP/app acquisition/UI/operationalization. `subreview --uncommitted` completed partially: Claude completed and found real lead-loop blockers; Codex wrapper failed on CLI args and Gemini quota was exhausted. Acted on Claude findings by wiring lead requests to Klaviyo events, adding protected `/api/admin/leads`, adding admin "Recent leads", normalizing emails/honeypot, and softening overpromising copy. Proconsult was attempted twice and failed at browser attachment upload timeout.
- Bead 33 deploy: `npm run deploy` and unsourced global `wrangler deploy` failed with Cloudflare `Authentication error [code: 10000]` because the OAuth token did not match `web/wrangler.jsonc` account. Sourcing the operator Cloudflare env from `/Users/kikimac/.hermes/.env` without printing values and running `wrangler deploy` succeeded. Live Worker version `df4d479d-9fbd-4f8a-af50-b2f3a88253a8` passed health, blog/resource/sitemap, synthetic lead POST, and protected admin-leads auth smokes.
- Bead 34 subreview: `subreview --base origin/main HEAD --intent "Commercial viability only..."` completed partially. Claude completed and stored the full review at `/Users/kikimac/.claude/plans/reviewer-prompt-template-keen-conway.md`; Codex failed because the wrapper passed `--base` with positional prompt in an incompatible way; Gemini quota was exhausted. Acted on the short-path findings: pilot invite, checkout success onboarding, checkout attribution/session status, lead/admin follow-up, demo labeling, and account-private run safety.
- Bead 34 validation: `make verify`, `git diff --check`, `npm --prefix web audit --audit-level=high`, local Worker smokes on `http://localhost:8788`, Playwright screenshots, Wrangler deploy, and live smokes all passed except protected admin lead readback. Screenshots: `handoff/bead-34-pricing-pilot.png`, `handoff/bead-34-checkout-success.png`, `handoff/bead-34-admin-pilot.png`, `handoff/bead-34-dashboard-demo.png`.
- Bead 34 live deploy: `wrangler deploy` with sourced Cloudflare env succeeded at Worker version `7940fc3a-f89e-4972-9352-e77424b541a6`. Live health returned all production surfaces configured (`db`, `stripe`, `lifecycle_email`, `pro_auth`, `pro_tokens`, `stripe_webhook`). Live homepage, checkout success, checkout email guard, invalid lead guard, and synthetic `codex-smoke+bead34@example.com` pilot request passed. `/api/admin/leads` readback is `UNCONFIRMED` from this shell because the local env did not include `BASELINE_ADMIN_TOKEN`; unauthenticated `401` was verified.
- Bead 34 website clarity source: standalone commit `9b0e90e944520346c494585169f8131d32b3e111` on `codex/feat/bead-34-website-clarity` passed `make verify`, `git diff --check`, local Worker route smokes for `/`, `/docs/mcp`, `/blog`, `/dashboard`, `/robots.txt`, `/sitemap.xml`, Playwright desktop/mobile overflow checks, and copy-button feedback checks. Screenshots were saved under `/tmp/baseline-website-clarity-*.png`.
- Bead 34 integration deploy: `make verify`, `git diff --check`, `npm --prefix web audit --audit-level=high`, local route smokes for `/`, `/docs/mcp`, `/blog`, `/dashboard`, `/robots.txt`, `/sitemap.xml`, `/checkout/success`, and `/admin`, Wrangler deploy, and live smokes all passed. Worker version `214cec6e-a79d-4360-8aa3-a19e2eb42939` serves the integrated website clarity plus commercial-viability surface. The high-severity audit gate passed with the known moderate Wrangler/Miniflare `ws` chain only.
- Bead 34 Fable copy polish: `subreview --reviewers claude --base HEAD^ HEAD` completed with Claude Fable 5 only; manifest at `/tmp/baseline-subreview-fable-copy-20260610T0315Z/manifest.json` records `model: claude-fable-5`, 1 completed reviewer, and 0 failed reviewers. Applied the actionable copy findings, then passed `npm --prefix web run typecheck`, `make verify`, `git diff --check`, `npm --prefix web audit --audit-level=high`, local route smokes on `http://localhost:8789`, production deploy, live `/api/health`, live homepage/blog/resource/checkout/dashboard/privacy smokes, and live negative copy sweep. Worker version `4966bc91-0e4a-4657-8589-96a14e78d2c1` is live.

## Open Risks
- Live Stripe, Klaviyo, Neon, and deployment verification require production/staging secrets and must never print secret values.
- DataFast token handling must use no-echo or secret storage; a plain TTY `read` echoed the token once during setup and is recorded in `MISTAKES.md`.
- Real paid pilot launch still needs an end-to-end checkout with an intended pilot email, webhook entitlement confirmation, magic-link login, token issuance, redacted sync, remote MCP account-status smoke, and Klaviyo flow verification for `Baseline Lead Magnet Requested`, `Baseline Pilot Requested`, `Baseline Magic Link`, and `Baseline Subscription Started`.
- The remote MCP implementation is an HTTP JSON-RPC endpoint shaped for MCP clients; before public announcement it should be smoke-tested against the exact target clients that will register it.
- `npm audit --audit-level=high` passes, but Wrangler/Miniflare currently pulls three moderate `ws` advisories; `npm audit fix --force` would downgrade Wrangler and is not applied.
- The skills-library repo has many unrelated pre-existing dirty changes on branch `codex/skill-audit-apply-downloads`; stage only BrandOS-specific files if committing that repair separately.
- Codex plugin v1 is valid but not fully productionized until it can handle missing/stale CLI binaries, pass clean-environment Codex installation smokes, ship production plugin assets, and validate through CI without relying on a local Codex skill path.
- Bead 33 content is intentionally broad enough for launch (8 guides, 5 resources), but the next SEO quality pass should deepen the 2-3 highest-intent pages with examples, screenshots, and stronger differentiated proof before scaling more programmatic pages.
