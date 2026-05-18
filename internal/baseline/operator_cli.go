package baseline

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"strings"
)

func cmdSetup(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	registerOpenClawFlag := fs.Bool("register-openclaw", false, "also register Baseline as an OpenClaw MCP server")
	agentCommand := fs.String("agent-command", "", "advanced: override configured target command for this setup run")
	packs := fs.String("packs", "", "advanced: baseline, enabled, all, or comma-separated pack ids")
	runID := fs.String("run-id", "", "internal: preassigned run id")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, operatorError("setup.config", err, "Run baseline config validate after fixing config.json."))
		return 1
	}
	if cfg.WorkspacePath == "" {
		cfg.WorkspacePath = runtimeWorkspace(cfg)
	}
	if err := saveConfig(cfg); err != nil {
		fmt.Fprintln(stderr, operatorError("setup.config_write", err, "Check permissions on "+configPath()+"."))
		return 1
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, operatorError("setup.database", err, "Check permissions on "+dbPath()+"."))
		return 1
	}
	db.Close()
	if err := ensureRedactionFile(); err != nil {
		fmt.Fprintln(stderr, operatorError("setup.redaction", err, "Check permissions on "+redactionPath()+"."))
		return 1
	}
	if err := writeBootstrapContract(cfg); err != nil {
		fmt.Fprintln(stderr, operatorError("setup.bootstrap_contract", err, "Check permissions on "+bootstrapContractPath()+"."))
		return 1
	}
	openClawRegistered := false
	if *registerOpenClawFlag {
		if err := registerOpenClaw(); err != nil {
			fmt.Fprintln(stderr, operatorError("setup.openclaw_registration", err, "Run baseline install openclaw after OpenClaw is healthy."))
			return 1
		}
		openClawRegistered = true
	}
	if *packs == "" {
		*packs = cfg.Target.Packs
	}
	assignedRunID := strings.TrimSpace(*runID)
	if assignedRunID == "" {
		assignedRunID = newRunID()
	}
	_ = writeRunLifecycleStatus(startedRunStatus(assignedRunID, "setup"))
	run, err := RunBaseline(ctx, RunOptions{Mode: "setup", RunID: assignedRunID, RunAgent: true, AgentCommand: *agentCommand, Packs: *packs})
	if err != nil {
		_ = writeRunLifecycleStatus(failedRunStatus(assignedRunID, "setup", err))
		fmt.Fprintln(stderr, operatorError("setup.eval", err, "Run baseline doctor, fix the reported target/config issue, then run baseline run."))
		return 1
	}
	artifacts, _ := writeRunArtifacts(run)
	run.Artifacts = artifacts
	_ = writeRunLifecycleStatus(completedRunStatus(run))
	payload := map[string]any{
		"status":              "setup_complete",
		"config_path":         configPath(),
		"database_path":       dbPath(),
		"bootstrap_contract":  bootstrapContractPath(),
		"openclaw_registered": openClawRegistered,
		"target":              cfg.Target,
		"run":                 run,
		"next_actions": []string{
			"Review baseline report " + run.ID,
			"Inspect full local responses before accepting",
			fmt.Sprintf("Accept only with: baseline accept %s --confirm \"accept %s\"", run.ID, run.ID),
		},
	}
	if *jsonOut {
		return writeJSON(stdout, stderr, payload)
	}
	fmt.Fprintln(stdout, "Baseline setup complete.")
	fmt.Fprintf(stdout, "Config: %s\n", configPath())
	fmt.Fprintf(stdout, "Agent bootstrap: %s\n", bootstrapContractPath())
	if cfg.WorkspacePath != "" {
		fmt.Fprintf(stdout, "Workspace: %s\n", cfg.WorkspacePath)
	}
	fmt.Fprintf(stdout, "Target: %s %s (%s)\n", cfg.Target.Runtime, cfg.Target.Entity, targetModelDisplay(cfg.Target))
	printRunSummary(stdout, run)
	return statusCode(run.Status)
}

func cmdRun(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	agentCommand := fs.String("agent-command", "", "advanced: override configured target command for this run")
	packs := fs.String("packs", "", "advanced: baseline, enabled, all, or comma-separated pack ids")
	runID := fs.String("run-id", "", "internal: preassigned run id")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, operatorError("run.config", err, "Run baseline setup to recreate defaults, or baseline config validate for details."))
		return 1
	}
	if *packs == "" {
		*packs = cfg.Target.Packs
	}
	assignedRunID := strings.TrimSpace(*runID)
	if assignedRunID == "" {
		assignedRunID = newRunID()
	}
	questionCount := len(selectedQuestions(cfg, *packs))
	_ = writeRunLifecycleStatus(plannedRunStatus(assignedRunID, "run", *packs, questionCount))
	if !*jsonOut {
		fmt.Fprintf(stdout, "Starting Baseline %s: target=%s %s, packs=%s, questions=%d, workspace=%s\n", assignedRunID, cfg.Target.Runtime, cfg.Target.Entity, *packs, questionCount, runtimeWorkspace(cfg))
	}
	run, err := RunBaseline(ctx, RunOptions{Mode: "run", RunID: assignedRunID, RunAgent: true, AgentCommand: *agentCommand, Packs: *packs})
	if err != nil {
		_ = writeRunLifecycleStatus(failedRunStatus(assignedRunID, "run", err))
		fmt.Fprintln(stderr, operatorError("run.eval", err, "Run baseline doctor, fix the reported target/config issue, then run baseline run again."))
		return 1
	}
	artifacts, _ := writeRunArtifacts(run)
	run.Artifacts = artifacts
	_ = writeRunLifecycleStatus(completedRunStatus(run))
	if *jsonOut {
		return writeJSON(stdout, stderr, run)
	}
	printRunSummary(stdout, run)
	return statusCode(run.Status)
}

func cmdAccept(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("accept", flag.ContinueOnError)
	fs.SetOutput(stderr)
	label := fs.String("label", "Good baseline", "Good Baseline label")
	notes := fs.String("notes", "", "acceptance notes")
	slotValue := fs.String("slot", "auto", "Good Baseline slot: auto, 1, 2, or 3")
	confirm := fs.String("confirm", "", "required confirmation: accept <run_id>")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	runID := fs.Arg(0)
	if runID == "" {
		fmt.Fprintln(stderr, "usage: baseline accept RUN_ID --confirm \"accept RUN_ID\"")
		return 2
	}
	want := "accept " + runID
	if strings.TrimSpace(*confirm) != want {
		fmt.Fprintf(stderr, "accept requires explicit operator confirmation: --confirm %q\n", want)
		return 1
	}
	good, err := acceptCandidateOrRun(runID, *label, *notes, parseSlot(*slotValue), false)
	if err != nil {
		fmt.Fprintln(stderr, operatorError("accept.good_baseline", err, "Run baseline report "+runID+" and verify this run belongs to the current workspace/config."))
		return 1
	}
	return writeJSON(stdout, stderr, good)
}

func cmdStatus(args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
		}
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	status, err := currentBootstrapStatus()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	schedule, _ := scheduleStatus()
	var latest *Run
	db, err := openDB()
	if err == nil {
		defer db.Close()
		run, latestErr := latestRun(db)
		if latestErr == nil {
			latest = &run
		} else if latestErr != sql.ErrNoRows {
			fmt.Fprintln(stderr, latestErr)
			return 1
		}
	}
	payload := map[string]any{
		"config_path":        configPath(),
		"bootstrap_contract": bootstrapContractPath(),
		"target":             cfg.Target,
		"good_baselines":     status.GoodBaselines,
		"needs_baseline":     status.NeedsBootstrap,
		"schedule":           schedule,
		"latest_run":         latest,
	}
	if jsonOut {
		return writeJSON(stdout, stderr, payload)
	}
	fmt.Fprintf(stdout, "Baseline status\n")
	fmt.Fprintf(stdout, "  config: %s\n", configPath())
	fmt.Fprintf(stdout, "  agent bootstrap: %s\n", bootstrapContractPath())
	if cfg.WorkspacePath != "" {
		fmt.Fprintf(stdout, "  workspace_path: %s\n", cfg.WorkspacePath)
	}
	fmt.Fprintf(stdout, "  target: %s %s (%s)\n", cfg.Target.Runtime, cfg.Target.Entity, targetModelDisplay(cfg.Target))
	fmt.Fprintf(stdout, "  good_baselines: %d\n", len(status.GoodBaselines))
	if latest != nil {
		fmt.Fprintf(stdout, "  latest: %s score=%d status=%s mode=%s\n", latest.ID, latest.HealthScore, latest.Status, latest.Mode)
	} else {
		fmt.Fprintf(stdout, "  latest: none\n")
	}
	fmt.Fprintf(stdout, "  schedule: %s\n", schedule.Message)
	return 0
}

func printRunSummary(stdout io.Writer, run Run) {
	fmt.Fprintf(stdout, "Baseline %s: score %d (%s) in %dms\n", run.ID, run.HealthScore, run.Status, run.DurationMS)
	if len(run.Findings) == 0 {
		fmt.Fprintln(stdout, "No findings.")
	} else {
		for _, f := range run.Findings {
			fmt.Fprintf(stdout, "- %s %s: %s\n", strings.ToUpper(f.Severity), f.CheckID, f.Message)
			if f.Fix != "" {
				fmt.Fprintf(stdout, "  Fix: %s\n", f.Fix)
			}
		}
	}
	if run.Artifacts.ReportPath != "" {
		fmt.Fprintf(stdout, "Report: %s\n", run.Artifacts.ReportPath)
		fmt.Fprintf(stdout, "Responses: %s\n", run.Artifacts.ResponsesPath)
	}
	fmt.Fprintf(stdout, "Review: baseline report %s\n", run.ID)
	fmt.Fprintf(stdout, "Accept only after review: baseline accept %s --confirm \"accept %s\"\n", run.ID, run.ID)
}

func operatorError(step string, err error, next string) string {
	return fmt.Sprintf("%s failed: %v\nNext: %s", step, err, next)
}
