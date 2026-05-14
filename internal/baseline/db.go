package baseline

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
			scope_key TEXT NOT NULL DEFAULT '',
			config_hash TEXT NOT NULL DEFAULT '',
			question_set_version TEXT NOT NULL DEFAULT '',
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
		`CREATE TABLE IF NOT EXISTS workspace_profiles (
			id TEXT PRIMARY KEY,
			scope_key TEXT NOT NULL UNIQUE,
			workspace_hash TEXT NOT NULL,
			config_hash TEXT NOT NULL,
			label TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS bootstrap_candidates (
			run_id TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
			scope_key TEXT NOT NULL,
			config_hash TEXT NOT NULL,
			status TEXT NOT NULL,
			label TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS accepted_baseline_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			scope_key TEXT NOT NULL,
			config_hash TEXT NOT NULL,
			slot INTEGER NOT NULL,
			label TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(scope_key, config_hash, slot),
			UNIQUE(scope_key, config_hash, run_id)
		);`,
		`CREATE TABLE IF NOT EXISTS consent_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			run_id TEXT NOT NULL DEFAULT '',
			scope_key TEXT NOT NULL DEFAULT '',
			config_hash TEXT NOT NULL DEFAULT '',
			details_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS probe_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			pack_id TEXT NOT NULL,
			pack_version TEXT NOT NULL DEFAULT '',
			probe_id TEXT NOT NULL,
			question_set_version TEXT NOT NULL DEFAULT '',
			prompt_hash TEXT NOT NULL DEFAULT '',
			expected_facts_hash TEXT NOT NULL DEFAULT '',
			session_id TEXT NOT NULL,
			system_send_at TEXT NOT NULL,
			baseline_received_at TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			token_status TEXT NOT NULL,
			token_source TEXT NOT NULL DEFAULT '',
			input_tokens INTEGER,
			output_tokens INTEGER,
			total_tokens INTEGER,
			context_tokens INTEGER,
			model TEXT NOT NULL DEFAULT '',
			model_provider TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS sync_outbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL UNIQUE REFERENCES runs(id) ON DELETE CASCADE,
			payload_json TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			next_attempt_at TEXT NOT NULL,
			last_error TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			synced_at TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_checks_run ON check_results(run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_observations_key ON observations(key, run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sync_outbox_status ON sync_outbox(status, next_attempt_at);`,
		`CREATE INDEX IF NOT EXISTS idx_good_scope ON accepted_baseline_snapshots(scope_key, config_hash, slot);`,
		`CREATE INDEX IF NOT EXISTS idx_probe_messages_run ON probe_messages(run_id);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	for _, col := range []struct {
		table string
		name  string
		ddl   string
	}{
		{"runs", "scope_key", "scope_key TEXT NOT NULL DEFAULT ''"},
		{"runs", "config_hash", "config_hash TEXT NOT NULL DEFAULT ''"},
		{"runs", "question_set_version", "question_set_version TEXT NOT NULL DEFAULT ''"},
		{"probe_messages", "pack_version", "pack_version TEXT NOT NULL DEFAULT ''"},
		{"probe_messages", "question_set_version", "question_set_version TEXT NOT NULL DEFAULT ''"},
		{"probe_messages", "prompt_hash", "prompt_hash TEXT NOT NULL DEFAULT ''"},
		{"probe_messages", "expected_facts_hash", "expected_facts_hash TEXT NOT NULL DEFAULT ''"},
		{"probe_messages", "token_source", "token_source TEXT NOT NULL DEFAULT ''"},
	} {
		if err := addColumnIfMissing(db, col.table, col.name, col.ddl); err != nil {
			return err
		}
	}
	return nil
}

func updateRunCloudSynced(db *sql.DB, runID string, synced bool) error {
	_, err := db.Exec(`UPDATE runs SET cloud_synced = ? WHERE id = ?`, boolInt(synced), runID)
	return err
}

func addColumnIfMissing(db *sql.DB, table, name, ddl string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if colName == name {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + ddl)
	return err
}

func saveRun(db *sql.DB, run Run, observations []Observation) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO runs (id, started_at, duration_ms, status, health_score, mode, workspace, scope_key, config_hash, question_set_version, agent_kind, cloud_synced, raw_exported, redaction_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.StartedAt.Format(time.RFC3339Nano), run.DurationMS, run.Status, run.HealthScore, run.Mode, run.Workspace, run.ScopeKey, run.ConfigHash, run.QuestionSetVersion, run.AgentKind, boolInt(run.CloudSynced), boolInt(run.RawExported), run.RedactionStatus)
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

func saveProbeMessage(db *sql.DB, msg ProbeMessage) error {
	_, err := db.Exec(`INSERT INTO probe_messages
		(run_id, pack_id, pack_version, probe_id, question_set_version, prompt_hash, expected_facts_hash, session_id, system_send_at, baseline_received_at, duration_ms, token_status, token_source, input_tokens, output_tokens, total_tokens, context_tokens, model, model_provider)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.RunID, msg.PackID, msg.PackVersion, msg.ProbeID, msg.QuestionSetVersion, msg.PromptHash, msg.ExpectedFactsHash, msg.SessionID,
		msg.SystemSendAt.Format(time.RFC3339Nano), msg.BaselineReceivedAt.Format(time.RFC3339Nano), msg.DurationMS, msg.TokenStatus,
		msg.TokenSource,
		nullableInt(msg.InputTokens), nullableInt(msg.OutputTokens), nullableInt(msg.TotalTokens), nullableInt(msg.ContextTokens),
		msg.Model, msg.ModelProvider)
	return err
}

func latestRun(db *sql.DB) (Run, error) {
	row := db.QueryRow(`SELECT id, started_at, duration_ms, status, health_score, mode, workspace, scope_key, config_hash, question_set_version, agent_kind, cloud_synced, raw_exported, redaction_status
		FROM runs ORDER BY started_at DESC LIMIT 1`)
	return scanRunWithChecks(db, row)
}

func runByID(db *sql.DB, id string) (Run, error) {
	row := db.QueryRow(`SELECT id, started_at, duration_ms, status, health_score, mode, workspace, scope_key, config_hash, question_set_version, agent_kind, cloud_synced, raw_exported, redaction_status
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
	if err := row.Scan(&r.ID, &started, &r.DurationMS, &r.Status, &r.HealthScore, &r.Mode, &r.Workspace, &r.ScopeKey, &r.ConfigHash, &r.QuestionSetVersion, &r.AgentKind, &cloud, &raw, &r.RedactionStatus); err != nil {
		return r, err
	}
	r.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
	if r.ScopeKey == "" {
		r.ScopeKey = scopeKeyForWorkspace(r.Workspace)
	}
	if r.QuestionSetVersion == "" {
		r.QuestionSetVersion = questionSetVersion
	}
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
		label = "Good baseline"
	}
	if _, err := db.Exec(`INSERT OR REPLACE INTO known_goods (run_id, label, created_at) VALUES (?, ?, ?)`, runID, label, time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	cfg, _ := loadConfig()
	run, err := runByID(db, runID)
	if err != nil {
		return err
	}
	_, err = acceptGoodBaseline(db, runID, label, "legacy known-good alias", 0, scopeKeyForWorkspace(run.Workspace), configHash(cfg))
	return err
}

func latestKnownGoodID(db *sql.DB) (string, error) {
	var id string
	err := db.QueryRow(`SELECT run_id FROM known_goods ORDER BY created_at DESC LIMIT 1`).Scan(&id)
	return id, err
}

func compareToKnownGood(db *sql.DB, latestID string) ([]Finding, error) {
	run, err := runByID(db, latestID)
	if err != nil {
		return nil, err
	}
	scopeKey := run.ScopeKey
	if scopeKey == "" {
		scopeKey = scopeKeyForWorkspace(run.Workspace)
	}
	cfgHash := run.ConfigHash
	if cfgHash == "" {
		cfg, _ := loadConfig()
		cfgHash = configHash(cfg)
	}
	goods, err := listGoodBaselines(db, scopeKey, cfgHash)
	if err != nil {
		return nil, err
	}
	if len(goods) == 0 {
		kg, err := latestKnownGoodID(db)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		goods = []GoodBaseline{{RunID: kg, Slot: 1, Label: "legacy known-good"}}
	}
	observations, err := observationsForRun(db, latestID)
	if err != nil {
		return nil, err
	}
	return compareObservationSetToGoods(db, observations, goods)
}

func compareLegacyKnownGood(db *sql.DB, latestID, kg string) ([]Finding, error) {
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
			CheckID:  "good_baseline.diff",
			Message:  fmt.Sprintf("%s changed: %s -> %s", key, prev, curr),
			Fix:      "Run baseline report for details; accept as Good Baseline only after verifying this state.",
		})
	}
	return findings, rows.Err()
}

func compareObservationsToKnownGood(db *sql.DB, observations []Observation) ([]Finding, error) {
	cfg, _ := loadConfig()
	findings, err := compareObservationsToGood(db, observations, scopeKeyForWorkspace(currentWorkspace()), configHash(cfg))
	if err != nil || len(findings) > 0 {
		return findings, err
	}
	kg, err := latestKnownGoodID(db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return compareObservationSetToGoods(db, observations, []GoodBaseline{{RunID: kg, Slot: 1, Label: "legacy known-good"}})
}

func compareObservationsToGood(db *sql.DB, observations []Observation, scopeKey, cfgHash string) ([]Finding, error) {
	goods, err := listGoodBaselines(db, scopeKey, cfgHash)
	if err != nil {
		return nil, err
	}
	if len(goods) == 0 {
		return nil, nil
	}
	return compareObservationSetToGoods(db, observations, goods)
}

func compareObservationSetToGoods(db *sql.DB, observations []Observation, goods []GoodBaseline) ([]Finding, error) {
	type previous struct {
		hashes   map[string]bool
		displays []string
		numbers  []float64
	}
	prev := map[string]*previous{}
	for _, good := range goods {
		rows, err := db.Query(`SELECT key, value_hash, numeric_value, redacted_display FROM observations WHERE run_id = ?`, good.RunID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var key, hash, display string
			var numeric sql.NullFloat64
			if err := rows.Scan(&key, &hash, &numeric, &display); err != nil {
				rows.Close()
				return nil, err
			}
			p := prev[key]
			if p == nil {
				p = &previous{hashes: map[string]bool{}}
				prev[key] = p
			}
			p.hashes[hash] = true
			p.displays = append(p.displays, display)
			if numeric.Valid {
				p.numbers = append(p.numbers, numeric.Float64)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	var findings []Finding
	for _, curr := range observations {
		p, ok := prev[curr.Key]
		if !ok || p.hashes[curr.ValueHash] {
			continue
		}
		if curr.NumericValue != nil && len(p.numbers) > 0 && withinGoodNumericEnvelope(*curr.NumericValue, p.numbers) {
			continue
		}
		prevDisplay := strings.Join(uniqueStrings(p.displays), " | ")
		findings = append(findings, Finding{
			Severity: "warning",
			CheckID:  "good_baseline.diff",
			Message:  fmt.Sprintf("%s changed outside Good Baseline variation: %s -> %s", curr.Key, prevDisplay, curr.RedactedDisplay),
			Fix:      "Run baseline report for details; accept this run as a Good Baseline only after verifying this state.",
		})
		if len(findings) >= 12 {
			break
		}
	}
	return findings, nil
}

func withinGoodNumericEnvelope(value float64, goods []float64) bool {
	if len(goods) == 0 {
		return false
	}
	min, max := goods[0], goods[0]
	for _, v := range goods[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	width := max - min
	tolerance := maxAbs(max, min) * 0.20
	if tolerance < 1 {
		tolerance = 1
	}
	if width > tolerance {
		tolerance = width
	}
	return value >= min-tolerance && value <= max+tolerance
}

func maxAbs(a, b float64) float64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	if a > b {
		return a
	}
	return b
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if len(out) == 3 {
			break
		}
	}
	return out
}

type KnownGood struct {
	RunID     string `json:"run_id"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}

func createBootstrapCandidate(db *sql.DB, runID, label, notes, scopeKey, cfgHash string) (BootstrapCandidate, error) {
	if label == "" {
		label = "Baseline candidate"
	}
	now := time.Now().Format(time.RFC3339)
	_, err := db.Exec(`INSERT INTO bootstrap_candidates (run_id, scope_key, config_hash, status, label, notes, created_at)
		VALUES (?, ?, ?, 'candidate', ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET status = 'candidate', label = excluded.label, notes = excluded.notes`,
		runID, scopeKey, cfgHash, label, notes, now)
	if err != nil {
		return BootstrapCandidate{}, err
	}
	return BootstrapCandidate{RunID: runID, ScopeKey: scopeKey, ConfigHash: cfgHash, Status: "candidate", Label: label, Notes: notes, CreatedAt: now}, nil
}

func rejectBootstrapCandidate(db *sql.DB, runID, notes string) error {
	_, err := db.Exec(`UPDATE bootstrap_candidates SET status = 'rejected', notes = ? WHERE run_id = ?`, notes, runID)
	return err
}

func latestBootstrapCandidate(db *sql.DB, scopeKey, cfgHash string) (BootstrapCandidate, error) {
	row := db.QueryRow(`SELECT run_id, scope_key, config_hash, status, label, notes, created_at
		FROM bootstrap_candidates WHERE scope_key = ? AND config_hash = ?
		ORDER BY created_at DESC LIMIT 1`, scopeKey, cfgHash)
	var c BootstrapCandidate
	err := row.Scan(&c.RunID, &c.ScopeKey, &c.ConfigHash, &c.Status, &c.Label, &c.Notes, &c.CreatedAt)
	return c, err
}

func acceptGoodBaseline(db *sql.DB, runID, label, notes string, slot int, scopeKey, cfgHash string) (GoodBaseline, error) {
	if label == "" {
		label = "Good baseline"
	}
	if slot == 0 {
		used := map[int]bool{}
		rows, err := db.Query(`SELECT slot FROM accepted_baseline_snapshots WHERE scope_key = ? AND config_hash = ? ORDER BY slot`, scopeKey, cfgHash)
		if err != nil {
			return GoodBaseline{}, err
		}
		for rows.Next() {
			var usedSlot int
			if err := rows.Scan(&usedSlot); err != nil {
				rows.Close()
				return GoodBaseline{}, err
			}
			used[usedSlot] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return GoodBaseline{}, err
		}
		rows.Close()
		for i := 1; i <= 3; i++ {
			if !used[i] {
				slot = i
				break
			}
		}
		if slot == 0 {
			return GoodBaseline{}, fmt.Errorf("all 3 Good Baseline slots are full; replace a slot explicitly")
		}
	}
	if slot < 1 || slot > 3 {
		return GoodBaseline{}, fmt.Errorf("Good Baseline slot must be 1, 2, or 3")
	}
	now := time.Now().Format(time.RFC3339)
	_, err := db.Exec(`INSERT INTO accepted_baseline_snapshots (run_id, scope_key, config_hash, slot, label, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(scope_key, config_hash, slot) DO UPDATE SET
			run_id = excluded.run_id,
			label = excluded.label,
			notes = excluded.notes,
			created_at = excluded.created_at`,
		runID, scopeKey, cfgHash, slot, label, notes, now)
	if err != nil {
		return GoodBaseline{}, err
	}
	_, _ = db.Exec(`UPDATE bootstrap_candidates SET status = 'accepted' WHERE run_id = ?`, runID)
	_, _ = db.Exec(`INSERT INTO consent_events (action, run_id, scope_key, config_hash, details_json, created_at)
		VALUES ('good.accept', ?, ?, ?, ?, ?)`, runID, scopeKey, cfgHash, fmt.Sprintf(`{"slot":%d}`, slot), now)
	return GoodBaseline{RunID: runID, ScopeKey: scopeKey, ConfigHash: cfgHash, Slot: slot, Label: label, Notes: notes, CreatedAt: now}, nil
}

func listGoodBaselines(db *sql.DB, scopeKey, cfgHash string) ([]GoodBaseline, error) {
	rows, err := db.Query(`SELECT run_id, scope_key, config_hash, slot, label, notes, created_at
		FROM accepted_baseline_snapshots WHERE scope_key = ? AND config_hash = ? ORDER BY slot`, scopeKey, cfgHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goods []GoodBaseline
	for rows.Next() {
		var g GoodBaseline
		if err := rows.Scan(&g.RunID, &g.ScopeKey, &g.ConfigHash, &g.Slot, &g.Label, &g.Notes, &g.CreatedAt); err != nil {
			return nil, err
		}
		goods = append(goods, g)
	}
	return goods, rows.Err()
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

type SyncOutboxCounts struct {
	Pending int `json:"pending"`
	Synced  int `json:"synced"`
	Failed  int `json:"failed"`
}

func syncOutboxCounts(db *sql.DB) (SyncOutboxCounts, error) {
	rows, err := db.Query(`SELECT status, count(*) FROM sync_outbox GROUP BY status`)
	if err != nil {
		return SyncOutboxCounts{}, err
	}
	defer rows.Close()
	var counts SyncOutboxCounts
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return counts, err
		}
		switch status {
		case "pending":
			counts.Pending = count
		case "synced":
			counts.Synced = count
		case "failed":
			counts.Failed = count
		}
	}
	return counts, rows.Err()
}

func stageSyncPayload(db *sql.DB, run Run) error {
	payload, err := json.Marshal(cloudPayload(run))
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`INSERT INTO sync_outbox (run_id, payload_json, status, attempts, next_attempt_at, last_error, created_at)
		VALUES (?, ?, 'pending', 0, ?, '', ?)
		ON CONFLICT(run_id) DO UPDATE SET
			payload_json = excluded.payload_json,
			status = CASE WHEN sync_outbox.status = 'synced' THEN sync_outbox.status ELSE 'pending' END,
			next_attempt_at = excluded.next_attempt_at,
			last_error = CASE WHEN sync_outbox.status = 'synced' THEN sync_outbox.last_error ELSE '' END`,
		run.ID, string(payload), now, now)
	return err
}

func stageUnsyncedRuns(db *sql.DB, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`SELECT id FROM runs WHERE cloud_synced = 0 ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	staged := 0
	for _, id := range ids {
		run, err := runByID(db, id)
		if err != nil {
			return staged, err
		}
		if err := stageSyncPayload(db, run); err != nil {
			return staged, err
		}
		staged++
	}
	return staged, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
