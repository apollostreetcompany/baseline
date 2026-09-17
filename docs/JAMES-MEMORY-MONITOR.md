# James snapshot recall monitor

`baseline memory` measures the current injected Hermes memory without using the
ordinary Baseline configuration, cloud sync, probe packs, Good acceptance, or scheduler.
It is **snapshot_recall**, not Oracle retrieval, session search, write-back, or a
long-term retention test. One fresh batched model turn answers four to six cases.

Use a dedicated absolute `BASELINE_HOME` outside Git workspaces and the source
profile. The default `~/.baseline`, symlinked homes, and homes containing a stock
`config.json` are refused. Do not run stock `setup` or `run` for this monitor.

```sh
BASELINE_HOME=/Users/future/.baseline-james-memory /path/to/dedicated/baseline memory run \
  --source-profile /Users/future/.hermes/profiles/orchestration \
  --cases /private/path/cases.json \
  --request-id manual-pilot-001 \
  --hermes-python /Users/future/.hermes/hermes-agent/venv/bin/python \
  --hermes-source /Users/future/.hermes/hermes-agent \
  --timeout 90

BASELINE_HOME=/Users/future/.baseline-james-memory /path/to/dedicated/baseline memory history
```

Instead of `--request-id`, a separately installed daily timer can supply
`--daily-at 08:50 --timezone Asia/Tokyo`. Before that time, the command returns
`not_due` without invoking Hermes. After it, the date in that IANA zone plus
source/case paths identifies the day's request. Repeated ticks return its retained
measurement; changed gold under the same request is a conflict, not another call.
No timer is installed by this command. Offline/missed-day alerting is not implemented.

A host deadline is 90 seconds by default (1–300 allowed). A nonblocking local lock
serializes invocations. A request directory is created before execution; a crash
or unfinished request never silently runs again. History labels unfinished requests
INCONCLUSIVE. Investigate retained state before explicitly choosing a new ID.

## Held-out cases

A private JSON file holds both questions and deterministic gold. Only IDs and
questions go into the model query. Keep the gold outside the source profile.

```json
{
  "version": "james-memory-v1",
  "source_revision": "profile-activation-reference",
  "gold_revision": "original-user-instruction-reference",
  "cases": [
    {
      "id": "timezone",
      "question": "What is the planning timezone?",
      "expected": ["Asia/Tokyo"],
      "source_file": "memories/USER.md",
      "evidence": ["An exact supporting quote from the authorized source."]
    },
    {
      "id": "unknown-control",
      "question": "What is my invented control code?",
      "expected_unknown": true
    }
  ]
}
```

This abbreviated example is illustrative; a runnable batch requires **4–6** unique
cases. Known cases require allowed value strings, exact supporting evidence anchors, and one
of `memories/MEMORY.md`, `memories/USER.md`, or `SOUL.md`. Unknown cases must omit
known gold. The response must be one JSON object:

```json
{"answers":[{"id":"timezone","value":"Asia/Tokyo","quote":"Exact source quote"},{"id":"unknown-control","value":null,"quote":""}]}
```

Values are compared with case/whitespace normalization and otherwise exact equality.
A known answer also needs a nonempty quote that is an exact substring of its
specified **current** snapshot file and contains an approved supporting evidence
anchor. A longer contiguous source quote is allowed. A quote from historical gold that has disappeared from current memory
does not pass. Null on a known case is `missing_recall`; answering an unknown is
`unsupported_answer`. Invalid/missing answers or infrastructure failures yield
INCONCLUSIVE. A complete valid batch with an incorrect answer yields FAIL.
Neither is converted into PASS to hide missing memory.

## Isolation and runtime contract

The command copies only the three context files into a private new profile at
`~/.hermes/.baseline-evals/<dedicated-monitor-home SHA256>/<run-id>`. The
`evaluation_profile` result field records its exact path; reports, SQLite and gold
stay in `BASELINE_HOME`. The profile is outside the normal `profiles/` directory.
Its ancestry must be real directories outside Git workspaces, and its dedicated
parent directories must be private (0700). Existing evaluation directories are
never overwritten. It
records source `config.yaml` size/hash but never copies it. It does not copy the
production session database, skills, auth, plugins, hooks, or MCP configuration.
All source files must be bounded regular files without symlinks. A before/after
source hash check detects concurrent context edits. The production session database
is never opened by the monitor.

A small Python preflight uses the installed Hermes source to verify James's source
model/provider/reasoning (`gpt-6-astra-900k`, `openai-codex`, `medium`), generate the
minimal isolated config, and resolve **zero native tool definitions**. It then
executes unmodified native Hermes:

```text
python <same-venv>/bin/hermes --profile default chat --max-turns 1 --run-budget 90 \
  --format stream-json --query-file PRIVATE_QUERY --source baseline-memory \
  --model gpt-6-astra-900k --provider openai-codex --reasoning medium
```

The installed console script beside `--hermes-python` is required. The explicit
`--profile default` selector precedes `chat` and preserves the evaluation
`HERMES_HOME`, bypassing sticky profile selection without changing it. Python
`-m hermes_cli.main` is not used: on the verified runtime it reapplies sticky
selection even with that flag. No constructor monkeypatch is used. The
environment retains only HOME/PATH, locale,
selected certificate/temp settings, and explicit isolation settings. Inherited
Hermes profile/task/provider variables are removed. `HERMES_SAFE_MODE=1` suppresses
plugins, MCP, hooks, and webhooks; the CLI `--safe-mode` is intentionally avoided
because it also disables the memory being measured. Toolsets are empty with native
hard suppression, external memory provider is empty, background review/compression
and routing/fallback are disabled. `security.tirith_enabled: false` disables the
optional command scanner only in this zero-tool profile. There are no terminal
commands to scan; this also prevents scanner installation attempts and its native
startup warning from contaminating structured stdout. The strict stream parser
and all production security settings remain unchanged. A Git ancestor is refused so context discovery
cannot inherit a repository's AGENTS file. A source-checkout `.env` fallback is
refused because native startup could load it and override isolation.

The source is an operator-trusted installed runtime, not an OS sandbox. Preflight
verifies that native `get_default_hermes_root()` resolves to `~/.hermes`: a profile
outside that root would incorrectly become its own credential root. Native auth
uses the existing global credential fallback and may perform normal OAuth refresh;
credentials are neither copied nor logged by Baseline. Every model session/state
write has a fresh isolated `HERMES_HOME`. Unexpected native tool events invalidate
the measurement. The outer deadline kills the entire isolated process group.

The native `stream-json` terminal result provides tokens. Model text cannot supply
usage. A nonzero terminal result preserves its native session ID, marks `failed=true`,
and retains any native token counts even though the measurement is INCONCLUSIVE.
Missing/all-zero telemetry is reported as unavailable; native cost and API
call count are unavailable in this protocol. Provider/reasoning are configured
provenance, while the native initialization model is checked. One model turn is
bounded; provider-internal network retries are not a claim of one HTTP request.

## History and evidence

Existing SQLite `runs`, `check_results`, `observations`, and `probe_messages` carry
measurements, provenance and usage once per **batch**, without a schema change.
Existing private run artifacts plus `memory.json`, `started.json`, `output.txt`,
`stderr.txt`, and the isolated profile preserve failed as well as successful runs.
The source hashes/sizes, source/gold revision references, complete case hash, runtime
Git revision and tracked-diff hash, generated-preflight hash, model and usage are retained. Raw local
context/output is private; no cloud outbox entry is created.

`memory history` derives its readable log from SQLite and unfinished run artifacts.
Score deltas compare only identical profile/case/gold/model/preflight/runtime identities;
source context can change so regressions remain observable. INCONCLUSIVE runs do not
become score baselines. There is no second longitudinal database or LLM judge.

Controlled fake-Hermes tests establish grading, isolation contract, duplicate and
conflicting requests, timeouts, lock contention, failure retention, native event
parsing, daily timezone logic, and comparable history. A read-only native tool
preflight validates the installed resolver. Neither proves production recall;
reviewed real-model pilots, installation and timer acceptance belong to deployment.
Weekly write fixtures and production historical retrieval remain deferred.
