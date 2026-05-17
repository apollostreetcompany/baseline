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
	"sync"
	"time"
)

type RunOptions struct {
	Mode         string
	RunID        string
	RunAgent     bool
	AgentCommand string
	Workspace    string
	Packs        string
	Ephemeral    bool
}

func RunBaseline(ctx context.Context, opts RunOptions) (Run, error) {
	if opts.Mode == "" {
		opts.Mode = "fast"
	}
	cfg, err := loadConfig()
	if err != nil {
		return Run{}, err
	}
	if opts.Workspace == "" {
		opts.Workspace = runtimeWorkspace(cfg)
	}
	opts.Workspace = normalizeWorkspacePath(opts.Workspace)
	if opts.AgentCommand != "" {
		cfg.AgentCommand = opts.AgentCommand
	}
	if opts.Mode == "run" || opts.Mode == "setup" {
		opts.RunAgent = true
	}
	if opts.Packs == "" && (opts.Mode == "run" || opts.Mode == "setup") {
		opts.Packs = cfg.Target.Packs
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
		runID:        runIDForOptions(opts),
		started:      start,
		observations: make([]Observation, 0, 24),
	}
	state.checkRuntime()
	state.checkRepo()
	state.checkOpenClawConfig()
	state.checkTargetConfig()
	state.checkScrubber()
	state.checkBaselineSpeed()
	if opts.Mode == "full" || opts.Mode == "bootstrap" || opts.Mode == "run" || opts.Mode == "setup" {
		if state.hasCriticalPreflight() {
			state.addCheck("questions.runner", "baseline", "agent_eval", "critical", 2, 0, time.Now(), "Agent evaluation was skipped because preflight found a critical target/config issue.", nil)
		} else {
			state.checkQuestions()
		}
	}

	checks := state.checks
	status, score := summarize(checks)
	findings := findingsFromChecks(checks)
	knownGoodFindings, err := compareObservationsToGood(db, state.observations, scopeKeyForWorkspace(opts.Workspace), configHash(cfg))
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
		ID:                 state.runID,
		Mode:               opts.Mode,
		StartedAt:          start,
		DurationMS:         time.Since(start).Milliseconds(),
		Status:             status,
		HealthScore:        score,
		Workspace:          opts.Workspace,
		ScopeKey:           scopeKeyForWorkspace(opts.Workspace),
		ConfigHash:         configHash(cfg),
		QuestionSetVersion: questionSetVersion,
		AgentKind:          state.agentKind(),
		CloudSynced:        false,
		RawExported:        false,
		RedactionStatus:    state.redactionStatus(),
		Checks:             checks,
		Findings:           findings,
		Responses:          state.responses,
	}
	if !opts.Ephemeral {
		if err := saveRun(db, run, state.observations); err != nil {
			return Run{}, err
		}
		for _, probe := range state.probes {
			if err := saveProbeMessage(db, probe); err != nil {
				return Run{}, err
			}
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
	}
	return run, nil
}

func runIDForOptions(opts RunOptions) string {
	if strings.TrimSpace(opts.RunID) != "" {
		return strings.TrimSpace(opts.RunID)
	}
	return newRunID()
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
	probes       []ProbeMessage
	responses    []ProbeResponse
	redactions   int
}

func (s *runState) addCheck(checkID, lane, kind, status string, severity int, score float64, started time.Time, finding string, metrics map[string]float64) {
	s.addCheckWithDuration(checkID, lane, kind, status, severity, score, time.Since(started).Milliseconds(), finding, metrics)
}

func (s *runState) addCheckWithDuration(checkID, lane, kind, status string, severity int, score float64, durationMS int64, finding string, metrics map[string]float64) {
	s.checks = append(s.checks, CheckResult{
		ID:         fmt.Sprintf("%s:%03d", s.runID, len(s.checks)+1),
		CheckID:    checkID,
		Lane:       lane,
		Kind:       kind,
		Status:     status,
		Severity:   severity,
		Score:      score,
		DurationMS: durationMS,
		Finding:    finding,
		Metrics:    metrics,
	})
}

func (s *runState) hasCriticalPreflight() bool {
	for _, check := range s.checks {
		if check.Severity >= 2 && (check.CheckID == "target.config" || check.CheckID == "safety.scrubber") {
			return true
		}
	}
	return false
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
	if s.cfg.Target.Runtime == "custom" {
		if strings.TrimSpace(s.cfg.AgentCommand) != "" || strings.TrimSpace(s.opts.AgentCommand) != "" || strings.TrimSpace(os.Getenv("BASELINE_AGENT_COMMAND")) != "" {
			s.observe("runtime.custom.command", "configured", "configured")
			s.addCheck("runtime.custom", "core", "environment", "ok", 0, 100, start, "Custom agent command is configured for this Baseline target.", nil)
			return
		}
		s.addCheck("runtime.custom", "core", "environment", "critical", 2, 20, start, "Target runtime is custom, but no agent command is configured.", nil)
		return
	}
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
	workspace := s.commandDir()
	if workspace == "" {
		s.addCheck("repo.state", "baseline", "awareness", "warning", 1, 75, start, "Configured workspace is not a readable directory: "+s.opts.Workspace, nil)
		return
	}
	root, err := commandOutput(s.ctx, 4*time.Second, "git", "-C", workspace, "rev-parse", "--show-toplevel")
	if err != nil {
		s.addCheck("repo.state", "baseline", "awareness", "warning", 1, 75, start, "Workspace is not a git repo or git is unavailable.", nil)
		return
	}
	branch, _ := commandOutput(s.ctx, 4*time.Second, "git", "-C", workspace, "branch", "--show-current")
	status, _ := commandOutput(s.ctx, 4*time.Second, "git", "-C", workspace, "status", "--porcelain")
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
	if s.cfg.Target.Runtime != "openclaw" {
		s.addCheck("mcp.openclaw.config", "baseline", "tooling", "ok", 0, 100, start, "OpenClaw MCP registration is not required for target runtime "+s.cfg.Target.Runtime+".", nil)
		return
	}
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

func (s *runState) checkTargetConfig() {
	start := time.Now()
	target := s.cfg.Target
	switch target.Runtime {
	case "openclaw":
		if _, err := exec.LookPath("openclaw"); err != nil {
			s.addCheck("target.config", "baseline", "configuration", "critical", 2, 20, start, "Target is OpenClaw, but the openclaw binary is not on PATH.", map[string]float64{"timeout_seconds": float64(targetTimeoutSeconds(target))})
			return
		}
	case "custom":
		if strings.TrimSpace(s.cfg.AgentCommand) == "" && strings.TrimSpace(s.opts.AgentCommand) == "" && strings.TrimSpace(os.Getenv("BASELINE_AGENT_COMMAND")) == "" {
			s.addCheck("target.config", "baseline", "configuration", "critical", 2, 20, start, "Target runtime is custom, but no agent command is configured.", map[string]float64{"timeout_seconds": float64(targetTimeoutSeconds(target))})
			return
		}
	default:
		s.addCheck("target.config", "baseline", "configuration", "critical", 2, 10, start, "Target runtime is not understood: "+target.Runtime, nil)
		return
	}
	if target.ModelPolicy == "pinned" && strings.TrimSpace(target.PinnedModel) == "" {
		s.addCheck("target.config", "baseline", "configuration", "critical", 2, 20, start, "Target model policy is pinned, but no pinned model is configured.", nil)
		return
	}
	if target.ModelPolicy != "follow_current" && target.ModelPolicy != "pinned" {
		s.addCheck("target.config", "baseline", "configuration", "critical", 2, 20, start, "Target model policy is not understood: "+target.ModelPolicy, nil)
		return
	}
	s.observe("target.runtime", target.Runtime, target.Runtime)
	s.observe("target.entity", target.Entity, target.Entity)
	s.observe("target.model_policy", target.ModelPolicy, targetModelDisplay(target))
	s.addCheck("target.config", "baseline", "configuration", "ok", 0, 100, start, "Baseline target is "+target.Runtime+" "+target.Entity+" using "+targetModelDisplay(target)+".", map[string]float64{"timeout_seconds": float64(targetTimeoutSeconds(target))})
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
	questions := selectedQuestions(s.cfg, s.opts.Packs)
	if !s.opts.RunAgent && os.Getenv("BASELINE_RUN_AGENT") != "1" && s.opts.Mode != "bootstrap" {
		s.addCheck("questions.runner", "baseline", "agent_eval", "warning", 1, 80, time.Now(), "Question pack was skipped; use baseline run for the real eval path, or pass --run-agent to legacy baseline check --full.", map[string]float64{"questions": float64(len(questions))})
		return
	}
	outcomes := s.runQuestionProbes(questions)
	for _, outcome := range outcomes {
		s.recordQuestionOutcome(outcome)
	}
}

type questionProbeOutcome struct {
	Index    int
	Question Question
	Started  time.Time
	Result   AgentProbeResult
	Err      error
}

func (s *runState) runQuestionProbes(questions []Question) []questionProbeOutcome {
	if len(questions) == 0 {
		return nil
	}
	concurrency := probeConcurrency(len(questions))
	jobs := make(chan questionProbeOutcome)
	results := make(chan questionProbeOutcome, len(questions))
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				job.Started = time.Now()
				job.Result, job.Err = s.askAgentMeasured(job.Question)
				results <- job
			}
		}()
	}
	for i, q := range questions {
		jobs <- questionProbeOutcome{Index: i, Question: q}
	}
	close(jobs)
	wg.Wait()
	close(results)
	outcomes := make([]questionProbeOutcome, len(questions))
	for outcome := range results {
		outcomes[outcome.Index] = outcome
	}
	return outcomes
}

func probeConcurrency(questionCount int) int {
	value := 2
	if configured := strings.TrimSpace(os.Getenv("BASELINE_PROBE_CONCURRENCY")); configured != "" {
		if parsed, err := strconv.Atoi(configured); err == nil {
			value = parsed
		}
	}
	if value < 1 {
		value = 1
	}
	if value > 6 {
		value = 6
	}
	if questionCount > 0 && value > questionCount {
		return questionCount
	}
	return value
}

func (s *runState) recordQuestionOutcome(outcome questionProbeOutcome) {
	q := outcome.Question
	result := outcome.Result
	start := outcome.Started
	duration := time.Since(start).Milliseconds()
	if !result.SystemSendAt.IsZero() && !result.BaselineReceivedAt.IsZero() {
		duration = result.DurationMS
		result.ProbeMessage.PackVersion = packVersionFor(s.cfg, q.PackID)
		result.ProbeMessage.QuestionSetVersion = questionSetVersion
		result.ProbeMessage.PromptHash = hashValue(q.Prompt)
		result.ProbeMessage.ExpectedFactsHash = hashStringSlice(q.ExpectedFacts)
		s.probes = append(s.probes, result.ProbeMessage)
		s.observe("question."+q.PackID+"."+q.ID+".system_send_at", result.SystemSendAt.Format(time.RFC3339Nano), result.SystemSendAt.Format(time.RFC3339Nano))
		s.observe("question."+q.PackID+"."+q.ID+".baseline_received_at", result.BaselineReceivedAt.Format(time.RFC3339Nano), result.BaselineReceivedAt.Format(time.RFC3339Nano))
		s.observeNumber("question."+q.PackID+"."+q.ID+".duration_ms", float64(result.DurationMS))
		if result.TokenStatus == "fresh" && result.TotalTokens != nil {
			s.observeNumber("question."+q.PackID+"."+q.ID+".total_tokens", float64(*result.TotalTokens))
		}
		s.observe("question."+q.PackID+"."+q.ID+".token_status", result.TokenStatus, result.TokenStatus)
	} else {
		s.observeNumber("question."+q.PackID+"."+q.ID+".latency_ms", float64(duration))
	}
	scrubbed, report := scrubText(result.Output)
	s.redactions += report.SecretsFound + report.PIIFound
	response := ProbeResponse{
		PackID:           q.PackID,
		ProbeID:          q.ID,
		Dimension:        q.Dimension,
		Prompt:           q.Prompt,
		ExpectedBehavior: q.ExpectedBehavior,
		Output:           result.Output,
		ScrubbedOutput:   scrubbed,
		DurationMS:       duration,
		Status:           "ok",
	}
	if outcome.Err != nil {
		response.Status = "failed"
		response.Error = outcome.Err.Error()
		s.responses = append(s.responses, response)
		s.addCheckWithDuration("question."+q.PackID+"."+q.ID, "baseline", q.Dimension, "critical", 2, 0, duration, "Agent question failed: "+outcome.Err.Error(), map[string]float64{"duration_ms": float64(duration)})
		return
	}
	s.responses = append(s.responses, response)
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
	metrics := map[string]float64{
		"duration_ms": float64(duration),
	}
	if result.TokenStatus == "fresh" && result.InputTokens != nil {
		metrics["input_tokens"] = float64(*result.InputTokens)
	}
	if result.TokenStatus == "fresh" && result.OutputTokens != nil {
		metrics["output_tokens"] = float64(*result.OutputTokens)
	}
	if result.TokenStatus == "fresh" && result.TotalTokens != nil {
		metrics["total_tokens"] = float64(*result.TotalTokens)
	}
	s.observe("question."+q.PackID+"."+q.ID+".answer_hash", scrubbed, displayHash(scrubbed))
	s.addCheckWithDuration("question."+q.PackID+"."+q.ID, "baseline", q.Dimension, status, severity, score*100, duration, finding, metrics)
}

type AgentProbeResult struct {
	Output string
	ProbeMessage
}

type OpenClawTokenMetadata struct {
	TokenStatus   string
	TokenSource   string
	InputTokens   *int
	OutputTokens  *int
	TotalTokens   *int
	ContextTokens *int
	Model         string
	ModelProvider string
}

func (s *runState) askAgentMeasured(q Question) (AgentProbeResult, error) {
	command := strings.TrimSpace(s.opts.AgentCommand)
	if command == "" {
		command = strings.TrimSpace(s.cfg.AgentCommand)
	}
	if command == "" {
		command = strings.TrimSpace(os.Getenv("BASELINE_AGENT_COMMAND"))
	}
	if command != "" {
		ctx, cancel := context.WithTimeout(s.ctx, time.Duration(targetTimeoutSeconds(s.cfg.Target))*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = s.commandDir()
		cmd.Env = append(os.Environ(), "BASELINE_PROMPT="+q.Prompt)
		sendAt := time.Now().UTC()
		out, err := cmd.CombinedOutput()
		receivedAt := time.Now().UTC()
		msg := ProbeMessage{
			RunID:              s.runID,
			PackID:             q.PackID,
			ProbeID:            q.ID,
			SessionID:          "",
			SystemSendAt:       sendAt,
			BaselineReceivedAt: receivedAt,
			DurationMS:         receivedAt.Sub(sendAt).Milliseconds(),
			TokenStatus:        "unavailable",
			TokenSource:        "custom agent command",
		}
		if ctx.Err() == context.DeadlineExceeded {
			return AgentProbeResult{Output: string(out), ProbeMessage: msg}, fmt.Errorf("agent command timed out")
		}
		if err != nil {
			return AgentProbeResult{Output: string(out), ProbeMessage: msg}, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return AgentProbeResult{Output: string(out), ProbeMessage: msg}, nil
	}
	path, err := exec.LookPath("openclaw")
	if err != nil {
		return AgentProbeResult{}, err
	}
	return runOpenClawProbeWithTarget(s.ctx, path, s.runID, q, s.cfg.Target, s.commandDir())
}

func (s *runState) askAgent(prompt string) (string, error) {
	command := strings.TrimSpace(s.opts.AgentCommand)
	if command == "" {
		command = strings.TrimSpace(s.cfg.AgentCommand)
	}
	if command == "" {
		command = strings.TrimSpace(os.Getenv("BASELINE_AGENT_COMMAND"))
	}
	if command != "" {
		ctx, cancel := context.WithTimeout(s.ctx, time.Duration(targetTimeoutSeconds(s.cfg.Target))*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = s.commandDir()
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
	if s.opts.AgentCommand != "" || s.cfg.AgentCommand != "" || os.Getenv("BASELINE_AGENT_COMMAND") != "" {
		return "custom"
	}
	if _, err := exec.LookPath("openclaw"); err == nil {
		return "openclaw"
	}
	return "unknown"
}

func (s *runState) commandDir() string {
	workspace := normalizeWorkspacePath(s.opts.Workspace)
	if workspace == "" {
		return ""
	}
	info, err := os.Stat(workspace)
	if err != nil || !info.IsDir() {
		return ""
	}
	return workspace
}

func (s *runState) redactionStatus() string {
	if s.redactions > 0 {
		return fmt.Sprintf("redacted:%d", s.redactions)
	}
	return "clean"
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
		return "Run baseline run. If this is a custom harness, set target.runtime=custom and agent_command, then run baseline doctor."
	case strings.Contains(checkID, "target.config"):
		return "Run baseline setup to review target configuration, or set target.model_policy and target.pinned_model explicitly."
	case strings.Contains(checkID, "safety.scrubber"):
		return "Keep cloud sync disabled until redaction passes locally."
	default:
		return "Run baseline report and compare against accepted Good Baselines."
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
	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
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
		if !cloudCheckAllowed(check.CheckID) {
			continue
		}
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

func cloudCheckAllowed(checkID string) bool {
	if !strings.HasPrefix(checkID, "question.") {
		return true
	}
	parts := strings.Split(checkID, ".")
	if len(parts) < 3 {
		return false
	}
	packID := parts[1]
	for _, pack := range canonicalMonitorPacks(defaultConfigSeeds()) {
		if pack.ID == packID {
			return pack.Risk.CloudExportAllowed
		}
	}
	return false
}
