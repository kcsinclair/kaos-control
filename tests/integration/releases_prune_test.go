// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/release"
)

// ── release-artefacts-9: markdown-authoritative reversal ─────────────────────

// seedReleaseRow inserts a release row into the cache without writing a disk
// file (nil sync), simulating a cache row that has drifted from disk.
func seedReleaseRow(t *testing.T, store *release.Store, name, status string) *release.Release {
	t.Helper()
	r := &release.Release{
		ProjectID: "testproject",
		Name:      name,
		Slug:      release.Slugify(name),
		Status:    status,
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(r, nil, ""); err != nil {
		t.Fatalf("seedReleaseRow %q: %v", name, err)
	}
	return r
}

// TestRehydratePrunesOrphanCacheRows verifies the DR-3 reversal: markdown is
// authoritative, so cache rows with no corresponding file on disk are pruned on
// rehydrate. The old DB→disk backfill (which resurrected files from cache rows)
// is gone — the cache must not write anything back to disk.
func TestRehydratePrunesOrphanCacheRows(t *testing.T) {
	env := newTestEnv(t, nil)
	store := release.NewStore(env.proj.Idx.DB())

	for _, name := range []string{"Q1 Orphan", "Q2 Orphan", "Q3 Orphan"} {
		seedReleaseRow(t, store, name, "planned")
	}

	// lifecycle/releases/ has no .md files for these rows.
	result, err := release.Rehydrate(context.Background(), store, "testproject", env.projectRoot)
	if err != nil {
		t.Fatalf("Rehydrate: %v", err)
	}
	if result.Pruned != 3 {
		t.Errorf("Pruned = %d, want 3", result.Pruned)
	}
	if count, _ := store.Count("testproject"); count != 0 {
		t.Errorf("cache Count = %d, want 0 after prune", count)
	}

	// No files were written back to disk (no backfill resurrection).
	releasesDir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	if entries, err := os.ReadDir(releasesDir); err == nil {
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".md" {
				t.Errorf("unexpected release file written by rehydrate: %s", e.Name())
			}
		}
	}
}

// TestReleaseCreateWritesFileFirst verifies the DR-1 write path: Store.Create
// writes the authoritative markdown file, and the cache reflects it. The file's
// frontmatter is the source of truth, and the cache row exists with a matching
// slug.
func TestReleaseCreateWritesFileFirst(t *testing.T) {
	env := newTestEnv(t, nil)
	store := release.NewStore(env.proj.Idx.DB())

	r := &release.Release{
		ProjectID: "testproject",
		Name:      "Q1 Authoritative",
		Status:    "planned",
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(r, env.proj.ReleaseSync, env.projectRoot); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The markdown file exists and carries the authoritative frontmatter.
	relPath := "lifecycle/releases/" + r.Slug + ".md"
	if _, err := os.Stat(filepath.Join(env.projectRoot, relPath)); err != nil {
		t.Fatalf("release file not written: %v", err)
	}
	fm := readReleaseFrontmatter(t, env.projectRoot, relPath)
	if title, _ := fm["title"].(string); title != "Q1 Authoritative" {
		t.Errorf("file title = %q, want %q", title, "Q1 Authoritative")
	}
	if status, _ := fm["status"].(string); status != "planned" {
		t.Errorf("file status = %q, want %q", status, "planned")
	}

	// The cache reflects the file.
	got, err := store.GetBySlug("testproject", r.Slug)
	if err != nil || got == nil {
		t.Fatalf("cache row missing after Create: got=%v err=%v", got, err)
	}
	if got.Name != "Q1 Authoritative" {
		t.Errorf("cache Name = %q, want %q", got.Name, "Q1 Authoritative")
	}

	// A duplicate create is rejected as a conflict.
	dup := &release.Release{ProjectID: "testproject", Name: "Q1 Authoritative", Status: "planned", UpdatedAt: time.Now().UTC()}
	if err := store.Create(dup, env.proj.ReleaseSync, env.projectRoot); err != release.ErrConflict {
		t.Errorf("duplicate Create err = %v, want ErrConflict", err)
	}
}
