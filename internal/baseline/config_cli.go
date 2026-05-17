package baseline

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func cmdConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return cmdConfigOverview(stdout, stderr)
	}
	switch args[0] {
	case "file":
		fmt.Fprintln(stdout, configPath())
		return 0
	case "show":
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, stderr, publicConfig(cfg))
	case "get":
		return cmdConfigGet(args[1:], stdout, stderr)
	case "set":
		return cmdConfigSet(args[1:], stdout, stderr)
	case "patch":
		return cmdConfigPatch(args[1:], stdout, stderr)
	case "unset":
		return cmdConfigUnset(args[1:], stdout, stderr)
	case "validate":
		return cmdConfigValidate(args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: baseline config file|show|get|set|patch|unset|validate")
		return 2
	}
}

func cmdConfigOverview(stdout, stderr io.Writer) int {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	status, _ := currentBootstrapStatus()
	fmt.Fprintf(stdout, "Baseline config\n")
	fmt.Fprintf(stdout, "  file: %s\n", configPath())
	fmt.Fprintf(stdout, "  workspace: %s\n", cfg.WorkspaceName)
	fmt.Fprintf(stdout, "  target: %s %s (%s)\n", cfg.Target.Runtime, cfg.Target.Entity, targetModelDisplay(cfg.Target))
	fmt.Fprintf(stdout, "  cloud_sync: %t\n", cfg.CloudSync)
	fmt.Fprintf(stdout, "  token_set: %t\n", cfg.APIToken != "")
	fmt.Fprintf(stdout, "  enabled_packs: %s\n", strings.Join(enabledPackIDs(cfg), ", "))
	fmt.Fprintf(stdout, "  good_baselines: %d\n", len(status.GoodBaselines))
	fmt.Fprintln(stdout, "\nCommands: baseline config get <path>, set <path> <value>, patch --file config.json, validate --json")
	return 0
}

func cmdConfigGet(args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	var positional []string
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			continue
		}
		positional = append(positional, arg)
	}
	if len(positional) != 1 {
		fmt.Fprintln(stderr, "usage: baseline config get <path> [--json]")
		return 2
	}
	cfgMap, err := configMap()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	value, ok := getPath(cfgMap, positional[0])
	if !ok {
		fmt.Fprintf(stderr, "config path not found: %s\n", positional[0])
		return 1
	}
	value = redactConfigValue(positional[0], value)
	if jsonOut {
		return writeJSON(stdout, stderr, value)
	}
	fmt.Fprintf(stdout, "%v\n", value)
	return 0
}

func cmdConfigSet(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	fs.SetOutput(stderr)
	strictJSON := fs.Bool("strict-json", false, "parse value as JSON")
	merge := fs.Bool("merge", false, "merge object value into existing object")
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(stderr, "usage: baseline config set <path> <value> [--strict-json] [--merge] [--dry-run]")
		return 2
	}
	cfgMap, err := configMap()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	value, err := parseConfigValue(fs.Arg(1), *strictJSON)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *merge {
		current, _ := getPath(cfgMap, fs.Arg(0))
		dst, dstOK := current.(map[string]any)
		src, srcOK := value.(map[string]any)
		if !dstOK || !srcOK {
			fmt.Fprintln(stderr, "--merge requires current and new values to be JSON objects")
			return 1
		}
		for k, v := range src {
			dst[k] = v
		}
		value = dst
	}
	setPath(cfgMap, fs.Arg(0), value)
	if *dryRun {
		return writeJSON(stdout, stderr, map[string]any{"dry_run": true, "path": fs.Arg(0), "value": redactConfigValue(fs.Arg(0), value)})
	}
	if err := saveConfigMap(cfgMap); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Updated %s\n", fs.Arg(0))
	return 0
}

func cmdConfigPatch(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config patch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "", "JSON patch file containing config fields to merge")
	stdin := fs.Bool("stdin", false, "read patch JSON from stdin")
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var b []byte
	var err error
	switch {
	case *file != "":
		b, err = os.ReadFile(*file)
	case *stdin:
		b, err = io.ReadAll(os.Stdin)
	default:
		fmt.Fprintln(stderr, "usage: baseline config patch --file <path>|--stdin [--dry-run]")
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var patch map[string]any
	if err := json.Unmarshal(b, &patch); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cfgMap, err := configMap()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	mergeMap(cfgMap, patch)
	if *dryRun {
		return writeJSON(stdout, stderr, publicConfigMap(cfgMap))
	}
	if err := saveConfigMap(cfgMap); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "Config patched.")
	return 0
}

func cmdConfigUnset(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config unset", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: baseline config unset <path> [--dry-run]")
		return 2
	}
	cfgMap, err := configMap()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !unsetPath(cfgMap, fs.Arg(0)) {
		fmt.Fprintf(stderr, "config path not found: %s\n", fs.Arg(0))
		return 1
	}
	if *dryRun {
		return writeJSON(stdout, stderr, map[string]any{"dry_run": true, "unset": fs.Arg(0)})
	}
	if err := saveConfigMap(cfgMap); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Unset %s\n", fs.Arg(0))
	return 0
}

func cmdConfigValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("config validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	issues := validateConfig(cfg)
	payload := map[string]any{"ok": len(issues) == 0, "issues": issues, "config_path": configPath()}
	if *jsonOut {
		return writeJSON(stdout, stderr, payload)
	}
	if len(issues) == 0 {
		fmt.Fprintln(stdout, "Config is valid.")
		return 0
	}
	for _, issue := range issues {
		fmt.Fprintf(stdout, "- %s\n", issue)
	}
	return 1
}

func configMap() (map[string]any, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func saveConfigMap(m map[string]any) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return err
	}
	if len(cfg.MonitorPacks) == 0 {
		cfg.MonitorPacks = defaultMonitorPackSelections()
	}
	if len(cfg.MemorySeeds) == 0 {
		cfg.MemorySeeds = defaultMemorySeeds()
	}
	cfg = normalizeConfig(cfg)
	return saveConfig(cfg)
}

func parseConfigValue(value string, strict bool) (any, error) {
	if strict {
		var parsed any
		if err := json.Unmarshal([]byte(value), &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	}
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}
	return value, nil
}

func validateConfig(cfg Config) []string {
	var issues []string
	if cfg.Version == 0 {
		issues = append(issues, "version is missing")
	}
	if cfg.WorkspaceName == "" {
		issues = append(issues, "workspace_name is missing")
	}
	switch cfg.Target.Runtime {
	case "openclaw", "custom":
	default:
		issues = append(issues, "target.runtime must be openclaw or custom")
	}
	if cfg.Target.Entity == "" {
		issues = append(issues, "target.entity is missing")
	}
	switch cfg.Target.ModelPolicy {
	case "follow_current":
	case "pinned":
		if strings.TrimSpace(cfg.Target.PinnedModel) == "" {
			issues = append(issues, "target.model_policy is pinned but target.pinned_model is empty")
		}
	default:
		issues = append(issues, "target.model_policy must be follow_current or pinned")
	}
	if cfg.Target.TimeoutSeconds < 30 || cfg.Target.TimeoutSeconds > 900 {
		issues = append(issues, "target.timeout_seconds must be between 30 and 900")
	}
	known := map[string]bool{}
	for _, pack := range canonicalMonitorPacks(configFacts(cfg)) {
		known[pack.ID] = true
	}
	for _, selection := range cfg.MonitorPacks {
		if !known[selection.ID] {
			issues = append(issues, "unknown monitor pack: "+selection.ID)
		}
	}
	if cfg.CloudSync && cfg.APIToken == "" {
		issues = append(issues, "cloud_sync is true but api_token is not set")
	}
	return issues
}

func publicConfig(cfg Config) map[string]any {
	b, _ := json.Marshal(cfg)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return publicConfigMap(m)
}

func publicConfigMap(m map[string]any) map[string]any {
	copy := map[string]any{}
	for k, v := range m {
		if strings.Contains(strings.ToLower(k), "token") {
			copy[k+"_set"] = fmt.Sprint(v) != ""
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			copy[k] = publicConfigMap(nested)
			continue
		}
		copy[k] = v
	}
	return copy
}

func redactConfigValue(path string, value any) any {
	if strings.Contains(strings.ToLower(path), "token") {
		return map[string]bool{"token_set": fmt.Sprint(value) != ""}
	}
	return value
}

func getPath(m map[string]any, path string) (any, bool) {
	if packID, field, ok := monitorPackPath(path); ok {
		return getMonitorPackField(m, packID, field)
	}
	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = obj[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func setPath(m map[string]any, path string, value any) {
	if packID, field, ok := monitorPackPath(path); ok {
		setMonitorPackField(m, packID, field, value)
		return
	}
	parts := strings.Split(path, ".")
	current := m
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func monitorPackPath(path string) (string, string, bool) {
	parts := strings.Split(path, ".")
	if len(parts) == 3 && parts[0] == "monitor_packs" {
		return parts[1], parts[2], true
	}
	return "", "", false
}

func getMonitorPackField(m map[string]any, packID, field string) (any, bool) {
	items, ok := m["monitor_packs"].([]any)
	if !ok {
		return nil, false
	}
	for _, item := range items {
		pack, ok := item.(map[string]any)
		if ok && pack["id"] == packID {
			value, ok := pack[field]
			return value, ok
		}
	}
	return nil, false
}

func setMonitorPackField(m map[string]any, packID, field string, value any) {
	items, _ := m["monitor_packs"].([]any)
	for _, item := range items {
		pack, ok := item.(map[string]any)
		if ok && pack["id"] == packID {
			pack[field] = value
			m["monitor_packs"] = items
			return
		}
	}
	items = append(items, map[string]any{
		"id":      packID,
		"version": questionSetVersion,
		field:     value,
	})
	m["monitor_packs"] = items
}

func unsetPath(m map[string]any, path string) bool {
	parts := strings.Split(path, ".")
	current := m
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			return false
		}
		current = next
	}
	_, ok := current[parts[len(parts)-1]]
	delete(current, parts[len(parts)-1])
	return ok
}

func mergeMap(dst, src map[string]any) {
	for k, v := range src {
		srcObj, srcOK := v.(map[string]any)
		dstObj, dstOK := dst[k].(map[string]any)
		if srcOK && dstOK {
			mergeMap(dstObj, srcObj)
			continue
		}
		dst[k] = v
	}
}
