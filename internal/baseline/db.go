package baseline

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

func openDB() (*sql.DB, error) {
	if err := ensureDirs(); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath())
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA synchronous=NORMAL; PRAGMA foreign_keys=ON;`); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS runs (
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
		);`,
		`CREATE TABLE IF NOT EXISTS check_results (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			check_id TEXT NOT NULL,
			lane TEXT NOT NULL,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			severity INTEGER NOT NULL,
			score REAL NOT NULL,
			duration_ms INTEGER NOT NULL,
			finding TEXT NOT NULL,
			metrics_json TEXT NOT NULL DEFAULT '{}'
		);`,
		`CREATE TABLE IF NOT EXISTS observations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			key TEXT NOT NULL,
			value_hash TEXT NOT NULL,
			numeric_value REAL,
			redacted_display TEXT NOT NULL,
			previous_value_hash TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS known_goods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL UNIQUE REFERENCES runs(id) ON DELETE CASCADE,
			label TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_checks_run ON check_results(run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_observations_key ON observations(key, run_id);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func saveRun(db *sql.DB, run Run, observations []Observation) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO runs (id, started_at, duration_ms, status, health_score, mode, workspace, agent_kind, cloud_synced, raw_exported, redaction_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.StartedAt.Format(time.RFC3339), run.DurationMS, run.Status, run.HealthScore, run.Mode, run.Workspace, run.AgentKind, boolInt(run.CloudSynced), boolInt(run.RawExported), run.RedactionStatus)
	if err != nil {
		return err
	}
	for _, c := range run.Checks {
		metrics, _ := json.Marshal(c.Metrics)
		_, err = tx.Exec(`INSERT INTO check_results (id, run_id, check_id, lane, kind, status, severity, score, duration_ms, finding, metrics_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.ID, run.ID, c.CheckID, c.Lane, c.Kind, c.Status, c.Severity, c.Score, c.DurationMS, c.Finding, string(metrics))
		if err != nil {
			return err
		}
	}
	for _, o := range observations {
		_, err = tx.Exec(`INSERT INTO observations (run_id, key, value_hash, numeric_value, redacted_display, previous_value_hash)
			VALUES (?, ?, ?, ?, ?, ?)`, run.ID, o.Key, o.ValueHash, o.NumericValue, o.RedactedDisplay, o.PreviousValueHash)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func latestRun(db *sql.DB) (Run, error) {
	row := db.QueryRow(`SELECT id, started_at, duration_ms, status, health_score, mode, workspace, agent_kind, cloud_synced, raw_exported, redaction_status
		FROM runs ORDER BY started_at DESC LIMIT 1`)
	return scanRunWithChecks(db, row)
}

func runByID(db *sql.DB, id string) (Run, error) {
	row := db.QueryRow(`SELECT id, started_at, duration_ms, status, health_score, mode, workspace, agent_kind, cloud_synced, raw_exported, redaction_status
		FROM runs WHERE id = ?`, id)
	return scanRunWithChecks(db, row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRunWithChecks(db *sql.DB, row rowScanner) (Run, error) {
	var r Run
	var started string
	var cloud, raw int
	if err := row.Scan(&r.ID, &started, &r.DurationMS, &r.Status, &r.HealthScore, &r.Mode, &r.Workspace, &r.AgentKind, &cloud, &raw, &r.RedactionStatus); err != nil {
		return r, err
	}
	r.StartedAt, _ = time.Parse(time.RFC3339, started)
	r.CloudSynced = cloud == 1
	r.RawExported = raw == 1
	rows, err := db.Query(`SELECT id, check_id, lane, kind, status, severity, score, duration_ms, finding, metrics_json FROM check_results WHERE run_id = ? ORDER BY id`, r.ID)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var c CheckResult
		var metrics string
		if err := rows.Scan(&c.ID, &c.CheckID, &c.Lane, &c.Kind, &c.Status, &c.Severity, &c.Score, &c.DurationMS, &c.Finding, &metrics); err != nil {
			return r, err
		}
		_ = json.Unmarshal([]byte(metrics), &c.Metrics)
		r.Checks = append(r.Checks, c)
		if c.Status != "ok" {
			r.Findings = append(r.Findings, Finding{Severity: c.Status, CheckID: c.CheckID, Message: c.Finding})
		}
	}
	return r, rows.Err()
}

func markKnownGood(db *sql.DB, runID, label string) error {
	if label == "" {
		label = "known-good"
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO known_goods (run_id, label, created_at) VALUES (?, ?, ?)`, runID, label, time.Now().Format(time.RFC3339))
	return err
}

func latestKnownGoodID(db *sql.DB) (string, error) {
	var id string
	err := db.QueryRow(`SELECT run_id FROM known_goods ORDER BY created_at DESC LIMIT 1`).Scan(&id)
	return id, err
}

func compareToKnownGood(db *sql.DB, latestID string) ([]Finding, error) {
	kg, err := latestKnownGoodID(db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rows, err := db.Query(`SELECT l.key, k.redacted_display, l.redacted_display
		FROM observations l
		JOIN observations k ON k.key = l.key
		WHERE l.run_id = ? AND k.run_id = ? AND l.value_hash <> k.value_hash
		ORDER BY l.key LIMIT 12`, latestID, kg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var findings []Finding
	for rows.Next() {
		var key, prev, curr string
		if err := rows.Scan(&key, &prev, &curr); err != nil {
			return nil, err
		}
		findings = append(findings, Finding{
			Severity: "warning",
			CheckID:  "known_good.diff",
			Message:  fmt.Sprintf("%s changed: %s -> %s", key, prev, curr),
			Fix:      "Run baseline report for details; mark known-good only after verifying this state.",
		})
	}
	return findings, rows.Err()
}

func compareObservationsToKnownGood(db *sql.DB, observations []Observation) ([]Finding, error) {
	kg, err := latestKnownGoodID(db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rows, err := db.Query(`SELECT key, value_hash, redacted_display FROM observations WHERE run_id = ?`, kg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type previous struct {
		hash    string
		display string
	}
	prev := map[string]previous{}
	for rows.Next() {
		var key, hash, display string
		if err := rows.Scan(&key, &hash, &display); err != nil {
			return nil, err
		}
		prev[key] = previous{hash: hash, display: display}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var findings []Finding
	for _, curr := range observations {
		p, ok := prev[curr.Key]
		if !ok || p.hash == curr.ValueHash {
			continue
		}
		findings = append(findings, Finding{
			Severity: "warning",
			CheckID:  "known_good.diff",
			Message:  fmt.Sprintf("%s changed: %s -> %s", curr.Key, p.display, curr.RedactedDisplay),
			Fix:      "Run baseline report for details; mark known-good only after verifying this state.",
		})
		if len(findings) >= 12 {
			break
		}
	}
	return findings, nil
}

type KnownGood struct {
	RunID     string `json:"run_id"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}

func listKnownGoods(db *sql.DB) ([]KnownGood, error) {
	rows, err := db.Query(`SELECT run_id, label, created_at FROM known_goods ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goods []KnownGood
	for rows.Next() {
		var kg KnownGood
		if err := rows.Scan(&kg.RunID, &kg.Label, &kg.CreatedAt); err != nil {
			return nil, err
		}
		goods = append(goods, kg)
	}
	return goods, rows.Err()
}

func observationsForRun(db *sql.DB, runID string) ([]Observation, error) {
	rows, err := db.Query(`SELECT key, value_hash, numeric_value, redacted_display, previous_value_hash
		FROM observations WHERE run_id = ? ORDER BY key`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var observations []Observation
	for rows.Next() {
		var o Observation
		var numeric sql.NullFloat64
		if err := rows.Scan(&o.Key, &o.ValueHash, &numeric, &o.RedactedDisplay, &o.PreviousValueHash); err != nil {
			return nil, err
		}
		if numeric.Valid {
			v := numeric.Float64
			o.NumericValue = &v
		}
		observations = append(observations, o)
	}
	return observations, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
