package baseline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureOpenClawCodexTimeoutPatchesConfigAndPreservesGoogle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("BASELINE_HOME", filepath.Join(home, ".baseline"))
	configPath := filepath.Join(home, ".openclaw", "openclaw.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	original := `{
  "models": {
    "providers": {
      "google": {
        "apiKey": "__OPENCLAW_REDACTED__"
      }
    }
  },
  "plugins": {
    "entries": {
      "google": {
        "enabled": true
      }
    }
  }
}
`
	if err := os.WriteFile(configPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := ensureOpenClawCodexTimeout()
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied || !status.Changed || status.SnapshotPath == "" {
		t.Fatalf("expected changed status with snapshot, got %+v", status)
	}
	if _, err := os.Stat(status.SnapshotPath); err != nil {
		t.Fatalf("expected snapshot: %v", err)
	}
	root := readJSONMap(t, configPath)
	appServer := root["plugins"].(map[string]any)["entries"].(map[string]any)["codex"].(map[string]any)["config"].(map[string]any)["appServer"].(map[string]any)
	if got := int(appServer["requestTimeoutMs"].(float64)); got != openClawCodexTimeoutMS {
		t.Fatalf("requestTimeoutMs=%d", got)
	}
	if got := int(appServer["turnCompletionIdleTimeoutMs"].(float64)); got != openClawCodexTimeoutMS {
		t.Fatalf("turnCompletionIdleTimeoutMs=%d", got)
	}
	if google := root["plugins"].(map[string]any)["entries"].(map[string]any)["google"].(map[string]any); google["enabled"] != true {
		t.Fatalf("google plugin should be preserved, got %+v", google)
	}
	if provider := root["models"].(map[string]any)["providers"].(map[string]any)["google"].(map[string]any); provider["apiKey"] != "__OPENCLAW_REDACTED__" {
		t.Fatalf("google provider should be preserved, got %+v", provider)
	}
	snapshot, err := os.ReadFile(status.SnapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot) != original {
		t.Fatalf("snapshot should contain original config")
	}
}

func TestEnsureOpenClawCodexTimeoutDoesNotLowerHigherOperatorTimeout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("BASELINE_HOME", filepath.Join(home, ".baseline"))
	configPath := filepath.Join(home, ".openclaw", "openclaw.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	config := `{"plugins":{"entries":{"codex":{"config":{"appServer":{"requestTimeoutMs":1200000,"turnCompletionIdleTimeoutMs":1200000}},"enabled":true}}}}`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := ensureOpenClawCodexTimeout()
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied || status.Changed || status.SnapshotPath != "" {
		t.Fatalf("expected no mutation for higher timeout, got %+v", status)
	}
	root := readJSONMap(t, configPath)
	appServer := root["plugins"].(map[string]any)["entries"].(map[string]any)["codex"].(map[string]any)["config"].(map[string]any)["appServer"].(map[string]any)
	if got := int(appServer["requestTimeoutMs"].(float64)); got != 1_200_000 {
		t.Fatalf("requestTimeoutMs should not be lowered, got %d", got)
	}
}

func TestBootstrapContractExplainsOpenClawTimeoutAndRedactedAuth(t *testing.T) {
	cfg := defaultConfig()
	contract := renderBootstrapContract(cfg)
	for _, want := range []string{
		"OpenClaw Codex Timeout Guardrail",
		"turnCompletionIdleTimeoutMs=900000",
		"turn_completion_idle_timeout",
		"__OPENCLAW_REDACTED__",
		"Do not remove Google/Gemini search",
	} {
		if !strings.Contains(contract, want) {
			t.Fatalf("expected bootstrap contract to contain %q:\n%s", want, contract)
		}
	}
}

func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	return root
}
