//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExternalEditPickedUp verifies that dropping a file into lifecycle/
// directly triggers the fsnotify watcher, which re-indexes the artifact
// within a reasonable time.
// Test plan §7: "External edit" scenario.
func TestExternalEditPickedUp(t *testing.T) {
	env := newTestEnv(t, nil)

	// Verify the file does not exist in the index.
	row, err := env.proj.Idx.Get("lifecycle/ideas/external.md")
	if err != nil {
		t.Fatal(err)
	}
	if row != nil {
		t.Fatal("expected external.md to not be in index before external edit")
	}

	// Write a file directly on disk (simulating an external editor).
	content := makeArtifact("External Edit", "idea", "draft", "external", "", "Written by an external tool.")
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "external.md")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to pick it up (150ms debounce + processing time).
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		row, err = env.proj.Idx.Get("lifecycle/ideas/external.md")
		if err != nil {
			t.Fatal(err)
		}
		if row != nil {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !found {
		t.Error("watcher did not pick up externally written file within 2s")
	}

	if row != nil {
		if row.Title != "External Edit" {
			t.Errorf("expected title 'External Edit', got %q", row.Title)
		}
		if row.Status != "draft" {
			t.Errorf("expected status 'draft', got %q", row.Status)
		}
	}
}

// TestExternalEditUpdateExisting verifies that modifying an existing file
// on disk is detected by the watcher and the index is updated.
func TestExternalEditUpdateExisting(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/watched.md",
			content: makeArtifact("Watched Idea", "idea", "draft", "watched", "", "Original content."),
		},
	}
	env := newTestEnv(t, seeds)

	// Verify initial state.
	row, _ := env.proj.Idx.Get("lifecycle/ideas/watched.md")
	if row == nil {
		t.Fatal("seed artifact not indexed")
	}
	if row.Title != "Watched Idea" {
		t.Fatalf("unexpected initial title: %q", row.Title)
	}

	// Overwrite the file on disk with a new title.
	updated := makeArtifact("Watched Idea Updated", "idea", "draft", "watched", "", "Modified content.")
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "watched.md")
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to pick it up.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, _ = env.proj.Idx.Get("lifecycle/ideas/watched.md")
		if row != nil && row.Title == "Watched Idea Updated" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if row == nil || row.Title != "Watched Idea Updated" {
		t.Errorf("expected updated title 'Watched Idea Updated', got %v", row)
	}
}

// ── Milestone 2 — Auto-refresh: backend API returns updated content ───────────

// TestAutoRefreshReadMode verifies the end-to-end backend half of the
// auto-refresh flow: seed an artifact, fetch it via the API (record file_sha),
// modify the file on disk, wait for the watcher to re-index it, then confirm
// the API returns the updated content and a different file_sha.
func TestAutoRefreshReadMode(t *testing.T) {
	const relPath = "lifecycle/ideas/autorefresh.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Auto Refresh Original", "idea", "draft", "autorefresh", "", "Original body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Fetch the artifact and record the initial file_sha.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	initialSHA, _ := data["file_sha"].(string)
	if initialSHA == "" {
		t.Fatal("initial file_sha is empty")
	}

	// Modify the file on disk (external editor simulation).
	updated := makeArtifact("Auto Refresh Updated", "idea", "draft", "autorefresh", "", "Updated body.")
	absPath := filepath.Join(env.projectRoot, relPath)
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Poll the API until the index reflects the updated title (watcher debounce
	// is 150 ms + processing; allow up to 5 s total).
	deadline := time.Now().Add(5 * time.Second)
	var finalData map[string]any
	for time.Now().Before(deadline) {
		r := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		if r.StatusCode == 200 {
			finalData = readJSON(t, r)
			art, _ := finalData["artifact"].(map[string]any)
			if title, _ := art["title"].(string); title == "Auto Refresh Updated" {
				break
			}
		} else {
			r.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	if finalData == nil {
		t.Fatal("artifact not found in API after disk change")
	}
	art, _ := finalData["artifact"].(map[string]any)
	if title, _ := art["title"].(string); title != "Auto Refresh Updated" {
		t.Errorf("expected updated title 'Auto Refresh Updated', got %q", title)
	}
	updatedSHA, _ := finalData["file_sha"].(string)
	if updatedSHA == "" {
		t.Error("updated file_sha is empty")
	}
	if updatedSHA == initialSHA {
		t.Errorf("file_sha did not change after disk update (still %q)", initialSHA)
	}
}

// TestRapidWritesCoalesce verifies that writing the same file three times in
// quick succession (< 150 ms apart) results in the final content being served
// by the API after a single re-index round.
func TestRapidWritesCoalesce(t *testing.T) {
	const relPath = "lifecycle/ideas/rapidcoalesce.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Rapid Write v0", "idea", "draft", "rapidcoalesce", "", "v0"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	absPath := filepath.Join(env.projectRoot, relPath)

	// Write three times < 150 ms apart (watcher debounce is 150 ms, so they
	// should be coalesced into a single re-index round).
	for i, title := range []string{"Rapid Write v1", "Rapid Write v2", "Rapid Write v3"} {
		content := makeArtifact(title, "idea", "draft", "rapidcoalesce", "", "body "+title)
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		time.Sleep(40 * time.Millisecond)
	}

	// Poll until the index and API return the last-written title.
	deadline := time.Now().Add(5 * time.Second)
	var finalTitle string
	for time.Now().Before(deadline) {
		r := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		if r.StatusCode == 200 {
			d := readJSON(t, r)
			art, _ := d["artifact"].(map[string]any)
			if t2, _ := art["title"].(string); t2 == "Rapid Write v3" {
				finalTitle = t2
				break
			}
		} else {
			r.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	if finalTitle != "Rapid Write v3" {
		t.Errorf("expected final title 'Rapid Write v3', got %q", finalTitle)
	}
}

// ── Milestone 3 — External edit while lineage is locked ───────────────────

// TestExternalEditWhileLocked verifies that holding a lineage lock does not
// prevent the watcher from re-indexing an externally modified file, and that
// the API serves the updated disk content regardless of the lock.
func TestExternalEditWhileLocked(t *testing.T) {
	const relPath = "lifecycle/ideas/locked-edit.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Locked Edit Original", "idea", "draft", "locked-edit", "", "Original body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Acquire an editor lock on the lineage.
	lockResp := env.doRequest("POST", "/api/p/testproject/locks", map[string]any{
		"lineage": "locked-edit",
		"kind":    "editor",
	})
	requireStatus(t, lockResp, 200)
	lockResp.Body.Close()

	// Fetch the artifact and record the initial file_sha.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	initialSHA, _ := data["file_sha"].(string)

	// Modify the file on disk (simulating an external editor while the lock is held).
	updated := makeArtifact("Locked Edit Updated", "idea", "draft", "locked-edit", "", "Updated while locked.")
	absPath := filepath.Join(env.projectRoot, relPath)
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Poll until the index and API reflect the disk change.
	deadline := time.Now().Add(5 * time.Second)
	var finalData map[string]any
	for time.Now().Before(deadline) {
		r := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		if r.StatusCode == 200 {
			finalData = readJSON(t, r)
			art, _ := finalData["artifact"].(map[string]any)
			if title, _ := art["title"].(string); title == "Locked Edit Updated" {
				break
			}
		} else {
			r.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	if finalData == nil {
		t.Fatal("artifact not accessible via API while lock is held")
	}
	art, _ := finalData["artifact"].(map[string]any)
	if title, _ := art["title"].(string); title != "Locked Edit Updated" {
		t.Errorf("expected updated title 'Locked Edit Updated', got %q", title)
	}
	if sha, _ := finalData["file_sha"].(string); sha == initialSHA {
		t.Errorf("file_sha did not change after locked external edit (still %q)", initialSHA)
	}

	// Release the lock and verify the artifact is still accessible.
	releaseResp := env.doRequest("DELETE", "/api/p/testproject/locks/locked-edit", nil)
	requireStatus(t, releaseResp, 204)
	releaseResp.Body.Close()

	verifyResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, verifyResp, 200)
	verifyData := readJSON(t, verifyResp)
	verifyArt, _ := verifyData["artifact"].(map[string]any)
	if title, _ := verifyArt["title"].(string); title != "Locked Edit Updated" {
		t.Errorf("artifact inaccessible or stale after lock release, got title %q", title)
	}
}

// ── Milestone 4 — Save does not self-suppress file.changed at the backend ──

// TestSaveDoesNotSelfTrigger verifies that saving an artifact via PUT causes
// the watcher to emit a file.changed WebSocket event.  The backend does NOT
// suppress this event — suppression is the frontend's job via SAVE_GRACE_MS.
// This test documents that contract.
func TestSaveDoesNotSelfTrigger(t *testing.T) {
	const relPath = "lifecycle/ideas/save-trigger.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Save Trigger", "idea", "draft", "save-trigger", "", "Original body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Register a hub channel before saving so no events are missed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Save the artifact via PUT.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Save Trigger Updated",
			"type":    "idea",
			"status":  "draft",
			"lineage": "save-trigger",
		},
		"body": "Saved body.",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Collect events for up to 4 s and look for a file.changed event whose
	// path matches the saved artifact.
	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	timeout := time.After(4 * time.Second)
	var gotFileChanged bool
	var gotPath string
COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "file.changed" {
				p, _ := evt.Payload["path"].(string)
				if p == relPath {
					gotFileChanged = true
					gotPath = p
					break COLLECT
				}
			}
		case <-timeout:
			break COLLECT
		}
	}

	// The backend MUST emit the event — the frontend is responsible for the
	// grace-window filter.
	if !gotFileChanged {
		t.Error("expected a file.changed event after PUT save, but none arrived within 4 s")
	}
	if gotPath != relPath {
		t.Errorf("file.changed event path: got %q, want %q", gotPath, relPath)
	}
}

// TestExternalDeleteRemovesFromIndex verifies that deleting a file on disk
// causes the watcher to remove it from the index.
func TestExternalDeleteRemovesFromIndex(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/ephemeral.md",
			content: makeArtifact("Ephemeral", "idea", "draft", "ephemeral", "", "Will be deleted."),
		},
	}
	env := newTestEnv(t, seeds)

	// Verify it's indexed.
	row, _ := env.proj.Idx.Get("lifecycle/ideas/ephemeral.md")
	if row == nil {
		t.Fatal("seed artifact not indexed")
	}

	// Delete the file on disk.
	absPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "ephemeral.md")
	if err := os.Remove(absPath); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to remove it from the index.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, _ = env.proj.Idx.Get("lifecycle/ideas/ephemeral.md")
		if row == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if row != nil {
		t.Error("expected deleted file to be removed from index")
	}
}
