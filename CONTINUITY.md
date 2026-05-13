## Goal (incl. success criteria)
Build Baseline.ai v0 as a local-first Go/SQLite CLI and MCP drift checker for coding-agent workstations, plus a deployed Cloudflare/Neon launch surface. Success is a working local check/known-good/compare loop, OpenClaw MCP install, redacted cloud sync, landing page, dashboard, payment hook, validation notes, and clear launch blockers.

## Constraints/Assumptions
- Repo is now a git repository, but no remote is configured; commit is possible, push is blocked until a remote is added.
- Bead 6 committed locally as `b00a1a7` before amend; final commit is the current `HEAD`.
- v0 is a local known-good drift checker, not a broad eval platform.
- Fast mode must never execute the agent. Full mode requires explicit opt-in for agent execution.
- Cloud sync must fail closed and export only redacted/hash summaries.
- Payment checkout is implemented but cannot go live without Stripe secrets, price IDs, or payment links.

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
- The first dogfood path is: `baseline check`, `baseline known-good mark`, `baseline compare`, `baseline install openclaw`, redacted cloud sync.

## State
### Done
- [x] Bead 1: Baseline.ai market, competitor, positioning, and smoke-test research
- [x] Bead 2: Refined OpenClaw-first MCP/CLI MVP direction
- [x] Bead 3: Safety and eval shape for Baseline MCP/CLI
- [x] Bead 4: Proconsult-attempted product shaping into smallest defensible v0
- [x] Bead 5: Fixed Proconsult browser login path and incorporated successful consult
- [x] Bead 6: Implemented and deployed Baseline v0 CLI/MCP, landing page, dashboard, Neon sync, and launch docs

### Now
- Bead 6 complete locally and deployed. Latest clean known-good is `post-mcp-clean`.

### Next
- Add Stripe secrets or payment links and verify checkout end-to-end.
- Add token issuance/rotation UI instead of a single Worker secret.
- Add scheduled local run instructions or daemon/cron helper.
- Add alert delivery after the local report earns trust.
- Refactor Go packages toward the Proconsult-recommended hard boundaries if v0 expands.

## Open Questions
- Which Stripe plan IDs or payment links should be used for Pro and Team?
- Should the first alert destination be local OpenClaw notification, Slack, GitHub Checks, or email?
- Should `baseline check --full --run-agent` be dogfooded now, or kept manual until prompt cost/runtime behavior is reviewed?
- Should token issuance be self-serve in the dashboard or manual for the first ten users?

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
- `/Users/future/dev/baseline/README.md`
- `/Users/future/dev/baseline/docs/VALIDATION.md`
- `/Users/future/dev/baseline/docs/SKILL_USAGE.md`


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
