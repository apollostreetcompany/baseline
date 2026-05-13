package baseline

import "time"

type Run struct {
	ID              string        `json:"run_id"`
	Mode            string        `json:"mode"`
	StartedAt       time.Time     `json:"started_at"`
	DurationMS      int64         `json:"duration_ms"`
	Status          string        `json:"status"`
	HealthScore     int           `json:"health_score"`
	Workspace       string        `json:"workspace"`
	AgentKind       string        `json:"agent_kind"`
	CloudSynced     bool          `json:"cloud_synced"`
	RawExported     bool          `json:"raw_exported"`
	RedactionStatus string        `json:"redaction_status"`
	Checks          []CheckResult `json:"checks"`
	Findings        []Finding     `json:"findings"`
}

type CheckResult struct {
	ID         string             `json:"id"`
	CheckID    string             `json:"check_id"`
	Lane       string             `json:"lane"`
	Kind       string             `json:"kind"`
	Status     string             `json:"status"`
	Severity   int                `json:"severity"`
	Score      float64            `json:"score"`
	DurationMS int64              `json:"duration_ms"`
	Finding    string             `json:"finding"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

type Finding struct {
	Severity string `json:"severity"`
	CheckID  string `json:"check_id"`
	Message  string `json:"message"`
	Fix      string `json:"suggested_fix,omitempty"`
}

type Observation struct {
	Key               string
	ValueHash         string
	NumericValue      *float64
	RedactedDisplay   string
	PreviousValueHash string
}

type Config struct {
	Version        int               `json:"version"`
	WorkspaceName  string            `json:"workspace_name"`
	UserFacts      map[string]string `json:"user_facts"`
	AgentCommand   string            `json:"agent_command"`
	CloudSync      bool              `json:"cloud_sync"`
	APIBaseURL     string            `json:"api_base_url"`
	APIToken       string            `json:"api_token"`
	AllowRawOutput bool              `json:"allow_raw_output"`
	Packs          PackConfig        `json:"packs"`
}

type PackConfig struct {
	FactChecks    bool `json:"fact_checks"`
	StyleChecks   bool `json:"style_checks"`
	RepoAwareness bool `json:"repo_awareness"`
	BrowserChecks bool `json:"browser_checks"`
	Custom        bool `json:"custom"`
}

type Question struct {
	ID               string
	Prompt           string
	ExpectedBehavior string
	ExpectedFacts    []string
	Dimension        string
}
