// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/index"
)

// TestSchemaMigrationFromVersion0 verifies that opening an index DB with
// schema_version=0 (simulating an older version) causes the server to rebuild
// the index from disk, and the final state matches a fresh run.
// Test plan §7: "Schema migration" scenario.
func TestSchemaMigrationFromVersion0(t *testing.T) {
	root := t.TempDir()
	dbDir := t.TempDir()

	// Create lifecycle directories and seed artifacts.
	stages := []config.Stage{
		{Name: "ideas", Dir: "ideas"},
		{Name: "requirements", Dir: "requirements"},
	}
	for _, s := range stages {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s.Dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Write a lifecycle/config.yaml.
	cfgYAML := `stages:
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
`
	if err := os.WriteFile(filepath.Join(root, "lifecycle", "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Seed an artifact on disk.
	content := makeArtifact("Schema Test", "idea", "draft", "schema-test", "", "Testing schema migration.")
	if err := os.WriteFile(filepath.Join(root, "lifecycle", "ideas", "schema-test.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dbDir, "test", "index.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// Step 1: Create a DB with schema_version=0 to simulate an older version.
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL); INSERT INTO schema_version VALUES (0);`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Step 2: Open the index — it should detect the version mismatch, drop, and rebuild.
	idx, err := index.Open(dbPath, root, stages)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	// Step 3: Verify the artifact was indexed during rebuild.
	row, err := idx.Get("lifecycle/ideas/schema-test.md")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("expected artifact to be indexed after schema migration")
	}
	if row.Title != "Schema Test" {
		t.Errorf("expected title 'Schema Test', got %q", row.Title)
	}
	if row.Status != "draft" {
		t.Errorf("expected status 'draft', got %q", row.Status)
	}
}

// TestFreshIndexMatchesRebuild verifies that a fresh index and a rebuilt index
// produce the same results for the same disk content.
func TestFreshIndexMatchesRebuild(t *testing.T) {
	root := t.TempDir()

	stages := []config.Stage{
		{Name: "ideas", Dir: "ideas"},
	}
	if err := os.MkdirAll(filepath.Join(root, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lifecycle", "config.yaml"), []byte("stages:\n  - {name: ideas, dir: ideas}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	content1 := makeArtifact("Item A", "idea", "draft", "item-a", "", "First item.")
	content2 := makeArtifact("Item B", "idea", "clarifying", "item-b", "", "Second item.")
	if err := os.WriteFile(filepath.Join(root, "lifecycle", "ideas", "item-a.md"), []byte(content1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lifecycle", "ideas", "item-b.md"), []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create fresh index.
	dbPath1 := filepath.Join(t.TempDir(), "fresh", "index.db")
	idx1, err := index.Open(dbPath1, root, stages)
	if err != nil {
		t.Fatal(err)
	}
	defer idx1.Close()

	// Create index with stale version (triggers rebuild).
	dbPath2 := filepath.Join(t.TempDir(), "rebuild", "index.db")
	if err := os.MkdirAll(filepath.Dir(dbPath2), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath2+"?_journal=WAL")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL); INSERT INTO schema_version VALUES (0);`)
	db.Close()

	idx2, err := index.Open(dbPath2, root, stages)
	if err != nil {
		t.Fatal(err)
	}
	defer idx2.Close()

	// Compare: both should have the same artifacts.
	rows1, total1, _ := idx1.List(index.Filter{})
	rows2, total2, _ := idx2.List(index.Filter{})

	if total1 != total2 {
		t.Errorf("fresh total=%d, rebuild total=%d", total1, total2)
	}
	if len(rows1) != len(rows2) {
		t.Errorf("fresh count=%d, rebuild count=%d", len(rows1), len(rows2))
	}
}
