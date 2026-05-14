package baseline

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
			"name":        "baseline_check",
			"description": "Run a local Baseline health check. Fast mode never executes the agent. Full mode runs timed question probes only when run_agent is true.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{
				"mode":      stringProp("fast or full"),
				"run_agent": boolProp("allow executing the configured local agent"),
			}},
		},
		{
			"name":        "baseline_latest",
			"description": "Return the latest local baseline run summary from SQLite.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "baseline_report",
			"description": "Return a run plus redacted observations. Defaults to latest run.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"run_id": stringProp("optional run id")}},
		},
		{
			"name":        "baseline_compare",
			"description": "Compare latest run against the marked known-good run and return changed local state hashes/displays.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "baseline_mark_known_good",
			"description": "Mark a run as known-good after the user accepts its state. Defaults to latest run.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"run_id": stringProp("optional run id"), "label": stringProp("label")}},
		},
		{
			"name":        "baseline_schedule",
			"description": "Install, remove, inspect, or trigger the daily local Baseline self-check. The run action performs a fast check and sync push.",
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
	case "baseline_check":
		mode := stringArg(args, "mode", "fast")
		runAgent := boolArg(args, "run_agent", false)
		payload, err = RunBaseline(context.Background(), RunOptions{Mode: mode, RunAgent: runAgent})
	case "baseline_latest":
		payload, err = withDB(func(db *sql.DB) (any, error) { return latestRun(db) })
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
	case "baseline_mark_known_good":
		payload, err = mcpMarkKnownGood(stringArg(args, "run_id", ""), stringArg(args, "label", "known-good"))
	case "baseline_schedule":
		payload, err = mcpSchedule(args)
	case "baseline_config":
		payload, err = mcpConfig(args)
	case "baseline_scrub_preview":
		out, report := scrubText(stringArg(args, "text", ""))
		payload = map[string]any{"scrubbed": out, "report": report}
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	if err != nil {
		return map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": err.Error()}}}, nil
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(b)}}}, nil
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
		return runScheduledBaseline(context.Background())
	default:
		return nil, fmt.Errorf("unknown schedule action %s", action)
	}
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
		if err != nil {
			return nil, err
		}
		observations, err := observationsForRun(db, run.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"run": run, "observations": observations}, nil
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
