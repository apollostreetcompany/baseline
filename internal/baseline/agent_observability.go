package baseline

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baselineSessionMarker = "BASELINE_HERMES_SESSION_ID:"
const baselineMetadataMarker = "BASELINE_AGENT_METADATA_JSON:"

var baselineSessionLineRE = regexp.MustCompile(`(?m)^\s*BASELINE_HERMES_SESSION_ID:\s*([^\s]+)\s*$`)
var baselineMetadataLineRE = regexp.MustCompile(`(?m)^\s*BASELINE_AGENT_METADATA_JSON:\s*(\{.*\})\s*$`)

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
- If your runtime exposes model/token metadata, append exactly one final machine-readable JSON line prefixed with 'BASELINE_AGENT_METADATA_JSON:' followed by {"model":"...","model_provider":"...","input_tokens":0,"output_tokens":0,"total_tokens":0,"context_tokens":0}.
- Use OpenTelemetry resource attributes from OTEL_RESOURCE_ATTRIBUTES if your runtime records spans; Baseline sets baseline.run_id, baseline.pack_id, and baseline.probe_id for correlation.
- If session id or metadata is unavailable, omit that line. Do not invent values.`
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

func extractBaselineAgentMetadata(output string, msg *ProbeMessage) string {
	matches := baselineMetadataLineRE.FindAllStringSubmatch(output, -1)
	if len(matches) > 0 {
		var meta struct {
			Model         string `json:"model"`
			ModelProvider string `json:"model_provider"`
			InputTokens   *int   `json:"input_tokens"`
			OutputTokens  *int   `json:"output_tokens"`
			TotalTokens   *int   `json:"total_tokens"`
			ContextTokens *int   `json:"context_tokens"`
		}
		if err := json.Unmarshal([]byte(matches[len(matches)-1][1]), &meta); err == nil {
			msg.Model = meta.Model
			msg.ModelProvider = meta.ModelProvider
			msg.InputTokens = meta.InputTokens
			msg.OutputTokens = meta.OutputTokens
			msg.TotalTokens = meta.TotalTokens
			msg.ContextTokens = meta.ContextTokens
			msg.TokenStatus = "fresh"
			msg.TokenSource = "agent metadata"
		}
	}
	cleanOutput := strings.TrimSpace(baselineMetadataLineRE.ReplaceAllString(output, ""))
	if cleanOutput != "" && strings.HasSuffix(output, "\n") {
		cleanOutput += "\n"
	}
	return cleanOutput
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
