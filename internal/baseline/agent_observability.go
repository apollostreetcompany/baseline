package baseline

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baselineSessionMarker = "BASELINE_HERMES_SESSION_ID:"

var baselineSessionLineRE = regexp.MustCompile(`(?m)^\s*BASELINE_HERMES_SESSION_ID:\s*([^\s]+)\s*$`)

func probeDeadline(sendAt time.Time, timeoutSeconds int) time.Time {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 90
	}
	return sendAt.Add(time.Duration(timeoutSeconds) * time.Second).UTC()
}

func probeDeadlineEnv(runID string, q Question, timeoutSeconds int, sendAt time.Time) []string {
	deadline := probeDeadline(sendAt, timeoutSeconds)
	return []string{
		"BASELINE_RUN_ID=" + runID,
		"BASELINE_PROBE_ID=" + q.ID,
		"BASELINE_PACK_ID=" + q.PackID,
		"BASELINE_TIMEOUT_SECONDS=" + strconv.Itoa(timeoutSeconds),
		"BASELINE_DEADLINE_AT=" + deadline.Format(time.RFC3339Nano),
		"BASELINE_EVAL_MODE=1",
		"OTEL_RESOURCE_ATTRIBUTES=" + mergeOTELResourceAttributes(map[string]string{
			"baseline.run_id":   runID,
			"baseline.pack_id":  q.PackID,
			"baseline.probe_id": q.ID,
		}),
	}
}

func baselinePromptForProbe(prompt string) string {
	prompt = strings.TrimRight(prompt, " \t\r\n")
	return prompt + `

Baseline harness observability instruction:
- Answer the user-facing probe normally.
- If your runtime exposes the current Hermes session id, append exactly one final machine-readable line in this form:
` + baselineSessionMarker + ` <session_id>
- If the session id is unavailable, omit that line. Do not invent one.`
}

func extractBaselineSessionID(output string) (cleanOutput string, sessionID string) {
	sessionID = ""
	matches := baselineSessionLineRE.FindAllStringSubmatch(output, -1)
	if len(matches) > 0 {
		sessionID = strings.TrimSpace(matches[len(matches)-1][1])
	}
	cleanOutput = strings.TrimSpace(baselineSessionLineRE.ReplaceAllString(output, ""))
	if cleanOutput != "" && strings.HasSuffix(output, "\n") {
		cleanOutput += "\n"
	}
	return cleanOutput, sessionID
}

func mergeOTELResourceAttributes(attrs map[string]string) string {
	parts := make([]string, 0, len(attrs)+1)
	if existing := strings.TrimSpace(getenvOTELResourceAttributes()); existing != "" {
		parts = append(parts, existing)
	}
	for _, key := range []string{"baseline.run_id", "baseline.pack_id", "baseline.probe_id"} {
		if value := strings.TrimSpace(attrs[key]); value != "" {
			parts = append(parts, key+"="+otelAttrValue(value))
		}
	}
	return strings.Join(parts, ",")
}

func otelAttrValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, "=", "\\=")
	return value
}

var getenvOTELResourceAttributes = func() string { return os.Getenv("OTEL_RESOURCE_ATTRIBUTES") }
