# AGENTS.md - Baseline.ai

## 1. Mission (North Star)
Build Baseline.ai as a local-first agent workstation monitor for coding agents. Goals:
1. Keep `baseline setup`, `baseline run`, `baseline report`, and `baseline accept` reliable for local OpenClaw-style workflows.
2. Detect drift in memory, repo awareness, MCP/tool visibility, latency, safety, style, and Good Baseline behavior.
3. Offer a cloud launch surface for redacted run history, Pro monitoring, checkout, account lifecycle events, and operator-facing docs.
4. Keep raw prompts and outputs local by default.

## 2. Core Architecture
```text
cmd/baseline/              Go CLI entrypoint
internal/baseline/         CLI, MCP server, SQLite, scheduling, sync, probes
openclaw-plugin/           Plugin bundle and baseline-health skill
package/                   npm wrapper package
web/src/index.ts           Cloudflare Worker app, landing, dashboard, APIs
web/schema.sql             Neon schema reference
docs/                      Publishing, validation, deployment, plans
handoff/                   Bead evidence ledger
```

## 3. Tech Stack
| Layer | Choice | Specifics |
|---|---|---|
| CLI/MCP | Go | SQLite via `modernc.org/sqlite`, local-first artifacts |
| Web app | Cloudflare Workers | TypeScript Worker, static assets, API routes |
| Cloud DB | Neon Postgres | Redacted run sync, events, question sets, entitlements |
| Payments | Stripe | Checkout sessions or payment links, webhook-backed entitlement model |
| Lifecycle email | Klaviyo | Transactional/subscription lifecycle events when configured |
| Package wrapper | npm/pnpm | JS wrapper around Go release path |

## 4. Agent and Sub-Agent Profiles

### Hybrid Agent Selection Policy (Mandatory)

Default behavior:
- Use contextual/dynamic agent selection for low-risk and single-domain beads.

Hard guardrails:
- If a bead changes schema/migrations, auth/policy/security logic, public API contracts, or deployment/runtime:
  - Required path: Architect review -> domain Engineer implementation -> Analyst review.
- If a bead includes Figma URL/node or visual parity requirement:
  - Required implementer: Frontend Engineer with Figma tool access.
- If a bead touches deploy targets, Worker runtime, Render, or infra config:
  - Required implementer: DevOps Engineer or equivalent deploy specialist.

Selection protocol per bead:
1. Primary agent chosen by context.
2. Record selection rationale in bead summary: chosen agent, why chosen, confidence, fallback agent.
3. If confidence is low or bead spans multiple domains, split bead or escalate to Architect before implementation.

Non-negotiable:
- Dynamic selection cannot bypass hard guardrails.

## 5. Branching & Commits
Convention: `<type>(bead-N): description`

Types: `feat`, `optimization`, `fix`, `test`, `docs`, `chore`.

Branch naming: `codex/feat/bead-N-description`, `codex/fix/...`, or `codex/chore/...`. Never commit directly to `main`.

## 6. Continuity Ledger
Protocol for `CONTINUITY.md`:
- Read and update every turn.
- Keep the required headings.
- Treat beads as atomic execution units and the ledger as durable state.
- Include a Ledger Snapshot in implementation and review replies.
- Mark uncertain items as `UNCONFIRMED`.

## 7. Workflow

### Bead Entry Gate (Mandatory)

Before implementation starts:
1. Bead scope and acceptance tests are explicit.
2. Agent selected using Hybrid Agent Selection Policy.
3. Required tools declared.
4. Risk class declared as `Low`, `Medium`, or `High`.

Risk classes:
- `Low`: single-domain, no contract/security/deploy impact.
- `Medium`: multi-file/domain, no hard-guardrail impact.
- `High`: any hard-guardrail triggered.

### Bead Exit Gate (Mandatory)

Before bead completion:
1. Required tests pass for risk class.
2. Reviewer checklist completed: completeness, quality, consistency, tests, security.
3. `CONTINUITY.md` updated.
4. `handoff/beads.jsonl` updated.
5. Chat bead summary posted.

TDD is default for code work unless explicitly overridden for non-code work.

## 8. Orchestration

### Spawn Contract (Mandatory)

Each spawned agent prompt must include:
1. Owned files/paths.
2. In-scope and out-of-scope work.
3. Required tools and constraints.
4. Acceptance tests and expected outputs.
5. Report fields: changes made, test commands/results, assumptions/risks, follow-up recommendations.

### Escalation Rules

Architect sign-off required before implementation if:
1. Public API shape changes.
2. Data schema/migration changes.
3. Policy/security model semantics change.
4. Deployment architecture/runtime behavior changes.

## Validation Matrix
- `code`: lint/format checks and relevant unit/integration tests.
- `docs/process`: markdown consistency, internal path verification, policy consistency check.
- `design/ui`: source reference, screenshot or visual artifact, responsive/accessibility notes.
- `research/analysis`: source list, labeled assumptions, recommendation/tradeoff summary.
- `ops/deploy`: deploy preflight, runtime binding verification, health check, rollback path.

## Safety
- Do not exfiltrate private data.
- Do not print secrets, JWTs, private keys, copied key file contents, or payment credentials.
- Use recoverable flows for destructive actions and ask before irreversible decisions.
- Keep `main` deployable.
