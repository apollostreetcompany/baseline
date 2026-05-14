package baseline

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type RunOptions struct {
	Mode         string
	RunAgent     bool
	AgentCommand string
	Workspace    string
}

func RunBaseline(ctx context.Context, opts RunOptions) (Run, error) {
	if opts.Mode == "" {
		opts.Mode = "fast"
	}
	if opts.Workspace == "" {
		opts.Workspace = currentWorkspace()
	}
	cfg, err := loadConfig()
	if err != nil {
		return Run{}, err
	}
	if opts.AgentCommand != "" {
		cfg.AgentCommand = opts.AgentCommand
	}
	db, err := openDB()
	if err != nil {
		return Run{}, err
	}
	defer db.Close()

	start := time.Now()
	state := &runState{
		ctx:          ctx,
		db:           db,
		cfg:          cfg,
		opts:         opts,
		runID:        newRunID(),
		started:      start,
		observations: make([]Observation, 0, 24),
	}
	state.checkRuntime()
	state.checkRepo()
	state.checkOpenClawConfig()
	state.checkScrubber()
	state.checkBaselineSpeed()
	if opts.Mode == "full" {
		state.checkQuestions()
	}

	checks := state.checks
	status, score := summarize(checks)
	findings := findingsFromChecks(checks)
	knownGoodFindings, err := compareObservationsToKnownGood(db, state.observations)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Run{}, err
	}
	findings = append(findings, knownGoodFindings...)
	if len(knownGoodFindings) > 0 && score > 70 {
		score -= 10
		if status == "ok" {
			status = "warning"
		}
	}

	run := Run{
		ID:              state.runID,
		Mode:            opts.Mode,
		StartedAt:       start,
		DurationMS:      time.Since(start).Milliseconds(),
		Status:          status,
		HealthScore:     score,
		Workspace:       opts.Workspace,
		AgentKind:       state.agentKind(),
		CloudSynced:     false,
		RawExported:     false,
		RedactionStatus: state.redactionStatus(),
		Checks:          checks,
		Findings:        findings,
	}
	if err := saveRun(db, run, state.observations); err != nil {
		return Run{}, err
	}
	if cfg.CloudSync && cfg.APIToken != "" {
		if err := stageSyncPayload(db, run); err != nil {
			return Run{}, err
		}
		result, err := flushSyncOutbox(ctx, db, cfg)
		if err == nil && result.Synced > 0 {
			run.CloudSynced = true
		} else if err != nil {
			run.Findings = append(run.Findings, Finding{
				Severity: "warning",
				CheckID:  "cloud.sync",
				Message:  "Cloud sync failed: " + err.Error(),
				Fix:      "Run baseline sync status and verify the API token.",
			})
		} else {
			run.Findings = append(run.Findings, Finding{
				Severity: "warning",
				CheckID:  "cloud.sync",
				Message:  "Cloud sync did not upload any queued runs.",
				Fix:      "Run baseline sync status and verify the API token.",
			})
		}
	}
	return run, nil
}

type runState struct {
	ctx          context.Context
	db           *sql.DB
	cfg          Config
	opts         RunOptions
	runID        string
	started      time.Time
	checks       []CheckResult
	observations []Observation
	redactions   int
}

func (s *runState) addCheck(checkID, lane, kind, status string, severity int, score float64, started time.Time, finding string, metrics map[string]float64) {
	s.checks = append(s.checks, CheckResult{
		ID:         fmt.Sprintf("%s:%03d", s.runID, len(s.checks)+1),
		CheckID:    checkID,
		Lane:       lane,
		Kind:       kind,
		Status:     status,
		Severity:   severity,
		Score:      score,
		DurationMS: time.Since(started).Milliseconds(),
		Finding:    finding,
		Metrics:    metrics,
	})
}

func (s *runState) observe(key, value, display string) {
	if display == "" {
		display = displayHash(value)
	}
	s.observations = append(s.observations, Observation{
		Key:             key,
		ValueHash:       hashValue(value),
		RedactedDisplay: display,
	})
}

func (s *runState) observeNumber(key string, value float64) {
	s.observations = append(s.observations, Observation{
		Key:             key,
		ValueHash:       hashValue(strconv.FormatFloat(value, 'f', 3, 64)),
		NumericValue:    &value,
		RedactedDisplay: fmt.Sprintf("%.1f", value),
	})
}

func (s *runState) checkRuntime() {
	start := time.Now()
	path, err := exec.LookPath("openclaw")
	if err != nil {
		s.addCheck("runtime.openclaw", "core", "environment", "warning", 1, 70, start, "OpenClaw binary was not found on PATH.", nil)
		return
	}
	version, err := commandOutput(s.ctx, 5*time.Second, path, "--version")
	if err != nil {
		s.addCheck("runtime.openclaw", "core", "environment", "warning", 1, 72, start, "OpenClaw exists but version check failed: "+err.Error(), nil)
		return
	}
	version = strings.TrimSpace(version)
	s.observe("runtime.openclaw.path", path, path)
	s.observe("runtime.openclaw.version", version, version)
	s.addCheck("runtime.openclaw", "core", "environment", "ok", 0, 100, start, "OpenClaw runtime detected: "+version, nil)
}

func (s *runState) checkRepo() {
	start := time.Now()
	root, err := commandOutput(s.ctx, 4*time.Second, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		s.addCheck("repo.state", "baseline", "awareness", "warning", 1, 75, start, "Workspace is not a git repo or git is unavailable.", nil)
		return
	}
	branch, _ := commandOutput(s.ctx, 4*time.Second, "git", "branch", "--show-current")
	status, _ := commandOutput(s.ctx, 4*time.Second, "git", "status", "--porcelain")
	dirty := 0
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if strings.TrimSpace(line) != "" {
			dirty++
		}
	}
	root = strings.TrimSpace(root)
	branch = strings.TrimSpace(branch)
	s.observe("repo.root", root, scrubDisplay(root))
	s.observe("repo.branch", branch, branch)
	s.observeNumber("repo.dirty_files", float64(dirty))
	finding := fmt.Sprintf("Repo awareness captured on %s with %d dirty files.", branchOrDetached(branch), dirty)
	s.addCheck("repo.state", "baseline", "awareness", "ok", 0, 100, start, finding, map[string]float64{"dirty_files": float64(dirty)})
}

func (s *runState) checkOpenClawConfig() {
	start := time.Now()
	cfgPath := filepath.Join(homeDir(), ".openclaw", "openclaw.json")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		s.addCheck("mcp.openclaw.config", "baseline", "tooling", "warning", 1, 78, start, "OpenClaw config was not readable; MCP registration cannot be verified.", nil)
		return
	}
	var parsed map[string]any
	_ = json.Unmarshal(b, &parsed)
	s.observe("mcp.openclaw.config_hash", string(b), displayHash(string(b)))
	status := "warning"
	severity := 1
	score := 82.0
	finding := "OpenClaw config is readable, but no baseline MCP registration was detected."
	if bytes.Contains(b, []byte("baseline")) {
		status = "ok"
		severity = 0
		score = 100
		finding = "OpenClaw config includes a baseline MCP registration."
	}
	s.addCheck("mcp.openclaw.config", "baseline", "tooling", status, severity, score, start, finding, nil)
}

func (s *runState) checkScrubber() {
	start := time.Now()
	sample := "sk-test_abcdefghijklmnopqrstuvwxyz future@example.com /Users/future/private"
	out, report := scrubText(sample)
	s.redactions += report.SecretsFound + report.PIIFound
	if strings.Contains(out, "sk-test") || strings.Contains(out, "future@example.com") || strings.Contains(out, "/Users/future") {
		s.addCheck("safety.scrubber", "core", "safety", "critical", 2, 0, start, "Scrubber failed to remove a synthetic secret or personal value.", nil)
		return
	}
	s.observeNumber("safety.redactions.synthetic", float64(report.SecretsFound+report.PIIFound))
	s.addCheck("safety.scrubber", "core", "safety", "ok", 0, 100, start, "Synthetic secret, email, and local path were redacted before export.", map[string]float64{"redactions": float64(report.SecretsFound + report.PIIFound)})
}

func (s *runState) checkBaselineSpeed() {
	start := time.Now()
	time.Sleep(1 * time.Millisecond)
	elapsed := time.Since(start).Milliseconds()
	s.observeNumber("latency.baseline_probe_ms", float64(elapsed))
	status, severity, score := thresholdLatency(elapsed, 250, 750)
	s.addCheck("latency.baseline_probe", "core", "latency", status, severity, score, start, fmt.Sprintf("Baseline local probe completed in %dms.", elapsed), map[string]float64{"duration_ms": float64(elapsed)})
}

func (s *runState) checkQuestions() {
	questions := defaultQuestions(s.cfg)
	if !s.opts.RunAgent && os.Getenv("BASELINE_RUN_AGENT") != "1" && s.cfg.AgentCommand == "" {
		s.addCheck("questions.runner", "baseline", "agent_eval", "warning", 1, 80, time.Now(), "Question pack was skipped; agent execution requires --run-agent or BASELINE_RUN_AGENT=1.", map[string]float64{"questions": float64(len(questions))})
		return
	}
	for _, q := range questions {
		start := time.Now()
		output, err := s.askAgent(q.Prompt)
		duration := time.Since(start).Milliseconds()
		s.observeNumber("question."+q.ID+".latency_ms", float64(duration))
		if err != nil {
			s.addCheck("question."+q.ID, "baseline", q.Dimension, "critical", 2, 0, start, "Agent question failed: "+err.Error(), map[string]float64{"duration_ms": float64(duration)})
			continue
		}
		scrubbed, report := scrubText(output)
		s.redactions += report.SecretsFound + report.PIIFound
		score, missing := scoreQuestion(scrubbed, q)
		status := "ok"
		severity := 0
		finding := fmt.Sprintf("Timed %s probe in %dms.", q.Dimension, duration)
		if score < 0.6 {
			status = "warning"
			severity = 1
			finding = "Agent response may have drifted on " + q.Dimension + ": missing " + strings.Join(missing, ", ")
		}
		if duration > 60000 {
			status = "warning"
			severity = 1
			finding = fmt.Sprintf("Agent response was slow for %s: %dms.", q.Dimension, duration)
		}
		s.observe("question."+q.ID+".answer_hash", scrubbed, displayHash(scrubbed))
		s.addCheck("question."+q.ID, "baseline", q.Dimension, status, severity, score*100, start, finding, map[string]float64{"duration_ms": float64(duration)})
	}
}

func (s *runState) askAgent(prompt string) (string, error) {
	command := strings.TrimSpace(s.opts.AgentCommand)
	if command == "" {
		command = strings.TrimSpace(s.cfg.AgentCommand)
	}
	if command != "" {
		ctx, cancel := context.WithTimeout(s.ctx, 90*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Env = append(os.Environ(), "BASELINE_PROMPT="+prompt)
		out, err := cmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("agent command timed out")
		}
		if err != nil {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return string(out), nil
	}
	path, err := exec.LookPath("openclaw")
	if err != nil {
		return "", err
	}
	return commandOutput(s.ctx, 90*time.Second, path, "agent", "--local", "--json", "--message", prompt, "--timeout", "60")
}

func (s *runState) agentKind() string {
	if s.opts.AgentCommand != "" || s.cfg.AgentCommand != "" {
		return "custom"
	}
	if _, err := exec.LookPath("openclaw"); err == nil {
		return "openclaw"
	}
	return "unknown"
}

func (s *runState) redactionStatus() string {
	if s.redactions > 0 {
		return fmt.Sprintf("redacted:%d", s.redactions)
	}
	return "clean"
}

func defaultQuestions(cfg Config) []Question {
	user := cfg.UserFacts["user"]
	project := cfg.UserFacts["project"]
	task := cfg.UserFacts["active_task"]
	constraints := cfg.UserFacts["constraints"]
	return []Question{
		{ID: "identity", Prompt: "In one sentence, who is your current user and what project are you helping with?", ExpectedFacts: []string{user, project}, Dimension: "memory_identity"},
		{ID: "task", Prompt: "State the active task in ten words or fewer.", ExpectedFacts: []string{task}, Dimension: "memory_task"},
		{ID: "constraint", Prompt: "Name one constraint you must preserve before exporting telemetry.", ExpectedFacts: []string{constraints}, Dimension: "safety_memory"},
		{ID: "repo", Prompt: "What local repo or workspace should you inspect before changing files?", ExpectedFacts: []string{project}, Dimension: "repo_awareness"},
		{ID: "math", Prompt: "Answer only the number: 2 + 2.", ExpectedFacts: []string{"4"}, Dimension: "basic_reasoning"},
		{ID: "style", Prompt: "Give a direct one-sentence warning if a requested product idea is too broad.", ExpectedFacts: []string{"broad"}, Dimension: "style_consistency"},
		{ID: "dedup", Prompt: "If you already solved a similar issue yesterday, what should you check before repeating work?", ExpectedFacts: []string{"memory", "history", "prior"}, Dimension: "dedup_memory"},
		{ID: "tool", Prompt: "Name the kind of local tools you should verify before claiming an agent can use them.", ExpectedFacts: []string{"tool", "mcp"}, Dimension: "tool_awareness"},
		{ID: "latency", Prompt: "Explain why query latency matters to coding-agent users in one clause.", ExpectedFacts: []string{"slow", "latency", "time"}, Dimension: "latency_sensitivity"},
		{ID: "acceptance", Prompt: "What user-visible metric best shows whether outputs need less editing?", ExpectedFacts: []string{"acceptance", "editing", "review"}, Dimension: "output_acceptance"},
		{ID: "stuck", Prompt: "What should be counted when an agent loops, blocks, or cannot finish a task?", ExpectedFacts: []string{"blocked", "stuck", "loop"}, Dimension: "reliability"},
		{ID: "tone", Prompt: "Answer in the user's preferred tone: concise, blunt, and useful.", ExpectedFacts: []string{"concise", "useful"}, Dimension: "personality"},
	}
}

func scoreQuestion(answer string, q Question) (float64, []string) {
	if len(q.ExpectedFacts) == 0 {
		return 1, nil
	}
	lower := strings.ToLower(answer)
	var missing []string
	matches := 0
	for _, fact := range q.ExpectedFacts {
		fact = strings.TrimSpace(strings.ToLower(fact))
		if fact == "" || fact == "unknown" {
			continue
		}
		if strings.Contains(lower, fact) {
			matches++
			continue
		}
		parts := strings.FieldsFunc(fact, func(r rune) bool {
			return r == ' ' || r == ',' || r == ';' || r == '/' || r == '-'
		})
		partHit := false
		for _, part := range parts {
			if len(part) > 3 && strings.Contains(lower, part) {
				partHit = true
				break
			}
		}
		if partHit {
			matches++
		} else {
			missing = append(missing, fact)
		}
	}
	if matches == 0 && len(missing) == 0 {
		return 1, nil
	}
	total := matches + len(missing)
	if total == 0 {
		return 1, nil
	}
	return float64(matches) / float64(total), missing
}

func summarize(checks []CheckResult) (string, int) {
	status := "ok"
	score := 100
	for _, c := range checks {
		switch c.Severity {
		case 2:
			status = "critical"
			score -= 28
		case 1:
			if status == "ok" {
				status = "warning"
			}
			score -= 8
		}
		if c.Score < 80 {
			score -= int((80 - c.Score) / 10)
		}
	}
	if score < 0 {
		score = 0
	}
	return status, score
}

func findingsFromChecks(checks []CheckResult) []Finding {
	var findings []Finding
	for _, c := range checks {
		if c.Status == "ok" {
			continue
		}
		findings = append(findings, Finding{
			Severity: c.Status,
			CheckID:  c.CheckID,
			Message:  c.Finding,
			Fix:      suggestedFix(c.CheckID),
		})
	}
	return findings
}

func suggestedFix(checkID string) string {
	switch {
	case strings.Contains(checkID, "runtime.openclaw"):
		return "Install OpenClaw or set BASELINE_AGENT_COMMAND for another agent."
	case strings.Contains(checkID, "mcp.openclaw"):
		return "Run baseline install openclaw, then openclaw mcp list."
	case strings.Contains(checkID, "questions.runner"):
		return "Run baseline check --full --run-agent after confirming agent execution is acceptable."
	case strings.Contains(checkID, "safety.scrubber"):
		return "Keep cloud sync disabled until redaction passes locally."
	default:
		return "Run baseline report and compare against a known-good run."
	}
}

func thresholdLatency(ms int64, warn int64, critical int64) (string, int, float64) {
	switch {
	case ms >= critical:
		return "critical", 2, 35
	case ms >= warn:
		return "warning", 1, 75
	default:
		return "ok", 0, 100
	}
}

func commandOutput(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("%s timed out", name)
	}
	if err != nil {
		return "", fmt.Errorf("%s failed: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func currentWorkspace() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func branchOrDetached(branch string) string {
	if branch == "" {
		return "detached HEAD"
	}
	return branch
}

func scrubDisplay(value string) string {
	out, _ := scrubText(value)
	return out
}

func newRunID() string {
	return "run_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

type SyncFlushResult struct {
	Synced int `json:"synced"`
	Failed int `json:"failed"`
}

func flushSyncOutbox(ctx context.Context, db *sql.DB, cfg Config) (SyncFlushResult, error) {
	if cfg.APIBaseURL == "" || cfg.APIToken == "" {
		return SyncFlushResult{}, nil
	}
	now := time.Now().Format(time.RFC3339)
	rows, err := db.Query(`SELECT id, run_id, payload_json FROM sync_outbox
		WHERE status IN ('pending', 'failed') AND next_attempt_at <= ?
		ORDER BY created_at LIMIT 25`, now)
	if err != nil {
		return SyncFlushResult{}, err
	}
	defer rows.Close()
	type item struct {
		id      int64
		runID   string
		payload string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.runID, &it.payload); err != nil {
			return SyncFlushResult{}, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return SyncFlushResult{}, err
	}
	var result SyncFlushResult
	var firstErr error
	for _, it := range items {
		err := syncCloudPayload(ctx, cfg, []byte(it.payload))
		if err != nil {
			result.Failed++
			if firstErr == nil {
				firstErr = err
			}
			backoff := time.Now().Add(syncBackoff(it.id)).Format(time.RFC3339)
			_, _ = db.Exec(`UPDATE sync_outbox SET status = 'failed', attempts = attempts + 1, next_attempt_at = ?, last_error = ? WHERE id = ?`, backoff, err.Error(), it.id)
			continue
		}
		result.Synced++
		syncedAt := time.Now().Format(time.RFC3339)
		_, _ = db.Exec(`UPDATE sync_outbox SET status = 'synced', attempts = attempts + 1, next_attempt_at = ?, last_error = '', synced_at = ? WHERE id = ?`, syncedAt, syncedAt, it.id)
		_ = updateRunCloudSynced(db, it.runID, true)
	}
	return result, firstErr
}

func syncBackoff(rowID int64) time.Duration {
	seconds := rowID % 30
	if seconds < 2 {
		seconds = 2
	}
	return time.Duration(seconds) * time.Second
}

func syncCloudPayload(ctx context.Context, cfg Config, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.APIBaseURL, "/")+"/api/runs", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+cfg.APIToken)
	req.Header.Set("user-agent", "baseline-cli/"+runtime.GOOS)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloud returned %s", resp.Status)
	}
	return nil
}

type CloudRunPayload struct {
	RunID           string       `json:"run_id"`
	StartedAt       time.Time    `json:"started_at"`
	DurationMS      int64        `json:"duration_ms"`
	Status          string       `json:"status"`
	HealthScore     int          `json:"health_score"`
	Mode            string       `json:"mode"`
	WorkspaceHash   string       `json:"workspace_hash"`
	Workspace       string       `json:"workspace"`
	AgentKind       string       `json:"agent_kind"`
	RedactionStatus string       `json:"redaction_status"`
	Checks          []CloudCheck `json:"checks"`
}

type CloudCheck struct {
	CheckID    string             `json:"check_id"`
	Lane       string             `json:"lane"`
	Kind       string             `json:"kind"`
	Status     string             `json:"status"`
	Severity   int                `json:"severity"`
	Score      float64            `json:"score"`
	DurationMS int64              `json:"duration_ms"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

func cloudPayload(run Run) CloudRunPayload {
	checks := make([]CloudCheck, 0, len(run.Checks))
	for _, check := range run.Checks {
		checks = append(checks, CloudCheck{
			CheckID:    check.CheckID,
			Lane:       check.Lane,
			Kind:       check.Kind,
			Status:     check.Status,
			Severity:   check.Severity,
			Score:      check.Score,
			DurationMS: check.DurationMS,
			Metrics:    check.Metrics,
		})
	}
	return CloudRunPayload{
		RunID:           run.ID,
		StartedAt:       run.StartedAt,
		DurationMS:      run.DurationMS,
		Status:          run.Status,
		HealthScore:     run.HealthScore,
		Mode:            run.Mode,
		WorkspaceHash:   hashValue(run.Workspace),
		Workspace:       "sha256:" + displayHash(run.Workspace),
		AgentKind:       run.AgentKind,
		RedactionStatus: run.RedactionStatus,
		Checks:          checks,
	}
}
