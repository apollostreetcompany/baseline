package baseline

import (
	"bytes"
	"os"
	"strings"
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
	if !strings.Contains(strings.Join(read.NextActions, "\n"), "baseline rerun run_stale") {
		t.Fatalf("expected rerun action, got %+v", read.NextActions)
	}
}

func TestReportJSONLifecycleReturnsNonZeroForFailedOrRunning(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	if db, err := openDB(); err != nil {
		t.Fatal(err)
	} else {
		db.Close()
	}
	failed := RunLifecycleStatus{
		RunID:     "run_failed",
		Mode:      "run",
		State:     "failed",
		Packs:     "enabled",
		Questions: 55,
		Error:     "boom",
		StartedAt: time.Now().UTC(),
	}
	if err := writeRunLifecycleStatus(failed); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := cmdReport([]string{"run_failed", "--json"}, &out, &errOut); code != 1 {
		t.Fatalf("failed lifecycle JSON report should exit 1, got %d stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), `"state": "failed"`) {
		t.Fatalf("expected failed status JSON, got %s", out.String())
	}

	running := RunLifecycleStatus{
		RunID:     "run_running",
		Mode:      "run",
		State:     "running",
		PID:       os.Getpid(),
		Packs:     "enabled",
		Questions: 55,
		StartedAt: time.Now().UTC(),
	}
	if err := writeRunLifecycleStatus(running); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := cmdReport([]string{"run_running", "--json"}, &out, &errOut); code != 2 {
		t.Fatalf("running lifecycle JSON report should exit 2, got %d stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
}
