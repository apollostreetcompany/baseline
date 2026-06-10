package baseline

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestMainPrintsVersionForLongFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected version success, code=%d stderr=%s", code, stderr.String())
	}
	if got, want := strings.TrimSpace(stdout.String()), "baseline 0.1.0"; got != want {
		t.Fatalf("version output mismatch: got %q want %q", got, want)
	}
}

func TestMainPrintsVersionForShortFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"-v"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected version success, code=%d stderr=%s", code, stderr.String())
	}
	if got, want := strings.TrimSpace(stdout.String()), "baseline 0.1.0"; got != want {
		t.Fatalf("version output mismatch: got %q want %q", got, want)
	}
}

func TestMainPrintsVersionForCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected version success, code=%d stderr=%s", code, stderr.String())
	}
	if got, want := strings.TrimSpace(stdout.String()), "baseline 0.1.0"; got != want {
		t.Fatalf("version output mismatch: got %q want %q", got, want)
	}
}

func TestRunCLIStartsLongNonInteractiveRunsInBackground(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_ASYNC_EXE", "/bin/echo")
	cfg := defaultConfig()
	cfg.Target.Runtime = "custom"
	cfg.Target.Packs = "enabled"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cmdRun(t.Context(), []string{"--packs", "enabled"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected background start success, code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{"Started Baseline", "in the background", "questions=55", "Poll: baseline report", "Accept only after review"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in output:\n%s", want, text)
		}
	}
}

func TestRerunStartsNewBackgroundRunFromFailedLifecycle(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_ASYNC_EXE", "/bin/echo")
	cfg := defaultConfig()
	cfg.Target.Runtime = "custom"
	cfg.Target.Packs = "enabled"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	source := RunLifecycleStatus{
		RunID:     "run_old",
		Mode:      "run",
		State:     "failed",
		Packs:     "enabled",
		Questions: 55,
		Error:     "old failure",
		StartedAt: time.Now().UTC(),
	}
	if err := writeRunLifecycleStatus(source); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := cmdRerun([]string{"run_old"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected rerun start success, code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{"Rerunning Baseline run_old as run_", "Started Baseline", "questions=55", "Poll: baseline report"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in output:\n%s", want, text)
		}
	}
}
