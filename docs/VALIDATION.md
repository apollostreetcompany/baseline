# Validation Notes

## User Pain Synthesis

The v0 shape was narrowed around the strongest supplied X research signals:

- Memory and self-improvement are the highest-intensity comparison topics.
- Speed and latency are frequent enough to make timed checks mandatory.
- Serious users track practical metrics: output acceptance, blocked jobs, dedup failures, token use, and time-to-review.
- "Drift" is not always the word users use. The more resonant language is agent health, known-good, memory, speed, and "is my agent worse today?"

`xf` was attempted for live X validation, but the local `xf` command was not installed. The launch copy and probes therefore use the supplied X research plus the social-learner framing rather than fresh archive queries.

## Simulated Users

### 1. OpenClaw Power User

Judgment: irresistible after setup became one command and preflight became clearly separate from real eval.

Objection: "Do not run my agent or export transcripts without asking."

Change made: `baseline doctor` is read-only preflight, while `baseline run` is the real timed eval. `baseline accept` requires an exact operator confirmation string. Cloud sync exports only a reduced redacted payload.

### 2. Agency Owner

Judgment: close to irresistible once the dashboard and pricing were visible.

Objection: "I need to know if a client workstation is broken before it burns a deadline."

Change made: landing page leads with "got slower, forgetful, unaware, or strange"; dashboard highlights health score, alerts, known-good compare, and latency changes. Pricing is $39 Pro and $129 Team as a testable anchor.

### 3. CTO

Judgment: viable for dogfooding, not enterprise-ready yet.

Objection: "Any bearer token accepting ingest is a security miss."

Change made: Worker `/api/runs` now fails closed unless `BASELINE_API_TOKEN` matches. Unauthorized ingest returns `403`; missing token config returns `503`.

### 4. Agent Framework Maintainer

Judgment: useful if not OpenClaw-only.

Objection: "Do not make this a single-framework benchmark."

Change made: the runner supports `BASELINE_AGENT_COMMAND` / `--agent-command` with the prompt supplied as `BASELINE_PROMPT`. OpenClaw is the first adapter because the local machine has it installed.

### 5. Hermes-Style Memory Power User

Judgment: compelling if known-good drift stays the center.

Objection: "Generic evals are not the pain. I care whether the agent remembers me, the repo, the task, and its own tooling."

Change made: the 12-question pack covers identity, active task, safety constraint, repo awareness, basic reasoning, style, dedup memory, MCP/tool awareness, latency sensitivity, output acceptance, stuck job rate, and tone.

## Provisional Go / No-Go

GO for dogfooding with OpenClaw users and agency owners.

CONDITIONAL GO for CTOs: token-gated redacted sync is enough for a pilot, but teams will need org tokens, retention controls, and audit events before procurement.

NO GO for broad "LLM observability" positioning. Baseline is narrower: known-good drift checks for local coding agents.

## Next Validation Test

Ask 10 OpenClaw or Hermes power users to run:

```sh
baseline setup
baseline report
baseline accept <RUN_ID> --confirm "accept <RUN_ID>" --label clean
baseline run
baseline compare
```

Success criterion: at least 4 say they would keep the MCP installed after the first day, and at least 2 ask for cloud history or alerts.
