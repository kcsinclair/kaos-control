package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// App is the top-level application configuration (install-dir/config.yaml).
type App struct {
	Server      ServerConfig  `yaml:"server"`
	Auth        AuthConfig    `yaml:"auth"`
	ProjectsDir string        `yaml:"projects_dir"`
	Limits      LimitsConfig  `yaml:"limits"`
	DataDir     string        `yaml:"data_dir"` // where app DBs live; defaults to projects_dir/../data
}

type ServerConfig struct {
	Listen string    `yaml:"listen"`
	TLS    TLSConfig `yaml:"tls"`
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
	MaxConcurrentAgents int `yaml:"max_concurrent_agents"`
}

func defaultApp() App {
	return App{
		Server: ServerConfig{
			Listen: ":8080",
		},
		Auth: AuthConfig{
			Method:     "local",
			SessionTTL: 24 * time.Hour,
		},
		Limits: LimitsConfig{
			MaxConcurrentAgents: 4,
		},
	}
}

// LoadApp reads the app-level config file, applying defaults for missing fields.
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
	}

	if err := validateApp(&cfg); err != nil {
		return nil, err
	}

	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(filepath.Dir(path), "data")
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
	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("server.tls.cert_file and server.tls.key_file are required when TLS is enabled")
		}
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

// SaveProjectEntry writes a project entry to projects_dir/<name>.yaml.
func SaveProjectEntry(dir string, e *ProjectEntry) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating projects dir: %w", err)
	}
	data, err := yaml.Marshal(e)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, e.Name+".yaml"), data, 0o644)
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
	Roles           []string          `yaml:"role"`
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

// Project is the per-project configuration (lifecycle/config.yaml).
type Project struct {
	Stages        []Stage       `yaml:"stages"`
	Git           GitConfig     `yaml:"git"`
	Roles         []string      `yaml:"roles"`
	Transitions   []Transition  `yaml:"transitions,omitempty"`
	Users         []UserBinding `yaml:"users"`
	Agents        []AgentConfig `yaml:"agents"`
	RequiredPlans RequiredPlans `yaml:"required_plans"`
	Ignore        []string      `yaml:"ignore"`
	Kanban        *KanbanConfig `yaml:"kanban,omitempty"`
	Feed          FeedConfig    `yaml:"feed"`
}

// Transition overrides one edge in the state machine.
type Transition struct {
	From  string   `yaml:"from"`
	To    string   `yaml:"to"`
	Roles []string `yaml:"roles"`
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
	"qa", "reviewer", "approver",
}

func defaultProject() Project {
	return Project{
		Stages: defaultStages,
		Git: GitConfig{
			DefaultBranch:  "main",
			BranchTemplate: "requirement/{slug}",
		},
		Roles:         defaultRoles,
		RequiredPlans: RequiredPlans{"requirement": {}},
		Ignore:        []string{"README.md"},
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
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing project config: %w", err)
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
	for _, a := range cfg.Agents {
		if a.Name == "" {
			return fmt.Errorf("project config: agent entry missing name")
		}
		if len(a.Roles) == 0 {
			return fmt.Errorf("project config: agent %q has no roles", a.Name)
		}
		if a.Driver == "" {
			return fmt.Errorf("project config: agent %q missing driver", a.Name)
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
	return nil
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
