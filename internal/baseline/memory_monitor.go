package baseline

// The memory monitor deliberately does not load the ordinary Baseline configuration,
// scoreQuestion, Good acceptance, or cloud outbox. This is a local snapshot experiment.
import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const memoryModel = "gpt-6-astra-900k"
const memoryProvider = "openai-codex"

var memoryFiles = []string{"memories/MEMORY.md", "memories/USER.md", "SOUL.md"}
var memoryID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,95}$`)

type memoryCase struct {
	ID              string   `json:"id"`
	Question        string   `json:"question"`
	Expected        []string `json:"expected,omitempty"`
	ExpectedUnknown bool     `json:"expected_unknown,omitempty"`
	SourceFile      string   `json:"source_file,omitempty"`
	Evidence        []string `json:"evidence,omitempty"`
}
type memoryCases struct {
	Version        string       `json:"version"`
	SourceRevision string       `json:"source_revision"`
	GoldRevision   string       `json:"gold_revision"`
	Cases          []memoryCase `json:"cases"`
}
type memoryAnswer struct {
	ID    string  `json:"id"`
	Value *string `json:"value"`
	Quote string  `json:"quote"`
}

func (a *memoryAnswer) UnmarshalJSON(b []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	if len(fields) != 3 || fields["id"] == nil || fields["value"] == nil || fields["quote"] == nil {
		return errors.New("answer requires exactly id, value and quote")
	}
	type plain memoryAnswer
	return json.Unmarshal(b, (*plain)(a))
}

type memoryUsage struct {
	Input      *int     `json:"input_tokens"`
	Output     *int     `json:"output_tokens"`
	Total      *int     `json:"total_tokens"`
	APICalls   *int     `json:"api_calls"`
	Model      string   `json:"model"`
	Provider   string   `json:"provider"`
	SessionID  string   `json:"session_id"`
	Completed  bool     `json:"completed"`
	Failed     bool     `json:"failed"`
	Cost       *float64 `json:"estimated_cost_usd"`
	CostStatus string   `json:"cost_status"`
}
type memoryResult struct {
	EvaluationProfile  string            `json:"evaluation_profile"`
	Run                Run               `json:"run"`
	RequestID          string            `json:"request_id"`
	Identity           string            `json:"identity"`
	CaseHash           string            `json:"case_hash"`
	ComparisonKey      string            `json:"comparison_key"`
	SourceRevision     string            `json:"source_revision"`
	GoldRevision       string            `json:"gold_revision"`
	SourceHashes       map[string]string `json:"source_hashes"`
	SourceBytes        map[string]int    `json:"source_bytes"`
	HermesRevision     string            `json:"hermes_revision"`
	HermesDiffHash     string            `json:"hermes_diff_hash"`
	ConfiguredModel    string            `json:"configured_model"`
	ConfiguredProvider string            `json:"configured_provider"`
	Reasoning          string            `json:"reasoning"`
	AdapterHash        string            `json:"adapter_hash"`
	UsageStatus        string            `json:"usage_status"`
	Usage              memoryUsage       `json:"usage"`
	Limitation         string            `json:"limitation"`
}

func memoryHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func memoryJSON(v any) []byte    { b, _ := json.MarshalIndent(v, "", "  "); return append(b, '\n') }
func memoryStrict(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}
func memoryHome() error {
	p := os.Getenv("BASELINE_HOME")
	if !filepath.IsAbs(p) {
		return errors.New("memory requires an explicit absolute dedicated BASELINE_HOME")
	}
	resolved, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	if resolved == filepath.Join(homeDir(), ".baseline") {
		return errors.New("refusing the default Baseline home")
	}
	if err := os.MkdirAll(p, 0700); err != nil {
		return err
	}
	for parent := filepath.Clean(p); ; parent = filepath.Dir(parent) {
		if _, e := os.Lstat(filepath.Join(parent, ".git")); e == nil {
			return errors.New("memory home must be outside Git workspaces to prevent ancestor context injection")
		}
		if filepath.Dir(parent) == parent {
			break
		}
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return err
	}
	if real != filepath.Clean(p) {
		return errors.New("BASELINE_HOME must not contain symlinks")
	}
	// A stock configuration is not a memory-monitor home, even if cloud happens to be off.
	if _, err := os.Stat(filepath.Join(p, "config.json")); err == nil {
		return errors.New("dedicated memory home must not contain a stock config.json")
	}
	return nil
}
func cmdMemory(ctx context.Context, args []string, out, errs io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errs, "usage: baseline memory run --source-profile PATH --cases FILE --request-id ID --hermes-python PATH --hermes-source PATH | memory history")
		return 2
	}
	if err := memoryHome(); err != nil {
		fmt.Fprintln(errs, err)
		return 2
	}
	if args[0] == "history" {
		if len(args) != 1 {
			return 2
		}
		if err := memoryHistory(out); err != nil {
			fmt.Fprintln(errs, err)
			return 1
		}
		return 0
	}
	if args[0] != "run" {
		return 2
	}
	fs := flag.NewFlagSet("memory run", flag.ContinueOnError)
	fs.SetOutput(errs)
	profile := fs.String("source-profile", "", "explicit Hermes source profile")
	casesPath := fs.String("cases", "", "held-out local cases and gold JSON")
	request := fs.String("request-id", "", "stable request identity (never automatically reissued)")
	python := fs.String("hermes-python", "", "absolute installed Hermes Python interpreter")
	source := fs.String("hermes-source", "", "absolute installed Hermes source directory")
	timeout := fs.Int("timeout", 90, "wall deadline in seconds (1–300)")
	daily := fs.String("daily-at", "", "optional local HH:MM, derives daily request identity")
	zone := fs.String("timezone", "", "IANA zone required with daily-at")
	if fs.Parse(args[1:]) != nil {
		return 2
	}
	if fs.NArg() != 0 || *timeout < 1 || *timeout > 300 {
		return 2
	}
	if *daily != "" {
		if *request != "" {
			fmt.Fprintln(errs, "request-id and daily-at are mutually exclusive")
			return 2
		}
		id, due, err := memoryDaily(time.Now(), *daily, *zone, *profile, *casesPath)
		if err != nil {
			fmt.Fprintln(errs, err)
			return 2
		}
		if !due {
			fmt.Fprintln(out, `{"status":"not_due","model_called":false}`)
			return 0
		}
		*request = id
	} else if *zone != "" {
		return 2
	}
	if !memoryID.MatchString(*request) {
		fmt.Fprintln(errs, "request-id must be 1–96 safe filename characters")
		return 2
	}
	r, err := memoryRun(ctx, *profile, *casesPath, *request, *python, *source, time.Duration(*timeout)*time.Second)
	if err != nil {
		fmt.Fprintln(errs, err)
		return 1
	}
	fmt.Fprint(out, string(memoryJSON(r)))
	if r.Run.Status != "PASS" {
		return 1
	}
	return 0
}
func memoryDaily(now time.Time, at, zone, profile, cases string) (string, bool, error) {
	loc, err := time.LoadLocation(zone)
	if err != nil || zone == "" {
		return "", false, errors.New("daily-at requires a valid IANA timezone")
	}
	parsed, err := time.Parse("15:04", at)
	if err != nil {
		return "", false, err
	}
	local := now.In(loc)
	due := local.Hour()*60+local.Minute() >= parsed.Hour()*60+parsed.Minute()
	// Stable across edits to a case file: a changed file cannot quietly buy a second call today.
	return "daily-" + local.Format("2006-01-02") + "-" + memoryHash([]byte(filepath.Clean(profile) + "\n" + filepath.Clean(cases) + "\n" + zone))[:16], due, nil
}
func memoryReadCases(path string) (memoryCases, []byte, error) {
	var c memoryCases
	b, err := memoryReadRegular(path, 65536)
	if err != nil {
		return c, nil, err
	}
	if err = memoryStrict(b, &c); err != nil {
		return c, nil, err
	}
	if c.Version == "" || c.SourceRevision == "" || c.GoldRevision == "" || len(c.Cases) < 4 || len(c.Cases) > 6 {
		return c, nil, errors.New("cases require version, source_revision, gold_revision and 4–6 cases")
	}
	ids := map[string]bool{}
	for _, q := range c.Cases {
		if !memoryID.MatchString(q.ID) || ids[q.ID] || strings.TrimSpace(q.Question) == "" || len(q.Question) > 1000 {
			return c, nil, errors.New("invalid or duplicate case")
		}
		ids[q.ID] = true
		if q.ExpectedUnknown {
			if len(q.Expected) > 0 || len(q.Evidence) > 0 || q.SourceFile != "" {
				return c, nil, errors.New("unknown case must not contain known gold")
			}
			continue
		}
		valid := false
		for _, f := range memoryFiles {
			valid = valid || q.SourceFile == f
		}
		if !valid || len(q.Expected) == 0 || len(q.Evidence) == 0 {
			return c, nil, errors.New("known case requires expected values, evidence quotes and an allowed source_file")
		}
		for _, s := range append(append([]string{}, q.Expected...), q.Evidence...) {
			if strings.TrimSpace(s) == "" {
				return c, nil, errors.New("empty gold value/evidence")
			}
		}
	}
	return c, b, nil
}
func memoryReadRegular(path string, limit int64) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > limit {
		return nil, fmt.Errorf("not a bounded regular file: %s", path)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	abs, _ := filepath.Abs(path)
	if real != abs {
		return nil, fmt.Errorf("symlink not allowed: %s", path)
	}
	return os.ReadFile(path)
}
func memorySnapshot(profile string) (map[string][]byte, error) {
	if !filepath.IsAbs(profile) {
		return nil, errors.New("source-profile must be absolute")
	}
	files := map[string][]byte{}
	for _, name := range memoryFiles {
		b, err := memoryReadRegular(filepath.Join(profile, name), 32768)
		if err != nil {
			return nil, err
		}
		files[name] = b
	}
	// Record model/config identity without copying credentials or production configuration.
	b, err := memoryReadRegular(filepath.Join(profile, "config.yaml"), 262144)
	if err != nil {
		return nil, err
	}
	files["config.yaml"] = b
	return files, nil
}
func memoryPrompt(c memoryCases) []byte {
	qs := []map[string]string{}
	for _, q := range c.Cases {
		qs = append(qs, map[string]string{"id": q.ID, "question": q.Question})
	}
	return []byte("This is an isolated snapshot_recall check. Answer only from the injected identity and memory. Do not infer or guess missing facts. Return exactly one JSON object {\"answers\":[{\"id\":\"case id\",\"value\":\"short factual value\",\"quote\":\"exact supporting quote from injected source\"}]}. For unknown facts use value:null and quote:\"\". No markdown. One answer per case.\nQuestions:\n" + string(memoryJSON(qs)))
}
func memoryGrade(c memoryCases, files map[string][]byte, output []byte) ([]CheckResult, string) {
	var response struct {
		Answers []memoryAnswer `json:"answers"`
	}
	if err := memoryStrict(output, &response); err != nil {
		return nil, "invalid_response"
	}
	answers := map[string]memoryAnswer{}
	for _, a := range response.Answers {
		if _, ok := answers[a.ID]; ok {
			return nil, "duplicate_answer"
		}
		answers[a.ID] = a
	}
	if len(answers) != len(c.Cases) {
		return nil, "missing_or_extra_answer"
	}
	checks := []CheckResult{}
	norm := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
	for _, q := range c.Cases {
		a, ok := answers[q.ID]
		if !ok {
			return nil, "missing_answer"
		}
		status, reason := "FAIL", "incorrect_recall"
		score := float64(0)
		if q.ExpectedUnknown {
			if a.Value == nil && a.Quote == "" {
				status, reason = "PASS", "expected_unknown"
			} else {
				reason = "unsupported_answer"
			}
		} else if a.Value == nil {
			reason = "missing_recall"
		} else {
			valueOK, quoteOK := false, false
			for _, v := range q.Expected {
				valueOK = valueOK || norm(v) == norm(*a.Value)
			}
			for _, v := range q.Evidence {
				quoteOK = quoteOK || strings.Contains(a.Quote, v)
			}
			if valueOK && quoteOK && strings.Contains(string(files[q.SourceFile]), a.Quote) {
				status, reason = "PASS", "sourced_recall"
			} else if valueOK {
				reason = "unsupported_evidence"
			}
		}
		if status == "PASS" {
			score = 100
		}
		checks = append(checks, CheckResult{ID: q.ID, CheckID: q.ID, Lane: "memory", Kind: "snapshot_recall", Status: status, Score: score, Finding: reason})
	}
	return checks, ""
}
func memoryRun(ctx context.Context, profile, casesPath, request, python, source string, timeout time.Duration) (memoryResult, error) {
	var result memoryResult
	if !filepath.IsAbs(python) || !filepath.IsAbs(source) {
		return result, errors.New("hermes-python and hermes-source must be absolute")
	}
	if filepath.Clean(profile) == filepath.Clean(baseDir()) || strings.HasPrefix(filepath.Clean(baseDir())+"/", filepath.Clean(profile)+"/") {
		return result, errors.New("monitor home must be outside the source profile")
	}
	if !filepath.IsAbs(casesPath) || filepath.Clean(casesPath) == filepath.Clean(profile) || strings.HasPrefix(filepath.Clean(casesPath)+"/", filepath.Clean(profile)+"/") {
		return result, errors.New("held-out cases must be an absolute path outside the source profile")
	}
	c, gold, err := memoryReadCases(casesPath)
	if err != nil {
		return result, err
	}
	files, err := memorySnapshot(profile)
	if err != nil {
		return result, err
	}
	lock, err := os.OpenFile(filepath.Join(baseDir(), "memory.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return result, err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return result, errors.New("another memory invocation holds the lock")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	// Request identity excludes changing snapshot contents: repeated daily requests return
	// their original measurement, rather than silently rerunning after source drift.
	identity := memoryHash([]byte(profile + "\n" + memoryHash(gold) + "\n" + python + "\n" + source + "\n" + memoryModel))
	runID := "memory-" + request
	dir := runArtifactDir(runID)
	evaluationProfile, err := memoryEvaluationProfile(runID)
	if err != nil {
		return result, err
	}
	resultPath := filepath.Join(dir, "memory.json")
	if b, e := os.ReadFile(resultPath); e == nil {
		if e = json.Unmarshal(b, &result); e != nil {
			return result, e
		}
		if result.Identity != identity {
			return result, errors.New("conflicting request retry")
		}
		return result, nil
	}
	if err = os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return result, err
	}
	if err = os.Mkdir(dir, 0700); err != nil {
		return result, errors.New("request already started without a final result; retained as inconclusive, use history and investigate before a new request")
	}
	started := time.Now().UTC()
	run := Run{ID: runID, Mode: "snapshot_recall", StartedAt: started, Status: "INCONCLUSIVE", Workspace: profile, AgentKind: "hermes", QuestionSetVersion: c.Version, RedactionStatus: "local_private_snapshot"}
	result = memoryResult{EvaluationProfile: evaluationProfile, ConfiguredModel: memoryModel, ConfiguredProvider: memoryProvider, Reasoning: "medium", Run: run, RequestID: request, Identity: identity, CaseHash: memoryHash(gold), SourceRevision: c.SourceRevision, GoldRevision: c.GoldRevision, SourceHashes: map[string]string{}, SourceBytes: map[string]int{}, AdapterHash: memoryHash([]byte(memoryAdapter)), UsageStatus: "unavailable", Limitation: "Tool-less snapshot recall only; not Oracle retrieval, session search, production write-back or long-term retention."}
	result.ComparisonKey = memoryHash([]byte(profile + "\n" + result.CaseHash + "\n" + memoryModel + "\n" + result.AdapterHash))
	result.Run.ScopeKey = result.ComparisonKey
	for name, b := range files {
		result.SourceHashes[name] = memoryHash(b)
		result.SourceBytes[name] = len(b)
	}
	result.Run.ConfigHash = memoryHash(memoryJSON(result.SourceHashes))
	if err = atomicWrite(filepath.Join(dir, "started.json"), memoryJSON(result), 0600); err != nil {
		return result, err
	}
	// From here every failure is retained and indexed, including startup and timeout failures.
	problem := func(reason string) {
		result.Run.Status = "INCONCLUSIVE"
		result.Run.Findings = append(result.Run.Findings, Finding{Severity: "high", CheckID: "infrastructure", Message: reason})
	}
	home := evaluationProfile
	if err = os.MkdirAll(filepath.Dir(home), 0700); err != nil {
		return result, err
	}
	if err = os.Mkdir(home, 0700); err != nil {
		return result, fmt.Errorf("evaluation profile already exists or cannot be created; request retained: %w", err)
	}
	work := filepath.Join(home, "work")
	if err = os.MkdirAll(filepath.Join(home, "memories"), 0700); err != nil {
		return result, err
	}
	if err = os.Mkdir(work, 0700); err != nil {
		return result, err
	}
	for _, name := range memoryFiles {
		if err = atomicWrite(filepath.Join(home, name), files[name], 0600); err != nil {
			return result, err
		}
	}
	prompt := memoryPrompt(c)
	query := filepath.Join(home, "query.txt")
	usagePath := filepath.Join(home, "usage.json")
	proofPath := filepath.Join(home, "isolation.json")
	if err = atomicWrite(query, prompt, 0600); err != nil {
		return result, err
	}
	adapter := filepath.Join(home, "adapter.py")
	if err = atomicWrite(adapter, []byte(memoryAdapter), 0600); err != nil {
		return result, err
	}
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, python, adapter, source, query, usagePath, proofPath, fmt.Sprint(int(timeout.Seconds())), filepath.Join(profile, "config.yaml"))
	cmd.Dir = work
	cmd.Env = append(memoryEnvironment(home), "PYTHONPATH="+source)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	var stdout, stderr memoryBoundedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	callErr := cmd.Run()
	finished := time.Now().UTC()
	_ = atomicWrite(filepath.Join(dir, "output.txt"), stdout.Bytes(), 0600)
	_ = atomicWrite(filepath.Join(dir, "stderr.txt"), stderr.Bytes(), 0600)
	answer, nativeUsage, streamErr := memoryParseStream(stdout.Bytes())
	result.Usage = nativeUsage
	if nativeUsage.Total != nil {
		result.UsageStatus = "native_stream_json"
	}
	if err = atomicWrite(usagePath, memoryJSON(nativeUsage), 0600); err != nil {
		return result, err
	}

	var proof struct {
		ToolCount         int    `json:"tool_count"`
		AuthRoot          string `json:"auth_root"`
		EvaluationProfile string `json:"evaluation_profile"`
		MaxTurns          int    `json:"max_turns"`
		Revision          string `json:"revision"`
		DiffHash          string `json:"diff_hash"`
		Model             string `json:"model"`
		Provider          string `json:"provider"`
		Reasoning         string `json:"reasoning"`
	}
	proofBytes, proofErr := os.ReadFile(proofPath)
	if proofErr == nil {
		proofErr = json.Unmarshal(proofBytes, &proof)
	}
	result.HermesRevision = proof.Revision
	result.HermesDiffHash = proof.DiffHash
	result.ComparisonKey = memoryHash([]byte(result.ComparisonKey + "\n" + proof.Revision + "\n" + proof.DiffHash))
	result.Run.ScopeKey = result.ComparisonKey
	if commandCtx.Err() != nil {
		problem("deadline_exceeded")
	} else if callErr != nil {
		problem("Hermes process failed; private stderr retained")
	} else if proofErr != nil || proof.AuthRoot != filepath.Join(homeDir(), ".hermes") || proof.EvaluationProfile != home || proof.ToolCount != 0 || proof.MaxTurns != 1 || proof.Revision == "" || proof.Model != memoryModel || proof.Provider != memoryProvider || proof.Reasoning != "medium" {
		problem("isolation/model proof unavailable or mismatched")
	} else if streamErr != nil {
		problem("native stream invalid: " + streamErr.Error())
	} else {
		checks, reason := memoryGrade(c, files, answer)
		for i := range checks {
			checks[i].ID = runID + "-" + checks[i].CheckID
		}
		result.Run.Checks = checks
		if reason != "" {
			problem(reason)
		} else {
			result.Run.Status = "PASS"
			for _, check := range checks {
				if check.Status != "PASS" {
					result.Run.Status = "FAIL"
				}
			}
		}
	}
	after, e := memorySnapshot(profile)
	if e != nil {
		problem("source profile unreadable after run")
	} else {
		for name, b := range after {
			if memoryHash(b) != result.SourceHashes[name] {
				problem("source profile changed during run")
				break
			}
		}
	}
	result.Run.DurationMS = finished.Sub(started).Milliseconds()
	passed := 0
	for _, ch := range result.Run.Checks {
		if ch.Status == "PASS" {
			passed++
		}
	}
	result.Run.HealthScore = passed * 100 / len(c.Cases)
	result.Run.Responses = []ProbeResponse{{PackID: "memory", ProbeID: "batch", Prompt: string(prompt), Output: string(stdout.Bytes()), ScrubbedOutput: "Private snapshot response retained locally.", Status: result.Run.Status, DurationMS: result.Run.DurationMS}}
	artifacts, e := writeRunArtifacts(result.Run)
	if e != nil {
		return result, e
	}
	result.Run.Artifacts = artifacts
	// Avoid generic report advice to accept a Good or run broad probes.
	report := fmt.Sprintf("# Snapshot recall — %s\n\nRun: %s\n\n%s\n\nCase version: %s; native usage: %s.\n", result.Run.Status, runID, result.Limitation, c.Version, result.UsageStatus)
	for _, ch := range result.Run.Checks {
		report += fmt.Sprintf("\n- %s: %s (%s)\n", ch.CheckID, ch.Status, ch.Finding)
	}
	for _, f := range result.Run.Findings {
		report += "\nInfrastructure: " + f.Message + "\n"
	}
	if err = atomicWrite(artifacts.ReportPath, []byte(report), 0600); err != nil {
		return result, err
	}
	if err = atomicWrite(artifacts.ReceiptPath, []byte(report), 0600); err != nil {
		return result, err
	}
	if err = atomicWrite(artifacts.MetricsPath, memoryJSON(map[string]any{"run_id": runID, "mode": "snapshot_recall", "status": result.Run.Status, "case_count": len(c.Cases), "passed_cases": passed, "duration_ms": result.Run.DurationMS, "usage_status": result.UsageStatus, "usage": result.Usage}), 0600); err != nil {
		return result, err
	}
	db, e := openDB()
	if e != nil {
		return result, e
	}
	defer db.Close()
	obs := []Observation{{Key: "memory.metadata", ValueHash: result.Run.ConfigHash, RedactedDisplay: string(memoryJSON(result))}}
	if err = saveRun(db, result.Run, obs); err != nil {
		return result, err
	}
	msg := ProbeMessage{RunID: runID, PackID: "memory", PackVersion: c.Version, ProbeID: "batch", QuestionSetVersion: c.Version, PromptHash: memoryHash(prompt), ExpectedFactsHash: result.CaseHash, SessionID: result.Usage.SessionID, SystemSendAt: started, BaselineReceivedAt: finished, DurationMS: result.Run.DurationMS, TokenStatus: result.UsageStatus, TokenSource: "hermes_chat_stream_json", InputTokens: result.Usage.Input, OutputTokens: result.Usage.Output, TotalTokens: result.Usage.Total, Model: result.Usage.Model, ModelProvider: result.Usage.Provider}
	if err = saveProbeMessage(db, msg); err != nil {
		return result, err
	}
	if err = atomicWrite(resultPath, memoryJSON(result), 0600); err != nil {
		return result, err
	}
	return result, nil
}

// Stay beneath the native root so Hermes can use its existing global credential
// fallback. This directory holds only disposable evaluation state, never credentials.
func memoryEvaluationProfile(runID string) (string, error) {
	home := filepath.Join(homeDir(), ".hermes", ".baseline-evals", memoryHash([]byte(filepath.Clean(baseDir()))), runID)
	for p := home; ; p = filepath.Dir(p) {
		if st, err := os.Lstat(p); err == nil {
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("evaluation ancestry must be real directories: %s", p)
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
			return "", errors.New("evaluation profile must be outside Git workspaces")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	privateRoot := filepath.Join(homeDir(), ".hermes", ".baseline-evals")
	for _, p := range []string{privateRoot, filepath.Dir(home)} {
		if st, err := os.Stat(p); err == nil && st.Mode().Perm()&0077 != 0 {
			return "", fmt.Errorf("evaluation parent must be private (0700): %s", p)
		}
	}
	return home, nil
}
func memoryEnvironment(home string) []string {
	env := []string{"HOME=" + homeDir(), "PATH=" + os.Getenv("PATH"), "LANG=en_US.UTF-8", "HERMES_HOME=" + home, "HERMES_SAFE_MODE=1", "PYTHONNOUSERSITE=1", "PYTHONDONTWRITEBYTECODE=1", "TERMINAL_CWD=" + filepath.Join(home, "work")}
	for _, key := range []string{"SSL_CERT_FILE", "SSL_CERT_DIR", "TMPDIR"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	return env
}

type memoryBoundedBuffer struct{ bytes.Buffer }

func (b *memoryBoundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 65536 - b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}
func memoryHistory(out io.Writer) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT o.redacted_display FROM observations o JOIN runs r ON r.id=o.run_id WHERE o.key='memory.metadata' AND r.mode='snapshot_recall' ORDER BY r.started_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Fprintln(out, "snapshot_recall — comparable only within the same comparison key; no Oracle/session-search claim")
	previous := map[string]memoryResult{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return err
		}
		var r memoryResult
		if err = json.Unmarshal([]byte(raw), &r); err != nil {
			return err
		}
		delta := "first comparable measurement"
		if r.Run.Status == "INCONCLUSIVE" {
			delta = "not comparable: infrastructure/measurement failure"
		}
		if old, ok := previous[r.ComparisonKey]; ok && r.Run.Status != "INCONCLUSIVE" {
			delta = fmt.Sprintf("score delta %+d vs %s", r.Run.HealthScore-old.Run.HealthScore, old.Run.ID)
		}
		fmt.Fprintf(out, "%s %s %s score=%d %s usage=%s key=%s\n", r.Run.StartedAt.Format(time.RFC3339), r.Run.ID, r.Run.Status, r.Run.HealthScore, delta, r.UsageStatus, r.ComparisonKey[:12])
		if r.Run.Status != "INCONCLUSIVE" {
			previous[r.ComparisonKey] = r
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	dirs, _ := filepath.Glob(filepath.Join(baseDir(), "reports", "memory-*", "started.json"))
	for _, p := range dirs {
		if _, err := os.Stat(filepath.Join(filepath.Dir(p), "memory.json")); errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(out, "%s INCONCLUSIVE unfinished request retained; no automatic repeat\n", filepath.Base(filepath.Dir(p)))
		}
	}
	return nil
}

// Native chat supplies the supported max-turns/run-budget/stream-json contract.
// This preflight only configures an isolated profile and checks the native tool
// resolver; it execs the unmodified CLI. No model or agent constructor is patched.
const memoryAdapter = `import hashlib, json, os, pathlib, subprocess, sys
from contextlib import redirect_stdout
os.umask(0o077)
source, query, usage, proof, deadline, source_config = sys.argv[1:]
home = pathlib.Path(os.environ["HERMES_HOME"])
if (pathlib.Path(source) / ".env").exists():
    raise RuntimeError("Hermes source .env fallback would defeat isolation")
sys.path.insert(0, source)
from hermes_constants import get_default_hermes_root
auth_root = get_default_hermes_root().resolve()
if auth_root != (pathlib.Path.home()/".hermes").resolve() or not home.resolve().is_relative_to(auth_root/".baseline-evals"):
    raise RuntimeError("isolated profile does not retain the native global auth root")
import yaml
original = yaml.safe_load(pathlib.Path(source_config).read_text())
if original.get("model",{}).get("default") != "gpt-6-astra-900k" or original.get("model",{}).get("provider") != "openai-codex" or original.get("agent",{}).get("reasoning_effort") != "medium":
    raise RuntimeError("source profile identity does not match James Astra900K medium")
from toolsets import TOOLSETS
disabled = sorted(k for k,v in TOOLSETS.items() if not k.startswith("hermes-") and not v.get("includes") and not v.get("posture"))
config = {"security":{"tirith_enabled":False},"model":{"default":"gpt-6-astra-900k","provider":"openai-codex"}, "platform_toolsets":{"cli":[]}, "agent":{"disabled_toolsets":disabled,"max_turns":1,"run_budget_seconds":int(deadline),"reasoning_effort":"medium","empty_response_guard":{"enabled":False}}, "memory":{"memory_enabled":True,"user_profile_enabled":True,"provider":"","memory_char_limit":2200,"user_char_limit":1375}, "auxiliary":{"background_review":{"enabled":False}}, "compression":{"enabled":False}, "context":{"engine":"compressor"}, "skills":{"auto_load":[]}, "fallback_model":None,"smart_model_routing":{"enabled":False}}
(home / "config.yaml").write_text(json.dumps(config))
with redirect_stdout(sys.stderr):
    from hermes_cli.config import load_config
    from hermes_cli.tools_config import _get_platform_tools
    from model_tools import get_tool_definitions
    cfg = load_config()
    selected = sorted(_get_platform_tools(cfg, "cli"))
    tools = get_tool_definitions(enabled_toolsets=selected, disabled_toolsets=disabled, quiet_mode=True)
if selected or tools:
    raise RuntimeError("nonempty native tools in isolated profile")
revision = subprocess.check_output(["git","-C",source,"rev-parse","HEAD"],text=True).strip()
diff_hash = hashlib.sha256(subprocess.check_output(["git","-C",source,"diff","HEAD","--"])).hexdigest()
pathlib.Path(proof).write_text(json.dumps({"auth_root":str(auth_root),"evaluation_profile":str(home.resolve()),"diff_hash":diff_hash,"tool_count":len(tools),"max_turns":1,"revision":revision,"model":"gpt-6-astra-900k","provider":"openai-codex","reasoning":"medium"}))
console = pathlib.Path(sys.executable).with_name("hermes")
if not console.is_file():
    raise RuntimeError("installed Hermes console entrypoint is missing beside the configured Python")
os.execv(sys.executable, [sys.executable,str(console),"--profile","default","chat","--max-turns","1","--run-budget",deadline,"--format","stream-json","--query-file",query,"--source","baseline-memory","--model","gpt-6-astra-900k","--provider","openai-codex","--reasoning","medium"])
`

func memoryParseStream(raw []byte) ([]byte, memoryUsage, error) {
	u := memoryUsage{CostStatus: "unavailable_native_stream_json"}
	var answer []byte
	initSeen, resultSeen := false, false
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 65536)
	for scanner.Scan() {
		var e struct {
			Type      string `json:"type"`
			Subtype   string `json:"subtype"`
			Model     string `json:"model"`
			SessionID string `json:"session_id"`
			ExitCode  *int   `json:"exit_code"`
			Text      string `json:"text"`
			Error     string `json:"error"`
			Tokens    struct {
				Input  *int `json:"input"`
				Output *int `json:"output"`
				Total  *int `json:"total"`
			} `json:"tokens"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, u, errors.New("non-JSON stdout")
		}
		if resultSeen {
			return nil, u, errors.New("events after terminal result")
		}
		switch e.Type {
		case "system":
			if e.Subtype != "init" || initSeen || e.Model != memoryModel {
				return nil, u, errors.New("model/init mismatch")
			}
			initSeen = true
			u.Model = e.Model
			u.SessionID = e.SessionID
		case "text":
			if !initSeen {
				return nil, u, errors.New("text before init")
			}
		case "result":
			resultSeen = true
			answer = []byte(e.Text)
			if e.SessionID != "" {
				u.SessionID = e.SessionID
			}
			u.Failed = !initSeen || e.ExitCode == nil || *e.ExitCode != 0 || e.Error != ""
			u.Completed = !u.Failed
			if e.Tokens.Total != nil && *e.Tokens.Total > 0 && e.Tokens.Input != nil && *e.Tokens.Input >= 0 && e.Tokens.Output != nil && *e.Tokens.Output >= 0 {
				u.Input = e.Tokens.Input
				u.Output = e.Tokens.Output
				u.Total = e.Tokens.Total
			}
			if u.Failed {
				return answer, u, errors.New("native result failed")
			}
		default:
			return nil, u, errors.New("unexpected event (tools forbidden)")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, u, err
	}
	if !resultSeen {
		return nil, u, errors.New("missing terminal result")
	}
	return answer, u, nil
}
