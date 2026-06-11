# Bead 37 Checkout Router Plan

Date: 2026-06-11

## Goal

Make Baseline Pro checkout commercially usable without weakening the current billing safety model. A buyer or test agent should be able to find checkout, understand the rules, enter `FounderBaseline`, complete the real Stripe Checkout flow at 100% off, receive the same account/magic-link/workspace-token path as a paid buyer, and leave useful Klaviyo/Datafa.st receipts.

## Entry Gate

- Workstream: code, docs, ops/deploy.
- Risk class: high, because checkout, Stripe discounts, lifecycle routing, Worker runtime behavior, and production deployment are touched.
- Agent path: primary Codex as architect/engineer, `subreview` as adversarial planning/review input, deterministic validation before merge/deploy.
- Required tools: RepoPrompt code map, `apollo-ecommerce-router` skill routing, `subreview`, Stripe API, Cloudflare `cf`, local Worker smokes.
- Definition of done: public checkout links exist; `/checkout` explains rules; `/api/checkout` accepts a validated coupon code; Stripe sessions attach a configured 100% promotion code only for `FounderBaseline`; webhook-verified entitlement remains the source of truth; Klaviyo gets customer and master events; Datafa.st gets checkout/coupon/return events; validation and production receipts are recorded.

## Current State

- Existing checkout is already email-first and uses Stripe Checkout Sessions instead of payment links when account provisioning is required.
- Verified Stripe webhooks grant entitlement; the return URL does not grant access.
- Checkout-start already emits best-effort buyer and master Klaviyo events.
- Checkout completion queues lifecycle outbox rows and sends a customer magic-link event, but direct master routing on webhook completion is weaker than checkout-start routing.
- Datafa.st currently tracks pricing checkout starts, redirects, success returns, cancel returns, install clicks, lead magnets, and pilot requests.
- Public pricing forms submit checkout but there is no dedicated checkout page or coupon field, so the rules are too implicit.

## Targeted Changes

1. Klaviyo routing
   - Preserve buyer checkout-start and buyer magic-link/subscription events.
   - Add a direct master notification on processed Stripe webhook events with redacted properties: event type, plan, coupon presence, customer email presence, session/subscription id, and account id when available.
   - Include coupon metadata in checkout-start and subscription-start properties.

2. Transaction processing
   - Keep the current rule: paid access is granted only from verified Stripe webhook events.
   - Keep payment links disabled for Baseline Pro onboarding because they cannot guarantee account metadata.
   - Extend `POST /api/checkout` input to accept `couponCode`.
   - Accept only the configured founder coupon code, defaulting to `FounderBaseline`.
   - Apply Stripe Checkout `discounts[0][promotion_code]` from `STRIPE_FOUNDER_PROMOTION_CODE_ID` when the code matches.
   - Fail closed with a clear message when a coupon is supplied but the promotion code secret is missing.
   - Use `payment_method_collection=if_required` for founder-code sessions so a $0 test checkout does not require a card when Stripe does not need one.
   - Allow failed Stripe webhook rows to reprocess on retry instead of treating all event-id conflicts as processed duplicates.

3. Datafa.st handling
   - Add client events for `checkout_coupon_applied`, richer `checkout_start`, `checkout_redirect`, and checkout return states.
   - Include `plan`, `coupon_present`, canonical `coupon_code` only after server validation, `provider`, and page location.
   - Carry the `datafast_visitor_id` cookie into Stripe metadata when present for payment attribution.
   - Do not add a server-side Datafa.st token dependency to the Worker.

4. Clear checkout rules and links
   - Add a dedicated `/checkout` page.
   - Link to it from nav, footer, pricing cards, and `/api/checkout` email-required fallback.
   - Explain the rules plainly: local CLI/MCP is free; Pro/Team require email-first Stripe Checkout; the founder coupon still goes through Stripe/webhook/magic-link/workspace-token setup; refunds/cancellations stay in Stripe Billing Portal, not MCP.
   - Do not print the live founder code on the public checkout page; agents/operators can pass the code explicitly.

5. FounderBaseline coupon
   - Create or verify a Stripe promotion code named `FounderBaseline` backed by a 100% coupon for agent testing.
   - Store the promotion-code id in the Worker secret `STRIPE_FOUNDER_PROMOTION_CODE_ID`.
   - Keep the live promotion code capped and backed by a `duration=forever`, `percent_off=100` coupon so founder-code test subscriptions do not renew into a paid invoice.
   - Smoke by creating a live checkout session with a synthetic email and the coupon, then inspect the session through Stripe without completing checkout.

## Validation Plan

- `npm --prefix web run typecheck`
- `make verify`
- `git diff --check`
- Local Worker smokes for `/`, `/checkout`, `/api/checkout?plan=pro`, `/checkout/success`, `/checkout/cancel`, `/api/health`.
- Live production smokes after merge/deploy: health, checkout page markers, email-required fallback, coupon checkout session creation, session readback through Stripe, and no unauthenticated admin/MCP regression.
- Update `CONTINUITY.md`, `HANDOFF.md`, `docs/DEPLOYMENT.md`, and `handoff/beads.jsonl`.

## Subreview Findings Acted On

- Claude Fable 5 implementation review completed through `subreview` at `/tmp/baseline-subreview-checkout-implementation/claude.md`.
- Acted on: do not publicly print `FounderBaseline`; verified Stripe-side `max_redemptions=50`, `percent_off=100`, and `duration=forever`; tag founder entitlements with `source=stripe_founder`; reprocess failed webhook events on Stripe retry; send canonical coupon telemetry only after server validation; return `coupon_present` rather than `coupon_hint` from checkout session status; rename master webhook `session_id` telemetry to `object_id`.
- Considered and kept: `@cloudflare/wrangler-bundler` remains because current `cf dev` requires a declared dev server package and `make web-dev` was broken in the fresh worktree.

## Open Questions

- None for the planned implementation. The final live checkout-session smoke remains pending until merge/deploy.
