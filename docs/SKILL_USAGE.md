# Skill Usage Ledger

This ledger records the skills and recipe files consulted for Baseline v0 and the concrete product changes they caused.

| Skill or recipe | Count | When used | Changes made |
| --- | ---: | --- | --- |
| `proconsult` | 2 | Product shaping and launch architecture | Narrowed v0 to local known-good drift checker; enforced Go CLI/MCP, local SQLite, scrubbed cloud sync, seven-tool MCP, and cutting generic eval-platform scope. |
| `mcp-server-design` | 1 | MCP design | Kept the server to seven tools, added clear tool descriptions, structured JSON output, and scrub preview. |
| `stripe-checkout` | 1 | Payment path | Added `/api/checkout` with Stripe Checkout support and payment-link fallback; left fail-closed `503` until secrets exist. |
| `wrangler` | 1 | Deployment | Added Worker config and deployed `baseline-ai` to Cloudflare Workers. |
| `supabase` | 1 | Database safety comparison | Rejected Supabase for v0 because user requested Neon; reused security posture around server-side DB access only. |
| `documentation-website-for-software-project` | 1 | Install docs | Added `/docs/mcp` and README install instructions with copy/paste commands. |
| `ux-audit` | 1 | CLI and landing review | Made fast/full behavior explicit, added safety defaults, and avoided hidden agent execution. |
| `xf` | 1 attempted | User-pain validation | Attempt failed because `xf` was not installed; used supplied X research and recorded the blocker. |
| `social-learner` | 1 | Pain clustering | Centered copy and probes on memory, latency, self-improvement, output acceptance, dedup, blocked jobs, and tool reliability. |
| `analytics-baseline` | 1 | Launch measurement | Added `/api/events`, CTA beacon events, health endpoint, and Neon `baseline_events`. |
| `baseline-ui` | 1 | Product UI | Built a visual dashboard with health score, run timeline bars, signal list, probe status, and alert states. |
| `deploy-ops` | 1 | Launch flow | Provisioned isolated Neon project, bound Worker secrets, deployed Worker, and ran smoke checks. |
| `launchability-audit` | 1 | Final pass | Identified Stripe credentials as the only launch blocker; verified health, docs, MCP, and sync paths. |
| `promise-integrity` | 1 | Copy and safety | Avoided claiming live payments; stated cloud sync payload limits and raw-export defaults. |
| `revenue-plumber` | 1 | Pricing and checkout | Added Local, Pro, and Team tiers with Pro/Team CTAs wired to checkout endpoint. |
| `seo-aeo` | 1 | Search/AEO structure | Added metadata, sitemap, robots, JSON-LD, direct answer copy, and install-doc page. |
| `ai-seo` | 1 | AI-readable positioning | Used direct, answer-like sections around what Baseline is, who it is for, and how to install. |
| `ui-animate` | 1 | Motion consideration | No animation added; v0 dashboard was kept still and utilitarian to preserve trust and reduce distraction. |
| `21-full-brand-os-production-landing.prose.md` | 1 | Landing structure | Shaped hero, promise, pricing, and dashboard preview. |
| `web-anti-slop-cleanup.prose.md` | 1 | Visual cleanup | Removed generic SaaS fluff and kept controls compact, legible, and concrete. |
| `web-landing-page-responsive.prose.md` | 1 | Responsive layout | Added mobile grid collapse, stable dashboard dimensions, and text-safe buttons. |

## Changes From Skill Review

- Cut broad "agent observability" positioning in favor of "local known-good drift checker."
- Added a concrete OpenClaw MCP install flow.
- Added token-gated ingest instead of accepting arbitrary bearer tokens.
- Swapped cloud payload from full local run object to a reduced redacted/hash payload.
- Left Stripe checkout blocked rather than faking payment readiness.
- Made the first useful path work without cloud: check, mark known-good, compare.
