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
