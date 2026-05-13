package baseline

import "testing"

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
	for _, want := range []string{"baseline_check", "baseline_compare", "baseline_scrub_preview"} {
		if !seen[want] {
			t.Fatalf("missing MCP tool %s in %+v", want, seen)
		}
	}
}
