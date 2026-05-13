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

func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}
	ctx := context.Background()
	switch args[0] {
	case "init":
		return cmdInit(args[1:], stdout, stderr)
	case "check":
		return cmdCheck(ctx, args[1:], stdout, stderr)
	case "latest":
		return cmdLatest(args[1:], stdout, stderr)
	case "report":
		return cmdReport(args[1:], stdout, stderr)
	case "compare":
		return cmdCompare(stdout, stderr)
	case "known-good":
		return cmdKnownGood(args[1:], stdout, stderr)
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
  baseline init [--register-openclaw]
  baseline check [--fast|--full] [--run-agent] [--json] [--agent-command CMD]
  baseline latest [--json]
  baseline report [RUN_ID]
  baseline compare
  baseline known-good mark [RUN_ID] [--label LABEL]
  baseline known-good list
  baseline install openclaw
  baseline serve mcp
  baseline sync status|on|off [--token TOKEN] [--url URL]
  baseline scrub preview <text>

Safety defaults:
  - Local SQLite only until sync is enabled.
  - Full question probes do not execute an agent unless --run-agent or BASELINE_RUN_AGENT=1 is set.
  - Cloud export stores redacted summaries by default.
`)
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
	if _, err := os.Stat(redactionPath()); errors.Is(err, os.ErrNotExist) {
		_ = atomicWrite(redactionPath(), []byte("# Baseline local redaction rules. Cloud sync exports summaries unless allow_raw_output is true.\n"), 0o600)
	}
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
	run, err := RunBaseline(ctx, RunOptions{Mode: mode, RunAgent: *runAgent, AgentCommand: *agentCommand})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	writeReportFile(run)
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
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	var run Run
	if len(args) > 0 {
		run, err = runByID(db, args[0])
	} else {
		run, err = latestRun(db)
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
	payload := map[string]any{"run": run, "observations": observations}
	return writeJSON(stdout, stderr, payload)
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
		fmt.Fprintln(stdout, "No known-good drift detected.")
		return 0
	}
	for _, f := range findings {
		fmt.Fprintf(stdout, "- %s: %s\n", f.CheckID, f.Message)
	}
	return 0
}

func cmdKnownGood(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: baseline known-good mark|list")
		return 2
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	switch args[0] {
	case "mark":
		fs := flag.NewFlagSet("known-good mark", flag.ContinueOnError)
		fs.SetOutput(stderr)
		label := fs.String("label", "known-good", "label")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		runID := ""
		if fs.NArg() > 0 {
			runID = fs.Arg(0)
		} else {
			run, err := latestRun(db)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			runID = run.ID
		}
		if err := markKnownGood(db, runID, *label); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Marked %s as %s\n", runID, *label)
		return 0
	case "list":
		goods, err := listKnownGoods(db)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, stderr, goods)
	default:
		fmt.Fprintln(stderr, "usage: baseline known-good mark|list")
		return 2
	}
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
	fmt.Fprintln(stdout, "Registered Baseline MCP with OpenClaw.")
	fmt.Fprintln(stdout, "Verify with: openclaw mcp list")
	return 0
}

func cmdSync(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: baseline sync status|on|off")
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
		fmt.Fprintf(stdout, "sync=%s url=%s token_set=%t allow_raw_output=%t\n", state, cfg.APIBaseURL, cfg.APIToken != "", cfg.AllowRawOutput)
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
	default:
		fmt.Fprintln(stderr, "usage: baseline sync status|on|off")
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
	run, err := RunBaseline(ctx, RunOptions{Mode: "fast"})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "config=%s db=%s sync=%t openclaw=%t latest_score=%d\n", configPath(), dbPath(), cfg.CloudSync, openClawErr == nil, run.HealthScore)
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
	b, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return
	}
	_ = ensureDirs()
	_ = atomicWrite(filepath.Join(baseDir(), "reports", run.ID+".json"), b, 0o600)
}

func statusCode(status string) int {
	if status == "critical" {
		return 3
	}
	return 0
}
