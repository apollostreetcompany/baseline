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

	args := []string{"chat", "-Q", "-q", q.Prompt, "--source", "baseline"}
	if target.ModelPolicy == "pinned" && strings.TrimSpace(target.PinnedModel) != "" {
		args = append([]string{"chat", "-Q", "-m", strings.TrimSpace(target.PinnedModel), "-q", q.Prompt, "--source", "baseline"})
	}

	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(targetTimeoutSeconds(target))*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, path, args...)
	if workspace != "" {
		cmd.Dir = workspace
	}
	cmd.Env = append(os.Environ(),
		"BASELINE_RUN_ID="+runID,
		"BASELINE_PROBE_ID="+q.ID,
		"BASELINE_PACK_ID="+q.PackID,
	)

	sendAt := time.Now().UTC()
	out, err := cmd.CombinedOutput()
	receivedAt := time.Now().UTC()
	msg := ProbeMessage{
		RunID:              runID,
		PackID:             q.PackID,
		ProbeID:            q.ID,
		SessionID:          "",
		SystemSendAt:       sendAt,
		BaselineReceivedAt: receivedAt,
		DurationMS:         receivedAt.Sub(sendAt).Milliseconds(),
		TokenStatus:        "unavailable",
		TokenSource:        "hermes cli",
	}
	output := string(out)
	if cmdCtx.Err() == context.DeadlineExceeded {
		return output, msg, fmt.Errorf("hermes timed out")
	}
	if err != nil {
		return output, msg, fmt.Errorf("hermes failed: %w: %s", err, strings.TrimSpace(output))
	}
	return output, msg, nil
}
