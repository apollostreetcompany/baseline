package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runArtifactDir(runID string) string {
	return filepath.Join(baseDir(), "reports", runID)
}

func runArtifactPaths(runID string) RunArtifacts {
	dir := runArtifactDir(runID)
	return RunArtifacts{
		ReportPath:    filepath.Join(dir, "REPORT.md"),
		ResponsesPath: filepath.Join(dir, "RESPONSES.md"),
		ReceiptPath:   filepath.Join(dir, "RECEIPT.md"),
		MetricsPath:   filepath.Join(dir, "metrics.json"),
		JSONPath:      filepath.Join(dir, "run.json"),
	}
}

func writeRunArtifacts(run Run) (RunArtifacts, error) {
	if err := ensureDirs(); err != nil {
		return RunArtifacts{}, err
	}
	dir := runArtifactDir(run.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return RunArtifacts{}, err
	}
	artifacts := runArtifactPaths(run.ID)
	run.Artifacts = artifacts
	b, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return RunArtifacts{}, err
	}
	if err := atomicWrite(artifacts.JSONPath, b, 0o600); err != nil {
		return RunArtifacts{}, err
	}
	if err := atomicWrite(filepath.Join(baseDir(), "reports", run.ID+".json"), b, 0o600); err != nil {
		return RunArtifacts{}, err
	}
	metrics, err := json.MarshalIndent(runMetrics(run), "", "  ")
	if err != nil {
		return RunArtifacts{}, err
	}
	if err := atomicWrite(artifacts.MetricsPath, metrics, 0o600); err != nil {
		return RunArtifacts{}, err
	}
	if err := atomicWrite(artifacts.ReportPath, []byte(renderRunReportMarkdown(run)), 0o600); err != nil {
		return RunArtifacts{}, err
	}
	if err := atomicWrite(artifacts.ResponsesPath, []byte(renderResponsesMarkdown(run)), 0o600); err != nil {
		return RunArtifacts{}, err
	}
	if err := atomicWrite(artifacts.ReceiptPath, []byte(renderReceiptMarkdown(run)), 0o600); err != nil {
		return RunArtifacts{}, err
	}
	return artifacts, nil
}

func runMetrics(run Run) map[string]any {
	questionCount := 0
	latencyTotal := int64(0)
	for _, check := range run.Checks {
		if strings.HasPrefix(check.CheckID, "question.") {
			questionCount++
			latencyTotal += check.DurationMS
		}
	}
	avgLatency := int64(0)
	if questionCount > 0 {
		avgLatency = latencyTotal / int64(questionCount)
	}
	return map[string]any{
		"run_id":               run.ID,
		"mode":                 run.Mode,
		"status":               run.Status,
		"health_score":         run.HealthScore,
		"duration_ms":          run.DurationMS,
		"agent_kind":           run.AgentKind,
		"question_count":       questionCount,
		"avg_question_ms":      avgLatency,
		"redaction_status":     run.RedactionStatus,
		"cloud_synced":         run.CloudSynced,
		"raw_exported_cloud":   run.RawExported,
		"local_response_count": len(run.Responses),
	}
}

func renderRunReportMarkdown(run Run) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Baseline Report: %s\n\n", run.ID)
	fmt.Fprintf(&b, "- Status: `%s`\n", run.Status)
	fmt.Fprintf(&b, "- Health score: `%d`\n", run.HealthScore)
	fmt.Fprintf(&b, "- Mode: `%s`\n", run.Mode)
	fmt.Fprintf(&b, "- Agent runtime: `%s`\n", run.AgentKind)
	fmt.Fprintf(&b, "- Workspace: `%s`\n", run.Workspace)
	fmt.Fprintf(&b, "- Started: `%s`\n", run.StartedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(&b, "- Duration: `%dms`\n", run.DurationMS)
	fmt.Fprintf(&b, "- Redaction: `%s`\n", run.RedactionStatus)
	fmt.Fprintf(&b, "- Cloud synced: `%t`\n", run.CloudSynced)
	fmt.Fprintf(&b, "- Raw cloud export: `%t`\n\n", run.RawExported)

	b.WriteString("## Findings\n\n")
	if len(run.Findings) == 0 {
		b.WriteString("No findings.\n\n")
	} else {
		for _, finding := range run.Findings {
			fmt.Fprintf(&b, "- **%s** `%s`: %s\n", strings.ToUpper(finding.Severity), finding.CheckID, finding.Message)
			if finding.Fix != "" {
				fmt.Fprintf(&b, "  Fix: %s\n", finding.Fix)
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## Checks\n\n")
	for _, check := range run.Checks {
		fmt.Fprintf(&b, "- `%s` %s score %.0f in %dms: %s\n", check.CheckID, check.Status, check.Score, check.DurationMS, check.Finding)
	}
	b.WriteString("\n")

	b.WriteString("## Operator Decision\n\n")
	fmt.Fprintf(&b, "Review `RESPONSES.md` before accepting this as a Good Baseline.\n\n")
	fmt.Fprintf(&b, "- Accept: `baseline accept %s --confirm \"accept %s\"`\n", run.ID, run.ID)
	fmt.Fprintf(&b, "- Defer: keep the report and rerun after fixes.\n")
	fmt.Fprintf(&b, "- Reject: do not accept this run; keep it as failure evidence.\n")
	return b.String()
}

func renderResponsesMarkdown(run Run) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Baseline Responses: %s\n\n", run.ID)
	if len(run.Responses) == 0 {
		b.WriteString("No agent responses were recorded for this run.\n")
		return b.String()
	}
	for i, response := range run.Responses {
		fmt.Fprintf(&b, "## %d. %s/%s\n\n", i+1, response.PackID, response.ProbeID)
		fmt.Fprintf(&b, "- Status: `%s`\n", response.Status)
		fmt.Fprintf(&b, "- Dimension: `%s`\n", response.Dimension)
		fmt.Fprintf(&b, "- Duration: `%dms`\n", response.DurationMS)
		if response.Error != "" {
			fmt.Fprintf(&b, "- Error: `%s`\n", response.Error)
		}
		b.WriteString("\nPrompt:\n\n````text\n")
		b.WriteString(response.Prompt)
		b.WriteString("\n````\n\nResponse:\n\n````text\n")
		if response.Output != "" {
			b.WriteString(response.Output)
		} else if response.Error != "" {
			b.WriteString(response.Error)
		} else {
			b.WriteString("(empty response)")
		}
		b.WriteString("\n````\n\n")
	}
	return b.String()
}

func renderReceiptMarkdown(run Run) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Baseline Receipt: %s\n\n", run.ID)
	fmt.Fprintf(&b, "- Last completed step: report artifacts written\n")
	fmt.Fprintf(&b, "- Status: `%s`\n", run.Status)
	fmt.Fprintf(&b, "- Score: `%d`\n", run.HealthScore)
	fmt.Fprintf(&b, "- Report: `%s`\n", runArtifactPaths(run.ID).ReportPath)
	fmt.Fprintf(&b, "- Responses: `%s`\n\n", runArtifactPaths(run.ID).ResponsesPath)
	if run.Status == "ok" || run.Status == "warning" {
		fmt.Fprintf(&b, "Next command: `baseline report %s`\n\n", run.ID)
		fmt.Fprintf(&b, "Accept only after operator review: `baseline accept %s --confirm \"accept %s\"`\n", run.ID, run.ID)
	} else {
		fmt.Fprintf(&b, "Next command: `baseline doctor`, then rerun `baseline run` after the target/config issue is fixed.\n")
	}
	return b.String()
}
