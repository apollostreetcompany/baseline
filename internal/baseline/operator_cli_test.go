package baseline

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunCLIStartsLongNonInteractiveRunsInBackground(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	t.Setenv("BASELINE_ASYNC_EXE", "/bin/echo")

	var stdout, stderr bytes.Buffer
	code := cmdRun(t.Context(), []string{"--packs", "enabled"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected background start success, code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{"Started Baseline", "in the background", "questions=55", "Poll: baseline report", "Accept only after review"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in output:\n%s", want, text)
		}
	}
}
