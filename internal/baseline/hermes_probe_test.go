package baseline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHermesProbeUsesHermesChatQuietQuery(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	envFile := filepath.Join(dir, "env.txt")
	hermes := filepath.Join(dir, "hermes")
	script := `#!/bin/sh
printf '%s\n' "$@" > "` + argsFile + `"
printenv | grep '^BASELINE_\|^OTEL_RESOURCE_ATTRIBUTES=' | sort > "` + envFile + `"
prompt=''
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-q" ]; then
    shift
    prompt="$1"
    break
  fi
  shift
done
printf 'Hermes answer: %s\n' "$prompt"
printf 'BASELINE_HERMES_SESSION_ID: session_test_123\n'
`
	if err := os.WriteFile(hermes, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=baseline-test")

	q := Question{PackID: "baseline", ID: "math", Prompt: "Say baseline ok."}
	result, err := runHermesProbeWithTarget(context.Background(), "run_test", q, BaselineTarget{Runtime: "hermes", ModelPolicy: "follow_current", TimeoutSeconds: 30}, dir)
	if err != nil {
		t.Fatalf("runHermesProbeWithTarget returned error: %v", err)
	}
	if !strings.Contains(result.Output, "Hermes answer: Say baseline ok.") {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if strings.Contains(result.Output, baselineSessionMarker) {
		t.Fatalf("session marker should be stripped from scored output: %q", result.Output)
	}
	if result.SessionID != "session_test_123" {
		t.Fatalf("expected parsed Hermes session id, got %+v", result.ProbeMessage)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(args))
	if !strings.Contains(got, "chat\n-Q\n--pass-session-id\n-q\n") || !strings.Contains(got, "\n--source\nbaseline") {
		t.Fatalf("args missing Hermes observability flags, got:\n%s", got)
	}
	if !strings.Contains(got, "Baseline harness observability instruction") {
		t.Fatalf("prompt should ask Hermes to disclose its session id, got:\n%s", got)
	}
	env, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	envText := string(env)
	for _, want := range []string{
		"BASELINE_RUN_ID=run_test",
		"BASELINE_PACK_ID=baseline",
		"BASELINE_PROBE_ID=math",
		"BASELINE_TIMEOUT_SECONDS=30",
		"BASELINE_EVAL_MODE=1",
		"BASELINE_DEADLINE_AT=",
		"OTEL_RESOURCE_ATTRIBUTES=service.name=baseline-test,baseline.run_id=run_test,baseline.pack_id=baseline,baseline.probe_id=math",
	} {
		if !strings.Contains(envText, want) {
			t.Fatalf("expected env %q in:\n%s", want, envText)
		}
	}
	if result.TokenStatus != "unavailable" || result.TokenSource != "hermes cli" {
		t.Fatalf("unexpected token metadata: %+v", result.ProbeMessage)
	}
}

func TestValidateConfigAcceptsHermesRuntime(t *testing.T) {
	cfg := defaultConfig()
	cfg.Target.Runtime = "hermes"
	cfg.Target.Entity = "agent:hermes"
	cfg.Target.TimeoutSeconds = 900
	for _, issue := range validateConfig(cfg) {
		if strings.Contains(issue, "target.runtime") {
			t.Fatalf("hermes runtime should validate, got issue: %s", issue)
		}
	}
}
