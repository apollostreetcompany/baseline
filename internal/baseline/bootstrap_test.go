package baseline

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBootstrapPreviewIncludesUpdatedQuestionSetAndRiskFlags(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	cfg := defaultConfig()
	preview := bootstrapPreview(cfg)
	if len(preview.Questions) < 45 {
		t.Fatalf("expected full v0.1 question set, got %d questions", len(preview.Questions))
	}
	if got := len(selectedQuestions(cfg, "baseline")); got != 14 {
		t.Fatalf("baseline timed run should stay 10-15 questions, got %d", got)
	}
	enabled := enabledMonitorPacks(cfg)
	for _, want := range []string{"baseline", "personality_identity", "user_priorities", "project_memory", "fact_memory", "process_memory", "execution_reliability", "long_term_health"} {
		if !enabled[want] {
			t.Fatalf("expected %s enabled by default", want)
		}
	}
	for _, wantDisabled := range []string{"workflow_test", "self_log_execution", "self_log_learning"} {
		if enabled[wantDisabled] {
			t.Fatalf("expected %s disabled by default", wantDisabled)
		}
	}
	var sawMutationRisk bool
	for _, pack := range preview.Packs {
		if pack.ID == "workflow_test" && pack.Risk.MutatesWorkspace {
			sawMutationRisk = true
		}
	}
	if !sawMutationRisk {
		t.Fatalf("expected workflow_test mutation risk flag")
	}
}

func TestBootstrapCandidateRequiresExplicitAcceptAndGoodSlotsAreCapped(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cfg := defaultConfig()
	scopeKey := scopeKeyForWorkspace("workspace-a")
	cfgHash := configHash(cfg)

	for i := 1; i <= 4; i++ {
		suffix := strconv.Itoa(i)
		run := Run{
			ID:              "run_slot_" + suffix,
			Mode:            "bootstrap",
			StartedAt:       time.Now().Add(time.Duration(i) * time.Second),
			Status:          "ok",
			HealthScore:     100,
			Workspace:       "workspace-a",
			AgentKind:       "openclaw",
			RedactionStatus: "clean",
			Checks:          []CheckResult{{ID: "check_slot_" + suffix, CheckID: "runtime.openclaw", Lane: "core", Kind: "environment", Status: "ok", Score: 100}},
		}
		if err := saveRun(db, run, []Observation{{Key: "probe", ValueHash: hashValue(run.ID), RedactedDisplay: run.ID}}); err != nil {
			t.Fatal(err)
		}
		if _, err := createBootstrapCandidate(db, run.ID, "candidate", "", scopeKey, cfgHash); err != nil {
			t.Fatal(err)
		}
	}
	goods, err := listGoodBaselines(db, scopeKey, cfgHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(goods) != 0 {
		t.Fatalf("candidate should not auto-create Good Baseline: %+v", goods)
	}
	for i := 1; i <= 3; i++ {
		if _, err := acceptGoodBaseline(db, "run_slot_"+strconv.Itoa(i), "good", "", 0, scopeKey, cfgHash); err != nil {
			t.Fatalf("accept slot %d: %v", i, err)
		}
	}
	if _, err := acceptGoodBaseline(db, "run_slot_4", "good", "", 0, scopeKey, cfgHash); err == nil {
		t.Fatalf("expected fourth auto Good Baseline to be rejected")
	}
	goods, err = listGoodBaselines(db, scopeKey, cfgHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(goods) != 3 {
		t.Fatalf("expected exactly 3 Good Baselines, got %+v", goods)
	}
}

func TestConfigCLISetGetAndRedactsToken(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	if code := cmdConfig([]string{"set", "api_token", "secret-token"}, &out, &errOut); code != 0 {
		t.Fatalf("config set failed: code=%d stderr=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cmdConfig([]string{"get", "api_token", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("config get failed: code=%d stderr=%s", code, errOut.String())
	}
	if strings.Contains(out.String(), "secret-token") || !strings.Contains(out.String(), "token_set") {
		t.Fatalf("token should be redacted, got %s", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cmdConfig([]string{"set", "monitor_packs.workflow_test.enabled", "true"}, &out, &errOut); code != 0 {
		t.Fatalf("pack toggle failed: code=%d stderr=%s", code, errOut.String())
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !enabledMonitorPacks(cfg)["workflow_test"] {
		t.Fatalf("workflow_test pack should be enabled by config path toggle")
	}
}

func TestMCPDoesNotAdvertiseSilentKnownGoodTool(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	tools := mcpTools()
	for _, tool := range tools {
		if tool["name"] == "baseline_mark_known_good" || tool["name"] == "baseline_config" {
			t.Fatalf("unsafe or legacy MCP tool should not be advertised: %+v", tool)
		}
	}
	payload, err := callMCPTool("baseline_mark_known_good", map[string]any{"label": "bad"})
	if err == nil {
		t.Fatalf("legacy mark-known-good call should return an error")
	}
	if payload != nil {
		t.Fatalf("unexpected payload on hard error: %+v", payload)
	}
	payload, err = callMCPTool("baseline_good", map[string]any{"action": "accept", "run_id": "run_missing"})
	if err != nil {
		t.Fatal(err)
	}
	text := mcpText(t, payload)
	if !strings.Contains(text, "requires confirm") {
		t.Fatalf("MCP accept without confirmation should be rejected, got %s", text)
	}
}

func TestBootstrapRunWithAgentCommandCreatesCandidate(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_RUN_AGENT", "1")
	var out, errOut bytes.Buffer
	if code := cmdBootstrap(context.Background(), []string{"preview"}, &out, &errOut); code != 0 {
		t.Fatalf("bootstrap preview failed: code=%d stderr=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code := cmdBootstrap(context.Background(), []string{"run", "--agent-command", "printf baseline"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("bootstrap run failed: code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"candidate"`) || !strings.Contains(out.String(), `"run"`) {
		t.Fatalf("expected candidate payload, got %s", out.String())
	}
	status, err := currentBootstrapStatus()
	if err != nil {
		t.Fatal(err)
	}
	if status.LatestCandidate == nil || !status.NeedsBootstrap {
		t.Fatalf("candidate should exist but not be accepted yet: %+v", status)
	}
}

func TestBootstrapRunRequiresPreviewBeforeAgentProbes(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_RUN_AGENT", "1")
	var out, errOut bytes.Buffer
	code := cmdBootstrap(context.Background(), []string{"run", "--agent-command", "printf baseline"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("bootstrap run should require preview before sending probes: %s", out.String())
	}
	if !strings.Contains(errOut.String(), "preview required") {
		t.Fatalf("expected preview requirement, got stderr=%s stdout=%s", errOut.String(), out.String())
	}
	status, err := currentBootstrapStatus()
	if err != nil {
		t.Fatal(err)
	}
	if status.LatestCandidate != nil {
		t.Fatalf("run without preview must not create candidate: %+v", status)
	}
}
