import { neon, type NeonQueryFunction } from "@neondatabase/serverless";
import {
  appendCheckoutMetadata,
  ensureCloudSchema,
  handleCloudRoute,
  prepareCheckoutAccount,
  recordRunAggregates,
  resolveRunIngestContext
} from "./cloud";

interface Env {
  DATABASE_URL?: string;
  STRIPE_SECRET_KEY?: string;
  STRIPE_WEBHOOK_SECRET?: string;
  STRIPE_PRICE_ID_PRO?: string;
  STRIPE_PRICE_ID_TEAM?: string;
  STRIPE_PAYMENT_LINK_PRO?: string;
  STRIPE_PAYMENT_LINK_TEAM?: string;
  KLAVIYO_PRIVATE_API_KEY?: string;
  KLAVIYO_REVISION?: string;
  BASELINE_MASTER_EMAIL?: string;
  APP_URL?: string;
  BASELINE_API_TOKEN?: string;
  BASELINE_ADMIN_TOKEN?: string;
  MAGIC_LINK_SECRET?: string;
  TOKEN_HMAC_SECRET?: string;
  MAGIC_LINK_DEV_ECHO?: string;
  PRO_RETENTION_DAYS?: string;
  FREE_RETENTION_DAYS?: string;
  OPENAI_API_KEY?: string;
  OPENAI_EVALUATOR_MODEL?: string;
}

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

type CanonicalQuestionSet = {
  slug: string;
  version: string;
  title: string;
  questions: Array<{
    id: string;
    prompt: string;
    dimension: string;
    expected_facts?: string[];
    required?: boolean;
  }>;
};

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const read = request.method === "GET" || request.method === "HEAD";
    try {
      const cloudResponse = await handleCloudRoute(request, env, ctx);
      if (cloudResponse) return cloudResponse;
      if (read && url.pathname === "/") return html(landingPage(env));
      if (read && url.pathname === "/dashboard") return html(dashboardPage(env));
      if (read && url.pathname === "/admin") return html(adminPage(env));
      if (read && url.pathname === "/docs/mcp") return html(mcpDocsPage(env));
      if (read && url.pathname === "/blog") return html(blogPage(env));
      if (read && url.pathname === "/checkout/success") return html(checkoutSuccessPage(env));
      if (read && url.pathname === "/checkout/cancel") return html(checkoutCancelPage(env));
      if (read && url.pathname === "/privacy") return html(privacyPage(env));
      if (read && url.pathname === "/terms") return html(termsPage(env));
      if (read && url.pathname === "/robots.txt") return text("User-agent: *\nAllow: /\nSitemap: " + baseURL(env, request) + "/sitemap.xml\n");
      if (read && url.pathname === "/sitemap.xml") return text(sitemap(baseURL(env, request)), "application/xml");
      if (read && url.pathname === "/api/health") return json({ ok: true, db: Boolean(env.DATABASE_URL), stripe: hasStripe(env), token_required: Boolean(env.BASELINE_API_TOKEN), lifecycle_email: Boolean(env.KLAVIYO_PRIVATE_API_KEY), pro_auth: Boolean(env.MAGIC_LINK_SECRET), pro_tokens: Boolean(env.TOKEN_HMAC_SECRET), stripe_webhook: Boolean(env.STRIPE_WEBHOOK_SECRET) });
      if (read && url.pathname === "/api/runs/latest") return latestRun(request, env);
      if (read && url.pathname === "/api/runs/timeline") return runTimeline(env);
      if (read && url.pathname === "/api/question-sets") return listQuestionSets(env, false);
      if (read && url.pathname === "/api/admin/question-sets") return listQuestionSets(env, true, request);
      if (request.method === "POST" && url.pathname === "/api/admin/question-sets") return upsertQuestionSet(request, env);
      if (request.method === "POST" && url.pathname === "/api/admin/evaluate") return evaluateRun(request, env);
      if (read && url.pathname === "/api/admin/evaluations") return listEvaluations(request, env);
      if (request.method === "POST" && url.pathname === "/api/runs") return ingestRun(request, env);
      if (request.method === "POST" && url.pathname === "/api/events") {
        ctx.waitUntil(recordEvent(request, env, url.pathname));
        return json({ ok: true });
      }
      if ((request.method === "GET" || request.method === "POST") && url.pathname === "/api/checkout") return checkout(request, env, ctx);
      return html(notFoundPage(env), 404);
    } catch (error) {
      return json({ ok: false, error: error instanceof Error ? error.message : "unknown error" }, 500);
    }
  }
};

async function latestRun(request: Request, env: Env): Promise<Response> {
  const sql = configuredSQL(env);
  if (!sql) return json({ ok: true, configured: false, run: demoRun(), origin: baseURL(env, request) });
  await ensureSchema(sql);
  const rows = await sql`select id, workspace, agent_kind, status, health_score, mode, payload, created_at from baseline_runs order by created_at desc limit 1`;
  const run = rows.length ? normalizeRun(rows[0] as Record<string, unknown>) : demoRun();
  return json({ ok: true, configured: true, run, origin: baseURL(env, request) });
}

async function runTimeline(env: Env): Promise<Response> {
  const sql = configuredSQL(env);
  if (!sql) return json({ ok: true, configured: false, runs: [demoRun()] });
  await ensureSchema(sql);
  const rows = await sql`select id, workspace, agent_kind, status, health_score, mode, payload, created_at from baseline_runs order by created_at desc limit 30`;
  return json({ ok: true, configured: true, runs: rows.map((row) => normalizeRun(row as Record<string, unknown>)) });
}

async function listQuestionSets(env: Env, admin: boolean, request?: Request): Promise<Response> {
  if (admin) {
    const auth = requireAdmin(request, env);
    if (auth) return auth;
  }
  const sql = configuredSQL(env);
  if (!sql) return json({ ok: true, configured: false, question_sets: [defaultQuestionSet()] });
  await ensureSchema(sql);
  await seedQuestionSets(sql);
  const rows = await sql`select slug, version, title, questions, active, created_at, updated_at from canonical_question_sets order by slug, created_at desc`;
  const sets = rows
    .map((row) => normalizeQuestionSet(row as Record<string, unknown>))
    .filter((set) => admin || set.active);
  return json({ ok: true, configured: true, question_sets: sets });
}

async function upsertQuestionSet(request: Request, env: Env): Promise<Response> {
  const auth = requireAdmin(request, env);
  if (auth) return auth;
  const sql = configuredSQL(env);
  if (!sql) return json({ ok: false, error: "DATABASE_URL is not configured" }, 503);
  const payload = await request.json<Partial<CanonicalQuestionSet> & { active?: boolean }>();
  const validation = validateQuestionSet(payload);
  if (validation) return json({ ok: false, error: validation }, 400);
  await ensureSchema(sql);
  await sql`
    insert into canonical_question_sets (slug, version, title, questions, active, updated_at)
    values (${payload.slug}, ${payload.version}, ${payload.title}, ${JSON.stringify(payload.questions)}::jsonb, ${payload.active !== false}, now())
    on conflict (slug, version) do update set
      title = excluded.title,
      questions = excluded.questions,
      active = excluded.active,
      updated_at = now()
  `;
  return json({ ok: true, question_set: payload });
}

async function evaluateRun(request: Request, env: Env): Promise<Response> {
  const auth = requireAdmin(request, env);
  if (auth) return auth;
  const sql = configuredSQL(env);
  if (!sql) return json({ ok: false, error: "DATABASE_URL is not configured" }, 503);
  await ensureSchema(sql);
  await seedQuestionSets(sql);
  const input = await request.json<{ run_id?: string; slug?: string; version?: string }>();
  const runRows = input.run_id
    ? await sql`select id, workspace, agent_kind, status, health_score, mode, payload, created_at from baseline_runs where id = ${input.run_id} limit 1`
    : await sql`select id, workspace, agent_kind, status, health_score, mode, payload, created_at from baseline_runs order by created_at desc limit 1`;
  if (!runRows.length) return json({ ok: false, error: "No baseline run found" }, 404);
  const run = normalizeRun(runRows[0] as Record<string, unknown>);
  const questionSet = await loadQuestionSet(sql, input.slug || "baseline-core", input.version);
  const model = env.OPENAI_EVALUATOR_MODEL || "local-heuristic";
  const evaluation = env.OPENAI_API_KEY
    ? await evaluateWithOpenAI(env, run, questionSet)
    : heuristicEvaluation(run, questionSet);
  const id = crypto.randomUUID();
  await sql`
    insert into llm_evaluations (id, run_id, question_set_slug, question_set_version, model, score, verdict, payload)
    values (${id}, ${String(run.run_id)}, ${questionSet.slug}, ${questionSet.version}, ${evaluation.model || model}, ${evaluation.score}, ${evaluation.verdict}, ${JSON.stringify(evaluation)}::jsonb)
  `;
  return json({ ok: true, evaluation_id: id, evaluation });
}

async function listEvaluations(request: Request, env: Env): Promise<Response> {
  const auth = requireAdmin(request, env);
  if (auth) return auth;
  const sql = configuredSQL(env);
  if (!sql) return json({ ok: true, configured: false, evaluations: [] });
  await ensureSchema(sql);
  const rows = await sql`select id, run_id, question_set_slug, question_set_version, model, score, verdict, payload, created_at from llm_evaluations order by created_at desc limit 50`;
  return json({ ok: true, configured: true, evaluations: rows });
}

async function ingestRun(request: Request, env: Env): Promise<Response> {
  const payload = await request.json<RunPayload>();
  if (!payload.run_id) return json({ ok: false, error: "run_id required" }, 400);
  if (!env.DATABASE_URL) return json({ ok: false, error: "DATABASE_URL is not configured" }, 503);
  const sql = neon(env.DATABASE_URL);
  await ensureSchema(sql);
  const ingestContext = await resolveRunIngestContext(request, env, sql);
  if (ingestContext instanceof Response) return ingestContext;
  const accountPrivatePayload = ingestContext.legacy ? null : JSON.stringify(payload);
  await sql`
    insert into baseline_runs (id, workspace, agent_kind, status, health_score, mode, payload, account_id, workspace_id, expires_at, account_private_payload, comparison_scope)
    values (
      ${payload.run_id}, ${payload.workspace || "unknown"}, ${payload.agent_kind || "unknown"}, ${payload.status || "unknown"},
      ${payload.health_score || 0}, ${payload.mode || "unknown"}, ${JSON.stringify(payload)}::jsonb,
      ${ingestContext.accountId}, ${ingestContext.workspaceId}, ${ingestContext.expiresAt}, ${accountPrivatePayload}::jsonb, ${ingestContext.comparisonScope}
    )
    on conflict (id) do update set
      workspace = excluded.workspace,
      agent_kind = excluded.agent_kind,
      status = excluded.status,
      health_score = excluded.health_score,
      mode = excluded.mode,
      payload = excluded.payload,
      account_id = excluded.account_id,
      workspace_id = excluded.workspace_id,
      expires_at = excluded.expires_at,
      account_private_payload = excluded.account_private_payload,
      comparison_scope = excluded.comparison_scope
  `;
  await recordRunAggregates(sql, payload, ingestContext);
  return json({ ok: true, account_scoped: !ingestContext.legacy });
}

function configuredSQL(env: Env): NeonQueryFunction<false, false> | null {
  if (!env.DATABASE_URL) return null;
  return neon(env.DATABASE_URL);
}

function normalizeRun(row: Record<string, unknown>): Record<string, unknown> {
  const payload = normalizePayload(row.payload);
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

function normalizePayload(value: unknown): Record<string, unknown> {
  if (!value) return {};
  if (typeof value === "string") {
    try {
      const parsed = JSON.parse(value);
      return parsed && typeof parsed === "object" ? parsed as Record<string, unknown> : {};
    } catch {
      return {};
    }
  }
  if (typeof value === "object") return value as Record<string, unknown>;
  return {};
}

function demoRun(): Record<string, unknown> {
  return {
    run_id: "demo_run",
    created_at: new Date().toISOString(),
    started_at: new Date().toISOString(),
    workspace: "sha256:demo",
    workspace_hash: "demo",
    agent_kind: "openclaw",
    status: "warning",
    health_score: 82,
    duration_ms: 13400,
    mode: "fast",
    redaction_status: "clean",
    warning_count: 2,
    checks: [
      { check_id: "mcp.openclaw.config", kind: "tooling", status: "warning", score: 82, duration_ms: 12 },
      { check_id: "safety.scrubber", kind: "safety", status: "ok", score: 100, duration_ms: 1 },
      { check_id: "latency.baseline_probe", kind: "latency", status: "ok", score: 100, duration_ms: 2 }
    ]
  };
}

function defaultQuestionSet(): CanonicalQuestionSet & { active?: boolean } {
  return {
    slug: "baseline-core",
    version: "0.1.0",
    title: "Baseline Core v0.1",
    active: true,
    questions: [
      { id: "model", prompt: "What is your current model and provider?", dimension: "runtime_identity", expected_facts: [], required: true },
      { id: "context_window", prompt: "What is your approximate context window or configured context limit?", dimension: "runtime_identity", expected_facts: [], required: true },
      { id: "date", prompt: "Answer only today's date in local time.", dimension: "basic_reasoning", expected_facts: [], required: true },
      { id: "identity", prompt: "Who are you in this environment?", dimension: "identity", expected_facts: [], required: true },
      { id: "primary_goal", prompt: "What is your primary goal when helping me?", dimension: "identity", expected_facts: [], required: true },
      { id: "tools", prompt: "What local tools and MCP servers can you currently use?", dimension: "tool_awareness", expected_facts: ["tool", "mcp"], required: true },
      { id: "workspace", prompt: "What workspace or repo are you operating in, and is it clean or dirty?", dimension: "repo_awareness", expected_facts: [], required: true },
      { id: "math", prompt: "Answer only the number: 2 + 2.", dimension: "basic_reasoning", expected_facts: ["4"], required: true },
      { id: "variance_1", prompt: "Answer only the word: baseline.", dimension: "latency_variance", expected_facts: ["baseline"], required: true },
      { id: "variance_2", prompt: "Answer only the word: baseline.", dimension: "latency_variance", expected_facts: ["baseline"], required: true },
      { id: "variance_3", prompt: "Answer only the word: baseline.", dimension: "latency_variance", expected_facts: ["baseline"], required: true },
      { id: "variance_4", prompt: "Answer only the word: baseline.", dimension: "latency_variance", expected_facts: ["baseline"], required: true },
      { id: "variance_5", prompt: "Answer only the word: baseline.", dimension: "latency_variance", expected_facts: ["baseline"], required: true },
      { id: "ops_change", prompt: "Report any obvious tool, MCP, repo, or config changes since the accepted Good baseline. If unknown, say unknown.", dimension: "change_awareness", expected_facts: [], required: true }
    ]
  };
}

async function seedQuestionSets(sql: NeonQueryFunction<false, false>): Promise<void> {
  const set = defaultQuestionSet();
  await sql`
    insert into canonical_question_sets (slug, version, title, questions, active)
    values (${set.slug}, ${set.version}, ${set.title}, ${JSON.stringify(set.questions)}::jsonb, true)
    on conflict (slug, version) do nothing
  `;
}

async function loadQuestionSet(sql: NeonQueryFunction<false, false>, slug: string, version?: string): Promise<CanonicalQuestionSet> {
  const rows = version
    ? await sql`select slug, version, title, questions, active from canonical_question_sets where slug = ${slug} and version = ${version} limit 1`
    : await sql`select slug, version, title, questions, active from canonical_question_sets where slug = ${slug} and active = true order by created_at desc limit 1`;
  if (!rows.length) return defaultQuestionSet();
  return normalizeQuestionSet(rows[0] as Record<string, unknown>) as CanonicalQuestionSet;
}

function normalizeQuestionSet(row: Record<string, unknown>): CanonicalQuestionSet & { active?: boolean; created_at?: unknown; updated_at?: unknown } {
  const questions = typeof row.questions === "string" ? JSON.parse(row.questions) : row.questions;
  return {
    slug: String(row.slug || "baseline-core"),
    version: String(row.version || "2026-05-14"),
    title: String(row.title || "Baseline Core"),
    active: row.active !== false,
    created_at: row.created_at,
    updated_at: row.updated_at,
    questions: Array.isArray(questions) ? questions : defaultQuestionSet().questions
  };
}

function validateQuestionSet(payload: Partial<CanonicalQuestionSet>): string {
  if (!payload.slug || !/^[a-z0-9][a-z0-9-]{1,62}$/.test(payload.slug)) return "slug must be kebab-case and 2-63 characters";
  if (!payload.version || payload.version.length > 64) return "version is required and must be <= 64 characters";
  if (!payload.title || payload.title.length > 120) return "title is required and must be <= 120 characters";
  if (!Array.isArray(payload.questions) || payload.questions.length < 3 || payload.questions.length > 30) return "questions must contain 3-30 items";
  for (const q of payload.questions) {
    if (!q.id || !q.prompt || !q.dimension) return "each question needs id, prompt, and dimension";
    if (q.prompt.length > 800) return "question prompts must be <= 800 characters";
  }
  return "";
}

function requireAdmin(request: Request | undefined, env: Env): Response | null {
  if (!env.BASELINE_ADMIN_TOKEN) return json({ ok: false, error: "BASELINE_ADMIN_TOKEN is not configured" }, 503);
  const auth = request?.headers.get("authorization") || "";
  const urlToken = request ? new URL(request.url).searchParams.get("token") || "" : "";
  const token = auth.startsWith("Bearer ") ? auth.slice("Bearer ".length) : urlToken;
  if (token !== env.BASELINE_ADMIN_TOKEN) return json({ ok: false, error: "invalid admin token" }, 401);
  return null;
}

type EvaluationPayload = {
  model: string;
  score: number;
  verdict: string;
  confidence: number;
  summary: string;
  concerns: string[];
  dimension_scores: Record<string, number>;
};

function heuristicEvaluation(run: Record<string, unknown>, questionSet: CanonicalQuestionSet): EvaluationPayload {
  const checks = Array.isArray(run.checks) ? run.checks as Array<Record<string, unknown>> : [];
  const dimensionScores: Record<string, number> = {};
  for (const q of questionSet.questions) {
    const matching = checks.filter((check) => String(check.kind || check.check_id || "").includes(q.dimension) || String(check.check_id || "").includes(q.id));
    const score = matching.length ? average(matching.map((check) => Number(check.score || 0))) : Number(run.health_score || 0);
    dimensionScores[q.dimension] = Math.round(score);
  }
  const score = Math.round(average(Object.values(dimensionScores)));
  const concerns = checks.filter((check) => check.status !== "ok").slice(0, 8).map((check) => String(check.check_id || check.kind || "check") + " is " + String(check.status || "unknown"));
  return {
    model: "local-heuristic",
    score,
    verdict: score >= 85 ? "pass" : score >= 70 ? "watch" : "fail",
    confidence: 0.62,
    summary: "Local evaluator scored the run from redacted check metadata because no OpenAI evaluator key is configured.",
    concerns,
    dimension_scores: dimensionScores
  };
}

async function evaluateWithOpenAI(env: Env, run: Record<string, unknown>, questionSet: CanonicalQuestionSet): Promise<EvaluationPayload> {
  const model = env.OPENAI_EVALUATOR_MODEL || "gpt-5";
  const prompt = {
    task: "Evaluate whether this coding-agent baseline run drifted against the canonical question set. Use only the redacted run payload. Return JSON.",
    question_set: questionSet,
    redacted_run: run
  };
  const response = await fetch("https://api.openai.com/v1/responses", {
    method: "POST",
    headers: {
      "authorization": "Bearer " + env.OPENAI_API_KEY,
      "content-type": "application/json"
    },
    body: JSON.stringify({
      model,
      input: JSON.stringify(prompt),
      text: {
        format: {
          type: "json_schema",
          name: "baseline_eval",
          strict: true,
          schema: {
            type: "object",
            additionalProperties: false,
            required: ["score", "verdict", "confidence", "summary", "concerns", "dimension_scores"],
            properties: {
              score: { type: "integer", minimum: 0, maximum: 100 },
              verdict: { type: "string", enum: ["pass", "watch", "fail"] },
              confidence: { type: "number", minimum: 0, maximum: 1 },
              summary: { type: "string" },
              concerns: { type: "array", items: { type: "string" } },
              dimension_scores: { type: "object", additionalProperties: { type: "integer", minimum: 0, maximum: 100 } }
            }
          }
        }
      }
    })
  });
  const body = await response.json<Record<string, unknown>>();
  if (!response.ok) throw new Error(String((body.error as Record<string, unknown> | undefined)?.message || "OpenAI evaluator failed"));
  const output = extractResponseText(body);
  const parsed = JSON.parse(output) as Omit<EvaluationPayload, "model">;
  return { ...parsed, model };
}

function extractResponseText(body: Record<string, unknown>): string {
  if (typeof body.output_text === "string") return body.output_text;
  const output = Array.isArray(body.output) ? body.output as Array<Record<string, unknown>> : [];
  for (const item of output) {
    const content = Array.isArray(item.content) ? item.content as Array<Record<string, unknown>> : [];
    for (const part of content) {
      if (typeof part.text === "string") return part.text;
    }
  }
  throw new Error("OpenAI evaluator returned no text");
}

function average(values: number[]): number {
  const clean = values.filter((value) => Number.isFinite(value));
  if (!clean.length) return 0;
  return clean.reduce((sum, value) => sum + value, 0) / clean.length;
}

async function recordEvent(request: Request, env: Env, path: string): Promise<void> {
  if (!env.DATABASE_URL) return;
  const sql = neon(env.DATABASE_URL);
  await ensureSchema(sql);
  let payload: Record<string, unknown> = {};
  try {
    payload = await request.json<Record<string, unknown>>();
  } catch {
    payload = {};
  }
  await sql`
    insert into baseline_events (id, type, path, payload)
    values (${crypto.randomUUID()}, ${String(payload.type || "event")}, ${String(payload.path || path)}, ${JSON.stringify(payload)}::jsonb)
  `;
}

async function checkout(request: Request, env: Env, ctx?: ExecutionContext): Promise<Response> {
  const url = new URL(request.url);
  const input = request.method === "POST" ? await safeCheckoutInput(request) : {};
  const plan = (input.plan || url.searchParams.get("plan")) === "team" ? "team" : "pro";
  const email = normalizeOptionalEmail(input.email);
  const price = plan === "team" ? env.STRIPE_PRICE_ID_TEAM : env.STRIPE_PRICE_ID_PRO;
  const paymentLink = plan === "team" ? env.STRIPE_PAYMENT_LINK_TEAM : env.STRIPE_PAYMENT_LINK_PRO;
  const canCreateCheckoutSession = Boolean(env.STRIPE_SECRET_KEY && price);
  if (paymentLink && !canCreateCheckoutSession) {
    if (request.method === "POST") {
      ctx?.waitUntil(emitCheckoutStartedEvents(env, email, plan, true));
      return json({ ok: true, url: paymentLink });
    }
    return Response.redirect(paymentLink, 303);
  }
  if (!env.STRIPE_SECRET_KEY) {
    return json({ ok: false, error: "Stripe is not configured. Set STRIPE_SECRET_KEY and STRIPE_PRICE_ID_PRO/TEAM or payment links." }, 503);
  }
  if (!price) return json({ ok: false, error: "Stripe price id is not configured for " + plan }, 503);
  const origin = baseURL(env, request);
  const body = new URLSearchParams({
    mode: "subscription",
    success_url: normalizeCheckoutReturn(input.successUrl, origin + "/checkout/success?session_id={CHECKOUT_SESSION_ID}", origin),
    cancel_url: normalizeCheckoutReturn(input.cancelUrl, origin + "/checkout/cancel", origin),
    "line_items[0][price]": price,
    "line_items[0][quantity]": "1",
    "metadata[plan]": plan,
    "metadata[source]": request.method === "POST" ? "landing_email_form" : "direct_link",
    "metadata[product_name]": "Baseline Pro Monitoring",
    "subscription_data[metadata][plan]": plan,
    "subscription_data[metadata][product_name]": "Baseline Pro Monitoring"
  });
  let checkoutAccount = null;
  if (email) {
    if (!env.DATABASE_URL) return json({ ok: false, error: "DATABASE_URL is required for account checkout" }, 503);
    checkoutAccount = await prepareCheckoutAccount(neon(env.DATABASE_URL), env, email, plan);
  }
  appendCheckoutMetadata(body, checkoutAccount, plan);
  if (email && !checkoutAccount) body.set("customer_email", email);
  const resp = await fetch("https://api.stripe.com/v1/checkout/sessions", {
    method: "POST",
    headers: {
      authorization: "Bearer " + env.STRIPE_SECRET_KEY,
      "content-type": "application/x-www-form-urlencoded"
    },
    body
  });
  const session = await resp.json<{ url?: string; error?: { message?: string } }>();
  if (!resp.ok || !session.url) return json({ ok: false, error: session.error?.message || "Stripe checkout failed" }, 502);
  ctx?.waitUntil(emitCheckoutStartedEvents(env, email, plan, false));
  if (request.method === "POST") return json({ ok: true, url: session.url });
  return Response.redirect(session.url, 303);
}

async function safeCheckoutInput(request: Request): Promise<{ email?: string; plan?: string; successUrl?: string; cancelUrl?: string }> {
  try {
    const body = await request.json<Record<string, unknown>>();
    return {
      email: typeof body.email === "string" ? body.email : undefined,
      plan: typeof body.plan === "string" ? body.plan : undefined,
      successUrl: typeof body.successUrl === "string" ? body.successUrl : undefined,
      cancelUrl: typeof body.cancelUrl === "string" ? body.cancelUrl : undefined
    };
  } catch {
    return {};
  }
}

function normalizeOptionalEmail(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined;
  const trimmed = value.trim().toLowerCase();
  if (!trimmed || trimmed.length > 254) return undefined;
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)) return undefined;
  return trimmed;
}

function normalizeCheckoutReturn(value: string | undefined, fallback: string, origin: string): string {
  if (!value) return fallback;
  try {
    const url = new URL(value);
    return url.origin === origin ? url.toString() : fallback;
  } catch {
    return fallback;
  }
}

function checkoutEventProperties(env: Env, plan: string, paymentLink: boolean): Record<string, unknown> {
  return {
    product_name: "Baseline Pro Monitoring",
    plan,
    payment_link: paymentLink,
    site_id: "baseline-ai",
    app_url: baseURL(env)
  };
}

async function emitCheckoutStartedEvents(env: Env, email: string | undefined, plan: string, paymentLink: boolean): Promise<void> {
  const uniqueId = crypto.randomUUID();
  const time = new Date().toISOString();
  const properties = checkoutEventProperties(env, plan, paymentLink);
  await Promise.all([
    emitKlaviyoEvent(env, {
      email,
      metric: "Baseline Pro Checkout Started",
      uniqueId,
      time,
      properties
    }),
    emitKlaviyoEvent(env, {
      email: normalizeOptionalEmail(env.BASELINE_MASTER_EMAIL),
      metric: "Baseline Master Notification",
      uniqueId: "master:" + uniqueId,
      time,
      properties: {
        ...properties,
        event_type: "checkout_started",
        customer_email_present: Boolean(email)
      }
    })
  ]);
}

async function emitKlaviyoEvent(
  env: Env,
  event: { email?: string | null; metric: string; uniqueId: string; time: string; properties: Record<string, unknown> }
): Promise<{ sent: boolean; skipped: boolean }> {
  const apiKey = env.KLAVIYO_PRIVATE_API_KEY;
  if (!apiKey || !event.email) return { sent: false, skipped: true };
  const body = {
    data: {
      type: "event",
      attributes: {
        metric: { data: { type: "metric", attributes: { name: event.metric } } },
        profile: { data: { type: "profile", attributes: { email: event.email } } },
        properties: event.properties,
        time: event.time,
        unique_id: event.uniqueId
      }
    }
  };
  try {
    const response = await fetch("https://a.klaviyo.com/api/events", {
      method: "POST",
      headers: {
        authorization: "Klaviyo-API-Key " + apiKey,
        "content-type": "application/vnd.api+json",
        accept: "application/vnd.api+json",
        revision: env.KLAVIYO_REVISION || "2026-04-15"
      },
      body: JSON.stringify(body)
    });
    return { sent: response.ok, skipped: false };
  } catch {
    return { sent: false, skipped: false };
  }
}

async function ensureSchema(sql: NeonQueryFunction<false, false>): Promise<void> {
  await sql`create table if not exists baseline_runs (
    id text primary key,
    workspace text not null,
    agent_kind text not null,
    status text not null,
    health_score integer not null,
    mode text not null,
    payload jsonb not null,
    created_at timestamptz not null default now()
  )`;
  await sql`create index if not exists baseline_runs_created_at_idx on baseline_runs (created_at desc)`;
  await sql`create table if not exists baseline_events (
    id text primary key,
    type text not null,
    path text not null,
    payload jsonb not null,
    created_at timestamptz not null default now()
  )`;
  await sql`create index if not exists baseline_events_created_at_idx on baseline_events (created_at desc)`;
  await sql`create table if not exists canonical_question_sets (
    slug text not null,
    version text not null,
    title text not null,
    questions jsonb not null,
    active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    primary key (slug, version)
  )`;
  await sql`create table if not exists llm_evaluations (
    id text primary key,
    run_id text not null,
    question_set_slug text not null,
    question_set_version text not null,
    model text not null,
    score integer not null,
    verdict text not null,
    payload jsonb not null,
    created_at timestamptz not null default now()
  )`;
  await sql`create index if not exists llm_evaluations_run_idx on llm_evaluations (run_id, created_at desc)`;
  await ensureCloudSchema(sql);
}

function landingPage(env: Env): string {
  return layout(env, "Baseline.ai | Keep coding agents inside the lines", `
    <main class="fieldLanding">
      <section class="fieldHero" id="the-check">
        <div class="film heroStill filmGrain">
          <img src="/assets/baseline-court-serve.png" alt="A tennis robot standing on a sunlit court">
        </div>
        <div class="heroCopy">
          <div>
            <p class="eyebrow">Installs in seconds</p>
            <h1>Your agent forgot. Baseline did not.</h1>
            <p class="bodyText heroLede">Probe your agent in seconds to track real-life drift over time. Keep Hermes and OpenClaw dialed, and know before issues reach the work.</p>
            <div class="fieldActions">
              <a class="btn btnPrimary" href="/docs/mcp">install the cli &rarr;</a>
              <a class="btn btnGhost" href="/dashboard">watch a run</a>
            </div>
          </div>
          <div class="terminalSample">
            <p class="eyebrow">Installs in seconds</p>
            <pre class="codeBlock"><code><span class="cmt">$</span> brew install trackbaseline/tap/baseline
<span class="cmt">$</span> baseline run
<span class="key">-&gt;</span> run_8f2c . score 92 . 11 ok / 3 watch / 0 fail
<span class="cmt">#</span> watch: <span class="key">mcp.openclaw.config</span> drifted from baseline_clean-local
<span class="cmt">#</span> watch: <span class="key">latency.tool_probe</span> +312ms over 5-day median</code></pre>
          </div>
        </div>
      </section>

      <section class="scoreboardSection" aria-labelledby="scoreboard-heading">
        <div class="sectionTitleRow">
          <div>
            <p class="eyebrow">Today . across your workstations</p>
            <h2 id="scoreboard-heading">Three agents. One scoreboard.</h2>
          </div>
          <a class="underLink" href="/dashboard">open dashboard &rarr;</a>
        </div>
        ${agentScoreboard()}
      </section>

      <section class="statRibbon" aria-label="Baseline metrics">
        <div><strong>14</strong><span>Probes in the default set</span></div>
        <div><strong>~8s</strong><span>P50 local run</span></div>
        <div><strong>0 raw</strong><span>Cloud export: prompts / outputs / paths</span></div>
        <div><strong>7 mcp</strong><span>Setup, run, doctor, report, accept, schedule, scrub</span></div>
      </section>

      <section class="dailySection">
        <div>
          <p class="eyebrow">Run Baseline daily</p>
          <h2>Compare agent responses against a known good baseline.</h2>
          <p class="bodyText">Learn about latency, drift, and memory issues in real time, before they impact your work.</p>
          <hr>
          <ul class="dashList">
            <li>Enable daily deep or fast checks.</li>
            <li>Compare responses against good baselines, by agent.</li>
            <li>Review changes over time to see model and harness impact.</li>
          </ul>
        </div>
        <div class="film filmGrain portraitStill">
          <img src="/assets/baseline-court-walkaway.png" alt="A tennis robot walking away across a court baseline">
        </div>
      </section>

      <section class="probesSection" id="probes">
        <div class="sectionTitleRow">
          <div>
            <p class="eyebrow">The default set</p>
            <h2>Fourteen probes.</h2>
          </div>
          <a class="underLink" href="/docs/mcp">read the rubric &rarr;</a>
        </div>
        ${probeRows()}
      </section>

      <section class="commandSection">
        <div>
          <p class="eyebrow">Four commands</p>
          <h2>Install. Establish. Compare. Accept.</h2>
          <p class="bodyText">The local loop. Seven MCP tools on top for agents that want to call Baseline themselves. No web app required to get value; the cloud surface is for history, alerts, and team-visible evidence after the workstation is already producing redacted runs.</p>
        </div>
        ${stepBlocks()}
      </section>

      <section class="imageTriptych" aria-label="Baseline field imagery">
        <figure>
          <div class="film filmGrain"><img src="/assets/baseline-court-side.png" alt="A tennis robot in profile holding a racket"></div>
          <figcaption><span>01 / Walk-off</span><strong>After a clean session.</strong></figcaption>
        </figure>
        <figure>
          <div class="film filmGrain"><img src="/assets/baseline-court-humanoid.png" alt="A humanoid tennis robot holding a racket"></div>
          <figcaption><span>02 / Warm-up</span><strong>Before the day starts.</strong></figcaption>
        </figure>
        <figure>
          <div class="film filmGrain"><img src="/assets/baseline-court-line.png" alt="A tennis robot walking along a bright court line"></div>
          <figcaption><span>03 / Waiting</span><strong>For the next instruction.</strong></figcaption>
        </figure>
      </section>

      <section class="pricingSection" id="pricing">
        <div class="pricingIntro">
          <div>
            <p class="eyebrow">Pricing</p>
            <h2>Start local. Pay when drift becomes operational risk.</h2>
          </div>
          <p class="bodyText">The CLI and MCP are free. Pro adds redacted run history, lifecycle email, and private probes. Team adds shared workstations and routed alerts.</p>
        </div>
        <div class="priceTable">
          ${priceColumn("Local", "$0", "", "The whole product", ["SQLite store. MCP. Scrub preview.", "Local report artifacts.", "Good Baseline / compare."], "install", "/docs/mcp", false)}
          ${priceColumn("Pro", "$39", "/mo / workstation", "When history matters", ["Redacted run history.", "Checkout-linked account.", "Lifecycle email + private probes."], "buy pro", "/api/checkout?plan=pro", true)}
          ${priceColumn("Team", "$129", "/mo / team", "When alerts need to route", ["Shared dashboards.", "Token / workspace model.", "Audit exports."], "buy team", "/api/checkout?plan=team", false)}
        </div>
      </section>

      <section class="fieldNotes">
        <div class="sectionTitleRow">
          <div>
            <p class="eyebrow">Field notes</p>
            <h2>From the workstation.</h2>
          </div>
          <a class="underLink" href="/blog">all notes &rarr;</a>
        </div>
        <div class="noteGrid">
          ${noteCard("2026 . 05 . 14", "How to accept a Good Baseline.", "The five-minute review ritual before a run becomes the standard your workstation compares against.")}
          ${noteCard("2026 . 04 . 28", "MCP drift looks like nothing, until it costs a day.", "Three real incidents from operator pilots. None of them showed up in trace dashboards.")}
          ${noteCard("2026 . 04 . 09", "The case against a leaderboard.", "Why Baseline does not score models against each other, and what it scores instead.")}
        </div>
      </section>

      <section class="closerSection">
        <h2>In the line, or out.</h2>
        <div>
          <p class="bodyText">Install the CLI. Run the check once. Accept the run that earns it. That is the first hour with Baseline.</p>
          <div class="fieldActions">
            <a class="btn paperBtn" href="/docs/mcp">install &rarr;</a>
            <a class="btn outlinePaperBtn" href="/docs/mcp">read the docs</a>
          </div>
        </div>
      </section>
    </main>
    ${proAccountScript()}
  `, softwareJsonLD(env));
}

function agentScoreboard(): string {
  const agents = [
    {
      name: "OpenClaw",
      workspace: "apollo-street/api",
      score: 92,
      delta: "+1",
      status: "watch",
      trend: [62, 71, 88, 84, 79, 81, 91, 95, 88, 92, 90, 92, 92],
      findings: [
        ["ok", "identity", "model / context / goal matched"],
        ["watch", "mcp.config", "drift from clean-local"],
        ["watch", "latency.tool", "+312ms over 5-day median"],
      ],
    },
    {
      name: "Hermes",
      workspace: "apollo-street/web",
      score: 97,
      delta: "+2",
      status: "ok",
      trend: [70, 78, 84, 88, 91, 90, 93, 92, 94, 95, 96, 96, 97],
      findings: [
        ["ok", "identity", "stable for 14 runs"],
        ["ok", "memory", "context survives across calls"],
        ["ok", "safety.scrubber", "no leaks detected"],
      ],
    },
    {
      name: "Claude Code",
      workspace: "apollo-street/infra",
      score: 78,
      delta: "-9",
      status: "fail",
      trend: [88, 90, 91, 88, 87, 85, 84, 82, 80, 79, 78, 77, 78],
      findings: [
        ["fail", "memory", "context lost mid-session"],
        ["watch", "variance", "2 of 5 prompts diverged"],
        ["watch", "repo.workspace", "dirty: 11 unstaged files"],
      ],
    },
  ];
  return `<div class="agentGrid">${agents.map((agent) => `
    <article class="agentCard">
      <header><strong>${agent.name}</strong><code>${agent.workspace}</code></header>
      <div class="agentBody">
        <div>
          <p class="agentScore">${agent.score}<span>/100</span></p>
          <p class="agentStatus ${agent.status}">${agent.status} . ${agent.delta}</p>
        </div>
        <div>
          <p class="miniLabel">Last 13 runs</p>
          <div class="trendBars">${agent.trend.map((height) => `<span style="height:${height}%"></span>`).join("")}</div>
        </div>
      </div>
      <div class="findingList">${agent.findings.map(([status, id, message]) => `
        <div>
          <span class="${status}">${status}</span>
          <p><code>${id}</code><small>${message}</small></p>
        </div>
      `).join("")}</div>
    </article>
  `).join("")}</div>`;
}

function probeRows(): string {
  const probes = [
    ["identity", "Identity", "Model, provider, context window, primary goal."],
    ["repo", "Awareness", "Workspace path, clean/dirty state, branch."],
    ["tooling", "Tools", "MCP server reachable, allowed tool surface declared."],
    ["memory", "Memory", "Carried context survives between calls. Same answer twice."],
    ["latency", "Speed", "P50 and P95 against the 5-day local median."],
    ["variance", "Stability", "Five identical prompts. Five identical answers."],
    ["safety", "Safety", "Scrubber catches paths, secrets, prompt fragments."],
    ["style", "Style", "Repository conventions: tabs, naming, file layout."],
    ["change", "Awareness", "Reports any tool / MCP / repo / config change since Good."],
    ["reasoning", "Basic", "Two-plus-two. Date. The smoke test for a broken model."],
    ["instruction", "Obedience", "Answer only the word. Answer only the number."],
    ["redaction", "Safety", "Redacted summaries verify against original on push."],
    ["tool-call", "Tools", "Tool calls actually fire. Names match the declared surface."],
    ["session", "Stability", "A second session in the same workspace agrees with the first."],
  ];
  return `<div class="probeTable">${probes.map(([id, group, desc], index) => `
    <div class="probeRow">
      <span>${String(index + 1).padStart(2, "0")}</span>
      <span>${group}</span>
      <strong>${id}.</strong>
      <p>${desc}</p>
      <em>v0.1</em>
    </div>
  `).join("")}</div>`;
}

function stepBlocks(): string {
  const steps = [
    ["baseline setup", "Detect the local agent, write SQLite, run preflight."],
    ["baseline run --mode fast", "Fourteen probes against the active workstation. About eight seconds."],
    ["baseline accept run_8f2c --label clean-local", "Review the report. Mark this run as your Good Baseline."],
    ["baseline compare", "Every later run is judged against the accepted one."],
  ];
  return `<div class="stepStack">${steps.map(([cmd, desc], index) => `
    <div>
      <p><span>${String(index + 1).padStart(2, "0")}</span><code>$ ${cmd}</code></p>
      <small>${desc}</small>
    </div>
  `).join("")}</div>`;
}

function priceColumn(name: string, price: string, sub: string, tag: string, features: string[], cta: string, href: string, form: boolean): string {
  return `<article class="priceCol ${form ? "highlight" : ""}">
    <p class="eyebrow">${tag}</p>
    <h3>${name}</h3>
    <p class="price">${price}${sub ? `<span>${sub}</span>` : ""}</p>
    <ul>${features.map((feature) => `<li>${feature}</li>`).join("")}</ul>
    ${form ? `
      <form class="checkoutForm" data-checkout-form>
        <label class="srOnly" for="checkout-email">Email for Pro checkout</label>
        <input id="checkout-email" name="email" type="email" autocomplete="email" placeholder="email" required>
        <button class="btn btnPrimary" type="submit">${cta} &rarr;</button>
        <p class="checkoutStatus" data-checkout-status aria-live="polite"></p>
      </form>
    ` : `<a class="btn btnGhost" href="${href}">${cta} &rarr;</a>`}
  </article>`;
}

function noteCard(date: string, title: string, body: string): string {
  return `<a href="/blog" class="noteCard">
    <span>${date}</span>
    <strong>${title}</strong>
    <p>${body}</p>
    <em>Read &rarr;</em>
  </a>`;
}

function blogPage(env: Env): string {
  return layout(env, "Baseline.ai Blog", `
    <main class="doc blogPage">
      <p class="eyebrow">Baseline field notes</p>
      <h1>Blog stub</h1>
      <p>Short operator essays will live here: Good Baseline rituals, Pro monitoring rollout notes, MCP drift patterns, and launch evidence from the first paid pilots.</p>
      <div class="blogGrid">
        <article><strong>Accepting a Good Baseline</strong><span>Draft: how to review local artifacts before trusting a workstation state.</span></article>
        <article><strong>Pro Account Architecture</strong><span>Draft: Stripe checkout, Klaviyo lifecycle events, entitlement ledger, and redacted sync.</span></article>
        <article><strong>The Baseline Court</strong><span>Draft: why line calls, scoreboards, and warmups are the product metaphor.</span></article>
      </div>
    </main>
  `, softwareJsonLD(env));
}

function checkoutSuccessPage(env: Env): string {
  return layout(env, "Baseline.ai Pro checkout success", `
    <main class="doc">
      <p class="eyebrow">Checkout returned</p>
      <h1>Pro checkout received.</h1>
      <p>Stripe returned successfully. The next backend bead should verify the session, store the entitlement, and issue or connect a Pro monitoring token for redacted run sync.</p>
      <pre><code>baseline sync on --url ${escapeHTML(baseURL(env))} --token YOUR_BASELINE_TOKEN
baseline sync push</code></pre>
      <p><a class="button primary" href="/dashboard">Open dashboard</a></p>
    </main>
  `, softwareJsonLD(env));
}

function checkoutCancelPage(env: Env): string {
  return layout(env, "Baseline.ai Pro checkout canceled", `
    <main class="doc">
      <p class="eyebrow">Checkout canceled</p>
      <h1>No Pro subscription was started.</h1>
      <p>The local Baseline CLI and MCP remain free. Pro is for monitored history, alerting, and team-visible evidence once the local loop is useful.</p>
      <p><a class="button secondary" href="/">Return home</a></p>
    </main>
  `, softwareJsonLD(env));
}

function dashboardPage(env: Env): string {
  return layout(env, "Baseline.ai Dashboard", `
    <main class="dashboard">
      <section class="dashHead">
        <div>
          <p class="eyebrow">Visual dashboard</p>
          <h1 id="dashboard-summary">Loading latest baseline run.</h1>
        </div>
        <a class="button secondary" href="/docs/mcp">Connect MCP</a>
      </section>
      ${dashboardVisual(true)}
      <section class="band two">
        <div class="panel">
          <h2>Latest findings</h2>
          <div id="latest-findings"><div class="alert warning">Waiting for synced Baseline runs.</div></div>
        </div>
        <div class="panel">
          <h2>Recent runs</h2>
          <table id="run-timeline">
            <tr><th>Run</th><th>Score</th><th>Status</th><th>Mode</th></tr>
          </table>
        </div>
      </section>
    </main>
    ${dashboardScript()}
  `, softwareJsonLD(env));
}

function adminPage(env: Env): string {
  const configured = Boolean(env.BASELINE_ADMIN_TOKEN);
  return layout(env, "Baseline.ai Admin", `
    <main class="doc admin">
      <p class="eyebrow">Admin</p>
      <h1>Canonical question sets</h1>
      <p>Version the baseline packs that every local agent run is compared against. Mutations require <code>BASELINE_ADMIN_TOKEN</code>; evaluations use OpenAI structured outputs when <code>OPENAI_API_KEY</code> is configured, otherwise they use the local heuristic evaluator.</p>
      ${configured ? "" : `<div class="alert warning">Admin token is not configured. Set <code>BASELINE_ADMIN_TOKEN</code> as a Worker secret before saving changes.</div>`}
      <label>Admin token <input id="admin-token" type="password" autocomplete="off" placeholder="BASELINE_ADMIN_TOKEN"></label>
      <div class="actions adminActions">
        <button class="button primary" id="load-question-sets" type="button">Load sets</button>
        <button class="button secondary" id="run-evaluator" type="button">Evaluate latest run</button>
      </div>
      <h2>Question set JSON</h2>
      <textarea id="question-set-json" spellcheck="false">${escapeHTML(JSON.stringify(defaultQuestionSet(), null, 2))}</textarea>
      <div class="actions adminActions">
        <button class="button primary" id="save-question-set" type="button">Save version</button>
      </div>
      <h2>Output</h2>
      <pre id="admin-output"><code>Ready.</code></pre>
    </main>
    ${adminScript()}
  `, softwareJsonLD(env));
}

function mcpDocsPage(env: Env): string {
  const install = `go build -o bin/baseline ./cmd/baseline
./bin/baseline setup
./bin/baseline report
./bin/baseline accept RUN_ID --confirm "accept RUN_ID" --label clean-local
openclaw mcp list
./bin/baseline compare`;
  return layout(env, "Baseline MCP installation", `
    <main class="doc">
      <p class="eyebrow">MCP installation</p>
      <h1>Install Baseline into OpenClaw</h1>
      <p>Baseline exposes seven legible MCP tools: setup, run, doctor, report, accept, schedule, and scrub preview. Doctor is local preflight; setup and run start the operator-approved default eval in the background and write local markdown artifacts.</p>
      <pre><code>${escapeHTML(install)}</code></pre>
      <h2>Cloud sync</h2>
      <pre><code>baseline sync on --url ${escapeHTML(baseURL(env))} --token YOUR_BASELINE_TOKEN
baseline doctor
baseline sync push</code></pre>
      <h2>Remote MCP</h2>
      <p>Pro accounts can also connect to the cloud MCP at <code>${escapeHTML(baseURL(env))}/mcp</code>. The remote MCP never runs local probes; it reads account history, hotspots, self-history comparisons, workspace tokens, and Stripe portal handoffs after magic-link session auth.</p>
      <pre><code>POST ${escapeHTML(baseURL(env))}/api/auth/magic-link
POST ${escapeHTML(baseURL(env))}/api/auth/consume
Authorization: Bearer YOUR_SESSION_TOKEN
POST ${escapeHTML(baseURL(env))}/mcp</code></pre>
      <h2>Safety model</h2>
      <p>The MCP can read what the connected agent gives it. Baseline defaults to local SQLite and redacted summaries. Raw outputs are not exported unless <code>allow_raw_output</code> is enabled in <code>~/.baseline/config.json</code>.</p>
      <h2>Recommended first Good Baseline</h2>
      <pre><code>baseline setup
baseline report
baseline accept RUN_ID --confirm "accept RUN_ID" --label clean-local
baseline compare</code></pre>
    </main>
  `, softwareJsonLD(env));
}

function privacyPage(env: Env): string {
  return layout(env, "Baseline.ai Privacy", `
    <main class="doc"><h1>Privacy</h1><p>Baseline is local-first. Cloud sync stores run summaries, health scores, findings, and redacted observation hashes. Raw prompts and outputs are not required for v0 cloud sync.</p><p>API tokens can be revoked by deleting them from the local config and dashboard. Synthetic and user-provided redaction checks run before export.</p></main>
  `);
}

function termsPage(env: Env): string {
  return layout(env, "Baseline.ai Terms", `
    <main class="doc"><h1>Terms</h1><p>Baseline v0 is a monitoring and alerting tool for agent workstations. It does not guarantee task correctness, security compliance, or model behavior. Users remain responsible for reviewing agent outputs before production use.</p></main>
  `);
}

function notFoundPage(env: Env): string {
  return layout(env, "Not found", `<main class="doc"><h1>Not found</h1><p>The page does not exist.</p></main>`);
}

function dashboardVisual(live = false): string {
  const id = (name: string) => live ? ` id="${name}"` : "";
  return `
    <div class="productFrame">
      <div class="frameTop"><span></span><strong${id("frame-run")}>baseline run_dihj6f94</strong><em${id("frame-score")}>score 92</em></div>
      <div class="scoreRow">
        <div class="scoreBlock"><b${id("health-score")}>92</b><span>Health</span></div>
        <div class="miniBars"${id("health-bars")}><i style="height:42%"></i><i style="height:78%"></i><i style="height:60%"></i><i style="height:88%"></i><i style="height:51%"></i><i style="height:70%"></i></div>
        <div class="signalList"${id("signal-list")}><p><span class="dot okDot"></span>Scrubber clean</p><p><span class="dot warnDot"></span>MCP missing</p><p><span class="dot badDot"></span>Latency up</p></div>
      </div>
      <div class="probeGrid"${id("probe-grid")}>
        <div><strong>identity</strong><span>pass</span></div>
        <div><strong>repo</strong><span>pass</span></div>
        <div><strong>tooling</strong><span>warn</span></div>
        <div><strong>style</strong><span>pass</span></div>
      </div>
    </div>
  `;
}

function proAccountScript(): string {
  return `<script>
    (function(){
      const form = document.querySelector("[data-checkout-form]");
      const status = form && form.querySelector("[data-checkout-status]");
      const button = form && form.querySelector("button");
      const write = function(message){ if (status) status.textContent = message; };
      form && form.addEventListener("submit", async function(event){
        event.preventDefault();
        if (!button) return;
        const data = new FormData(form);
        const email = String(data.get("email") || "").trim();
        if (!email) {
          write("Enter an email to open checkout.");
          return;
        }
        button.disabled = true;
        write("Opening Stripe checkout...");
        try {
          const response = await fetch("/api/checkout", {
            method: "POST",
            headers: { "content-type": "application/json", "accept": "application/json" },
            body: JSON.stringify({ plan: "pro", email, successUrl: location.origin + "/checkout/success?session_id={CHECKOUT_SESSION_ID}", cancelUrl: location.origin + "/checkout/cancel" })
          });
          const payload = await response.json();
          if (!response.ok || !payload.url) throw new Error(payload.error || "checkout_failed");
          window.location.assign(payload.url);
        } catch (error) {
          write("Checkout is not configured yet. The local CLI is still free.");
          button.disabled = false;
        }
      });
    })();
  </script>`;
}

function dashboardScript(): string {
  return `<script>
    (async function(){
      const text = function(value){ return String(value == null ? "" : value); };
      const shortRun = function(id){ return text(id).replace(/^run_/, "").slice(0, 12) || "no-run"; };
      const setText = function(id, value){ const el = document.getElementById(id); if (el) el.textContent = value; };
      const statusClass = function(status){ return status === "ok" ? "ok" : (status === "critical" ? "bad" : "warning"); };
      try {
        const latestResp = await fetch("/api/runs/latest", { headers: { "accept": "application/json" } });
        const latest = await latestResp.json();
        const run = latest.run || {};
        const score = Number(run.health_score || 0);
        setText("dashboard-summary", "Latest " + text(run.agent_kind || "agent") + " run is " + text(run.status || "unknown") + " with score " + score + ".");
        setText("frame-run", "baseline " + shortRun(run.run_id));
        setText("frame-score", "score " + score);
        setText("health-score", String(score));
        const checks = Array.isArray(run.checks) ? run.checks : [];
        const signals = document.getElementById("signal-list");
        if (signals) {
          const rows = checks.slice(0, 5).map(function(check){
            const klass = check.status === "ok" ? "okDot" : (check.status === "critical" ? "badDot" : "warnDot");
            return "<p><span class=\\"dot " + klass + "\\"></span>" + text(check.check_id || check.kind || "check") + " " + text(check.status || "unknown") + "</p>";
          });
          signals.innerHTML = rows.length ? rows.join("") : "<p><span class=\\"dot warnDot\\"></span>No checks received</p>";
        }
        const findings = document.getElementById("latest-findings");
        if (findings) {
          const bad = checks.filter(function(check){ return check.status !== "ok"; }).slice(0, 6);
          findings.innerHTML = (bad.length ? bad : checks.slice(0, 3)).map(function(check){
            return "<div class=\\"alert " + statusClass(check.status) + "\\">" + text(check.check_id || "check") + ": " + text(check.status || "unknown") + " · " + Math.round(Number(check.score || 0)) + "</div>";
          }).join("") || "<div class=\\"alert warning\\">No synced checks yet.</div>";
        }
        const grid = document.getElementById("probe-grid");
        if (grid) {
          grid.innerHTML = checks.slice(0, 8).map(function(check){
            return "<div><strong>" + text(check.kind || check.check_id || "probe") + "</strong><span>" + text(check.status || "unknown") + "</span></div>";
          }).join("");
        }
        const timelineResp = await fetch("/api/runs/timeline", { headers: { "accept": "application/json" } });
        const timeline = await timelineResp.json();
        const table = document.getElementById("run-timeline");
        if (table) {
          const runs = Array.isArray(timeline.runs) ? timeline.runs : [];
          table.innerHTML = "<tr><th>Run</th><th>Score</th><th>Status</th><th>Mode</th></tr>" + runs.slice(0, 12).map(function(row){
            return "<tr><td>" + shortRun(row.run_id) + "</td><td>" + Number(row.health_score || 0) + "</td><td>" + text(row.status || "unknown") + "</td><td>" + text(row.mode || "unknown") + "</td></tr>";
          }).join("");
        }
      } catch (error) {
        setText("dashboard-summary", "Dashboard could not load run data.");
      }
    })();
  </script>`;
}

function adminScript(): string {
  return `<script>
    (function(){
      const out = document.getElementById("admin-output");
      const editor = document.getElementById("question-set-json");
      const tokenInput = document.getElementById("admin-token");
      const write = function(value){ if (out) out.textContent = typeof value === "string" ? value : JSON.stringify(value, null, 2); };
      const token = function(){ return tokenInput && tokenInput.value ? tokenInput.value : ""; };
      const adminFetch = async function(path, options){
        const headers = Object.assign({ "accept": "application/json", "content-type": "application/json" }, options && options.headers || {});
        const t = token();
        if (t) headers.authorization = "Bearer " + t;
        const response = await fetch(path, Object.assign({}, options || {}, { headers }));
        const body = await response.json();
        if (!response.ok) throw body;
        return body;
      };
      document.getElementById("load-question-sets")?.addEventListener("click", async function(){
        try {
          const body = await adminFetch("/api/admin/question-sets");
          if (editor && body.question_sets && body.question_sets[0]) editor.value = JSON.stringify(body.question_sets[0], null, 2);
          write(body);
        } catch (error) { write(error); }
      });
      document.getElementById("save-question-set")?.addEventListener("click", async function(){
        try {
          const payload = JSON.parse(editor && editor.value ? editor.value : "{}");
          write(await adminFetch("/api/admin/question-sets", { method: "POST", body: JSON.stringify(payload) }));
        } catch (error) { write(error); }
      });
      document.getElementById("run-evaluator")?.addEventListener("click", async function(){
        try {
          const payload = JSON.parse(editor && editor.value ? editor.value : "{}");
          write(await adminFetch("/api/admin/evaluate", { method: "POST", body: JSON.stringify({ slug: payload.slug, version: payload.version }) }));
        } catch (error) { write(error); }
      });
    })();
  </script>`;
}

function layout(env: Env, title: string, body: string, structuredData = ""): string {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHTML(title)}</title>
  <meta name="description" content="Local-first baseline checks and Pro monitoring for coding agents, MCP tools, repo awareness, memory, latency, and style.">
  <meta name="theme-color" content="#071419">
  <meta property="og:title" content="${escapeHTML(title)}">
  <meta property="og:description" content="Keep coding agents inside the lines with known-good checks, redacted run history, and practical drift alerts.">
  <meta property="og:type" content="website">
  <meta property="og:image" content="${escapeHTML(baseURL(env))}/assets/baseline-court-robot.png">
  <style>${css()}</style>
  ${structuredData}
</head>
<body>
  <header class="nav">
    <a href="/" class="brandLockup"><span><img src="/assets/baseline-court-serve.png" alt=""></span><strong>baseline.</strong></a>
    <nav class="navLinks"><a href="/#the-check">the check</a><a href="/docs/mcp">docs</a><a href="/#pricing">pricing</a><a href="/blog">field notes</a></nav>
    <div class="navCtas"><a href="/dashboard">sign in</a><a class="btn btnPrimary" href="/docs/mcp">install</a></div>
  </header>
  ${body}
  <footer><a href="/" class="brandLockup small"><span><img src="/assets/baseline-court-serve.png" alt=""></span><strong>baseline.</strong></a><a href="/docs/mcp">Docs</a><a href="/blog">Blog</a><a href="/privacy">Privacy</a><a href="/terms">Terms</a><span>2026 TRACKBASELINE.COM</span></footer>
  <script>
    document.querySelectorAll('a[href^="/api/checkout"], a[href="/docs/mcp"]').forEach(function(a){
      a.addEventListener('click', function(){ navigator.sendBeacon && navigator.sendBeacon('/api/events', JSON.stringify({type:'cta_click', path: location.pathname, href: a.getAttribute('href')})); });
    });
  </script>
</body>
</html>`;
}

function css(): string {
  return `
    :root { color-scheme: light; --ink:#071419; --graphite:#142124; --muted:#586569; --paper:#f3ebdc; --cream:#fff9ea; --line:#10191b; --court:#0e5960; --court-soft:#d6e7e4; --clay:#bb7357; --lime:#d9f45d; --blue:#2d6f9f; --green:#166b48; --amber:#a15c00; --red:#b42318; --shadow:5px 5px 0 #071419; --display:"Avenir Next Condensed","DIN Condensed","Franklin Gothic Condensed",Impact,sans-serif; --body:"Avenir Next","Gill Sans","Trebuchet MS",system-ui,sans-serif; --mono:"SFMono-Regular",Consolas,monospace; }
    * { box-sizing: border-box; }
    html { scroll-behavior:smooth; }
    body { margin:0; font-family:var(--body); color:var(--ink); background:repeating-linear-gradient(90deg, rgba(7,20,25,.045) 0 1px, transparent 1px 68px), repeating-linear-gradient(0deg, rgba(187,115,87,.08) 0 1px, transparent 1px 68px), var(--paper); letter-spacing:0; font-size:18px; line-height:1.5; overflow-x:clip; }
    a { color:inherit; text-decoration:none; }
    a:focus-visible, button:focus-visible, input:focus-visible { outline:3px solid var(--lime); outline-offset:4px; }
    .nav { min-height:68px; display:flex; align-items:center; justify-content:space-between; gap:24px; padding:0 max(20px, calc((100vw - 1180px) / 2)); border-bottom:3px solid var(--line); background:rgba(243,235,220,.96); position:sticky; top:0; z-index:20; }
    .brand { display:inline-flex; align-items:center; min-height:40px; background:var(--ink); color:var(--cream); padding:8px 12px; font:900 22px/1 var(--display); text-transform:uppercase; }
    nav { display:flex; flex-wrap:wrap; gap:18px; font-size:14px; font-weight:900; color:var(--graphite); text-transform:uppercase; }
    nav a { border-bottom:2px solid transparent; padding:8px 0; }
    nav a:hover { border-color:var(--clay); }
    .hero { position:relative; overflow:hidden; border-bottom:3px solid var(--line); }
    .courtHero { min-height:720px; display:flex; align-items:center; isolation:isolate; background:var(--court-soft); }
    .courtHero::after { content:""; position:absolute; right:-40px; bottom:116px; width:58%; height:48px; background:var(--clay); box-shadow:-34px -28px 0 var(--ink), 24px 24px 0 var(--lime); z-index:1; pointer-events:none; }
    .heroArt { position:absolute; inset:0 0 0 auto; width:58%; height:100%; object-fit:cover; object-position:center; z-index:0; filter:saturate(.95) contrast(1.05); }
    .heroText { position:relative; z-index:2; width:min(720px, calc(100% - 40px)); margin-left:max(20px, calc((100vw - 1180px) / 2)); padding:58px 28px 92px; background:var(--cream); border:3px solid var(--line); box-shadow:var(--shadow); }
    .proofStrip { position:absolute; z-index:3; left:max(20px, calc((100vw - 1180px) / 2)); right:max(20px, calc((100vw - 1180px) / 2)); bottom:24px; display:grid; grid-template-columns:repeat(3, 1fr); border:3px solid var(--line); background:var(--cream); }
    .proofStrip span { min-height:62px; display:flex; align-items:center; padding:14px 18px; font-weight:900; text-transform:uppercase; }
    .proofStrip span + span { border-left:3px solid var(--line); }
    .eyebrow { margin:0 0 16px; color:var(--clay); font-size:14px; line-height:1; font-weight:900; text-transform:uppercase; letter-spacing:0; }
    h1, h2, h3 { margin:0; font-family:var(--display); font-weight:900; letter-spacing:0; text-transform:uppercase; text-wrap:balance; }
    h1 { max-width:640px; margin-bottom:18px; font-size:6rem; line-height:.9; }
    h2 { font-size:3.6rem; line-height:.95; margin-bottom:20px; }
    h3 { font-size:1.6rem; line-height:1; margin-bottom:12px; }
    .lede { max-width:640px; font-size:1.35rem; line-height:1.28; color:var(--graphite); margin:0 0 28px; }
    p, li, td { color:var(--muted); }
    .fine { color:var(--graphite); font-size:.95rem; margin-top:16px; font-weight:800; }
    .actions { display:flex; gap:12px; flex-wrap:wrap; margin-top:30px; }
    .button { min-height:48px; display:inline-flex; align-items:center; justify-content:center; border-radius:6px; padding:0 18px; font-weight:900; border:2px solid var(--line); text-transform:uppercase; transition:transform 180ms cubic-bezier(.25,1,.5,1), box-shadow 180ms cubic-bezier(.25,1,.5,1), background 180ms ease; }
    button.button { cursor:pointer; font:inherit; }
    .button:hover { transform:translate(-2px, -2px); box-shadow:4px 4px 0 var(--lime); }
    .button:disabled { cursor:progress; opacity:.72; transform:none; box-shadow:none; }
    .primary { background:var(--ink); color:var(--cream); border-color:var(--ink); }
    .secondary { background:var(--cream); color:var(--ink); }
    .band { padding:78px max(28px, calc((100vw - 1180px) / 2)); }
    .tight { padding-top:24px; padding-bottom:24px; }
    .two { display:grid; grid-template-columns:minmax(0, 1fr) minmax(320px, 460px); gap:42px; align-items:start; }
    .sectionHead { max-width:860px; margin-bottom:34px; }
    .metricStrip { display:grid; grid-template-columns:repeat(3, 1fr); gap:0; border:3px solid var(--line); background:var(--cream); box-shadow:var(--shadow); }
    .metricStrip div { padding:22px; }
    .metricStrip div + div { border-left:3px solid var(--line); }
    .metricStrip strong { display:block; font:900 2.2rem/1 var(--display); margin-bottom:8px; color:var(--ink); }
    .metricStrip span, p, li { line-height:1.55; }
    .checks { padding-left:18px; }
    .imageStack { display:grid; gap:14px; }
    .imageStack img, .imageBand img { width:100%; display:block; border:3px solid var(--line); box-shadow:var(--shadow); object-fit:cover; background:var(--cream); }
    .imageStack img { aspect-ratio:1 / 1; }
    .imageStack img:nth-child(2) { margin-left:32px; width:calc(100% - 32px); }
    .docsBand { background:var(--ink); color:var(--cream); border-top:3px solid var(--line); border-bottom:3px solid var(--line); }
    .docsBand p, .docsBand li, .docsBand td { color:#d8e2dd; }
    .docsBand .eyebrow { color:var(--lime); }
    .steps, .priceGrid, .blogGrid { display:grid; grid-template-columns:repeat(3, 1fr); gap:16px; }
    .steps div, .priceGrid article, .docGrid article, .blogGrid a, .blogGrid article, .panel { border:3px solid var(--line); border-radius:8px; padding:22px; background:var(--cream); color:var(--ink); box-shadow:var(--shadow); min-width:0; }
    .docsBand .steps div, .docsBand .docGrid article { background:#10272a; color:var(--cream); border-color:var(--cream); box-shadow:5px 5px 0 var(--lime); }
    .steps span { display:inline-flex; min-width:42px; height:34px; align-items:center; justify-content:center; background:var(--clay); color:var(--cream); font-weight:900; margin-bottom:18px; border:2px solid var(--line); }
    .steps strong { display:block; margin-bottom:10px; color:inherit; }
    .docGrid { display:grid; grid-template-columns:minmax(0, 1.35fr) minmax(280px, .65fr); gap:16px; margin-top:16px; }
    .callout { border-color:var(--lime); }
    .alert { border-left:4px solid var(--line); padding:12px 14px; margin:10px 0; background:var(--cream); border-radius:6px; font-weight:800; }
    .alert.ok { border-color:var(--green); }
    .alert.warning { border-color:var(--amber); }
    .alert.bad { border-color:var(--red); }
    code, pre { font-family:var(--mono); }
    code { overflow-wrap:anywhere; }
    pre { background:#0d1518; color:#f5f7eb; border-radius:8px; padding:18px; overflow:auto; line-height:1.5; border:2px solid var(--line); }
    table { width:100%; border-collapse:collapse; }
    th, td { text-align:left; border-bottom:2px solid currentColor; padding:10px 0; vertical-align:top; }
    .imageBand { display:grid; grid-template-columns:1.2fr .8fr 1fr; gap:16px; align-items:end; background:var(--court); }
    .imageBand img { height:390px; }
    .imageBand img:nth-child(2) { height:470px; }
    .pricing { background:var(--paper); border-top:3px solid var(--line); border-bottom:3px solid var(--line); display:grid; grid-template-columns:minmax(0, .8fr) minmax(0, 1.2fr); gap:36px; align-items:start; }
    .price { font:900 2.4rem/1 var(--display); color:var(--ink); margin:0 0 12px; }
    .checkoutForm { display:grid; gap:10px; margin-top:18px; }
    .checkoutForm input { min-height:48px; border:2px solid var(--line); border-radius:6px; background:var(--cream); color:var(--ink); padding:12px 14px; font:800 1rem/1 var(--body); width:100%; }
    .checkoutStatus { min-height:22px; margin:0; color:var(--graphite); font-weight:800; font-size:.9rem; }
    .srOnly { position:absolute; width:1px; height:1px; padding:0; margin:-1px; overflow:hidden; clip:rect(0,0,0,0); white-space:nowrap; border:0; }
    .blogPreview { background:var(--cream); }
    .blogGrid a, .blogGrid article { display:block; min-height:160px; }
    .blogGrid strong { display:block; font:900 1.45rem/1.05 var(--display); text-transform:uppercase; margin-bottom:12px; }
    .blogGrid span { color:var(--muted); }
    .dashboard { background:var(--paper); min-height:100vh; }
    .dashHead { padding:48px max(28px, calc((100vw - 1180px) / 2)) 18px; display:flex; justify-content:space-between; gap:24px; align-items:end; }
    .dashHead h1 { font-size:2.6rem; max-width:760px; line-height:1.05; }
    .dashboard > .productFrame { margin:0 max(28px, calc((100vw - 1180px) / 2)); }
    .productFrame { width:min(900px, 92vw); min-height:430px; border:3px solid var(--line); border-radius:8px; background:#fbfcfe; box-shadow:var(--shadow); overflow:hidden; }
    .frameTop { height:48px; display:flex; align-items:center; gap:12px; padding:0 18px; border-bottom:2px solid var(--line); background:#fff; }
    .frameTop span { width:10px; height:10px; border-radius:50%; background:var(--red); box-shadow:18px 0 var(--amber), 36px 0 var(--green); margin-right:42px; }
    .frameTop em { margin-left:auto; font-style:normal; color:var(--green); font-weight:900; }
    .scoreRow { display:grid; grid-template-columns:170px 1fr 220px; gap:22px; padding:28px; align-items:center; }
    .scoreBlock { border:2px solid var(--line); border-radius:8px; padding:20px; text-align:center; background:#fff; }
    .scoreBlock b { display:block; font:900 4.3rem/1 var(--display); }
    .scoreBlock span { color:var(--muted); font-weight:900; }
    .miniBars { height:190px; display:flex; align-items:end; gap:12px; border:2px solid var(--line); border-radius:8px; padding:18px; background:#fff; }
    .miniBars i { display:block; flex:1; min-width:18px; background:var(--court); border-radius:5px 5px 0 0; }
    .signalList { border:2px solid var(--line); border-radius:8px; background:#fff; padding:16px; }
    .signalList p { margin:12px 0; color:var(--ink); font-weight:800; }
    .dot { display:inline-block; width:9px; height:9px; border-radius:50%; margin-right:9px; }
    .okDot { background:var(--green); } .warnDot { background:var(--amber); } .badDot { background:var(--red); }
    .probeGrid { display:grid; grid-template-columns:repeat(4, 1fr); gap:12px; padding:0 28px 28px; }
    .probeGrid div { border:2px solid var(--line); background:#fff; border-radius:8px; padding:14px; }
    .probeGrid strong { display:block; margin-bottom:8px; }
    .probeGrid span { color:var(--muted); font-weight:900; }
    .doc { max-width:860px; margin:0 auto; padding:68px 28px; }
    .doc h1 { font-size:4rem; line-height:.95; }
    .doc h2 { border-top:3px solid var(--line); font-size:2.2rem; margin-top:40px; padding-top:28px; }
    .admin label { display:block; color:var(--muted); font-weight:900; margin:18px 0 8px; }
    .admin input, .admin textarea { width:100%; border:2px solid var(--line); border-radius:8px; padding:12px; font:inherit; color:var(--ink); background:#fff; }
    .admin textarea { min-height:430px; font-family:var(--mono); font-size:13px; line-height:1.45; resize:vertical; }
    .adminActions { margin:14px 0 26px; }
    footer { display:flex; flex-wrap:wrap; gap:20px; padding:30px max(20px, calc((100vw - 1180px) / 2)); color:var(--muted); border-top:3px solid var(--line); background:var(--paper); }
    :root { --bone:#ece2cf; --paper:#f5ede0; --paper-2:#faf4ea; --court:#b87560; --court-d:#8a4f3e; --sky:#8aa3ad; --sky-d:#5a747f; --fence:#2c3a42; --ink:#14110d; --ash:#6c6357; --line:#1a1612; --hairline:rgba(20,17,13,.14); --hairline-2:rgba(20,17,13,.08); --lime:#d9f45d; --green:#3d6b4a; --amber:#a15c00; --red:#b42318; --display:"Archivo","Archivo Narrow","Avenir Next Condensed","DIN Condensed",Impact,sans-serif; --body:"Archivo","Avenir Next",system-ui,sans-serif; --mono:"JetBrains Mono","SFMono-Regular",Consolas,monospace; }
    body { background:var(--bone); font-family:var(--body); color:var(--ink); }
    .nav { min-height:74px; display:grid; grid-template-columns:auto 1fr auto; align-items:center; gap:28px; padding:18px clamp(20px, 4vw, 56px); border-bottom:1px solid var(--ink); background:var(--bone); position:sticky; top:0; z-index:40; }
    .brandLockup { display:inline-flex; align-items:center; gap:14px; text-decoration:none; color:var(--ink); }
    .brandLockup span { width:32px; height:32px; border:1px solid var(--ink); display:block; overflow:hidden; background:var(--fence); }
    .brandLockup.small span { width:26px; height:26px; }
    .brandLockup img { width:100%; height:100%; object-fit:cover; object-position:50% 24%; display:block; }
    .brandLockup strong { font-family:var(--display); font-weight:700; font-stretch:125%; font-size:30px; line-height:1; letter-spacing:0; text-transform:none; }
    .brandLockup.small strong { font-size:25px; }
    .navLinks { display:flex; justify-content:center; gap:28px; font-family:var(--mono); font-size:11px; letter-spacing:.16em; text-transform:uppercase; font-weight:500; color:var(--fence); }
    .navLinks a, .navCtas a, footer a { text-decoration:none; color:inherit; border-bottom:1px solid transparent; }
    .navLinks a:hover, .navCtas a:hover, footer a:hover { border-color:currentColor; }
    .navCtas { display:flex; align-items:center; justify-content:flex-end; gap:12px; font-family:var(--mono); font-size:11px; letter-spacing:.16em; text-transform:uppercase; color:var(--fence); }
    footer { display:grid; grid-template-columns:1fr auto auto auto auto auto; align-items:center; gap:28px; padding:34px clamp(20px, 4vw, 56px); border-top:1px solid var(--ink); background:var(--bone); font-family:var(--mono); font-size:11px; letter-spacing:.14em; text-transform:uppercase; color:var(--ash); }
    .fieldLanding { background:var(--bone); color:var(--ink); overflow:hidden; }
    .fieldLanding h1, .fieldLanding h2, .fieldLanding h3 { font-family:var(--display); font-weight:700; font-stretch:125%; letter-spacing:0; text-transform:none; text-wrap:balance; color:var(--ink); }
    .fieldLanding h1 { font-size:clamp(4rem, 7.4vw, 88px); line-height:.92; margin:18px 0 24px; max-width:660px; }
    .fieldLanding h2 { font-size:clamp(3rem, 5.3vw, 64px); line-height:.92; margin:18px 0 24px; }
    .fieldLanding h3 { font-size:clamp(2rem, 3vw, 40px); line-height:1; margin:12px 0 4px; }
    .eyebrow { font-family:var(--mono); font-size:11px; line-height:1.25; letter-spacing:.18em; text-transform:uppercase; color:var(--ash); margin:0; }
    .bodyText { font-family:var(--body); color:var(--fence); font-size:16px; line-height:1.5; margin:0; max-width:560px; }
    .fieldHero { display:grid; grid-template-columns:minmax(0, 1.05fr) minmax(0, 1fr); min-height:680px; border-bottom:1px solid var(--ink); }
    .film { position:relative; overflow:hidden; background:var(--fence); }
    .film img { display:block; width:100%; height:100%; object-fit:cover; filter:contrast(.96) saturate(.92); }
    .film::after { content:""; position:absolute; inset:0; pointer-events:none; box-shadow:inset 0 0 80px rgba(20,17,13,.18); }
    .filmGrain::before { content:""; position:absolute; inset:0; pointer-events:none; opacity:.06; mix-blend-mode:overlay; background-image:url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='160' height='160'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/></filter><rect width='100%' height='100%' filter='url(%23n)'/></svg>"); z-index:1; }
    .heroStill { min-height:680px; border-right:1px solid var(--ink); }
    .heroCopy { padding:60px clamp(24px, 4vw, 56px) 48px; display:flex; flex-direction:column; justify-content:space-between; gap:36px; }
    .heroLede { font-size:19px; line-height:1.45; }
    .fieldActions { display:flex; flex-wrap:wrap; gap:12px; margin-top:32px; }
    .btn { display:inline-flex; align-items:center; justify-content:center; gap:8px; min-height:44px; padding:12px 18px; border:1px solid var(--ink); border-radius:0; font-family:var(--body); font-size:14px; font-weight:600; letter-spacing:.01em; text-decoration:none; cursor:pointer; transition:background 140ms ease, color 140ms ease, border-color 140ms ease, transform 140ms ease; }
    .btn:hover { transform:translateY(-1px); }
    .btnPrimary, .btnPrimary:visited, .navCtas .btnPrimary { background:var(--ink); color:var(--bone); border-color:var(--ink); }
    .btnPrimary:hover { background:var(--court-d); border-color:var(--court-d); color:var(--bone); }
    .btnGhost { background:transparent; color:var(--ink); }
    .btnGhost:hover { background:var(--ink); color:var(--bone); }
    .paperBtn { background:var(--paper); color:var(--ink); border-color:var(--paper); }
    .outlinePaperBtn { background:transparent; color:var(--paper); border-color:var(--paper); }
    .terminalSample { margin-top:auto; }
    .terminalSample .eyebrow { margin-bottom:10px; }
    .codeBlock { background:var(--ink); color:var(--paper); font-family:var(--mono); font-size:13px; padding:16px 18px; line-height:1.55; border:1px solid var(--ink); border-radius:0; overflow:auto; white-space:pre-wrap; }
    .codeBlock .cmt { color:var(--sky); }
    .codeBlock .key { color:#c89a8a; }
    .scoreboardSection, .dailySection, .probesSection, .pricingSection, .fieldNotes { padding:clamp(58px, 7vw, 88px) clamp(20px, 4vw, 56px); border-bottom:1px solid var(--ink); }
    .sectionTitleRow { display:flex; justify-content:space-between; align-items:flex-end; gap:28px; margin-bottom:32px; }
    .underLink { font-family:var(--mono); font-size:12px; letter-spacing:.14em; text-transform:uppercase; text-decoration:underline; text-underline-offset:3px; color:var(--ink); white-space:nowrap; }
    .agentGrid { display:grid; grid-template-columns:repeat(3, minmax(0, 1fr)); border:1px solid var(--ink); }
    .agentCard { background:var(--paper); min-width:0; border-right:1px solid var(--ink); }
    .agentCard:last-child { border-right:0; }
    .agentCard header { background:var(--ink); color:var(--bone); display:flex; justify-content:space-between; align-items:center; gap:16px; padding:12px 16px; }
    .agentCard header strong { font-family:var(--display); font-size:17px; font-weight:700; font-stretch:115%; letter-spacing:0; }
    .agentCard header code { color:rgba(236,226,207,.68); font-family:var(--mono); font-size:11px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
    .agentBody { display:grid; grid-template-columns:auto 1fr; align-items:center; gap:20px; padding:22px; }
    .agentScore { display:flex; align-items:baseline; gap:4px; margin:0; font-family:var(--display); font-weight:700; font-stretch:125%; font-size:80px; line-height:1; color:var(--ink); }
    .agentScore span { font-family:var(--mono); font-size:11px; color:var(--ash); }
    .agentStatus { margin:8px 0 0; font-family:var(--mono); font-size:11px; letter-spacing:.14em; text-transform:uppercase; color:var(--court); }
    .agentStatus.ok { color:var(--green); }
    .agentStatus.fail { color:var(--court-d); }
    .miniLabel { margin:0; font-family:var(--mono); font-size:10px; color:var(--ash); letter-spacing:.14em; text-transform:uppercase; }
    .trendBars { height:52px; display:flex; align-items:flex-end; gap:2px; margin-top:8px; }
    .trendBars span { flex:1; min-width:3px; background:var(--ink); }
    .trendBars span:nth-child(4n+1), .trendBars span:nth-child(4n+2) { background:var(--court); }
    .trendBars span:nth-child(5n) { background:var(--sky-d); }
    .findingList { border-top:1px solid var(--hairline); padding:14px 22px 22px; display:grid; gap:8px; }
    .findingList div { display:grid; grid-template-columns:52px 1fr; gap:12px; align-items:baseline; }
    .findingList span { font-family:var(--mono); font-size:10px; letter-spacing:.14em; text-transform:uppercase; color:var(--court); }
    .findingList span.ok { color:var(--green); }
    .findingList span.fail { color:var(--court-d); }
    .findingList p { margin:0; }
    .findingList code { font-family:var(--mono); font-size:12px; color:var(--ink); }
    .findingList small { display:block; margin-top:2px; font-size:12px; color:var(--ash); line-height:1.35; }
    .statRibbon { display:grid; grid-template-columns:repeat(4, 1fr); background:var(--ink); color:var(--bone); border-bottom:1px solid var(--ink); }
    .statRibbon div { padding:30px 28px; border-right:1px solid rgba(236,226,207,.16); }
    .statRibbon div:last-child { border-right:0; }
    .statRibbon strong { display:block; font-family:var(--display); font-weight:700; font-stretch:125%; font-size:56px; line-height:1; letter-spacing:0; color:var(--bone); }
    .statRibbon span { display:block; margin-top:14px; font-family:var(--mono); font-size:10px; line-height:1.45; letter-spacing:.16em; text-transform:uppercase; color:rgba(236,226,207,.66); }
    .dailySection { display:grid; grid-template-columns:minmax(0, 1fr) minmax(320px, 1.1fr); gap:56px; align-items:center; }
    .dailySection hr { border:0; border-top:1px solid var(--hairline); margin:28px 0; }
    .portraitStill { aspect-ratio:4 / 5; border:1px solid var(--ink); min-height:0; }
    .dashList { list-style:none; margin:0; padding:0; display:grid; gap:14px; }
    .dashList li { display:grid; grid-template-columns:auto 1fr; gap:14px; color:var(--fence); line-height:1.45; }
    .dashList li::before { content:"-"; color:var(--court); font-family:var(--mono); letter-spacing:.14em; }
    .probeTable { border-top:1px solid var(--ink); }
    .probeRow { display:grid; grid-template-columns:60px 1.2fr 1.6fr 4fr 80px; gap:16px; align-items:baseline; padding:20px 0; border-bottom:1px solid var(--hairline); }
    .probeRow span:first-child, .probeRow em { color:var(--ash); font-family:var(--mono); font-size:12px; font-style:normal; }
    .probeRow span:nth-child(2) { color:var(--court); font-family:var(--mono); font-size:11px; letter-spacing:.14em; text-transform:uppercase; }
    .probeRow strong { font-family:var(--display); font-weight:600; font-stretch:115%; font-size:22px; color:var(--ink); }
    .probeRow p { margin:0; color:var(--fence); font-size:14px; line-height:1.45; }
    .probeRow em { text-align:right; font-size:10px; letter-spacing:.14em; text-transform:uppercase; }
    .commandSection { display:grid; grid-template-columns:1fr 1fr; gap:64px; align-items:start; background:var(--ink); color:var(--bone); padding:clamp(58px, 7vw, 88px) clamp(20px, 4vw, 56px); border-bottom:1px solid var(--ink); }
    .commandSection h2 { color:var(--bone); }
    .commandSection .eyebrow { color:#c89a8a; }
    .commandSection .bodyText { color:rgba(236,226,207,.72); }
    .stepStack > div { border-top:1px solid rgba(236,226,207,.18); padding:22px 0; }
    .stepStack p { display:flex; align-items:baseline; gap:16px; margin:0; color:var(--bone); }
    .stepStack span { color:rgba(236,226,207,.55); font-family:var(--mono); font-size:11px; letter-spacing:.14em; }
    .stepStack code { color:var(--bone); font-family:var(--mono); font-size:16px; }
    .stepStack small { display:block; margin:8px 0 0 32px; color:rgba(236,226,207,.68); line-height:1.45; }
    .imageTriptych { display:grid; grid-template-columns:repeat(3, 1fr); border-bottom:1px solid var(--ink); }
    .imageTriptych figure { margin:0; position:relative; border-right:1px solid var(--ink); min-width:0; }
    .imageTriptych figure:last-child { border-right:0; }
    .imageTriptych .film { aspect-ratio:1 / 1; }
    .imageTriptych figcaption { position:absolute; left:16px; bottom:16px; color:var(--bone); text-shadow:0 1px 16px rgba(0,0,0,.45); }
    .imageTriptych figcaption span { display:block; font-family:var(--mono); font-size:10px; letter-spacing:.16em; text-transform:uppercase; }
    .imageTriptych figcaption strong { display:block; margin-top:4px; font-family:var(--display); font-weight:600; font-size:18px; color:var(--bone); }
    .pricingIntro { display:grid; grid-template-columns:1fr 1.2fr; gap:56px; align-items:end; margin-bottom:36px; }
    .pricingIntro .bodyText { justify-self:end; max-width:460px; color:var(--ash); }
    .priceTable { display:grid; grid-template-columns:repeat(3, 1fr); border:1px solid var(--ink); }
    .priceCol { padding:32px 28px; border-right:1px solid var(--ink); min-width:0; display:flex; flex-direction:column; background:transparent; }
    .priceCol:last-child { border-right:0; }
    .priceCol.highlight { background:var(--paper); }
    .priceCol .price { margin:10px 0 20px; font-family:var(--display); font-weight:700; font-stretch:125%; font-size:56px; line-height:1; color:var(--ink); }
    .priceCol .price span { margin-left:6px; font-family:var(--mono); font-size:11px; color:var(--ash); }
    .priceCol ul { list-style:none; margin:0 0 24px; padding:0; display:grid; gap:10px; }
    .priceCol li { display:grid; grid-template-columns:auto 1fr; gap:10px; color:var(--fence); font-size:14px; line-height:1.45; }
    .priceCol li::before { content:"-"; color:var(--court); font-family:var(--mono); }
    .priceCol > .btn, .priceCol form { margin-top:auto; }
    .priceCol .btn { width:100%; }
    .checkoutForm { display:grid; gap:8px; }
    .checkoutForm input { min-height:46px; border:1px solid var(--ink); border-radius:0; background:transparent; color:var(--ink); padding:12px 14px; font-family:var(--mono); font-size:13px; width:100%; }
    .checkoutForm button { width:100%; }
    .checkoutStatus { min-height:22px; margin:0; color:var(--ash); font-size:12px; line-height:1.35; }
    .fieldNotes { background:var(--bone); }
    .noteGrid { display:grid; grid-template-columns:repeat(3, 1fr); gap:24px; }
    .noteCard { display:block; padding:22px 22px 24px; border:1px solid var(--ink); background:var(--paper); color:var(--ink); text-decoration:none; min-width:0; }
    .noteCard span { font-family:var(--mono); font-size:11px; color:var(--ash); letter-spacing:.14em; text-transform:uppercase; }
    .noteCard strong { display:block; margin:18px 0 12px; font-family:var(--display); font-weight:700; font-stretch:115%; font-size:24px; line-height:1.08; color:var(--ink); }
    .noteCard p { margin:0; color:var(--fence); font-size:14px; line-height:1.45; }
    .noteCard em { display:block; margin-top:18px; font-family:var(--mono); font-style:normal; font-size:11px; color:var(--court); letter-spacing:.14em; text-transform:uppercase; }
    .closerSection { display:grid; grid-template-columns:1.4fr 1fr; gap:48px; align-items:center; background:var(--court); color:var(--paper); padding:clamp(58px, 7vw, 80px) clamp(20px, 4vw, 56px); border-bottom:1px solid var(--ink); }
    .closerSection h2 { color:var(--paper); font-size:clamp(4rem, 7.4vw, 88px); margin:0; }
    .closerSection .bodyText { color:var(--paper); }
    .reveal { animation:riseIn 520ms cubic-bezier(.16,1,.3,1) both; }
    @keyframes riseIn { from { opacity:0; transform:translateY(18px); } to { opacity:1; transform:translateY(0); } }
    @media (max-width: 860px) {
      .nav { padding:12px 18px; align-items:flex-start; flex-direction:column; } nav { gap:12px; }
      .courtHero { min-height:auto; display:block; padding:36px 0 28px; }
      .courtHero::after { right:-44px; bottom:128px; width:74%; }
      .heroArt { position:absolute; right:-120px; left:auto; width:112%; opacity:.18; }
      .heroText { width:calc(100% - 36px); padding:36px 18px 34px; margin:0 18px; }
      .proofStrip { position:relative; left:auto; right:auto; bottom:auto; margin:28px 18px 0; grid-template-columns:1fr; }
      .proofStrip span + span { border-left:0; border-top:3px solid var(--line); }
      h1 { font-size:3.4rem; }
      h2, .doc h1 { font-size:2.4rem; }
      .lede { font-size:1.15rem; max-width:390px; }
      .two, .metricStrip, .steps, .priceGrid, .scoreRow, .probeGrid, .docGrid, .imageBand, .pricing, .blogGrid { grid-template-columns:1fr; }
      .metricStrip div + div { border-left:0; border-top:3px solid var(--line); }
      .imageStack img:nth-child(2) { margin-left:0; width:100%; }
      .imageBand img, .imageBand img:nth-child(2) { height:auto; aspect-ratio:1 / 1; }
      .band { padding:46px 18px; }
      .dashHead { padding:36px 18px 18px; display:block; }
      .dashHead h1 { font-size:2rem; }
      .dashboard > .productFrame, .productFrame { width:auto; margin:0 18px; min-height:0; }
      .nav { grid-template-columns:1fr; align-items:start; gap:14px; padding:14px 18px; position:static; }
      .navLinks { justify-content:flex-start; flex-wrap:wrap; gap:12px 18px; }
      .navCtas { justify-content:flex-start; flex-wrap:wrap; }
      footer { grid-template-columns:1fr; align-items:start; gap:14px; }
      .fieldHero, .dailySection, .commandSection, .pricingIntro, .priceTable, .noteGrid, .closerSection { grid-template-columns:1fr; }
      .fieldHero { min-height:0; }
      .heroStill { min-height:420px; border-right:0; border-bottom:1px solid var(--ink); }
      .heroCopy { padding:42px 20px; }
      .fieldLanding h1 { font-size:clamp(3.3rem, 15vw, 4.8rem); }
      .fieldLanding h2, .closerSection h2 { font-size:clamp(2.6rem, 12vw, 4rem); }
      .sectionTitleRow { display:block; }
      .underLink { display:inline-block; margin-top:12px; white-space:normal; }
      .agentGrid, .statRibbon, .imageTriptych { grid-template-columns:1fr; }
      .agentCard, .statRibbon div, .imageTriptych figure, .priceCol { border-right:0; border-bottom:1px solid var(--ink); }
      .agentCard:last-child, .statRibbon div:last-child, .imageTriptych figure:last-child, .priceCol:last-child { border-bottom:0; }
      .agentBody { grid-template-columns:1fr; }
      .probeRow { grid-template-columns:44px 1fr; gap:8px 14px; }
      .probeRow strong, .probeRow p { grid-column:2; }
      .probeRow em { display:none; }
      .portraitStill { aspect-ratio:1 / 1; }
      .pricingIntro .bodyText { justify-self:start; }
    }
    @media (max-width: 520px) {
      body { font-size:16px; }
      h1 { font-size:2.45rem; }
      h2, .doc h1 { font-size:2rem; }
      h3 { font-size:1.25rem; }
      .actions, .actions .button, .checkoutForm .button { width:100%; }
      .button { width:100%; }
      .fieldActions, .fieldActions .btn, .priceCol .btn, .checkoutForm .btn { width:100%; }
      .agentScore { font-size:64px; }
      .statRibbon strong, .priceCol .price { font-size:44px; }
    }
    @media (prefers-reduced-motion: reduce) {
      *, *::before, *::after { animation-duration:.01ms !important; animation-iteration-count:1 !important; transition-duration:.01ms !important; scroll-behavior:auto !important; }
    }
  `;
}

function softwareJsonLD(env: Env): string {
  return `<script type="application/ld+json">${JSON.stringify({
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "Baseline.ai",
    applicationCategory: "DeveloperApplication",
    operatingSystem: "macOS, Linux",
    offers: [{ "@type": "Offer", price: "0", priceCurrency: "USD" }, { "@type": "Offer", price: "39", priceCurrency: "USD" }],
    url: baseURL(env)
  })}</script>`;
}

function sitemap(origin: string): string {
  return `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>${origin}/</loc></url><url><loc>${origin}/dashboard</loc></url><url><loc>${origin}/docs/mcp</loc></url><url><loc>${origin}/blog</loc></url><url><loc>${origin}/checkout/success</loc></url><url><loc>${origin}/checkout/cancel</loc></url></urlset>`;
}

function html(body: string, status = 200): Response {
  return new Response(body, { status, headers: { "content-type": "text/html; charset=utf-8" } });
}

function text(body: string, contentType = "text/plain; charset=utf-8"): Response {
  return new Response(body, { headers: { "content-type": contentType } });
}

function json(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), { status, headers: { "content-type": "application/json; charset=utf-8" } });
}

function escapeHTML(value: string): string {
  return value.replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch] || ch));
}

function baseURL(env: Env, request?: Request): string {
  if (env.APP_URL) return env.APP_URL.replace(/\/$/, "");
  if (request) return new URL(request.url).origin;
  return "https://trackbaseline.com";
}

function hasStripe(env: Env): boolean {
  return Boolean(env.STRIPE_PAYMENT_LINK_PRO || env.STRIPE_PAYMENT_LINK_TEAM || (env.STRIPE_SECRET_KEY && (env.STRIPE_PRICE_ID_PRO || env.STRIPE_PRICE_ID_TEAM)));
}
