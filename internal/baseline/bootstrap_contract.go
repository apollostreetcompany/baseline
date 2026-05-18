package baseline

import (
	"fmt"
	"os"
	"strings"
)

func writeBootstrapContract(cfg Config) error {
	if err := ensureDirs(); err != nil {
		return err
	}
	return atomicWrite(bootstrapContractPath(), []byte(renderBootstrapContract(cfg)), 0o600)
}

func renderBootstrapContract(cfg Config) string {
	var b strings.Builder
	b.WriteString("# Baseline Agent Bootstrap\n\n")
	b.WriteString("Baseline evaluates the operator-approved agent setup. The agent may run Baseline, explain failures, and ask for a decision. The agent must not silently accept or overwrite a Good Baseline.\n\n")
	b.WriteString("## Current Target\n\n")
	fmt.Fprintf(&b, "- Runtime: `%s`\n", cfg.Target.Runtime)
	fmt.Fprintf(&b, "- Entity: `%s`\n", cfg.Target.Entity)
	fmt.Fprintf(&b, "- Model policy: `%s` (%s)\n", cfg.Target.ModelPolicy, targetModelDisplay(cfg.Target))
	fmt.Fprintf(&b, "- Packs: `%s`\n", cfg.Target.Packs)
	fmt.Fprintf(&b, "- Timeout: `%ds`\n\n", targetTimeoutSeconds(cfg.Target))
	b.WriteString("## Commands\n\n")
	b.WriteString("- `baseline setup`: first-run setup, preflight, real eval, report artifacts.\n")
	b.WriteString("- `baseline run`: normal real eval of the default target. Long non-interactive runs start in the background and print a run id so agent turns do not time out.\n")
	b.WriteString("- `baseline doctor`: read-only preflight/troubleshooting. This is not a Good Baseline candidate by itself.\n")
	b.WriteString("- `baseline report [RUN_ID]`: print the markdown report and recorded local responses.\n")
	b.WriteString("- `baseline accept RUN_ID --confirm \"accept RUN_ID\"`: accept only after operator review.\n")
	b.WriteString("- `baseline status`: show target, latest run, Good Baselines, and schedule state.\n\n")
	b.WriteString("Baseline runs agent questions serially by default so timing reflects the evaluated agent, not local runner contention. `BASELINE_PROBE_CONCURRENCY` is an advanced override.\n\n")
	b.WriteString("## Autonomy Modes\n\n")
	b.WriteString("- `observe`: inspect status/report only.\n")
	b.WriteString("- `run`: run `baseline doctor` or `baseline run`, then report results.\n")
	b.WriteString("- `repair-propose`: propose config or install changes, but do not apply them.\n")
	b.WriteString("- `repair-allowed`: apply only the specific operator-approved repair, then rerun `baseline doctor`.\n\n")
	b.WriteString("## State Machine\n\n")
	b.WriteString("`preflight -> eval -> report -> compare -> operator_decision -> accept/reject/defer`\n\n")
	b.WriteString("If a step fails, stop after the report or receipt is written. Tell the operator the failing step, the likely cause, and the next command.\n\n")
	b.WriteString("## Error Handling\n\n")
	b.WriteString("- Missing runtime: explain which binary/config is missing and suggest `baseline doctor` after repair.\n")
	b.WriteString("- Model/auth failure: store the failed run, skip acceptance, and ask the operator whether to pin a different model or follow the current agent model.\n")
	b.WriteString("- Timeout: first check whether the agent held a long foreground command. Use `baseline report RUN_ID` for managed background runs, store the partial responses/timing when present, and do not rerun the same failing command more than twice.\n")
	b.WriteString("- Cloud sync issue: keep local results, report redacted sync failure separately, and never enable raw sync without operator approval.\n\n")
	b.WriteString("## Retention And Privacy\n\n")
	b.WriteString("Local `RESPONSES.md` may contain full responses for operator review. Cloud sync exports redacted summaries unless `allow_raw_output` is explicitly enabled by the operator.\n")
	return b.String()
}

func bootstrapContractExists() bool {
	_, err := os.Stat(bootstrapContractPath())
	return err == nil
}
