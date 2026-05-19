# Bead 23B: Pro Account Architecture

Date: 2026-05-19

Owned file: `docs/plans/2026-05-19-001-pro-account-architecture.md`

## Goals

1. Add a real Baseline Pro account layer for paid monitoring without weakening the local-first privacy promise.
2. Reuse the useful Bible-Coder billing lifecycle pattern: public checkout creates or finds a user by email, Stripe receives internal metadata, success pages verify session state, webhooks are idempotent, entitlement state is stored in Neon, and Klaviyo plus owner notifications are emitted from lifecycle events.
3. Keep the first implementation Cloudflare-first because Baseline already has a Worker, Neon run sync, Stripe Checkout scaffolding, dashboard APIs, and admin/evaluator endpoints.
4. Avoid fake access grants. Checkout redirect success is not authorization; verified Stripe webhooks are the source of truth.

## Assumptions

- ASSUMPTION: Pro v1 is one paid user/account with one or more local workspaces. Team seats and account members can be added after paid pilot proof.
- ASSUMPTION: Baseline should keep raw prompts, raw outputs, full local paths, and secrets local by default; cloud Pro stores redacted monitoring summaries and hashes.
- ASSUMPTION: Stripe subscription mode is the launch billing shape. One monthly Pro price is enough for the first rollout.
- ASSUMPTION: Klaviyo is for lifecycle/customer communication, not operational authorization.
- ASSUMPTION: The current global `BASELINE_API_TOKEN` stays as a dogfood fallback until real workspace API tokens are shipped.
- ASSUMPTION: Cloudflare Queues are optional for v1; `ctx.waitUntil` plus a Neon outbox is enough until retry volume proves otherwise.

## Current State

- `web/src/index.ts` is a Cloudflare Worker with:
  - public landing, dashboard, admin, privacy, terms, MCP docs, robots, sitemap;
  - `/api/health`;
  - `/api/runs` ingest guarded by one global `BASELINE_API_TOKEN`;
  - `/api/runs/latest` and `/api/runs/timeline`;
  - admin question-set and evaluator endpoints guarded by `BASELINE_ADMIN_TOKEN`;
  - `/api/events` for simple event capture;
  - `/api/checkout` with Stripe Checkout or payment-link fallback.
- `web/schema.sql` currently defines `baseline_runs`, `baseline_events`, `canonical_question_sets`, and `llm_evaluations`.
- Local CLI sync sends reduced cloud payloads from `sync_outbox` to `/api/runs`; the payload contains timing, score, mode, agent kind, workspace hash/display hash, redaction status, check metadata, and metrics.
- There is no full user, account, workspace, API token, subscription, entitlement, Stripe event, Klaviyo event, or audit-log model yet.
- `docs/DEPLOYMENT.md` says the deployed Worker has `DATABASE_URL`, `BASELINE_API_TOKEN`, and `BASELINE_ADMIN_TOKEN`; Stripe is intentionally not live.

## Recommended Cloudflare-First Architecture

Use the current Worker as the billing/account edge and Neon as the canonical account store.

```text
Landing/dashboard/checkout form
  -> Cloudflare Worker
      -> Neon: users/accounts/workspaces/tokens/entitlements/audit
      -> Stripe: Checkout Sessions, Customers, Subscriptions, Webhooks
      -> Klaviyo: lifecycle events via async outbox/waitUntil
      -> Owner notification: redacted event notification
  -> Local CLI
      -> workspace API token
      -> /api/runs ingest
      -> Neon redacted run history
```

Cloudflare remains the recommended path because the current app is already there, webhook work is request-sized, and the latency-sensitive parts are database writes plus API calls. Use Cloudflare Workers secrets for sensitive values and keep only non-sensitive public config in plain vars. Use the current Neon serverless driver for the next bead; consider Cloudflare Hyperdrive once connection churn or p95 latency becomes a real issue.

Implementation shape:

- `POST /api/checkout` accepts email and plan, normalizes the email, creates or finds a `user`, creates or finds the billing `account`, creates or finds `stripe_customer`, then creates a Stripe Checkout Session.
- Stripe Checkout Session includes `client_reference_id`, `customer`, session metadata, and `subscription_data[metadata]` with internal IDs only: `user_id`, `account_id`, `plan_key`, and `checkout_intent_id`.
- `success_url` includes Stripe's checkout session placeholder and lands on a status page that calls a verification endpoint. This endpoint can say `pending`, `active`, `requires_payment_method`, or `unknown`; it must not grant entitlement by itself.
- `POST /api/stripe/webhook` reads the raw request body, verifies `Stripe-Signature`, inserts `stripe_events` by Stripe event id, and then performs idempotent state transitions.
- Entitlement state is derived from Stripe webhook events and stored in Neon. The dashboard and API-token issuance read from Neon entitlement state, not from query params or local client claims.
- Klaviyo and owner notifications are emitted after durable state updates. Failed lifecycle sends should be retried through an outbox instead of blocking Stripe webhook acknowledgment.

## Render Alternative

Use Render only if the Worker path starts fighting the product:

- webhook processing needs long-running background jobs;
- Stripe/Klaviyo SDK ergonomics or raw-body handling become too costly at the edge;
- account onboarding requires stateful queues, scheduled workers, or larger CPU windows;
- operational preference shifts toward a conventional service with logs, health checks, and deploy rollbacks.

Render shape:

```text
Cloudflare Worker/Pages
  -> landing, dashboard shell, static docs
  -> proxy billing/account API if desired

Render Web Service
  -> Node/Fastify or Go HTTP API
  -> Stripe Checkout and webhook handler
  -> Klaviyo and owner notification workers
  -> Neon connection pool
```

Render service requirements:

- bind to `0.0.0.0:$PORT`;
- expose `GET /healthz` returning `2xx` when the database and required secrets are usable;
- keep Cloudflare as the public frontend unless moving the whole app;
- keep the same Neon schema, token hashing, webhook idempotency, and audit-log contract;
- document rollback as reverting DNS/proxy routes to the Worker checkout-disabled path.

Tradeoff: Render gives a conventional server and simpler background processing, but adds another deploy surface and is unnecessary until the Worker has a concrete limitation.

## Data Model

Add tables with additive migrations only.

```sql
users (
  id text primary key,
  email text not null,
  email_normalized text not null unique,
  name text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

accounts (
  id text primary key,
  primary_user_id text not null references users(id),
  billing_email text not null,
  plan_key text not null default 'free',
  status text not null default 'pending',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

stripe_customers (
  id text primary key,
  account_id text not null references accounts(id),
  user_id text not null references users(id),
  stripe_customer_id text not null unique,
  email_normalized text not null,
  livemode boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

stripe_subscriptions (
  id text primary key,
  account_id text not null references accounts(id),
  stripe_customer_id text not null,
  stripe_subscription_id text not null unique,
  status text not null,
  price_id text not null,
  plan_key text not null,
  current_period_end timestamptz,
  cancel_at_period_end boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

entitlements (
  id text primary key,
  account_id text not null references accounts(id),
  key text not null,
  status text not null,
  source text not null,
  retention_days integer not null default 14,
  max_workspaces integer not null default 1,
  monitoring_enabled boolean not null default false,
  starts_at timestamptz,
  expires_at timestamptz,
  updated_at timestamptz not null default now(),
  unique (account_id, key)
);

workspaces (
  id text primary key,
  account_id text not null references accounts(id),
  workspace_hash text not null,
  display_name_redacted text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (account_id, workspace_hash)
);

api_tokens (
  id text primary key,
  workspace_id text not null references workspaces(id),
  token_prefix text not null,
  token_hash text not null unique,
  scopes text[] not null,
  created_at timestamptz not null default now(),
  last_seen_at timestamptz,
  revoked_at timestamptz
);

stripe_events (
  id text primary key,
  stripe_event_id text not null unique,
  event_type text not null,
  livemode boolean not null default false,
  payload_hash text not null,
  processed_at timestamptz,
  status text not null default 'pending',
  error text,
  created_at timestamptz not null default now()
);

audit_log (
  id text primary key,
  actor_type text not null,
  actor_id text,
  action text not null,
  subject_type text not null,
  subject_id text,
  idempotency_key text,
  metadata_redacted jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

lifecycle_event_outbox (
  id text primary key,
  provider text not null,
  event_name text not null,
  subject_type text not null,
  subject_id text not null,
  destination text not null,
  payload_redacted jsonb not null,
  idempotency_key text not null unique,
  status text not null default 'pending',
  attempts integer not null default 0,
  next_attempt_at timestamptz not null default now(),
  last_error text,
  created_at timestamptz not null default now(),
  sent_at timestamptz
);
```

Extend existing `baseline_runs` instead of replacing it:

- add nullable `account_id`;
- add nullable `workspace_id`;
- add `expires_at`;
- keep current payload shape for compatibility;
- resolve `workspace_id` from the API token for Pro ingest, not from caller-submitted workspace text.

## Routes

Keep current routes working. Add versioned routes where the behavior becomes account-aware.

Public/account:

- `POST /api/checkout`: create/find user and account by email, create Stripe Checkout Session, redirect or return JSON.
- `GET /api/checkout/session?session_id=...`: verify checkout/session status for the success page without granting entitlement.
- `POST /api/stripe/webhook`: verified Stripe webhook handler.
- `GET /api/pro/status?session_id=...`: dashboard helper for onboarding status.

Authenticated workspace/API:

- `POST /api/tokens`: create a workspace token after Pro entitlement is active; first version can be admin-assisted.
- `DELETE /api/tokens/:id`: revoke token.
- `POST /api/runs`: continue accepting current dogfood token; when a workspace token is supplied, resolve `account_id` and `workspace_id`.
- `GET /api/runs/latest` and `GET /api/runs/timeline`: filter by authenticated workspace once dashboard auth exists; keep current public/demo behavior until then.

Admin/internal:

- `POST /api/admin/entitlements/recompute`: admin-only repair path that re-reads Stripe state and updates entitlements.
- `POST /api/admin/lifecycle/retry`: admin-only outbox retry.
- `GET /api/admin/accounts`: pilot support view with redacted account, entitlement, and token status.

## Env Vars And Secrets

Cloudflare Worker secrets:

- `DATABASE_URL`
- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `STRIPE_PRICE_ID_PRO`
- `TOKEN_HMAC_SECRET`
- `BASELINE_ADMIN_TOKEN`
- `KLAVIYO_PRIVATE_API_KEY`
- `OWNER_NOTIFICATION_WEBHOOK_URL` or provider-specific owner notification secret

Plain vars:

- `APP_URL`
- `KLAVIYO_REVISION`
- `KLAVIYO_OWNER_EMAIL`
- `PRO_RETENTION_DAYS`
- `FREE_RETENTION_DAYS`
- `CHECKOUT_ENABLED`
- `KLAVIYO_ENABLED`
- `OWNER_NOTIFICATIONS_ENABLED`

Optional/future:

- `STRIPE_PRICE_ID_TEAM`
- `STRIPE_BILLING_PORTAL_RETURN_URL`
- `HYPERDRIVE` binding or `DATABASE_HYPERDRIVE`
- `RENDER` and `PORT` if the Render alternative is used

Do not place secrets in `wrangler.jsonc`, committed docs, logs, lifecycle payloads, Klaviyo event properties, or owner notifications.

## Webhook Lifecycle

1. Checkout request arrives with email and `plan=pro`.
2. Worker normalizes email, creates/finds `users`, `accounts`, and `stripe_customers`.
3. Worker creates a `checkout_intent` audit-log entry and Stripe Checkout Session with internal metadata.
4. User completes or cancels Checkout on Stripe.
5. Success page calls session verification endpoint and displays pending/active state.
6. Stripe sends webhook.
7. Worker reads raw body, verifies signature, hashes payload, inserts `stripe_events` by Stripe event id.
8. Duplicate event ids return `200` after confirming the original was already processed.
9. Handler processes event in a transaction-like sequence:
   - `checkout.session.completed`: verify customer/subscription, upsert subscription, set Pro entitlement active or trialing, record audit event, enqueue Klaviyo and owner events.
   - `customer.subscription.created|updated`: upsert subscription, recompute entitlement status and expiry.
   - `customer.subscription.deleted`: mark subscription canceled and entitlement inactive after period end rules.
   - `invoice.payment_succeeded`: refresh subscription period, set entitlement active, emit payment-success lifecycle event.
   - `invoice.payment_failed`: mark entitlement `past_due` or grace state, emit payment-failed lifecycle event.
10. Worker acknowledges Stripe only after the state transition and outbox insert succeed.
11. Outbox dispatch runs in `ctx.waitUntil`; failed sends remain retryable.

## Klaviyo Events

All Klaviyo payloads must use the normalized email or Klaviyo profile id plus an internal Baseline account id. Do not include raw prompt text, raw response text, local paths, API tokens, Stripe secrets, or full Stripe payloads.

Recommended metrics:

- `Baseline Checkout Started`
- `Baseline Checkout Completed`
- `Baseline Pro Activated`
- `Baseline Pro Past Due`
- `Baseline Pro Canceled`
- `Baseline Payment Failed`
- `Baseline Workspace Connected`
- `Baseline API Token Created`
- `Baseline Run Synced`
- `Baseline Monitor Warning`

Suggested event properties:

- `account_id`
- `plan_key`
- `checkout_intent_id`
- `stripe_customer_id_last4` or hash, not the full id if unnecessary
- `workspace_count`
- `health_score`
- `status`
- `warning_count`
- `redaction_status`
- `retention_days`

Owner notifications should mirror only redacted summaries:

- new checkout started;
- Pro activated;
- first workspace connected;
- first synced warning;
- payment failed or subscription canceled;
- webhook processing failure requiring manual repair.

## Security And Privacy Constraints

- Verified webhooks, not redirect URLs, grant access.
- Stripe webhook signature verification must use the raw request body.
- Store one row per Stripe event id before side effects and make handlers idempotent.
- Store API tokens as prefix plus HMAC hash using `TOKEN_HMAC_SECRET`; never store raw tokens.
- Enforce token scopes: `ingest:write`, `reports:read`, and future `alerts:write`.
- Resolve workspace/account from token server-side.
- Keep raw prompts, raw outputs, local paths, secrets, and private keys out of cloud payloads, Klaviyo, owner notifications, and logs.
- Keep checkout and webhook responses generic; detailed failure context belongs in `audit_log` with redacted metadata.
- Rate-limit public checkout by IP/email where Cloudflare tooling is available.
- Use additive migrations and feature flags so checkout can be disabled without taking down run sync.
- Separate dogfood admin token from user/workspace API tokens before external pilot.
- Retention is application-level. Set `expires_at` on run rows and purge by policy; do not confuse Neon backup windows with product retention.

## Rollout Beads

1. Bead 24: Add account/billing schema.
   - Risk: high, because it changes auth/billing data contracts.
   - Scope: additive Neon schema for users, accounts, stripe customers, subscriptions, entitlements, events, audit, lifecycle outbox.
   - Validation: migration applies twice, existing run APIs still pass, duplicate unique constraints behave as expected.

2. Bead 25: Implement public Pro checkout.
   - Risk: high, because it touches billing and public routes.
   - Scope: email form/POST path, create/find user/account/customer, Checkout Session metadata, fail-closed config.
   - Validation: missing Stripe config returns `503`; checkout creates metadata; no entitlement is granted from success redirect.

3. Bead 26: Implement Stripe webhook entitlement lifecycle.
   - Risk: high, because it grants paid access.
   - Scope: raw-body signature verification, idempotent `stripe_events`, subscription upsert, entitlement recompute.
   - Validation: invalid signature fails; duplicate event is safe; mocked completed/updated/deleted events update entitlements correctly.

4. Bead 27: Implement Klaviyo and owner notification outbox.
   - Risk: medium.
   - Scope: lifecycle event payload builder, redaction guard, async dispatch, retry admin endpoint.
   - Validation: no secrets/raw payload in event JSON; failed dispatch retries; disabled Klaviyo does not block billing.

5. Bead 28: Add workspace API tokens and token-scoped ingest.
   - Risk: high, because it changes auth behavior.
   - Scope: token creation/revocation, HMAC storage, scopes, `/api/runs` workspace resolution.
   - Validation: unknown/revoked/wrong-scope token cannot ingest; global token fallback still works for dogfood if enabled.

6. Bead 29: Add Pro onboarding dashboard state.
   - Risk: medium.
   - Scope: checkout/session status UI, token setup instructions, current entitlement display, redacted latest-run state.
   - Validation: active/pending/past-due states render; no token is shown after creation; dashboard does not reveal other workspaces.

7. Bead 30: Decide whether Render is necessary.
   - Risk: low unless moved.
   - Scope: review Worker p95, webhook failure rate, lifecycle retry volume, and operational pain.
   - Validation: documented keep/move decision with rollback plan.

## Validation Checklist

Docs/process for this bead:

- [x] Architecture goals, current state, Cloudflare recommendation, Render alternative, data model, routes, secrets, webhook lifecycle, Klaviyo events, security/privacy, rollout beads, validation, and rollback are covered.
- [x] Assumptions are labeled.
- [x] No secrets are included.
- [x] No files outside this owned plan were edited.

Future implementation validation:

- [ ] Unit tests for email normalization and create/find account behavior.
- [ ] Unit tests for token generation, HMAC hashing, prefix lookup, revocation, and scope checks.
- [ ] Stripe webhook tests for missing signature, invalid signature, valid signature, duplicate event, unknown event, completed checkout, subscription updated, subscription canceled, invoice succeeded, and invoice failed.
- [ ] Checkout tests for missing config, missing price, metadata presence, customer reuse, and no entitlement on redirect.
- [ ] Klaviyo tests proving event payloads include only allowed redacted fields.
- [ ] Ingest tests proving workspace/account resolve from token and caller-submitted workspace text cannot cross accounts.
- [ ] Smoke test with Stripe CLI in test mode.
- [ ] Smoke test for deployed `/api/health`, checkout disabled/enabled state, webhook endpoint, and one redacted run sync.
- [ ] Audit-log review showing every entitlement transition has a durable reason.

## Rollback Path

1. Disable `CHECKOUT_ENABLED` or unset Stripe price/secret values so public checkout returns fail-closed.
2. Disable Stripe webhook endpoint in Stripe Dashboard or rotate `STRIPE_WEBHOOK_SECRET` if webhook integrity is in question.
3. Keep existing `/api/runs` dogfood token path available until workspace token ingest is proven.
4. Recompute or manually revoke `entitlements` through an admin-only repair command; do not delete account history during incident response.
5. Disable `KLAVIYO_ENABLED` and `OWNER_NOTIFICATIONS_ENABLED` if lifecycle sends leak, fail, or create noise.
6. Revert dashboard routing to the existing public/demo dashboard if authenticated account views fail.
7. Roll back database reads by ignoring new nullable columns. Migrations should be additive so existing `baseline_runs` and admin/evaluator flows keep working.

## Reference Links Checked

- Stripe Checkout Sessions: https://docs.stripe.com/api/checkout/sessions
- Stripe webhook signature verification: https://docs.stripe.com/webhooks/signature
- Cloudflare Workers secrets: https://developers.cloudflare.com/workers/configuration/secrets/
- Cloudflare Workers with Neon: https://developers.cloudflare.com/workers/databases/third-party-integrations/neon/
- Klaviyo Events API: https://developers.klaviyo.com/en/reference/create_event
- Render web service health checks: https://render.com/docs/health-checks
