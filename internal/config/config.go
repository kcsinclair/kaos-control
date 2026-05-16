// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// OllamaInstance is one registered Ollama server shared across all projects.
type OllamaInstance struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key,omitempty"`
}

// App is the top-level application configuration (install-dir/config.yaml).
type App struct {
	Server          ServerConfig     `yaml:"server"`
	Auth            AuthConfig       `yaml:"auth"`
	ProjectsDir     string           `yaml:"projects_dir"`
	Limits          LimitsConfig     `yaml:"limits"`
	DataDir         string           `yaml:"data_dir"` // where app DBs live; defaults to projects_dir/../data
	OllamaInstances []OllamaInstance `yaml:"ollama_instances,omitempty"`
	Agent           AppAgentConfig   `yaml:"agent"`
}

type ServerConfig struct {
	Listen     string    `yaml:"listen"`
	TLS        TLSConfig `yaml:"tls"`
	PublicHost string    `yaml:"public_host"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type AuthConfig struct {
	Method     string        `yaml:"method"`      // "local" (v1 only)
	SessionTTL time.Duration `yaml:"session_ttl"` // e.g. "24h"
}

type LimitsConfig struct {
	MaxConcurrentAgents        int `yaml:"max_concurrent_agents"`
	MaxConcurrentSchedulerJobs int `yaml:"max_concurrent_scheduler_jobs"`
	SchedulerRunRetentionDays  int `yaml:"scheduler_run_retention_days"`
}

// AppAgentConfig holds app-level agent runtime settings that apply to every project.
type AppAgentConfig struct {
	// InitEventTimeoutSeconds is the maximum time (in seconds) to wait for the
	// Claude Code system/init event before declaring the run failed with reason
	// "precheck_timeout". Default: 10.
	InitEventTimeoutSeconds int `yaml:"init_event_timeout_seconds,omitempty"`

	// RequireBypassPermissions controls whether agent runs are rejected when the
	// Claude Code process reports a permission mode other than bypassPermissions.
	// Defaults to true. Set to false to allow runs in any permission mode (escape
	// hatch for environments where bypass mode cannot be enabled).
	RequireBypassPermissions *bool `yaml:"require_bypass_permissions,omitempty"`
}

func defaultApp() App {
	requireBypass := true
	return App{
		Server: ServerConfig{
			Listen: ":8042",
		},
		Auth: AuthConfig{
			Method:     "local",
			SessionTTL: 30 * 24 * time.Hour,
		},
		Limits: LimitsConfig{
			MaxConcurrentAgents:        4,
			MaxConcurrentSchedulerJobs: 2,
			SchedulerRunRetentionDays:  90,
		},
		Agent: AppAgentConfig{
			InitEventTimeoutSeconds:  10,
			RequireBypassPermissions: &requireBypass,
		},
	}
}

// LoadApp reads the app-level config file, applying defaults for missing fields.
// If the file does not exist it is created with the defaults before returning.
func LoadApp(path string) (*App, error) {
	cfg := defaultApp()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading app config: %w", err)
	}
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing app config: %w", err)
		}
	} else {
		// File does not exist: apply path-relative defaults and persist so the
		// user has a concrete file to edit on the next run.
		if cfg.ProjectsDir == "" {
			cfg.ProjectsDir = filepath.Join(filepath.Dir(path), "projects")
		}
		if cfg.DataDir == "" {
			cfg.DataDir = filepath.Join(filepath.Dir(path), "data")
		}
		if err2 := SaveApp(path, cfg); err2 != nil {
			return nil, fmt.Errorf("creating default app config at %s: %w", path, err2)
		}
	}

	if err := validateApp(&cfg); err != nil {
		return nil, err
	}

	if cfg.ProjectsDir == "" {
		cfg.ProjectsDir = filepath.Join(filepath.Dir(path), "projects")
	}
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(filepath.Dir(path), "data")
	}

	if err := os.MkdirAll(cfg.ProjectsDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating projects dir %s: %w", cfg.ProjectsDir, err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating data dir %s: %w", cfg.DataDir, err)
	}

	return &cfg, nil
}

func validateApp(cfg *App) error {
	if cfg.Server.Listen == "" {
		return fmt.Errorf("server.listen must not be empty")
	}
	if cfg.Auth.Method != "local" {
		return fmt.Errorf("auth.method %q is not supported in v1 (only \"local\")", cfg.Auth.Method)
	}
	if cfg.Auth.SessionTTL <= 0 {
		return fmt.Errorf("auth.session_ttl must be positive")
	}
	if cfg.Limits.MaxConcurrentAgents <= 0 {
		cfg.Limits.MaxConcurrentAgents = 4
	}
	if cfg.Limits.MaxConcurrentSchedulerJobs <= 0 {
		cfg.Limits.MaxConcurrentSchedulerJobs = 2
	}
	if cfg.Limits.SchedulerRunRetentionDays <= 0 {
		cfg.Limits.SchedulerRunRetentionDays = 90
	}
	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("server.tls.cert_file and server.tls.key_file are required when TLS is enabled")
		}
	}
	// Agent precheck defaults.
	if cfg.Agent.InitEventTimeoutSeconds <= 0 {
		cfg.Agent.InitEventTimeoutSeconds = 10
	}
	if cfg.Agent.RequireBypassPermissions == nil {
		v := true
		cfg.Agent.RequireBypassPermissions = &v
	}

	seen := make(map[string]bool, len(cfg.OllamaInstances))
	for i, inst := range cfg.OllamaInstances {
		if inst.Name == "" {
			return fmt.Errorf("ollama_instances[%d]: name must not be empty", i)
		}
		if seen[inst.Name] {
			return fmt.Errorf("ollama_instances: duplicate name %q", inst.Name)
		}
		seen[inst.Name] = true
		if inst.BaseURL == "" {
			return fmt.Errorf("ollama_instances[%d] %q: base_url must not be empty", i, inst.Name)
		}
		u, err := url.ParseRequestURI(inst.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("ollama_instances[%d] %q: base_url %q is not a valid http/https URL", i, inst.Name, inst.BaseURL)
		}
	}
	return nil
}

// SaveApp writes the app config atomically (temp file + rename) to path.
func SaveApp(path string, cfg App) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshalling app config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing tmp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming tmp config: %w", err)
	}
	return nil
}

// ProjectEntry is one registration record from projects_dir/*.yaml.
type ProjectEntry struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Owner       string `yaml:"owner"`
}

// LoadProjectRegistry enumerates *.yaml files under dir and returns parsed entries.
func LoadProjectRegistry(dir string) ([]*ProjectEntry, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("globbing projects dir: %w", err)
	}

	var entries []*ProjectEntry
	for _, path := range matches {
		e, err := loadProjectEntry(path)
		if err != nil {
			return nil, fmt.Errorf("loading project entry %s: %w", path, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func loadProjectEntry(path string) (*ProjectEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var e ProjectEntry
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	if e.Name == "" {
		return nil, fmt.Errorf("project entry missing required field: name")
	}
	if e.Path == "" {
		return nil, fmt.Errorf("project entry %q missing required field: path", e.Name)
	}
	return &e, nil
}

// SaveProjectEntry atomically writes a project entry to projects_dir/<name>.yaml.
// It uses os.CreateTemp + os.Rename so that a crash mid-write never leaves a
// corrupt destination file and concurrent writes to different projects do not
// interfere with each other.
func SaveProjectEntry(dir string, e *ProjectEntry) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating projects dir: %w", err)
	}
	data, err := yaml.Marshal(e)
	if err != nil {
		return err
	}
	dest := filepath.Join(dir, e.Name+".yaml")
	tmp, err := os.CreateTemp(dir, e.Name+"-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("creating temp project entry: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing tmp project entry: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("chmod tmp project entry: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing tmp project entry: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("renaming tmp project entry: %w", err)
	}
	return nil
}

// ValidateProjectName returns an error if name is not a valid project slug:
// lowercase alphanumeric and hyphens only, 3–80 characters.
func ValidateProjectName(name string) error {
	if len(name) < 3 {
		return fmt.Errorf("name must be at least 3 characters")
	}
	if len(name) > 80 {
		return fmt.Errorf("name must be at most 80 characters")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return fmt.Errorf("name must contain only lowercase letters, digits, and hyphens")
		}
	}
	return nil
}

// kaosControlConfigDir returns the kaos-control app configuration directory,
// honouring XDG_CONFIG_HOME.
func kaosControlConfigDir() (string, error) {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "kaos-control"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kaos-control"), nil
}

// ValidatePathFormat checks that path is absolute and does not resolve into
// the kaos-control config directory. It does NOT check whether the path exists.
func ValidatePathFormat(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute")
	}
	clean := filepath.Clean(path)
	cfgDir, err := kaosControlConfigDir()
	if err == nil {
		sep := string(filepath.Separator)
		if clean == cfgDir || strings.HasPrefix(clean+sep, cfgDir+sep) {
			return fmt.Errorf("path must not be inside the kaos-control config directory")
		}
	}
	return nil
}

// ValidatePath validates path format, resolves symlinks, and returns the
// canonicalised path. Returns an error if the path does not exist, is relative,
// or falls inside the kaos-control config directory.
func ValidatePath(path string) (string, error) {
	if err := ValidatePathFormat(path); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %w", err)
	}
	// Re-check the resolved path against the config dir (symlinks can redirect).
	cfgDir, err2 := kaosControlConfigDir()
	if err2 == nil {
		sep := string(filepath.Separator)
		if resolved == cfgDir || strings.HasPrefix(resolved+sep, cfgDir+sep) {
			return "", fmt.Errorf("path must not be inside the kaos-control config directory")
		}
	}
	return resolved, nil
}

// DeleteProjectEntry removes a project registration file.
func DeleteProjectEntry(dir, name string) error {
	return os.Remove(filepath.Join(dir, name+".yaml"))
}

// Stage is one entry in a project's stage list.
type Stage struct {
	Name string `yaml:"name"`
	Dir  string `yaml:"dir"`
}

// GitConfig is the per-project git configuration section.
type GitConfig struct {
	DefaultBranch  string `yaml:"default_branch"`
	BranchTemplate string `yaml:"branch_template"`
}

// AgentConfig is one configured agent binding.
type AgentConfig struct {
	Name            string            `yaml:"name"`
	Roles           []string          `yaml:"-"` // populated by UnmarshalYAML from "role" or "roles"
	Driver          string            `yaml:"driver"`
	Model           string            `yaml:"model,omitempty"`
	Endpoint        string            `yaml:"endpoint,omitempty"`
	AllowedPaths    []string          `yaml:"allowed_write_paths,omitempty"`
	TimeoutMinutes  int               `yaml:"timeout_minutes,omitempty"` // 0 = no timeout
	GitIdentity     GitIdentity       `yaml:"git_identity"`
	PromptTemplates map[string]string `yaml:"prompt_templates,omitempty"` // role -> template
	// Status lifecycle: set target artifact status at run start/end.
	ActiveStatus  string `yaml:"active_status,omitempty"`   // status to set when run starts (empty = no change)
	DoneOnSuccess bool   `yaml:"done_on_success,omitempty"` // if true, set status=done when run completes successfully
	// SourceTypes lists the artifact types this agent consumes. When set, the
	// ready-counts endpoint filters by both status and type. When empty, the
	// existing behaviour (count by status only) is preserved.
	SourceTypes []string `yaml:"source_types,omitempty"`
	// Ollama-specific fields (only used when Driver == "ollama").
	OllamaInstanceName string `yaml:"ollama_instance,omitempty"` // name of OllamaInstance in app config
	OllamaEndpoint     string `yaml:"ollama_endpoint,omitempty"` // "chat" (default) or "generate"
	// claude-mediated driver fields.
	// BashAllowlist is a list of glob patterns; when non-empty only matching
	// commands are permitted. Checked after BashDenylist.
	BashAllowlist []string `yaml:"bash_allowlist,omitempty"`
	// BashDenylist is a list of glob patterns; matching commands are denied
	// regardless of BashAllowlist. Merged with the built-in default denylist.
	BashDenylist []string `yaml:"bash_denylist,omitempty"`
	// OnDenial controls what happens when a tool call is denied: "continue"
	// (default) lets the agent keep running; "abort" kills it immediately.
	OnDenial string `yaml:"on_denial,omitempty"`
	// ObserveOnly puts the permission endpoint in log-only mode: every tool
	// call is logged but always allowed. Useful for auditing without blocking.
	ObserveOnly bool `yaml:"observe_only,omitempty"`
	// ShellCommand is only used by the shell-stub driver (driver: shell-stub).
	// It is the shell command to run as the agent process.
	// If empty, the stub emits one synthetic result event and exits 0.
	ShellCommand string `yaml:"shell_command,omitempty"`
}

// agentConfigRaw is used internally to unmarshal AgentConfig and accept both
// "role" (canonical) and "roles" (alias) YAML keys for the roles list.
type agentConfigRaw struct {
	Name               string            `yaml:"name"`
	Role               []string          `yaml:"role"`
	Roles              []string          `yaml:"roles"`
	Driver             string            `yaml:"driver"`
	Model              string            `yaml:"model,omitempty"`
	Endpoint           string            `yaml:"endpoint,omitempty"`
	AllowedPaths       []string          `yaml:"allowed_write_paths,omitempty"`
	TimeoutMinutes     int               `yaml:"timeout_minutes,omitempty"`
	GitIdentity        GitIdentity       `yaml:"git_identity"`
	PromptTemplates    map[string]string `yaml:"prompt_templates,omitempty"`
	ActiveStatus       string            `yaml:"active_status,omitempty"`
	DoneOnSuccess      bool              `yaml:"done_on_success,omitempty"`
	SourceTypes        []string          `yaml:"source_types,omitempty"`
	OllamaInstanceName string            `yaml:"ollama_instance,omitempty"`
	OllamaEndpoint     string            `yaml:"ollama_endpoint,omitempty"`
	BashAllowlist      []string          `yaml:"bash_allowlist,omitempty"`
	BashDenylist       []string          `yaml:"bash_denylist,omitempty"`
	OnDenial           string            `yaml:"on_denial,omitempty"`
	ObserveOnly        bool              `yaml:"observe_only,omitempty"`
	ShellCommand       string            `yaml:"shell_command,omitempty"`
}

// UnmarshalYAML implements yaml.Unmarshaler so that AgentConfig accepts both
// "role:" (canonical singular key) and "roles:" (plural alias) for the agent
// roles list. Values from both keys are merged with duplicates removed.
func (a *AgentConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw agentConfigRaw
	if err := value.Decode(&raw); err != nil {
		return err
	}
	a.Name = raw.Name
	a.Driver = raw.Driver
	a.Model = raw.Model
	a.Endpoint = raw.Endpoint
	a.AllowedPaths = raw.AllowedPaths
	a.TimeoutMinutes = raw.TimeoutMinutes
	a.GitIdentity = raw.GitIdentity
	a.PromptTemplates = raw.PromptTemplates
	a.ActiveStatus = raw.ActiveStatus
	a.DoneOnSuccess = raw.DoneOnSuccess
	a.SourceTypes = raw.SourceTypes
	a.OllamaInstanceName = raw.OllamaInstanceName
	a.OllamaEndpoint = raw.OllamaEndpoint
	a.BashAllowlist = raw.BashAllowlist
	a.BashDenylist = raw.BashDenylist
	a.OnDenial = raw.OnDenial
	a.ObserveOnly = raw.ObserveOnly
	a.ShellCommand = raw.ShellCommand

	// Merge "role" and "roles" entries, preserving order and deduplicating.
	seen := make(map[string]bool)
	for _, r := range raw.Role {
		if !seen[r] {
			a.Roles = append(a.Roles, r)
			seen[r] = true
		}
	}
	for _, r := range raw.Roles {
		if !seen[r] {
			a.Roles = append(a.Roles, r)
			seen[r] = true
		}
	}
	return nil
}

// GitIdentity is the git author identity for an agent or user commit.
type GitIdentity struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

// UserBinding binds a user email to one or more roles in a project.
type UserBinding struct {
	Email string   `yaml:"email"`
	Roles []string `yaml:"roles"`
}

// RequiredPlans maps artifact type to a list of required plan types before advancing.
type RequiredPlans map[string][]string

// KanbanColumn is one column definition in the kanban board.
type KanbanColumn struct {
	Name     string   `yaml:"name"     json:"name"`
	Statuses []string `yaml:"statuses" json:"statuses"`
}

// KanbanConfig is the optional kanban board configuration.
type KanbanConfig struct {
	Columns       []KanbanColumn `yaml:"columns"                  json:"columns"`
	Uncategorised *bool          `yaml:"uncategorised,omitempty"  json:"uncategorised,omitempty"` // default true
	CardFields    []string       `yaml:"card_fields,omitempty"    json:"card_fields,omitempty"`
}

// FeedConfig controls event feed retention.
type FeedConfig struct {
	RetentionDays int `yaml:"retention_days"`
	MaxEvents     int `yaml:"max_events"`
}

// SchedulerConfig holds per-project scheduler defaults.
type SchedulerConfig struct {
	DefaultTimeout time.Duration `yaml:"default_timeout"` // default job timeout, e.g. "30m"
}

// DashboardConfig controls how the dashboard's stat cards, status-distribution,
// and velocity chart count "work items". The default tracks "ticket" — the
// spec's canonical work-item type. Projects whose lifecycle uses a different
// type for tracked work (e.g. "requirement", "defect") should override.
type DashboardConfig struct {
	// TrackedTypes is the artifact types counted by dashboard widgets.
	// Empty falls back to the default (["ticket"]).
	TrackedTypes []string `yaml:"tracked_types,omitempty"`
}

// RoadmapConfig holds roadmap display settings for a project.
type RoadmapConfig struct {
	// DefaultPeriodMode controls the initial time-axis mode shown in the Gantt view.
	// Accepted values: "autoscale" (default), "month", "quarter", "half-year", "year".
	DefaultPeriodMode string `yaml:"default_period_mode" json:"default_period_mode"`
}

// Project is the per-project configuration (lifecycle/config.yaml).
type Project struct {
	Stages        []Stage         `yaml:"stages"`
	Git           GitConfig       `yaml:"git"`
	Roles         []string        `yaml:"roles"`
	Transitions   []Transition    `yaml:"transitions,omitempty"`
	Users         []UserBinding   `yaml:"users"`
	Agents        []AgentConfig   `yaml:"agents"`
	RequiredPlans RequiredPlans   `yaml:"required_plans"`
	Ignore        []string        `yaml:"ignore"`
	Kanban        *KanbanConfig   `yaml:"kanban,omitempty"`
	Feed          FeedConfig      `yaml:"feed"`
	Scheduler     SchedulerConfig `yaml:"scheduler"`
	Dashboard     DashboardConfig `yaml:"dashboard"`
	Roadmap       RoadmapConfig   `yaml:"roadmap,omitempty" json:"roadmap,omitempty"`
}

// Transition overrides one edge in the state machine.
type Transition struct {
	From  string   `yaml:"from"`
	To    string   `yaml:"to"`
	Roles []string `yaml:"roles"`
	// Types restricts this transition to artifacts of the listed types.
	// Empty means the rule applies to all artifact types.
	Types []string `yaml:"types,omitempty"`
}

var defaultStages = []Stage{
	{Name: "ideas", Dir: "ideas"},
	{Name: "requirements", Dir: "requirements"},
	{Name: "backend-plans", Dir: "backend-plans"},
	{Name: "frontend-plans", Dir: "frontend-plans"},
	{Name: "dev-plans", Dir: "dev-plans"},
	{Name: "test-plans", Dir: "test-plans"},
	{Name: "tests", Dir: "tests"},
	{Name: "prototypes", Dir: "prototypes"},
	{Name: "releases", Dir: "releases"},
	{Name: "sprints", Dir: "sprints"},
	{Name: "defects", Dir: "defects"},
}

var defaultRoles = []string{
	"product-owner", "analyst",
	"backend-developer", "frontend-developer", "test-developer",
	"qa", "reviewer", "approver", "devops", "system",
}

func defaultProject() Project {
	return Project{
		Stages: defaultStages,
		Git: GitConfig{
			DefaultBranch:  "main",
			BranchTemplate: "requirement/{slug}",
		},
		Roles:         defaultRoles,
		RequiredPlans: RequiredPlans{"requirement": {"plan-backend", "plan-frontend", "plan-test"}},
		Ignore:        []string{"README.md"},
		Scheduler:     SchedulerConfig{DefaultTimeout: 30 * time.Minute},
		Dashboard:     DashboardConfig{TrackedTypes: []string{"ticket"}},
		Roadmap:       RoadmapConfig{DefaultPeriodMode: "autoscale"},
	}
}

// LoadProject reads lifecycle/config.yaml from the project root.
func LoadProject(projectRoot string) (*Project, error) {
	cfg := defaultProject()
	path := filepath.Join(projectRoot, "lifecycle", "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading project config: %w", err)
	}
	if err == nil {
		// Snapshot defaults before unmarshal — yaml may replace the map
		// when the YAML has a required_plans block that omits some types.
		defaultRequiredPlans := make(RequiredPlans, len(cfg.RequiredPlans))
		for k, v := range cfg.RequiredPlans {
			defaultRequiredPlans[k] = v
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing project config: %w", err)
		}
		// Restore default entries for types the YAML did not explicitly configure.
		// This ensures canonical types (e.g. "requirement") retain their default
		// plan gate even when the project config only names other types (e.g. "ticket").
		if cfg.RequiredPlans == nil {
			cfg.RequiredPlans = defaultRequiredPlans
		} else {
			for k, v := range defaultRequiredPlans {
				if _, ok := cfg.RequiredPlans[k]; !ok {
					cfg.RequiredPlans[k] = v
				}
			}
		}
	}

	if err := validateProject(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validateProject(cfg *Project) error {
	if len(cfg.Stages) == 0 {
		return fmt.Errorf("project config: stages must not be empty")
	}
	for _, s := range cfg.Stages {
		if s.Name == "" || s.Dir == "" {
			return fmt.Errorf("project config: each stage must have name and dir")
		}
	}
	if cfg.Git.DefaultBranch == "" {
		cfg.Git.DefaultBranch = "main"
	}
	if cfg.Git.BranchTemplate == "" {
		cfg.Git.BranchTemplate = "requirement/{slug}"
	}
	for i := range cfg.Agents {
		a := &cfg.Agents[i]
		if a.Name == "" {
			return fmt.Errorf("project config: agent entry missing name")
		}
		if len(a.Roles) == 0 {
			return fmt.Errorf("project config: agent %q has no roles", a.Name)
		}
		if a.Driver == "" {
			return fmt.Errorf("project config: agent %q missing driver", a.Name)
		}
		if a.Driver == "ollama" {
			if a.OllamaInstanceName == "" {
				return fmt.Errorf("project config: agent %q has driver=ollama but missing ollama_instance", a.Name)
			}
			if a.Model == "" {
				return fmt.Errorf("project config: agent %q has driver=ollama but missing model", a.Name)
			}
			if a.OllamaEndpoint == "" {
				a.OllamaEndpoint = "chat"
			} else if a.OllamaEndpoint != "chat" && a.OllamaEndpoint != "generate" {
				return fmt.Errorf("project config: agent %q ollama_endpoint must be \"chat\" or \"generate\", got %q", a.Name, a.OllamaEndpoint)
			}
		}
		if a.Driver == "claude-mediated" {
			if a.OnDenial == "" {
				a.OnDenial = "continue"
			} else if a.OnDenial != "continue" && a.OnDenial != "abort" {
				return fmt.Errorf("project config: agent %q on_denial must be \"continue\" or \"abort\", got %q", a.Name, a.OnDenial)
			}
		}
	}
	for _, pat := range cfg.Ignore {
		if _, err := filepath.Match(pat, ""); err != nil {
			return fmt.Errorf("project config: invalid ignore pattern %q: %w", pat, err)
		}
	}
	if cfg.Feed.RetentionDays <= 0 {
		cfg.Feed.RetentionDays = 30
	}
	if cfg.Feed.MaxEvents <= 0 {
		cfg.Feed.MaxEvents = 5000
	}
	if cfg.Scheduler.DefaultTimeout <= 0 {
		cfg.Scheduler.DefaultTimeout = 30 * time.Minute
	}
	if cfg.Roadmap.DefaultPeriodMode == "" {
		cfg.Roadmap.DefaultPeriodMode = "autoscale"
	}
	validPeriodModes := map[string]bool{
		"autoscale": true,
		"month":     true,
		"quarter":   true,
		"half-year": true,
		"year":      true,
	}
	if !validPeriodModes[cfg.Roadmap.DefaultPeriodMode] {
		return fmt.Errorf("project config: roadmap.default_period_mode %q is not valid; accepted values: autoscale, month, quarter, half-year, year", cfg.Roadmap.DefaultPeriodMode)
	}
	return nil
}

// IsInitialised reports whether a project at projectPath has been initialised
// (i.e. lifecycle/config.yaml exists on disk).
func IsInitialised(projectPath string) bool {
	_, err := os.Stat(filepath.Join(projectPath, "lifecycle", "config.yaml"))
	return err == nil
}

// DefaultStages returns the directory names for the default lifecycle stages.
func DefaultStages() []string {
	dirs := make([]string, len(defaultStages))
	for i, s := range defaultStages {
		dirs[i] = s.Dir
	}
	return dirs
}

// DefaultProjectConfigYAML returns the content for a freshly-initialised
// lifecycle/config.yaml, derived from the built-in project defaults.
func DefaultProjectConfigYAML() string {
	data, _ := yaml.Marshal(defaultProject())
	return string(data)
}

// ShouldIgnore reports whether the file at path should be excluded from indexing.
// It matches the base name of path against each glob pattern using filepath.Match.
func ShouldIgnore(path string, patterns []string) bool {
	base := filepath.Base(path)
	for _, pat := range patterns {
		matched, err := filepath.Match(pat, base)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// StageDir returns the filesystem directory for a named stage, or "" if not found.
func (p *Project) StageDir(stageName string) string {
	for _, s := range p.Stages {
		if s.Name == stageName {
			return s.Dir
		}
	}
	return ""
}

// RolesFor returns the roles bound to the given user email, or nil.
func (p *Project) RolesFor(email string) []string {
	for _, u := range p.Users {
		if u.Email == email {
			return u.Roles
		}
	}
	return nil
}
