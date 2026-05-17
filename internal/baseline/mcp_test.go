package baseline

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPToolsListIsSmallAndIncludesCoreTools(t *testing.T) {
	tools := mcpTools()
	if len(tools) > 7 {
		t.Fatalf("MCP tool list should stay legible, got %d tools", len(tools))
	}
	seen := map[string]bool{}
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		seen[name] = true
	}
	for _, want := range []string{"baseline_setup", "baseline_run", "baseline_doctor", "baseline_report", "baseline_accept", "baseline_schedule", "baseline_scrub_preview"} {
		if !seen[want] {
			t.Fatalf("missing MCP tool %s in %+v", want, seen)
		}
	}
}

func TestMCPScheduleRunTriggersConfiguredEval(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_ASYNC_EXE", "/bin/echo")
	payload, err := callMCPTool("baseline_schedule", map[string]any{"action": "run"})
	if err != nil {
		t.Fatal(err)
	}
	text := mcpText(t, payload)
	if !strings.Contains(text, `"state": "running"`) || !strings.Contains(text, `"run_id":`) || !strings.Contains(text, `"mode": "run"`) {
		t.Fatalf("expected schedule run payload, got %s", text)
	}
}

func TestMCPRunStartsAsyncAndReportCanSeeRunningStatus(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_ASYNC_EXE", "/bin/echo")
	payload, err := callMCPTool("baseline_run", map[string]any{"packs": "baseline"})
	if err != nil {
		t.Fatal(err)
	}
	text := mcpText(t, payload)
	if !strings.Contains(text, `"state": "running"`) {
		t.Fatalf("expected async running payload, got %s", text)
	}
	var envelope struct {
		RunStatus struct {
			RunID string `json:"run_id"`
		} `json:"run_status"`
	}
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatal(err)
	}
	reportPayload, err := callMCPTool("baseline_report", map[string]any{"run_id": envelope.RunStatus.RunID})
	if err != nil {
		t.Fatal(err)
	}
	reportText := mcpText(t, reportPayload)
	if !strings.Contains(reportText, `"run_status"`) {
		t.Fatalf("expected report to return lifecycle status, got %s", reportText)
	}
}

func mcpText(t *testing.T, payload any) string {
	t.Helper()
	result, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected MCP payload: %#v", payload)
	}
	content, ok := result["content"].([]map[string]string)
	if !ok || len(content) == 0 {
		t.Fatalf("missing MCP content: %#v", payload)
	}
	return content[0]["text"]
}
