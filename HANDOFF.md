# HANDOFF.md - Baseline.ai

## Current Thread
- Working branch: `codex/feat/bead-23-brand-landing-pro-flow`.
- Current request handled in two beads:
  - Bead 23B: Pro account architecture doc committed as `96d2e28`.
  - Bead 23A: landing/design/docs/blog/pro checkout stub implementation committed as `257c17f`.
  - Bead 24: refreshed Worker deployed to https://baseline-ai.ryan-borker.workers.dev at version `5cc879a3-983d-4e59-a620-e8abd8d70a99`.

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

## Active Beads
- Bead 25 is next: dogfood/admin token split, Stripe entitlement, or API token/workspace model.

## Commands To Re-run
- `cd /Users/kikimac/.hermes/repos/apollostreetcompany/baseline`
- `cd web && npm run typecheck`
- `cd web && npm run dev`
- `go test ./...`
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

## Open Risks
- Live Stripe, Klaviyo, Neon, and deployment verification require production/staging secrets and should not print secret values.
- Pro account persistence can be scaffolded in Worker/Neon, but real entitlement launch requires configured Stripe price ids or payment links.
- `npm audit --audit-level=high` passes, but Wrangler/Miniflare currently pulls three moderate `ws` advisories; `npm audit fix --force` would downgrade Wrangler and is not applied.
