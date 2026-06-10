# CONTINUITY.md - Baseline.ai

## Goal (incl. success criteria)
Build Baseline.ai v0 as a local-first Go/SQLite CLI and MCP drift checker for coding-agent workstations, plus a deployed Cloudflare/Neon launch surface. Current Bead 34 success is a commercial-viability pass that turns the Bead 33 acquisition surface into a first-customer path: pilot request, admin invite, paid checkout, magic-link onboarding, workspace token, redacted sync, and account-private history.

## Constraints/Assumptions
- Git remote `origin` is configured as `https://github.com/apollostreetcompany/baseline.git`.
- Bead 6 committed locally as `b00a1a7` before amend; final commit is the current `HEAD`.
- v0 is a local known-good drift checker, not a broad eval platform.
- `baseline doctor` must never execute the agent. `baseline run`, `baseline setup`, and scheduled runs execute the operator-approved default target and write local report/response artifacts.
- Cloud sync must fail closed and export only redacted/hash summaries.
- Payment checkout is implemented but cannot go live without Stripe secrets, price IDs, or payment links.
- OpenProse Codex skill was repaired to upstream 0.13.1 on 2026-05-14; stale local copy backed up at `/Users/future/.codex/skills/open-prose.backup-20260513172352`.
- Bead 27 work is isolated in `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-landing-a-brand-os` on branch `codex/feat/bead-27-landing-a-brand-os` so the original dirty worktree can remain available to the other agent.

## Key Decisions
- Broad "LLM observability" positioning is not viable as a solo wedge because incumbent trace/eval platforms already own it.
- The strongest provisional wedge is OpenClaw-first, local-first coding-agent workstation health checks and drift baselines.
- Validation should be paid and narrow: sell a 7-day agent health pilot before building a full SaaS.
- The product should be an async, fast, tidy MCP/CLI that self-configures on first run.
- OpenTelemetry concepts should shape the data model, but OpenTelemetry must remain optional.
- Current buyers are OpenClaw users, agency owners, CTOs, and founder-CTOs.
- Drift checks do not need to be deterministic. The core value is detecting behavioral changes across repeated prompts, user/project memory, speed, substance, style, and tool reliability.
- Personality drift can be scored as behavior drift: verbosity, warmth, directness, sycophancy, pushback, substance consistency, and user-style adherence.
- Go was selected for the CLI/MCP binary and Cloudflare Workers + Neon for the launch surface.
- MCP is intentionally limited to seven legible tools.
- The first dogfood path is: `baseline setup`, `baseline report`, explicit `baseline accept RUN_ID --confirm "accept RUN_ID"`, `baseline run`, `baseline compare`, redacted cloud sync.
- Bootstrap agent probes require a recent preview receipt before messages are sent.
- Baseline Core OpenClaw probes run with bounded concurrency and store actual per-probe send/receive durations, not recorder lag.
- Attached recipe-style `.prose.md` files are legacy frontmatter workflows without `kind:`; they now have compatibility run receipts under `.prose/runs/`.
- Bead 23 is split into two tracks: 23A refreshes brand/design/docs/blog landing surfaces around the supplied tennis-robot imagery; 23B defines and scaffolds the Pro monitoring account architecture using Bibe Code's Stripe/Klaviyo lifecycle pattern as reference without copying secrets or religious product semantics.
- MagicPath theme input selected for Bead 23A is a blend: Brutalism supplies the hard-edged editorial stance, Ramp supplies the operational SaaS restraint, and the actual palette is derived around the court images: film teal, clay, tennis-line cream, signal lime, and graphite.
- Bead 25 locks Baseline Pro as a cloud-backed product on Cloudflare Worker + Neon, not Supabase and not local-only MCP. REST remains canonical, the remote MCP is an authenticated adapter over account/history/hotspot/billing operations, and the local CLI remains the redacted probe runner/sync client.
- Bead 25 commercial target is a paid pilot at `$39/mo`; billing access and destructive operations must use Stripe portal handoff or explicit confirmation, not silent MCP mutation.
- Bead 25 comparison v1 exposes self-history only while storing account-private and benchmark-ready aggregate-safe fields for later team/anonymous modes behind consent and feature flags.
- Bead 27 preserves the deployed Bead 25 cloud account and remote MCP surface while replacing the homepage with the `landing-a` design/assets from `/Users/kikimac/Downloads/baseline.zip`; the latest Cloudflare Worker deploy is version `4f1b94a0-543a-4cb2-8207-62825fb29594`.
- BrandOS on this machine must use `python3` and the bundled `.prose` validator fallback when `prose` or PyYAML are unavailable; the local `brand-os-studio` skill has been repaired accordingly.
- Bead 28 makes `https://trackbaseline.com` the canonical production URL, attaches `trackbaseline.com` and `www.trackbaseline.com` as Cloudflare Worker custom domains, keeps the workers.dev fallback route enabled, and sets Worker `APP_URL` to the apex domain.
- Bead 29 distribution decision: keep the local CLI binary free and easy to install; charge Pro for hosted history, workspace tokens, remote MCP account operations, monitoring, billing lifecycle, and retention. The first public download path is GitHub Releases plus `https://trackbaseline.com/install.sh`, with npm as an auto-downloading wrapper and Homebrew as a later tap.
- Bead 30 analytics decision: use DataFast for launch funnel tracking with client-side script plus click/scroll goals; keep tokens out of files and use `DATAFAST_TOKEN` only in shell/secret storage for CLI reports.
- Bead 31 favicon decision: use the existing `baseline-court-robot.png` photo as the source for browser/app icons so the tab icon matches the launch imagery rather than adding a separate logo mark.
- Bead 32 Codex plugin decision: keep `openclaw-plugin/` as the legacy/OpenClaw compatibility bundle and introduce `plugins/baseline/` as the release-oriented Codex plugin. The v1 plugin is valid for local Codex development when the `baseline` CLI is already on `PATH`; productionization still needs CLI auto-install/preflight, clean-environment Codex smoke tests, plugin assets, and CI-backed schema validation.
- Bead 33 entry gate: focus on MARKET EFFECTIVENESS ONLY. Scope is organic acquisition via 5-10 SEO/AEO blog posts, 3-5 lead magnets attached to the public surface, dashboard/admin UX clarity, and package/core changes that support install-to-value. Risk class is High because deploy/runtime and public web/API surfaces may be touched. Agent path: Architect/strategy synthesis by primary Codex, fresh skill-specific RepoPrompt subagents, domain implementation by engineer agents or primary Codex as needed, then Proconsult and subreview advisory review before deploy.
- Bead 33 market decision: own the "local coding-agent health/drift/MCP workstation check" wedge instead of broad LLM observability. Ship eight guide routes, five resource/lead-magnet routes, an actionable lead queue, live Klaviyo lead/master events when configured, clearer dashboard/admin next-action UX, and `baseline --version` as a first-run smoke.
- Bead 33 deployment decision: deployed Cloudflare Worker version `df4d479d-9fbd-4f8a-af50-b2f3a88253a8` to `https://trackbaseline.com`; rollback target is previous version `b4f73e11-7540-4e97-8112-7698467b0484`. Wrangler deploy requires the Cloudflare account/token env values to be sourced; an unsourced OAuth token failed with Cloudflare `Authentication error [code: 10000]`.
- Bead 34 commercial viability decision: treat first paid customer conversion as a hand-held paid-pilot/account-provisioning problem, not only an SEO/content problem. The public path now captures pilot requests, paid checkout requires email-first account attribution, checkout success requests the magic link and shows token/sync steps, admin can grant a Pro/Team pilot invite, public demo/dashboard surfaces avoid exposing account-private runs, and Stripe webhooks fall back through customer/email before granting entitlement.
- Bead 34 deployment decision: deployed Cloudflare Worker version `7940fc3a-f89e-4972-9352-e77424b541a6` to `https://trackbaseline.com`. Rollback target is previous Bead 33 version `df4d479d-9fbd-4f8a-af50-b2f3a88253a8`. Live admin lead readback remains `UNCONFIRMED` from this shell because the local deploy env has Cloudflare credentials but not `BASELINE_ADMIN_TOKEN`.

## State
### Done
- [x] Bead 1: Baseline.ai market, competitor, positioning, and smoke-test research
- [x] Bead 2: Refined OpenClaw-first MCP/CLI MVP direction
- [x] Bead 3: Safety and eval shape for Baseline MCP/CLI
- [x] Bead 4: Proconsult-attempted product shaping into smallest defensible v0
- [x] Bead 5: Fixed Proconsult browser login path and incorporated successful consult
- [x] Bead 6: Implemented and deployed Baseline v0 CLI/MCP, landing page, dashboard, Neon sync, and launch docs
- [x] Bead 7: Repaired OpenProse VM surface and ran attached `.prose.md` recipes with filesystem receipts
- [x] Bead 8: Consolidated recommendations into sequenced implementation beads
- [x] Bead 9: Added retryable local sync outbox and real Worker dashboard run APIs
- [x] Bead 10: Added admin/versioned canonical question sets and LLM evaluator endpoint
- [x] Bead 11: Added pnpm/npm wrapper package, OpenClaw plugin bundle, and Go release path
- [x] Bead 12: Deployed Worker and verified local run sync renders on dashboard APIs
- [x] Bead 13: Added daily launchd self-check schedule and OpenClaw-triggerable `baseline_schedule` MCP tool
- [x] Bead 14: Launched and hardened v0.1 bootstrap/Good Baseline flow with updated 14-question Baseline Core, preview-before-run receipts, scoped Good Baseline slots, bounded real OpenClaw send/receive timing, fresh-only token metadata, OpenClaw-style config CLI, updated MCP tools, local binary install, and deployed Worker docs/question set
- [x] Bead 15: Added operator-first Baseline setup/run/report/accept UX, local response artifacts, agent BOOTSTRAP.md contract, default target config, real scheduled evals, structured MCP recovery errors, and seven workflow-first MCP tools.
- [x] Bead 16: Fixed first-run lifecycle issues by making doctor/preflight ephemeral, making latest/status prefer real eval runs over local preflight rows, adding async MCP setup/run/schedule execution with run status files, and adding a bounded cloud sync HTTP timeout.
- [x] Bead 17: Hardened scheduled Baseline runs by persisting `workspace_path`, installing launchd with `WorkingDirectory`, `BASELINE_WORKSPACE`, `HOME`, and a Homebrew-aware `PATH`, running repo/agent probes from the configured workspace, and preventing newer preflight-only scheduled failures from hiding the last completed eval.
- [x] Bead 18: Fixed expanded-run ambiguity by marking stale async lifecycle runs failed when their child PID is gone, showing lifecycle status through `baseline report RUN_ID`, running MCP child processes from the configured workspace, and printing the planned pack/question count before long direct CLI runs.
- [x] Bead 19: Made long OpenClaw Baseline evals agent-safe by starting long non-interactive `baseline run` commands in the managed background, preventing recursive async children with `BASELINE_FOREGROUND=1`, running probes serially by default, updating the agent bootstrap timeout guidance, installing the local binary, and setting the local OpenClaw target timeout to 900s.
- [x] Bead 20: Added an OpenClaw Codex app-server timeout guardrail to `baseline setup`, `baseline install openclaw`, and MCP `baseline_setup`; setup now snapshots `~/.openclaw/openclaw.json`, ensures Codex request/turn-idle timeouts are at least 900 seconds, preserves Google/Gemini provider surfaces, and teaches agents to distinguish 60s Codex watchdog timeouts from degraded fallback and redacted-key auth failures.
- [x] Bead 21: Fixed Baseline background runner lifecycle by detaching long async child processes into their own process session, writing per-question progress into lifecycle status/logs, making lifecycle JSON reports exit `0` completed / `2` running / `1` failed, adding `baseline repair openclaw`, adding `baseline rerun RUN_ID`, and allowing MCP `baseline_run` to recover a failed lifecycle run with `rerun_id`.
- [x] Bead 22: Completed the expanded OpenClaw dogfood eval `run_dilv9nm3rhkg` with all 55 enabled-pack agent questions, confirmed lifecycle completion/report artifacts, captured warning-grade findings for slow long-term health/project/fact-memory probes, and identified the OpenClaw memory-search redacted-key configuration warning as the remaining infrastructure repair.
- [x] Bead 23B: Documented Pro account architecture for Cloudflare-first Stripe/Klaviyo/Neon entitlement flow with Render fallback, rollout beads, validation, and rollback.
- [x] Bead 23A: Refreshed Baseline landing brand identity around supplied tennis-robot images, added Worker static assets, documentation-style landing sections, Pro checkout email form stub, Klaviyo checkout-start event hook, checkout success/cancel pages, blog stub, updated deployment/readme/project scaffolding, and verified desktop/mobile local Worker rendering.
- [x] Bead 24: Deployed the refreshed Baseline landing page to Cloudflare Workers version `5cc879a3-983d-4e59-a620-e8abd8d70a99` and verified live landing, blog, image asset, health, and checkout fallback behavior.
- [x] Bead 25: Implemented and deployed cloud accounts, invite/magic-link sessions, Stripe webhook entitlement lifecycle, account-scoped HMAC workspace tokens, self-history/hotspot/compare APIs, remote MCP adapter, SwiftUI macOS hotspot dashboard, and skill-audited deployment notes. Latest Worker deploy version: `dfc2198f-9151-4a64-8511-4e25d3c2d529`.
- [x] Bead 27: Rebuilt the homepage to match `landing-a`, preserved Bead 25 cloud routes/schema, repaired the local BrandOS skill runtime assumptions, and deployed Cloudflare Worker version `4f1b94a0-543a-4cb2-8207-62825fb29594`.
- [x] Integration: Opened PR #1 (`https://github.com/apollostreetcompany/baseline/pull/1`) from `codex/integrate/bead-27-main-ready` to merge Bead 25 cloud/Mac functionality and Bead 27 Landing A into `main`.
- [x] Bead 28: Deployed the Cloudflare Worker to `https://trackbaseline.com` and `https://www.trackbaseline.com`, verified DNS, health, landing, MCP docs/auth challenge, protected-resource metadata, asset, checkout fail-closed, and fallback workers.dev route. Latest Worker deploy version: `0d0924c3-5c8e-4029-9327-369a73588786`.
- [x] Bead 29: Added the public distribution path (`install.sh`, GitHub Release workflow, npm auto-download wrapper), configured production Stripe/Klaviyo/auth/token secrets, deployed Worker version `e38523fc-d11a-41d9-b05e-6dcef5f4b5f0`, and published GitHub Release `v0.1.0`.
- [x] Bead 30: Added DataFast script and launch funnel events, created DataFast install/Pro funnels with the CLI, added `make analytics-report`, and deployed Worker version `fb899682-a797-4201-9842-4dfb72d5cecd`.
- [x] Bead 31: Added robot photo favicon/app icon assets, wired icon metadata and web manifest, and deployed Worker version `b4f73e11-7540-4e97-8112-7698467b0484`.
- [x] Bead 32: Added a validated Codex plugin v1 under `plugins/baseline/`, repo-local marketplace metadata, `baseline-codex-plugin.tgz` release packaging, plugin validation target, and productionization roadmap.
- [x] Bead 33: Added an SEO/AEO content and lead-magnet acquisition surface, lead request capture/notification/admin queue, dashboard/admin market clarity, CLI version smoke, docs/package first-run guidance, and deployed Worker version `df4d479d-9fbd-4f8a-af50-b2f3a88253a8`.
- [x] Bead 34: Added commercial viability fixes from subreview: pilot request capture, admin pilot invite, email-first Pro/Team checkout, operational checkout success onboarding, scoped checkout-session status, account-safe public dashboard APIs, account-scoped run upsert protection, and paid-pilot deployment docs.

### Now
- Bead 34 is implemented locally from sibling worktree `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability` on branch `codex/feat/bead-34-commercial-viability`, based on Bead 33 (`0ac35af`) so the source checkout remains untouched.
- Bead 34 acceptance tests passed: `make verify`, `git diff --check`, `npm --prefix web audit --audit-level=high`, local Worker smokes for `/`, `/checkout/success`, `/api/checkout`, `/api/events`, `/admin`, `/dashboard`, `/api/runs/latest`, Playwright screenshots for pricing/pilot, checkout success, admin pilot, and dashboard demo, Wrangler deploy, and live smokes for health, homepage markers, checkout success markers, email-required checkout guard, invalid lead guard, and synthetic pilot request storage. `subreview` completed partially: Claude completed with 14 findings; Codex failed on CLI argument incompatibility; Gemini quota was exhausted. Acted on the short-path commercial findings.
- Branch `codex/feat/bead-32-codex-plugin` contains the first Codex plugin v1. `make plugin-validate`, `make test`, `make package-test`, `make web-typecheck`, JSON/path checks, shell syntax checks, and temp `DIST_DIR` release build all pass locally.
- `https://trackbaseline.com` is the canonical share URL for later today. `https://www.trackbaseline.com` and `https://baseline-ai.ryan-borker.workers.dev` also serve the Worker.
- Public install works through `curl -fsSL https://trackbaseline.com/install.sh | sh`, backed by GitHub Release `v0.1.0` assets and checksums. The npm wrapper can auto-download the same release, but the npm package is not published yet because this machine is not logged into npm.
- CI now includes a macOS `verify` workflow for PRs, and local `make mac-build` uses Swift strict concurrency to match GitHub's Swift 6 behavior.
- Production Pro configuration is active: Stripe Checkout prices are set for Pro `$39/mo` and Team `$129/mo`, Stripe webhook signature verification is configured, Klaviyo lifecycle email is configured, and magic-link/session/workspace-token HMAC secrets are configured.
- DataFast analytics is live on `trackbaseline.com` with website id `6a0c48aa9a21aee7bf04cf6e`, tracking id `dfid_PYprhfTkwwQKhkzRUhVtO`, install funnel `baseline-install-funnel`, Pro funnel `baseline-pro-funnel`, and CLI reporting through `DATAFAST_TOKEN=... make analytics-report`.
- `https://trackbaseline.com/favicon.ico`, PNG favicons, Apple touch icon, `icon-192.png`, `icon-512.png`, and `site.webmanifest` now return `HTTP 200`.
- `/opt/homebrew/bin/baseline` points to `/Users/future/go/bin/baseline`, OpenClaw plugin loads the `baseline` MCP server, daily LaunchAgent `ai.baseline.daily` is installed for 09:00 local time with `WorkingDirectory=/Users/future/.openclaw/workspace`, `BASELINE_WORKSPACE=/Users/future/.openclaw/workspace`, and a PATH that includes `/opt/homebrew/bin`; the Worker is deployed at version `e38523fc-d11a-41d9-b05e-6dcef5f4b5f0`.
- Primary path is now `baseline setup`, `baseline run`, `baseline report`, and `baseline accept RUN_ID --confirm "accept RUN_ID"`. `baseline doctor` is read-only preflight; legacy `check`/`bootstrap` remains available for compatibility.
- First real OpenClaw eval `run_dil295nlwpug` completed with status warning, health 92, 14 Baseline Core probes, and one slow `ops_change` warning at 95026ms. A later scheduled run `run_dil2s3gle45k` did fire but failed preflight from `/` with launchd's stripped PATH; `baseline latest` and `baseline status` now point back to the real eval instead of that preflight-only failure.
- MCP `baseline_run`, `baseline_setup`, and `baseline_schedule action=run` now return quickly with a lifecycle `run_status.run_id`; agents should poll `baseline_report` for completion instead of holding the MCP call open for the whole eval. If the child process disappears before a DB row is written, `baseline_report`/`baseline report` marks the run failed, includes stdout/stderr paths, and suggests `baseline rerun RUN_ID`.
- The attempted expanded eval `run_dils8v7hqioo` did not reach the database because the background child disappeared without a result row. It now reports a failed lifecycle with exit code 1 through `baseline report run_dils8v7hqioo --json`. The repaired detached rerun `run_dilv9nm3rhkg` completed with status `warning`, wrote report/response artifacts under `/Users/future/.baseline/reports/run_dilv9nm3rhkg/`, covered all 55 enabled-pack questions in about 86.7 minutes, and should be treated as evidence rather than accepted as a Good Baseline until scoring, token metadata, and memory-search secret repair are addressed.
- Long non-interactive `baseline run --packs enabled` now returns a managed run id immediately instead of holding an OpenClaw/Codex agent turn open until `codex app-server attempt timed out`. Foreground terminal runs still wait, and `baseline run --foreground --packs enabled` remains available for deliberate blocking runs. Probe concurrency now defaults to 1; `BASELINE_PROBE_CONCURRENCY` is the advanced override.
- OpenClaw Codex app-server request and turn-idle timeouts are now managed as a setup guardrail at 900s minimum. The live `/Users/future/.baseline/BOOTSTRAP.md` has been regenerated with the timeout/fallback/redacted-key diagnosis rules, and `baseline install openclaw` reports `OpenClaw Codex timeout: already >= 900s`.
- `baseline doctor` now surfaces the current OpenClaw memory-search redacted placeholder as a warning instead of passing silently: `openclaw.memory.redacted_key`. This is separate from Google/Gemini search config and should be repaired through OpenClaw's secret/config path, not by removing providers.

### Next
- End-to-end Pro pilot smoke with a real invited account: checkout, webhook entitlement, magic-link login, workspace token creation, redacted sync, history/hotspot/compare, and remote MCP account status.
- Publish `@baseline-ai/cli` once npm auth for the `@baseline-ai` scope is available; the package is ready and `npm pack --dry-run` passes.
- Productionize the Codex plugin: add missing-CLI preflight/auto-install, vendor or officialize plugin schema validation in CI, add icon/logo/screenshots, smoke in a clean Codex install, and publish `baseline-codex-plugin.tgz` with the next release.
- Create a Homebrew tap for persistent macOS installs after the first pilot users validate the install script.
- Later sequence: app-level retention enforcement, OpenClaw runner pack, MCP schema drift testing against target clients, local scheduling, local alert preview, OpenProse contract migration, 10-user paid pilot, package boundary refactor.

## Open Questions
- Which real pilot email should receive the first invite and live checkout test?
- Which npm account/org should publish the `@baseline-ai/cli` package?
- Should Homebrew live under `apollostreetcompany/homebrew-tap` or a dedicated `trackbaseline/homebrew-tap` repo?
- What separate admin token should replace the temporary dogfood reuse of the sync token?
- Which OpenAI evaluator key/model should be used for paid pilot evaluation?
- Should the first alert destination be local OpenClaw notification, Slack, GitHub Checks, or email?
- Whether the two stale-token OpenClaw probes (`tools`, `ops_change`) are OpenClaw session freshness limitations or prompt/runtime issues to tune.
- Should token issuance be self-serve in the dashboard or manual for the first ten users?
- Should the recipe library be migrated in place to OpenProse 0.13.1 contract frontmatter, or should compatibility mode remain supported for older `.prose.md` recipes?

## Working Set
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability/web/src/index.ts`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability/web/src/cloud.ts`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability/handoff/bead-34-pricing-pilot.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability/handoff/bead-34-checkout-success.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability/handoff/bead-34-admin-pilot.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-34-commercial-viability/handoff/bead-34-dashboard-demo.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/web/src/index.ts`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/docs/DEPLOYMENT.md`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/handoff/bead-33-blog.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/handoff/bead-33-dashboard.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/handoff/bead-33-lead-resource-final.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/handoff/bead-33-admin-final.png`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/web/src/cloud.ts`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/package`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/internal/baseline`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-bead-33-seo-lead-magnets/plugins/baseline`
- `/Users/future/.openclaw/workspace/repos/skills-library/recipes/00-kill-gate.prose.md`
- `/Users/future/.openclaw/workspace/repos/skills-library/recipes/03-offer-smoke-test.prose.md`
- `/Users/future/.openclaw/workspace/repos/skills-library/skills/app-intel`
- `/Users/future/dev/baseline/BASELINE_MVP.md`
- `/Users/future/dev/baseline/BASELINE_V0_SHAPE.md`
- `/Users/future/dev/baseline/PROCONSULT_BASELINE_V0.md`
- `/Users/future/dev/baseline/PROCONSULT_LAUNCH_ARCHITECTURE.md`
- `/Users/future/dev/baseline/cmd/baseline/main.go`
- `/Users/future/dev/baseline/internal/baseline`
- `/Users/future/dev/baseline/web/src/index.ts`
- `/Users/future/dev/baseline/web/wrangler.jsonc`
- `/Users/future/dev/baseline/web/public/assets/baseline-court-*.png`
- `/Users/future/dev/baseline/README.md`
- `/Users/future/dev/baseline/docs/PUBLISHING.md`
- `/Users/future/dev/baseline/docs/DEPLOYMENT.md`
- `/Users/future/dev/baseline/package`
- `/Users/future/dev/baseline/openclaw-plugin`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline/plugins/baseline`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline/.agents/plugins/marketplace.json`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline/docs/CODEX_PLUGIN.md`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline/scripts/validate-codex-plugin.sh`
- `/Users/future/dev/baseline/docs/VALIDATION.md`
- `/Users/future/dev/baseline/docs/SKILL_USAGE.md`
- `/Users/future/dev/baseline/docs/OPENPROSE_RUN_RESULTS.md`
- `/Users/future/dev/baseline/docs/plans/2026-05-14-001-feat-baseline-next-beads-plan.md`
- `/Users/future/dev/baseline/AGENTS.md`
- `/Users/future/dev/baseline/HANDOFF.md`
- `/Users/future/dev/baseline/MISTAKES.md`
- `/Users/future/dev/baseline/handoff/beads.jsonl`
- `/Users/future/dev/baseline/web/public/assets`
- `/Users/future/dev/baseline/docs/plans/2026-05-19-001-pro-account-architecture.md`
- `/Users/future/dev/baseline/.prose/runs/20260514-002532-*`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/baseline-landing-a-brand-os`
- `/Users/kikimac/Downloads/baseline.zip`
- `/Users/kikimac/.hermes/repos/apollostreetcompany/skills-library/skills/brand-os-studio`


<!-- BEGIN COMPOUND CODEX TOOL MAP -->
## Compound Codex Tool Mapping (Claude Compatibility)

This section maps Claude Code plugin tool references to Codex behavior.
Only this block is managed automatically.

Tool mapping:
- Read: use shell reads (cat/sed) or rg
- Write: create files via shell redirection or apply_patch
- Edit/MultiEdit: use apply_patch
- Bash: use shell_command
- Grep: use rg (fallback: grep)
- Glob: use rg --files or find
- LS: use ls via shell_command
- WebFetch/WebSearch: use curl or Context7 for library docs
- AskUserQuestion/Question: present choices as a numbered list in chat and wait for a reply number. For multi-select (multiSelect: true), accept comma-separated numbers. Never skip or auto-configure — always wait for the user's response before proceeding.
- Task/Subagent/Parallel: run sequentially in main thread; use multi_tool_use.parallel for tool calls
- TodoWrite/TodoRead: use file-based todos in todos/ with file-todos skill
- Skill: open the referenced SKILL.md and follow it
- ExitPlanMode: ignore
<!-- END COMPOUND CODEX TOOL MAP -->
