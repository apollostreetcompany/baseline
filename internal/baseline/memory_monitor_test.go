package baseline

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func memoryFixture(t *testing.T) (profile, cases, python, source string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("BASELINE_HOME", filepath.Join(root, "monitor"))
	if err := memoryHome(); err != nil {
		t.Fatal(err)
	}
	profile = filepath.Join(root, "source-profile")
	source = filepath.Join(root, "hermes-code")
	os.MkdirAll(filepath.Join(profile, "memories"), 0700)
	os.Mkdir(source, 0700)
	for _, f := range memoryFiles {
		os.WriteFile(filepath.Join(profile, f), []byte("Planning timezone is Asia/Tokyo. Garden owns project state."), 0600)
	}
	os.WriteFile(filepath.Join(profile, "config.yaml"), []byte("private source config; never copied"), 0600)
	os.WriteFile(filepath.Join(profile, "state.db"), []byte("production database sentinel"), 0600)
	os.Mkdir(filepath.Join(profile, "skills"), 0700)
	os.WriteFile(filepath.Join(profile, "skills", "private.txt"), []byte("not for snapshot"), 0600)
	gold := memoryCases{Version: "v1", SourceRevision: "source-ref", GoldRevision: "gold-ref", Cases: []memoryCase{
		{ID: "timezone", Question: "Planning timezone?", Expected: []string{"Asia/Tokyo"}, SourceFile: "memories/USER.md", Evidence: []string{"Planning timezone is Asia/Tokyo."}},
		{ID: "authority", Question: "Project state authority?", Expected: []string{"Garden"}, SourceFile: "SOUL.md", Evidence: []string{"Garden owns project state."}},
		{ID: "unknown", Question: "Personal secret code?", ExpectedUnknown: true},
		{ID: "missing", Question: "Cron task alert threshold?", Expected: []string{"3"}, SourceFile: "memories/MEMORY.md", Evidence: []string{"Alert when jobs exceed 3."}},
	}}
	cases = filepath.Join(root, "gold.json")
	os.WriteFile(cases, memoryJSON(gold), 0600)
	python = filepath.Join(root, "fake-python")
	fake := `#!/usr/bin/env python3
import json,os,pathlib,sys
adapter,source,query,usage,proof,deadline,source_config=sys.argv[1:]
home=pathlib.Path(os.environ['HERMES_HOME'])
assert os.environ['HERMES_SAFE_MODE']=='1'
assert 'HERMES_PROFILE' not in os.environ
assert 'HERMES_KANBAN_TASK' not in os.environ
assert not (home/'state.db').exists()
assert not (home/'skills').exists()
assert 'Alert when jobs exceed 3.' not in pathlib.Path(query).read_text()
assert 'private source config' not in (home/'memories/MEMORY.md').read_text()
assert str(home/'work')==os.getcwd()
pathlib.Path(proof).write_text(json.dumps(dict(auth_root=str(pathlib.Path(os.environ['HOME'])/'.hermes'),evaluation_profile=str(home),tool_count=0,max_turns=1,revision='hermes-fixture',model='gpt-6-astra-900k',provider='openai-codex',reasoning='medium')))
(home/'state.db').write_text('isolated new database')
answers=dict(answers=[dict(id='timezone',value='Asia/Tokyo',quote='Planning timezone is Asia/Tokyo.'),dict(id='authority',value='Garden',quote='Garden owns project state.'),dict(id='unknown',value=None,quote=''),dict(id='missing',value=None,quote='')])
print(json.dumps(dict(type='system',subtype='init',model='gpt-6-astra-900k',session_id='fresh')))
print(json.dumps(dict(type='result',exit_code=0,session_id='fresh',text=json.dumps(answers),tokens=dict(input=1200,output=100,total=1300))))
`
	os.WriteFile(python, []byte(fake), 0700)
	return
}
func TestMemoryStrictGrading(t *testing.T) {
	profile, cases, _, _ := memoryFixture(t)
	c, _, _ := memoryReadCases(cases)
	files, _ := memorySnapshot(profile)
	good := []memoryAnswer{{ID: "timezone", Value: ptrMemory("Asia/Tokyo"), Quote: "Planning timezone is Asia/Tokyo."}, {ID: "authority", Value: ptrMemory("Garden"), Quote: "Garden owns project state."}, {ID: "unknown"}, {ID: "missing"}}
	grade := func(a []memoryAnswer) []CheckResult {
		t.Helper()
		b := memoryJSON(map[string]any{"answers": a})
		checks, why := memoryGrade(c, files, b)
		if why != "" {
			t.Fatal(why)
		}
		return checks
	}
	checks := grade(good)
	if checks[0].Status != "PASS" || checks[2].Status != "PASS" || checks[3].Finding != "missing_recall" {
		t.Fatal(checks)
	}
	good[0].Quote = "Planning timezone is Asia/Tokyo. Garden owns project state."
	if grade(good)[0].Status != "PASS" {
		t.Fatal("valid larger source quote failed")
	}
	good[0].Quote = "Planning timezone is Asia/Tokyo. Fabricated extra words."
	if grade(good)[0].Status != "FAIL" {
		t.Fatal("fabricated larger quote passed")
	}
	good[0].Quote = "Planning timezone is Asia/Tokyo."
	good[0].Value = ptrMemory("Not Asia/Tokyo")
	if grade(good)[0].Status != "FAIL" {
		t.Fatal("negation passed")
	}
	good[0].Value = ptrMemory("Asia/Tokyo")
	good[0].Quote = "Garden owns project state."
	if grade(good)[0].Finding != "unsupported_evidence" {
		t.Fatal("unrelated evidence passed")
	}
	good[0].Quote = "Planning timezone is Asia/Tokyo."
	good[3].Value = ptrMemory("3")
	good[3].Quote = "Alert when jobs exceed 3."
	if grade(good)[3].Status != "FAIL" {
		t.Fatal("absent snapshot evidence passed")
	}
	good[2].Value = ptrMemory("invented")
	if grade(good)[2].Finding != "unsupported_answer" {
		t.Fatal("hallucination passed")
	}
	for _, raw := range []string{`{}`, `{"answers":[]}`, `{"answers":[{"id":"unknown","quote":""}]}`, `{"answers":[{"id":"x"},{"id":"x"}]}`, "```json\n{}\n```"} {
		if _, why := memoryGrade(c, files, []byte(raw)); why == "" {
			t.Fatalf("malformed accepted: %s", raw)
		}
	}
}
func ptrMemory(s string) *string { return &s }
func TestMemoryIsolatedBatchDedupAndHistory(t *testing.T) {
	profile, cases, python, source := memoryFixture(t)
	t.Setenv("HERMES_PROFILE", "orchestration")
	t.Setenv("HERMES_KANBAN_TASK", "do-not-inherit")
	r, err := memoryRun(context.Background(), profile, cases, "first", python, source, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if r.Run.Status != "FAIL" || r.UsageStatus != "native_stream_json" || r.Usage.Total == nil || *r.Usage.Total != 1300 {
		t.Fatalf("%+v", r)
	}
	b, _ := os.ReadFile(filepath.Join(profile, "state.db"))
	if string(b) != "production database sentinel" {
		t.Fatal("source database changed")
	}
	// An exact retry returns the original result without executing the now-broken binary.
	os.Chmod(python, 0600)
	retry, err := memoryRun(context.Background(), profile, cases, "first", python, source, time.Second)
	if err != nil || retry.Run.ID != r.Run.ID {
		t.Fatal(err, retry)
	}
	c, _, _ := memoryReadCases(cases)
	c.GoldRevision = "changed"
	os.WriteFile(cases, memoryJSON(c), 0600)
	if _, err = memoryRun(context.Background(), profile, cases, "first", python, source, time.Second); err == nil {
		t.Fatal("conflicting retry accepted")
	}
	c.GoldRevision = "gold-ref"
	os.WriteFile(cases, memoryJSON(c), 0600)
	os.Chmod(python, 0700)
	r2, err := memoryRun(context.Background(), profile, cases, "second", python, source, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if r2.ComparisonKey != r.ComparisonKey {
		t.Fatal("same-case run not comparable")
	}
	var history bytes.Buffer
	if err = memoryHistory(&history); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(history.String(), "score delta +0 vs memory-first") {
		t.Fatal(history.String())
	}
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	db.QueryRow(`SELECT count(*) FROM probe_messages`).Scan(&n)
	if n != 2 {
		t.Fatalf("usage counted %d times", n)
	}
	db.QueryRow(`SELECT count(*) FROM sync_outbox`).Scan(&n)
	if n != 0 {
		t.Fatal("cloud staged")
	}
}
func TestMemoryTimeoutRetainedWithoutRetry(t *testing.T) {
	profile, cases, python, source := memoryFixture(t)
	os.WriteFile(python, []byte("#!/bin/sh\nsleep 10\n"), 0700)
	start := time.Now()
	r, err := memoryRun(context.Background(), profile, cases, "timeout", python, source, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second || r.Run.Status != "INCONCLUSIVE" || r.Usage.Total != nil {
		t.Fatal(r)
	}
	if r.Run.Findings[0].Message != "deadline_exceeded" {
		t.Fatal(r.Run.Findings)
	}
	_, err = memoryRun(context.Background(), profile, cases, "timeout", python, source, time.Second)
	if err != nil {
		t.Fatal(err)
	}
}
func TestMemoryLockAndInterruptedRequest(t *testing.T) {
	profile, cases, python, source := memoryFixture(t)
	lock, _ := os.OpenFile(filepath.Join(baseDir(), "memory.lock"), os.O_CREATE|os.O_RDWR, 0600)
	defer lock.Close()
	syscall.Flock(int(lock.Fd()), syscall.LOCK_EX)
	if _, err := memoryRun(context.Background(), profile, cases, "lock", python, source, time.Second); err == nil {
		t.Fatal("concurrent invocation accepted")
	}
	syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	dir := runArtifactDir("memory-interrupted")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "started.json"), []byte(`{}`), 0600)
	if _, err := memoryRun(context.Background(), profile, cases, "interrupted", python, source, time.Second); err == nil {
		t.Fatal("interrupted request reissued")
	}
	var b bytes.Buffer
	memoryHistory(&b)
	if !strings.Contains(b.String(), "INCONCLUSIVE unfinished") {
		t.Fatal(b.String())
	}
}
func TestMemoryNativeStream(t *testing.T) {
	init := `{"type":"system","subtype":"init","model":"gpt-6-astra-900k"}` + "\n"
	result := `{"type":"result","exit_code":0,"text":"{}","tokens":{"input":4,"output":2,"total":6}}`
	_, u, err := memoryParseStream([]byte(init + result))
	if err != nil || u.Total == nil || *u.Total != 6 {
		t.Fatal(u, err)
	}
	_, u, err = memoryParseStream([]byte(init + strings.ReplaceAll(result, `"total":6`, `"total":0`)))
	if err != nil || u.Total != nil {
		t.Fatal("zero telemetry fabricated", u, err)
	}
	for _, s := range []string{init, init + `{"type":"tool_use","name":"memory"}`, init + result + "\n" + result, strings.ReplaceAll(init, "gpt-6-astra-900k", "wrong") + result, init + strings.ReplaceAll(result, `"exit_code":0`, `"exit_code":1`)} {
		if _, _, err = memoryParseStream([]byte(s)); err == nil {
			t.Fatal("invalid native stream accepted")
		}
	}
}
func TestMemoryHomeCasesAndDaily(t *testing.T) {
	t.Setenv("BASELINE_HOME", "")
	if memoryHome() == nil {
		t.Fatal("default accepted")
	}
	t.Setenv("BASELINE_HOME", filepath.Join(homeDir(), ".baseline"))
	if memoryHome() == nil {
		t.Fatal("production home accepted")
	}
	profile, cases, _, _ := memoryFixture(t)
	os.WriteFile(filepath.Join(baseDir(), "config.json"), []byte(`{"cloud_sync":true}`), 0600)
	if memoryHome() == nil {
		t.Fatal("stock config accepted")
	}
	when := time.Date(2026, 9, 17, 23, 49, 0, 0, time.UTC)
	id, due, err := memoryDaily(when, "08:50", "Asia/Tokyo", profile, cases)
	if err != nil || due {
		t.Fatal(id, due, err)
	}
	id2, due, err := memoryDaily(when.Add(time.Minute), "08:50", "Asia/Tokyo", profile, cases)
	if err != nil || !due || id != id2 {
		t.Fatal(id2, due, err)
	}
	os.Remove(filepath.Join(profile, "SOUL.md"))
	os.Symlink(cases, filepath.Join(profile, "SOUL.md"))
	if _, err := memorySnapshot(profile); err == nil {
		t.Fatal("symlink snapshot accepted")
	}
	c, _, _ := memoryReadCases(cases)
	c.Cases[0].Evidence = nil
	os.WriteFile(cases, memoryJSON(c), 0600)
	if _, _, err := memoryReadCases(cases); err == nil {
		t.Fatal("known case without evidence accepted")
	}
}
func TestMemoryAdapterUsesNativeBoundedCLI(t *testing.T) {
	// This asserts only the generated contract. The real installed runtime is pilot acceptance.
	for _, part := range []string{`"--max-turns","1"`, `"--format","stream-json"`, `"--run-budget",deadline`, `"--source","baseline-memory"`, `if selected or tools:`, `"provider":""`, `"background_review":{"enabled":False}`} {
		if !strings.Contains(memoryAdapter, part) {
			t.Fatal(part)
		}
	}
	if strings.Contains(memoryAdapter, "AIAgent") {
		t.Fatal("agent monkeypatch returned")
	}
	env := strings.Join(memoryEnvironment("/isolated"), "\n")
	if !strings.Contains(env, "HERMES_SAFE_MODE=1") {
		t.Fatal(env)
	}
	var v map[string]any
	if json.Unmarshal(memoryJSON(memoryUsage{}), &v) != nil {
		t.Fatal("usage serialization")
	}
}

func TestMemoryFailureAndGoldBoundary(t *testing.T) {
	profile, cases, python, source := memoryFixture(t)
	b, _ := os.ReadFile(cases)
	inside := filepath.Join(profile, "heldout.json")
	os.WriteFile(inside, b, 0600)
	if _, err := memoryRun(context.Background(), profile, inside, "leak", python, source, time.Second); err == nil {
		t.Fatal("gold inside source profile accepted")
	}
	os.WriteFile(python, []byte("#!/bin/sh\necho infrastructure-failure >&2\nexit 7\n"), 0700)
	r, err := memoryRun(context.Background(), profile, cases, "failure", python, source, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if r.Run.Status != "INCONCLUSIVE" || len(r.Run.Checks) != 0 || r.Usage.Total != nil {
		t.Fatal(r)
	}
	retained, err := os.ReadFile(filepath.Join(runArtifactDir(r.Run.ID), "stderr.txt"))
	if err != nil || !strings.Contains(string(retained), "infrastructure-failure") {
		t.Fatal(err)
	}
}

func TestMemoryNativeAuthRootProfileLayout(t *testing.T) {
	profile, cases, python, source := memoryFixture(t)
	// Native Hermes resolves auth globally only when HERMES_HOME is under HOME/.hermes.
	b, _ := os.ReadFile(python)
	b = bytes.Replace(b, []byte("assert os.environ['HERMES_SAFE_MODE']=='1'"), []byte("assert home.is_relative_to(pathlib.Path(os.environ['HOME'])/'.hermes'/'.baseline-evals')"), 1)
	os.WriteFile(python, b, 0700)
	r, err := memoryRun(context.Background(), profile, cases, "auth-layout", python, source, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if r.Run.Status != "FAIL" || r.Usage.SessionID != "fresh" {
		t.Fatalf("native-root isolated run did not reach recall: %+v", r.Run.Findings)
	}
	var result map[string]any
	json.Unmarshal(memoryJSON(r), &result)
	want := filepath.Join(homeDir(), ".hermes", ".baseline-evals", memoryHash([]byte(filepath.Clean(baseDir()))), r.Run.ID)
	if result["evaluation_profile"] != want {
		t.Fatalf("untraceable evaluation profile: %v", result["evaluation_profile"])
	}
	if _, err := os.Stat(filepath.Join(want, "state.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(want, "auth.json")); !os.IsNotExist(err) {
		t.Fatal("credentials copied/linked")
	}
}
func TestMemoryNativeFailureKeepsSessionAndUsage(t *testing.T) {
	raw := []byte(`{"type":"system","subtype":"init","model":"gpt-6-astra-900k","session_id":"native-init"}` + "\n" + `{"type":"result","session_id":"native-failure","exit_code":1,"error":"credentials unavailable","text":"","tokens":{"input":12,"output":0,"total":12}}`)
	_, u, err := memoryParseStream(raw)
	if err == nil || !u.Failed || u.Completed || u.SessionID != "native-failure" || u.Total == nil || *u.Total != 12 {
		t.Fatalf("lost native failure evidence: %+v, %v", u, err)
	}
}

func TestMemoryEvaluationAncestryRefusesAliasesAndGit(t *testing.T) {
	memoryFixture(t)
	native := filepath.Join(homeDir(), ".hermes")
	os.Mkdir(native, 0700)
	alias := filepath.Join(native, ".baseline-evals")
	os.Symlink(baseDir(), alias)
	if _, err := memoryEvaluationProfile("memory-test"); err == nil {
		t.Fatal("symlink evaluation parent accepted")
	}
	os.Remove(alias)
	os.Mkdir(alias, 0700)
	os.Mkdir(filepath.Join(native, ".git"), 0700)
	if _, err := memoryEvaluationProfile("memory-test"); err == nil {
		t.Fatal("Git ancestor evaluation accepted")
	}
}
