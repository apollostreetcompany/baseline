package baseline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func runHermesProbeWithTarget(ctx context.Context, runID string, q Question, target BaselineTarget, workspace string) (AgentProbeResult, error) {
	output, msg, err := runHermesPromptMeasured(ctx, runID, q, target, workspace)
	return AgentProbeResult{Output: output, ProbeMessage: msg}, err
}

func runHermesPrompt(ctx context.Context, prompt string, target BaselineTarget, workspace string) (string, error) {
	q := Question{Prompt: prompt}
	output, _, err := runHermesPromptMeasured(ctx, "", q, target, workspace)
	return output, err
}

func runHermesPromptMeasured(ctx context.Context, runID string, q Question, target BaselineTarget, workspace string) (string, ProbeMessage, error) {
	path, err := exec.LookPath("hermes")
	if err != nil {
		return "", ProbeMessage{}, err
	}

	prompt := baselinePromptForProbe(q.Prompt)
	args := []string{"chat", "-Q", "--pass-session-id", "-q", prompt, "--source", "baseline"}
	if target.ModelPolicy == "pinned" && strings.TrimSpace(target.PinnedModel) != "" {
		args = append([]string{"chat", "-Q", "--pass-session-id", "-m", strings.TrimSpace(target.PinnedModel), "-q", prompt, "--source", "baseline"})
	}

	timeoutSeconds := targetTimeoutSeconds(target)
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	sendAt := time.Now().UTC()
	cmd := exec.CommandContext(cmdCtx, path, args...)
	if workspace != "" {
		cmd.Dir = workspace
	}
	cmd.Env = append(os.Environ(), probeDeadlineEnv(runID, q, timeoutSeconds, sendAt)...)

	out, err := cmd.CombinedOutput()
	receivedAt := time.Now().UTC()
	output, sessionID := extractBaselineSessionID(string(out))
	msg := ProbeMessage{
		RunID:              runID,
		PackID:             q.PackID,
		ProbeID:            q.ID,
		SessionID:          sessionID,
		SystemSendAt:       sendAt,
		BaselineReceivedAt: receivedAt,
		DurationMS:         receivedAt.Sub(sendAt).Milliseconds(),
		TokenStatus:        "unavailable",
		TokenSource:        "hermes cli",
	}
	if cmdCtx.Err() == context.DeadlineExceeded {
		return output, msg, fmt.Errorf("hermes timed out")
	}
	if err != nil {
		return output, msg, fmt.Errorf("hermes failed: %w: %s", err, strings.TrimSpace(output))
	}
	return output, msg, nil
}
