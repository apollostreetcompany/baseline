package baseline

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestBootstrapQuestionProbesUseBoundedConcurrency(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_PROBE_CONCURRENCY", "4")
	cfg := defaultConfig()
	cfg.Target.Runtime = "custom"
	cfg.AgentCommand = "sleep 0.2; printf baseline"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	run, err := RunBaseline(context.Background(), RunOptions{
		Mode:         "bootstrap",
		RunAgent:     true,
		AgentCommand: cfg.AgentCommand,
		Workspace:    "test",
		Packs:        "baseline",
	})
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	var questionChecks int
	for _, check := range run.Checks {
		if strings.HasPrefix(check.CheckID, "question.baseline.") {
			questionChecks++
		}
	}
	if questionChecks != 14 {
		t.Fatalf("expected 14 baseline question probes, got %d", questionChecks)
	}
	if elapsed > 2500*time.Millisecond {
		t.Fatalf("question probes appear sequential, elapsed=%s", elapsed)
	}
}

func TestRunModeExecutesDefaultTargetAndCapturesResponses(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	cfg := defaultConfig()
	cfg.Target.Runtime = "custom"
	cfg.AgentCommand = "printf baseline"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	run, err := RunBaseline(context.Background(), RunOptions{
		Mode:      "run",
		Workspace: "test",
		Packs:     "baseline",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Mode != "run" {
		t.Fatalf("expected run mode, got %s", run.Mode)
	}
	if len(run.Responses) != 14 {
		t.Fatalf("expected recorded local responses for operator review, got %d", len(run.Responses))
	}
	artifacts, err := writeRunArtifacts(run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifacts.ReportPath); err != nil {
		t.Fatalf("expected report artifact: %v", err)
	}
	if _, err := os.Stat(artifacts.ResponsesPath); err != nil {
		t.Fatalf("expected responses artifact: %v", err)
	}
}

func TestDoctorModeIsEphemeral(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	run, err := RunBaseline(context.Background(), RunOptions{Mode: "doctor", Ephemeral: true, Workspace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == "" || len(run.Checks) == 0 {
		t.Fatalf("expected doctor checks without persistence, got %+v", run)
	}
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := latestRun(db); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("doctor should not persist latest run, got err=%v", err)
	}
}

func TestRecordedQuestionCheckUsesProbeDuration(t *testing.T) {
	state := &runState{runID: "run_duration"}
	sendAt := time.Now().UTC()
	state.recordQuestionOutcome(questionProbeOutcome{
		Started: time.Now().Add(-1 * time.Hour),
		Question: Question{
			PackID:        "baseline",
			ID:            "math",
			Prompt:        "Answer 2 + 2.",
			ExpectedFacts: []string{"4"},
			Dimension:     "basic_reasoning",
		},
		Result: AgentProbeResult{
			Output: "4",
			ProbeMessage: ProbeMessage{
				RunID:              "run_duration",
				PackID:             "baseline",
				ProbeID:            "math",
				SystemSendAt:       sendAt,
				BaselineReceivedAt: sendAt.Add(123 * time.Millisecond),
				DurationMS:         123,
				TokenStatus:        "unavailable",
			},
		},
	})
	if len(state.checks) != 1 {
		t.Fatalf("expected one check, got %+v", state.checks)
	}
	if state.checks[0].DurationMS != 123 || state.checks[0].Metrics["duration_ms"] != 123 {
		t.Fatalf("check should use measured probe duration, got %+v", state.checks[0])
	}
}

func TestCustomAgentCommandReceivesDeadlineEnvAndRecordsHermesSessionID(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env.txt")
	promptFile := filepath.Join(dir, "prompt.txt")
	command := "printenv | grep '^BASELINE_\\|^OTEL_RESOURCE_ATTRIBUTES=' | sort > " + quoteShell(envFile) + "; printf '%s' \"$BASELINE_PROMPT\" > " + quoteShell(promptFile) + "; printf 'baseline ok\\nBASELINE_HERMES_SESSION_ID: custom_session_456\\n'"
	state := &runState{
		ctx:   context.Background(),
		runID: "run_custom",
		cfg: Config{
			AgentCommand: command,
			Target:       BaselineTarget{Runtime: "custom", TimeoutSeconds: 45},
		},
		opts: RunOptions{Workspace: dir},
	}
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=custom-test")
	result, err := state.askAgentMeasured(Question{PackID: "baseline", ID: "memory", Prompt: "What changed?"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != "baseline ok\n" {
		t.Fatalf("expected marker-stripped output, got %q", result.Output)
	}
	if result.SessionID != "custom_session_456" {
		t.Fatalf("expected custom command session id to be recorded, got %+v", result.ProbeMessage)
	}
	promptBytes, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(promptBytes), "Baseline harness observability instruction") {
		t.Fatalf("custom command prompt did not include observability instruction: %s", promptBytes)
	}
	envBytes, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	envText := string(envBytes)
	for _, want := range []string{
		"BASELINE_RUN_ID=run_custom",
		"BASELINE_PACK_ID=baseline",
		"BASELINE_PROBE_ID=memory",
		"BASELINE_TIMEOUT_SECONDS=45",
		"BASELINE_EVAL_MODE=1",
		"BASELINE_DEADLINE_AT=",
		"OTEL_RESOURCE_ATTRIBUTES=service.name=custom-test,baseline.run_id=run_custom,baseline.pack_id=baseline,baseline.probe_id=memory",
	} {
		if !strings.Contains(envText, want) {
			t.Fatalf("expected env %q in:\n%s", want, envText)
		}
	}
}

func quoteShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
