package baseline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunOpenClawProbeCapturesSystemTimestampsAndFreshTokens(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "openclaw")
	sessionID := "baseline-run_probe-baseline-math"
	scriptBody := `#!/bin/sh
if [ "$1" = "agent" ]; then
  echo '{"response":"4"}'
  exit 0
fi
if [ "$1" = "sessions" ]; then
  cat <<'JSON'
[{"session_id":"` + sessionID + `","inputTokens":7,"outputTokens":3,"totalTokens":10,"contextTokens":99,"model":"gpt-test","modelProvider":"openai","totalTokensFresh":true}]
JSON
  exit 0
fi
echo "unexpected command: $*" >&2
exit 1
`
	if err := os.WriteFile(script, []byte(scriptBody), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := runOpenClawProbe(context.Background(), script, "run_probe", Question{
		PackID:        "baseline",
		ID:            "math",
		Prompt:        "Answer only the number: 2 + 2.",
		ExpectedFacts: []string{"4"},
		Dimension:     "basic_reasoning",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.Output) != "4" {
		t.Fatalf("expected parsed response text, got %q", result.Output)
	}
	if result.SystemSendAt.IsZero() || result.BaselineReceivedAt.IsZero() {
		t.Fatalf("expected send/receive timestamps: %+v", result.ProbeMessage)
	}
	if result.BaselineReceivedAt.Before(result.SystemSendAt) || result.DurationMS < 0 {
		t.Fatalf("invalid timing: %+v", result.ProbeMessage)
	}
	if result.SessionID != sessionID || result.TokenStatus != "fresh" {
		t.Fatalf("expected correlated fresh session, got %+v", result.ProbeMessage)
	}
	if result.TotalTokens == nil || *result.TotalTokens != 10 {
		t.Fatalf("expected total tokens from sessions metadata, got %+v", result.ProbeMessage)
	}
	if result.Model != "gpt-test" || result.ModelProvider != "openai" {
		t.Fatalf("expected model metadata, got %+v", result.ProbeMessage)
	}
}

func TestRunOpenClawProbeDropsStaleTokenCounts(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "openclaw")
	sessionID := "baseline-run_stale-baseline-math"
	scriptBody := `#!/bin/sh
if [ "$1" = "agent" ]; then
  echo '{"response":"4"}'
  exit 0
fi
if [ "$1" = "sessions" ]; then
  cat <<'JSON'
[{"session_id":"` + sessionID + `","inputTokens":7,"outputTokens":3,"totalTokens":10,"totalTokensFresh":false}]
JSON
  exit 0
fi
exit 1
`
	if err := os.WriteFile(script, []byte(scriptBody), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := runOpenClawProbe(context.Background(), script, "run_stale", Question{PackID: "baseline", ID: "math", Prompt: "Answer only 4."})
	if err != nil {
		t.Fatal(err)
	}
	if result.TokenStatus != "stale" {
		t.Fatalf("expected stale token status, got %+v", result.ProbeMessage)
	}
	if result.TotalTokens != nil || result.InputTokens != nil || result.OutputTokens != nil {
		t.Fatalf("stale token counts must not be retained, got %+v", result.ProbeMessage)
	}
}

func TestOpenClawProbeSessionIDStaysWithinProviderLimit(t *testing.T) {
	q := Question{PackID: "personality_identity", ID: "broad_idea_warning"}
	sessionID := openClawProbeSessionID("run_diitfaegybzc", q)
	if len(sessionID) > 64 {
		t.Fatalf("session id must stay within provider cache key limit, got %d: %s", len(sessionID), sessionID)
	}
	if !strings.HasPrefix(sessionID, "baseline-run_diitfaegybzc-") {
		t.Fatalf("expected readable baseline prefix, got %s", sessionID)
	}
	if sessionID != openClawProbeSessionID("run_diitfaegybzc", q) {
		t.Fatalf("session id must be deterministic")
	}
}

func TestRunOpenClawProbeHonorsOpenClawEnvOverrides(t *testing.T) {
	t.Setenv("BASELINE_OPENCLAW_MODEL", "openai/gpt-5.5")
	t.Setenv("BASELINE_OPENCLAW_THINKING", "low")
	t.Setenv("BASELINE_OPENCLAW_AGENT_TIMEOUT_SECONDS", "45")
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	t.Setenv("BASELINE_TEST_ARGS_FILE", argsFile)
	script := filepath.Join(dir, "openclaw")
	scriptBody := `#!/bin/sh
if [ "$1" = "agent" ]; then
  printf '%s\n' "$@" > "$BASELINE_TEST_ARGS_FILE"
  echo '{"response":"baseline"}'
  exit 0
fi
if [ "$1" = "sessions" ]; then
  echo '[]'
  exit 0
fi
echo "unexpected command: $*" >&2
exit 1
`
	if err := os.WriteFile(script, []byte(scriptBody), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := runOpenClawProbe(context.Background(), script, "run_env", Question{
		PackID: "baseline",
		ID:     "variance_1",
		Prompt: "Answer only the word: baseline.",
	})
	if err != nil {
		t.Fatal(err)
	}
	argsBytes, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args := "\n" + string(argsBytes)
	for _, want := range []string{
		"\n--model\nopenai/gpt-5.5\n",
		"\n--thinking\nlow\n",
		"\n--timeout\n45\n",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("expected args to contain %q, got:\n%s", want, args)
		}
	}
}

func TestTokenFreshnessUsesInWindowTimestamps(t *testing.T) {
	sentAt := time.Now().UTC()
	receivedAt := sentAt.Add(2 * time.Second)
	status := tokenFreshness(map[string]any{
		"updatedAt": sentAt.Add(500 * time.Millisecond).Format(time.RFC3339Nano),
	}, sentAt, receivedAt)
	if status != "fresh" {
		t.Fatalf("expected in-window timestamp to be fresh, got %s", status)
	}
}

func TestTokenFreshnessRejectsOutOfWindowTimestamps(t *testing.T) {
	sentAt := time.Now().UTC()
	receivedAt := sentAt.Add(2 * time.Second)
	status := tokenFreshness(map[string]any{
		"updatedAt": sentAt.Add(-1 * time.Minute).Format(time.RFC3339Nano),
	}, sentAt, receivedAt)
	if status != "stale" {
		t.Fatalf("expected old timestamp to be stale, got %s", status)
	}
}
