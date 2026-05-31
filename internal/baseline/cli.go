package baseline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const Version = "0.1.0"

func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}
	ctx := context.Background()
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "baseline %s\n", Version)
		return 0
	case "init":
		return cmdInit(args[1:], stdout, stderr)
	case "setup":
		return cmdSetup(ctx, args[1:], stdout, stderr)
	case "run":
		return cmdRun(ctx, args[1:], stdout, stderr)
	case "accept":
		return cmdAccept(args[1:], stdout, stderr)
	case "status":
		return cmdStatus(args[1:], stdout, stderr)
	case "bootstrap":
		return cmdBootstrap(ctx, args[1:], stdout, stderr)
	case "check":
		return cmdCheck(ctx, args[1:], stdout, stderr)
	case "latest":
		return cmdLatest(args[1:], stdout, stderr)
	case "report":
		return cmdReport(args[1:], stdout, stderr)
	case "rerun":
		return cmdRerun(args[1:], stdout, stderr)
	case "repair":
		return cmdRepair(args[1:], stdout, stderr)
	case "compare":
		return cmdCompare(stdout, stderr)
	case "good":
		return cmdGood(args[1:], stdout, stderr)
	case "known-good":
		return cmdKnownGood(args[1:], stdout, stderr)
	case "config":
		return cmdConfig(args[1:], stdout, stderr)
	case "install":
		return cmdInstall(args[1:], stdout, stderr)
	case "serve":
		if len(args) > 1 && args[1] == "mcp" {
			return ServeMCP(os.Stdin, stdout, stderr)
		}
		fmt.Fprintln(stderr, "usage: baseline serve mcp")
		return 2
	case "sync":
		return cmdSync(args[1:], stdout, stderr)
	case "schedule":
		return cmdSchedule(ctx, args[1:], stdout, stderr)
	case "scrub":
		return cmdScrub(args[1:], stdout, stderr)
	case "doctor":
		return cmdDoctor(ctx, stdout, stderr)
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `Baseline v0

Usage:
  baseline --version
  baseline version
  baseline setup [--json]
  baseline run [--json]
  baseline report [RUN_ID]
  baseline rerun RUN_ID
  baseline repair openclaw
  baseline accept RUN_ID --confirm "accept RUN_ID"
  baseline status [--json]
  baseline doctor
  baseline compare
  baseline latest [--json]
  baseline good accept [RUN_ID] [--slot auto|1|2|3] [--label LABEL]
  baseline good list
  baseline config file|show|get|set|patch|unset|validate
  baseline install openclaw
  baseline serve mcp
  baseline sync status|on|off|push [--token TOKEN] [--url URL]
  baseline schedule install|status|run|remove [--at HH:MM]
  baseline scrub preview <text>

Advanced:
  baseline check [--fast|--full] [--run-agent] [--packs enabled|all|baseline] [--json] [--agent-command CMD]
  baseline bootstrap [--openclaw] | status|defaults|preview|run|accept|reject

Safety defaults:
  - baseline run executes the operator-approved target and records response quality/timing.
  - baseline doctor is read-only preflight; it is not a Good Baseline candidate.
  - Good Baselines require explicit operator confirmation.
  - Cloud export stores redacted summaries by default; full responses stay local.
`)
}

func cmdSchedule(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: baseline schedule install|status|run|remove [--at HH:MM]")
		return 2
	}
	switch args[0] {
	case "install":
		fs := flag.NewFlagSet("schedule install", flag.ContinueOnError)
		fs.SetOutput(stderr)
		at := fs.String("at", "09:00", "daily local time in HH:MM")
		exe := fs.String("exe", "", "baseline executable path")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		status, err := installSchedule(*exe, *at)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, stderr, status)
	case "status":
		status, err := scheduleStatus()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, stderr, status)
	case "run":
		result, err := runScheduledBaseline(ctx)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, stderr, result)
	case "remove":
		status, err := removeSchedule()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, stderr, status)
	default:
		fmt.Fprintln(stderr, "usage: baseline schedule install|status|run|remove [--at HH:MM]")
		return 2
	}
}

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	register := fs.Bool("register-openclaw", false, "register Baseline as an OpenClaw MCP")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := saveConfig(cfg); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, err := openDB(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	_ = ensureRedactionFile()
	_ = writeBootstrapContract(cfg)
	fmt.Fprintf(stdout, "Initialized Baseline at %s\n", baseDir())
	fmt.Fprintf(stdout, "Config: %s\nDatabase: %s\n", configPath(), dbPath())
	if *register {
		if err := registerOpenClaw(); err != nil {
			fmt.Fprintf(stderr, "OpenClaw registration failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "Registered Baseline MCP with OpenClaw.")
		return 0
	}
	fmt.Fprintln(stdout, "Install MCP: baseline install openclaw")
	return 0
}

func cmdCheck(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	full := fs.Bool("full", false, "run the timed question pack")
	fast := fs.Bool("fast", false, "run local checks only")
	runAgent := fs.Bool("run-agent", false, "allow Baseline to execute the configured agent")
	jsonOut := fs.Bool("json", false, "print JSON")
	agentCommand := fs.String("agent-command", "", "agent command; prompt is available as BASELINE_PROMPT")
	packs := fs.String("packs", "enabled", "question packs to run: enabled, all, or comma-separated pack ids")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	mode := "fast"
	if *full {
		mode = "full"
	}
	if *fast {
		mode = "fast"
	}
	run, err := RunBaseline(ctx, RunOptions{Mode: mode, RunAgent: *runAgent, AgentCommand: *agentCommand, Packs: *packs})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	artifacts, _ := writeRunArtifacts(run)
	run.Artifacts = artifacts
	if *jsonOut {
		return writeJSON(stdout, stderr, run)
	}
	fmt.Fprintf(stdout, "Baseline %s: score %d (%s) in %dms\n", run.ID, run.HealthScore, run.Status, run.DurationMS)
	for _, f := range run.Findings {
		fmt.Fprintf(stdout, "- %s %s: %s\n", strings.ToUpper(f.Severity), f.CheckID, f.Message)
	}
	if len(run.Findings) == 0 {
		fmt.Fprintln(stdout, "No findings.")
	}
	return statusCode(run.Status)
}

func cmdLatest(args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
		}
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	run, err := latestRun(db)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Fprintln(stderr, "no baseline runs yet")
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if jsonOut {
		return writeJSON(stdout, stderr, run)
	}
	fmt.Fprintf(stdout, "%s %s score=%d status=%s checks=%d\n", run.ID, run.StartedAt.Format(time.RFC3339), run.HealthScore, run.Status, len(run.Checks))
	return 0
}

func cmdReport(args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	var positional []string
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			continue
		}
		positional = append(positional, arg)
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	var run Run
	if len(positional) > 0 {
		run, err = runByID(db, positional[0])
	} else {
		run, err = latestRun(db)
	}
	if errors.Is(err, sql.ErrNoRows) && len(positional) > 0 {
		status, statusErr := readRunLifecycleStatus(positional[0])
		if statusErr == nil {
			if jsonOut {
				if code := writeJSON(stdout, stderr, map[string]any{"run_status": status}); code != 0 {
					return code
				}
				return statusCodeForLifecycle(status)
			}
			fmt.Fprintf(stdout, "Baseline %s is %s", status.RunID, status.State)
			if status.Packs != "" || status.Questions > 0 {
				fmt.Fprintf(stdout, " (packs=%s questions=%d)", status.Packs, status.Questions)
			}
			fmt.Fprintln(stdout)
			if status.CurrentQuestion != "" || status.CompletedQuestions > 0 {
				fmt.Fprintf(stdout, "Progress: %d/%d current=%s", status.CompletedQuestions, status.Questions, status.CurrentQuestion)
				if status.ProgressNote != "" {
					fmt.Fprintf(stdout, " note=%s", status.ProgressNote)
				}
				fmt.Fprintln(stdout)
			}
			if status.Error != "" {
				fmt.Fprintf(stdout, "Error: %s\n", status.Error)
			}
			if status.StdoutPath != "" {
				fmt.Fprintf(stdout, "Stdout: %s\n", status.StdoutPath)
			}
			if status.StderrPath != "" {
				fmt.Fprintf(stdout, "Stderr: %s\n", status.StderrPath)
			}
			for _, action := range status.NextActions {
				fmt.Fprintf(stdout, "- %s\n", action)
			}
			return statusCodeForLifecycle(status)
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	observations, err := observationsForRun(db, run.ID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !jsonOut {
		artifacts := runArtifactPaths(run.ID)
		if report, err := os.ReadFile(artifacts.ReportPath); err == nil {
			fmt.Fprint(stdout, string(report))
			if responses, err := os.ReadFile(artifacts.ResponsesPath); err == nil {
				fmt.Fprint(stdout, "\n\n")
				fmt.Fprint(stdout, string(responses))
			}
			return 0
		}
		fmt.Fprintf(stdout, "Baseline %s: score %d (%s) in %dms\n", run.ID, run.HealthScore, run.Status, run.DurationMS)
		for _, f := range run.Findings {
			fmt.Fprintf(stdout, "- %s %s: %s\n", strings.ToUpper(f.Severity), f.CheckID, f.Message)
		}
		fmt.Fprintf(stdout, "\nNo markdown artifacts found. Re-run `baseline run` to write REPORT.md and RESPONSES.md.\n")
		return 0
	}
	payload := map[string]any{"run": run, "observations": observations}
	return writeJSON(stdout, stderr, payload)
}

func statusCodeForLifecycle(status RunLifecycleStatus) int {
	switch status.State {
	case "completed":
		return 0
	case "running":
		return 2
	default:
		return 1
	}
}

func cmdCompare(stdout, stderr io.Writer) int {
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	run, err := latestRun(db)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	findings, err := compareToKnownGood(db, run.ID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "No Good Baseline drift detected.")
		return 0
	}
	for _, f := range findings {
		fmt.Fprintf(stdout, "- %s: %s\n", f.CheckID, f.Message)
	}
	return 0
}

func cmdInstall(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "openclaw" {
		fmt.Fprintln(stderr, "usage: baseline install openclaw")
		return 2
	}
	if err := registerOpenClaw(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	timeoutStatus, err := ensureOpenClawCodexTimeout()
	if err != nil {
		fmt.Fprintln(stderr, operatorError("install.openclaw_codex_timeout", err, "Fix ~/.openclaw/openclaw.json, then rerun baseline install openclaw."))
		return 1
	}
	fmt.Fprintln(stdout, "Registered Baseline MCP with OpenClaw.")
	printOpenClawCodexTimeoutStatus(stdout, timeoutStatus)
	fmt.Fprintln(stdout, "Verify with: openclaw mcp list")
	return 0
}

func cmdSync(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: baseline sync status|on|off|push")
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch args[0] {
	case "status":
		state := "off"
		if cfg.CloudSync {
			state = "on"
		}
		db, err := openDB()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer db.Close()
		counts, err := syncOutboxCounts(db)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "sync=%s url=%s token_set=%t allow_raw_output=%t outbox_pending=%d outbox_failed=%d outbox_synced=%d\n", state, cfg.APIBaseURL, cfg.APIToken != "", cfg.AllowRawOutput, counts.Pending, counts.Failed, counts.Synced)
		return 0
	case "on":
		fs := flag.NewFlagSet("sync on", flag.ContinueOnError)
		fs.SetOutput(stderr)
		token := fs.String("token", "", "API token")
		url := fs.String("url", "", "API base URL")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		cfg.CloudSync = true
		if *token != "" {
			cfg.APIToken = *token
		}
		if *url != "" {
			cfg.APIBaseURL = *url
		}
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "Cloud sync enabled for redacted summaries.")
		return 0
	case "off":
		cfg.CloudSync = false
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "Cloud sync disabled.")
		return 0
	case "push":
		db, err := openDB()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer db.Close()
		staged, err := stageUnsyncedRuns(db, 50)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		result, err := flushSyncOutbox(context.Background(), db, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "sync push failed after staging %d runs: %v\n", staged, err)
			return 1
		}
		fmt.Fprintf(stdout, "Sync push staged=%d synced=%d failed=%d\n", staged, result.Synced, result.Failed)
		return 0
	default:
		fmt.Fprintln(stderr, "usage: baseline sync status|on|off|push")
		return 2
	}
}

func cmdScrub(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || args[0] != "preview" {
		fmt.Fprintln(stderr, "usage: baseline scrub preview <text>")
		return 2
	}
	out, report := scrubText(strings.Join(args[1:], " "))
	payload := map[string]any{"scrubbed": out, "report": report}
	return writeJSON(stdout, stderr, payload)
}

func cmdDoctor(ctx context.Context, stdout, stderr io.Writer) int {
	cfg, _ := loadConfig()
	_, openClawErr := exec.LookPath("openclaw")
	run, err := RunBaseline(ctx, RunOptions{Mode: "doctor", Ephemeral: true})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "Baseline doctor")
	fmt.Fprintf(stdout, "  config: %s\n", configPath())
	fmt.Fprintf(stdout, "  database: %s\n", dbPath())
	fmt.Fprintf(stdout, "  target: %s %s (%s)\n", cfg.Target.Runtime, cfg.Target.Entity, targetModelDisplay(cfg.Target))
	fmt.Fprintf(stdout, "  sync: %t\n", cfg.CloudSync)
	fmt.Fprintf(stdout, "  openclaw_on_path: %t\n", openClawErr == nil)
	fmt.Fprintf(stdout, "  preflight_score: %d (%s)\n", run.HealthScore, run.Status)
	for _, f := range run.Findings {
		fmt.Fprintf(stdout, "- %s %s: %s\n", strings.ToUpper(f.Severity), f.CheckID, f.Message)
		if f.Fix != "" {
			fmt.Fprintf(stdout, "  Fix: %s\n", f.Fix)
		}
	}
	if len(run.Findings) == 0 {
		fmt.Fprintln(stdout, "No preflight findings. Next: baseline run")
	}
	return statusCode(run.Status)
}

func registerOpenClaw() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.Abs(exe)
	payload := fmt.Sprintf(`{"command":%q,"args":["serve","mcp"]}`, exe)
	if _, err := exec.LookPath("openclaw"); err != nil {
		return err
	}
	_, err = commandOutput(context.Background(), 10*time.Second, "openclaw", "mcp", "set", "baseline", payload)
	return err
}

func writeJSON(stdout, stderr io.Writer, v any) int {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func writeReportFile(run Run) {
	_, _ = writeRunArtifacts(run)
}

func statusCode(status string) int {
	if status == "critical" {
		return 3
	}
	return 0
}
