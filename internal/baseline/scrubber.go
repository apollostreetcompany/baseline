package baseline

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

type ScrubReport struct {
	SecretsFound int      `json:"secrets_found"`
	PIIFound     int      `json:"pii_found"`
	Replacements []string `json:"replacements"`
	CloudSafe    bool     `json:"cloud_safe"`
}

var scrubPatterns = []struct {
	name string
	re   *regexp.Regexp
	repl string
}{
	{"openai_key", regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`), "[REDACTED_OPENAI_KEY]"},
	{"anthropic_key", regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`), "[REDACTED_ANTHROPIC_KEY]"},
	{"stripe_secret", regexp.MustCompile(`sk_(test|live)_[A-Za-z0-9]{20,}`), "[REDACTED_STRIPE_KEY]"},
	{"jwt", regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), "[REDACTED_JWT]"},
	{"private_key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`), "[REDACTED_PRIVATE_KEY]"},
	{"email", regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`), "[REDACTED_EMAIL]"},
	{"home_path", regexp.MustCompile(`/Users/[A-Za-z0-9._\-]+`), "/Users/[REDACTED_USER]"},
}

func scrubText(input string) (string, ScrubReport) {
	out := input
	report := ScrubReport{CloudSafe: true}
	for _, p := range scrubPatterns {
		matches := p.re.FindAllString(out, -1)
		if len(matches) == 0 {
			continue
		}
		out = p.re.ReplaceAllString(out, p.repl)
		report.Replacements = append(report.Replacements, p.name)
		if strings.Contains(p.name, "key") || p.name == "jwt" {
			report.SecretsFound += len(matches)
		} else {
			report.PIIFound += len(matches)
		}
	}
	report.CloudSafe = report.SecretsFound == 0
	return out, report
}

func hashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func displayHash(value string) string {
	h := hashValue(value)
	if len(h) < 12 {
		return h
	}
	return h[:12]
}
