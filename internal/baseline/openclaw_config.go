package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const openClawCodexTimeoutMS = 900_000

type OpenClawCodexTimeoutStatus struct {
	ConfigPath                  string `json:"config_path"`
	SnapshotPath                string `json:"snapshot_path,omitempty"`
	Applied                     bool   `json:"applied"`
	Changed                     bool   `json:"changed"`
	Skipped                     bool   `json:"skipped"`
	Reason                      string `json:"reason,omitempty"`
	RequestTimeoutMS            int    `json:"request_timeout_ms"`
	TurnCompletionIdleTimeoutMS int    `json:"turn_completion_idle_timeout_ms"`
}

func ensureOpenClawCodexTimeout() (OpenClawCodexTimeoutStatus, error) {
	status := OpenClawCodexTimeoutStatus{
		ConfigPath:                  filepath.Join(homeDir(), ".openclaw", "openclaw.json"),
		RequestTimeoutMS:            openClawCodexTimeoutMS,
		TurnCompletionIdleTimeoutMS: openClawCodexTimeoutMS,
	}
	info, statErr := os.Stat(status.ConfigPath)
	if errors.Is(statErr, os.ErrNotExist) {
		status.Skipped = true
		status.Reason = "OpenClaw config not found; run OpenClaw once, then rerun baseline setup or baseline install openclaw."
		return status, nil
	}
	if statErr != nil {
		return status, statErr
	}
	b, err := os.ReadFile(status.ConfigPath)
	if err != nil {
		return status, err
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return status, fmt.Errorf("parse OpenClaw config %s: %w", status.ConfigPath, err)
	}
	appServer := ensureJSONPath(root, "plugins", "entries", "codex", "config", "appServer")
	if setMinimumJSONInt(appServer, "requestTimeoutMs", openClawCodexTimeoutMS) {
		status.Changed = true
	}
	if setMinimumJSONInt(appServer, "turnCompletionIdleTimeoutMs", openClawCodexTimeoutMS) {
		status.Changed = true
	}
	status.Applied = true
	if !status.Changed {
		status.Reason = "OpenClaw Codex app-server timeouts are already at least 900 seconds."
		return status, nil
	}
	snapshotPath, err := snapshotOpenClawConfig(b)
	if err != nil {
		return status, err
	}
	status.SnapshotPath = snapshotPath
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return status, err
	}
	out = append(out, '\n')
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o600
	}
	if err := atomicWrite(status.ConfigPath, out, mode); err != nil {
		return status, err
	}
	status.Reason = "Set OpenClaw Codex app-server request and turn-idle timeouts to at least 900 seconds."
	return status, nil
}

func ensureJSONPath(root map[string]any, keys ...string) map[string]any {
	current := root
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	return current
}

func setMinimumJSONInt(m map[string]any, key string, minimum int) bool {
	if current, ok := jsonNumberAsInt(m[key]); ok && current >= minimum {
		return false
	}
	m[key] = minimum
	return true
}

func jsonNumberAsInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

func snapshotOpenClawConfig(contents []byte) (string, error) {
	dir := filepath.Join(baseDir(), "snapshots")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "openclaw-"+time.Now().UTC().Format("20060102T150405Z")+".json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
