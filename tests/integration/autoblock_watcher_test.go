// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── Milestone 4 — Watcher-triggered auto-block ────────────────────────────────

// TestAutoBlock_WatcherTriggersBlock writes a file with "## Open Questions"
// to disk and waits for the fsnotify watcher to pick it up and auto-block the
// artifact. It then removes the questions section and confirms the artifact is
// auto-unblocked to "draft".
//
// Run with: go test ./tests/integration/... -tags=integration -run TestAutoBlock_WatcherTriggersBlock
func TestAutoBlock_WatcherTriggersBlock(t *testing.T) {
	const relPath = "lifecycle/ideas/watcher-autoblock.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Watcher AutoBlock", "idea", "draft", "watcher-autoblock", "", "Initial body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	absPath := filepath.Join(env.projectRoot, relPath)

	// ── Step 1: rewrite the file on disk to add open questions ───────────────
	withOQ := makeArtifact("Watcher AutoBlock", "idea", "draft", "watcher-autoblock", "",
		"## Open Questions\n\n- Why is X?\n")
	if err := os.WriteFile(absPath, []byte(withOQ), 0o644); err != nil {
		t.Fatalf("writing OQ file: %v", err)
	}

	// ── Step 2: poll GET until status is "blocked" (timeout 3s) ─────────────
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
		t.Fatal("artifact did not reach 'blocked' status within 3s after writing OQ section")
	}

	// ── Step 3: assert assignees include product-owner/agent ─────────────────
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

	// ── Step 4: rewrite the file to remove open questions ────────────────────
	withoutOQ := makeArtifact("Watcher AutoBlock", "idea", "blocked", "watcher-autoblock", "",
		"Questions resolved.\n")
	if err := os.WriteFile(absPath, []byte(withoutOQ), 0o644); err != nil {
		t.Fatalf("writing no-OQ file: %v", err)
	}

	// ── Step 5: poll until status returns to "draft" ─────────────────────────
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
		t.Fatal("artifact did not return to 'draft' status within 3s after removing OQ section")
	}
}

// ── Milestone 7 — Manual transitions are not broken ──────────────────────────

// TestAutoBlock_ManualTransitionsUnaffected verifies that the auto-block
// feature does not prevent or corrupt manual workflow transitions on artifacts
// that have no open questions. A product-owner transitions an artifact through
// a standard path; no spurious auto-block events are generated.
//
// Note on design: manually blocking an artifact that has no open questions
// triggers an immediate auto-unblock (status reverts to draft), because the
// indexer fires applyOpenQuestionTransition synchronously during every
// IndexFile call. This is the expected system behaviour; the test documents it.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestAutoBlock_ManualTransitionsUnaffected
func TestAutoBlock_ManualTransitionsUnaffected(t *testing.T) {
	const relPath = "lifecycle/ideas/manual-transition.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Manual Transition", "idea", "draft", "manual-transition", "", "No open questions."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// ── Manual draft → clarifying (no auto-block involvement) ────────────────
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, 200)
	d := readJSON(t, resp)
	art, _ := d["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	if status, _ := fm["status"].(string); status != "clarifying" {
		t.Errorf("expected status 'clarifying' after manual transition, got %q", status)
	}

	// ── Manual clarifying → draft ─────────────────────────────────────────────
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/transition",
		map[string]any{"to": "draft"})
	requireStatus(t, resp, 200)
	d = readJSON(t, resp)
	art, _ = d["artifact"].(map[string]any)
	fm, _ = art["frontmatter"].(map[string]any)
	if status, _ := fm["status"].(string); status != "draft" {
		t.Errorf("expected status 'draft' after manual transition back, got %q", status)
	}

	// ── No spurious auto-block events (artifact has no OQ) ───────────────────
	// Wait briefly for any watcher events to settle before checking.
	time.Sleep(300 * time.Millisecond)

	feedResp := env.doRequest("GET", "/api/p/testproject/feed?limit=20", nil)
	requireStatus(t, feedResp, 200)
	feedData := readJSON(t, feedResp)
	events, _ := feedData["events"].([]any)
	for _, ev := range events {
		entry, _ := ev.(map[string]any)
		if entry["event_type"] == "status_changed" {
			p, _ := entry["artifact_path"].(string)
			if p == relPath {
				payload, _ := entry["payload_json"].(string)
				t.Errorf("unexpected auto-block status_changed event for no-OQ artifact (payload: %s)", payload)
			}
		}
	}
}

// TestAutoBlock_ManualBlockRevertsBecauseNoOQ documents the interaction
// between a manual block transition and the auto-unblock logic: manually
// transitioning a no-OQ artifact to "blocked" is immediately overridden by
// auto-unblock because the indexer fires synchronously. The final status is
// "draft", not "blocked".
//
// Run with: go test ./tests/integration/... -tags=integration -run TestAutoBlock_ManualBlockRevertsBecauseNoOQ
func TestAutoBlock_ManualBlockRevertsBecauseNoOQ(t *testing.T) {
	const relPath = "lifecycle/ideas/manual-block-reverts.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Manual Block Reverts", "idea", "draft", "manual-block-reverts", "", "No open questions."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Manually transition to blocked. Because the artifact has no OQ, the
	// indexer immediately auto-unblocks it; the final status is draft.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/transition",
		map[string]any{"to": "blocked"})
	requireStatus(t, resp, 200)
	d := readJSON(t, resp)
	art, _ := d["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	status, _ := fm["status"].(string)
	// Auto-unblock fires synchronously inside applyTransition → IndexFile, so
	// the response must already show "draft".
	if status != "draft" {
		t.Errorf("expected auto-unblock to revert manual block to 'draft', got %q", status)
	}
}

// ── Milestone 8 — WebSocket event verification ───────────────────────────────

// TestAutoBlock_WSEventsOnWatcherBlock registers a hub listener before
// triggering a watcher-based auto-block and verifies that both
// "artifact.indexed" (with blocked_reason) and "feed.new" events arrive.
// It then triggers auto-unblock and checks for the corresponding events.
//
// drainHub collects all hub events for up to drainFor after the index state
// is confirmed, ensuring no events are missed from the buffered channel.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestAutoBlock_WSEventsOnWatcherBlock
func TestAutoBlock_WSEventsOnWatcherBlock(t *testing.T) {
	const relPath = "lifecycle/ideas/ws-autoblock.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("WS AutoBlock", "idea", "draft", "ws-autoblock", "", "Initial body."),
		},
	}
	env := newTestEnv(t, seeds)

	absPath := filepath.Join(env.projectRoot, relPath)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	// drainHub collects all events from ch for up to dur after calling. It
	// returns when the timer fires (no pending items). Uses a select with a
	// short per-item timeout so all buffered events are consumed.
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
				// No new event in 50ms; channel is quiet.
				if time.Now().Before(deadline) {
					// Keep waiting until deadline in case more events arrive.
					continue
				}
			}
		}
		return collected
	}

	// ── Trigger auto-block via disk write ────────────────────────────────────
	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	t.Cleanup(func() { env.proj.Hub.Unregister(ch) })

	withOQ := makeArtifact("WS AutoBlock", "idea", "draft", "ws-autoblock", "",
		"## Open Questions\n\n- Why?\n")
	if err := os.WriteFile(absPath, []byte(withOQ), 0o644); err != nil {
		t.Fatalf("writing OQ file: %v", err)
	}

	// Wait until the index shows "blocked" (confirms auto-block fired).
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
		t.Fatal("artifact did not reach 'blocked' status within 6s; cannot verify WS events")
	}

	// Drain all hub events that arrived during the block (allow 500ms for
	// any in-flight broadcasts from atomicWrite's watcher re-trigger).
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

	// ── Trigger auto-unblock via disk write ──────────────────────────────────
	// Use a fresh channel so block-phase residual events don't bleed in.
	ch2 := make(chan []byte, 256)
	env.proj.Hub.Register(ch2)
	t.Cleanup(func() { env.proj.Hub.Unregister(ch2) })

	withoutOQ := makeArtifact("WS AutoBlock", "idea", "blocked", "ws-autoblock", "",
		"Questions resolved.\n")
	if err := os.WriteFile(absPath, []byte(withoutOQ), 0o644); err != nil {
		t.Fatalf("writing no-OQ file: %v", err)
	}

	// Wait until the index shows "draft".
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
		t.Fatal("artifact did not return to 'draft' within 6s; cannot verify unblock WS events")
	}

	unblockEvents := drainHub(ch2, 500*time.Millisecond)

	// Verify artifact.indexed for the path.
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
		t.Errorf("expected artifact.indexed event after auto-unblock; got events: %v", unblockEvents)
	}

	// Verify feed.new for auto-unblock.
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
		t.Errorf("expected feed.new containing 'open_questions_resolved'; got events: %v", unblockEvents)
	}
}
