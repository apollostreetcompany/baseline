package baseline

import "time"

type Run struct {
	ID                 string          `json:"run_id"`
	Mode               string          `json:"mode"`
	StartedAt          time.Time       `json:"started_at"`
	DurationMS         int64           `json:"duration_ms"`
	Status             string          `json:"status"`
	HealthScore        int             `json:"health_score"`
	QualityScore       int             `json:"quality_score"`
	SlowScore          int             `json:"slow_score"`
	Workspace          string          `json:"workspace"`
	ScopeKey           string          `json:"scope_key"`
	ConfigHash         string          `json:"config_hash"`
	QuestionSetVersion string          `json:"question_set_version"`
	AgentKind          string          `json:"agent_kind"`
	CloudSynced        bool            `json:"cloud_synced"`
	RawExported        bool            `json:"raw_exported"`
	RedactionStatus    string          `json:"redaction_status"`
	Checks             []CheckResult   `json:"checks"`
	Findings           []Finding       `json:"findings"`
	Artifacts          RunArtifacts    `json:"artifacts,omitempty"`
	Responses          []ProbeResponse `json:"-"`
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

type RunArtifacts struct {
	ReportPath    string `json:"report_path,omitempty"`
	ResponsesPath string `json:"responses_path,omitempty"`
	ReceiptPath   string `json:"receipt_path,omitempty"`
	MetricsPath   string `json:"metrics_path,omitempty"`
	JSONPath      string `json:"json_path,omitempty"`
}

type ProbeResponse struct {
	PackID           string `json:"pack_id"`
	ProbeID          string `json:"probe_id"`
	Dimension        string `json:"dimension"`
	Prompt           string `json:"prompt"`
	ExpectedBehavior string `json:"expected_behavior,omitempty"`
	Output           string `json:"output,omitempty"`
	ScrubbedOutput   string `json:"scrubbed_output,omitempty"`
	Error            string `json:"error,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	DurationMS       int64  `json:"duration_ms"`
	Status           string `json:"status"`
}

type Observation struct {
	Key               string
	ValueHash         string
	NumericValue      *float64
	RedactedDisplay   string
	PreviousValueHash string
}

type Config struct {
	Version        int                    `json:"version"`
	WorkspaceName  string                 `json:"workspace_name"`
	WorkspacePath  string                 `json:"workspace_path,omitempty"`
	Target         BaselineTarget         `json:"target"`
	UserFacts      map[string]string      `json:"user_facts"`
	MemorySeeds    []MemorySeed           `json:"memory_seeds"`
	AgentCommand   string                 `json:"agent_command"`
	CloudSync      bool                   `json:"cloud_sync"`
	APIBaseURL     string                 `json:"api_base_url"`
	APIToken       string                 `json:"api_token"`
	AllowRawOutput bool                   `json:"allow_raw_output"`
	MonitorPacks   []MonitorPackSelection `json:"monitor_packs"`
	Packs          PackConfig             `json:"packs,omitempty"`
}

type BaselineTarget struct {
	Runtime        string `json:"runtime"`
	Entity         string `json:"entity"`
	ModelPolicy    string `json:"model_policy"`
	PinnedModel    string `json:"pinned_model,omitempty"`
	Thinking       string `json:"thinking,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
	Packs          string `json:"packs,omitempty"`
}

type PackConfig struct {
	FactChecks    bool `json:"fact_checks"`
	StyleChecks   bool `json:"style_checks"`
	RepoAwareness bool `json:"repo_awareness"`
	BrowserChecks bool `json:"browser_checks"`
	Custom        bool `json:"custom"`
}

type MonitorPackSelection struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
}

type MemorySeed struct {
	ID          string `json:"id"`
	PackID      string `json:"pack_id"`
	Label       string `json:"label"`
	Value       string `json:"value"`
	Sensitivity string `json:"sensitivity"`
}

type PackRisk struct {
	RequiresAgent      bool `json:"requires_agent"`
	ReadsSelfLog       bool `json:"reads_self_log"`
	MutatesWorkspace   bool `json:"mutates_workspace"`
	CloudExportAllowed bool `json:"cloud_export_allowed"`
}

type MonitorPack struct {
	ID             string     `json:"id"`
	Version        string     `json:"version"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	EnabledDefault bool       `json:"enabled_default"`
	Risk           PackRisk   `json:"risk"`
	Questions      []Question `json:"questions"`
}

type Question struct {
	ID               string   `json:"id"`
	PackID           string   `json:"pack_id"`
	Prompt           string   `json:"prompt"`
	ExpectedBehavior string   `json:"expected_behavior"`
	ExpectedFacts    []string `json:"expected_facts"`
	Dimension        string   `json:"dimension"`
	Risk             PackRisk `json:"risk"`
	EnabledDefault   bool     `json:"enabled_default"`
}

type ProbeMessage struct {
	RunID              string    `json:"run_id"`
	PackID             string    `json:"pack_id"`
	PackVersion        string    `json:"pack_version"`
	ProbeID            string    `json:"probe_id"`
	QuestionSetVersion string    `json:"question_set_version"`
	PromptHash         string    `json:"prompt_hash"`
	ExpectedFactsHash  string    `json:"expected_facts_hash"`
	SessionID          string    `json:"session_id"`
	SystemSendAt       time.Time `json:"system_send_at"`
	BaselineReceivedAt time.Time `json:"baseline_received_at"`
	DurationMS         int64     `json:"duration_ms"`
	TokenStatus        string    `json:"token_status"`
	TokenSource        string    `json:"token_source"`
	InputTokens        *int      `json:"input_tokens,omitempty"`
	OutputTokens       *int      `json:"output_tokens,omitempty"`
	TotalTokens        *int      `json:"total_tokens,omitempty"`
	ContextTokens      *int      `json:"context_tokens,omitempty"`
	Model              string    `json:"model,omitempty"`
	ModelProvider      string    `json:"model_provider,omitempty"`
}

type BootstrapCandidate struct {
	RunID      string `json:"run_id"`
	ScopeKey   string `json:"scope_key"`
	ConfigHash string `json:"config_hash"`
	Status     string `json:"status"`
	Label      string `json:"label"`
	Notes      string `json:"notes"`
	CreatedAt  string `json:"created_at"`
}

type GoodBaseline struct {
	RunID      string `json:"run_id"`
	ScopeKey   string `json:"scope_key"`
	ConfigHash string `json:"config_hash"`
	Slot       int    `json:"slot"`
	Label      string `json:"label"`
	Notes      string `json:"notes"`
	CreatedAt  string `json:"created_at"`
}
