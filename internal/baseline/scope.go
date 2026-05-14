package baseline

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

func scopeKeyForWorkspace(workspace string) string {
	if workspace == "" {
		workspace = currentWorkspace()
	}
	abs, err := filepath.Abs(workspace)
	if err == nil {
		workspace = abs
	}
	return hashValue(strings.TrimSpace(workspace))
}

func configHash(cfg Config) string {
	copy := cfg
	copy.APIToken = ""
	copy.AgentCommand = ""
	b, _ := json.Marshal(struct {
		Version      int                    `json:"version"`
		Workspace    string                 `json:"workspace_name"`
		Seeds        []MemorySeed           `json:"memory_seeds"`
		MonitorPacks []MonitorPackSelection `json:"monitor_packs"`
	}{
		Version:      copy.Version,
		Workspace:    copy.WorkspaceName,
		Seeds:        copy.MemorySeeds,
		MonitorPacks: copy.MonitorPacks,
	})
	return hashValue(string(b))
}
