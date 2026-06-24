import { neon, type NeonQueryFunction } from "@neondatabase/serverless";

export interface CloudEnv {
  DATABASE_URL?: string;
  APP_URL?: string;
  BASELINE_API_TOKEN?: string;
  BASELINE_ADMIN_TOKEN?: string;
  STRIPE_SECRET_KEY?: string;
  STRIPE_WEBHOOK_SECRET?: string;
  STRIPE_PRICE_ID_PRO?: string;
  STRIPE_PRICE_ID_TEAM?: string;
  STRIPE_FOUNDER_PROMOTION_CODE_ID?: string;
  BASELINE_FOUNDER_COUPON_CODE?: string;
  KLAVIYO_PRIVATE_API_KEY?: string;
  KLAVIYO_REVISION?: string;
  BASELINE_MASTER_EMAIL?: string;
  MAGIC_LINK_SECRET?: string;
  TOKEN_HMAC_SECRET?: string;
  MAGIC_LINK_DEV_ECHO?: string;
  PRO_RETENTION_DAYS?: string;
  FREE_RETENTION_DAYS?: string;
}

export type IngestContext =
  | { legacy: true; accountId: null; workspaceId: null; expiresAt: null; comparisonScope: "legacy"; tokenId: null }
  | { legacy: false; accountId: string; workspaceId: string; expiresAt: string | null; comparisonScope: "account-private"; tokenId: string };

type AccountSession = {
  session_id: string;
  user_id: string;
  account_id: string;
  email_normalized: string;
  role: string;
  account_status: string;
  plan_key: string;
};

type CheckoutAccount = {
  userId: string;
  accountId: string;
  customerId: string;
  checkoutIntentId: string;
};

type RunPayload = {
  run_id?: string;
  started_at?: string;
  duration_ms?: number;
  workspace?: string;
  workspace_hash?: string;
  agent_kind?: string;
  redaction_status?: string;
  status?: string;
  health_score?: number;
  mode?: string;
  checks?: Array<{
    check_id?: string;
    lane?: string;
    kind?: string;
    status?: string;
    severity?: number;
    score?: number;
    duration_ms?: number;
    metrics?: Record<string, number>;
  }>;
};

type KlaviyoEventResult = {
  configured: boolean;
  accepted: boolean;
  status?: number;
  error?: string;
};

const PRO_ENTITLEMENT = "baseline_pro";
const SESSION_TTL_DAYS = 30;

export async function handleCloudRoute(request: Request, env: CloudEnv, ctx?: ExecutionContext): Promise<Response | null> {
  const url = new URL(request.url);
  const read = request.method === "GET" || request.method === "HEAD";
  if (read && url.pathname === "/.well-known/oauth-protected-resource") return oauthProtectedResourceMetadata(env, request);
  if (read && url.pathname === "/.well-known/oauth-authorization-server") return oauthAuthorizationMetadata(env, request);
  if (request.method === "POST" && url.pathname === "/mcp") return remoteMCP(request, env);
  if (request.method === "POST" && url.pathname === "/api/auth/magic-link") return requestMagicLink(request, env, ctx);
  if ((request.method === "POST" || request.method === "GET") && url.pathname === "/api/auth/consume") return consumeMagicLink(request, env);
  if (request.method === "POST" && url.pathname === "/api/admin/invites") return adminCreateInvite(request, env, ctx);
  if (read && url.pathname === "/api/account/status") return accountStatus(request, env);
  if (read && url.pathname === "/api/workspaces") return listWorkspaces(request, env);
  if (request.method === "POST" && url.pathname === "/api/workspaces") return createWorkspace(request, env);
  if (request.method === "POST" && url.pathname === "/api/tokens") return createWorkspaceToken(request, env);
  if (request.method === "POST" && url.pathname === "/api/tokens/revoke") return revokeWorkspaceToken(request, env);
  if (read && url.pathname === "/api/history") return history(request, env);
  if (read && url.pathname === "/api/hotspots") return hotspots(request, env);
  if (read && url.pathname === "/api/compare") return compareSelfHistory(request, env);
  if (request.method === "POST" && url.pathname === "/api/billing/portal") return billingPortal(request, env);
  if (request.method === "POST" && url.pathname === "/api/stripe/webhook") return stripeWebhook(request, env, ctx);
  if (read && url.pathname === "/api/checkout/session") return checkoutSessionStatus(request, env);
  return null;
}

export async function ensureCloudSchema(sql: NeonQueryFunction<false, false>): Promise<void> {
  await sql`create table if not exists users (
    id text primary key,
    email text not null,
    email_normalized text not null unique,
    name text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
  )`;
  await sql`create table if not exists accounts (
    id text primary key,
    primary_user_id text not null references users(id),
    billing_email text not null,
    plan_key text not null default 'free',
    status text not null default 'pending',
    benchmark_consent boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
  )`;
  await sql`create table if not exists account_members (
    id text primary key,
    account_id text not null references accounts(id),
    user_id text not null references users(id),
    role text not null default 'owner',
    status text not null default 'active',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique(account_id, user_id)
  )`;
  await sql`create table if not exists account_invites (
    id text primary key,
    account_id text not null references accounts(id),
    user_id text not null references users(id),
    email_normalized text not null,
    role text not null default 'owner',
    status text not null default 'pending',
    invited_by text,
    expires_at timestamptz not null,
    accepted_at timestamptz,
    created_at timestamptz not null default now()
  )`;
  await sql`create index if not exists account_invites_email_idx on account_invites (email_normalized, created_at desc)`;
  await sql`create table if not exists magic_links (
    id text primary key,
    user_id text not null references users(id),
    account_id text not null references accounts(id),
    email_normalized text not null,
    token_prefix text not null,
    token_hash text not null unique,
    purpose text not null,
    expires_at timestamptz not null,
    consumed_at timestamptz,
    created_at timestamptz not null default now()
  )`;
  await sql`create index if not exists magic_links_email_idx on magic_links (email_normalized, created_at desc)`;
  await sql`create table if not exists account_sessions (
    id text primary key,
    user_id text not null references users(id),
    account_id text not null references accounts(id),
    token_prefix text not null,
    token_hash text not null unique,
    expires_at timestamptz not null,
    revoked_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz not null default now()
  )`;
  await sql`create index if not exists account_sessions_account_idx on account_sessions (account_id, created_at desc)`;
  await sql`create table if not exists stripe_customers (
    id text primary key,
    account_id text not null references accounts(id),
    user_id text not null references users(id),
    stripe_customer_id text not null unique,
    email_normalized text not null,
    livemode boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
  )`;
  await sql`create table if not exists stripe_subscriptions (
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
  )`;
  await sql`create table if not exists entitlements (
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
    unique(account_id, key)
  )`;
  await sql`create table if not exists workspaces (
    id text primary key,
    account_id text not null references accounts(id),
    workspace_hash text not null,
    display_name_redacted text not null,
    benchmark_consent boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique(account_id, workspace_hash)
  )`;
  await sql`create index if not exists workspaces_account_idx on workspaces (account_id, updated_at desc)`;
  await sql`create table if not exists api_tokens (
    id text primary key,
    account_id text not null references accounts(id),
    workspace_id text not null references workspaces(id),
    token_prefix text not null,
    token_hash text not null unique,
    scopes text[] not null,
    created_at timestamptz not null default now(),
    last_seen_at timestamptz,
    revoked_at timestamptz
  )`;
  await sql`create index if not exists api_tokens_workspace_idx on api_tokens (workspace_id, created_at desc)`;
  await sql`create table if not exists stripe_events (
    id text primary key,
    stripe_event_id text not null unique,
    event_type text not null,
    livemode boolean not null default false,
    payload_hash text not null,
    processed_at timestamptz,
    status text not null default 'pending',
    error text,
    created_at timestamptz not null default now()
  )`;
  await sql`create table if not exists audit_log (
    id text primary key,
    actor_type text not null,
    actor_id text,
    action text not null,
    subject_type text not null,
    subject_id text,
    idempotency_key text,
    metadata_redacted jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
  )`;
  await sql`create index if not exists audit_log_subject_idx on audit_log (subject_type, subject_id, created_at desc)`;
  await sql`create table if not exists lifecycle_event_outbox (
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
    updated_at timestamptz not null default now()
  )`;
  await sql`create table if not exists comparison_aggregates (
    id text primary key,
    account_id text not null references accounts(id),
    workspace_id text not null references workspaces(id),
    check_id text not null,
    lane text,
    kind text,
    observations_count integer not null default 0,
    warning_count integer not null default 0,
    average_score numeric,
    p95_duration_ms integer,
    latest_run_id text,
    latest_status text,
    latest_seen_at timestamptz,
    aggregate_scope text not null default 'account-private',
    updated_at timestamptz not null default now(),
    unique(account_id, workspace_id, check_id)
  )`;
  await sql`alter table baseline_runs add column if not exists account_id text`;
  await sql`alter table baseline_runs add column if not exists workspace_id text`;
  await sql`alter table baseline_runs add column if not exists expires_at timestamptz`;
  await sql`alter table baseline_runs add column if not exists account_private_payload jsonb`;
  await sql`alter table baseline_runs add column if not exists comparison_scope text not null default 'legacy'`;
  await sql`create index if not exists baseline_runs_account_created_idx on baseline_runs (account_id, created_at desc)`;
  await sql`create index if not exists baseline_runs_workspace_created_idx on baseline_runs (workspace_id, created_at desc)`;
}

export async function resolveRunIngestContext(
  request: Request,
  env: CloudEnv,
  sql: NeonQueryFunction<false, false>
): Promise<IngestContext | Response> {
  const auth = request.headers.get("authorization") || "";
  if (!auth.startsWith("Bearer ")) return json({ ok: false, error: "missing bearer token" }, 401);
  const token = auth.slice("Bearer ".length);
  if (env.BASELINE_API_TOKEN && token === env.BASELINE_API_TOKEN) {
    return { legacy: true, accountId: null, workspaceId: null, expiresAt: null, comparisonScope: "legacy", tokenId: null };
  }
  if (!env.TOKEN_HMAC_SECRET) return json({ ok: false, error: "TOKEN_HMAC_SECRET is not configured" }, 503);
  const tokenHash = await hmacHex(env.TOKEN_HMAC_SECRET, token);
  const rows = await sql`
    select t.id as token_id, t.account_id, t.workspace_id, e.status as entitlement_status, e.expires_at
    from api_tokens t
    join entitlements e on e.account_id = t.account_id and e.key = ${PRO_ENTITLEMENT}
    where t.token_hash = ${tokenHash}
      and t.revoked_at is null
      and 'runs:write' = any(t.scopes)
      and (e.status in ('active', 'trialing', 'pilot', 'past_due') and (e.expires_at is null or e.expires_at > now()))
    limit 1
  `;
  if (!rows.length) return json({ ok: false, error: "invalid, revoked, or inactive workspace token" }, 403);
  const row = rows[0] as Record<string, unknown>;
  await sql`update api_tokens set last_seen_at = now() where id = ${String(row.token_id)}`;
  return {
    legacy: false,
    accountId: String(row.account_id),
    workspaceId: String(row.workspace_id),
    expiresAt: row.expires_at ? String(row.expires_at) : null,
    comparisonScope: "account-private",
    tokenId: String(row.token_id)
  };
}

export async function recordRunAggregates(
  sql: NeonQueryFunction<false, false>,
  payload: RunPayload,
  context: IngestContext
): Promise<void> {
  if (context.legacy) return;
  const checks = Array.isArray(payload.checks) ? payload.checks.slice(0, 80) : [];
  for (const check of checks) {
    const checkID = String(check.check_id || check.kind || "unknown").slice(0, 180);
    const status = String(check.status || "unknown");
    const score = Number.isFinite(Number(check.score)) ? Number(check.score) : null;
    const duration = Number.isFinite(Number(check.duration_ms)) ? Math.max(0, Math.round(Number(check.duration_ms))) : null;
    await sql`
      insert into comparison_aggregates (
        id, account_id, workspace_id, check_id, lane, kind, observations_count, warning_count,
        average_score, p95_duration_ms, latest_run_id, latest_status, latest_seen_at, aggregate_scope, updated_at
      )
      values (
        ${crypto.randomUUID()}, ${context.accountId}, ${context.workspaceId}, ${checkID},
        ${check.lane || null}, ${check.kind || null}, 1, ${status === "ok" ? 0 : 1},
        ${score}, ${duration}, ${payload.run_id || ""}, ${status}, now(), 'account-private', now()
      )
      on conflict (account_id, workspace_id, check_id) do update set
        lane = excluded.lane,
        kind = excluded.kind,
        observations_count = comparison_aggregates.observations_count + 1,
        warning_count = comparison_aggregates.warning_count + excluded.warning_count,
        average_score = case
          when excluded.average_score is null then comparison_aggregates.average_score
          when comparison_aggregates.average_score is null then excluded.average_score
          else ((comparison_aggregates.average_score * comparison_aggregates.observations_count) + excluded.average_score) / (comparison_aggregates.observations_count + 1)
        end,
        p95_duration_ms = greatest(coalesce(comparison_aggregates.p95_duration_ms, 0), coalesce(excluded.p95_duration_ms, 0)),
        latest_run_id = excluded.latest_run_id,
        latest_status = excluded.latest_status,
        latest_seen_at = now(),
        updated_at = now()
    `;
  }
}

export async function prepareCheckoutAccount(
  sql: NeonQueryFunction<false, false>,
  env: CloudEnv,
  email: string,
  plan: string
): Promise<CheckoutAccount> {
  if (!env.STRIPE_SECRET_KEY) throw new Error("STRIPE_SECRET_KEY is not configured");
  await ensureCloudSchema(sql);
  const account = await ensureAccountForEmail(sql, email, plan === "team" ? "team" : "pro");
  const existing = await sql`select stripe_customer_id from stripe_customers where account_id = ${account.accountId} order by created_at desc limit 1`;
  let customerId = existing.length ? String((existing[0] as Record<string, unknown>).stripe_customer_id) : "";
  if (!customerId) {
    const body = new URLSearchParams({
      email,
      "metadata[account_id]": account.accountId,
      "metadata[user_id]": account.userId,
      "metadata[source]": "baseline_checkout"
    });
    const response = await fetch("https://api.stripe.com/v1/customers", {
      method: "POST",
      headers: {
        authorization: "Bearer " + env.STRIPE_SECRET_KEY,
        "content-type": "application/x-www-form-urlencoded"
      },
      body
    });
    const payload = await response.json<Record<string, unknown>>();
    if (!response.ok || typeof payload.id !== "string") throw new Error(stripeError(payload, "Stripe customer creation failed"));
    customerId = payload.id;
    await sql`
      insert into stripe_customers (id, account_id, user_id, stripe_customer_id, email_normalized, livemode)
      values (${crypto.randomUUID()}, ${account.accountId}, ${account.userId}, ${customerId}, ${email}, ${Boolean(payload.livemode)})
      on conflict (stripe_customer_id) do update set updated_at = now()
    `;
  }
  const checkoutIntentId = crypto.randomUUID();
  await audit(sql, "system", account.userId, "checkout.intent_created", "account", account.accountId, checkoutIntentId, { plan });
  return { ...account, customerId, checkoutIntentId };
}

export function appendCheckoutMetadata(body: URLSearchParams, prepared: CheckoutAccount | null, plan: string): void {
  body.set("metadata[plan_key]", plan);
  body.set("metadata[site_id]", "baseline-ai");
  body.set("subscription_data[metadata][plan_key]", plan);
  body.set("subscription_data[metadata][site_id]", "baseline-ai");
  if (!prepared) return;
  body.set("customer", prepared.customerId);
  body.set("client_reference_id", prepared.accountId);
  body.set("metadata[user_id]", prepared.userId);
  body.set("metadata[account_id]", prepared.accountId);
  body.set("metadata[checkout_intent_id]", prepared.checkoutIntentId);
  body.set("subscription_data[metadata][user_id]", prepared.userId);
  body.set("subscription_data[metadata][account_id]", prepared.accountId);
  body.set("subscription_data[metadata][checkout_intent_id]", prepared.checkoutIntentId);
}

async function adminCreateInvite(request: Request, env: CloudEnv, ctx?: ExecutionContext): Promise<Response> {
  const admin = requireCloudAdmin(request, env);
  if (admin) return admin;
  const sql = sqlOrResponse(env);
  if (sql instanceof Response) return sql;
  const input = await safeJSON<{ email?: string; role?: string; plan?: string; pilot?: boolean }>(request);
  const email = normalizeEmail(input.email);
  if (!email) return json({ ok: false, error: "valid email required" }, 400);
  if (!env.MAGIC_LINK_SECRET) return json({ ok: false, error: "MAGIC_LINK_SECRET is not configured" }, 503);
  await ensureCloudSchema(sql);
  const account = await ensureAccountForEmail(sql, email, input.plan === "team" ? "team" : "pro");
  const inviteId = crypto.randomUUID();
  const expiresAt = futureISO(7);
  await sql`
    insert into account_invites (id, account_id, user_id, email_normalized, role, status, invited_by, expires_at)
    values (${inviteId}, ${account.accountId}, ${account.userId}, ${email}, ${input.role || "owner"}, 'pending', 'admin', ${expiresAt})
  `;
  if (input.pilot) {
    await grantEntitlement(sql, account.accountId, "pilot", "admin_invite", input.plan === "team" ? "team" : "pro", null);
  }
  const link = await createMagicLink(sql, env, account.userId, account.accountId, email, "invite");
  await audit(sql, "admin", "admin", "invite.created", "account", account.accountId, inviteId, { email_present: true, pilot: Boolean(input.pilot) });
  ctx?.waitUntil(sendKlaviyoAuthEvent(env, email, "Baseline Magic Link", link.url, { purpose: "invite", account_id: account.accountId }));
  return json({
    ok: true,
    invite_id: inviteId,
    account_id: account.accountId,
    user_id: account.userId,
    delivery: { provider: "klaviyo", configured: Boolean(env.KLAVIYO_PRIVATE_API_KEY) },
    magic_link: env.MAGIC_LINK_DEV_ECHO === "true" ? link.url : undefined,
    next_actions: ["User opens the magic link, then creates workspace tokens after Pro or pilot entitlement is active."]
  });
}

async function requestMagicLink(request: Request, env: CloudEnv, ctx?: ExecutionContext): Promise<Response> {
  const sql = sqlOrResponse(env);
  if (sql instanceof Response) return sql;
  const input = await safeJSON<{ email?: string }>(request);
  const email = normalizeEmail(input.email);
  if (!email) return json({ ok: false, error: "valid email required" }, 400);
  if (!env.MAGIC_LINK_SECRET) return json({ ok: false, error: "MAGIC_LINK_SECRET is not configured" }, 503);
  await ensureCloudSchema(sql);
  const rows = await sql`
    select u.id as user_id, a.id as account_id
    from users u
    join account_members m on m.user_id = u.id and m.status = 'active'
    join accounts a on a.id = m.account_id
    where u.email_normalized = ${email}
    order by a.created_at desc
    limit 1
  `;
  if (rows.length) {
    const row = rows[0] as Record<string, unknown>;
    const link = await createMagicLink(sql, env, String(row.user_id), String(row.account_id), email, "login");
    ctx?.waitUntil(sendKlaviyoAuthEvent(env, email, "Baseline Magic Link", link.url, { purpose: "login" }));
  }
  return json({
    ok: true,
    message: "If that email is invited, a Baseline sign-in link has been sent.",
    next_actions: ["Open the link from email or paste it into the Baseline Hotspots app."]
  });
}

async function consumeMagicLink(request: Request, env: CloudEnv): Promise<Response> {
  const sql = sqlOrResponse(env);
  if (sql instanceof Response) return sql;
  if (!env.MAGIC_LINK_SECRET) return json({ ok: false, error: "MAGIC_LINK_SECRET is not configured" }, 503);
  await ensureCloudSchema(sql);
  const url = new URL(request.url);
  const input = request.method === "POST" ? await safeJSON<{ token?: string }>(request) : { token: url.searchParams.get("token") || undefined };
  const token = input.token || "";
  if (!token) return json({ ok: false, error: "token required" }, 400);
  const tokenHash = await hmacHex(env.MAGIC_LINK_SECRET, token);
  const rows = await sql`
    select id, user_id, account_id, email_normalized
    from magic_links
    where token_hash = ${tokenHash}
      and consumed_at is null
      and expires_at > now()
    limit 1
  `;
  if (!rows.length) return json({ ok: false, error: "magic link expired, consumed, or invalid" }, 401);
  const link = rows[0] as Record<string, unknown>;
  await sql`update magic_links set consumed_at = now() where id = ${String(link.id)} and consumed_at is null`;
  await sql`
    update account_invites
    set status = 'accepted', accepted_at = now()
    where account_id = ${String(link.account_id)} and user_id = ${String(link.user_id)} and status = 'pending'
  `;
  const sessionToken = randomToken("bls_");
  const sessionHash = await hmacHex(env.MAGIC_LINK_SECRET, sessionToken);
  const sessionId = crypto.randomUUID();
  const expiresAt = futureISO(SESSION_TTL_DAYS);
  await sql`
    insert into account_sessions (id, user_id, account_id, token_prefix, token_hash, expires_at)
    values (${sessionId}, ${String(link.user_id)}, ${String(link.account_id)}, ${sessionToken.slice(0, 14)}, ${sessionHash}, ${expiresAt})
  `;
  await audit(sql, "user", String(link.user_id), "session.created", "account", String(link.account_id), sessionId, {});
  return json({
    ok: true,
    session_token: sessionToken,
    expires_at: expiresAt,
    account_id: String(link.account_id),
    user_id: String(link.user_id),
    next_actions: ["Paste the session token into the Baseline Hotspots app or use it as a Bearer token for remote MCP."]
  }, 200, { "set-cookie": sessionCookie(sessionToken, SESSION_TTL_DAYS) });
}

async function accountStatus(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  const entitlement = await activeEntitlement(auth.sql, auth.session.account_id);
  return json({
    ok: true,
    account: {
      id: auth.session.account_id,
      status: auth.session.account_status,
      plan_key: auth.session.plan_key,
      role: auth.session.role,
      email_normalized: auth.session.email_normalized
    },
    entitlement,
    next_actions: entitlement?.monitoring_enabled
      ? ["Create a workspace token and run baseline sync push from the local CLI."]
      : ["Start Pro checkout or ask the owner to grant pilot access before creating workspace tokens."]
  });
}

async function listWorkspaces(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  const rows = await auth.sql`
    select w.id, w.workspace_hash, w.display_name_redacted, w.benchmark_consent, w.created_at, w.updated_at,
      count(t.id) filter (where t.revoked_at is null) as active_tokens
    from workspaces w
    left join api_tokens t on t.workspace_id = w.id
    where w.account_id = ${auth.session.account_id}
    group by w.id
    order by w.updated_at desc
  `;
  return json({ ok: true, workspaces: rows, next_actions: ["Use POST /api/tokens with a workspace_id to create a token."] });
}

async function createWorkspace(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  const input = await safeJSON<{ workspace_hash?: string; display_name_redacted?: string; benchmark_consent?: boolean }>(request);
  const workspaceHash = cleanToken(input.workspace_hash, 180);
  if (!workspaceHash) return json({ ok: false, error: "workspace_hash required" }, 400);
  const display = cleanToken(input.display_name_redacted, 120) || workspaceHash.slice(0, 32);
  const rows = await auth.sql`
    insert into workspaces (id, account_id, workspace_hash, display_name_redacted, benchmark_consent, updated_at)
    values (${crypto.randomUUID()}, ${auth.session.account_id}, ${workspaceHash}, ${display}, ${Boolean(input.benchmark_consent)}, now())
    on conflict (account_id, workspace_hash) do update set
      display_name_redacted = excluded.display_name_redacted,
      benchmark_consent = excluded.benchmark_consent,
      updated_at = now()
    returning id, workspace_hash, display_name_redacted, benchmark_consent, created_at, updated_at
  `;
  await audit(auth.sql, "user", auth.session.user_id, "workspace.upserted", "workspace", String((rows[0] as Record<string, unknown>).id), undefined, {});
  return json({ ok: true, workspace: rows[0], next_actions: ["Create a token for this workspace once Pro or pilot entitlement is active."] });
}

async function createWorkspaceToken(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  if (!env.TOKEN_HMAC_SECRET) return json({ ok: false, error: "TOKEN_HMAC_SECRET is not configured" }, 503);
  const entitlement = await activeEntitlement(auth.sql, auth.session.account_id);
  if (!entitlement?.monitoring_enabled) return json({ ok: false, error: "active or pilot entitlement required" }, 403);
  if (!canCreateWorkspaceToken(entitlement)) {
    return json({
      ok: false,
      error: "settle billing before creating new tokens",
      error_type: "BILLING_GRACE_LOCKED",
      recoverable: true,
      next_actions: ["Open the Stripe Billing Portal and update the payment method, then retry token creation."]
    }, 402);
  }
  const input = await safeJSON<{ workspace_id?: string; label?: string }>(request);
  if (!input.workspace_id) return json({ ok: false, error: "workspace_id required" }, 400);
  const workspace = await auth.sql`select id from workspaces where id = ${input.workspace_id} and account_id = ${auth.session.account_id} limit 1`;
  if (!workspace.length) return json({ ok: false, error: "workspace not found" }, 404);
  const token = randomToken("blp_");
  const tokenHash = await hmacHex(env.TOKEN_HMAC_SECRET, token);
  const tokenId = crypto.randomUUID();
  await auth.sql`
    insert into api_tokens (id, account_id, workspace_id, token_prefix, token_hash, scopes)
    values (${tokenId}, ${auth.session.account_id}, ${input.workspace_id}, ${token.slice(0, 14)}, ${tokenHash}, array['runs:write'])
  `;
  await audit(auth.sql, "user", auth.session.user_id, "api_token.created", "workspace", input.workspace_id, tokenId, { scopes: ["runs:write"] });
  return json({
    ok: true,
    token_id: tokenId,
    token,
    token_prefix: token.slice(0, 14),
    scopes: ["runs:write"],
    next_actions: ["Store this token locally now. It will not be shown again.", "Run baseline sync on --url " + baseURL(env, request) + " --token <token>."]
  });
}

async function revokeWorkspaceToken(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  const input = await safeJSON<{ token_id?: string; confirm?: string }>(request);
  if (!input.token_id) return json({ ok: false, error: "token_id required" }, 400);
  const expected = "revoke " + input.token_id;
  if (input.confirm !== expected) {
    return json({
      ok: false,
      error: "confirmation required",
      error_type: "CONFIRMATION_REQUIRED",
      recoverable: true,
      confirmation: expected,
      next_actions: ["Repeat with confirm set exactly to the confirmation string."]
    }, 409);
  }
  const rows = await auth.sql`
    update api_tokens
    set revoked_at = now()
    where id = ${input.token_id} and account_id = ${auth.session.account_id} and revoked_at is null
    returning id, workspace_id
  `;
  if (!rows.length) return json({ ok: false, error: "token not found or already revoked" }, 404);
  await audit(auth.sql, "user", auth.session.user_id, "api_token.revoked", "api_token", input.token_id, input.confirm, {});
  return json({ ok: true, revoked: rows[0], next_actions: ["Remove the token from local Baseline config if it was still installed."] });
}

async function history(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  return json(await historyPayload(auth.sql, auth.session, new URL(request.url)));
}

async function hotspots(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  return json(await hotspotsPayload(auth.sql, auth.session, new URL(request.url)));
}

async function compareSelfHistory(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  return json(await comparePayload(auth.sql, auth.session, new URL(request.url)));
}

async function billingPortal(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env);
  if (auth instanceof Response) return auth;
  const portal = await createBillingPortal(auth.sql, env, auth.session, baseURL(env, request) + "/dashboard");
  if (portal instanceof Response) return portal;
  return json({ ok: true, url: portal.url, next_actions: ["Open the Stripe Billing Portal to manage payment method, invoices, or cancellation."] });
}

async function checkoutSessionStatus(request: Request, env: CloudEnv): Promise<Response> {
  const sql = sqlOrResponse(env);
  if (sql instanceof Response) return sql;
  await ensureCloudSchema(sql);
  const sessionID = new URL(request.url).searchParams.get("session_id") || "";
  if (!sessionID) return json({ ok: false, error: "session_id required" }, 400);
  if (!env.STRIPE_SECRET_KEY) return json({ ok: false, error: "STRIPE_SECRET_KEY is not configured" }, 503);
  const response = await fetch("https://api.stripe.com/v1/checkout/sessions/" + encodeURIComponent(sessionID), {
    headers: { authorization: "Bearer " + env.STRIPE_SECRET_KEY }
  });
  const session = await response.json<Record<string, unknown>>();
  if (!response.ok) return json({ ok: false, error: stripeError(session, "Stripe session lookup failed") }, 502);
  const metadata = session.metadata && typeof session.metadata === "object" ? session.metadata as Record<string, unknown> : {};
  let accountID = String(metadata.account_id || session.client_reference_id || "");
  const customerID = String(session.customer || "");
  const email = checkoutObjectEmail(session);
  if (!accountID && customerID) {
    const customers = await sql`select account_id from stripe_customers where stripe_customer_id = ${customerID} limit 1`;
    accountID = customers.length ? String((customers[0] as Record<string, unknown>).account_id) : "";
  }
  const rows = await sql`
    select e.account_id, e.status, e.monitoring_enabled, e.expires_at, a.plan_key
    from entitlements e
    join accounts a on a.id = e.account_id
    where e.key = ${PRO_ENTITLEMENT}
      and e.account_id = ${accountID}
    order by e.updated_at desc
    limit 1
  `;
  return json({
    ok: true,
    session_id: sessionID,
    payment_status: session.payment_status || null,
    plan_hint: metadata.plan_key || metadata.plan || null,
    coupon_present: Boolean(metadata.coupon_code),
    email_hint: email || null,
    entitlement_hint: rows[0] || null,
    next_actions: ["Stripe webhook is the source of truth. If status is pending, wait for webhook delivery or check Stripe events."]
  });
}

async function stripeWebhook(request: Request, env: CloudEnv, ctx?: ExecutionContext): Promise<Response> {
  const sql = sqlOrResponse(env);
  if (sql instanceof Response) return sql;
  if (!env.STRIPE_WEBHOOK_SECRET) return json({ ok: false, error: "STRIPE_WEBHOOK_SECRET is not configured" }, 503);
  await ensureCloudSchema(sql);
  const signature = request.headers.get("stripe-signature") || "";
  const rawBody = await request.text();
  const verified = await verifyStripeSignature(rawBody, signature, env.STRIPE_WEBHOOK_SECRET);
  if (!verified) return json({ ok: false, error: "invalid Stripe signature" }, 400);
  const event = JSON.parse(rawBody) as Record<string, unknown>;
  const eventID = String(event.id || "");
  const eventType = String(event.type || "");
  if (!eventID || !eventType) return json({ ok: false, error: "invalid Stripe event" }, 400);
  const payloadHash = await sha256Hex(rawBody);
  const inserted = await sql`
    insert into stripe_events (id, stripe_event_id, event_type, livemode, payload_hash, status)
    values (${crypto.randomUUID()}, ${eventID}, ${eventType}, ${Boolean(event.livemode)}, ${payloadHash}, 'pending')
    on conflict (stripe_event_id) do update set
      payload_hash = excluded.payload_hash,
      status = 'pending',
      error = null,
      processed_at = null
    where stripe_events.status = 'failed'
    returning id
  `;
  if (!inserted.length) return json({ ok: true, duplicate: true });
  try {
    await processStripeEvent(sql, event, env, ctx);
    await sql`update stripe_events set status = 'processed', processed_at = now() where stripe_event_id = ${eventID}`;
    const object = stripeObject(event);
    const metadata = object.metadata && typeof object.metadata === "object" ? object.metadata as Record<string, unknown> : {};
    const email = normalizeEmail(String(object.customer_email || object.email || "")) || "";
    if (email) ctx?.waitUntil(sendKlaviyoAuthEvent(env, email, "Baseline Billing Updated", "", { stripe_event_type: eventType }));
    const masterEmail = normalizeEmail(env.BASELINE_MASTER_EMAIL || "");
    if (masterEmail) {
      ctx?.waitUntil(sendKlaviyoAuthEvent(env, masterEmail, "Apollo Master Notification", "", {
        event_type: "stripe_webhook_processed",
        stripe_event_type: eventType,
        stripe_event_id: eventID,
        object_id: String(object.id || ""),
        subscription_id: String(object.subscription || ""),
        account_id: String(metadata.account_id || object.client_reference_id || ""),
        plan: String(metadata.plan_key || metadata.plan || ""),
        coupon_present: Boolean(metadata.coupon_code),
        coupon_code: String(metadata.coupon_code || ""),
        customer_email_present: Boolean(email),
        site_id: "trackbaseline.com",
        language: "en"
      }));
    }
    return json({ ok: true });
  } catch (error) {
    const message = error instanceof Error ? error.message : "unknown error";
    await sql`update stripe_events set status = 'failed', error = ${message}, processed_at = now() where stripe_event_id = ${eventID}`;
    return json({ ok: false, error: message }, 500);
  }
}

async function remoteMCP(request: Request, env: CloudEnv): Promise<Response> {
  const auth = await authenticated(request, env, true);
  if (auth instanceof Response) return auth;
  let rpc: { jsonrpc?: string; id?: unknown; method?: string; params?: Record<string, unknown> };
  try {
    rpc = await request.json();
  } catch {
    return mcpError(null, -32700, "Parse error");
  }
  try {
    if (rpc.method === "initialize") {
      return mcpResult(rpc.id, {
        protocolVersion: "2025-06-18",
        serverInfo: { name: "Baseline Remote MCP", version: "0.1.0" },
        capabilities: { tools: {} }
      });
    }
    if (rpc.method === "tools/list") return mcpResult(rpc.id, { tools: mcpTools() });
    if (rpc.method === "tools/call") {
      const params = rpc.params || {};
      const name = String(params.name || "");
      const args = (params.arguments && typeof params.arguments === "object" ? params.arguments : {}) as Record<string, unknown>;
      const payload = await callMCPTool(name, args, auth.sql, auth.session, env, request);
      return mcpResult(rpc.id, { content: [{ type: "text", text: JSON.stringify(payload, null, 2) }], structuredContent: payload });
    }
    return mcpError(rpc.id, -32601, "Method not found");
  } catch (error) {
    return mcpError(rpc.id, -32000, error instanceof Error ? error.message : "MCP tool failed");
  }
}

async function callMCPTool(
  name: string,
  args: Record<string, unknown>,
  sql: NeonQueryFunction<false, false>,
  session: AccountSession,
  env: CloudEnv,
  request: Request
): Promise<Record<string, unknown>> {
  const action = String(args.action || "status");
  if (name === "baseline_account") {
    const entitlement = await activeEntitlement(sql, session.account_id);
    if (action === "billing_portal") {
      const portal = await createBillingPortal(sql, env, session, baseURL(env, request) + "/dashboard");
      if (portal instanceof Response) return { ok: false, error: "portal_unavailable", next_actions: ["Check Stripe customer and STRIPE_SECRET_KEY configuration."] };
      return { ok: true, url: portal.url, next_actions: ["Open Stripe portal."] };
    }
    return { ok: true, account: session, entitlement, next_actions: entitlement?.monitoring_enabled ? ["Query baseline_history or baseline_hotspots."] : ["Start checkout or ask for pilot access."] };
  }
  if (name === "baseline_workspaces") {
    if (action === "list") return (await (await listWorkspaces(asRequest(request, "GET", "/api/workspaces"), env)).json()) as Record<string, unknown>;
    if (action === "create_workspace") {
      const workspaceHash = cleanToken(args.workspace_hash, 180);
      if (!workspaceHash) return { ok: false, error: "workspace_hash required" };
      const fake = jsonRequest(request, "/api/workspaces", { workspace_hash: workspaceHash, display_name_redacted: args.display_name_redacted });
      return (await (await createWorkspace(fake, env)).json()) as Record<string, unknown>;
    }
    if (action === "create_token") {
      const fake = jsonRequest(request, "/api/tokens", { workspace_id: args.workspace_id });
      return (await (await createWorkspaceToken(fake, env)).json()) as Record<string, unknown>;
    }
    if (action === "revoke_token") {
      const fake = jsonRequest(request, "/api/tokens/revoke", { token_id: args.token_id, confirm: args.confirm });
      return (await (await revokeWorkspaceToken(fake, env)).json()) as Record<string, unknown>;
    }
    return { ok: false, error: "unknown workspace action", next_actions: ["Use list, create_workspace, create_token, or revoke_token."] };
  }
  if (name === "baseline_history") return historyPayload(sql, session, new URL(request.url), Number(args.limit || 20));
  if (name === "baseline_hotspots") return hotspotsPayload(sql, session, new URL(request.url), Number(args.limit || 50));
  if (name === "baseline_compare") return comparePayload(sql, session, new URL(request.url));
  if (name === "baseline_subscription") {
    if (action === "portal") {
      const portal = await createBillingPortal(sql, env, session, baseURL(env, request) + "/dashboard");
      if (portal instanceof Response) return { ok: false, error: "portal_unavailable", next_actions: ["Subscription may not have a Stripe customer yet."] };
      return { ok: true, url: portal.url, next_actions: ["Open Stripe portal."] };
    }
    return { ok: true, checkout_url: baseURL(env, request) + "/api/checkout?plan=pro", next_actions: ["Start checkout from the web page if you need Pro access."] };
  }
  if (name === "baseline_admin") {
    if (session.role !== "owner") return { ok: false, error: "owner role required", next_actions: ["Ask an account owner to perform pilot support actions."] };
    if (action === "request_pilot_support") {
      await audit(sql, "user", session.user_id, "pilot_support.requested", "account", session.account_id, undefined, { source: "mcp" });
      return { ok: true, next_actions: ["Baseline support can review the audit log and grant pilot access manually."] };
    }
    const rows = await sql`select action, subject_type, subject_id, metadata_redacted, created_at from audit_log where subject_id = ${session.account_id} or actor_id = ${session.user_id} order by created_at desc limit 20`;
    return { ok: true, audit_events: rows, next_actions: ["Use request_pilot_support to create a support audit event."] };
  }
  return { ok: false, error: "unknown MCP tool", next_actions: ["Call tools/list for the supported Baseline tool cluster."] };
}

function mcpTools(): Array<Record<string, unknown>> {
  const accountSchema = {
    type: "object",
    additionalProperties: false,
    properties: {
      action: {
        type: "string",
        enum: ["status", "billing_portal"],
        description: "Use status for account/entitlement state. Use billing_portal for Stripe portal handoff; no direct cancellation mutation is exposed."
      }
    }
  };
  const workspaceSchema = {
    type: "object",
    additionalProperties: false,
    properties: {
      action: { type: "string", enum: ["list", "create_workspace", "create_token", "revoke_token"] },
      workspace_hash: { type: "string", description: "Redacted or hashed workspace identifier from local Baseline config or run payload." },
      display_name_redacted: { type: "string", description: "Human-friendly redacted label. Do not send local absolute paths." },
      workspace_id: { type: "string", description: "Workspace id discovered from action=list or create_workspace." },
      token_id: { type: "string", description: "Token id discovered from account token creation/audit UI." },
      confirm: { type: "string", description: "Required for revoke_token. Must be exactly: revoke <token_id>." }
    }
  };
  const limitSchema = {
    type: "object",
    additionalProperties: false,
    properties: {
      limit: { type: "integer", minimum: 1, maximum: 100 },
      workspace_id: { type: "string" }
    }
  };
  const subscriptionSchema = {
    type: "object",
    additionalProperties: false,
    properties: {
      action: { type: "string", enum: ["checkout", "portal", "status"], description: "checkout returns a checkout URL hint; portal returns Stripe Billing Portal handoff." }
    }
  };
  const adminSchema = {
    type: "object",
    additionalProperties: false,
    properties: {
      action: { type: "string", enum: ["audit", "request_pilot_support"], description: "Owner-only support action. Mutations write audit_log." }
    }
  };
  return [
    {
      name: "baseline_account",
      description: "Read account status, entitlement, and Stripe portal handoff. Discovery: call this first after auth to learn plan/status and next_actions. Do not use for cancellation; use portal handoff.",
      inputSchema: accountSchema
    },
    {
      name: "baseline_workspaces",
      description: "Manage account workspaces and scoped ingest tokens. Discovery: action=list returns workspace ids. Mutations audit. revoke_token requires confirm='revoke <token_id>'. Do not send raw local paths.",
      inputSchema: workspaceSchema
    },
    {
      name: "baseline_history",
      description: "Read account-private run history. Discovery: workspace_id is optional and comes from baseline_workspaces. Returns redacted runs only; no raw prompts/responses.",
      inputSchema: limitSchema
    },
    {
      name: "baseline_hotspots",
      description: "Read recurring failures, slow probes, model/token anomalies, and warning clusters from redacted history. Use after baseline_history when deciding where to investigate.",
      inputSchema: limitSchema
    },
    {
      name: "baseline_compare",
      description: "Compare self-history only. Team and anonymous benchmark modes are intentionally hidden until feature flags and consent are added.",
      inputSchema: limitSchema
    },
    {
      name: "baseline_subscription",
      description: "Start checkout or open Stripe Billing Portal handoff. No direct cancellation, refund, or payment-method mutation is available through MCP.",
      inputSchema: subscriptionSchema
    },
    {
      name: "baseline_admin",
      description: "Owner-only support and audit actions. Use request_pilot_support to create an auditable support request; action=audit reads recent audit events.",
      inputSchema: adminSchema
    }
  ];
}

async function historyPayload(
  sql: NeonQueryFunction<false, false>,
  session: AccountSession,
  url: URL,
  rawLimit?: number
): Promise<Record<string, unknown>> {
  const limit = boundedLimit(rawLimit || Number(url.searchParams.get("limit") || 30), 1, 100);
  const workspaceID = url.searchParams.get("workspace_id") || "";
  const rows = workspaceID
    ? await sql`select id, workspace, agent_kind, status, health_score, mode, payload, account_private_payload, created_at from baseline_runs where account_id = ${session.account_id} and workspace_id = ${workspaceID} order by created_at desc limit ${limit}`
    : await sql`select id, workspace, agent_kind, status, health_score, mode, payload, account_private_payload, created_at from baseline_runs where account_id = ${session.account_id} order by created_at desc limit ${limit}`;
  return { ok: true, runs: rows.map(normalizeRun), next_actions: ["Use baseline_hotspots for grouped failures or baseline_compare for latest deltas."] };
}

async function hotspotsPayload(
  sql: NeonQueryFunction<false, false>,
  session: AccountSession,
  url: URL,
  rawLimit?: number
): Promise<Record<string, unknown>> {
  const limit = boundedLimit(rawLimit || Number(url.searchParams.get("limit") || 50), 1, 100);
  const rows = await sql`
    select id, workspace, agent_kind, status, health_score, mode, payload, created_at
    from baseline_runs
    where account_id = ${session.account_id}
    order by created_at desc
    limit ${limit}
  `;
  const grouped = new Map<string, { check_id: string; kind: string; warning_count: number; run_count: number; max_duration_ms: number; latest_status: string; latest_run_id: string; average_score: number }>();
  for (const row of rows.map(normalizeRun)) {
    const checks = Array.isArray(row.checks) ? row.checks as Array<Record<string, unknown>> : [];
    for (const check of checks) {
      const key = String(check.check_id || check.kind || "unknown");
      const existing = grouped.get(key) || { check_id: key, kind: String(check.kind || ""), warning_count: 0, run_count: 0, max_duration_ms: 0, latest_status: "", latest_run_id: "", average_score: 0 };
      existing.run_count += 1;
      existing.warning_count += String(check.status || "") === "ok" ? 0 : 1;
      existing.max_duration_ms = Math.max(existing.max_duration_ms, Number(check.duration_ms || 0));
      if (!existing.latest_run_id) {
        existing.latest_status = String(check.status || "unknown");
        existing.latest_run_id = String(row.run_id || "");
      }
      existing.average_score = existing.average_score + (Number(check.score || 0) - existing.average_score) / existing.run_count;
      grouped.set(key, existing);
    }
  }
  const hotspots = Array.from(grouped.values())
    .filter((item) => item.warning_count > 0 || item.max_duration_ms > 30000)
    .sort((a, b) => (b.warning_count - a.warning_count) || (b.max_duration_ms - a.max_duration_ms))
    .slice(0, 20);
  return { ok: true, hotspots, next_actions: hotspots.length ? ["Open a hotspot detail in the Mac app for LLM-assisted diagnosis."] : ["No recurring hotspots in the requested window."] };
}

async function comparePayload(sql: NeonQueryFunction<false, false>, session: AccountSession, url: URL): Promise<Record<string, unknown>> {
  const runs = await historyPayload(sql, session, url, 2);
  const rows = Array.isArray(runs.runs) ? runs.runs as Array<Record<string, unknown>> : [];
  if (rows.length < 2) return { ok: true, mode: "self-history", comparison: null, next_actions: ["Sync at least two runs to compare history."] };
  const latest = rows[0];
  const previous = rows[1];
  const latestWarnings = Number(latest.warning_count || 0);
  const previousWarnings = Number(previous.warning_count || 0);
  const delta = Number(latest.health_score || 0) - Number(previous.health_score || 0);
  return {
    ok: true,
    mode: "self-history",
    comparison: {
      latest_run_id: latest.run_id,
      previous_run_id: previous.run_id,
      health_delta: delta,
      warning_delta: latestWarnings - previousWarnings,
      latest_status: latest.status,
      previous_status: previous.status
    },
    hidden_modes: ["team", "anonymous_benchmark"],
    next_actions: delta < 0 ? ["Review worsening checks in baseline_hotspots."] : ["No score regression in the latest self-history comparison."]
  };
}

async function authenticated(
  request: Request,
  env: CloudEnv,
  mcp = false
): Promise<{ sql: NeonQueryFunction<false, false>; session: AccountSession } | Response> {
  const token = bearerToken(request) || cookieToken(request);
  if (!token) return authChallenge(env, request, mcp);
  const sql = sqlOrResponse(env);
  if (sql instanceof Response) return sql;
  if (!env.MAGIC_LINK_SECRET) return json({ ok: false, error: "MAGIC_LINK_SECRET is not configured" }, 503);
  await ensureCloudSchema(sql);
  const tokenHash = await hmacHex(env.MAGIC_LINK_SECRET, token);
  const rows = await sql`
    select s.id as session_id, s.user_id, s.account_id, u.email_normalized, m.role, a.status as account_status, a.plan_key
    from account_sessions s
    join users u on u.id = s.user_id
    join accounts a on a.id = s.account_id
    join account_members m on m.account_id = s.account_id and m.user_id = s.user_id
    where s.token_hash = ${tokenHash}
      and s.revoked_at is null
      and s.expires_at > now()
      and m.status = 'active'
    limit 1
  `;
  if (!rows.length) return authChallenge(env, request, mcp);
  const row = rows[0] as Record<string, unknown>;
  await sql`update account_sessions set last_seen_at = now() where id = ${String(row.session_id)}`;
  return {
    sql,
    session: {
      session_id: String(row.session_id),
      user_id: String(row.user_id),
      account_id: String(row.account_id),
      email_normalized: String(row.email_normalized),
      role: String(row.role || "member"),
      account_status: String(row.account_status || "pending"),
      plan_key: String(row.plan_key || "free")
    }
  };
}

async function ensureAccountForEmail(sql: NeonQueryFunction<false, false>, email: string, plan: string): Promise<{ userId: string; accountId: string }> {
  const userID = crypto.randomUUID();
  const userRows = await sql`
    insert into users (id, email, email_normalized, updated_at)
    values (${userID}, ${email}, ${email}, now())
    on conflict (email_normalized) do update set email = excluded.email, updated_at = now()
    returning id
  `;
  const userId = String((userRows[0] as Record<string, unknown>).id);
  const accountRows = await sql`
    select a.id
    from accounts a
    join account_members m on m.account_id = a.id
    where m.user_id = ${userId}
    order by a.created_at asc
    limit 1
  `;
  if (accountRows.length) return { userId, accountId: String((accountRows[0] as Record<string, unknown>).id) };
  const accountId = crypto.randomUUID();
  await sql`
    insert into accounts (id, primary_user_id, billing_email, plan_key, status)
    values (${accountId}, ${userId}, ${email}, ${plan}, 'invited')
  `;
  await sql`
    insert into account_members (id, account_id, user_id, role, status)
    values (${crypto.randomUUID()}, ${accountId}, ${userId}, 'owner', 'active')
    on conflict (account_id, user_id) do update set status = 'active', updated_at = now()
  `;
  return { userId, accountId };
}

async function createMagicLink(
  sql: NeonQueryFunction<false, false>,
  env: CloudEnv,
  userID: string,
  accountID: string,
  email: string,
  purpose: string
): Promise<{ url: string; token: string }> {
  if (!env.MAGIC_LINK_SECRET) throw new Error("MAGIC_LINK_SECRET is not configured");
  const token = randomToken("blm_");
  const tokenHash = await hmacHex(env.MAGIC_LINK_SECRET, token);
  await sql`
    insert into magic_links (id, user_id, account_id, email_normalized, token_prefix, token_hash, purpose, expires_at)
    values (${crypto.randomUUID()}, ${userID}, ${accountID}, ${email}, ${token.slice(0, 14)}, ${tokenHash}, ${purpose}, ${futureISO(1)})
  `;
  return { token, url: baseURL(env) + "/api/auth/consume?token=" + encodeURIComponent(token) };
}

async function activeEntitlement(sql: NeonQueryFunction<false, false>, accountID: string): Promise<Record<string, unknown> | null> {
  const rows = await sql`
    select key, status, source, retention_days, max_workspaces, monitoring_enabled, starts_at, expires_at, updated_at
    from entitlements
    where account_id = ${accountID}
      and key = ${PRO_ENTITLEMENT}
      and status in ('active', 'trialing', 'pilot', 'past_due')
      and (expires_at is null or expires_at > now())
    order by updated_at desc
    limit 1
  `;
  return rows.length ? rows[0] as Record<string, unknown> : null;
}

async function grantEntitlement(
  sql: NeonQueryFunction<false, false>,
  accountID: string,
  status: string,
  source: string,
  plan: string,
  expiresAt: string | null
): Promise<void> {
  const monitoringEnabled = ["active", "trialing", "pilot", "past_due"].includes(status);
  const retentionDays = Number(source.startsWith("stripe") ? 365 : 90);
  await sql`
    insert into entitlements (id, account_id, key, status, source, retention_days, max_workspaces, monitoring_enabled, starts_at, expires_at, updated_at)
    values (${crypto.randomUUID()}, ${accountID}, ${PRO_ENTITLEMENT}, ${status}, ${source}, ${retentionDays}, ${plan === "team" ? 10 : 3}, ${monitoringEnabled}, now(), ${expiresAt}, now())
    on conflict (account_id, key) do update set
      status = excluded.status,
      source = excluded.source,
      retention_days = excluded.retention_days,
      max_workspaces = excluded.max_workspaces,
      monitoring_enabled = excluded.monitoring_enabled,
      expires_at = excluded.expires_at,
      updated_at = now()
  `;
  await sql`update accounts set status = ${status}, plan_key = ${plan}, updated_at = now() where id = ${accountID}`;
}

async function processStripeEvent(sql: NeonQueryFunction<false, false>, event: Record<string, unknown>, env: CloudEnv, ctx?: ExecutionContext): Promise<void> {
  const type = String(event.type || "");
  const object = stripeObject(event);
  if (type === "checkout.session.completed") {
    const metadata = object.metadata && typeof object.metadata === "object" ? object.metadata as Record<string, unknown> : {};
    let accountID = String(metadata.account_id || "");
    const plan = String(metadata.plan_key || metadata.plan || "pro");
    const customerID = String(object.customer || "");
    const email = checkoutObjectEmail(object);
    if (!accountID && customerID) {
      const customers = await sql`select account_id from stripe_customers where stripe_customer_id = ${customerID} limit 1`;
      accountID = customers.length ? String((customers[0] as Record<string, unknown>).account_id) : "";
    }
    if (!accountID && email) {
      const account = await ensureAccountForEmail(sql, email, plan === "team" ? "team" : "pro");
      accountID = account.accountId;
      if (customerID) {
        await sql`
          insert into stripe_customers (id, account_id, user_id, stripe_customer_id, email_normalized, livemode)
          values (${crypto.randomUUID()}, ${account.accountId}, ${account.userId}, ${customerID}, ${email}, ${Boolean(event.livemode)})
          on conflict (stripe_customer_id) do update set account_id = excluded.account_id, user_id = excluded.user_id, email_normalized = excluded.email_normalized, updated_at = now()
        `;
      }
    }
    if (!accountID) return;
    const couponCode = String(metadata.coupon_code || "");
    const entitlementSource = couponCode ? "stripe_founder" : "stripe";
    await grantEntitlement(sql, accountID, "active", entitlementSource, plan === "team" ? "team" : "pro", null);
    await audit(sql, "stripe", String(event.id || ""), "checkout.completed", "account", accountID, String(object.id || ""), { plan, coupon_code: couponCode || undefined, entitlement_source: entitlementSource });
    let checkoutMagicLink = "";
    if (email && env.MAGIC_LINK_SECRET) {
      const accountRows = await sql`select primary_user_id from accounts where id = ${accountID} limit 1`;
      const userID = accountRows.length ? String((accountRows[0] as Record<string, unknown>).primary_user_id) : "";
      if (userID) {
        const link = await createMagicLink(sql, env, userID, accountID, email, "checkout");
        checkoutMagicLink = link.url;
        const magicQueued = await queueAndDispatchLifecycleEvent(sql, env, ctx, "klaviyo", "Baseline Magic Link", "account", accountID, "customer", {
          purpose: "checkout",
          stripe_event_id: String(event.id || ""),
          account_id: accountID,
          plan,
          coupon_present: Boolean(couponCode),
          coupon_code: couponCode || ""
        }, link.url, true);
        if (!magicQueued) throw new Error("checkout magic-link delivery failed");
      }
    }
    await queueAndDispatchLifecycleEvent(sql, env, ctx, "klaviyo", "Baseline Subscription Started", "account", accountID, "customer", {
      plan,
      coupon_code: couponCode || "",
      coupon_present: Boolean(couponCode),
      magic_link_created: Boolean(checkoutMagicLink),
      stripe_event_id: String(event.id || ""),
      stripe_event_type: type
    }, checkoutMagicLink);
    return;
  }
  if (type === "invoice.payment_failed") {
    await handlePaymentFailed(sql, env, ctx, object, String(event.id || ""));
    return;
  }
  if (type.startsWith("customer.subscription.")) {
    await syncSubscription(sql, env, ctx, object, String(event.id || ""));
  }
}

async function handlePaymentFailed(sql: NeonQueryFunction<false, false>, env: CloudEnv, ctx: ExecutionContext | undefined, invoice: Record<string, unknown>, eventID: string): Promise<void> {
  const subscriptionID = String(invoice.subscription || "");
  const customerID = String(invoice.customer || "");
  let accountID = "";
  let plan = "pro";
  if (subscriptionID) {
    const rows = await sql`select account_id, plan_key from stripe_subscriptions where stripe_subscription_id = ${subscriptionID} limit 1`;
    if (rows.length) {
      accountID = String((rows[0] as Record<string, unknown>).account_id);
      plan = String((rows[0] as Record<string, unknown>).plan_key || "pro");
      await sql`update stripe_subscriptions set status = 'past_due', updated_at = now() where stripe_subscription_id = ${subscriptionID}`;
    }
  }
  if (!accountID && customerID) {
    const customers = await sql`select account_id from stripe_customers where stripe_customer_id = ${customerID} limit 1`;
    accountID = customers.length ? String((customers[0] as Record<string, unknown>).account_id) : "";
  }
  if (!accountID) return;
  await grantEntitlement(sql, accountID, "past_due", "stripe_dunning", plan, null);
  await audit(sql, "stripe", eventID, "invoice.payment_failed", "account", accountID, subscriptionID || customerID, {
    amount_due: invoice.amount_due || null,
    attempt_count: invoice.attempt_count || null
  });
  await queueAndDispatchLifecycleEvent(sql, env, ctx, "klaviyo", "Baseline Payment Failed", "account", accountID, "customer", {
    stripe_event_id: eventID,
    subscription_id: subscriptionID,
    amount_due: invoice.amount_due || null,
    attempt_count: invoice.attempt_count || null
  });
}

async function syncSubscription(sql: NeonQueryFunction<false, false>, env: CloudEnv, ctx: ExecutionContext | undefined, subscription: Record<string, unknown>, eventID: string): Promise<void> {
  const metadata = subscription.metadata && typeof subscription.metadata === "object" ? subscription.metadata as Record<string, unknown> : {};
  let accountID = String(metadata.account_id || "");
  const customerID = String(subscription.customer || "");
  if (!accountID && customerID) {
    const customers = await sql`select account_id from stripe_customers where stripe_customer_id = ${customerID} limit 1`;
    accountID = customers.length ? String((customers[0] as Record<string, unknown>).account_id) : "";
  }
  if (!accountID) return;
  const status = String(subscription.status || "unknown");
  const plan = String(metadata.plan_key || metadata.plan || "pro") === "team" ? "team" : "pro";
  const couponCode = String(metadata.coupon_code || "");
  const entitlementSource = couponCode || String(metadata.discount_kind || "") === "founder_100" ? "stripe_founder" : "stripe";
  const priceID = firstSubscriptionPrice(subscription);
  const periodEnd = Number(subscription.current_period_end || 0) > 0 ? new Date(Number(subscription.current_period_end) * 1000).toISOString() : null;
  await sql`
    insert into stripe_subscriptions (id, account_id, stripe_customer_id, stripe_subscription_id, status, price_id, plan_key, current_period_end, cancel_at_period_end, updated_at)
    values (${crypto.randomUUID()}, ${accountID}, ${customerID}, ${String(subscription.id || "")}, ${status}, ${priceID}, ${plan}, ${periodEnd}, ${Boolean(subscription.cancel_at_period_end)}, now())
    on conflict (stripe_subscription_id) do update set
      status = excluded.status,
      price_id = excluded.price_id,
      plan_key = excluded.plan_key,
      current_period_end = excluded.current_period_end,
      cancel_at_period_end = excluded.cancel_at_period_end,
      updated_at = now()
  `;
  await grantEntitlement(sql, accountID, status, entitlementSource, plan, ["canceled", "unpaid", "incomplete_expired"].includes(status) ? new Date().toISOString() : periodEnd);
  await audit(sql, "stripe", eventID, "subscription." + status, "account", accountID, String(subscription.id || ""), { price_id: priceID, entitlement_source: entitlementSource, coupon_code: couponCode || undefined });
  await queueAndDispatchLifecycleEvent(sql, env, ctx, "klaviyo", "Baseline Subscription Updated", "account", accountID, "customer", {
    stripe_event_id: eventID,
    status,
    plan,
    coupon_code: couponCode || "",
    coupon_present: Boolean(couponCode),
    cancel_at_period_end: Boolean(subscription.cancel_at_period_end)
  });
}

async function queueLifecycleEvent(
  sql: NeonQueryFunction<false, false>,
  provider: string,
  eventName: string,
  subjectType: string,
  subjectID: string,
  destination: string,
  payload: Record<string, unknown>
): Promise<{ id: string; idempotencyKey: string } | null> {
  const idempotencyKey = `${provider}:${eventName}:${subjectType}:${subjectID}:${payload.stripe_event_id || payload.subscription_id || "manual"}`;
  const id = crypto.randomUUID();
  const rows = await sql`
    insert into lifecycle_event_outbox (id, provider, event_name, subject_type, subject_id, destination, payload_redacted, idempotency_key)
    values (${id}, ${provider}, ${eventName}, ${subjectType}, ${subjectID}, ${destination}, ${JSON.stringify(payload)}::jsonb, ${idempotencyKey})
    on conflict (idempotency_key) do update set
      payload_redacted = excluded.payload_redacted,
      status = 'pending',
      next_attempt_at = now(),
      last_error = null,
      updated_at = now()
    where lifecycle_event_outbox.status <> 'sent'
    returning id
  `;
  return rows.length ? { id: String((rows[0] as Record<string, unknown>).id), idempotencyKey } : null;
}

async function queueAndDispatchLifecycleEvent(
  sql: NeonQueryFunction<false, false>,
  env: CloudEnv,
  ctx: ExecutionContext | undefined,
  provider: string,
  eventName: string,
  subjectType: string,
  subjectID: string,
  destination: string,
  payload: Record<string, unknown>,
  magicLink = "",
  waitForDelivery = false
): Promise<boolean> {
  const queued = await queueLifecycleEvent(sql, provider, eventName, subjectType, subjectID, destination, payload);
  if (!queued) return true;
  const delivery = dispatchLifecycleEvent(sql, env, queued.idempotencyKey, eventName, subjectType, subjectID, destination, payload, magicLink);
  if (waitForDelivery) return await delivery;
  if (ctx) ctx.waitUntil(delivery); else await delivery;
  return true;
}

async function dispatchLifecycleEvent(
  sql: NeonQueryFunction<false, false>,
  env: CloudEnv,
  idempotencyKey: string,
  eventName: string,
  subjectType: string,
  subjectID: string,
  destination: string,
  payload: Record<string, unknown>,
  magicLink = ""
): Promise<boolean> {
  try {
    const email = await lifecycleDestinationEmail(sql, env, subjectType, subjectID, destination);
    if (!email) throw new Error("lifecycle destination email unavailable");
    const result = await sendKlaviyoAuthEvent(env, email, eventName, magicLink, payload);
    if (!result.configured) throw new Error("KLAVIYO_PRIVATE_API_KEY is not configured");
    if (!result.accepted) throw new Error(result.error || `Klaviyo returned ${result.status || "unknown status"}`);
    await sql`
      update lifecycle_event_outbox
      set status = 'sent', attempts = attempts + 1, last_error = null, updated_at = now()
      where idempotency_key = ${idempotencyKey}
    `;
    return true;
  } catch (error) {
    const message = error instanceof Error ? error.message.slice(0, 500) : "unknown lifecycle dispatch error";
    await sql`
      update lifecycle_event_outbox
      set status = 'failed', attempts = attempts + 1, last_error = ${message}, next_attempt_at = now() + interval '5 minutes', updated_at = now()
      where idempotency_key = ${idempotencyKey}
    `;
    return false;
  }
}

async function lifecycleDestinationEmail(
  sql: NeonQueryFunction<false, false>,
  env: CloudEnv,
  subjectType: string,
  subjectID: string,
  destination: string
): Promise<string> {
  if (destination === "master") return normalizeEmail(env.BASELINE_MASTER_EMAIL || "") || "";
  if (destination !== "customer" || subjectType !== "account") return "";
  const rows = await sql`
    select coalesce(u.email, a.billing_email) as email
    from accounts a
    left join users u on u.id = a.primary_user_id
    where a.id = ${subjectID}
    limit 1
  `;
  return rows.length ? normalizeEmail(String((rows[0] as Record<string, unknown>).email || "")) || "" : "";
}

async function createBillingPortal(
  sql: NeonQueryFunction<false, false>,
  env: CloudEnv,
  session: AccountSession,
  returnURL: string
): Promise<{ url: string } | Response> {
  if (!env.STRIPE_SECRET_KEY) return json({ ok: false, error: "STRIPE_SECRET_KEY is not configured" }, 503);
  const customer = await sql`select stripe_customer_id from stripe_customers where account_id = ${session.account_id} order by created_at desc limit 1`;
  if (!customer.length) return json({ ok: false, error: "No Stripe customer found for account" }, 404);
  const body = new URLSearchParams({
    customer: String((customer[0] as Record<string, unknown>).stripe_customer_id),
    return_url: returnURL
  });
  const response = await fetch("https://api.stripe.com/v1/billing_portal/sessions", {
    method: "POST",
    headers: {
      authorization: "Bearer " + env.STRIPE_SECRET_KEY,
      "content-type": "application/x-www-form-urlencoded"
    },
    body
  });
  const payload = await response.json<Record<string, unknown>>();
  if (!response.ok || typeof payload.url !== "string") return json({ ok: false, error: stripeError(payload, "Stripe portal failed") }, 502);
  await audit(sql, "user", session.user_id, "billing.portal_created", "account", session.account_id, undefined, {});
  return { url: payload.url };
}

async function audit(
  sql: NeonQueryFunction<false, false>,
  actorType: string,
  actorID: string | undefined,
  action: string,
  subjectType: string,
  subjectID: string | undefined,
  idempotencyKey: string | undefined,
  metadata: Record<string, unknown>
): Promise<void> {
  await sql`
    insert into audit_log (id, actor_type, actor_id, action, subject_type, subject_id, idempotency_key, metadata_redacted)
    values (${crypto.randomUUID()}, ${actorType}, ${actorID || null}, ${action}, ${subjectType}, ${subjectID || null}, ${idempotencyKey || null}, ${JSON.stringify(metadata)}::jsonb)
  `;
}

async function sendKlaviyoAuthEvent(env: CloudEnv, email: string, metric: string, magicLink: string, properties: Record<string, unknown>): Promise<KlaviyoEventResult> {
  if (!env.KLAVIYO_PRIVATE_API_KEY) return { configured: false, accepted: false, error: "KLAVIYO_PRIVATE_API_KEY is not configured" };
  const body = {
    data: {
      type: "event",
      attributes: {
        metric: { data: { type: "metric", attributes: { name: metric } } },
        profile: { data: { type: "profile", attributes: { email } } },
        properties: { ...properties, magic_link: magicLink || undefined, product_name: "Baseline Pro Monitoring" },
        time: new Date().toISOString(),
        unique_id: crypto.randomUUID()
      }
    }
  };
  try {
    const response = await fetch("https://a.klaviyo.com/api/events", {
      method: "POST",
      headers: {
        authorization: "Klaviyo-API-Key " + env.KLAVIYO_PRIVATE_API_KEY,
        "content-type": "application/vnd.api+json",
        accept: "application/vnd.api+json",
        revision: env.KLAVIYO_REVISION || "2026-04-15"
      },
      body: JSON.stringify(body)
    });
    if (!response.ok) {
      const detail = await response.text();
      return { configured: true, accepted: false, status: response.status, error: detail.slice(0, 500) };
    }
    return { configured: true, accepted: true, status: response.status };
  } catch (error) {
    return { configured: true, accepted: false, error: error instanceof Error ? error.message : "Klaviyo request failed" };
  }
}

function sqlOrResponse(env: CloudEnv): NeonQueryFunction<false, false> | Response {
  if (!env.DATABASE_URL) return json({ ok: false, error: "DATABASE_URL is not configured" }, 503);
  return neon(env.DATABASE_URL);
}

function requireCloudAdmin(request: Request, env: CloudEnv): Response | null {
  if (!env.BASELINE_ADMIN_TOKEN) return json({ ok: false, error: "BASELINE_ADMIN_TOKEN is not configured" }, 503);
  const token = bearerToken(request) || new URL(request.url).searchParams.get("token") || "";
  if (token !== env.BASELINE_ADMIN_TOKEN) return json({ ok: false, error: "invalid admin token" }, 401);
  return null;
}

function authChallenge(env: CloudEnv, request: Request, mcp: boolean): Response {
  const resource = baseURL(env, request) + "/.well-known/oauth-protected-resource";
  const headers = {
    "www-authenticate": `Bearer resource_metadata="${resource}"`,
    "content-type": "application/json; charset=utf-8"
  };
  return new Response(JSON.stringify({
    ok: false,
    error: "authentication_required",
    authorization: baseURL(env, request) + "/api/auth/magic-link",
    resource_metadata: resource,
    next_actions: ["Request a magic link, consume it for a session token, then retry with Authorization: Bearer <session_token>."]
  }), { status: mcp ? 401 : 401, headers });
}

function oauthProtectedResourceMetadata(env: CloudEnv, request: Request): Response {
  const origin = baseURL(env, request);
  return json({
    resource: origin + "/mcp",
    authorization_servers: [origin + "/.well-known/oauth-authorization-server"],
    bearer_methods_supported: ["header"],
    scopes_supported: ["baseline.account", "baseline.history", "baseline.workspaces"]
  });
}

function oauthAuthorizationMetadata(env: CloudEnv, request: Request): Response {
  const origin = baseURL(env, request);
  return json({
    issuer: origin,
    authorization_endpoint: origin + "/api/auth/magic-link",
    token_endpoint: origin + "/api/auth/consume",
    response_types_supported: ["magic_link"],
    grant_types_supported: ["urn:baseline:magic-link"],
    token_endpoint_auth_methods_supported: ["none"]
  });
}

function normalizeRun(row: Record<string, unknown>): Record<string, unknown> {
  const payload = parseObject(row.account_private_payload) || parseObject(row.payload) || {};
  const checks = Array.isArray(payload.checks) ? payload.checks : [];
  const warningCount = checks.filter((check) => check && typeof check === "object" && (check as Record<string, unknown>).status !== "ok").length;
  return {
    run_id: String(row.id || payload.run_id || ""),
    created_at: row.created_at || payload.started_at || "",
    started_at: payload.started_at || row.created_at || "",
    workspace: String(row.workspace || payload.workspace || "unknown"),
    workspace_hash: String(payload.workspace_hash || ""),
    agent_kind: String(row.agent_kind || payload.agent_kind || "unknown"),
    status: String(row.status || payload.status || "unknown"),
    health_score: Number(row.health_score || payload.health_score || 0),
    duration_ms: Number(payload.duration_ms || 0),
    mode: String(row.mode || payload.mode || "unknown"),
    redaction_status: String(payload.redaction_status || "unknown"),
    warning_count: warningCount,
    checks
  };
}

function parseObject(value: unknown): Record<string, unknown> | null {
  if (!value) return null;
  if (typeof value === "string") {
    try {
      const parsed = JSON.parse(value);
      return parsed && typeof parsed === "object" ? parsed as Record<string, unknown> : null;
    } catch {
      return null;
    }
  }
  return typeof value === "object" ? value as Record<string, unknown> : null;
}

function stripeObject(event: Record<string, unknown>): Record<string, unknown> {
  const data = event.data && typeof event.data === "object" ? event.data as Record<string, unknown> : {};
  return data.object && typeof data.object === "object" ? data.object as Record<string, unknown> : {};
}

function checkoutObjectEmail(object: Record<string, unknown>): string {
  const details = object.customer_details && typeof object.customer_details === "object" ? object.customer_details as Record<string, unknown> : {};
  return normalizeEmail(String(object.customer_email || object.email || details.email || "")) || "";
}

function firstSubscriptionPrice(subscription: Record<string, unknown>): string {
  const items = subscription.items && typeof subscription.items === "object" ? subscription.items as Record<string, unknown> : {};
  const data = Array.isArray(items.data) ? items.data as Array<Record<string, unknown>> : [];
  const first = data[0] || {};
  const price = first.price && typeof first.price === "object" ? first.price as Record<string, unknown> : {};
  return String(price.id || "");
}

function canCreateWorkspaceToken(entitlement: Record<string, unknown>): boolean {
  const status = String(entitlement.status || "");
  return ["active", "trialing", "pilot"].includes(status);
}

async function verifyStripeSignature(rawBody: string, signature: string, secret: string): Promise<boolean> {
  const parts = Object.fromEntries(signature.split(",").map((part) => {
    const [key, value] = part.split("=");
    return [key, value];
  }));
  const timestamp = parts.t || "";
  const expected = parts.v1 || "";
  if (!timestamp || !expected) return false;
  const signedPayload = timestamp + "." + rawBody;
  const digest = await hmacHex(secret, signedPayload);
  return timingSafeEqualHex(digest, expected);
}

async function hmacHex(secret: string, value: string): Promise<string> {
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey("raw", encoder.encode(secret), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  const signature = await crypto.subtle.sign("HMAC", key, encoder.encode(value));
  return hex(signature);
}

async function sha256Hex(value: string): Promise<string> {
  return hex(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(value)));
}

function hex(buffer: ArrayBuffer): string {
  return Array.from(new Uint8Array(buffer)).map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

function timingSafeEqualHex(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let result = 0;
  for (let i = 0; i < a.length; i++) result |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return result === 0;
}

function randomToken(prefix: string): string {
  const bytes = new Uint8Array(30);
  crypto.getRandomValues(bytes);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return prefix + btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function normalizeEmail(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined;
  const trimmed = value.trim().toLowerCase();
  if (!trimmed || trimmed.length > 254) return undefined;
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)) return undefined;
  return trimmed;
}

function cleanToken(value: unknown, max: number): string {
  if (typeof value !== "string") return "";
  return value.trim().slice(0, max);
}

function boundedLimit(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, Math.round(value)));
}

function futureISO(days: number): string {
  return new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString();
}

function bearerToken(request: Request): string {
  const auth = request.headers.get("authorization") || "";
  return auth.startsWith("Bearer ") ? auth.slice("Bearer ".length) : "";
}

function cookieToken(request: Request): string {
  const cookie = request.headers.get("cookie") || "";
  const match = cookie.match(/(?:^|;\s*)baseline_session=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : "";
}

function sessionCookie(token: string, days: number): string {
  const maxAge = days * 24 * 60 * 60;
  return `baseline_session=${encodeURIComponent(token)}; Max-Age=${maxAge}; Path=/; HttpOnly; Secure; SameSite=Lax`;
}

async function safeJSON<T>(request: Request): Promise<T> {
  try {
    return await request.json<T>();
  } catch {
    return {} as T;
  }
}

function stripeError(payload: Record<string, unknown>, fallback: string): string {
  const error = payload.error && typeof payload.error === "object" ? payload.error as Record<string, unknown> : {};
  return String(error.message || fallback);
}

function asRequest(request: Request, method: string, path: string): Request {
  const url = new URL(request.url);
  url.pathname = path;
  url.search = "";
  return new Request(url.toString(), { method, headers: request.headers });
}

function jsonRequest(request: Request, path: string, payload: Record<string, unknown>): Request {
  const url = new URL(request.url);
  url.pathname = path;
  url.search = "";
  return new Request(url.toString(), {
    method: "POST",
    headers: { authorization: request.headers.get("authorization") || "", "content-type": "application/json" },
    body: JSON.stringify(payload)
  });
}

function mcpResult(id: unknown, result: Record<string, unknown>): Response {
  return json({ jsonrpc: "2.0", id: id ?? null, result });
}

function mcpError(id: unknown, code: number, message: string): Response {
  return json({ jsonrpc: "2.0", id: id ?? null, error: { code, message } }, 200);
}

function json(payload: unknown, status = 200, extraHeaders?: Record<string, string>): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "content-type": "application/json; charset=utf-8", ...(extraHeaders || {}) }
  });
}

function baseURL(env: CloudEnv, request?: Request): string {
  if (env.APP_URL) return env.APP_URL.replace(/\/$/, "");
  if (request) return new URL(request.url).origin;
  return "https://trackbaseline.com";
}
