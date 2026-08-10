// SPDX-License-Identifier: AGPL-3.0-or-later

package release

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE releases (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id  TEXT NOT NULL,
			name        TEXT NOT NULL,
			slug        TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL DEFAULT 'planned',
			start_date  TEXT,
			end_date    TEXT,
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL,
			UNIQUE(project_id, name)
		);
		CREATE UNIQUE INDEX idx_releases_project_slug
			ON releases(project_id, slug) WHERE slug != '';
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func validReleaseMD(title, status string) string {
	return "---\ntitle: " + title + "\ntype: release\nstatus: " + status +
		"\nupdated_at: 2026-01-01T00:00:00Z\n---\n\nBody text.\n"
}

func TestRehydrate_ValidFilesInserted(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "lifecycle", "releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write 3 valid + 1 invalid release files.
	files := map[string]string{
		"q1-2026.md": validReleaseMD("Q1 2026", "planned"),
		"q2-2026.md": validReleaseMD("Q2 2026", "active"),
		"q3-2026.md": validReleaseMD("Q3 2026", "shipped"),
		"bad.md":     "---\ntitle: bad\ntype: release\nstatus: planned\nstart_date: 2026-06-01\nend_date: 2026-01-01\nupdated_at: 2026-01-01T00:00:00Z\n---\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	db := openTestDB(t)
	defer db.Close()
	store := NewStore(db)

	result, err := Rehydrate(context.Background(), store, "proj", root)
	if err != nil {
		t.Fatalf("Rehydrate: %v", err)
	}

	if result.Inserted != 3 {
		t.Errorf("Inserted = %d, want 3", result.Inserted)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}
}

func TestRehydrate_InvalidFileSkippedWithWarn(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "lifecycle", "releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// File with end_date before start_date.
	bad := "---\ntitle: Bad Release\ntype: release\nstatus: planned\n" +
		"start_date: 2026-06-01\nend_date: 2026-01-01\nupdated_at: 2026-01-01T00:00:00Z\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "bad.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	db := openTestDB(t)
	defer db.Close()
	store := NewStore(db)

	result, err := Rehydrate(context.Background(), store, "proj", root)
	if err != nil {
		t.Fatalf("Rehydrate: %v", err)
	}
	if result.Inserted != 0 {
		t.Errorf("Inserted = %d, want 0", result.Inserted)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}

	// Project should still load (no DB rows).
	count, _ := store.Count("proj")
	if count != 0 {
		t.Errorf("expected 0 rows in DB after skipping, got %d", count)
	}
}

func TestRehydrate_Idempotent(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "lifecycle", "releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "q1-2026.md"),
		[]byte(validReleaseMD("Q1 2026", "planned")), 0o644); err != nil {
		t.Fatal(err)
	}

	db := openTestDB(t)
	defer db.Close()
	store := NewStore(db)

	// Run twice.
	r1, err := Rehydrate(context.Background(), store, "proj", root)
	if err != nil {
		t.Fatalf("first Rehydrate: %v", err)
	}
	r2, err := Rehydrate(context.Background(), store, "proj", root)
	if err != nil {
		t.Fatalf("second Rehydrate: %v", err)
	}
	if r1.Inserted != 1 || r2.Inserted != 1 {
		t.Errorf("expected 1 inserted each time, got %d and %d", r1.Inserted, r2.Inserted)
	}

	count, _ := store.Count("proj")
	if count != 1 {
		t.Errorf("expected exactly 1 DB row after two runs, got %d", count)
	}
}

func TestRehydrate_EmptyDir(t *testing.T) {
	root := t.TempDir()
	// No lifecycle/releases directory at all.
	db := openTestDB(t)
	defer db.Close()
	store := NewStore(db)

	result, err := Rehydrate(context.Background(), store, "proj", root)
	if err != nil {
		t.Fatalf("Rehydrate with missing dir: %v", err)
	}
	if result.Inserted != 0 || result.Skipped != 0 {
		t.Errorf("expected 0/0, got %d/%d", result.Inserted, result.Skipped)
	}
}

// TestRehydrate_EmptyDirPrunesCache verifies the DR-3 reversal: markdown is
// authoritative, so a populated cache whose lifecycle/releases/ directory is
// empty (or missing) is pruned to zero on rehydrate. The old DB→disk Backfill
// behaviour (resurrecting files from cache rows) is gone.
func TestRehydrate_EmptyDirPrunesCache(t *testing.T) {
	root := t.TempDir()

	db := openTestDB(t)
	defer db.Close()
	store := NewStore(db)

	// Seed 3 cache rows with no corresponding files on disk.
	now := time.Now().UTC()
	for _, name := range []string{"Q1 2026", "Q2 2026", "Q3 2026"} {
		r := &Release{
			ProjectID: "proj",
			Name:      name,
			Slug:      Slugify(name),
			Status:    "planned",
			UpdatedAt: now,
		}
		if err := store.Create(r, nil, ""); err != nil {
			t.Fatalf("seeding %q: %v", name, err)
		}
	}

	// Missing releases directory: rehydrate must prune the cache to empty and
	// must NOT write any files back to disk.
	result, err := Rehydrate(context.Background(), store, "proj", root)
	if err != nil {
		t.Fatalf("Rehydrate: %v", err)
	}
	if result.Pruned != 3 {
		t.Errorf("Pruned = %d, want 3", result.Pruned)
	}
	if n, err := store.Count("proj"); err != nil || n != 0 {
		t.Errorf("cache Count = %d (err=%v), want 0", n, err)
	}
	dir := filepath.Join(root, "lifecycle", "releases")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("releases directory should not have been created, stat err = %v", err)
	}
}
