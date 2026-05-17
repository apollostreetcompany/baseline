package baseline

import "strings"

func targetTimeoutSeconds(target BaselineTarget) int {
	if target.TimeoutSeconds == 0 {
		return defaultConfig().Target.TimeoutSeconds
	}
	if target.TimeoutSeconds < 5 {
		return 5
	}
	if target.TimeoutSeconds > 900 {
		return 900
	}
	return target.TimeoutSeconds
}

func targetModelDisplay(target BaselineTarget) string {
	if target.ModelPolicy == "pinned" && strings.TrimSpace(target.PinnedModel) != "" {
		return "pinned model " + strings.TrimSpace(target.PinnedModel)
	}
	return "the agent's current model"
}

func targetAgentName(target BaselineTarget) string {
	entity := strings.TrimSpace(target.Entity)
	if strings.HasPrefix(entity, "agent:") {
		entity = strings.TrimSpace(strings.TrimPrefix(entity, "agent:"))
	}
	if entity == "" {
		return "main"
	}
	return entity
}
