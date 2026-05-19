# HANDOFF.md - Baseline.ai

## Current Thread
- Working branch: `codex/feat/bead-25-cloud-accounts-remote-mcp-mac`.
- Current request handled in two beads:
  - Bead 23B: Pro account architecture doc committed as `96d2e28`.
  - Bead 23A: landing/design/docs/blog/pro checkout stub implementation committed as `257c17f`.
  - Bead 24: refreshed Worker deployed to https://baseline-ai.ryan-borker.workers.dev at version `5cc879a3-983d-4e59-a620-e8abd8d70a99`.
  - Bead 25: cloud accounts, Stripe/Klaviyo entitlement lifecycle, account-scoped tokens, remote MCP, SwiftUI macOS hotspot client, skill audit, and Worker deployment version `dfc2198f-9151-4a64-8511-4e25d3c2d529`.

## Key Context
- Existing app is a Cloudflare Worker in `web/src/index.ts`.
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

## Active Beads
- Bead 25 implementation is complete locally and deployed; commit/push/evidence is pending.

## Commands To Re-run
- `cd /Users/kikimac/.hermes/repos/apollostreetcompany/baseline`
- `make verify-all`
- `cd web && npm run typecheck`
- `cd web && npm run dev`
- `go test ./...`
- `cd macos/BaselineHotspots && swift build`
- `cd web && npm audit --audit-level=high`

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

## Open Risks
- Live Stripe, Klaviyo, Neon, and deployment verification require production/staging secrets and should not print secret values.
- Real paid pilot launch requires production `MAGIC_LINK_SECRET`, `TOKEN_HMAC_SECRET`, `STRIPE_SECRET_KEY`, `STRIPE_PRICE_ID_PRO`, `STRIPE_WEBHOOK_SECRET`, and Klaviyo settings.
- The remote MCP implementation is an HTTP JSON-RPC endpoint shaped for MCP clients; before public announcement it should be smoke-tested against the exact target clients that will register it.
- `npm audit --audit-level=high` passes, but Wrangler/Miniflare currently pulls three moderate `ws` advisories; `npm audit fix --force` would downgrade Wrangler and is not applied.
