package baseline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFastBaselineDoesNotRequireAgentExecution(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	run, err := RunBaseline(context.Background(), RunOptions{Mode: "fast", Workspace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == "" || len(run.Checks) == 0 {
		t.Fatalf("expected persisted check results, got %+v", run)
	}
	for _, check := range run.Checks {
		if check.Kind == "agent_eval" {
			t.Fatalf("fast baseline should not execute agent evals: %+v", check)
		}
	}
}

func TestFullBaselineDoesNotRunConfiguredAgentWithoutConsent(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	marker := filepath.Join(t.TempDir(), "agent-ran")
	cfg := defaultConfig()
	cfg.AgentCommand = "touch " + marker
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	run, err := RunBaseline(context.Background(), RunOptions{Mode: "full", Workspace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("configured agent command ran without explicit consent")
	}
	var sawSkip bool
	for _, check := range run.Checks {
		if check.CheckID == "questions.runner" {
			sawSkip = true
		}
	}
	if !sawSkip {
		t.Fatalf("expected question runner skip warning, got %+v", run.Checks)
	}
}
