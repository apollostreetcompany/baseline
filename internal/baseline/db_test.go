package baseline

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMigrationAddsBead14ColumnsToExistingDB(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	old, err := sql.Open("sqlite", dbPath())
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`CREATE TABLE runs (
		id TEXT PRIMARY KEY,
		started_at TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		status TEXT NOT NULL,
		health_score INTEGER NOT NULL,
		mode TEXT NOT NULL,
		workspace TEXT NOT NULL,
		agent_kind TEXT NOT NULL,
		cloud_synced INTEGER NOT NULL DEFAULT 0,
		raw_exported INTEGER NOT NULL DEFAULT 0,
		redaction_status TEXT NOT NULL
	);`)
	if err != nil {
		t.Fatal(err)
	}
	old.Close()

	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, col := range []string{"scope_key", "config_hash", "question_set_version"} {
		if !testHasColumn(t, db, "runs", col) {
			t.Fatalf("migration did not add runs.%s", col)
		}
	}
	for _, col := range []string{"pack_version", "prompt_hash", "token_source"} {
		if !testHasColumn(t, db, "probe_messages", col) {
			t.Fatalf("migration did not add probe_messages.%s", col)
		}
	}
}

func testHasColumn(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return false
}

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
	if findings[0].CheckID != "good_baseline.diff" {
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

func TestLatestRunPrefersRealEvalOverNewerDoctorStyleRows(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	eval := Run{
		ID:              "run_eval",
		Mode:            "run",
		StartedAt:       time.Now().Add(-1 * time.Minute),
		Status:          "warning",
		HealthScore:     92,
		Workspace:       "test",
		AgentKind:       "openclaw",
		RedactionStatus: "clean",
		Checks:          []CheckResult{{ID: "eval_check", CheckID: "question.baseline.math", Lane: "baseline", Kind: "basic_reasoning", Status: "ok", Score: 100}},
	}
	if err := saveRun(db, eval, nil); err != nil {
		t.Fatal(err)
	}
	fast := Run{
		ID:              "run_fast",
		Mode:            "fast",
		StartedAt:       time.Now(),
		Status:          "ok",
		HealthScore:     100,
		Workspace:       "test",
		AgentKind:       "openclaw",
		RedactionStatus: "clean",
		Checks:          []CheckResult{{ID: "fast_check", CheckID: "runtime.openclaw", Lane: "core", Kind: "environment", Status: "ok", Score: 100}},
	}
	if err := saveRun(db, fast, nil); err != nil {
		t.Fatal(err)
	}
	latest, err := latestRun(db)
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != eval.ID {
		t.Fatalf("expected latest meaningful eval %s, got %s", eval.ID, latest.ID)
	}
}
