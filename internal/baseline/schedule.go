package baseline

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const scheduleLabel = "ai.baseline.daily"

type ScheduleStatus struct {
	Installed     bool      `json:"installed"`
	Label         string    `json:"label"`
	PlistPath     string    `json:"plist_path"`
	Hour          int       `json:"hour"`
	Minute        int       `json:"minute"`
	WorkspacePath string    `json:"workspace_path,omitempty"`
	NextRun       time.Time `json:"next_run,omitempty"`
	Message       string    `json:"message"`
}

type ScheduleRunResult struct {
	Action      string `json:"action"`
	RunID       string `json:"run_id"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	HealthScore int    `json:"health_score"`
	CloudSynced bool   `json:"cloud_synced"`
	SyncSynced  int    `json:"sync_synced"`
	SyncFailed  int    `json:"sync_failed"`
	Workspace   string `json:"workspace,omitempty"`
	ReportPath  string `json:"report_path,omitempty"`
}

func installSchedule(exe, at string) (ScheduleStatus, error) {
	if runtime.GOOS != "darwin" {
		return ScheduleStatus{}, fmt.Errorf("automatic install currently supports launchd on macOS; use cron/systemd with: %s schedule run", exe)
	}
	hour, minute, err := parseScheduleTime(at)
	if err != nil {
		return ScheduleStatus{}, err
	}
	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return ScheduleStatus{}, err
		}
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return ScheduleStatus{}, err
	}
	cfg, err := loadConfig()
	if err != nil {
		return ScheduleStatus{}, err
	}
	workspace := runtimeWorkspace(cfg)
	if workspace == string(os.PathSeparator) {
		return ScheduleStatus{}, fmt.Errorf("refusing to install schedule with workspace /; run from the intended workspace or set baseline config set workspace_path <path>")
	}
	if cfg.WorkspacePath == "" {
		cfg.WorkspacePath = workspace
		if err := saveConfig(cfg); err != nil {
			return ScheduleStatus{}, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(launchdPlistPath()), 0o700); err != nil {
		return ScheduleStatus{}, err
	}
	logPath := filepath.Join(baseDir(), "logs", "schedule.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return ScheduleStatus{}, err
	}
	if err := atomicWrite(launchdPlistPath(), []byte(launchdPlist(exe, fmt.Sprintf("%02d:%02d", hour, minute), logPath, workspace)), 0o600); err != nil {
		return ScheduleStatus{}, err
	}
	_, _ = commandOutput(context.Background(), 5*time.Second, "launchctl", "unload", launchdPlistPath())
	_, _ = commandOutput(context.Background(), 5*time.Second, "launchctl", "load", launchdPlistPath())
	return scheduleStatus()
}

func removeSchedule() (ScheduleStatus, error) {
	path := launchdPlistPath()
	_, _ = commandOutput(context.Background(), 5*time.Second, "launchctl", "unload", path)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return ScheduleStatus{}, err
	}
	return ScheduleStatus{Installed: false, Label: scheduleLabel, PlistPath: path, Message: "schedule removed"}, nil
}

func scheduleStatus() (ScheduleStatus, error) {
	path := launchdPlistPath()
	status := ScheduleStatus{Label: scheduleLabel, PlistPath: path}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		status.Message = "daily schedule is not installed"
		return status, nil
	}
	if err != nil {
		return status, err
	}
	hour, minute := plistHourMinute(string(b))
	status.WorkspacePath = plistStringAfter(string(b), "WorkingDirectory")
	status.Installed = true
	status.Hour = hour
	status.Minute = minute
	status.NextRun = nextRunAt(time.Now(), hour, minute)
	status.Message = fmt.Sprintf("daily schedule installed for %02d:%02d local time", hour, minute)
	if status.WorkspacePath != "" {
		status.Message = fmt.Sprintf("daily schedule installed for %02d:%02d local time in %s", hour, minute, status.WorkspacePath)
	}
	return status, nil
}

func runScheduledBaseline(ctx context.Context) (ScheduleRunResult, error) {
	cfg, err := loadConfig()
	if err != nil {
		return ScheduleRunResult{}, err
	}
	workspace := runtimeWorkspace(cfg)
	run, err := RunBaseline(ctx, RunOptions{Mode: "run", RunAgent: true, Packs: cfg.Target.Packs, Workspace: workspace})
	if err != nil {
		return ScheduleRunResult{}, err
	}
	artifacts, _ := writeRunArtifacts(run)
	run.Artifacts = artifacts
	var syncResult SyncFlushResult
	if cfg.CloudSync && cfg.APIToken != "" {
		db, err := openDB()
		if err != nil {
			return ScheduleRunResult{}, err
		}
		defer db.Close()
		if _, err := stageUnsyncedRuns(db, 50); err != nil {
			return ScheduleRunResult{}, err
		}
		syncResult, _ = flushSyncOutbox(ctx, db, cfg)
	}
	return ScheduleRunResult{
		Action:      "run",
		RunID:       run.ID,
		Mode:        run.Mode,
		Status:      run.Status,
		HealthScore: run.HealthScore,
		CloudSynced: run.CloudSynced || syncResult.Synced > 0,
		SyncSynced:  syncResult.Synced,
		SyncFailed:  syncResult.Failed,
		Workspace:   workspace,
		ReportPath:  artifacts.ReportPath,
	}, nil
}

func launchdPlistPath() string {
	return filepath.Join(homeDir(), "Library", "LaunchAgents", scheduleLabel+".plist")
}

func parseScheduleTime(value string) (int, int, error) {
	if value == "" {
		value = "09:00"
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("time must be HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("time must be between 00:00 and 23:59")
	}
	return hour, minute, nil
}

func nextRunAt(now time.Time, hour, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func launchdPlist(exe, at, logPath, workspace string) string {
	hour, minute, _ := parseScheduleTime(at)
	workspace = normalizeWorkspacePath(workspace)
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>` + escapeXML(scheduleLabel) + `</string>
  <key>WorkingDirectory</key>
  <string>` + escapeXML(workspace) + `</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>BASELINE_WORKSPACE</key>
    <string>` + escapeXML(workspace) + `</string>
    <key>HOME</key>
    <string>` + escapeXML(homeDir()) + `</string>
    <key>PATH</key>
    <string>` + escapeXML(launchdPath(exe)) + `</string>
  </dict>
  <key>ProgramArguments</key>
  <array>
    <string>` + escapeXML(exe) + `</string>
    <string>schedule</string>
    <string>run</string>
  </array>
  <key>StartCalendarInterval</key>
  <dict>
    <key>Hour</key>
    <integer>` + strconv.Itoa(hour) + `</integer>
    <key>Minute</key>
    <integer>` + strconv.Itoa(minute) + `</integer>
  </dict>
  <key>StandardOutPath</key>
  <string>` + escapeXML(logPath) + `</string>
  <key>StandardErrorPath</key>
  <string>` + escapeXML(logPath) + `</string>
  <key>RunAtLoad</key>
  <false/>
</dict>
</plist>
`
}

func plistHourMinute(plist string) (int, int) {
	hour := plistIntAfter(plist, "Hour")
	minute := plistIntAfter(plist, "Minute")
	return hour, minute
}

func plistIntAfter(plist, key string) int {
	needle := "<key>" + key + "</key>"
	idx := strings.Index(plist, needle)
	if idx < 0 {
		return 0
	}
	rest := plist[idx+len(needle):]
	start := strings.Index(rest, "<integer>")
	end := strings.Index(rest, "</integer>")
	if start < 0 || end < 0 || end <= start {
		return 0
	}
	value, _ := strconv.Atoi(strings.TrimSpace(rest[start+len("<integer>") : end]))
	return value
}

func plistStringAfter(plist, key string) string {
	needle := "<key>" + key + "</key>"
	idx := strings.Index(plist, needle)
	if idx < 0 {
		return ""
	}
	rest := plist[idx+len(needle):]
	start := strings.Index(rest, "<string>")
	end := strings.Index(rest, "</string>")
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	return strings.TrimSpace(rest[start+len("<string>") : end])
}

func launchdPath(exe string) string {
	parts := []string{
		filepath.Dir(exe),
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		filepath.Join(homeDir(), "go", "bin"),
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
	}
	seen := map[string]bool{}
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return strings.Join(out, ":")
}

func escapeXML(value string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}

func openClawScheduleCommand() map[string]string {
	return map[string]string{
		"install": `Ask Baseline to call MCP tool baseline_schedule with {"action":"install","at":"09:00"}.`,
		"status":  `Ask Baseline to call MCP tool baseline_schedule with {"action":"status"}.`,
		"run_now": `Ask Baseline to call MCP tool baseline_schedule with {"action":"run"}.`,
	}
}

func launchctlAvailable() bool {
	_, err := exec.LookPath("launchctl")
	return err == nil
}
