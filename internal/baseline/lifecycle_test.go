package baseline

import (
	"testing"
	"time"
)

func TestReadRunLifecycleStatusMarksMissingProcessFailed(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	status := RunLifecycleStatus{
		RunID:     "run_stale",
		Mode:      "run",
		State:     "running",
		PID:       999999,
		Packs:     "enabled",
		Questions: 55,
		StartedAt: time.Now().UTC(),
		NextActions: []string{
			"Wait for the run to complete",
			"Then run baseline report run_stale",
		},
	}
	if err := writeRunLifecycleStatus(status); err != nil {
		t.Fatal(err)
	}
	read, err := readRunLifecycleStatus("run_stale")
	if err != nil {
		t.Fatal(err)
	}
	if read.State != "failed" {
		t.Fatalf("expected stale running status to become failed, got %+v", read)
	}
	if read.Packs != "enabled" || read.Questions != 55 {
		t.Fatalf("expected lifecycle plan to be preserved, got %+v", read)
	}
	if read.Error == "" {
		t.Fatalf("expected stale lifecycle error")
	}
}
