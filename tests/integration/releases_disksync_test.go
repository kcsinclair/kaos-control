// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// readReleaseFrontmatter reads and parses the YAML frontmatter from a release
// markdown file at the given project-relative path. Returns a generic map
// so callers can access any field without type assertions against generated structs.
func readReleaseFrontmatter(t *testing.T, projectRoot, relPath string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(projectRoot, relPath))
	if err != nil {
		t.Fatalf("readReleaseFrontmatter: read %s: %v", relPath, err)
	}
	s := string(content)
	if !strings.HasPrefix(s, "---") {
		t.Fatalf("release file %q missing frontmatter opening ---", relPath)
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		t.Fatalf("release file %q missing frontmatter closing ---", relPath)
	}
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		t.Fatalf("parsing frontmatter in %q: %v", relPath, err)
	}
	return fm
}

// apiErrorMessage extracts the nested message string from an apiError response:
// {"error": {"code": "...", "message": "..."}}.
func apiErrorMessage(body map[string]any) string {
	errObj, _ := body["error"].(map[string]any)
	msg, _ := errObj["message"].(string)
	return msg
}

// ── Milestone T1: DB→disk round-trip per HTTP verb ────────────────────────────

// TestReleaseCreate_WritesFile verifies that POST /releases with name "Q1 2026"
// returns file_path "lifecycle/releases/q1-2026.md", that file exists on disk,
// and its frontmatter has title, type, and status set correctly (DR-2).
func TestReleaseCreate_WritesFile(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{
		"name":   "Q1 2026",
		"status": "planned",
	})
	rel, _ := data["release"].(map[string]any)

	const wantFilePath = "lifecycle/releases/q1-2026.md"
	fp, _ := rel["file_path"].(string)
	if fp != wantFilePath {
		t.Errorf("file_path: want %q, got %q", wantFilePath, fp)
	}

	// File must exist on disk.
	if _, err := os.Stat(filepath.Join(env.projectRoot, wantFilePath)); err != nil {
		t.Fatalf("release file not on disk: %v", err)
	}

	// Frontmatter must match the request.
	fm := readReleaseFrontmatter(t, env.projectRoot, wantFilePath)
	if title, _ := fm["title"].(string); title != "Q1 2026" {
		t.Errorf("frontmatter title: want %q, got %q", "Q1 2026", title)
	}
	if typ, _ := fm["type"].(string); typ != "release" {
		t.Errorf("frontmatter type: want %q, got %q", "release", typ)
	}
	if status, _ := fm["status"].(string); status != "planned" {
		t.Errorf("frontmatter status: want %q, got %q", "planned", status)
	}
}

// fallbackSlugFileRe matches the fallback file path lifecycle/releases/release-{digits}.md.
var fallbackSlugFileRe = regexp.MustCompile(`^lifecycle/releases/release-\d+\.md$`)

// TestReleaseCreate_FallbackSlug verifies that POST with an emoji-only name
// assigns a fallback slug "release-{ID}" and the file exists at that path.
func TestReleaseCreate_FallbackSlug(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{
		"name":   "🚀",
		"status": "planned",
	})
	rel, _ := data["release"].(map[string]any)
	fp, _ := rel["file_path"].(string)

	if !fallbackSlugFileRe.MatchString(fp) {
		t.Errorf("file_path %q does not match fallback pattern %q", fp, fallbackSlugFileRe.String())
	}

	if _, err := os.Stat(filepath.Join(env.projectRoot, fp)); err != nil {
		t.Fatalf("fallback-slug release file not found at %s: %v", fp, err)
	}
}

// TestReleaseCreate_CollisionReturns409 verifies that a second POST with the same
// name returns 409, the body contains "release slug already in use", and no
// second file is created on disk.
func TestReleaseCreate_CollisionReturns409(t *testing.T) {
	env := newTestEnv(t, nil)

	createRelease(t, env, map[string]any{"name": "Q1-2026", "status": "planned"})

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   "Q1-2026",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusConflict)
	body := readJSON(t, resp)

	if msg := apiErrorMessage(body); !strings.Contains(msg, "release slug already in use") {
		t.Errorf("409 message should contain %q, got %q", "release slug already in use", msg)
	}

	// Only one .md file should exist.
	dir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var mdFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) != 1 {
		t.Errorf("expected exactly 1 .md file after collision, got %d: %v", len(mdFiles), mdFiles)
	}
}

// TestReleaseRenamePropagatesToDisk verifies that a PUT changing the name removes
// the old file and creates a new file with the updated slug and title.
func TestReleaseRenamePropagatesToDisk(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{"name": "Old Name DS", "status": "planned"})
	id := releaseID(t, data)
	rel, _ := data["release"].(map[string]any)
	oldFilePath, _ := rel["file_path"].(string)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "New Name DS",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	updatedRel, _ := body["release"].(map[string]any)
	newFilePath, _ := updatedRel["file_path"].(string)

	if oldFilePath == "" || newFilePath == "" {
		t.Fatalf("file_path missing in response (old=%q new=%q)", oldFilePath, newFilePath)
	}

	// Old file must be gone.
	if _, err := os.Stat(filepath.Join(env.projectRoot, oldFilePath)); !os.IsNotExist(err) {
		t.Errorf("old release file %q should be absent after rename (stat err: %v)", oldFilePath, err)
	}

	// New file must exist.
	if _, err := os.Stat(filepath.Join(env.projectRoot, newFilePath)); err != nil {
		t.Errorf("new release file %q should exist after rename: %v", newFilePath, err)
	}

	// New file frontmatter title must be updated.
	fm := readReleaseFrontmatter(t, env.projectRoot, newFilePath)
	if title, _ := fm["title"].(string); title != "New Name DS" {
		t.Errorf("frontmatter title: want %q, got %q", "New Name DS", title)
	}
}

// TestReleaseInPlaceEditDoesNotRenameFile verifies that a PUT changing only
// start_date keeps the filename unchanged and updates the frontmatter dates.
func TestReleaseInPlaceEditDoesNotRenameFile(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{
		"name":       "Q2 2026 DS",
		"status":     "planned",
		"start_date": "2026-04-01",
		"end_date":   "2026-06-30",
	})
	id := releaseID(t, data)
	rel, _ := data["release"].(map[string]any)
	filePath, _ := rel["file_path"].(string) // lifecycle/releases/q2-2026-ds.md

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":       "Q2 2026 DS",
		"status":     "planned",
		"start_date": "2026-05-01",
		"end_date":   "2026-07-31",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// File at original path must still exist.
	if _, err := os.Stat(filepath.Join(env.projectRoot, filePath)); err != nil {
		t.Errorf("release file %q must remain after in-place edit: %v", filePath, err)
	}

	// Frontmatter must reflect updated dates.
	fm := readReleaseFrontmatter(t, env.projectRoot, filePath)
	if sd, _ := fm["start_date"].(string); sd != "2026-05-01" {
		t.Errorf("frontmatter start_date: want %q, got %q", "2026-05-01", sd)
	}
	if ed, _ := fm["end_date"].(string); ed != "2026-07-31" {
		t.Errorf("frontmatter end_date: want %q, got %q", "2026-07-31", ed)
	}
}

// TestReleaseDeleteRemovesFile verifies that DELETE /releases/:id removes the
// associated markdown file from disk (DR-6).
func TestReleaseDeleteRemovesFile(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{"name": "To Delete DS", "status": "planned"})
	id := releaseID(t, data)
	rel, _ := data["release"].(map[string]any)
	filePath, _ := rel["file_path"].(string)

	// Confirm file exists before delete.
	if _, err := os.Stat(filepath.Join(env.projectRoot, filePath)); err != nil {
		t.Fatalf("release file should exist before delete: %v", err)
	}

	resp := env.doRequest("DELETE", releasePath(id), nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// File must be absent after delete.
	if _, err := os.Stat(filepath.Join(env.projectRoot, filePath)); !os.IsNotExist(err) {
		t.Errorf("release file %q should be absent after DELETE (stat err: %v)", filePath, err)
	}
}

// TestReleaseUpdateWithStaleUpdatedAt_Returns409 verifies that a PUT with a
// stale updated_at returns 409 with "release was modified" (DR-2 optimistic lock).
func TestReleaseUpdateWithStaleUpdatedAt_Returns409(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{"name": "Concurrent DS", "status": "planned"})
	id := releaseID(t, data)

	// A timestamp well in the past simulates a stale client.
	stale := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":       "Concurrent DS",
		"status":     "active",
		"updated_at": stale,
	})
	requireStatus(t, resp, http.StatusConflict)
	body := readJSON(t, resp)

	if msg := apiErrorMessage(body); !strings.Contains(msg, "release was modified") {
		t.Errorf("409 message should contain %q, got %q", "release was modified", msg)
	}
}

// TestReleaseStatusUnscheduledAccepted verifies that POST with status "unscheduled"
// returns 201, the response status is "unscheduled", and the disk frontmatter matches.
func TestReleaseStatusUnscheduledAccepted(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{
		"name":   "Unscheduled DS",
		"status": "unscheduled",
	})
	rel, _ := data["release"].(map[string]any)
	filePath, _ := rel["file_path"].(string)

	if status, _ := rel["status"].(string); status != "unscheduled" {
		t.Errorf("response status: want %q, got %q", "unscheduled", status)
	}

	if _, err := os.Stat(filepath.Join(env.projectRoot, filePath)); err != nil {
		t.Fatalf("release file not found: %v", err)
	}

	fm := readReleaseFrontmatter(t, env.projectRoot, filePath)
	if status, _ := fm["status"].(string); status != "unscheduled" {
		t.Errorf("frontmatter status: want %q, got %q", "unscheduled", status)
	}
}
