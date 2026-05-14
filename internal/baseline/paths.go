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

func ensureDirs() error {
	for _, dir := range []string{baseDir(), filepath.Join(baseDir(), "reports"), filepath.Join(baseDir(), "raw")} {
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
		UserFacts:     defaultConfigSeeds(),
		MemorySeeds:   defaultMemorySeeds(),
		CloudSync:     false,
		APIBaseURL:    "https://baseline-ai.ryan-borker.workers.dev",
		MonitorPacks:  defaultMonitorPackSelections(),
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
	if cfg.UserFacts == nil {
		cfg.UserFacts = defaultConfig().UserFacts
	}
	if len(cfg.MemorySeeds) == 0 {
		cfg.MemorySeeds = defaultMemorySeeds()
	}
	if len(cfg.MonitorPacks) == 0 {
		cfg.MonitorPacks = defaultMonitorPackSelections()
	}
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = defaultConfig().APIBaseURL
	}
	return cfg, nil
}

func saveConfig(cfg Config) error {
	if err := ensureDirs(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(configPath(), b, 0o600)
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
