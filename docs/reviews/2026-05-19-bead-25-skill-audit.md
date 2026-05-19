# Bead 25 Skill Audit

Date: 2026-05-19

Scope: cloud accounts, remote MCP, Stripe/Klaviyo Pro lifecycle, workspace tokens, comparison APIs, deployment, and macOS hotspot client.

## Skills Applied

- `revenue-plumber`: subscription lifecycle, Stripe webhooks, Klaviyo revenue events, metrics discipline.
- `stripe-checkout`: hosted Checkout, database as source of truth, webhook signature verification, CLI/token subscription enforcement.
- `apollo-ecommerce-router`: purchase-point routing, `site_id` metadata, Stripe-to-Klaviyo flow, master-notification mindset.
- `churn-prevention`: Stripe Billing Portal handoff, dunning/payment-failed lifecycle, avoid dark-pattern cancellation.
- `mcp-server-design`: bounded tools, confirmation for mutations, discoverable schemas, structured recovery hints.
- `documentation-website-for-software-project`: narrative plus reference, orientation before route tables, deployment/runbook freshness.

## Findings

1. Stripe Checkout should be preferred over payment links when `STRIPE_SECRET_KEY` and price IDs exist because Checkout can attach account/user metadata. Payment links remain a fallback only.
2. Webhook handling needed a failed-payment path. Added `invoice.payment_failed` handling, `past_due` entitlement state, audit events, and lifecycle outbox rows for dunning.
3. Workspace tokens needed explicit scope enforcement. `/api/runs` now requires `runs:write` on account-scoped tokens and still supports the temporary dogfood token.
4. Past-due accounts should keep existing monitoring during dunning grace but should not create new workspace tokens. Token creation now rejects `past_due` with a recoverable billing error and portal next action.
5. MCP tool schemas were too generic for agents. Tool descriptions now include discovery hints, confirmation requirements, raw-data boundaries, and action enums.
6. Documentation needed to record the remote MCP endpoint, auth flow, required secrets, live smoke results, rollback path, and Mac app build command. README and deployment notes now cover those.

## Remaining Gaps

- Production secrets are not configured, so live Pro signup/auth/token/webhook flows remain fail-closed.
- The lifecycle outbox is durable, but a retry worker/drainer is not implemented yet.
- Exact remote MCP compatibility should be smoke-tested against the target MCP clients before public launch.
- A user-facing cancellation/save-offer flow is not implemented; billing management is delegated to Stripe Billing Portal for v1.
- Team and anonymous benchmark comparison modes are stored-for-later only and not exposed.

## Validation

- `make verify-all`
- `git diff --check`
- `cd web && npm audit --audit-level=high`
- Local `/mcp` unauthenticated smoke returned a `401` challenge.
- Live `/mcp` unauthenticated smoke returned a `401` challenge after deploy.
