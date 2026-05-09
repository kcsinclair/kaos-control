// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// openTestIndex opens a fresh in-temp-dir SQLite index with no stages.
// It registers cleanup automatically via t.Cleanup.
func openTestIndex(t *testing.T) *Index {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	projRoot := dir
	// Minimal lifecycle directory so Scan has a root to inspect (but no stages are passed).
	if err := os.MkdirAll(projRoot+"/lifecycle", 0o755); err != nil {
		t.Fatal(err)
	}
	idx, err := Open(dbPath, projRoot, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

// makeTestArtifact builds a minimal Artifact suitable for upserting in tests.
func makeTestArtifact(path, slug, created string) *artifact.Artifact {
	a := &artifact.Artifact{
		Path:  path,
		Slug:  slug,
		Stage: "ideas",
		Index: 0,
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   slug,
			Type:    "idea",
			Status:  "draft",
			Lineage: slug,
			Created: created,
		},
	}
	return a
}

// TestUpsert_CreatedTimestamp verifies that upserting an artifact with a `created`
// frontmatter field correctly stores and retrieves the timestamp via Get.
func TestUpsert_CreatedTimestamp(t *testing.T) {
	idx := openTestIndex(t)

	const created = "2026-04-27T10:00:00Z"
	wantTime, err := time.Parse(time.RFC3339, created)
	if err != nil {
		t.Fatalf("parsing reference time: %v", err)
	}

	a := makeTestArtifact("lifecycle/ideas/created-ts.md", "created-ts", created)
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	row, err := idx.Get("lifecycle/ideas/created-ts.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("Get returned nil")
	}
	if row.Created.IsZero() {
		t.Fatal("expected non-zero Created in row")
	}
	if row.Created.Unix() != wantTime.Unix() {
		t.Errorf("Created mismatch: want %v (unix %d), got %v (unix %d)",
			wantTime, wantTime.Unix(), row.Created, row.Created.Unix())
	}
}

// TestUpsert_CreatedZero verifies that upserting an artifact without `created`
// in frontmatter (and no CreatedAt backfill) stores a zero value and retrieves
// a zero time.Time without error.
func TestUpsert_CreatedZero(t *testing.T) {
	idx := openTestIndex(t)

	a := makeTestArtifact("lifecycle/ideas/no-created.md", "no-created", "")
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	row, err := idx.Get("lifecycle/ideas/no-created.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("Get returned nil")
	}
	// created column = 0 → time.Unix(0,0) maps back to zero-check via the
	// scanRows helper which only sets Created when createdUnix != 0.
	if !row.Created.IsZero() {
		t.Errorf("expected zero Created, got %v", row.Created)
	}
}

// TestSchemaUpgrade verifies that opening an index whose stored schema_version
// does not match the current schemaVersion constant causes a drop-and-recreate
// that includes the `created` column. After the upgrade, upserts that set
// Created should round-trip correctly.
func TestSchemaUpgrade(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	projRoot := dir
	if err := os.MkdirAll(projRoot+"/lifecycle", 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed a DB with schema version 0 (deliberately wrong) to force a rebuild.
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("seeding: sql.Open: %v", err)
	}
	_, err = db.Exec(
		`CREATE TABLE schema_version (version INTEGER NOT NULL); INSERT INTO schema_version VALUES (0);`,
	)
	if err != nil {
		db.Close()
		t.Fatalf("seeding old schema: %v", err)
	}
	db.Close()

	// Open via index.Open — should detect version mismatch and rebuild.
	idx, err := Open(dbPath, projRoot, nil)
	if err != nil {
		t.Fatalf("Open after schema mismatch: %v", err)
	}
	defer idx.Close()

	// After rebuild the `created` column must exist; exercise it with a round-trip.
	const created = "2026-04-27T10:00:00Z"
	a := makeTestArtifact("lifecycle/ideas/upgrade.md", "upgrade", created)
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert after schema upgrade: %v", err)
	}

	row, err := idx.Get("lifecycle/ideas/upgrade.md")
	if err != nil {
		t.Fatalf("Get after schema upgrade: %v", err)
	}
	if row == nil {
		t.Fatal("row not found after schema upgrade")
	}
	if row.Created.IsZero() {
		t.Errorf("expected non-zero Created after schema upgrade")
	}

	// Also verify the schema_version was bumped to the current version.
	var v int
	if err := idx.db.QueryRow(`SELECT version FROM schema_version`).Scan(&v); err != nil {
		t.Fatalf("reading schema_version: %v", err)
	}
	if v != schemaVersion {
		t.Errorf("schema_version: want %d, got %d", schemaVersion, v)
	}
}
