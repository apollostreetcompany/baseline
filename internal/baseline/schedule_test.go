package baseline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLaunchdPlistIncludesDailyBaselineRun(t *testing.T) {
	plist := launchdPlist("/usr/local/bin/baseline", "09:30", "/tmp/baseline.log")
	for _, want := range []string{
		"<key>Label</key>",
		"ai.baseline.daily",
		"<string>/usr/local/bin/baseline</string>",
		"<string>schedule</string>",
		"<string>run</string>",
		"<key>Hour</key>",
		"<integer>9</integer>",
		"<key>Minute</key>",
		"<integer>30</integer>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("plist missing %q:\n%s", want, plist)
		}
	}
}

func TestScheduleStatusReadsLaunchdPlist(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "Library", "LaunchAgents"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(launchdPlistPath(), []byte(launchdPlist("/tmp/baseline", "07:15", "/tmp/log")), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := scheduleStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !status.Installed || status.Hour != 7 || status.Minute != 15 {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestParseScheduleTime(t *testing.T) {
	hour, minute, err := parseScheduleTime("23:05")
	if err != nil {
		t.Fatal(err)
	}
	if hour != 23 || minute != 5 {
		t.Fatalf("unexpected time: %d:%d", hour, minute)
	}
	if _, _, err := parseScheduleTime("25:00"); err == nil {
		t.Fatalf("expected invalid hour to fail")
	}
}

func TestNextRunAt(t *testing.T) {
	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.Local)
	next := nextRunAt(now, 9, 30)
	if next.Day() != 15 || next.Hour() != 9 || next.Minute() != 30 {
		t.Fatalf("unexpected next run: %s", next)
	}
}

func TestScheduledRunIsFastOnlyAndDoesNotExecuteAgent(t *testing.T) {
	t.Setenv("BASELINE_HOME", t.TempDir())
	marker := filepath.Join(t.TempDir(), "agent-ran")
	cfg := defaultConfig()
	cfg.AgentCommand = "touch " + marker
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	result, err := runScheduledBaseline(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "run" || result.RunID == "" {
		t.Fatalf("unexpected schedule result: %+v", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("scheduled daily run must stay fast/local and not execute the configured agent")
	}
}
