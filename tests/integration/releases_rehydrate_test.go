// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/release"
	_ "modernc.org/sqlite"
)

// makeReleaseMD returns minimal valid release markdown content.
func makeReleaseMD(title, status string) string {
	return fmt.Sprintf("---\ntitle: %s\ntype: release\nstatus: %s\nupdated_at: 2026-01-01T00:00:00Z\n---\n\nRelease notes.\n",
		title, status)
}

// ── Milestone T2: Rehydrate (disk→DB) and CLI trigger ────────────────────────

// TestRehydrateOnEmptyDB writes 3 release files to disk, calls POST /rehydrate,
// and verifies inserted==3 and GET /releases returns all 3 rows.
func TestRehydrateOnEmptyDB(t *testing.T) {
	env := newTestEnv(t, nil)

	dir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	files := []struct{ slug, title, status string }{
		{"alpha-1", "Alpha 1", "planned"},
		{"beta-2", "Beta 2", "active"},
		{"gamma-3", "Gamma 3", "shipped"},
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.slug+".md"),
			[]byte(makeReleaseMD(f.title, f.status)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	resp := env.doRequest("POST", "/api/p/testproject/releases/rehydrate", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	if inserted, _ := body["inserted"].(float64); int(inserted) != 3 {
		t.Errorf("inserted: want 3, got %d", int(inserted))
	}

	resp2 := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp2, http.StatusOK)
	body2 := readJSON(t, resp2)
	relList, _ := body2["releases"].([]any)
	if len(relList) != 3 {
		t.Fatalf("GET /releases: want 3 rows, got %d", len(relList))
	}

	wantTitles := map[string]bool{"Alpha 1": true, "Beta 2": true, "Gamma 3": true}
	for _, r := range relList {
		rel, _ := r.(map[string]any)
		if name, _ := rel["name"].(string); !wantTitles[name] {
			t.Errorf("unexpected release name %q", name)
		}
	}
}

// TestRehydrateOnEmptyDB_200FilesPerformance verifies that Rehydrate handles
// 200 files in under 250 ms (DR-7). Skipped in -short mode.
func TestRehydrateOnEmptyDB_200FilesPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 200-file performance budget in -short mode")
	}

	// Use a self-contained temp root to avoid watcher interference.
	root := t.TempDir()
	dir := filepath.Join(root, "lifecycle", "releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		slug := fmt.Sprintf("perf-release-%03d", i)
		title := fmt.Sprintf("Perf Release %03d", i)
		if err := os.WriteFile(filepath.Join(dir, slug+".md"),
			[]byte(makeReleaseMD(title, "planned")), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE releases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id TEXT NOT NULL, name TEXT NOT NULL,
			slug TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'planned',
			start_date TEXT, end_date TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
			UNIQUE(project_id, name)
		);
		CREATE UNIQUE INDEX idx_perf ON releases(project_id, slug) WHERE slug != '';
	`); err != nil {
		t.Fatal(err)
	}

	store := release.NewStore(db)
	start := time.Now()
	result, err := release.Rehydrate(context.Background(), store, "testproject", root)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Rehydrate: %v", err)
	}
	if result.Inserted != 200 {
		t.Errorf("Inserted = %d, want 200", result.Inserted)
	}
	if elapsed > 250*time.Millisecond {
		t.Errorf("Rehydrate of 200 files took %v, budget is 250ms", elapsed)
	}
}

// TestRehydrateSkipsInvalidFrontmatter places 2 valid + 1 invalid release file,
// calls POST /rehydrate, and asserts inserted==2, skipped==1 plus a WARN log.
func TestRehydrateSkipsInvalidFrontmatter(t *testing.T) {
	env := newTestEnv(t, nil)

	dir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	for _, f := range []struct{ slug, title string }{{"valid-a", "Valid A"}, {"valid-b", "Valid B"}} {
		if err := os.WriteFile(filepath.Join(dir, f.slug+".md"),
			[]byte(makeReleaseMD(f.title, "planned")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// end_date before start_date → parse error.
	bad := "---\ntitle: Bad Release\ntype: release\nstatus: planned\n" +
		"start_date: 2026-06-01\nend_date: 2026-01-01\nupdated_at: 2026-01-01T00:00:00Z\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "bad-dates.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := newCaptureHandler()
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(orig) })

	resp := env.doRequest("POST", "/api/p/testproject/releases/rehydrate", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	if inserted, _ := body["inserted"].(float64); int(inserted) != 2 {
		t.Errorf("inserted: want 2, got %d", int(inserted))
	}
	if skipped, _ := body["skipped"].(float64); int(skipped) != 1 {
		t.Errorf("skipped: want 1, got %d", int(skipped))
	}
	if r := handler.findRecord("rehydrate: skipping invalid release file"); r == nil {
		t.Error("expected WARN log 'rehydrate: skipping invalid release file' not found")
	}

	// Project is still functional.
	resp2 := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp2, http.StatusOK)
	resp2.Body.Close()
}

// TestRehydrateIdempotent calls POST /rehydrate twice and verifies the DB holds
// exactly 2 rows after both calls (DR-7 idempotency via ON CONFLICT DO UPDATE).
func TestRehydrateIdempotent(t *testing.T) {
	env := newTestEnv(t, nil)

	dir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	for _, f := range []struct{ slug, title string }{{"idem-a", "Idem A"}, {"idem-b", "Idem B"}} {
		if err := os.WriteFile(filepath.Join(dir, f.slug+".md"),
			[]byte(makeReleaseMD(f.title, "planned")), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 2; i++ {
		resp := env.doRequest("POST", "/api/p/testproject/releases/rehydrate", nil)
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	}

	store := release.NewStore(env.proj.Idx.DB())
	count, err := store.Count("testproject")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("DB count after two rehydrates: want 2, got %d", count)
	}
}

// TestRehydrateAPIRequiresAuth verifies that an unauthenticated POST to
// /releases/rehydrate is rejected with 401.
func TestRehydrateAPIRequiresAuth(t *testing.T) {
	env := newTestEnv(t, nil)
	env.logout()

	resp := env.doRequest("POST", "/api/p/testproject/releases/rehydrate", nil)
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}
