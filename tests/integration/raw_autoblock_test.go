// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for auto-block behaviour on 'raw' artefacts.
//
// A 'raw' artefact whose body contains "## Open Questions" must be auto-
// transitioned to 'blocked' by the watcher/indexer.  When the section is
// removed the artefact auto-unblocks to 'draft' (not back to 'raw') per the
// existing auto-unblock contract.
//
// Test plan: lifecycle/test-plans/raw-artefact-status-5-test.md §Milestone 4

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRawAutoBlock_WatcherTriggersBlock writes a 'raw' artefact with an
// "## Open Questions" section to disk and waits for the fsnotify watcher to
// auto-block it.  It then removes the section and confirms auto-unblock to
// 'draft' (the existing unblock contract — not back to 'raw').
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawAutoBlock_WatcherTriggersBlock
func TestRawAutoBlock_WatcherTriggersBlock(t *testing.T) {
	const relPath = "lifecycle/ideas/raw-watcher-autoblock.md"
	seeds := []seedArtifact{{
		relPath: relPath,
		content: makeArtifact("Raw Watcher AutoBlock", "idea", "raw", "raw-watcher-autoblock", "", "Initial raw body."),
	}}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	absPath := filepath.Join(env.projectRoot, relPath)

	// ── Step 1: rewrite file to add open questions ────────────────────────────
	withOQ := makeArtifact("Raw Watcher AutoBlock", "idea", "raw", "raw-watcher-autoblock", "",
		"## Open Questions\n\n- Why is this raw?\n")
	if err := os.WriteFile(absPath, []byte(withOQ), 0o644); err != nil {
		t.Fatalf("writing OQ file: %v", err)
	}

	// ── Step 2: poll GET until status is "blocked" (timeout 3s) ──────────────
	deadline := time.Now().Add(3 * time.Second)
	var blockedData map[string]any
	for time.Now().Before(deadline) {
		r := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		if r.StatusCode == 200 {
			d := readJSON(t, r)
			art, _ := d["artifact"].(map[string]any)
			fm, _ := art["frontmatter"].(map[string]any)
			if status, _ := fm["status"].(string); status == "blocked" {
				blockedData = d
				break
			}
		} else {
			r.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	if blockedData == nil {
		t.Fatal("raw artefact did not reach 'blocked' status within 3s after writing OQ section")
	}

	// ── Step 3: verify assignees include product-owner/agent ──────────────────
	art, _ := blockedData["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	assignees, _ := fm["assignees"].([]any)
	found := false
	for _, a := range assignees {
		entry, _ := a.(map[string]any)
		if entry["role"] == "product-owner" && entry["who"] == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected {role:product-owner, who:agent} in assignees after auto-block; got: %v", assignees)
	}

	// ── Step 4: rewrite file to remove open questions ─────────────────────────
	// Write with status "blocked" (as the on-disk file now has) but no OQ section.
	withoutOQ := makeArtifact("Raw Watcher AutoBlock", "idea", "blocked", "raw-watcher-autoblock", "",
		"Questions resolved.\n")
	if err := os.WriteFile(absPath, []byte(withoutOQ), 0o644); err != nil {
		t.Fatalf("writing no-OQ file: %v", err)
	}

	// ── Step 5: poll until status is "draft" (not "raw") ─────────────────────
	// Auto-unblock always goes to "draft", per the existing blocked→draft contract.
	deadline = time.Now().Add(3 * time.Second)
	var draftData map[string]any
	for time.Now().Before(deadline) {
		r := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		if r.StatusCode == 200 {
			d := readJSON(t, r)
			art2, _ := d["artifact"].(map[string]any)
			fm2, _ := art2["frontmatter"].(map[string]any)
			if status, _ := fm2["status"].(string); status == "draft" {
				draftData = d
				break
			}
		} else {
			r.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	if draftData == nil {
		t.Fatal("artefact did not return to 'draft' within 3s after removing OQ section (auto-unblock must go to draft, not raw)")
	}
}

// TestRawAutoBlock_StartupScanBlocksRawWithOQ verifies that the startup index
// scan auto-blocks a seeded 'raw' artefact that already contains open questions.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawAutoBlock_StartupScanBlocksRawWithOQ
func TestRawAutoBlock_StartupScanBlocksRawWithOQ(t *testing.T) {
	const relPath = "lifecycle/ideas/raw-startup-block.md"
	seeds := []seedArtifact{{
		relPath: relPath,
		content: makeArtifact("Raw Startup Block", "idea", "raw",
			"raw-startup-block", "",
			"## Open Questions\n\n- What should this do?\n"),
	}}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Startup scan is synchronous — the artefact must already be blocked.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "blocked" {
		t.Errorf("expected startup scan to auto-block the raw artefact, got status %q", status)
	}

	assignees, _ := fm["assignees"].([]any)
	found := false
	for _, a := range assignees {
		entry, _ := a.(map[string]any)
		if entry["role"] == "product-owner" && entry["who"] == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected {role:product-owner, who:agent} assignee after startup auto-block; got: %v", assignees)
	}
}

// TestRawAutoBlock_WSEventsOnWatcherBlock registers a hub listener before
// triggering a watcher-based auto-block on a 'raw' artefact and verifies that
// both "artifact.indexed" (with blocked_reason) and "feed.new" events arrive.
// It then triggers auto-unblock and checks the corresponding events.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawAutoBlock_WSEventsOnWatcherBlock
func TestRawAutoBlock_WSEventsOnWatcherBlock(t *testing.T) {
	const relPath = "lifecycle/ideas/raw-ws-autoblock.md"
	seeds := []seedArtifact{{
		relPath: relPath,
		content: makeArtifact("Raw WS AutoBlock", "idea", "raw", "raw-ws-autoblock", "", "Initial raw body."),
	}}
	env := newTestEnv(t, seeds)

	absPath := filepath.Join(env.projectRoot, relPath)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	drainHub := func(ch chan []byte, dur time.Duration) []wsEvent {
		deadline := time.Now().Add(dur)
		var collected []wsEvent
		for time.Now().Before(deadline) {
			select {
			case raw := <-ch:
				var evt wsEvent
				if err := json.Unmarshal(raw, &evt); err == nil {
					collected = append(collected, evt)
				}
			case <-time.After(50 * time.Millisecond):
				if time.Now().Before(deadline) {
					continue
				}
			}
		}
		return collected
	}

	// ── Trigger auto-block ────────────────────────────────────────────────────
	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	t.Cleanup(func() { env.proj.Hub.Unregister(ch) })

	withOQ := makeArtifact("Raw WS AutoBlock", "idea", "raw", "raw-ws-autoblock", "",
		"## Open Questions\n\n- Why?\n")
	if err := os.WriteFile(absPath, []byte(withOQ), 0o644); err != nil {
		t.Fatalf("writing OQ file: %v", err)
	}

	// Wait until the index shows "blocked".
	blockDeadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(blockDeadline) {
		row, _ := env.proj.Idx.Get(relPath)
		if row != nil && row.Status == "blocked" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	row, _ := env.proj.Idx.Get(relPath)
	if row == nil || row.Status != "blocked" {
		t.Fatal("raw artefact did not reach 'blocked' within 6s; cannot verify WS events")
	}

	blockEvents := drainHub(ch, 500*time.Millisecond)

	// Verify artifact.indexed with blocked_reason.
	gotBlockIndexed := false
	for _, evt := range blockEvents {
		if evt.Type != "artifact.indexed" {
			continue
		}
		p, _ := evt.Payload["path"].(string)
		reason, _ := evt.Payload["blocked_reason"].(string)
		if p == relPath && reason == "open_questions_detected" {
			gotBlockIndexed = true
			break
		}
	}
	if !gotBlockIndexed {
		t.Errorf("expected artifact.indexed{blocked_reason:open_questions_detected} for %s; got events: %v",
			relPath, blockEvents)
	}

	// Verify feed.new for auto-block.
	gotBlockFeed := false
	for _, evt := range blockEvents {
		if evt.Type != "feed.new" {
			continue
		}
		raw2, _ := json.Marshal(evt.Payload)
		if strings.Contains(string(raw2), "open_questions_detected") {
			gotBlockFeed = true
			break
		}
	}
	if !gotBlockFeed {
		t.Errorf("expected feed.new containing 'open_questions_detected'; got events: %v", blockEvents)
	}

	// ── Trigger auto-unblock ──────────────────────────────────────────────────
	ch2 := make(chan []byte, 256)
	env.proj.Hub.Register(ch2)
	t.Cleanup(func() { env.proj.Hub.Unregister(ch2) })

	withoutOQ := makeArtifact("Raw WS AutoBlock", "idea", "blocked", "raw-ws-autoblock", "",
		"Questions resolved.\n")
	if err := os.WriteFile(absPath, []byte(withoutOQ), 0o644); err != nil {
		t.Fatalf("writing no-OQ file: %v", err)
	}

	unblockDeadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(unblockDeadline) {
		row2, _ := env.proj.Idx.Get(relPath)
		if row2 != nil && row2.Status == "draft" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	row2, _ := env.proj.Idx.Get(relPath)
	if row2 == nil || row2.Status != "draft" {
		t.Fatal("artefact did not return to 'draft' within 6s; cannot verify unblock WS events")
	}

	unblockEvents := drainHub(ch2, 500*time.Millisecond)

	gotUnblockIndexed := false
	for _, evt := range unblockEvents {
		if evt.Type == "artifact.indexed" {
			if p, _ := evt.Payload["path"].(string); p == relPath {
				gotUnblockIndexed = true
				break
			}
		}
	}
	if !gotUnblockIndexed {
		t.Errorf("expected artifact.indexed event after raw auto-unblock; got events: %v", unblockEvents)
	}

	gotUnblockFeed := false
	for _, evt := range unblockEvents {
		if evt.Type != "feed.new" {
			continue
		}
		raw2, _ := json.Marshal(evt.Payload)
		if strings.Contains(string(raw2), "open_questions_resolved") {
			gotUnblockFeed = true
			break
		}
	}
	if !gotUnblockFeed {
		t.Errorf("expected feed.new containing 'open_questions_resolved' after raw auto-unblock; got events: %v", unblockEvents)
	}
}
