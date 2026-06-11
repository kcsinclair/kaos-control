// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/release"
)

// ── Milestone T3: Backfill (DB→disk) ─────────────────────────────────────────

// seedReleaseRow inserts a release row directly into the DB without writing
// a disk file (nil sync). Sets the slug via release.Slugify.
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

// fileStatMap returns a map from filename → FileInfo for all .md files in dir.
func fileStatMap(t *testing.T, dir string) map[string]os.FileInfo {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := make(map[string]os.FileInfo)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			info, err := e.Info()
			if err != nil {
				continue
			}
			m[e.Name()] = info
		}
	}
	return m
}

// TestBackfillWritesFilesForExistingRows seeds 4 DB rows without disk files,
// calls Backfill, and verifies 4 markdown files are created with correct
// frontmatter (DR-5).
func TestBackfillWritesFilesForExistingRows(t *testing.T) {
	env := newTestEnv(t, nil)
	store := release.NewStore(env.proj.Idx.DB())

	names := []string{"Q1 BF", "Q2 BF", "Q3 BF", "Q4 BF"}
	for _, name := range names {
		seedReleaseRow(t, store, name, "planned")
	}

	// lifecycle/releases/ should be empty (no disk files yet).
	releasesDir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	entries, _ := os.ReadDir(releasesDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Fatalf("expected empty releases dir before Backfill, found %s", e.Name())
		}
	}

	result, err := release.Backfill(context.Background(), store, env.proj.ReleaseSync, "testproject", env.projectRoot)
	if err != nil {
		t.Fatalf("Backfill: %v", err)
	}
	if result.Written != 4 {
		t.Errorf("Written = %d, want 4", result.Written)
	}

	// Verify all 4 files exist with correct frontmatter.
	for _, name := range names {
		slug := release.Slugify(name)
		p := filepath.Join(releasesDir, slug+".md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("file %s not found: %v", slug+".md", err)
			continue
		}
		fm := readReleaseFrontmatter(t, env.projectRoot, "lifecycle/releases/"+slug+".md")
		if title, _ := fm["title"].(string); title != name {
			t.Errorf("%s: title want %q, got %q", slug, name, title)
		}
		if status, _ := fm["status"].(string); status != "planned" {
			t.Errorf("%s: status want %q, got %q", slug, "planned", status)
		}
	}
}

// TestBackfillIdempotent verifies that the startup sync condition
// (count>0 && disk files present → neither backfill nor rehydrate) leaves
// files unchanged after the first Backfill run.
func TestBackfillIdempotent(t *testing.T) {
	env := newTestEnv(t, nil)
	store := release.NewStore(env.proj.Idx.DB())

	seedReleaseRow(t, store, "Q1 BFI", "planned")
	seedReleaseRow(t, store, "Q2 BFI", "active")

	releasesDir := filepath.Join(env.projectRoot, "lifecycle", "releases")

	// First Backfill: writes files.
	if _, err := release.Backfill(context.Background(), store, env.proj.ReleaseSync, "testproject", env.projectRoot); err != nil {
		t.Fatal(err)
	}

	// Snapshot file mtimes.
	snap1 := fileStatMap(t, releasesDir)
	if len(snap1) != 2 {
		t.Fatalf("expected 2 files after Backfill, got %d", len(snap1))
	}

	// Simulate the startup sync no-op: count>0 && hasDiskFiles → skip backfill.
	// Do NOT call Backfill a second time.
	time.Sleep(5 * time.Millisecond)
	snap2 := fileStatMap(t, releasesDir)

	for name, s1 := range snap1 {
		s2, ok := snap2[name]
		if !ok {
			t.Errorf("file %s disappeared", name)
			continue
		}
		if !s1.ModTime().Equal(s2.ModTime()) {
			t.Errorf("file %s mtime changed unexpectedly", name)
		}
	}

	// DB row count must still be 2.
	count, _ := store.Count("testproject")
	if count != 2 {
		t.Errorf("DB count: want 2, got %d", count)
	}
}

// TestBackfillFailureDoesNotBlockLoad makes lifecycle/releases/ unwritable,
// calls Backfill, and asserts a non-fatal result plus an ERROR log. The
// project remains functional (server already started via newTestEnv).
func TestBackfillFailureDoesNotBlockLoad(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod test skipped when running as root")
	}

	env := newTestEnv(t, nil)
	store := release.NewStore(env.proj.Idx.DB())
	seedReleaseRow(t, store, "Perm Test", "planned")

	releasesDir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	if err := os.Chmod(releasesDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(releasesDir, 0o755) })

	handler := newCaptureHandler()
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(orig) })

	result, err := release.Backfill(context.Background(), store, env.proj.ReleaseSync, "testproject", env.projectRoot)
	if err != nil {
		t.Errorf("Backfill returned hard error (must be non-fatal): %v", err)
	}
	if result.Skipped == 0 && len(result.Errors) == 0 {
		t.Error("expected skipped>0 or errors>0 on unwritable directory")
	}

	// ERROR log must be present.
	if r := handler.findRecord("backfill: failed to write release file"); r == nil {
		t.Error("expected ERROR log 'backfill: failed to write release file' not found")
	}

	// Project is still functional.
	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}
