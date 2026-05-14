package baseline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKnownGoodCompareFindsChangedObservation(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	first := Run{
		ID:              "run_first",
		Mode:            "fast",
		StartedAt:       time.Now(),
		Status:          "ok",
		HealthScore:     100,
		Workspace:       "test",
		AgentKind:       "test",
		RedactionStatus: "clean",
		Checks:          []CheckResult{{ID: "001", CheckID: "repo.state", Lane: "core", Kind: "awareness", Status: "ok", Score: 100}},
	}
	firstObs := []Observation{{Key: "repo.branch", ValueHash: hashValue("main"), RedactedDisplay: "main"}}
	if err := saveRun(db, first, firstObs); err != nil {
		t.Fatal(err)
	}
	if err := markKnownGood(db, first.ID, "accepted"); err != nil {
		t.Fatal(err)
	}

	changedObs := []Observation{{Key: "repo.branch", ValueHash: hashValue("feature"), RedactedDisplay: "feature"}}
	findings, err := compareObservationsToKnownGood(db, changedObs)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].CheckID != "known_good.diff" {
		t.Fatalf("unexpected finding: %+v", findings[0])
	}
}

func TestCloudSyncFailureLeavesRetryableOutboxRow(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	cfg := defaultConfig()
	cfg.CloudSync = true
	cfg.APIBaseURL = server.URL
	cfg.APIToken = "test-token"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	run, err := RunBaseline(context.Background(), RunOptions{Mode: "fast", Workspace: "/Users/future/private/baseline"})
	if err != nil {
		t.Fatal(err)
	}
	if run.CloudSynced {
		t.Fatalf("failed sync should not report cloud synced")
	}

	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	counts, err := syncOutboxCounts(db)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Pending != 0 || counts.Failed != 1 || counts.Synced != 0 {
		t.Fatalf("expected one retryable failed outbox row, got %+v", counts)
	}
}

func TestFlushSyncOutboxMarksRunSynced(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	var seenAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("authorization")
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := defaultConfig()
	cfg.APIBaseURL = server.URL
	cfg.APIToken = "test-token"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	run := Run{
		ID:              "run_sync",
		Mode:            "fast",
		StartedAt:       time.Now(),
		Status:          "ok",
		HealthScore:     97,
		Workspace:       "/Users/future/private/baseline",
		AgentKind:       "test",
		RedactionStatus: "clean",
		Checks:          []CheckResult{{ID: "001", CheckID: "repo.state", Lane: "core", Kind: "awareness", Status: "ok", Score: 100}},
	}
	if err := saveRun(db, run, nil); err != nil {
		t.Fatal(err)
	}
	if err := stageSyncPayload(db, run); err != nil {
		t.Fatal(err)
	}

	result, err := flushSyncOutbox(context.Background(), db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Synced != 1 || result.Failed != 0 {
		t.Fatalf("unexpected flush result: %+v", result)
	}
	if seenAuth != "Bearer test-token" {
		t.Fatalf("expected bearer token, got %q", seenAuth)
	}
	counts, err := syncOutboxCounts(db)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Pending != 0 || counts.Synced != 1 {
		t.Fatalf("expected synced outbox row, got %+v", counts)
	}
	updated, err := runByID(db, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.CloudSynced {
		t.Fatalf("expected run to be marked cloud synced")
	}
}

func TestCloudPayloadDoesNotExposeRawWorkspacePath(t *testing.T) {
	run := Run{
		ID:              "run_safe",
		Mode:            "fast",
		StartedAt:       time.Now(),
		Status:          "ok",
		HealthScore:     100,
		Workspace:       "/Users/future/private/client-repo",
		AgentKind:       "openclaw",
		RedactionStatus: "clean",
	}
	b, err := json.Marshal(cloudPayload(run))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if strings.Contains(body, "/Users/future") || strings.Contains(body, "client-repo") {
		t.Fatalf("cloud payload exposed raw workspace path: %s", body)
	}
	if !strings.Contains(body, "sha256:") {
		t.Fatalf("cloud payload should expose a display hash, got %s", body)
	}
}
