# feat: Baseline Next Beads

## Problem Frame

Baseline v0 is live and dogfoodable, but the recommended changes are scattered across Proconsult, OpenProse run receipts, validation notes, and launch evidence. This plan consolidates them into minimal beads that preserve the product wedge:

> Local known-good drift checker for coding-agent workstations, with optional redacted cloud history.

Do not expand into generic LLM observability. Do not build external alert delivery, public benchmarks, browser probes, or broad eval tooling until the local known-good loop and paid pilot are trusted.

## Source Inputs

- `PROCONSULT_LAUNCH_ARCHITECTURE.md`
- `PROCONSULT_BASELINE_V0.md`
- `BASELINE_V0_SHAPE.md`
- `docs/OPENPROSE_RUN_RESULTS.md`
- `docs/VALIDATION.md`
- `docs/SKILL_EXECUTION_EVIDENCE.md`
- Current code under `cmd/baseline`, `internal/baseline`, and `web/src/index.ts`

## Bead Rules

Every bead is one commit. Each implementation bead needs a non-trivial fail-first test or smoke check. Push remains blocked until a remote is configured.

Use this sequencing unless a user explicitly reprioritizes. Earlier beads protect trust; later beads monetize or broaden.

## Bead 8 — Consolidate Recommended Changes

Status: current planning bead.

Scope:
- Create this bead plan.
- Update continuity so the next bead is unambiguous.

Acceptance:
- Future beads are ordered, scoped, and testable.
- Deferrals and kill criteria are explicit.

Files:
- `docs/plans/2026-05-14-001-feat-baseline-next-beads-plan.md`
- `CONTINUITY.md`

Tests:
- Docs-only verification: no code tests required, but run `git status` and preserve clean commit.

## Bead 9 — Activate Stripe Pro Checkout

Why now:
Offer smoke test is blocked on payment. Without paid intent, more SaaS build is noise.

Scope:
- Add Stripe webhook endpoint.
- Verify webhook signatures.
- Store subscription/entitlement state in Neon.
- Keep success redirect non-authoritative.
- Support Pro only first. Team/Agency remains copy or disabled CTA.

Files:
- `web/src/index.ts`
- `web/schema.sql`
- `README.md`
- `docs/VALIDATION.md`

Fail-first tests:
- Checkout without Stripe secrets returns `503`.
- Fake success redirect does not grant entitlement.
- Invalid webhook signature returns `400` or `401`.
- Valid mocked `checkout.session.completed` / `customer.subscription.*` event updates entitlement.

Acceptance:
- `/api/checkout?plan=pro` creates a Stripe Checkout Session when secrets are present.
- Paid access is granted only from verified webhook.
- `curl /api/health` reports `stripe:true` only when configured.

Blocker:
- Requires `STRIPE_SECRET_KEY`, Pro price ID or payment link, and webhook secret.

## Bead 10 — Real API Tokens and Workspaces

Why:
Current Worker uses one global `BASELINE_API_TOKEN`. That is fine for dogfood, not for pilots.

Scope:
- Add `accounts`, `workspaces`, and `api_tokens` tables.
- Store only token prefix and HMAC hash.
- Add token creation/revocation endpoints.
- Add token scopes: `ingest:write`, `reports:read`, future `alerts:write`.
- Update CLI docs for token setup.

Files:
- `web/schema.sql`
- `web/src/index.ts`
- `internal/baseline/paths.go`
- `internal/baseline/cli.go`
- `README.md`

Fail-first tests:
- Unknown token cannot ingest.
- Revoked token cannot ingest.
- Token without `ingest:write` cannot ingest.
- Raw token is never stored in Neon payload.

Acceptance:
- Dogfood token can be rotated without changing Worker global secret.
- `/api/runs` resolves workspace from token, not caller-submitted workspace text.

## Bead 11 — App-Level 14-Day Retention

Why:
The landing page promises 14 days of free history. Neon retention is not product retention.

Scope:
- Add `expires_at` to cloud run/event tables.
- Set free runs to 14-day expiry.
- Add paid retention duration field.
- Add Cloudflare scheduled handler to purge expired runs/events.
- Add retention status to dashboard.

Files:
- `web/schema.sql`
- `web/wrangler.jsonc`
- `web/src/index.ts`
- `README.md`

Fail-first tests:
- Run older than `expires_at` is purged by scheduled handler.
- Paid workspace with longer retention is not purged early.
- New free ingest receives `expires_at = created_at + 14 days`.

Acceptance:
- The 14-day promise is enforced by app code.
- Dashboard does not imply longer free history.

## Bead 12 — Local Sync Outbox

Why:
Proconsult was explicit: cloud sync should read only scrubbed payloads already staged in an outbox.

Scope:
- Add local SQLite `sync_outbox`.
- Stage reduced `CloudRunPayload` after scrub gate.
- Retry with backoff.
- Mark synced/failed state.
- Keep raw run records local only.

Files:
- `internal/baseline/db.go`
- `internal/baseline/run.go`
- `internal/baseline/cli.go`
- `internal/baseline/*_test.go`

Fail-first tests:
- Cloud sync failure leaves a pending outbox row.
- Retry succeeds and marks row synced.
- Outbox payload never contains local absolute workspace path or raw findings.

Acceptance:
- `baseline sync status` reports pending/synced/failed counts.
- Direct sync path does not bypass scrubbed payload staging.

## Bead 13 — MCP Safety Hardening

Why:
Current MCP can mark known-good and mutate config directly. Proconsult flagged this as a footgun.

Scope:
- Make MCP mostly read-only.
- Change `baseline_mark_known_good` into pending confirmation output.
- Add `baseline known-good confirm <pending_id>` CLI command.
- Make MCP config changes return suggested CLI commands, not mutate config.
- Scrub tool outputs before storage and before MCP response.
- Add MCP annotations where supported, but do not rely on them for safety.

Files:
- `internal/baseline/mcp.go`
- `internal/baseline/cli.go`
- `internal/baseline/db.go`
- `internal/baseline/scrubber.go`
- `internal/baseline/mcp_test.go`

Fail-first tests:
- MCP mark-known-good does not alter `known_goods`.
- CLI confirm does alter `known_goods`.
- MCP config call does not expose or mutate API token.
- Scrubber runs on MCP response payload.

Acceptance:
- A degraded agent cannot bless itself as known-good through MCP alone.

## Bead 14 — OpenClaw Runner Pack

Why:
Generic MCP cannot evaluate host model behavior. OpenClaw is the first real runner.

Scope:
- Implement explicit OpenClaw runner adapter.
- Keep behavior probes prompt-only or tool-disabled where possible.
- Add timeouts and cost/runtime guardrails.
- Capture 12-question pack metrics: identity, active task, repo awareness, safety constraint, tool awareness, dedup, latency, output acceptance, stuck rate, tone.
- Keep `--run-agent` opt-in.

Files:
- `internal/baseline/run.go`
- `internal/baseline/cli.go`
- new `internal/baseline/openclaw_runner.go`
- `internal/baseline/run_test.go`

Fail-first tests:
- `baseline check --full` skips agent execution without opt-in.
- `baseline check --full --run-agent` calls runner with timeout.
- Runner failure produces finding, not silent success.
- Runner output is scrubbed before hashing/sync.

Acceptance:
- Full run produces timed question results when explicitly enabled.
- Local fast path remains under the same safety default.

## Bead 15 — MCP Tool/Schema Drift With Pagination

Why:
MCP `tools/list` can paginate. One-page hashing is not defensible.

Scope:
- Add MCP client inspection helper.
- Follow `nextCursor` until exhausted.
- Hash tool names, descriptions, input schemas, and annotations separately.
- Record missing/added/changed tools as observations.

Files:
- new `internal/baseline/mcpclient.go`
- `internal/baseline/run.go`
- `internal/baseline/db.go`
- tests under `internal/baseline`

Fail-first tests:
- Mock MCP server returns two pages; hash includes both.
- Description-only change is detected separately from schema change.
- Pagination loop stops on empty cursor and handles repeated cursor defensively.

Acceptance:
- Known-good diff can say exactly which MCP tool/schema changed.

## Bead 16 — Local Scheduled Runs

Why:
The product promise is daily health checks. Cloud alerts can wait, but local scheduling is core.

Scope:
- Add `baseline schedule install|status|remove`.
- Prefer launchd on macOS first, with cron fallback docs.
- Scheduled run executes `baseline check --fast`.
- Write local report and optionally sync if enabled.

Files:
- `internal/baseline/cli.go`
- new `internal/baseline/schedule.go`
- `README.md`
- `/docs` install docs

Fail-first tests:
- Generated launchd plist contains absolute binary path.
- Schedule status detects installed/missing job.
- Remove is idempotent.

Acceptance:
- User can install a daily local baseline check in one command.

## Bead 17 — Local Alert Preview

Why:
External alerts are explicitly deferred, but local alert summaries are needed to learn false-positive rate.

Scope:
- Add local alert rules for score drop, latency spike, missing MCP tools, scrub failures, and known-good diffs.
- Add `baseline alerts preview`.
- Add user feedback command: `baseline alerts rate <finding_id> noisy|useful`.
- Store alert judgments locally.

Files:
- `internal/baseline/db.go`
- `internal/baseline/cli.go`
- new `internal/baseline/alerts.go`
- tests

Fail-first tests:
- Score drop generates local alert.
- No change generates no alert.
- Alert judgment is persisted.

Acceptance:
- False/noisy alert rate can be measured before adding Slack/email/GitHub delivery.

## Bead 18 — Dashboard Reads Real Data

Why:
The visual dashboard is currently demo-like. The next trust step is showing synced runs.

Scope:
- Add `/api/runs/latest` and `/api/runs/timeline`.
- Render dashboard from Neon data when token/session is available.
- Keep demo fallback for anonymous page.
- Show retention status and checkout state.

Files:
- `web/src/index.ts`
- `web/schema.sql`
- `README.md`

Fail-first tests:
- Empty workspace shows empty state, not fake real data.
- Ingested run appears on timeline.
- Expired runs do not appear after purge.

Acceptance:
- Dogfood workspace sees real run history on the dashboard.

## Bead 19 — Install and Release Packaging

Why:
The validation test requires install under five minutes.

Scope:
- Add version command.
- Add install script or Homebrew tap plan.
- Produce checksums for release binaries.
- Update MCP docs to avoid source-build-only install.

Files:
- `cmd/baseline/main.go`
- `internal/baseline/cli.go`
- `README.md`
- `web/src/index.ts`
- release docs

Fail-first tests:
- `baseline version` returns commit/version.
- Install docs use absolute binary path for MCP registration.
- Fresh temp home can init, install, check, mark known-good, compare.

Acceptance:
- New user can install and run first baseline in under five minutes.

## Bead 20 — OpenProse Contract Migration

Why:
The attached recipe files ran only in compatibility mode. Strict OpenProse 0.13.1 expects `kind: service` or `kind: system`.

Scope:
- Decide whether to migrate copies in this repo or upstream skills-library.
- Create current Contract Markdown wrappers for the five recipes.
- Add strict-run/lint receipts.
- Preserve compatibility note for legacy recipe files.

Files:
- new `openprose/` or `docs/openprose/` contract files
- `docs/OPENPROSE_RUN_RESULTS.md`
- possibly upstream `/Users/future/.openclaw/workspace/repos/skills-library/recipes/*` only if explicitly approved

Fail-first tests:
- Strict runner rejects legacy no-`kind` file.
- Strict runner accepts migrated `kind: service` wrapper.
- Output contract matches old compatibility result.

Acceptance:
- Future `prose run <file.prose.md>` has a strict path that does not rely on compatibility mode.

## Bead 21 — 10-User Paid Pilot

Why:
Do not build deeper SaaS features until willingness to pay is real.

Scope:
- Create pilot script and outreach tracker.
- Track install success, day-two return, checkout/payment intent, and requested paid features.
- Add a dashboard/admin view or CSV export for pilot signals.
- Do not add product scope unless signal clears thresholds.

Files:
- `docs/VALIDATION.md`
- new `docs/PILOT.md`
- optional `web/src/index.ts` for export/admin if needed

Fail-first tests:
- Pilot checklist cannot mark PASS without required counts.
- Manual validation record includes install, day-two return, and payment/alert/retention request.

Acceptance:
- 10 qualified users asked.
- Pass requires 5 installs, 3 day-two returns, 2 requests for alerts or longer retention, and at least 2 payment signals.

## Bead 22 — Package Boundary Refactor

Why:
The current Go code is intentionally compact. Refactor only after behavior stabilizes enough to justify boundaries.

Scope:
- Split current `internal/baseline` into app/domain/store/checks/sync/mcpserver/scrub/platform packages.
- Preserve CLI behavior.
- Avoid feature changes.

Files:
- `internal/**`
- tests

Fail-first tests:
- Golden CLI output tests around `check`, `latest`, `report`, `compare`.
- MCP tools list remains compatible.
- Cloud payload stays reduced/redacted.

Acceptance:
- Same behavior, clearer trust boundaries.

## Explicit Deferrals

Do not schedule these until Bead 21 passes:

- Slack/GitHub/email alert delivery
- Public anonymized benchmarks
- Browser probes
- Workflow execution from cloud
- Team/Agency billing
- Usage billing
- Auto-known-good
- Raw transcript cloud storage
- Broad "LLM observability" trace dashboard

## Recommended Next Move

Bead 9 is the next highest-leverage bead if Stripe credentials/payment links are available.

If Stripe is still blocked, do Bead 12 or Bead 13 first. Those improve trust without expanding the product.
