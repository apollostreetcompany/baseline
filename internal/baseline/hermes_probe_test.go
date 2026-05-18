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
	hermes := filepath.Join(dir, "hermes")
	script := `#!/bin/sh
printf '%s\n' "$@" > "` + argsFile + `"
printf 'Hermes answer: %s\n' "$4"
`
	if err := os.WriteFile(hermes, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	q := Question{PackID: "baseline", ID: "math", Prompt: "Say baseline ok."}
	result, err := runHermesProbeWithTarget(context.Background(), "run_test", q, BaselineTarget{Runtime: "hermes", ModelPolicy: "follow_current", TimeoutSeconds: 30}, dir)
	if err != nil {
		t.Fatalf("runHermesProbeWithTarget returned error: %v", err)
	}
	if !strings.Contains(result.Output, "Hermes answer: Say baseline ok.") {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(args))
	want := "chat\n-Q\n-q\nSay baseline ok.\n--source\nbaseline"
	if got != want {
		t.Fatalf("args mismatch\nwant:\n%s\ngot:\n%s", want, got)
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
