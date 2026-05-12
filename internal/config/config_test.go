// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestShouldIgnore verifies the ShouldIgnore helper with various pattern/path
// combinations.  Run with: go test ./internal/config/ -run ShouldIgnore
func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "README.md matches exact pattern in ideas stage",
			path:     "lifecycle/ideas/README.md",
			patterns: []string{"README.md"},
			want:     true,
		},
		{
			name:     "README.md matches exact pattern in requirements stage",
			path:     "lifecycle/requirements/README.md",
			patterns: []string{"README.md"},
			want:     true,
		},
		{
			name:     "my-readme.md does not match README.md pattern",
			path:     "lifecycle/ideas/my-readme.md",
			patterns: []string{"README.md"},
			want:     false,
		},
		{
			name:     "glob *.draft.md matches feature.draft.md",
			path:     "lifecycle/ideas/feature.draft.md",
			patterns: []string{"*.draft.md"},
			want:     true,
		},
		{
			name:     "glob *.draft.md does not match feature.md",
			path:     "lifecycle/ideas/feature.md",
			patterns: []string{"*.draft.md"},
			want:     false,
		},
		{
			name:     "empty pattern list matches nothing",
			path:     "lifecycle/ideas/README.md",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "nil pattern list matches nothing",
			path:     "lifecycle/ideas/README.md",
			patterns: nil,
			want:     false,
		},
		{
			name:     "second pattern in list matches",
			path:     "lifecycle/ideas/CHANGELOG.md",
			patterns: []string{"README.md", "CHANGELOG.md"},
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldIgnore(tc.path, tc.patterns)
			if got != tc.want {
				t.Errorf("ShouldIgnore(%q, %v) = %v, want %v",
					tc.path, tc.patterns, got, tc.want)
			}
		})
	}
}

// TestLoadProjectIgnoreField verifies that the Ignore field is correctly parsed
// from lifecycle/config.yaml and that the default (["README.md"]) is applied when
// the field is absent.
func TestLoadProjectIgnoreField(t *testing.T) {
	t.Run("explicit ignore patterns are loaded", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, `ignore: ["README.md", "CHANGELOG.md"]`)
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if len(cfg.Ignore) != 2 {
			t.Fatalf("expected 2 ignore patterns, got %d: %v", len(cfg.Ignore), cfg.Ignore)
		}
		if cfg.Ignore[0] != "README.md" || cfg.Ignore[1] != "CHANGELOG.md" {
			t.Errorf("unexpected ignore patterns: %v", cfg.Ignore)
		}
	})

	t.Run("missing ignore key uses default README.md", func(t *testing.T) {
		// No ignore: key in the YAML; the default from defaultProject() must apply.
		dir := writeMinimalProjectConfig(t, "")
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if len(cfg.Ignore) != 1 || cfg.Ignore[0] != "README.md" {
			t.Errorf(`expected default ignore ["README.md"], got %v`, cfg.Ignore)
		}
	})

	t.Run("invalid glob pattern returns validation error", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, `ignore: ["[invalid"]`)
		_, err := LoadProject(dir)
		if err == nil {
			t.Fatal("expected error for invalid glob pattern, got nil")
		}
	})
}

// TestKanbanConfig verifies that the kanban section of lifecycle/config.yaml is
// correctly parsed (or absent) by LoadProject.
// Run with: go test ./internal/config/ -run TestKanban
func TestKanbanConfig(t *testing.T) {
	t.Run("full kanban config parses correctly", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, `kanban:
  columns:
    - name: Backlog
      statuses: [draft, clarifying]
    - name: In Progress
      statuses: [in-development, in-qa]
    - name: Done
      statuses: [done]
  uncategorised: false
  card_fields: [title, type, priority]
`)
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if cfg.Kanban == nil {
			t.Fatal("expected Kanban to be non-nil")
		}
		if len(cfg.Kanban.Columns) != 3 {
			t.Fatalf("expected 3 columns, got %d", len(cfg.Kanban.Columns))
		}
		if cfg.Kanban.Columns[0].Name != "Backlog" {
			t.Errorf("column[0].Name = %q, want %q", cfg.Kanban.Columns[0].Name, "Backlog")
		}
		if len(cfg.Kanban.Columns[0].Statuses) != 2 {
			t.Errorf("column[0].Statuses len = %d, want 2", len(cfg.Kanban.Columns[0].Statuses))
		}
		if cfg.Kanban.Uncategorised == nil || *cfg.Kanban.Uncategorised != false {
			t.Errorf("expected Uncategorised=false, got %v", cfg.Kanban.Uncategorised)
		}
		if len(cfg.Kanban.CardFields) != 3 {
			t.Errorf("expected 3 card_fields, got %d", len(cfg.Kanban.CardFields))
		}
	})

	t.Run("minimal kanban config — only columns, uncategorised defaults to true semantics", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, `kanban:
  columns:
    - name: Backlog
      statuses: [draft]
`)
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if cfg.Kanban == nil {
			t.Fatal("expected Kanban to be non-nil")
		}
		if len(cfg.Kanban.Columns) != 1 {
			t.Fatalf("expected 1 column, got %d", len(cfg.Kanban.Columns))
		}
		// Uncategorised is nil when not specified; callers treat nil as true.
		if cfg.Kanban.Uncategorised != nil {
			t.Errorf("expected Uncategorised to be nil (default-true), got %v", *cfg.Kanban.Uncategorised)
		}
		if len(cfg.Kanban.CardFields) != 0 {
			t.Errorf("expected empty card_fields, got %v", cfg.Kanban.CardFields)
		}
	})

	t.Run("no kanban key leaves Kanban nil", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, "")
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if cfg.Kanban != nil {
			t.Errorf("expected Kanban to be nil, got %+v", cfg.Kanban)
		}
	})

	t.Run("empty columns list parses without error", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, `kanban:
  columns: []
`)
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if cfg.Kanban == nil {
			t.Fatal("expected Kanban to be non-nil")
		}
		if len(cfg.Kanban.Columns) != 0 {
			t.Errorf("expected empty columns slice, got %v", cfg.Kanban.Columns)
		}
	})
}

// TestRoadmapConfig verifies parsing and validation of the roadmap.default_period_mode field.
// Run with: go test ./internal/config/ -run Roadmap -v
func TestRoadmapConfig(t *testing.T) {
	validModes := []string{"autoscale", "month", "quarter", "half-year", "year"}

	for _, mode := range validModes {
		mode := mode
		t.Run("valid mode: "+mode, func(t *testing.T) {
			dir := writeMinimalProjectConfig(t, "roadmap:\n  default_period_mode: "+mode+"\n")
			cfg, err := LoadProject(dir)
			if err != nil {
				t.Fatalf("LoadProject with mode %q: %v", mode, err)
			}
			if cfg.Roadmap.DefaultPeriodMode != mode {
				t.Errorf("DefaultPeriodMode = %q, want %q", cfg.Roadmap.DefaultPeriodMode, mode)
			}
		})
	}

	t.Run("invalid mode returns descriptive error", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, "roadmap:\n  default_period_mode: weekly\n")
		_, err := LoadProject(dir)
		if err == nil {
			t.Fatal("expected error for invalid mode \"weekly\", got nil")
		}
		if !containsString(err.Error(), "weekly") {
			t.Errorf("error %q does not mention the invalid value \"weekly\"", err.Error())
		}
		if !containsString(err.Error(), "default_period_mode") {
			t.Errorf("error %q does not mention the field name", err.Error())
		}
	})

	t.Run("omitted roadmap section defaults to autoscale", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, "")
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if cfg.Roadmap.DefaultPeriodMode != "autoscale" {
			t.Errorf("DefaultPeriodMode = %q, want \"autoscale\"", cfg.Roadmap.DefaultPeriodMode)
		}
	})

	t.Run("empty default_period_mode defaults to autoscale", func(t *testing.T) {
		dir := writeMinimalProjectConfig(t, "roadmap:\n  default_period_mode: \"\"\n")
		cfg, err := LoadProject(dir)
		if err != nil {
			t.Fatalf("LoadProject: %v", err)
		}
		if cfg.Roadmap.DefaultPeriodMode != "autoscale" {
			t.Errorf("DefaultPeriodMode = %q, want \"autoscale\"", cfg.Roadmap.DefaultPeriodMode)
		}
	})
}

// containsString is a helper used by TestRoadmapConfig.
func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// TestLoadAppDefaultDataDir verifies that when LoadApp creates a fresh config file
// (first run, no existing file), the generated YAML contains a non-empty data_dir
// set to <config-dir>/data.
func TestLoadAppDefaultDataDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg, err := LoadApp(cfgPath)
	if err != nil {
		t.Fatalf("LoadApp: %v", err)
	}

	wantDataDir := filepath.Join(dir, "data")
	if cfg.DataDir != wantDataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, wantDataDir)
	}

	// Reload from the persisted file and confirm data_dir is present.
	cfg2, err := LoadApp(cfgPath)
	if err != nil {
		t.Fatalf("LoadApp (second call): %v", err)
	}
	if cfg2.DataDir != wantDataDir {
		t.Errorf("DataDir after reload = %q, want %q", cfg2.DataDir, wantDataDir)
	}
}

// TestLoadApp_AgentPrecheckDefaults verifies that when no agent: section is
// present in the app config file, the default values are applied correctly.
// Run with: go test ./internal/config/ -run TestLoadApp_AgentPrecheckDefaults
func TestLoadApp_AgentPrecheckDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Write a minimal valid app config without an agent: section.
	minimalCfg := "server:\n  listen: \":9999\"\nauth:\n  method: local\n  session_ttl: 24h\nprojects_dir: " + filepath.Join(dir, "projects") + "\ndata_dir: " + filepath.Join(dir, "data") + "\n"
	if err := os.WriteFile(cfgPath, []byte(minimalCfg), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := LoadApp(cfgPath)
	if err != nil {
		t.Fatalf("LoadApp: %v", err)
	}

	if cfg.Agent.InitEventTimeoutSeconds != 10 {
		t.Errorf("InitEventTimeoutSeconds = %d, want 10", cfg.Agent.InitEventTimeoutSeconds)
	}
	if cfg.Agent.RequireBypassPermissions == nil {
		t.Fatal("RequireBypassPermissions is nil, want non-nil pointer to true")
	}
	if !*cfg.Agent.RequireBypassPermissions {
		t.Errorf("RequireBypassPermissions = false, want true")
	}
}

// TestLoadApp_AgentPrecheckExplicitFalse verifies that explicitly setting
// require_bypass_permissions: false in the config survives the load (pointer-bool
// semantics: false != unset).
// Run with: go test ./internal/config/ -run TestLoadApp_AgentPrecheckExplicitFalse
func TestLoadApp_AgentPrecheckExplicitFalse(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfgWithFalse := "server:\n  listen: \":9999\"\nauth:\n  method: local\n  session_ttl: 24h\nprojects_dir: " + filepath.Join(dir, "projects") + "\ndata_dir: " + filepath.Join(dir, "data") + "\nagent:\n  init_event_timeout_seconds: 30\n  require_bypass_permissions: false\n"
	if err := os.WriteFile(cfgPath, []byte(cfgWithFalse), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := LoadApp(cfgPath)
	if err != nil {
		t.Fatalf("LoadApp: %v", err)
	}

	if cfg.Agent.InitEventTimeoutSeconds != 30 {
		t.Errorf("InitEventTimeoutSeconds = %d, want 30", cfg.Agent.InitEventTimeoutSeconds)
	}
	if cfg.Agent.RequireBypassPermissions == nil {
		t.Fatal("RequireBypassPermissions is nil, want non-nil pointer to false")
	}
	if *cfg.Agent.RequireBypassPermissions {
		t.Errorf("RequireBypassPermissions = true, want false (explicit override)")
	}
}

// writeMinimalProjectConfig writes a lifecycle/config.yaml with a minimal valid
// base configuration plus an optional extra YAML snippet (e.g. an ignore: line),
// and returns the temp project root directory.
func writeMinimalProjectConfig(t *testing.T, extraYAML string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lifecycle"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "stages:\n  - {name: ideas, dir: ideas}\ngit:\n  default_branch: main\nroles:\n  - analyst\n"
	if extraYAML != "" {
		content += extraYAML + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "lifecycle", "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
