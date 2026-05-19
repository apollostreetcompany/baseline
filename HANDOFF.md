# HANDOFF.md - Baseline.ai

## Current Thread
- Working branch: `codex/feat/bead-29-distribution-and-pro-activation`.
- Current request history:
  - Bead 23B: Pro account architecture doc committed as `96d2e28`.
  - Bead 23A: landing/design/docs/blog/pro checkout stub implementation committed as `257c17f`.
  - Bead 24: refreshed Worker deployed to https://baseline-ai.ryan-borker.workers.dev at version `5cc879a3-983d-4e59-a620-e8abd8d70a99`.
  - Bead 25: cloud accounts, Stripe/Klaviyo entitlement lifecycle, account-scoped tokens, remote MCP, SwiftUI macOS hotspot client, skill audit, and Worker deployment version `dfc2198f-9151-4a64-8511-4e25d3c2d529`.
  - Bead 27: `landing-a` homepage redesign plus local BrandOS runtime repair, deployed to Cloudflare Worker version `4f1b94a0-543a-4cb2-8207-62825fb29594`.
  - Integration: PR #1 (`https://github.com/apollostreetcompany/baseline/pull/1`) combines Bead 25 cloud/Mac app functionality with Bead 27 landing before merge to `main`.
  - Bead 28: Cloudflare custom domain deployment makes `https://trackbaseline.com` the canonical public URL, with `www.trackbaseline.com` and workers.dev fallback triggers.
  - Bead 29: public distribution and Pro activation. Implementation commit `63e8d1bb59663fee502c18aad36141b5bd5fa1dd`; Worker deploy `e38523fc-d11a-41d9-b05e-6dcef5f4b5f0`; GitHub Release `v0.1.0` published.

## Key Context
- Existing app is a Cloudflare Worker in `web/src/index.ts`.
- Canonical production URL is now `https://trackbaseline.com`.
- Public install command is now `curl -fsSL https://trackbaseline.com/install.sh | sh`, backed by GitHub Release assets and checksum verification.
- Production Pro secrets are active: Stripe Checkout, Stripe webhook verification, Klaviyo lifecycle email, magic-link auth, and HMAC workspace tokens. Do not print secret values.
- Existing checkout route supports Stripe payment links or direct Stripe Checkout sessions.
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
- Bead 29 evidence/update commit is active after public release and production secret activation.

## Commands To Re-run
- `cd /Users/kikimac/.hermes/repos/apollostreetcompany/baseline`
- `make verify-all`
- `cd web && npm run typecheck`
- `cd web && npm run dev`
- `go test ./...`
- `cd macos/BaselineHotspots && swift build`
- `cd web && npm audit --audit-level=high`
- `bash scripts/build-release.sh`
- `npm --prefix package pack --dry-run`
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

## Open Risks
- Live Stripe, Klaviyo, Neon, and deployment verification require production/staging secrets and must never print secret values.
- Real paid pilot launch still needs an end-to-end checkout with an intended pilot email, webhook entitlement confirmation, magic-link login, token issuance, redacted sync, and remote MCP account-status smoke.
- The remote MCP implementation is an HTTP JSON-RPC endpoint shaped for MCP clients; before public announcement it should be smoke-tested against the exact target clients that will register it.
- `npm audit --audit-level=high` passes, but Wrangler/Miniflare currently pulls three moderate `ws` advisories; `npm audit fix --force` would downgrade Wrangler and is not applied.
- The skills-library repo has many unrelated pre-existing dirty changes on branch `codex/skill-audit-apply-downloads`; stage only BrandOS-specific files if committing that repair separately.
