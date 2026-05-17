package baseline

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func ServeMCP(stdin io.Reader, stdout, stderr io.Writer) int {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	enc := json.NewEncoder(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(stderr, "invalid MCP JSON: %v\n", err)
			continue
		}
		if req.ID == nil {
			continue
		}
		result, rpcErr := handleMCP(req)
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
		if rpcErr != nil {
			resp.Result = nil
			resp.Error = map[string]any{"code": -32000, "message": rpcErr.Error()}
		}
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(stderr, "write MCP response: %v\n", err)
			return 1
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "read MCP stdin: %v\n", err)
		return 1
	}
	return 0
}

func handleMCP(req rpcRequest) (any, error) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": "baseline", "version": "0.1.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		}, nil
	case "tools/list":
		return map[string]any{"tools": mcpTools()}, nil
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return callMCPTool(params.Name, params.Arguments)
	default:
		return nil, fmt.Errorf("unsupported MCP method %s", req.Method)
	}
}

func mcpTools() []map[string]any {
	stringProp := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	boolProp := func(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }
	return []map[string]any{
		{
			"name":        "baseline_setup",
			"description": "Discovery: first tool to call when Baseline is not configured or the operator says \"run baseline\" for the first time. It writes Baseline-owned setup files, runs the real default target eval, returns report paths and next actions. Do not accept the result for the user.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{
				"packs":             stringProp("advanced: baseline, enabled, all, or comma-separated pack ids"),
				"agent_command":     stringProp("advanced escape hatch; prompt is passed as BASELINE_PROMPT"),
				"register_openclaw": boolProp("also register Baseline as an OpenClaw MCP server"),
			}},
		},
		{
			"name":        "baseline_run",
			"description": "Run the normal Baseline evaluation for the operator-approved default target. This sends real probe messages, records latency/quality, writes REPORT.md and RESPONSES.md, and returns next actions.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{
				"packs":         stringProp("advanced: baseline, enabled, all, or comma-separated pack ids"),
				"agent_command": stringProp("advanced escape hatch; prompt is passed as BASELINE_PROMPT"),
			}},
		},
		{
			"name":        "baseline_doctor",
			"description": "Read-only preflight for troubleshooting. It checks config, runtime, MCP registration, scrubber, and local latency. It does not create a Good Baseline candidate and should be used before proposing repairs.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "baseline_report",
			"description": "Return the latest or requested run, markdown report, local responses when available, observations, and Good Baseline comparison. Use this before asking the operator to accept/reject/defer.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"run_id": stringProp("optional run id")}},
		},
		{
			"name":        "baseline_accept",
			"description": "Accept a reviewed run as a Good Baseline. Requires explicit operator confirmation: confirm must exactly equal accept <run_id>. Never call this without showing the report/responses to the operator first.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{
				"run_id":  stringProp("run id to accept"),
				"label":   stringProp("Good Baseline label"),
				"notes":   stringProp("operator notes"),
				"slot":    stringProp("Good Baseline slot: auto, 1, 2, or 3"),
				"confirm": stringProp("required confirmation: accept <run_id>"),
			}, "required": []string{"run_id", "confirm"}},
		},
		{
			"name":        "baseline_schedule",
			"description": "Install, remove, inspect, or trigger the daily local Baseline evaluation. The run action uses the configured default target and writes report artifacts.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"action": stringProp("status, install, remove, or run"), "at": stringProp("daily local time for install, HH:MM")}},
		},
		{
			"name":        "baseline_scrub_preview",
			"description": "Preview redaction before any text is exported.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"text": stringProp("text to scrub")}, "required": []string{"text"}},
		},
	}
}

func callMCPTool(name string, args map[string]any) (any, error) {
	var payload any
	var err error
	switch name {
	case "baseline_setup":
		payload, err = mcpSetup(args)
	case "baseline_run":
		payload, err = mcpRun(args)
	case "baseline_doctor":
		payload, err = mcpDoctor()
	case "baseline_accept":
		payload, err = mcpAccept(args)
	case "baseline_check":
		mode := stringArg(args, "mode", "fast")
		runAgent := boolArg(args, "run_agent", false)
		payload, err = RunBaseline(context.Background(), RunOptions{Mode: mode, RunAgent: runAgent})
	case "baseline_bootstrap":
		payload, err = mcpBootstrap(args)
	case "baseline_good":
		payload, err = mcpGood(args)
	case "baseline_report":
		payload, err = mcpReport(stringArg(args, "run_id", ""))
	case "baseline_compare":
		payload, err = withDB(func(db *sql.DB) (any, error) {
			run, err := latestRun(db)
			if err != nil {
				return nil, err
			}
			return compareToKnownGood(db, run.ID)
		})
	case "baseline_schedule":
		payload, err = mcpSchedule(args)
	case "baseline_config":
		return nil, fmt.Errorf("baseline_config is no longer advertised; use baseline bootstrap/config CLI")
	case "baseline_mark_known_good":
		return nil, fmt.Errorf("baseline_mark_known_good is retired; use baseline_accept after operator report review")
	case "baseline_scrub_preview":
		out, report := scrubText(stringArg(args, "text", ""))
		payload = map[string]any{"scrubbed": out, "report": report}
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	if err != nil {
		return mcpErrorResult(name, err), nil
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(b)}}}, nil
}

func mcpSetup(args map[string]any) (any, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.WorkspacePath == "" {
		cfg.WorkspacePath = runtimeWorkspace(cfg)
	}
	if err := saveConfig(cfg); err != nil {
		return nil, err
	}
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	db.Close()
	if err := ensureRedactionFile(); err != nil {
		return nil, err
	}
	if err := writeBootstrapContract(cfg); err != nil {
		return nil, err
	}
	openClawRegistered := false
	if boolArg(args, "register_openclaw", false) {
		if err := registerOpenClaw(); err != nil {
			return nil, err
		}
		openClawRegistered = true
	}
	status, err := startAsyncMCPRun("setup", args, cfg.Target.Packs)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status":              "setup_started",
		"config_path":         configPath(),
		"bootstrap_contract":  bootstrapContractPath(),
		"openclaw_registered": openClawRegistered,
		"target":              cfg.Target,
		"run_status":          status,
		"next_actions": []string{
			"Tell the operator the Baseline eval started in the background",
			fmt.Sprintf("Poll with baseline_report run_id=%s until state is completed", status.RunID),
			fmt.Sprintf("If accepted after review, call baseline_accept run_id=%s confirm=%q", status.RunID, "accept "+status.RunID),
		},
	}, nil
}

func mcpRun(args map[string]any) (any, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	status, err := startAsyncMCPRun("run", args, cfg.Target.Packs)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"run_status": status,
		"next_actions": []string{
			fmt.Sprintf("Poll with baseline_report run_id=%s until state is completed", status.RunID),
			"Show the operator REPORT.md plus RESPONSES.md when complete",
			fmt.Sprintf("Accept only after operator says yes: baseline_accept run_id=%s confirm=%q", status.RunID, "accept "+status.RunID),
		},
	}, nil
}

func startAsyncMCPRun(mode string, args map[string]any, defaultPacks string) (RunLifecycleStatus, error) {
	runID := newRunID()
	if err := ensureDirs(); err != nil {
		return RunLifecycleStatus{}, err
	}
	stdoutPath, stderrPath := runLifecycleLogPaths(runID)
	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return RunLifecycleStatus{}, err
	}
	defer stdoutFile.Close()
	stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return RunLifecycleStatus{}, err
	}
	defer stderrFile.Close()
	exe := strings.TrimSpace(os.Getenv("BASELINE_ASYNC_EXE"))
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			exe, err = exec.LookPath("baseline")
			if err != nil {
				return RunLifecycleStatus{}, err
			}
		}
	}
	cmdArgs := []string{mode, "--run-id", runID}
	if packs := stringArg(args, "packs", defaultPacks); packs != "" {
		cmdArgs = append(cmdArgs, "--packs", packs)
	}
	if agentCommand := stringArg(args, "agent_command", ""); agentCommand != "" {
		cmdArgs = append(cmdArgs, "--agent-command", agentCommand)
	}
	cmd := exec.Command(exe, cmdArgs...)
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return RunLifecycleStatus{}, err
	}
	status := startedRunStatus(runID, mode)
	status.PID = cmd.Process.Pid
	status.StdoutPath = stdoutPath
	status.StderrPath = stderrPath
	if err := writeRunLifecycleStatus(status); err != nil {
		return RunLifecycleStatus{}, err
	}
	go func() {
		err := cmd.Wait()
		if err == nil {
			return
		}
		current, readErr := readRunLifecycleStatus(runID)
		if readErr == nil && current.State != "running" {
			return
		}
		failed := failedRunStatus(runID, mode, err)
		failed.PID = status.PID
		failed.StdoutPath = stdoutPath
		failed.StderrPath = stderrPath
		_ = writeRunLifecycleStatus(failed)
	}()
	return status, nil
}

func mcpDoctor() (any, error) {
	run, err := RunBaseline(context.Background(), RunOptions{Mode: "doctor", Ephemeral: true})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": run.Status,
		"score":  run.HealthScore,
		"run":    run,
		"next_actions": []string{
			"If doctor is ok, run baseline_run for a real eval",
			"If doctor has findings, propose repairs before running the eval again",
		},
	}, nil
}

func mcpAccept(args map[string]any) (any, error) {
	runID := stringArg(args, "run_id", "")
	if err := requireMCPConfirmation("accept", runID, 0, stringArg(args, "confirm", "")); err != nil {
		return nil, err
	}
	good, err := acceptCandidateOrRun(runID, stringArg(args, "label", "Good baseline"), stringArg(args, "notes", ""), parseSlot(stringArg(args, "slot", "auto")), false)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"accepted": good,
		"next_actions": []string{
			"Run baseline_report to verify the accepted Good Baseline",
		},
	}, nil
}

func mcpErrorResult(tool string, err error) map[string]any {
	payload := map[string]any{
		"isError": true,
		"error": map[string]any{
			"type":         classifyMCPError(err),
			"message":      err.Error(),
			"recoverable":  true,
			"tool":         tool,
			"next_actions": suggestedMCPNextActions(tool, err),
		},
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": string(b)}}}
}

func classifyMCPError(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "confirm"):
		return "operator_confirmation_required"
	case strings.Contains(msg, "openclaw"), strings.Contains(msg, "runtime"):
		return "target_runtime_unavailable"
	case strings.Contains(msg, "timeout"):
		return "target_timeout"
	case strings.Contains(msg, "config"):
		return "configuration_error"
	default:
		return "baseline_error"
	}
}

func suggestedMCPNextActions(tool string, err error) []string {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "confirm") {
		return []string{"Show the report and responses to the operator", "Ask for explicit confirmation", "Retry with confirm=\"accept <run_id>\" only if the operator approves"}
	}
	if strings.Contains(msg, "timeout") {
		return []string{"Read the run receipt if one was written", "Do not repeat the same full run more than twice", "Ask the operator whether to pin a faster model or raise the timeout"}
	}
	return []string{"Run baseline_doctor", "Report the failing step to the operator", "Propose a repair before changing configuration"}
}

func mcpSchedule(args map[string]any) (any, error) {
	action := stringArg(args, "action", "status")
	switch action {
	case "status":
		return scheduleStatus()
	case "install":
		return installSchedule("", stringArg(args, "at", "09:00"))
	case "remove":
		return removeSchedule()
	case "run":
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		return startAsyncMCPRun("run", args, cfg.Target.Packs)
	default:
		return nil, fmt.Errorf("unknown schedule action %s", action)
	}
}

func mcpBootstrap(args map[string]any) (any, error) {
	action := stringArg(args, "action", "status")
	switch action {
	case "status":
		return currentBootstrapStatus()
	case "defaults":
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		cfg.MemorySeeds = defaultMemorySeeds()
		cfg.MonitorPacks = defaultMonitorPackSelections()
		if err := saveConfig(cfg); err != nil {
			return nil, err
		}
		return map[string]any{"status": "defaults_written", "config_path": configPath(), "enabled_packs": enabledPackIDs(cfg)}, nil
	case "preview":
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		return createBootstrapPreview(cfg)
	case "run":
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		db, err := openDB()
		if err != nil {
			return nil, err
		}
		defer db.Close()
		if err := requireBootstrapPreview(db, scopeKeyForWorkspace(currentWorkspace()), configHash(cfg), stringArg(args, "preview_id", "")); err != nil {
			return nil, err
		}
		run, err := RunBaseline(context.Background(), RunOptions{Mode: "bootstrap", RunAgent: true, Packs: stringArg(args, "packs", "baseline")})
		if err != nil {
			return nil, err
		}
		candidate, err := createBootstrapCandidate(db, run.ID, stringArg(args, "label", "Baseline candidate"), stringArg(args, "notes", ""), scopeKeyForWorkspace(run.Workspace), configHash(cfg))
		if err != nil {
			return nil, err
		}
		return map[string]any{"candidate": candidate, "run": run}, nil
	case "accept":
		runID := stringArg(args, "run_id", "")
		if err := requireMCPConfirmation("accept", runID, 0, stringArg(args, "confirm", "")); err != nil {
			return nil, err
		}
		return acceptCandidateOrRun(runID, stringArg(args, "label", "Good baseline"), stringArg(args, "notes", ""), parseSlot(stringArg(args, "slot", "auto")), true)
	case "reject":
		return withDB(func(db *sql.DB) (any, error) {
			runID := stringArg(args, "run_id", "")
			if runID == "" {
				cfg, err := loadConfig()
				if err != nil {
					return nil, err
				}
				candidate, err := latestBootstrapCandidate(db, scopeKeyForWorkspace(currentWorkspace()), configHash(cfg))
				if err != nil {
					return nil, err
				}
				runID = candidate.RunID
			}
			if err := rejectBootstrapCandidate(db, runID, stringArg(args, "notes", "")); err != nil {
				return nil, err
			}
			return map[string]string{"rejected": runID}, nil
		})
	default:
		return nil, fmt.Errorf("unknown bootstrap action %s", action)
	}
}

func mcpGood(args map[string]any) (any, error) {
	action := stringArg(args, "action", "list")
	switch action {
	case "list":
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		return withDB(func(db *sql.DB) (any, error) {
			return listGoodBaselines(db, scopeKeyForWorkspace(currentWorkspace()), configHash(cfg))
		})
	case "accept":
		runID := stringArg(args, "run_id", "")
		if err := requireMCPConfirmation("accept", runID, 0, stringArg(args, "confirm", "")); err != nil {
			return nil, err
		}
		return acceptCandidateOrRun(runID, stringArg(args, "label", "Good baseline"), stringArg(args, "notes", ""), parseSlot(stringArg(args, "slot", "auto")), false)
	case "replace":
		runID := stringArg(args, "run_id", "")
		slot := parseSlot(stringArg(args, "slot", ""))
		if slot == 0 {
			return nil, fmt.Errorf("replace requires slot 1, 2, or 3")
		}
		if err := requireMCPConfirmation("replace", runID, slot, stringArg(args, "confirm", "")); err != nil {
			return nil, err
		}
		return acceptCandidateOrRun(runID, stringArg(args, "label", "Good baseline"), stringArg(args, "notes", ""), slot, false)
	case "compare":
		return withDB(func(db *sql.DB) (any, error) {
			run, err := latestRun(db)
			if err != nil {
				return nil, err
			}
			return compareToKnownGood(db, run.ID)
		})
	default:
		return nil, fmt.Errorf("unknown Good Baseline action %s", action)
	}
}

func requireMCPConfirmation(action, runID string, slot int, confirm string) error {
	if runID == "" {
		return fmt.Errorf("MCP %s requires explicit run_id and confirmation; use CLI review flow if you want latest candidate defaults", action)
	}
	want := action + " " + runID
	if action == "replace" {
		want = fmt.Sprintf("replace %s slot %d", runID, slot)
	}
	if strings.TrimSpace(confirm) != want {
		return fmt.Errorf("MCP %s requires confirm=%q", action, want)
	}
	return nil
}

func mcpReport(runID string) (any, error) {
	return withDB(func(db *sql.DB) (any, error) {
		var run Run
		var err error
		if runID == "" {
			run, err = latestRun(db)
		} else {
			run, err = runByID(db, runID)
		}
		if errors.Is(err, sql.ErrNoRows) && runID != "" {
			status, statusErr := readRunLifecycleStatus(runID)
			if statusErr == nil {
				return map[string]any{
					"run_status": status,
					"next_actions": []string{
						"If state is running, wait and call baseline_report again",
						"If state is failed, read stderr_path and run baseline_doctor",
					},
				}, nil
			}
		}
		if err != nil {
			return nil, err
		}
		observations, err := observationsForRun(db, run.ID)
		if err != nil {
			return nil, err
		}
		compare, err := compareToKnownGood(db, run.ID)
		if err != nil {
			return nil, err
		}
		artifacts := runArtifactPaths(run.ID)
		reportMarkdown := ""
		responsesMarkdown := ""
		if b, err := os.ReadFile(artifacts.ReportPath); err == nil {
			reportMarkdown = string(b)
		}
		if b, err := os.ReadFile(artifacts.ResponsesPath); err == nil {
			responsesMarkdown = string(b)
		}
		return map[string]any{
			"run":                run,
			"observations":       observations,
			"compare_findings":   compare,
			"artifacts":          artifacts,
			"report_markdown":    reportMarkdown,
			"responses_markdown": responsesMarkdown,
			"next_actions": []string{
				"Show report_markdown and responses_markdown to the operator",
				fmt.Sprintf("Accept only after approval: baseline_accept run_id=%s confirm=%q", run.ID, "accept "+run.ID),
			},
		}, nil
	})
}

func mcpMarkKnownGood(runID, label string) (any, error) {
	return withDB(func(db *sql.DB) (any, error) {
		if runID == "" {
			run, err := latestRun(db)
			if err != nil {
				return nil, err
			}
			runID = run.ID
		}
		if err := markKnownGood(db, runID, label); err != nil {
			return nil, err
		}
		return map[string]string{"marked": runID, "label": label}, nil
	})
}

func mcpConfig(args map[string]any) (any, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if _, ok := args["cloud_sync"]; ok {
		cfg.CloudSync = boolArg(args, "cloud_sync", cfg.CloudSync)
	}
	if url := stringArg(args, "api_base_url", ""); url != "" {
		cfg.APIBaseURL = url
	}
	if err := saveConfig(cfg); err != nil {
		return nil, err
	}
	return map[string]any{
		"version":          cfg.Version,
		"workspace_name":   cfg.WorkspaceName,
		"cloud_sync":       cfg.CloudSync,
		"api_base_url":     cfg.APIBaseURL,
		"api_token_set":    cfg.APIToken != "",
		"allow_raw_output": cfg.AllowRawOutput,
		"packs":            cfg.Packs,
	}, nil
}

func withDB(fn func(*sql.DB) (any, error)) (any, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return fn(db)
}

func stringArg(args map[string]any, key, fallback string) string {
	if args == nil {
		return fallback
	}
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func boolArg(args map[string]any, key string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	if v, ok := args[key].(bool); ok {
		return v
	}
	return fallback
}
