package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type RunLifecycleStatus struct {
	RunID       string       `json:"run_id"`
	Mode        string       `json:"mode"`
	State       string       `json:"state"`
	PID         int          `json:"pid,omitempty"`
	Packs       string       `json:"packs,omitempty"`
	Questions   int          `json:"questions,omitempty"`
	StartedAt   time.Time    `json:"started_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Status      string       `json:"status,omitempty"`
	HealthScore int          `json:"health_score,omitempty"`
	Error       string       `json:"error,omitempty"`
	Artifacts   RunArtifacts `json:"artifacts,omitempty"`
	StdoutPath  string       `json:"stdout_path,omitempty"`
	StderrPath  string       `json:"stderr_path,omitempty"`
	NextActions []string     `json:"next_actions,omitempty"`
}

func runLifecycleDir() string {
	return filepath.Join(baseDir(), "runs")
}

func runLifecyclePath(runID string) string {
	return filepath.Join(runLifecycleDir(), runID+".json")
}

func runLifecycleLogPaths(runID string) (string, string) {
	return filepath.Join(runLifecycleDir(), runID+".stdout.log"), filepath.Join(runLifecycleDir(), runID+".stderr.log")
}

func writeRunLifecycleStatus(status RunLifecycleStatus) error {
	if err := os.MkdirAll(runLifecycleDir(), 0o700); err != nil {
		return err
	}
	status.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(runLifecyclePath(status.RunID), b, 0o600)
}

func readRunLifecycleStatus(runID string) (RunLifecycleStatus, error) {
	var status RunLifecycleStatus
	b, err := os.ReadFile(runLifecyclePath(runID))
	if err != nil {
		return status, err
	}
	if err := json.Unmarshal(b, &status); err != nil {
		return status, err
	}
	return refreshRunLifecycleStatus(status), nil
}

func startedRunStatus(runID, mode string) RunLifecycleStatus {
	stdoutPath, stderrPath := runLifecycleLogPaths(runID)
	now := time.Now().UTC()
	return RunLifecycleStatus{
		RunID:       runID,
		Mode:        mode,
		State:       "running",
		PID:         os.Getpid(),
		StartedAt:   now,
		UpdatedAt:   now,
		Artifacts:   runArtifactPaths(runID),
		StdoutPath:  stdoutPath,
		StderrPath:  stderrPath,
		NextActions: []string{"Wait for the run to complete", "Then run baseline report " + runID},
	}
}

func plannedRunStatus(runID, mode, packs string, questions int) RunLifecycleStatus {
	status := startedRunStatus(runID, mode)
	status.Packs = packs
	status.Questions = questions
	if questions > 0 {
		status.NextActions = []string{
			fmt.Sprintf("Wait for %d %s questions to complete", questions, packs),
			"Then run baseline report " + runID,
		}
	}
	return status
}

func completedRunStatus(run Run) RunLifecycleStatus {
	stdoutPath, stderrPath := runLifecycleLogPaths(run.ID)
	return RunLifecycleStatus{
		RunID:       run.ID,
		Mode:        run.Mode,
		State:       "completed",
		PID:         os.Getpid(),
		StartedAt:   run.StartedAt.UTC(),
		Status:      run.Status,
		HealthScore: run.HealthScore,
		Artifacts:   run.Artifacts,
		StdoutPath:  stdoutPath,
		StderrPath:  stderrPath,
		NextActions: []string{
			"Review baseline report " + run.ID,
			"Accept only after operator review: baseline accept " + run.ID + " --confirm \"accept " + run.ID + "\"",
		},
	}
}

func failedRunStatus(runID, mode string, err error) RunLifecycleStatus {
	status := startedRunStatus(runID, mode)
	status.State = "failed"
	status.Error = err.Error()
	status.NextActions = []string{"Run baseline doctor", "Fix the reported runtime/config issue", "Rerun baseline run"}
	return status
}

func refreshRunLifecycleStatus(status RunLifecycleStatus) RunLifecycleStatus {
	if status.State != "running" || status.PID <= 0 || processAlive(status.PID) {
		return status
	}
	status.State = "failed"
	status.Error = fmt.Sprintf("run process pid %d is no longer running and no result row was written", status.PID)
	status.NextActions = []string{
		"Read stdout_path and stderr_path for the missing child output",
		"Run baseline doctor",
		"Rerun baseline run after fixing the reported issue",
	}
	_ = writeRunLifecycleStatus(status)
	return status
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
