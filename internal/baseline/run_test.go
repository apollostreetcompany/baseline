package baseline

import (
	"context"
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
