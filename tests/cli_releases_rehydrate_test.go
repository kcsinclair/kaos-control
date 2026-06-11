// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// CLI integration tests for `kaos-control releases rehydrate`.
// These tests invoke the compiled binary (built by TestMain in cli_init_test.go)
// and exercise the full CLI path from argument parsing through SQLite writes to
// JSON output.
package cli_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// setupRehydrateEnv creates a fully isolated temp environment for the CLI
// rehydrate tests:
//   - cfg/config.yaml   — app config with data_dir pointing at cfg/data/
//   - cfg/projects/<name>.yaml — project registration pointing at projectRoot
//   - projectRoot/lifecycle/releases/*.md — seeded release files
//   - projectRoot/lifecycle/config.yaml — minimal project config
//   - cfg/data/<name>/index.db — SQLite DB with releases schema
//
// Returns (cfgPath, projectRoot).
func setupRehydrateEnv(t *testing.T, name string, releaseFiles []struct{ slug, title, status string }) (cfgPath, projectRoot string) {
	t.Helper()

	cfgDir := t.TempDir()
	projectRoot = t.TempDir()
	dataDir := filepath.Join(cfgDir, "data")

	// Write project lifecycle structure.
	releasesDir := filepath.Join(projectRoot, "lifecycle", "releases")
	if err := os.MkdirAll(releasesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	minimalConfig := `git:
  default_branch: main
stages:
  - {name: releases, dir: releases}
`
	if err := os.WriteFile(filepath.Join(projectRoot, "lifecycle", "config.yaml"),
		[]byte(minimalConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write release markdown files.
	for _, f := range releaseFiles {
		content := fmt.Sprintf("---\ntitle: %s\ntype: release\nstatus: %s\nupdated_at: 2026-01-01T00:00:00Z\n---\n\nBody.\n",
			f.title, f.status)
		if err := os.WriteFile(filepath.Join(releasesDir, f.slug+".md"),
			[]byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Write app config.yaml.
	cfgPath = filepath.Join(cfgDir, "config.yaml")
	cfgContent := fmt.Sprintf("data_dir: %q\n", dataDir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write project registration at cfg/projects/<name>.yaml.
	projectsDir := filepath.Join(cfgDir, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entryContent := fmt.Sprintf("name: %s\npath: %q\n", name, projectRoot)
	if err := os.WriteFile(filepath.Join(projectsDir, name+".yaml"),
		[]byte(entryContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create the SQLite DB with the releases schema.
	dbDir := filepath.Join(dataDir, name)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "index.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE releases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'planned',
			start_date TEXT,
			end_date TEXT,
			created_at TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT '',
			UNIQUE(project_id, name)
		);
		CREATE UNIQUE INDEX idx_releases_project_slug
			ON releases(project_id, slug) WHERE slug != '';
	`)
	db.Close()
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return cfgPath, projectRoot
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestReleasesRehydrateCLI_Success writes 3 release files, runs the binary,
// and asserts that stdout is valid JSON with inserted==3, skipped==0, and
// errors==[] (or null).
func TestReleasesRehydrateCLI_Success(t *testing.T) {
	files := []struct{ slug, title, status string }{
		{"cli-alpha", "CLI Alpha", "planned"},
		{"cli-beta", "CLI Beta", "active"},
		{"cli-gamma", "CLI Gamma", "shipped"},
	}
	cfgPath, _ := setupRehydrateEnv(t, "cli-test-project", files)

	stdout, stderr, code := runBin(t, "releases", "rehydrate",
		"--project", "cli-test-project",
		"--config", cfgPath)

	if code != 0 {
		t.Fatalf("want exit 0, got %d\nstderr: %s", code, stderr)
	}

	var result struct {
		Inserted int      `json:"inserted"`
		Skipped  int      `json:"skipped"`
		Errors   []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\ngot: %s", err, stdout)
	}
	if result.Inserted != 3 {
		t.Errorf("inserted: want 3, got %d", result.Inserted)
	}
	if result.Skipped != 0 {
		t.Errorf("skipped: want 0, got %d", result.Skipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("errors: want [], got %v", result.Errors)
	}
}

// TestReleasesRehydrateCLI_ProjectNotFound runs the binary with an unknown
// project name and asserts exit code 1 with an error message containing
// "not found".
func TestReleasesRehydrateCLI_ProjectNotFound(t *testing.T) {
	cfgPath, _ := setupRehydrateEnv(t, "cli-test-nf", nil)

	_, stderr, code := runBin(t, "releases", "rehydrate",
		"--project", "does-not-exist",
		"--config", cfgPath)

	if code == 0 {
		t.Error("want non-zero exit code for unknown project, got 0")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("stderr should contain 'not found', got: %s", stderr)
	}
}
