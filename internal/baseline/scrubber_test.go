package baseline

import (
	"strings"
	"testing"
)

func TestScrubTextRedactsSecretsPIIAndHomePaths(t *testing.T) {
	input := "token sk-test_abcdefghijklmnopqrstuvwxyz user future@example.com file /Users/future/.env"
	out, report := scrubText(input)
	for _, leaked := range []string{"sk-test_", "future@example.com", "/Users/future"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("scrubbed output leaked %q: %s", leaked, out)
		}
	}
	if report.SecretsFound == 0 {
		t.Fatalf("expected a synthetic secret to be counted: %+v", report)
	}
	if report.PIIFound == 0 {
		t.Fatalf("expected PII/path to be counted: %+v", report)
	}
	if report.CloudSafe {
		t.Fatalf("secret-bearing input should not be marked cloud-safe before redaction review")
	}
}
