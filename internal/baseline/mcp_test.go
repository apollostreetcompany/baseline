package baseline

import (
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
	cfg := defaultConfig()
	cfg.Target.Runtime = "custom"
	cfg.AgentCommand = "printf baseline"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	payload, err := callMCPTool("baseline_schedule", map[string]any{"action": "run"})
	if err != nil {
		t.Fatal(err)
	}
	text := mcpText(t, payload)
	if !strings.Contains(text, `"action": "run"`) || !strings.Contains(text, `"run_id":`) || !strings.Contains(text, `"mode": "run"`) {
		t.Fatalf("expected schedule run payload, got %s", text)
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
