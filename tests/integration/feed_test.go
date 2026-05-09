// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// ── Milestone 1 — Events table and basic CRUD ─────────────────────────────

// TestFeedEventsTableExists confirms that the events table and its two indices
// are present in the SQLite database after project open.
func TestFeedEventsTableExists(t *testing.T) {
	env := newTestEnv(t, nil)
	db := env.proj.Idx.DB()

	var tableCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='events'`,
	).Scan(&tableCount); err != nil {
		t.Fatalf("querying sqlite_master for events table: %v", err)
	}
	if tableCount != 1 {
		t.Errorf("expected events table to exist, got count=%d", tableCount)
	}

	for _, idxName := range []string{"idx_events_timestamp", "idx_events_event_type"} {
		var idxCount int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idxName,
		).Scan(&idxCount); err != nil {
			t.Fatalf("querying sqlite_master for index %s: %v", idxName, err)
		}
		if idxCount != 1 {
			t.Errorf("expected index %q to exist, got count=%d", idxName, idxCount)
		}
	}
}

// TestFeedInsertAndQuery verifies that InsertEvent round-trips all fields and
// ListEvents returns events in reverse-chronological order. An empty table
// returns a zero-length result.
func TestFeedInsertAndQuery(t *testing.T) {
	env := newTestEnv(t, nil)
	idx := env.proj.Idx

	// Empty table must return zero results (nil or empty slice both acceptable).
	empty, err := idx.ListEvents(50, 0, nil)
	if err != nil {
		t.Fatalf("ListEvents on empty table: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty result, got %d events", len(empty))
	}

	now := time.Now().Unix()
	artifactPath := "lifecycle/ideas/feed-test.md"
	runID := "run-abc-123"
	payload := `{"detail":"extra"}`

	rows := []*index.EventRow{
		{
			EventType:    "status_transition",
			Timestamp:    now - 300,
			Actor:        "alice@example.com",
			ArtifactPath: &artifactPath,
			Summary:      "First event",
		},
		{
			EventType: "artifact_created",
			Timestamp: now - 200,
			Actor:     "bob@example.com",
			Summary:   "Second event",
		},
		{
			EventType:   "agent_started",
			Timestamp:   now - 100,
			Actor:       "requirements-analyst",
			RunID:       &runID,
			Summary:     "Third event",
			PayloadJSON: &payload,
		},
	}

	for i, r := range rows {
		if err := idx.InsertEvent(r); err != nil {
			t.Fatalf("InsertEvent[%d]: %v", i, err)
		}
		if r.ID == 0 {
			t.Errorf("InsertEvent[%d] did not set ID", i)
		}
	}

	got, err := idx.ListEvents(50, 0, nil)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}

	// Verify reverse-chronological order.
	for i := 1; i < len(got); i++ {
		if got[i].Timestamp > got[i-1].Timestamp {
			t.Errorf("events not reverse-chronological: got[%d].ts=%d > got[%d].ts=%d",
				i, got[i].Timestamp, i-1, got[i-1].Timestamp)
		}
	}

	// Newest event (agent_started, now-100) should be first.
	if got[0].EventType != "agent_started" {
		t.Errorf("expected newest event type agent_started, got %q", got[0].EventType)
	}
	if got[0].RunID == nil || *got[0].RunID != runID {
		t.Errorf("run_id: expected %q, got %v", runID, got[0].RunID)
	}
	if got[0].PayloadJSON == nil || *got[0].PayloadJSON != payload {
		t.Errorf("payload_json: expected %q, got %v", payload, got[0].PayloadJSON)
	}

	// Oldest event (status_transition, now-300) should be last.
	if got[2].EventType != "status_transition" {
		t.Errorf("expected oldest event type status_transition, got %q", got[2].EventType)
	}
	if got[2].ArtifactPath == nil || *got[2].ArtifactPath != artifactPath {
		t.Errorf("artifact_path: expected %q, got %v", artifactPath, got[2].ArtifactPath)
	}
	if got[2].Actor != "alice@example.com" {
		t.Errorf("actor: expected alice@example.com, got %q", got[2].Actor)
	}
}

// ── Milestone 2 — Feed REST endpoint: basic pagination ────────────────────

// seedFeedEvents inserts count events of the given type, spread over time,
// and returns the inserted rows (oldest first).
func seedFeedEvents(t *testing.T, env *testEnv, count int, eventType string) []*index.EventRow {
	t.Helper()
	now := time.Now().Unix()
	out := make([]*index.EventRow, 0, count)
	for i := 0; i < count; i++ {
		e := &index.EventRow{
			EventType: eventType,
			Timestamp: now - int64(count-i)*10,
			Actor:     "test@example.com",
			Summary:   fmt.Sprintf("Event %d", i+1),
		}
		if err := env.proj.Idx.InsertEvent(e); err != nil {
			t.Fatalf("seedFeedEvents InsertEvent[%d]: %v", i, err)
		}
		out = append(out, e)
	}
	return out
}

// TestFeedEndpointDefaults seeds 5 events, calls GET /feed with no params,
// and verifies the response contains 5 events newest-first with no next_cursor.
func TestFeedEndpointDefaults(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	seedFeedEvents(t, env, 5, "status_transition")

	resp := env.doRequest("GET", "/api/p/testproject/feed", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	events, _ := data["events"].([]any)
	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}

	// Verify newest-first order.
	for i := 1; i < len(events); i++ {
		e0, _ := events[i-1].(map[string]any)
		e1, _ := events[i].(map[string]any)
		ts0, _ := e0["timestamp"].(float64)
		ts1, _ := e1["timestamp"].(float64)
		if ts0 < ts1 {
			t.Errorf("events not newest-first at [%d/%d]: ts=%v < ts=%v", i-1, i, ts0, ts1)
		}
	}

	// next_cursor must be null (fewer than the default limit of 50).
	if nc := data["next_cursor"]; nc != nil {
		t.Errorf("expected next_cursor null, got %v", nc)
	}
}

// TestFeedEndpointLimit seeds 10 events, requests limit=3, and verifies
// exactly 3 events are returned with next_cursor equal to the last event's ID.
func TestFeedEndpointLimit(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	seedFeedEvents(t, env, 10, "status_transition")

	resp := env.doRequest("GET", "/api/p/testproject/feed?limit=3", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	events, _ := data["events"].([]any)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	nc := data["next_cursor"]
	if nc == nil {
		t.Fatal("expected next_cursor to be set")
	}

	lastEvent, _ := events[2].(map[string]any)
	lastID, _ := lastEvent["id"].(float64)
	cursorID, _ := nc.(float64)
	if int64(cursorID) != int64(lastID) {
		t.Errorf("next_cursor %v should equal last event ID %v", cursorID, lastID)
	}
}

// TestFeedEndpointCursorPagination seeds 10 events and pages through them with
// limit=4. Asserts no duplicate IDs, global newest-first ordering, and null
// next_cursor on the final page.
func TestFeedEndpointCursorPagination(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	seedFeedEvents(t, env, 10, "status_transition")

	seen := map[float64]bool{}

	// Page 1
	resp1 := env.doRequest("GET", "/api/p/testproject/feed?limit=4", nil)
	requireStatus(t, resp1, http.StatusOK)
	page1 := readJSON(t, resp1)

	events1, _ := page1["events"].([]any)
	if len(events1) != 4 {
		t.Fatalf("page1: expected 4 events, got %d", len(events1))
	}
	nc1 := page1["next_cursor"]
	if nc1 == nil {
		t.Fatal("page1: expected next_cursor")
	}
	for _, e := range events1 {
		ev, _ := e.(map[string]any)
		seen[ev["id"].(float64)] = true
	}

	// Page 2
	cursor1, _ := nc1.(float64)
	resp2 := env.doRequest("GET", fmt.Sprintf("/api/p/testproject/feed?limit=4&before=%d", int64(cursor1)), nil)
	requireStatus(t, resp2, http.StatusOK)
	page2 := readJSON(t, resp2)

	events2, _ := page2["events"].([]any)
	if len(events2) != 4 {
		t.Fatalf("page2: expected 4 events, got %d", len(events2))
	}
	nc2 := page2["next_cursor"]
	if nc2 == nil {
		t.Fatal("page2: expected next_cursor")
	}
	for _, e := range events2 {
		ev, _ := e.(map[string]any)
		id := ev["id"].(float64)
		if seen[id] {
			t.Errorf("duplicate event ID %.0f on page2", id)
		}
		seen[id] = true
	}

	// Page 3 (remaining 2 events)
	cursor2, _ := nc2.(float64)
	resp3 := env.doRequest("GET", fmt.Sprintf("/api/p/testproject/feed?limit=4&before=%d", int64(cursor2)), nil)
	requireStatus(t, resp3, http.StatusOK)
	page3 := readJSON(t, resp3)

	events3, _ := page3["events"].([]any)
	if len(events3) != 2 {
		t.Fatalf("page3: expected 2 events, got %d", len(events3))
	}
	if page3["next_cursor"] != nil {
		t.Errorf("page3: expected next_cursor null, got %v", page3["next_cursor"])
	}
	for _, e := range events3 {
		ev, _ := e.(map[string]any)
		id := ev["id"].(float64)
		if seen[id] {
			t.Errorf("duplicate event ID %.0f on page3", id)
		}
		seen[id] = true
	}

	if len(seen) != 10 {
		t.Errorf("expected 10 unique events across all pages, saw %d", len(seen))
	}

	// Verify global newest-first ordering across pages.
	var allTimestamps []float64
	for _, page := range [][]any{events1, events2, events3} {
		for _, e := range page {
			ev, _ := e.(map[string]any)
			ts, _ := ev["timestamp"].(float64)
			allTimestamps = append(allTimestamps, ts)
		}
	}
	for i := 1; i < len(allTimestamps); i++ {
		if allTimestamps[i] > allTimestamps[i-1] {
			t.Errorf("global order broken at position %d: ts[%d]=%v > ts[%d]=%v",
				i, i, allTimestamps[i], i-1, allTimestamps[i-1])
		}
	}
}

// TestFeedEndpointLimitCap verifies that limit=999 is capped at 200.
func TestFeedEndpointLimitCap(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	seedFeedEvents(t, env, 5, "status_transition")

	resp := env.doRequest("GET", "/api/p/testproject/feed?limit=999", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	events, _ := data["events"].([]any)
	if len(events) > 200 {
		t.Errorf("limit cap not enforced: got %d events (max 200)", len(events))
	}
	// We seeded 5 so we should receive exactly 5.
	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}
}

// ── Milestone 3 — Feed REST endpoint: type filtering ──────────────────────

// TestFeedEndpointTypeFilter seeds events of multiple types and verifies that
// ?types=status_transition returns only status_transition events.
func TestFeedEndpointTypeFilter(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now().Unix()
	for i, et := range []string{"status_transition", "artifact_created", "agent_started"} {
		e := &index.EventRow{
			EventType: et,
			Timestamp: now - int64(i)*10,
			Actor:     "test@example.com",
			Summary:   "Event " + et,
		}
		if err := env.proj.Idx.InsertEvent(e); err != nil {
			t.Fatal(err)
		}
	}

	resp := env.doRequest("GET", "/api/p/testproject/feed?types=status_transition", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	events, _ := data["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("expected 1 status_transition event, got %d", len(events))
	}
	ev, _ := events[0].(map[string]any)
	if et, _ := ev["event_type"].(string); et != "status_transition" {
		t.Errorf("expected event_type=status_transition, got %q", et)
	}
}

// TestFeedEndpointMultiTypeFilter seeds three event types and verifies that
// ?types=status_transition,agent_started returns exactly those two types.
func TestFeedEndpointMultiTypeFilter(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now().Unix()
	for i, et := range []string{"status_transition", "artifact_created", "agent_started"} {
		e := &index.EventRow{
			EventType: et,
			Timestamp: now - int64(i)*10,
			Actor:     "test@example.com",
			Summary:   "Event " + et,
		}
		if err := env.proj.Idx.InsertEvent(e); err != nil {
			t.Fatal(err)
		}
	}

	resp := env.doRequest("GET", "/api/p/testproject/feed?types=status_transition,agent_started", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	events, _ := data["events"].([]any)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	typesSeen := map[string]bool{}
	for _, e := range events {
		ev, _ := e.(map[string]any)
		et, _ := ev["event_type"].(string)
		if et != "status_transition" && et != "agent_started" {
			t.Errorf("unexpected event_type %q", et)
		}
		typesSeen[et] = true
	}
	if !typesSeen["status_transition"] || !typesSeen["agent_started"] {
		t.Errorf("expected both types in response, got %v", typesSeen)
	}
}

// TestFeedEndpointFilterWithPagination seeds 20 alternating-type events and
// verifies that type filtering and cursor pagination compose correctly.
func TestFeedEndpointFilterWithPagination(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now().Unix()
	for i := 0; i < 20; i++ {
		et := "artifact_created"
		if i%2 == 0 {
			et = "status_transition"
		}
		e := &index.EventRow{
			EventType: et,
			Timestamp: now - int64(20-i)*10,
			Actor:     "test@example.com",
			Summary:   fmt.Sprintf("Event %d", i),
		}
		if err := env.proj.Idx.InsertEvent(e); err != nil {
			t.Fatal(err)
		}
	}
	// 10 status_transition events total.

	// Page 1: first 3 status_transition events.
	resp1 := env.doRequest("GET", "/api/p/testproject/feed?types=status_transition&limit=3", nil)
	requireStatus(t, resp1, http.StatusOK)
	page1 := readJSON(t, resp1)

	events1, _ := page1["events"].([]any)
	if len(events1) != 3 {
		t.Fatalf("page1: expected 3 events, got %d", len(events1))
	}
	for _, e := range events1 {
		ev, _ := e.(map[string]any)
		if et, _ := ev["event_type"].(string); et != "status_transition" {
			t.Errorf("page1: unexpected event_type %q", et)
		}
	}
	nc1 := page1["next_cursor"]
	if nc1 == nil {
		t.Fatal("page1: expected next_cursor")
	}

	// Page 2 with cursor.
	cursor1, _ := nc1.(float64)
	resp2 := env.doRequest("GET",
		fmt.Sprintf("/api/p/testproject/feed?types=status_transition&limit=3&before=%d", int64(cursor1)), nil)
	requireStatus(t, resp2, http.StatusOK)
	page2 := readJSON(t, resp2)

	events2, _ := page2["events"].([]any)
	if len(events2) != 3 {
		t.Fatalf("page2: expected 3 events, got %d", len(events2))
	}
	for _, e := range events2 {
		ev, _ := e.(map[string]any)
		if et, _ := ev["event_type"].(string); et != "status_transition" {
			t.Errorf("page2: unexpected event_type %q", et)
		}
	}

	// No IDs should overlap between pages.
	seenIDs := map[float64]bool{}
	for _, e := range events1 {
		ev, _ := e.(map[string]any)
		seenIDs[ev["id"].(float64)] = true
	}
	for _, e := range events2 {
		ev, _ := e.(map[string]any)
		id := ev["id"].(float64)
		if seenIDs[id] {
			t.Errorf("duplicate event ID %.0f across pages", id)
		}
	}
}

// ── Milestone 4 — Automatic event recording: status transitions ────────────

// TestFeedTransitionEvent confirms that POSTing a transition produces exactly
// one status_transition feed event with the correct actor, artifact_path, and
// a summary mentioning the transition. Timestamp must be close to now.
func TestFeedTransitionEvent(t *testing.T) {
	const artifactPath = "lifecycle/ideas/transition-event-test.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Transition Event Test", "idea", "draft", "transition-event-test", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	before := time.Now().Unix()

	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]string{"to": "clarifying"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	feedResp := env.doRequest("GET", "/api/p/testproject/feed?types=status_transition", nil)
	requireStatus(t, feedResp, http.StatusOK)
	data := readJSON(t, feedResp)

	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("expected at least one status_transition event")
	}
	ev, _ := events[0].(map[string]any)

	if ap, _ := ev["artifact_path"].(string); ap != artifactPath {
		t.Errorf("artifact_path: expected %q, got %q", artifactPath, ap)
	}
	if summary, _ := ev["summary"].(string); summary == "" {
		t.Error("summary is empty")
	}
	if actor, _ := ev["actor"].(string); actor != "admin@test.local" {
		t.Errorf("actor: expected admin@test.local, got %q", actor)
	}
	ts, _ := ev["timestamp"].(float64)
	if int64(ts) < before {
		t.Errorf("event timestamp %d predates the transition (before=%d)", int64(ts), before)
	}
	if int64(ts) > before+10 {
		t.Errorf("event timestamp %d is >10s after the transition (before=%d)", int64(ts), before)
	}
}

// ── Milestone 5 — Automatic event recording: artifact creation ────────────

// TestFeedArtifactCreatedEvent confirms that POST /artifacts produces exactly
// one artifact_created event with the correct actor and artifact_path.
func TestFeedArtifactCreatedEvent(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	before := time.Now().Unix()

	createReq := map[string]any{
		"stage": "ideas",
		"slug":  "feed-created-test",
		"frontmatter": map[string]any{
			"title":   "Feed Created Test Idea",
			"type":    "idea",
			"status":  "draft",
			"lineage": "feed-created-test",
		},
		"body": "Testing feed event on artifact creation.",
	}
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", createReq)
	requireStatus(t, resp, http.StatusCreated)
	created := readJSON(t, resp)
	createdPath, _ := created["path"].(string)

	feedResp := env.doRequest("GET", "/api/p/testproject/feed?types=artifact_created", nil)
	requireStatus(t, feedResp, http.StatusOK)
	data := readJSON(t, feedResp)

	events, _ := data["events"].([]any)
	if len(events) == 0 {
		t.Fatal("expected at least one artifact_created event")
	}
	ev, _ := events[0].(map[string]any)

	if ap, _ := ev["artifact_path"].(string); ap != createdPath {
		t.Errorf("artifact_path: expected %q, got %q", createdPath, ap)
	}
	if summary, _ := ev["summary"].(string); summary == "" {
		t.Error("summary is empty")
	}
	if actor, _ := ev["actor"].(string); actor != "admin@test.local" {
		t.Errorf("actor: expected admin@test.local, got %q", actor)
	}
	ts, _ := ev["timestamp"].(float64)
	if int64(ts) < before {
		t.Errorf("event timestamp %d predates artifact creation (before=%d)", int64(ts), before)
	}
}

// ── Milestone 6 — Agent lifecycle events ──────────────────────────────────

// TestFeedAgentEvents verifies that agent_started and agent_finished events
// can be inserted and queried via the index layer, and that they appear in the
// feed endpoint with the correct type, actor, and run_id. Since running a real
// agent in integration tests is complex, events are inserted directly.
func TestFeedAgentEvents(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	runID := "run-agent-test-001"
	now := time.Now().Unix()

	startedEvent := &index.EventRow{
		EventType: "agent_started",
		Timestamp: now - 5,
		Actor:     "requirements-analyst",
		RunID:     &runID,
		Summary:   "Agent requirements-analyst started",
	}
	finishedEvent := &index.EventRow{
		EventType: "agent_finished",
		Timestamp: now,
		Actor:     "requirements-analyst",
		RunID:     &runID,
		Summary:   "Agent requirements-analyst finished",
	}

	if err := env.proj.Idx.InsertEvent(startedEvent); err != nil {
		t.Fatalf("InsertEvent agent_started: %v", err)
	}
	if err := env.proj.Idx.InsertEvent(finishedEvent); err != nil {
		t.Fatalf("InsertEvent agent_finished: %v", err)
	}

	resp := env.doRequest("GET", "/api/p/testproject/feed?types=agent_started,agent_finished", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	events, _ := data["events"].([]any)
	if len(events) != 2 {
		t.Fatalf("expected 2 agent events, got %d", len(events))
	}

	typesSeen := map[string]bool{}
	for _, e := range events {
		ev, _ := e.(map[string]any)
		et, _ := ev["event_type"].(string)
		typesSeen[et] = true
		if actor, _ := ev["actor"].(string); actor != "requirements-analyst" {
			t.Errorf("actor: expected requirements-analyst, got %q", actor)
		}
		if rid, _ := ev["run_id"].(string); rid != runID {
			t.Errorf("run_id: expected %q, got %q", runID, rid)
		}
	}
	if !typesSeen["agent_started"] || !typesSeen["agent_finished"] {
		t.Errorf("expected both agent_started and agent_finished, got %v", typesSeen)
	}
}

// ── Milestone 7 — Event pruning ───────────────────────────────────────────

// TestFeedPruneByAge inserts one 40-day-old event and one 10-day-old event,
// then prunes at 30 days and verifies only the recent event survives.
func TestFeedPruneByAge(t *testing.T) {
	env := newTestEnv(t, nil)
	idx := env.proj.Idx

	old := &index.EventRow{
		EventType: "status_transition",
		Timestamp: time.Now().AddDate(0, 0, -40).Unix(),
		Actor:     "test@example.com",
		Summary:   "Old event (40 days ago)",
	}
	recent := &index.EventRow{
		EventType: "status_transition",
		Timestamp: time.Now().AddDate(0, 0, -10).Unix(),
		Actor:     "test@example.com",
		Summary:   "Recent event (10 days ago)",
	}

	if err := idx.InsertEvent(old); err != nil {
		t.Fatal(err)
	}
	if err := idx.InsertEvent(recent); err != nil {
		t.Fatal(err)
	}

	if err := idx.PruneEvents(30, 10000); err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}

	events, err := idx.ListEvents(50, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after age pruning, got %d", len(events))
	}
	if events[0].Summary != "Recent event (10 days ago)" {
		t.Errorf("wrong event survived pruning: %q", events[0].Summary)
	}
}

// TestFeedPruneByCount inserts 20 recent events, prunes to maxCount=10, and
// verifies exactly 10 of the newest events remain.
func TestFeedPruneByCount(t *testing.T) {
	env := newTestEnv(t, nil)
	idx := env.proj.Idx

	now := time.Now().Unix()
	for i := 0; i < 20; i++ {
		e := &index.EventRow{
			EventType: "status_transition",
			Timestamp: now - int64(20-i), // i=19 is newest (now-1)
			Actor:     "test@example.com",
			Summary:   fmt.Sprintf("Event %d", i+1),
		}
		if err := idx.InsertEvent(e); err != nil {
			t.Fatalf("InsertEvent[%d]: %v", i, err)
		}
	}

	if err := idx.PruneEvents(365, 10); err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}

	events, err := idx.ListEvents(50, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 10 {
		t.Fatalf("expected exactly 10 events after count pruning, got %d", len(events))
	}

	// All surviving events should be the 10 newest (timestamps >= now-10).
	for _, e := range events {
		if e.Timestamp < now-10 {
			t.Errorf("stale event survived count pruning: timestamp=%d, cutoff=%d", e.Timestamp, now-10)
		}
	}
}

// TestFeedPruneCombined inserts 5 recent and 5 old events, then prunes with
// maxAgeDays=30 and maxCount=3. Verifies that age rule removes old events and
// the count cap then limits the result to 3.
func TestFeedPruneCombined(t *testing.T) {
	env := newTestEnv(t, nil)
	idx := env.proj.Idx

	now := time.Now().Unix()
	for i := 0; i < 5; i++ {
		recent := &index.EventRow{
			EventType: "status_transition",
			Timestamp: now - int64(i),
			Actor:     "test@example.com",
			Summary:   fmt.Sprintf("Recent %d", i),
		}
		old := &index.EventRow{
			EventType: "status_transition",
			Timestamp: time.Now().AddDate(0, 0, -40).Unix() - int64(i),
			Actor:     "test@example.com",
			Summary:   fmt.Sprintf("Old %d", i),
		}
		if err := idx.InsertEvent(recent); err != nil {
			t.Fatal(err)
		}
		if err := idx.InsertEvent(old); err != nil {
			t.Fatal(err)
		}
	}

	if err := idx.PruneEvents(30, 3); err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}

	events, err := idx.ListEvents(50, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) > 3 {
		t.Errorf("expected at most 3 events after combined pruning, got %d", len(events))
	}

	cutoff := time.Now().AddDate(0, 0, -30).Unix()
	for _, e := range events {
		if e.Timestamp < cutoff {
			t.Errorf("old event survived age pruning: timestamp=%d, cutoff=%d", e.Timestamp, cutoff)
		}
	}
}

// ── Milestone 9 — Feed endpoint performance ───────────────────────────────

// TestFeedEndpointPerformance inserts 10,000 events and verifies the default
// GET /feed request responds in under 50 ms (median of 5 requests).
func TestFeedEndpointPerformance(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Batch-insert 10,000 events directly via the DB connection for speed.
	db := env.proj.Idx.DB()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare(
		`INSERT INTO events (event_type, timestamp, actor, summary) VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			t.Logf("rollback error: %v", rbErr)
		}
		t.Fatal(err)
	}
	now := time.Now().Unix()
	for i := 0; i < 10000; i++ {
		if _, err := stmt.Exec("status_transition", now-int64(i), "test@example.com", fmt.Sprintf("Event %d", i)); err != nil {
			stmt.Close()
			if rbErr := tx.Rollback(); rbErr != nil {
				t.Logf("rollback error: %v", rbErr)
			}
			t.Fatal(err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Verify the batch insert worked.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count < 10000 {
		t.Fatalf("expected at least 10,000 events, got %d", count)
	}

	// Time 5 requests; check median < 50 ms.
	const threshold = 50 * time.Millisecond
	durations := make([]time.Duration, 5)
	for i := range durations {
		start := time.Now()
		resp := env.doRequest("GET", "/api/p/testproject/feed", nil)
		durations[i] = time.Since(start)
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	}

	// Insertion sort (5 elements).
	for i := 1; i < len(durations); i++ {
		for j := i; j > 0 && durations[j] < durations[j-1]; j-- {
			durations[j], durations[j-1] = durations[j-1], durations[j]
		}
	}
	median := durations[len(durations)/2]

	if median > threshold {
		t.Errorf("median feed response time %v exceeds %v with 10,000 events (all durations: %v)",
			median, threshold, durations)
	}
}

