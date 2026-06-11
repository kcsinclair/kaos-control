// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/release"
)

// ── Milestone T5: Cross-machine reproducibility & artifact-parser sanity ─────

// TestCreateWipeRestartRehydrates creates 3 releases via the API (files on
// disk), opens a second project instance backed by a fresh data directory
// (simulating a wiped SQLite DB), and verifies that the startup rehydrate
// restores all 3 releases from disk (DR-3).
func TestCreateWipeRestartRehydrates(t *testing.T) {
	env := newTestEnv(t, nil)

	// Create 3 releases via API.
	wantNames := []string{"Roundtrip Alpha", "Roundtrip Beta", "Roundtrip Gamma"}
	for _, name := range wantNames {
		createRelease(t, env, map[string]any{"name": name, "status": "planned"})
	}

	// Verify disk files exist before simulating the wipe.
	for _, name := range wantNames {
		slug := release.Slugify(name)
		p := filepath.Join(env.projectRoot, "lifecycle", "releases", slug+".md")
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("release file for %q not on disk: %v", name, err)
		}
	}

	// Open a second project instance backed by a fresh (empty) data directory.
	// count=0 && disk files present → startup sync runs Rehydrate.
	freshDataDir := t.TempDir()
	entry := &config.ProjectEntry{
		Name: "testproject",
		Path: env.projectRoot,
	}
	proj2, err := project.Open(entry, freshDataDir, project.OpenOptions{
		DevopsLogDir: freshDataDir,
	})
	if err != nil {
		t.Fatalf("project.Open (fresh DB): %v", err)
	}
	defer proj2.Close()

	// Poll until the startup rehydrate goroutine inserts all 3 rows.
	store2 := release.NewStore(proj2.Idx.DB())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if n, _ := store2.Count("testproject"); n == 3 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	count, err := store2.Count("testproject")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("after rehydrate: want 3 DB rows, got %d", count)
	}

	releases, err := store2.List("testproject")
	if err != nil {
		t.Fatal(err)
	}
	gotNames := make(map[string]bool)
	for _, r := range releases {
		gotNames[r.Name] = true
	}
	for _, name := range wantNames {
		if !gotNames[name] {
			t.Errorf("release %q missing after rehydrate", name)
		}
	}
}

// TestArtifactParserAcceptsReleaseType seeds a release markdown file before
// project start, then verifies GET /parse-errors contains no error with
// message containing "unknown type \"release\"" (the type IS in KnownTypes).
func TestArtifactParserAcceptsReleaseType(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/releases/parser-check.md",
			content: makeReleaseMD("Parser Check Release", "planned"),
		},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/parse-errors", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	errs, _ := body["errors"].([]any)
	for _, e := range errs {
		errObj, _ := e.(map[string]any)
		msg, _ := errObj["message"].(string)
		if strings.Contains(msg, `unknown type "release"`) {
			t.Errorf("unexpected parse error for release type in %v: %s",
				errObj["path"], msg)
		}
	}
}
