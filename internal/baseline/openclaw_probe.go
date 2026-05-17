package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func runOpenClawProbe(ctx context.Context, openclawPath, runID string, q Question) (AgentProbeResult, error) {
	return runOpenClawProbeWithTarget(ctx, openclawPath, runID, q, defaultConfig().Target)
}

func runOpenClawProbeWithTarget(ctx context.Context, openclawPath, runID string, q Question, target BaselineTarget) (AgentProbeResult, error) {
	sessionID := openClawProbeSessionID(runID, q)
	agentTimeout := openClawAgentTimeoutSeconds(target)
	args := []string{"agent", "--json", "--session-id", sessionID, "--message", q.Prompt, "--timeout", strconv.Itoa(agentTimeout)}
	if agentName := targetAgentName(target); agentName != "" {
		args = append(args, "--agent", agentName)
	}
	envModel := strings.TrimSpace(os.Getenv("BASELINE_OPENCLAW_MODEL"))
	if target.ModelPolicy == "pinned" && strings.TrimSpace(target.PinnedModel) != "" && envModel == "" {
		args = append(args, "--model", strings.TrimSpace(target.PinnedModel))
	}
	if thinking := strings.TrimSpace(target.Thinking); thinking != "" {
		args = append(args, "--thinking", thinking)
	}
	if envModel != "" {
		args = append(args, "--model", envModel)
	}
	if thinking := strings.TrimSpace(os.Getenv("BASELINE_OPENCLAW_THINKING")); thinking != "" {
		args = append(args, "--thinking", thinking)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(agentTimeout+30)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, openclawPath, args...)
	stdoutFile, stdoutErr := os.CreateTemp("", "baseline-openclaw-stdout-*")
	if stdoutErr != nil {
		return AgentProbeResult{}, stdoutErr
	}
	defer os.Remove(stdoutFile.Name())
	defer stdoutFile.Close()
	stderrFile, stderrErr := os.CreateTemp("", "baseline-openclaw-stderr-*")
	if stderrErr != nil {
		return AgentProbeResult{}, stderrErr
	}
	defer os.Remove(stderrFile.Name())
	defer stderrFile.Close()
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	systemSendAt := time.Now().UTC()
	err := cmd.Run()
	baselineReceivedAt := time.Now().UTC()
	_ = stdoutFile.Sync()
	_ = stderrFile.Sync()
	out, _ := os.ReadFile(stdoutFile.Name())
	errOut, _ := os.ReadFile(stderrFile.Name())
	output := extractAgentText(out)
	meta := OpenClawTokenMetadata{TokenStatus: "unavailable"}
	if ctx.Err() == nil {
		meta = readOpenClawTokenMetadata(ctx, openclawPath, sessionID, systemSendAt, baselineReceivedAt)
	}
	msg := ProbeMessage{
		RunID:              runID,
		PackID:             q.PackID,
		ProbeID:            q.ID,
		SessionID:          sessionID,
		SystemSendAt:       systemSendAt,
		BaselineReceivedAt: baselineReceivedAt,
		DurationMS:         baselineReceivedAt.Sub(systemSendAt).Milliseconds(),
		TokenStatus:        meta.TokenStatus,
		TokenSource:        meta.TokenSource,
		InputTokens:        meta.InputTokens,
		OutputTokens:       meta.OutputTokens,
		TotalTokens:        meta.TotalTokens,
		ContextTokens:      meta.ContextTokens,
		Model:              meta.Model,
		ModelProvider:      meta.ModelProvider,
	}
	if ctx.Err() == context.DeadlineExceeded {
		return AgentProbeResult{Output: output, ProbeMessage: msg}, fmt.Errorf("openclaw agent timed out")
	}
	if err != nil {
		detail := strings.TrimSpace(string(errOut))
		if detail == "" {
			detail = strings.TrimSpace(string(out))
		}
		return AgentProbeResult{Output: output, ProbeMessage: msg}, fmt.Errorf("openclaw agent failed: %w: %s", err, detail)
	}
	return AgentProbeResult{Output: output, ProbeMessage: msg}, nil
}

func openClawProbeSessionID(runID string, q Question) string {
	raw := "baseline-" + runID + "-" + q.PackID + "-" + q.ID
	const maxLen = 64
	if len(raw) <= maxLen {
		return raw
	}
	suffix := displayHash(raw)
	prefix := "baseline-" + runID + "-"
	maxSlugLen := maxLen - len(prefix) - len(suffix) - 1
	if maxSlugLen < 8 {
		return raw[:maxLen-len(suffix)-1] + "-" + suffix
	}
	slug := compactSessionSlug(q.PackID + "-" + q.ID)
	if len(slug) > maxSlugLen {
		slug = strings.Trim(slug[:maxSlugLen], "-_")
	}
	if slug == "" {
		slug = "probe"
	}
	return prefix + slug + "-" + suffix
}

func compactSessionSlug(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSep = false
		case r == '_' || r == '-':
			if !lastSep {
				b.WriteByte('-')
				lastSep = true
			}
		default:
			if !lastSep {
				b.WriteByte('-')
				lastSep = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func openClawAgentTimeoutSeconds(target BaselineTarget) int {
	fallback := targetTimeoutSeconds(target)
	raw := strings.TrimSpace(os.Getenv("BASELINE_OPENCLAW_AGENT_TIMEOUT_SECONDS"))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < 5 {
		return 5
	}
	if value > 600 {
		return 600
	}
	return value
}

func extractAgentText(out []byte) string {
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return ""
	}
	var value any
	if err := json.Unmarshal(out, &value); err != nil {
		lines := strings.Split(raw, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			var event any
			if json.Unmarshal([]byte(line), &event) == nil {
				if text := firstString(event, "response", "answer", "content", "text", "message", "output"); text != "" {
					return text
				}
			}
		}
		return raw
	}
	if text := firstString(value, "response", "answer", "content", "text", "message", "output"); text != "" {
		return text
	}
	return raw
}

func readOpenClawTokenMetadata(ctx context.Context, openclawPath, sessionID string, sentAt, receivedAt time.Time) OpenClawTokenMetadata {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, openclawPath, "sessions", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil || ctx.Err() != nil {
		return OpenClawTokenMetadata{TokenStatus: "unavailable"}
	}
	var value any
	if err := json.Unmarshal(out, &value); err != nil {
		return OpenClawTokenMetadata{TokenStatus: "unavailable"}
	}
	session := findSessionObject(value, sessionID)
	if session == nil {
		return OpenClawTokenMetadata{TokenStatus: "unavailable"}
	}
	meta := OpenClawTokenMetadata{
		TokenStatus:   tokenFreshness(session, sentAt, receivedAt),
		TokenSource:   "openclaw sessions --json",
		InputTokens:   firstIntPtr(session, "inputTokens", "input_tokens", "promptTokens", "prompt_tokens"),
		OutputTokens:  firstIntPtr(session, "outputTokens", "output_tokens", "completionTokens", "completion_tokens"),
		TotalTokens:   firstIntPtr(session, "totalTokens", "total_tokens"),
		ContextTokens: firstIntPtr(session, "contextTokens", "context_tokens"),
		Model:         firstString(session, "model", "modelName", "model_name"),
		ModelProvider: firstString(session, "modelProvider", "model_provider", "provider"),
	}
	if meta.InputTokens == nil && meta.OutputTokens == nil && meta.TotalTokens == nil && meta.ContextTokens == nil {
		meta.TokenStatus = "unavailable"
	}
	if meta.TokenStatus != "fresh" {
		meta.InputTokens = nil
		meta.OutputTokens = nil
		meta.TotalTokens = nil
		meta.ContextTokens = nil
	}
	return meta
}

func findSessionObject(value any, sessionID string) map[string]any {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if found := findSessionObject(item, sessionID); found != nil {
				return found
			}
		}
	case map[string]any:
		for _, key := range []string{"session_id", "sessionId", "id"} {
			if fmt.Sprint(v[key]) == sessionID {
				return v
			}
		}
		for _, item := range v {
			if found := findSessionObject(item, sessionID); found != nil {
				return found
			}
		}
	}
	return nil
}

func tokenFreshness(session map[string]any, sentAt, receivedAt time.Time) string {
	if fresh, ok := boolField(session, "totalTokensFresh", "total_tokens_fresh", "tokensFresh", "tokens_fresh"); ok {
		if fresh {
			return "fresh"
		}
		return "stale"
	}
	if ts := firstString(session, "updatedAt", "updated_at", "lastUpdatedAt", "last_updated_at", "createdAt", "created_at"); ts != "" {
		parsed, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return "stale"
		}
		if parsed.Before(sentAt.Add(-5*time.Second)) || parsed.After(receivedAt.Add(30*time.Second)) {
			return "stale"
		}
		return "fresh"
	}
	return "stale"
}

func boolField(value map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		switch v := value[key].(type) {
		case bool:
			return v, true
		case string:
			if parsed, err := strconv.ParseBool(v); err == nil {
				return parsed, true
			}
		}
	}
	return false, false
}

func firstString(value any, keys ...string) string {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if s, ok := v[key].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
			if nested, ok := v[key].(map[string]any); ok {
				if s := firstString(nested, keys...); s != "" {
					return s
				}
			}
		}
		for _, item := range v {
			if s := firstString(item, keys...); s != "" {
				return s
			}
		}
	case []any:
		for _, item := range v {
			if s := firstString(item, keys...); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstIntPtr(value any, keys ...string) *int {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if n, ok := intFromAny(v[key]); ok {
				return &n
			}
		}
		for _, item := range v {
			if n := firstIntPtr(item, keys...); n != nil {
				return n
			}
		}
	case []any:
		for _, item := range v {
			if n := firstIntPtr(item, keys...); n != nil {
				return n
			}
		}
	}
	return nil
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case json.Number:
		i, err := strconv.Atoi(v.String())
		return i, err == nil
	case string:
		i, err := strconv.Atoi(v)
		return i, err == nil
	default:
		return 0, false
	}
}
