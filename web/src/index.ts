import { neon, type NeonQueryFunction } from "@neondatabase/serverless";

interface Env {
  DATABASE_URL?: string;
  STRIPE_SECRET_KEY?: string;
  STRIPE_PRICE_ID_PRO?: string;
  STRIPE_PRICE_ID_TEAM?: string;
  STRIPE_PAYMENT_LINK_PRO?: string;
  STRIPE_PAYMENT_LINK_TEAM?: string;
  APP_URL?: string;
  BASELINE_API_TOKEN?: string;
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

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const read = request.method === "GET" || request.method === "HEAD";
    try {
      if (read && url.pathname === "/") return html(landingPage(env));
      if (read && url.pathname === "/dashboard") return html(dashboardPage(env));
      if (read && url.pathname === "/docs/mcp") return html(mcpDocsPage(env));
      if (read && url.pathname === "/privacy") return html(privacyPage(env));
      if (read && url.pathname === "/terms") return html(termsPage(env));
	      if (read && url.pathname === "/robots.txt") return text("User-agent: *\nAllow: /\nSitemap: " + baseURL(env, request) + "/sitemap.xml\n");
	      if (read && url.pathname === "/sitemap.xml") return text(sitemap(baseURL(env, request)), "application/xml");
	      if (read && url.pathname === "/api/health") return json({ ok: true, db: Boolean(env.DATABASE_URL), stripe: hasStripe(env), token_required: Boolean(env.BASELINE_API_TOKEN) });
	      if (read && url.pathname === "/api/runs/latest") return latestRun(request, env);
	      if (read && url.pathname === "/api/runs/timeline") return runTimeline(env);
	      if (request.method === "POST" && url.pathname === "/api/runs") return ingestRun(request, env);
      if (request.method === "POST" && url.pathname === "/api/events") {
        ctx.waitUntil(recordEvent(request, env, url.pathname));
        return json({ ok: true });
      }
      if ((request.method === "GET" || request.method === "POST") && url.pathname === "/api/checkout") return checkout(request, env);
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

async function ingestRun(request: Request, env: Env): Promise<Response> {
  const auth = request.headers.get("authorization") || "";
  if (!auth.startsWith("Bearer ")) return json({ ok: false, error: "missing bearer token" }, 401);
  if (!env.BASELINE_API_TOKEN) return json({ ok: false, error: "BASELINE_API_TOKEN is not configured" }, 503);
  if (auth.slice("Bearer ".length) !== env.BASELINE_API_TOKEN) return json({ ok: false, error: "invalid bearer token" }, 403);
  const payload = await request.json<RunPayload>();
  if (!payload.run_id) return json({ ok: false, error: "run_id required" }, 400);
  if (!env.DATABASE_URL) return json({ ok: false, error: "DATABASE_URL is not configured" }, 503);
  const sql = neon(env.DATABASE_URL);
  await ensureSchema(sql);
  await sql`
    insert into baseline_runs (id, workspace, agent_kind, status, health_score, mode, payload)
    values (${payload.run_id}, ${payload.workspace || "unknown"}, ${payload.agent_kind || "unknown"}, ${payload.status || "unknown"}, ${payload.health_score || 0}, ${payload.mode || "unknown"}, ${JSON.stringify(payload)}::jsonb)
    on conflict (id) do update set
      workspace = excluded.workspace,
      agent_kind = excluded.agent_kind,
      status = excluded.status,
      health_score = excluded.health_score,
      mode = excluded.mode,
      payload = excluded.payload
  `;
  return json({ ok: true });
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

async function checkout(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const plan = url.searchParams.get("plan") === "team" ? "team" : "pro";
  const paymentLink = plan === "team" ? env.STRIPE_PAYMENT_LINK_TEAM : env.STRIPE_PAYMENT_LINK_PRO;
  if (paymentLink) return Response.redirect(paymentLink, 303);
  if (!env.STRIPE_SECRET_KEY) {
    return json({ ok: false, error: "Stripe is not configured. Set STRIPE_SECRET_KEY and STRIPE_PRICE_ID_PRO/TEAM or payment links." }, 503);
  }
  const price = plan === "team" ? env.STRIPE_PRICE_ID_TEAM : env.STRIPE_PRICE_ID_PRO;
  if (!price) return json({ ok: false, error: "Stripe price id is not configured for " + plan }, 503);
  const origin = baseURL(env, request);
  const body = new URLSearchParams({
    mode: "subscription",
    success_url: origin + "/dashboard?checkout=success",
    cancel_url: origin + "/?checkout=cancelled",
    "line_items[0][price]": price,
    "line_items[0][quantity]": "1"
  });
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
  return Response.redirect(session.url, 303);
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
}

function landingPage(env: Env): string {
  return layout(env, "Baseline.ai | Agent health checks before work gets expensive", `
    <main>
      <section class="hero">
        <div class="heroBackdrop" aria-hidden="true">${dashboardVisual()}</div>
        <div class="heroText">
          <p class="eyebrow">Local-first drift monitor for coding agents</p>
          <h1>Baseline.ai</h1>
          <p class="lede">Know when OpenClaw, Hermes, Codex, Claude Code, or a local agent got slower, forgetful, unaware, or strange before it burns a client job.</p>
          <div class="actions">
            <a class="button primary" href="/docs/mcp">Install MCP</a>
            <a class="button secondary" href="/api/checkout?plan=pro">Start Pro</a>
          </div>
          <p class="fine">Free local CLI. 14 days cloud history. No raw prompts exported unless you explicitly enable it.</p>
        </div>
      </section>

      <section class="band tight">
        <div class="metricStrip">
          <div><strong>5s vs 60s</strong><span>Latency and context bloat are the pain users notice first.</span></div>
          <div><strong>60% to 91%</strong><span>Output acceptance rate is the metric serious agent users already track.</span></div>
          <div><strong>10-15</strong><span>Timed baseline questions plus local repo, MCP, tool, and scrub checks.</span></div>
        </div>
      </section>

      <section class="band two">
        <div>
          <h2>The wedge</h2>
          <p>Baseline is not another generic LLM observability product. It is a daily health check for local coding-agent workstations: memory, tools, MCP state, repo awareness, speed, style, and known-good drift.</p>
          <ul class="checks">
            <li>Detects OpenClaw MCP registration changes and local tool visibility.</li>
            <li>Times the agent on simple memory, repo, safety, style, and reliability prompts.</li>
            <li>Hashes observations so cloud sync can compare state without raw prompt export.</li>
          </ul>
        </div>
        <div class="panel">
          <h3>Daily alert</h3>
          <div class="alert warning">Memory identity changed from accepted baseline.</div>
          <div class="alert ok">Scrubber passed synthetic secret test.</div>
          <div class="alert bad">p95 agent response +63% versus known-good.</div>
        </div>
      </section>

      <section class="band">
        <h2>Three minute setup</h2>
        <div class="steps">
          <div><span>1</span><strong>Install</strong><code>go install github.com/future/baseline/cmd/baseline@latest</code></div>
          <div><span>2</span><strong>Register</strong><code>baseline init --register-openclaw</code></div>
          <div><span>3</span><strong>Check</strong><code>baseline check --full --run-agent</code></div>
        </div>
      </section>

      <section class="band pricing">
        <div>
          <h2>Pricing hypothesis</h2>
          <p>Local stays free. People pay when drift becomes team risk, client risk, or procurement evidence.</p>
        </div>
        <div class="priceGrid">
          <article>
            <h3>Local</h3>
            <p class="price">$0</p>
            <p>SQLite, MCP, scrub preview, known-good compare.</p>
            <a class="button secondary" href="/docs/mcp">Install</a>
          </article>
          <article>
            <h3>Pro</h3>
            <p class="price">$39/mo</p>
            <p>14+ day history, alerts, summaries, private custom probes.</p>
            <a class="button primary" href="/api/checkout?plan=pro">Buy Pro</a>
          </article>
          <article>
            <h3>Team</h3>
            <p class="price">$129/mo</p>
            <p>Shared dashboards, API tokens, anonymized benchmarks, webhooks.</p>
            <a class="button primary" href="/api/checkout?plan=team">Buy Team</a>
          </article>
        </div>
      </section>
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

function mcpDocsPage(env: Env): string {
  const install = `go build -o bin/baseline ./cmd/baseline
./bin/baseline init
./bin/baseline install openclaw
openclaw mcp list
./bin/baseline check --fast
./bin/baseline check --full --run-agent`;
  return layout(env, "Baseline MCP installation", `
    <main class="doc">
      <p class="eyebrow">MCP installation</p>
      <h1>Install Baseline into OpenClaw</h1>
      <p>Baseline exposes seven legible MCP tools: check, latest, report, compare, mark known-good, config, and scrub preview. Fast checks are local-only. Full checks execute the agent only with explicit opt-in.</p>
      <pre><code>${escapeHTML(install)}</code></pre>
      <h2>Cloud sync</h2>
      <pre><code>baseline sync on --url ${escapeHTML(baseURL(env))} --token YOUR_BASELINE_TOKEN
baseline check --fast
baseline sync push</code></pre>
      <h2>Safety model</h2>
      <p>The MCP can read what the connected agent gives it. Baseline defaults to local SQLite and redacted summaries. Raw outputs are not exported unless <code>allow_raw_output</code> is enabled in <code>~/.baseline/config.json</code>.</p>
      <h2>Recommended first known-good</h2>
      <pre><code>baseline check --fast
baseline known-good mark --label clean-local
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

function layout(env: Env, title: string, body: string, structuredData = ""): string {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHTML(title)}</title>
  <meta name="description" content="Local-first daily baseline checks and drift monitoring for coding agents, MCP tools, repo awareness, memory, latency, and style.">
  <meta property="og:title" content="${escapeHTML(title)}">
  <meta property="og:description" content="Know when your coding agent got worse, slower, forgetful, unaware, or weird before it costs you work.">
  <meta property="og:type" content="website">
  <style>${css()}</style>
  ${structuredData}
</head>
<body>
  <header class="nav"><a href="/" class="brand">Baseline.ai</a><nav><a href="/dashboard">Dashboard</a><a href="/docs/mcp">MCP</a><a href="/api/checkout?plan=pro">Pricing</a></nav></header>
  ${body}
  <footer><span>Baseline.ai</span><a href="/privacy">Privacy</a><a href="/terms">Terms</a></footer>
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
    :root { color-scheme: light; --ink:#111214; --muted:#5a6472; --line:#d9dee6; --paper:#f7f8fa; --white:#ffffff; --blue:#1f6feb; --green:#16794a; --amber:#a15c00; --red:#b42318; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color:var(--ink); background:var(--white); letter-spacing:0; }
    a { color:inherit; text-decoration:none; }
    .nav { height:64px; display:flex; align-items:center; justify-content:space-between; padding:0 28px; border-bottom:1px solid var(--line); background:rgba(255,255,255,.94); position:sticky; top:0; z-index:20; }
    .brand { font-weight:800; }
    nav { display:flex; gap:22px; font-size:14px; color:var(--muted); }
    .hero { min-height:calc(100vh - 64px); position:relative; display:flex; align-items:center; overflow:hidden; background:#eef2f7; border-bottom:1px solid var(--line); }
    .heroBackdrop { position:absolute; inset:42px auto auto 54%; width:min(740px, 43vw); opacity:.95; transform:scale(1.02); transform-origin:top right; }
    .heroBackdrop .productFrame { width:100%; }
    .heroText { position:relative; z-index:2; width:min(620px, 92vw); padding:72px 28px 96px; margin-left:max(28px, calc((100vw - 1180px) / 2)); }
    .eyebrow { margin:0 0 14px; color:var(--blue); font-size:13px; font-weight:800; text-transform:uppercase; letter-spacing:.08em; }
    h1 { margin:0 0 18px; font-size:clamp(44px, 8vw, 96px); line-height:.94; letter-spacing:0; }
    h2 { font-size:34px; line-height:1.08; margin:0 0 18px; letter-spacing:0; }
    h3 { margin:0 0 12px; font-size:18px; }
    .lede { max-width:670px; font-size:22px; line-height:1.35; color:#22272e; margin:0 0 28px; }
    .fine { color:var(--muted); font-size:14px; margin-top:16px; }
    .actions { display:flex; gap:12px; flex-wrap:wrap; }
    .button { min-height:44px; display:inline-flex; align-items:center; justify-content:center; border-radius:8px; padding:0 18px; font-weight:800; border:1px solid var(--line); }
    .primary { background:var(--ink); color:var(--white); border-color:var(--ink); }
    .secondary { background:var(--white); color:var(--ink); }
    .band { padding:78px max(28px, calc((100vw - 1180px) / 2)); }
    .tight { padding-top:24px; padding-bottom:24px; }
    .two { display:grid; grid-template-columns:minmax(0, 1fr) minmax(320px, 460px); gap:42px; align-items:start; }
    .metricStrip { display:grid; grid-template-columns:repeat(3, 1fr); gap:1px; background:var(--line); border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    .metricStrip div { background:var(--white); padding:20px; }
    .metricStrip strong { display:block; font-size:26px; margin-bottom:6px; }
    .metricStrip span, p, li { color:var(--muted); line-height:1.55; }
    .checks { padding-left:18px; }
    .panel, .priceGrid article { border:1px solid var(--line); border-radius:8px; padding:22px; background:var(--white); }
    .alert { border-left:4px solid var(--line); padding:12px 14px; margin:10px 0; background:var(--paper); border-radius:6px; font-weight:700; }
    .alert.ok { border-color:var(--green); }
    .alert.warning { border-color:var(--amber); }
    .alert.bad { border-color:var(--red); }
    .steps, .priceGrid { display:grid; grid-template-columns:repeat(3, 1fr); gap:16px; }
    .steps div { border:1px solid var(--line); border-radius:8px; padding:18px; background:var(--paper); min-width:0; }
    .steps span { display:inline-flex; width:28px; height:28px; align-items:center; justify-content:center; background:var(--ink); color:var(--white); border-radius:50%; font-weight:800; margin-bottom:18px; }
    .steps strong { display:block; margin-bottom:8px; }
    code, pre { font-family:"SFMono-Regular", Consolas, monospace; }
    code { overflow-wrap:anywhere; }
    pre { background:#101317; color:#f5f7fb; border-radius:8px; padding:18px; overflow:auto; line-height:1.5; }
    .pricing { background:var(--paper); border-top:1px solid var(--line); border-bottom:1px solid var(--line); }
    .price { font-size:32px; color:var(--ink); font-weight:900; margin:0 0 10px; }
    .dashboard { background:var(--paper); min-height:100vh; }
    .dashHead { padding:48px max(28px, calc((100vw - 1180px) / 2)) 18px; display:flex; justify-content:space-between; gap:24px; align-items:end; }
    .dashHead h1 { font-size:42px; max-width:760px; line-height:1.05; }
    .dashboard > .productFrame { margin:0 max(28px, calc((100vw - 1180px) / 2)); }
    .productFrame { width:min(900px, 92vw); min-height:430px; border:1px solid #c8d0db; border-radius:8px; background:#fbfcfe; box-shadow:0 24px 70px rgba(17,18,20,.16); overflow:hidden; }
    .frameTop { height:48px; display:flex; align-items:center; gap:12px; padding:0 18px; border-bottom:1px solid var(--line); background:#fff; }
    .frameTop span { width:10px; height:10px; border-radius:50%; background:var(--red); box-shadow:18px 0 var(--amber), 36px 0 var(--green); margin-right:42px; }
    .frameTop em { margin-left:auto; font-style:normal; color:var(--green); font-weight:800; }
    .scoreRow { display:grid; grid-template-columns:170px 1fr 220px; gap:22px; padding:28px; align-items:center; }
    .scoreBlock { border:1px solid var(--line); border-radius:8px; padding:20px; text-align:center; background:#fff; }
    .scoreBlock b { display:block; font-size:68px; line-height:1; }
    .scoreBlock span { color:var(--muted); font-weight:800; }
    .miniBars { height:190px; display:flex; align-items:end; gap:12px; border:1px solid var(--line); border-radius:8px; padding:18px; background:#fff; }
    .miniBars i { display:block; flex:1; min-width:18px; background:var(--blue); border-radius:5px 5px 0 0; }
    .signalList { border:1px solid var(--line); border-radius:8px; background:#fff; padding:16px; }
    .signalList p { margin:12px 0; color:var(--ink); font-weight:700; }
    .dot { display:inline-block; width:9px; height:9px; border-radius:50%; margin-right:9px; }
    .okDot { background:var(--green); } .warnDot { background:var(--amber); } .badDot { background:var(--red); }
    .probeGrid { display:grid; grid-template-columns:repeat(4, 1fr); gap:12px; padding:0 28px 28px; }
    .probeGrid div { border:1px solid var(--line); background:#fff; border-radius:8px; padding:14px; }
    .probeGrid strong { display:block; margin-bottom:8px; }
    .probeGrid span { color:var(--muted); font-weight:800; }
    .doc { max-width:860px; margin:0 auto; padding:68px 28px; }
    table { width:100%; border-collapse:collapse; }
    th, td { text-align:left; border-bottom:1px solid var(--line); padding:10px 0; }
    footer { display:flex; gap:20px; padding:30px 28px; color:var(--muted); border-top:1px solid var(--line); }
    @media (max-width: 860px) {
      .nav { padding:0 18px; } nav { gap:12px; }
      .hero { align-items:flex-start; }
      .heroBackdrop { position:relative; inset:auto; width:auto; transform:none; order:2; margin:0 18px 36px; opacity:1; }
      .heroText { padding:46px 18px 24px; margin-left:0; }
      .hero { flex-direction:column; }
      .lede { font-size:18px; }
      .two, .metricStrip, .steps, .priceGrid, .scoreRow, .probeGrid { grid-template-columns:1fr; }
      .band { padding:46px 18px; }
      .dashHead { padding:36px 18px 18px; display:block; }
      .dashHead h1 { font-size:32px; }
      .dashboard > .productFrame, .productFrame { width:auto; margin:0 18px; min-height:0; }
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
  return `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>${origin}/</loc></url><url><loc>${origin}/dashboard</loc></url><url><loc>${origin}/docs/mcp</loc></url></urlset>`;
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
  return "https://baseline-ai.workers.dev";
}

function hasStripe(env: Env): boolean {
  return Boolean(env.STRIPE_PAYMENT_LINK_PRO || env.STRIPE_PAYMENT_LINK_TEAM || (env.STRIPE_SECRET_KEY && (env.STRIPE_PRICE_ID_PRO || env.STRIPE_PRICE_ID_TEAM)));
}
