package baseline

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

func baseDir() string {
	if v := os.Getenv("BASELINE_HOME"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), ".baseline")
}

func dbPath() string {
	return filepath.Join(baseDir(), "baseline.db")
}

func configPath() string {
	return filepath.Join(baseDir(), "config.json")
}

func redactionPath() string {
	return filepath.Join(baseDir(), "redaction.toml")
}

func ensureRedactionFile() error {
	if _, err := os.Stat(redactionPath()); errors.Is(err, os.ErrNotExist) {
		return atomicWrite(redactionPath(), []byte("# Baseline local redaction rules. Cloud sync exports summaries unless allow_raw_output is true.\n"), 0o600)
	} else if err != nil {
		return err
	}
	return nil
}

func bootstrapContractPath() string {
	return filepath.Join(baseDir(), "BOOTSTRAP.md")
}

func ensureDirs() error {
	for _, dir := range []string{baseDir(), filepath.Join(baseDir(), "reports"), filepath.Join(baseDir(), "raw"), runLifecycleDir()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		Version:       1,
		WorkspaceName: "baseline-local",
		Target: BaselineTarget{
			Runtime:        "openclaw",
			Entity:         "agent:main",
			ModelPolicy:    "follow_current",
			TimeoutSeconds: 240,
			Packs:          "baseline",
		},
		UserFacts:    defaultConfigSeeds(),
		MemorySeeds:  defaultMemorySeeds(),
		CloudSync:    false,
		APIBaseURL:   "https://baseline-ai.ryan-borker.workers.dev",
		MonitorPacks: defaultMonitorPackSelections(),
		Packs: PackConfig{
			FactChecks:    true,
			StyleChecks:   true,
			RepoAwareness: true,
			BrowserChecks: false,
			Custom:        false,
		},
	}
}

func loadConfig() (Config, error) {
	var cfg Config
	b, err := os.ReadFile(configPath())
	if errors.Is(err, os.ErrNotExist) {
		cfg = defaultConfig()
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	return normalizeConfig(cfg), nil
}

func saveConfig(cfg Config) error {
	if err := ensureDirs(); err != nil {
		return err
	}
	cfg = normalizeConfig(cfg)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(configPath(), b, 0o600)
}

func normalizeConfig(cfg Config) Config {
	defaults := defaultConfig()
	if cfg.Version == 0 {
		cfg.Version = defaults.Version
	}
	if cfg.WorkspaceName == "" {
		cfg.WorkspaceName = defaults.WorkspaceName
	}
	if cfg.Target.Runtime == "" {
		cfg.Target.Runtime = defaults.Target.Runtime
	}
	if cfg.Target.Entity == "" {
		cfg.Target.Entity = defaults.Target.Entity
	}
	if cfg.Target.ModelPolicy == "" {
		cfg.Target.ModelPolicy = defaults.Target.ModelPolicy
	}
	if cfg.Target.TimeoutSeconds == 0 {
		cfg.Target.TimeoutSeconds = defaults.Target.TimeoutSeconds
	}
	if cfg.Target.Packs == "" {
		cfg.Target.Packs = defaults.Target.Packs
	}
	if cfg.UserFacts == nil {
		cfg.UserFacts = defaults.UserFacts
	}
	if len(cfg.MemorySeeds) == 0 {
		cfg.MemorySeeds = defaults.MemorySeeds
	}
	if len(cfg.MonitorPacks) == 0 {
		cfg.MonitorPacks = defaults.MonitorPacks
	}
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = defaults.APIBaseURL
	}
	return cfg
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
