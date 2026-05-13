## Goal (incl. success criteria)
Research Baseline.ai as a daily baseline test and drift monitor for AI agents and coding-agent workstations. Success is a blunt go/no-go recommendation, ranked positioning angles, competitor map, smallest paid validation test, risks, kill criteria, and source-backed evidence.

## Constraints/Assumptions
- Current workspace is not a git repository, so commit and push steps cannot be completed here.
- Research was performed as a market/offer bead using the OpenProse kill-gate and offer-smoke-test recipes as workflow guides.
- No market-size numbers are assumed or fabricated.

## Key Decisions
- Broad "LLM observability" positioning is not viable as a solo wedge because incumbent trace/eval platforms already own it.
- The strongest provisional wedge is OpenClaw-first, local-first coding-agent workstation health checks and drift baselines.
- Validation should be paid and narrow: sell a 7-day agent health pilot before building a full SaaS.
- The product should be an async, fast, tidy MCP/CLI that self-configures on first run.
- OpenTelemetry concepts should shape the data model, but OpenTelemetry must remain optional.
- Current buyers are OpenClaw users, agency owners, CTOs, and founder-CTOs.
- Drift checks do not need to be deterministic. The core value is detecting behavioral changes across repeated prompts, user/project memory, speed, substance, style, and tool reliability.
- Personality drift can be scored as behavior drift: verbosity, warmth, directness, sycophancy, pushback, substance consistency, and user-style adherence.

## State
### Done
- [x] Bead 1: Baseline.ai market, competitor, positioning, and smoke-test research
- [x] Bead 2: Refined OpenClaw-first MCP/CLI MVP direction
- [x] Bead 3: Safety and eval shape for Baseline MCP/CLI
- [x] Bead 4: Proconsult-attempted product shaping into smallest defensible v0
- [x] Bead 5: Fixed Proconsult browser login path and incorporated successful consult

### Now
- Bead 5 complete; Proconsult output captured in `PROCONSULT_BASELINE_V0.md` and incorporated into `BASELINE_V0_SHAPE.md`.

### Next
- Build the first Go CLI/MCP prototype.
- Create fail-first tests for OpenClaw runner fallback behavior, config lanes, local storage, scrubber behavior, known-good diff, paginated MCP tool hashing, scoring deltas, and alert thresholds.

## Open Questions
- Which OpenClaw config paths and MCP registration flow should be supported first?
- Which alert destination matters first after OpenClaw native alerts: Slack, GitHub Checks, email, or OpenTelemetry export?
- What is the smallest useful built-in prompt pack for OpenClaw users, agency owners, and CTOs?
- Should cloud sync be opt-in during `baseline init`, or a separate `baseline sync on` command only?
- What is the minimum acceptable dashboard: single workspace timeline, or compare view plus token management?

## Working Set
- `/Users/future/.openclaw/workspace/repos/skills-library/recipes/00-kill-gate.prose.md`
- `/Users/future/.openclaw/workspace/repos/skills-library/recipes/03-offer-smoke-test.prose.md`
- `/Users/future/.openclaw/workspace/repos/skills-library/skills/app-intel`
- `/Users/future/dev/baseline/BASELINE_MVP.md`
- `/Users/future/dev/baseline/BASELINE_V0_SHAPE.md`
- `/Users/future/dev/baseline/PROCONSULT_BASELINE_V0.md`


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
