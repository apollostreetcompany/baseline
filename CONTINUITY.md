# CONTINUITY.md - Baseline.ai

## Goal (incl. success criteria)
Build Baseline.ai v0 as a local-first Go/SQLite CLI and MCP drift checker for coding-agent workstations, plus a deployed Cloudflare/Neon launch surface. Success is a working local check/known-good/compare loop, OpenClaw MCP install, redacted cloud sync, landing page, dashboard, payment hook, validation notes, and clear launch blockers.

## Constraints/Assumptions
- Git remote `origin` is configured as `https://github.com/apollostreetcompany/baseline.git`.
- Bead 6 committed locally as `b00a1a7` before amend; final commit is the current `HEAD`.
- v0 is a local known-good drift checker, not a broad eval platform.
- `baseline doctor` must never execute the agent. `baseline run`, `baseline setup`, and scheduled runs execute the operator-approved default target and write local report/response artifacts.
- Cloud sync must fail closed and export only redacted/hash summaries.
- Payment checkout is implemented but cannot go live without Stripe secrets, price IDs, or payment links.
- OpenProse Codex skill was repaired to upstream 0.13.1 on 2026-05-14; stale local copy backed up at `/Users/future/.codex/skills/open-prose.backup-20260513172352`.

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

### Now
- Bead 25: Dogfood/admin token split, Stripe entitlement, or API token/workspace model is next, using `docs/plans/2026-05-19-001-pro-account-architecture.md` as the architecture source. `/opt/homebrew/bin/baseline` points to `/Users/future/go/bin/baseline`, OpenClaw plugin loads the `baseline` MCP server, daily LaunchAgent `ai.baseline.daily` is installed for 09:00 local time with `WorkingDirectory=/Users/future/.openclaw/workspace`, `BASELINE_WORKSPACE=/Users/future/.openclaw/workspace`, and a PATH that includes `/opt/homebrew/bin`; the Worker is deployed at version `5cc879a3-983d-4e59-a620-e8abd8d70a99`.
- Primary path is now `baseline setup`, `baseline run`, `baseline report`, and `baseline accept RUN_ID --confirm "accept RUN_ID"`. `baseline doctor` is read-only preflight; legacy `check`/`bootstrap` remains available for compatibility.
- First real OpenClaw eval `run_dil295nlwpug` completed with status warning, health 92, 14 Baseline Core probes, and one slow `ops_change` warning at 95026ms. A later scheduled run `run_dil2s3gle45k` did fire but failed preflight from `/` with launchd's stripped PATH; `baseline latest` and `baseline status` now point back to the real eval instead of that preflight-only failure.
- MCP `baseline_run`, `baseline_setup`, and `baseline_schedule action=run` now return quickly with a lifecycle `run_status.run_id`; agents should poll `baseline_report` for completion instead of holding the MCP call open for the whole eval. If the child process disappears before a DB row is written, `baseline_report`/`baseline report` marks the run failed, includes stdout/stderr paths, and suggests `baseline rerun RUN_ID`.
- The attempted expanded eval `run_dils8v7hqioo` did not reach the database because the background child disappeared without a result row. It now reports a failed lifecycle with exit code 1 through `baseline report run_dils8v7hqioo --json`. The repaired detached rerun `run_dilv9nm3rhkg` completed with status `warning`, wrote report/response artifacts under `/Users/future/.baseline/reports/run_dilv9nm3rhkg/`, covered all 55 enabled-pack questions in about 86.7 minutes, and should be treated as evidence rather than accepted as a Good Baseline until scoring, token metadata, and memory-search secret repair are addressed.
- Long non-interactive `baseline run --packs enabled` now returns a managed run id immediately instead of holding an OpenClaw/Codex agent turn open until `codex app-server attempt timed out`. Foreground terminal runs still wait, and `baseline run --foreground --packs enabled` remains available for deliberate blocking runs. Probe concurrency now defaults to 1; `BASELINE_PROBE_CONCURRENCY` is the advanced override.
- OpenClaw Codex app-server request and turn-idle timeouts are now managed as a setup guardrail at 900s minimum. The live `/Users/future/.baseline/BOOTSTRAP.md` has been regenerated with the timeout/fallback/redacted-key diagnosis rules, and `baseline install openclaw` reports `OpenClaw Codex timeout: already >= 900s`.
- `baseline doctor` now surfaces the current OpenClaw memory-search redacted placeholder as a warning instead of passing silently: `openclaw.memory.redacted_key`. This is separate from Google/Gemini search config and should be repaired through OpenClaw's secret/config path, not by removing providers.

### Next
- Bead 25: Dogfood/admin token split, Stripe entitlement, or API token/workspace model, depending on available credentials and the Bead 23B architecture note.
- Later sequence: Stripe entitlement, token/workspace model, app-level retention, OpenClaw runner pack, MCP schema drift, local scheduling, local alert preview, OpenProse contract migration, 10-user paid pilot, package boundary refactor.

## Open Questions
- Which Stripe plan IDs or payment links should be used for Pro and Team?
- UNCONFIRMED: whether Pro monitoring should launch first on the current Cloudflare Worker + Neon stack or split transactional billing/webhook handling into a Render service.
- What separate admin token should replace the temporary dogfood reuse of the sync token?
- Which OpenAI evaluator key/model should be used for paid pilot evaluation?
- Should the first alert destination be local OpenClaw notification, Slack, GitHub Checks, or email?
- Whether the two stale-token OpenClaw probes (`tools`, `ops_change`) are OpenClaw session freshness limitations or prompt/runtime issues to tune.
- Should token issuance be self-serve in the dashboard or manual for the first ten users?
- Should the recipe library be migrated in place to OpenProse 0.13.1 contract frontmatter, or should compatibility mode remain supported for older `.prose.md` recipes?

## Working Set
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
