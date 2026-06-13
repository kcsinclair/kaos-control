// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// debounceWait is 2 × the 150 ms watcher debounce plus a 100 ms processing buffer.
const debounceWait = 400 * time.Millisecond

// pollReleaseBySlug polls GET /releases until a release with the given slug
// appears (or not appears when wantPresent=false), returning early on success.
// Returns true on success, false on timeout.
func pollReleaseBySlug(env *testEnv, slug string, wantPresent bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
		if resp.StatusCode == http.StatusOK {
			body := readJSON(env.t, resp)
			relList, _ := body["releases"].([]any)
			found := false
			for _, r := range relList {
				rel, _ := r.(map[string]any)
				if s, _ := rel["slug"].(string); s == slug {
					found = true
					break
				}
			}
			if found == wantPresent {
				return true
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(30 * time.Millisecond)
	}
	return false
}

// ── Milestone T4: Watcher behaviour & WebSocket events ───────────────────────

// TestWatcherUpsertsFromDiskEdit writes a release .md file directly to disk
// (bypassing the API), waits for the watcher debounce, and asserts that
// GET /releases returns the new row and a release.changed WS event was received.
func TestWatcherUpsertsFromDiskEdit(t *testing.T) {
	env := newTestEnv(t, nil)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	slug := "watcher-upsert-ds"
	absPath := filepath.Join(env.projectRoot, "lifecycle", "releases", slug+".md")
	content := makeReleaseMD("Watcher Upsert DS", "planned")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for watcher debounce + handler.
	time.Sleep(debounceWait)

	if !pollReleaseBySlug(env, slug, true, 2*time.Second) {
		t.Fatalf("release %q did not appear in DB after watcher debounce", slug)
	}

	// Verify WS event.
	payload := collectReleaseEvent(ch, "release.changed", 2*time.Second)
	if payload == nil {
		t.Fatal("did not receive release.changed WS event")
	}
	rel, _ := payload["release"].(map[string]any)
	if name, _ := rel["name"].(string); name != "Watcher Upsert DS" {
		t.Errorf("release.changed payload.release.name: want %q, got %q", "Watcher Upsert DS", name)
	}
}

// TestWatcherDeletesRowOnFileRemoval creates a release via the API, then
// removes its disk file directly and verifies the DB row is deleted and a
// release.changed event with action:"deleted" is broadcast.
func TestWatcherDeletesRowOnFileRemoval(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{"name": "Watcher Del DS", "status": "planned"})
	rel, _ := data["release"].(map[string]any)
	slug, _ := rel["slug"].(string)
	if slug == "" {
		t.Fatal("release slug missing from create response")
	}

	absPath := filepath.Join(env.projectRoot, "lifecycle", "releases", slug+".md")
	if _, err := os.Stat(absPath); err != nil {
		t.Fatalf("release file should exist before removal: %v", err)
	}

	// Wait for the API-driven write event to be processed and consumed from ExpectedEvents
	// before we delete the file, otherwise the CREATE and REMOVE events will be coalesced.
	time.Sleep(debounceWait)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	if err := os.Remove(absPath); err != nil {
		t.Fatal(err)
	}

	time.Sleep(debounceWait)

	if !pollReleaseBySlug(env, slug, false, 2*time.Second) {
		t.Fatalf("release %q should be absent from DB after file removal", slug)
	}

	// WS should carry release.changed with action:"deleted".
	payload := collectReleaseEvent(ch, "release.changed", 2*time.Second)
	if payload == nil {
		t.Fatal("did not receive release.changed WS event after file removal")
	}
	if action, _ := payload["action"].(string); action != "deleted" {
		t.Errorf("release.changed payload.action: want %q, got %q", "deleted", action)
	}
}

// TestWatcherRenameUpdatesSlug renames a release file on disk and verifies
// the old slug is removed from the DB while the new slug is present.
// Note: the fsnotify rename emits separate REMOVE+CREATE events, so the row
// ID changes; this test only checks slug presence/absence.
func TestWatcherRenameUpdatesSlug(t *testing.T) {
	env := newTestEnv(t, nil)

	data := createRelease(t, env, map[string]any{"name": "Watcher Rename Old", "status": "planned"})
	rel, _ := data["release"].(map[string]any)
	oldSlug, _ := rel["slug"].(string)
	if oldSlug == "" {
		t.Fatal("old slug missing")
	}

	oldPath := filepath.Join(env.projectRoot, "lifecycle", "releases", oldSlug+".md")
	newSlug := "watcher-rename-new-ds"
	newPath := filepath.Join(env.projectRoot, "lifecycle", "releases", newSlug+".md")

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	// Wait for both REMOVE and CREATE debounce handlers to fire.
	time.Sleep(debounceWait * 2)

	if !pollReleaseBySlug(env, oldSlug, false, 2*time.Second) {
		t.Errorf("old slug %q should be absent after rename", oldSlug)
	}
	if !pollReleaseBySlug(env, newSlug, true, 2*time.Second) {
		t.Errorf("new slug %q should be present after rename", newSlug)
	}
}

// TestAPIWriteDoesNotProduceWatcherEvent creates a release via the API and
// waits 2× the watcher debounce window, then asserts that exactly one
// release.changed event was received (the API's own broadcast) and no
// duplicate from the watcher. Validates loop prevention via ExpectedEvents.
func TestAPIWriteDoesNotProduceWatcherEvent(t *testing.T) {
	env := newTestEnv(t, nil)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   "No Dupe DS",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Wait long enough that any watcher event would have fired.
	time.Sleep(debounceWait * 2)

	// Drain all events from the channel.
	count := 0
	draining := true
	for draining {
		select {
		case raw := <-ch:
			var evt struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &evt); err == nil && evt.Type == "release.changed" {
				count++
			}
		default:
			draining = false
		}
	}

	if count != 1 {
		t.Errorf("expected exactly 1 release.changed event (API only), got %d", count)
	}
}

// TestWatcherIgnoresInvalidFrontmatter writes a release file with an
// invalid status, waits for the watcher, and verifies no DB row was inserted
// and a WARN log was emitted.
func TestWatcherIgnoresInvalidFrontmatter(t *testing.T) {
	env := newTestEnv(t, nil)

	handler := newCaptureHandler()
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(orig) })

	absPath := filepath.Join(env.projectRoot, "lifecycle", "releases", "bad-status-ds.md")
	bad := "---\ntitle: Bad Status\ntype: release\nstatus: badstatus\nupdated_at: 2026-01-01T00:00:00Z\n---\n"
	if err := os.WriteFile(absPath, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(debounceWait)

	// No row should have been inserted.
	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	relList, _ := body["releases"].([]any)
	for _, r := range relList {
		rel, _ := r.(map[string]any)
		if slug, _ := rel["slug"].(string); slug == "bad-status-ds" {
			t.Error("invalid release file should not produce a DB row")
		}
	}

	// WARN log must be present.
	if r := handler.findRecord("release handler: skipping invalid release file"); r == nil {
		t.Error("expected WARN log 'release handler: skipping invalid release file' not found")
	}
}
