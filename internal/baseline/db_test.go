package baseline

import (
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
