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
	"strconv"
	"strings"
	"time"
)

type BootstrapPreview struct {
	PreviewID  string        `json:"preview_id,omitempty"`
	CreatedAt  string        `json:"created_at,omitempty"`
	ConfigPath string        `json:"config_path"`
	ScopeKey   string        `json:"scope_key"`
	ConfigHash string        `json:"config_hash"`
	Packs      []MonitorPack `json:"packs"`
	Questions  []Question    `json:"questions"`
	Next       string        `json:"next,omitempty"`
}

type BootstrapStatus struct {
	NeedsBootstrap  bool                `json:"needs_bootstrap"`
	ScopeKey        string              `json:"scope_key"`
	ConfigHash      string              `json:"config_hash"`
	ConfigPath      string              `json:"config_path"`
	GoodBaselines   []GoodBaseline      `json:"good_baselines"`
	LatestCandidate *BootstrapCandidate `json:"latest_candidate,omitempty"`
}

func cmdBootstrap(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return cmdBootstrapRoot(args, stdout, stderr)
	}
	switch args[0] {
	case "status":
		return writeBootstrapStatus(stdout, stderr)
	case "defaults":
		return cmdBootstrapDefaults(args[1:], stdout, stderr)
	case "preview", "questions":
		return cmdBootstrapPreview(args[1:], stdout, stderr)
	case "run":
		return cmdBootstrapRun(ctx, args[1:], stdout, stderr)
	case "accept":
		return cmdBootstrapAccept(args[1:], stdout, stderr)
	case "reject":
		return cmdBootstrapReject(args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: baseline bootstrap status|defaults|preview|run|accept|reject")
		return 2
	}
}

func cmdBootstrapRoot(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	openclaw := fs.Bool("openclaw", false, "register Baseline MCP with OpenClaw")
	syncURL := fs.String("sync-url", "", "optional cloud sync URL")
	syncToken := fs.String("sync-token", "", "optional cloud sync token")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(cfg.MemorySeeds) == 0 {
		cfg.MemorySeeds = defaultMemorySeeds()
	}
	if len(cfg.MonitorPacks) == 0 {
		cfg.MonitorPacks = defaultMonitorPackSelections()
	}
	if *syncURL != "" {
		cfg.APIBaseURL = *syncURL
	}
	if *syncToken != "" {
		cfg.CloudSync = true
		cfg.APIToken = *syncToken
	}
	if err := saveConfig(cfg); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	db.Close()
	if _, err := os.Stat(redactionPath()); errors.Is(err, os.ErrNotExist) {
		_ = atomicWrite(redactionPath(), []byte("# Baseline local redaction rules. Cloud sync exports summaries unless allow_raw_output is true.\n"), 0o600)
	}
	if *openclaw {
		if err := registerOpenClaw(); err != nil {
			fmt.Fprintf(stderr, "OpenClaw registration failed: %v\n", err)
			return 1
		}
	}
	status, err := currentBootstrapStatus()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, map[string]any{
		"status":          "ready",
		"config_path":     configPath(),
		"database_path":   dbPath(),
		"openclaw":        *openclaw,
		"sync_enabled":    cfg.CloudSync,
		"needs_bootstrap": status.NeedsBootstrap,
		"enabled_packs":   enabledPackIDs(cfg),
		"next":            "baseline bootstrap preview, then baseline bootstrap run",
	})
}

func writeBootstrapStatus(stdout, stderr io.Writer) int {
	status, err := currentBootstrapStatus()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, status)
}

func cmdBootstrapDefaults(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bootstrap defaults", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cfg.MemorySeeds = defaultMemorySeeds()
	cfg.MonitorPacks = defaultMonitorPackSelections()
	if err := saveConfig(cfg); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	preview := bootstrapPreview(cfg)
	return writeJSON(stdout, stderr, map[string]any{
		"status":         "defaults_written",
		"config_path":    configPath(),
		"enabled_packs":  enabledPackIDs(cfg),
		"question_count": len(preview.Questions),
	})
}

func cmdBootstrapPreview(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bootstrap preview", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	preview, err := createBootstrapPreview(cfg)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, preview)
}

func cmdBootstrapRun(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bootstrap run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	label := fs.String("label", "Baseline candidate", "candidate label")
	notes := fs.String("notes", "", "candidate notes")
	agentCommand := fs.String("agent-command", "", "test-only/custom agent command; prompt is BASELINE_PROMPT")
	packs := fs.String("packs", "baseline", "question packs to run: baseline, enabled, all, or comma-separated pack ids")
	previewID := fs.String("preview-id", "", "optional preview id from baseline bootstrap preview")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	if err := requireBootstrapPreview(db, scopeKeyForWorkspace(currentWorkspace()), configHash(cfg), *previewID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	run, err := RunBaseline(ctx, RunOptions{Mode: "bootstrap", RunAgent: true, AgentCommand: *agentCommand, Packs: *packs})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	candidate, err := createBootstrapCandidate(db, run.ID, *label, *notes, scopeKeyForWorkspace(run.Workspace), configHash(cfg))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, map[string]any{
		"candidate": candidate,
		"run":       run,
		"next":      "Review this candidate, then run baseline bootstrap accept " + run.ID,
	})
}

func cmdBootstrapAccept(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bootstrap accept", flag.ContinueOnError)
	fs.SetOutput(stderr)
	label := fs.String("label", "Good baseline", "Good Baseline label")
	notes := fs.String("notes", "", "acceptance notes")
	slotValue := fs.String("slot", "auto", "Good Baseline slot: auto, 1, 2, or 3")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	good, err := acceptCandidateOrRun(fs.Arg(0), *label, *notes, parseSlot(*slotValue), true)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, good)
}

func cmdBootstrapReject(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bootstrap reject", flag.ContinueOnError)
	fs.SetOutput(stderr)
	notes := fs.String("notes", "", "rejection notes")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	runID := fs.Arg(0)
	if runID == "" {
		candidate, err := latestBootstrapCandidate(db, scopeKeyForWorkspace(currentWorkspace()), configHash(cfg))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		runID = candidate.RunID
	}
	if err := rejectBootstrapCandidate(db, runID, *notes); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, map[string]string{"rejected": runID})
}

func cmdGood(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: baseline good list|accept|replace|compare")
		return 2
	}
	switch args[0] {
	case "list":
		return cmdGoodList(stdout, stderr)
	case "accept":
		return cmdGoodAccept(args[1:], stdout, stderr, false)
	case "replace":
		return cmdGoodAccept(args[1:], stdout, stderr, true)
	case "compare":
		return cmdGoodCompare(args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: baseline good list|accept|replace|compare")
		return 2
	}
}

func cmdGoodList(stdout, stderr io.Writer) int {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	goods, err := listGoodBaselines(db, scopeKeyForWorkspace(currentWorkspace()), configHash(cfg))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, goods)
}

func cmdGoodAccept(args []string, stdout, stderr io.Writer, replace bool) int {
	fs := flag.NewFlagSet("good accept", flag.ContinueOnError)
	fs.SetOutput(stderr)
	label := fs.String("label", "Good baseline", "Good Baseline label")
	notes := fs.String("notes", "", "notes")
	slotValue := fs.String("slot", "auto", "Good Baseline slot")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	slot := parseSlot(*slotValue)
	if replace && slot == 0 {
		fmt.Fprintln(stderr, "baseline good replace requires --slot 1, 2, or 3")
		return 2
	}
	good, err := acceptCandidateOrRun(fs.Arg(0), *label, *notes, slot, false)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, good)
}

func cmdGoodCompare(args []string, stdout, stderr io.Writer) int {
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer db.Close()
	runID := ""
	if len(args) > 0 {
		runID = args[0]
	} else {
		run, err := latestRun(db)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		runID = run.ID
	}
	findings, err := compareToKnownGood(db, runID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return writeJSON(stdout, stderr, findings)
}

func cmdKnownGood(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: baseline known-good mark|list")
		return 2
	}
	switch args[0] {
	case "mark":
		rewritten := append([]string{"accept"}, args[1:]...)
		return cmdGood(rewritten, stdout, stderr)
	case "list":
		return cmdGood([]string{"list"}, stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: baseline known-good mark|list")
		return 2
	}
}

func currentBootstrapStatus() (BootstrapStatus, error) {
	cfg, err := loadConfig()
	if err != nil {
		return BootstrapStatus{}, err
	}
	db, err := openDB()
	if err != nil {
		return BootstrapStatus{}, err
	}
	defer db.Close()
	scopeKey := scopeKeyForWorkspace(currentWorkspace())
	cfgHash := configHash(cfg)
	goods, err := listGoodBaselines(db, scopeKey, cfgHash)
	if err != nil {
		return BootstrapStatus{}, err
	}
	status := BootstrapStatus{
		NeedsBootstrap: len(goods) == 0,
		ScopeKey:       scopeKey,
		ConfigHash:     cfgHash,
		ConfigPath:     configPath(),
		GoodBaselines:  goods,
	}
	candidate, err := latestBootstrapCandidate(db, scopeKey, cfgHash)
	if err == nil {
		status.LatestCandidate = &candidate
	} else if !errors.Is(err, sql.ErrNoRows) {
		return BootstrapStatus{}, err
	}
	return status, nil
}

func bootstrapPreview(cfg Config) BootstrapPreview {
	enabled := enabledMonitorPacks(cfg)
	var packs []MonitorPack
	var questions []Question
	for _, pack := range canonicalMonitorPacks(configFacts(cfg)) {
		pack.EnabledDefault = enabled[pack.ID]
		packs = append(packs, pack)
		if !enabled[pack.ID] {
			continue
		}
		questions = append(questions, pack.Questions...)
	}
	return BootstrapPreview{
		ConfigPath: configPath(),
		ScopeKey:   scopeKeyForWorkspace(currentWorkspace()),
		ConfigHash: configHash(cfg),
		Packs:      packs,
		Questions:  questions,
	}
}

func createBootstrapPreview(cfg Config) (BootstrapPreview, error) {
	preview := bootstrapPreview(cfg)
	preview.PreviewID = "preview_" + newRunID()
	preview.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	preview.Next = "Review the questions, then run baseline bootstrap run --preview-id " + preview.PreviewID
	db, err := openDB()
	if err != nil {
		return BootstrapPreview{}, err
	}
	defer db.Close()
	details, _ := json.Marshal(map[string]any{
		"question_set_version": questionSetVersion,
		"question_count":       len(preview.Questions),
	})
	_, err = db.Exec(`INSERT INTO consent_events (action, run_id, scope_key, config_hash, details_json, created_at)
		VALUES ('bootstrap.preview', ?, ?, ?, ?, ?)`, preview.PreviewID, preview.ScopeKey, preview.ConfigHash, string(details), preview.CreatedAt)
	if err != nil {
		return BootstrapPreview{}, err
	}
	return preview, nil
}

func requireBootstrapPreview(db *sql.DB, scopeKey, cfgHash, previewID string) error {
	cutoff := time.Now().Add(-2 * time.Hour)
	query := `SELECT created_at FROM consent_events
		WHERE action = 'bootstrap.preview' AND scope_key = ? AND config_hash = ?
		ORDER BY created_at DESC LIMIT 1`
	args := []any{scopeKey, cfgHash}
	if strings.TrimSpace(previewID) != "" {
		query = `SELECT created_at FROM consent_events
			WHERE action = 'bootstrap.preview' AND run_id = ? AND scope_key = ? AND config_hash = ?
			ORDER BY created_at DESC LIMIT 1`
		args = []any{previewID, scopeKey, cfgHash}
	}
	var created string
	err := db.QueryRow(query, args...).Scan(&created)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("preview required before sending agent probes; run baseline bootstrap preview, review the question set, then run baseline bootstrap run")
	}
	if err != nil {
		return err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return fmt.Errorf("stored bootstrap preview timestamp is invalid; run baseline bootstrap preview again")
	}
	if createdAt.Before(cutoff) {
		return fmt.Errorf("bootstrap preview expired; run baseline bootstrap preview again before sending agent probes")
	}
	return nil
}

func acceptCandidateOrRun(runID, label, notes string, slot int, requireCandidate bool) (GoodBaseline, error) {
	cfg, err := loadConfig()
	if err != nil {
		return GoodBaseline{}, err
	}
	db, err := openDB()
	if err != nil {
		return GoodBaseline{}, err
	}
	defer db.Close()
	scopeKey := scopeKeyForWorkspace(currentWorkspace())
	cfgHash := configHash(cfg)
	if runID == "" {
		candidate, err := latestBootstrapCandidate(db, scopeKey, cfgHash)
		if err != nil {
			if requireCandidate && errors.Is(err, sql.ErrNoRows) {
				return GoodBaseline{}, fmt.Errorf("no bootstrap candidate exists; run baseline bootstrap run first")
			}
			run, runErr := latestRun(db)
			if runErr != nil {
				return GoodBaseline{}, err
			}
			runID = run.ID
		} else {
			runID = candidate.RunID
		}
	}
	run, err := runByID(db, runID)
	if err != nil {
		return GoodBaseline{}, err
	}
	if run.ScopeKey != "" {
		scopeKey = run.ScopeKey
	}
	if run.ConfigHash != "" {
		cfgHash = run.ConfigHash
	}
	if requireCandidate {
		var status string
		err := db.QueryRow(`SELECT status FROM bootstrap_candidates WHERE run_id = ? AND scope_key = ? AND config_hash = ?`, runID, scopeKey, cfgHash).Scan(&status)
		if err != nil {
			return GoodBaseline{}, fmt.Errorf("run %s is not a bootstrap candidate for this workspace/config", runID)
		}
		if status == "rejected" {
			return GoodBaseline{}, fmt.Errorf("run %s was rejected and cannot be accepted", runID)
		}
	}
	return acceptGoodBaseline(db, runID, label, notes, slot, scopeKey, cfgHash)
}

func parseSlot(value string) int {
	if value == "" || strings.EqualFold(value, "auto") {
		return 0
	}
	slot, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return slot
}

func enabledPackIDs(cfg Config) []string {
	var ids []string
	for _, pack := range cfg.MonitorPacks {
		if pack.Enabled {
			ids = append(ids, pack.ID)
		}
	}
	return ids
}
