# Skill Execution Evidence

This corrects the wording in `docs/SKILL_USAGE.md`: most attached "skills" were not independently executable jobs with their own stdout. They were instruction files consulted during implementation. The skills that did produce standalone tool output are listed separately below.

## Actual Standalone Runs

### `proconsult`

Output artifact:

- `PROCONSULT_BASELINE_V0.md`
- `PROCONSULT_LAUNCH_ARCHITECTURE.md`

Observed completion output:

```text
Saved assistant output to /Users/future/dev/baseline/PROCONSULT_LAUNCH_ARCHITECTURE.md
```

Implemented changes:

- v0 narrowed to a local known-good drift checker, not a broad eval platform.
- CLI/MCP kept in Go; Cloudflare web/API kept in TypeScript Worker.
- Cloud sync changed from full run export to a reduced redacted/hash payload.
- Full question probes require explicit opt-in.

### `xf`

Command output:

```text
zsh:1: command not found: xf
```

Implemented changes:

- No live X archive scrape was used.
- Validation used the user-supplied X research and recorded the blocker in `docs/VALIDATION.md`.

### `wrangler` / deploy path

Representative deploy output:

```text
Uploaded baseline-ai
Deployed baseline-ai triggers
  https://baseline-ai.ryan-borker.workers.dev
```

Implemented changes:

- Added `web/wrangler.jsonc`.
- Deployed the Cloudflare Worker.
- Bound `DATABASE_URL` and `BASELINE_API_TOKEN` as Worker secrets.

### Neon setup

Representative project creation output:

```text
"id": "summer-cake-63602849"
"name": "baseline-v0"
```

Implemented changes:

- Created isolated Neon project `baseline-v0`.
- Added Worker-backed schema for `baseline_runs` and `baseline_events`.

### OpenClaw MCP dogfood

Representative output:

```text
Registered Baseline MCP with OpenClaw.
Verify with: openclaw mcp list

MCP servers (/Users/future/.openclaw/openclaw.json):
- baseline
```

Implemented changes:

- Registered `baseline` MCP in local OpenClaw.
- Fixed a real SQLite persistence bug found during the second baseline run.
- Marked `post-mcp-clean` as known-good.

### Verification Commands

Representative outputs:

```text
go test ./...
ok  	github.com/future/baseline/internal/baseline

npm run typecheck
tsc --noEmit

curl /api/health
{"ok":true,"db":true,"stripe":false,"token_required":true}

bad ingest token
403

checkout without Stripe config
503
```

Implemented changes:

- Kept checkout fail-closed until Stripe credentials or payment links exist.
- Added token-gated ingest instead of accepting any bearer token.
- Verified dashboard and landing page with Playwright screenshots.

## Consulted Skills Without Standalone Run Output

These skills were read as design/build instructions and applied directly. There is no separate stdout transcript for each one:

| Skill | Evidence in code/docs |
| --- | --- |
| `mcp-server-design` | Seven MCP tools in `internal/baseline/mcp.go`; structured tool descriptions and scrub preview. |
| `stripe-checkout` | `/api/checkout` in `web/src/index.ts`; Stripe Checkout plus payment-link fallback. |
| `documentation-website-for-software-project` | `/docs/mcp` route and `README.md` install docs. |
| `ux-audit` | Explicit fast/full safety copy and no hidden agent execution in CLI help and docs. |
| `social-learner` | Pain clustering in landing copy, question pack, and `docs/VALIDATION.md`. |
| `analytics-baseline` | `/api/events`, CTA beacons, `baseline_events` table. |
| `baseline-ui` | Dashboard visual with health score, bars, probes, and alert states. |
| `deploy-ops` | Neon project, Worker secrets, deploy checks, health endpoint. |
| `launchability-audit` | Stripe blocker called out instead of hidden; health/docs/MCP/sync verified. |
| `promise-integrity` | Copy says raw prompts are not exported unless explicitly enabled; checkout is not claimed live. |
| `revenue-plumber` | Local/Pro/Team pricing and checkout CTAs. |
| `seo-aeo` | Metadata, sitemap, robots, JSON-LD, direct install docs. |
| `ai-seo` | Direct answer-style positioning and install page. |
| `ui-animate` | No animation added; trust-first dashboard kept still. |
| `supabase` | Consulted only as DB/security comparison; rejected because user requested Neon. |

## User Preference Mapping

| User preference | Implemented result |
| --- | --- |
| Lightweight and legible MCP | Seven MCP tools, no broad tool sprawl. |
| Configurable with toggles | Config has core packs and sync toggles; MCP exposes config without leaking tokens. |
| 14 days stored data | Landing and pricing promise this; Neon storage is wired, retention enforcement still needs scheduled cleanup. |
| Paid beyond 14 days | Checkout endpoint and pricing CTAs implemented; Stripe credentials are the blocker. |
| Safety scrubs keys/personal data | Synthetic scrubber check, scrub preview tool, reduced cloud payload, token-gated ingest. |
| Prefer Go | CLI/MCP implemented in Go. |
| Cloudflare + Neon | Worker deployed to Cloudflare, Neon project provisioned and bound. |
| OpenClaw dogfood | Local OpenClaw MCP registered and baseline run completed. |
| User pain first | Copy and baseline questions focus on latency, memory, repo awareness, tools, dedup, blocked jobs, output acceptance, and tone. |
