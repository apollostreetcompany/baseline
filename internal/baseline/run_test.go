package baseline

import (
	"context"
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
