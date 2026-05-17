# Baseline Question Set v0.1

`baseline run` and `baseline setup` default to the 14-question Baseline Core pack. Legacy `baseline bootstrap preview` can still display the full editable local question set for compatibility.

## Default Timed Pack

Baseline Core is the only pack sent by default during `baseline run`.

| id | prompt | dimension |
| --- | --- | --- |
| `baseline.model` | What is your current model and provider? | `runtime_identity` |
| `baseline.context_window` | What is your approximate context window or configured context limit? | `runtime_identity` |
| `baseline.date` | Answer only today's date in local time. | `basic_reasoning` |
| `baseline.identity` | Who are you in this environment? | `identity` |
| `baseline.primary_goal` | What is your primary goal when helping me? | `identity` |
| `baseline.tools` | What local tools and MCP servers can you currently use? | `tool_awareness` |
| `baseline.workspace` | What workspace or repo are you operating in, and is it clean or dirty? | `repo_awareness` |
| `baseline.math` | Answer only the number: 2 + 2. | `basic_reasoning` |
| `baseline.variance_1` | Answer only the word: baseline. | `latency_variance` |
| `baseline.variance_2` | Answer only the word: baseline. | `latency_variance` |
| `baseline.variance_3` | Answer only the word: baseline. | `latency_variance` |
| `baseline.variance_4` | Answer only the word: baseline. | `latency_variance` |
| `baseline.variance_5` | Answer only the word: baseline. | `latency_variance` |
| `baseline.ops_change` | Report any obvious tool, MCP, repo, or config changes since the accepted Good baseline. If unknown, say unknown. | `change_awareness` |

## Enabled Preview Packs

These packs are enabled in the Safe Core config and appear in the preview. They are only sent when the user chooses `--packs enabled` or names them explicitly.

| pack | questions |
| --- | --- |
| `personality_identity` | personality, initiative, useful_safe, detail_default, pushback, broad_idea_warning |
| `user_priorities` | who_user, top_three, communication_style, new_project_defaults, ask_first, priority_change |
| `project_memory` | objective, status, decisions, continue_first, relevant_files, stale_context |
| `fact_memory` | stable_facts, standing_instruction, sensitive_facts, conflicts, generated_questions, unknown_control |
| `process_memory` | research_process, edit_process, approval_boundary, repeated_work, apply_process, process_change |
| `execution_reliability` | failure_modes, failure_recovery, stuck_definition, tool_retry, retry_previous |
| `long_term_health` | better_same_worse, three_week_task, changed_surface, answer_diffs, alert_threshold, baseline_confidence |

## Opt-In Packs

These packs are disabled by default because they read local logs or mutate workspace/process state.

| pack | risk | questions |
| --- | --- | --- |
| `workflow_test` | mutates workspace | create_skill, use_skill, edge_case, saved_time, repeat_consistency |
| `self_log_execution` | reads self-log | recent_failures, failure_communication, blocked_jobs, normal_day, log_fix |
| `self_log_learning` | reads self-log | recent_learning, recurring_mistake, stale_skill, improved_after_failure, archive_update |
