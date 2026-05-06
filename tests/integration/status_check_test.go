//go:build integration

package integration

// Tests for the lineage status checker feature — Milestones 2 and 3.
//
// Covered API endpoints:
//   GET  /api/p/{project}/status-check[?lineage=slug]
//   POST /api/p/{project}/status-check/advance
//
// Staleness definition: an artifact is stale when ALL of its non-terminal
// children have advanced past the artifact's own status. Terminal statuses
// (rejected, abandoned, done) are excluded from the comparison. The suggested
// status is the minimum non-terminal child status (i.e. the smallest step the
// parent needs to take to catch up).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// staleChild describes one direct child artifact included in a staleEntry.
type staleChild struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// staleEntry represents one entry in the stale array returned by GET /status-check.
type staleEntry struct {
	Path            string       `json:"path"`
	Lineage         string       `json:"lineage"`
	CurrentStatus   string       `json:"current_status"`
	SuggestedStatus string       `json:"suggested_status"`
	CanAdvance      bool         `json:"can_advance"`
	BlockedReason   string       `json:"blocked_reason"`
	Children        []staleChild `json:"children"`
}

// advanceResult represents one entry in the results array returned by POST /advance.
type advanceResult struct {
	Path       string `json:"path"`
	Outcome    string `json:"outcome"` // "advanced" | "skipped" | "error"
	Ok         bool   `json:"ok"`
	AdvancedTo string `json:"advanced_to,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// decodeStaleEntries decodes the "stale" field from a /status-check response body.
func decodeStaleEntries(t *testing.T, data map[string]any) []staleEntry {
	t.Helper()
	raw, ok := data["stale"]
	if !ok {
		t.Fatal("response missing 'stale' field")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("re-marshalling stale field: %v", err)
	}
	var entries []staleEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		t.Fatalf("decoding stale entries: %v", err)
	}
	return entries
}

// decodeAdvanceResults decodes the "results" field from a /advance response body.
func decodeAdvanceResults(t *testing.T, data map[string]any) []advanceResult {
	t.Helper()
	raw, ok := data["results"]
	if !ok {
		t.Fatal("response missing 'results' field")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("re-marshalling results field: %v", err)
	}
	var results []advanceResult
	if err := json.Unmarshal(b, &results); err != nil {
		t.Fatalf("decoding advance results: %v", err)
	}
	return results
}

// findStaleByPath returns the staleEntry whose path matches, or nil.
func findStaleByPath(entries []staleEntry, path string) *staleEntry {
	for i := range entries {
		if entries[i].Path == path {
			return &entries[i]
		}
	}
	return nil
}

// ── Milestone 2: GET /status-check ───────────────────────────────────────────

// TestStatusCheck_SingleLineage verifies that a single stale parent is returned
// when its children have all advanced past its status, and that filtering by
// lineage slug returns only that lineage's stale artifacts.
func TestStatusCheck_SingleLineage(t *testing.T) {
	// Lineage: idea (draft) → requirement (planning)
	// The idea is stale: its child is at planning, parent is at draft.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-single.md",
			content: makeArtifact("SC Single Idea", "idea", "draft", "sc-single", "", "Originating idea."),
		},
		{
			relPath: "lifecycle/requirements/sc-single-2.md",
			content: makeArtifact("SC Single Req", "ticket", "planning", "sc-single",
				"lifecycle/ideas/sc-single.md", "Requirement body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=sc-single", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	entries := decodeStaleEntries(t, data)

	// The idea should be stale; the requirement should not (no children).
	entry := findStaleByPath(entries, "lifecycle/ideas/sc-single.md")
	if entry == nil {
		t.Fatal("expected lifecycle/ideas/sc-single.md to appear in stale list")
	}
	if entry.CurrentStatus != "draft" {
		t.Errorf("current_status: want %q, got %q", "draft", entry.CurrentStatus)
	}
	if entry.SuggestedStatus != "planning" {
		t.Errorf("suggested_status: want %q, got %q", "planning", entry.SuggestedStatus)
	}
	if entry.Lineage != "sc-single" {
		t.Errorf("lineage: want %q, got %q", "sc-single", entry.Lineage)
	}

	// Filtering should exclude other lineages (none seeded here, but verify count).
	for _, e := range entries {
		if e.Lineage != "sc-single" {
			t.Errorf("lineage filter returned artifact from unexpected lineage %q", e.Lineage)
		}
	}
}

// TestStatusCheck_ProjectWide verifies that a project-wide check (no lineage
// filter) returns stale artifacts across multiple lineages and omits lineages
// where nothing is stale.
func TestStatusCheck_ProjectWide(t *testing.T) {
	// Lineage A: stale — idea (draft), child req (planning)
	// Lineage B: stale — idea (draft), child req (clarifying)
	// Lineage C: not stale — idea (planning), child req (planning)
	seeds := []seedArtifact{
		// A
		{
			relPath: "lifecycle/ideas/sc-pw-a.md",
			content: makeArtifact("SC PW A Idea", "idea", "draft", "sc-pw-a", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-pw-a-2.md",
			content: makeArtifact("SC PW A Req", "ticket", "planning", "sc-pw-a",
				"lifecycle/ideas/sc-pw-a.md", "Body."),
		},
		// B
		{
			relPath: "lifecycle/ideas/sc-pw-b.md",
			content: makeArtifact("SC PW B Idea", "idea", "draft", "sc-pw-b", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-pw-b-2.md",
			content: makeArtifact("SC PW B Req", "ticket", "clarifying", "sc-pw-b",
				"lifecycle/ideas/sc-pw-b.md", "Body."),
		},
		// C
		{
			relPath: "lifecycle/ideas/sc-pw-c.md",
			content: makeArtifact("SC PW C Idea", "idea", "planning", "sc-pw-c", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-pw-c-2.md",
			content: makeArtifact("SC PW C Req", "ticket", "planning", "sc-pw-c",
				"lifecycle/ideas/sc-pw-c.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	// Lineage A and B should have stale artifacts; C should not.
	aEntry := findStaleByPath(entries, "lifecycle/ideas/sc-pw-a.md")
	bEntry := findStaleByPath(entries, "lifecycle/ideas/sc-pw-b.md")
	cEntry := findStaleByPath(entries, "lifecycle/ideas/sc-pw-c.md")

	if aEntry == nil {
		t.Error("expected sc-pw-a idea to be stale")
	} else if aEntry.SuggestedStatus != "planning" {
		t.Errorf("sc-pw-a suggested_status: want planning, got %q", aEntry.SuggestedStatus)
	}

	if bEntry == nil {
		t.Error("expected sc-pw-b idea to be stale")
	} else if bEntry.SuggestedStatus != "clarifying" {
		t.Errorf("sc-pw-b suggested_status: want clarifying, got %q", bEntry.SuggestedStatus)
	}

	if cEntry != nil {
		t.Errorf("sc-pw-c idea should NOT be stale (parent and child both at planning)")
	}
}

// TestStatusCheck_NoStaleness verifies that when all artifacts are current,
// the stale array is empty (not null).
func TestStatusCheck_NoStaleness(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-nostale.md",
			content: makeArtifact("SC No Stale", "idea", "planning", "sc-nostale", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-nostale-2.md",
			content: makeArtifact("SC No Stale Req", "ticket", "planning", "sc-nostale",
				"lifecycle/ideas/sc-nostale.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	if len(entries) != 0 {
		t.Errorf("expected empty stale list, got %d entries", len(entries))
	}
}

// TestStatusCheck_CanAdvancePermissions verifies that when a user lacks the
// roles required to make the suggested transition, can_advance is false and
// blocked_reason is non-empty.
//
// Setup: idea at draft, child requirement at clarifying.
// Suggested transition: draft → clarifying (requires product-owner or analyst).
// Dev user holds backend-developer/frontend-developer/test-developer — cannot advance.
func TestStatusCheck_CanAdvancePermissions(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-perm.md",
			content: makeArtifact("SC Perm Idea", "idea", "draft", "sc-perm", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-perm-2.md",
			content: makeArtifact("SC Perm Req", "ticket", "clarifying", "sc-perm",
				"lifecycle/ideas/sc-perm.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("dev@test.local", "dev-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	entry := findStaleByPath(entries, "lifecycle/ideas/sc-perm.md")
	if entry == nil {
		t.Fatal("expected lifecycle/ideas/sc-perm.md in stale list")
	}
	if entry.CanAdvance {
		t.Error("can_advance should be false for dev user (lacks product-owner/analyst role)")
	}
	if entry.BlockedReason == "" {
		t.Error("blocked_reason should be non-empty when can_advance is false")
	}

	// Admin (product-owner) should be able to advance.
	env.login("admin@test.local", "admin-pass-123")
	resp = env.doRequest("GET", "/api/p/testproject/status-check", nil)
	requireStatus(t, resp, 200)
	data = readJSON(t, resp)
	entries = decodeStaleEntries(t, data)

	adminEntry := findStaleByPath(entries, "lifecycle/ideas/sc-perm.md")
	if adminEntry == nil {
		t.Fatal("expected lifecycle/ideas/sc-perm.md in stale list for admin")
	}
	if !adminEntry.CanAdvance {
		t.Error("can_advance should be true for admin (holds product-owner role)")
	}
}

// TestStatusCheck_Performance seeds 1 000 artifacts across 10 lineages and
// verifies that GET /status-check responds in under 500 ms.
func TestStatusCheck_Performance(t *testing.T) {
	const numLineages = 10
	const artifactsPerLineage = 100 // 1 000 total

	seeds := make([]seedArtifact, 0, numLineages*artifactsPerLineage)
	for l := 0; l < numLineages; l++ {
		slug := fmt.Sprintf("sc-perf-%03d", l)
		// One idea (draft) as the lineage root.
		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("lifecycle/ideas/%s.md", slug),
			content: makeArtifact(
				fmt.Sprintf("SC Perf Idea %d", l),
				"idea", "draft", slug, "", "Performance test body.",
			),
		})
		// Remaining artifacts in requirements, alternating draft / planning.
		for i := 1; i < artifactsPerLineage; i++ {
			status := "draft"
			if i%2 == 0 {
				status = "planning"
			}
			name := fmt.Sprintf("%s-%d", slug, i+1)
			seeds = append(seeds, seedArtifact{
				relPath: fmt.Sprintf("lifecycle/requirements/%s.md", name),
				content: makeArtifact(
					fmt.Sprintf("SC Perf Req %d/%d", l, i),
					"ticket", status, slug,
					fmt.Sprintf("lifecycle/ideas/%s.md", slug),
					"Body.",
				),
			})
		}
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/status-check", nil)
	elapsed := time.Since(start)

	requireStatus(t, resp, 200)
	if elapsed > 500*time.Millisecond {
		t.Errorf("status-check took %v; must respond within 500 ms for %d artifacts",
			elapsed, numLineages*artifactsPerLineage)
	}
	t.Logf("status-check with %d artifacts responded in %v", numLineages*artifactsPerLineage, elapsed)
}

// ── Milestone 3: POST /status-check/advance ──────────────────────────────────

// TestAdvance_Single advances one stale artifact and verifies the status is
// updated on disk and in the index.
func TestAdvance_Single(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-single.md",
			content: makeArtifact("SC Adv Single Idea", "idea", "draft", "sc-adv-single", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-single-2.md",
			content: makeArtifact("SC Adv Single Req", "ticket", "clarifying", "sc-adv-single",
				"lifecycle/ideas/sc-adv-single.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const ideaPath = "lifecycle/ideas/sc-adv-single.md"

	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": []string{ideaPath},
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != ideaPath {
		t.Errorf("result path: want %q, got %q", ideaPath, results[0].Path)
	}
	if results[0].Outcome != "advanced" {
		t.Errorf("outcome: want %q, got %q", "advanced", results[0].Outcome)
	}
	if results[0].AdvancedTo != "clarifying" {
		t.Errorf("advanced_to: want %q, got %q", "clarifying", results[0].AdvancedTo)
	}

	// Verify status updated on disk.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: clarifying") {
		t.Error("status not updated to 'clarifying' in file on disk")
	}

	// Verify status updated in index.
	row, err := env.proj.Idx.Get(ideaPath)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("artifact not found in index after advance")
	}
	if row.Status != "clarifying" {
		t.Errorf("index status: want %q, got %q", "clarifying", row.Status)
	}
}

// TestAdvance_MultipleSequential advances three artifacts from a chain where
// each artifact is stale relative to its children.
func TestAdvance_MultipleSequential(t *testing.T) {
	// Chain: idea (draft) → req (clarifying) → be-plan (planning)
	// After first advance: idea should go to clarifying.
	// Then if we also advance req, it goes to planning.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-multi.md",
			content: makeArtifact("SC Adv Multi Idea", "idea", "draft", "sc-adv-multi", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-multi-2.md",
			content: makeArtifact("SC Adv Multi Req", "ticket", "clarifying", "sc-adv-multi",
				"lifecycle/ideas/sc-adv-multi.md", "Body."),
		},
		{
			relPath: "lifecycle/backend-plans/sc-adv-multi-3-be.md",
			content: makeArtifact("SC Adv Multi BE Plan", "plan-backend", "planning", "sc-adv-multi",
				"lifecycle/requirements/sc-adv-multi-2.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Advance idea (draft → clarifying) and req (clarifying → planning) in one call.
	paths := []string{
		"lifecycle/ideas/sc-adv-multi.md",
		"lifecycle/requirements/sc-adv-multi-2.md",
	}
	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": paths,
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Find results by path.
	byPath := make(map[string]*advanceResult, len(results))
	for i := range results {
		byPath[results[i].Path] = &results[i]
	}

	ideaRes := byPath["lifecycle/ideas/sc-adv-multi.md"]
	if ideaRes == nil {
		t.Fatal("missing result for idea")
	}
	if ideaRes.Outcome != "advanced" {
		t.Errorf("idea outcome: want advanced, got %q", ideaRes.Outcome)
	}

	reqRes := byPath["lifecycle/requirements/sc-adv-multi-2.md"]
	if reqRes == nil {
		t.Fatal("missing result for req")
	}
	if reqRes.Outcome != "advanced" {
		t.Errorf("req outcome: want advanced, got %q", reqRes.Outcome)
	}

	// Verify disk state.
	ideaRaw, _ := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/sc-adv-multi.md"))
	if !containsLine(string(ideaRaw), "status: clarifying") {
		t.Error("idea status not updated to 'clarifying' on disk")
	}
	reqRaw, _ := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/requirements/sc-adv-multi-2.md"))
	if !containsLine(string(reqRaw), "status: planning") {
		t.Error("req status not updated to 'planning' on disk")
	}
}

// TestAdvance_PermissionDenied verifies that when the authenticated user lacks
// the required role to make the suggested transition, the artifact is skipped
// with an appropriate error outcome.
func TestAdvance_PermissionDenied(t *testing.T) {
	// idea (draft) → req (clarifying)
	// Suggested advance: draft → clarifying requires product-owner or analyst.
	// Dev user cannot perform this transition.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-perm.md",
			content: makeArtifact("SC Adv Perm Idea", "idea", "draft", "sc-adv-perm", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-perm-2.md",
			content: makeArtifact("SC Adv Perm Req", "ticket", "clarifying", "sc-adv-perm",
				"lifecycle/ideas/sc-adv-perm.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("dev@test.local", "dev-pass-123")

	const ideaPath = "lifecycle/ideas/sc-adv-perm.md"
	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": []string{ideaPath},
	})
	// The overall response should still be 200; the error is per-artifact.
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Outcome == "advanced" {
		t.Error("dev user should not be able to advance draft → clarifying")
	}
	// Outcome should be "error" or "skipped" (not "advanced").
	if results[0].Outcome != "error" && results[0].Outcome != "skipped" {
		t.Errorf("expected outcome 'error' or 'skipped', got %q", results[0].Outcome)
	}
	if results[0].Reason == "" {
		t.Error("reason should be non-empty when permission is denied")
	}

	// Verify file was NOT modified.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	if containsLine(string(raw), "status: clarifying") {
		t.Error("file status should not have been changed on permission denial")
	}
	if !containsLine(string(raw), "status: draft") {
		t.Error("file status should still be 'draft' after failed advance")
	}
}

// TestAdvance_Idempotent verifies that advancing an artifact that is already
// at the correct (suggested) status returns no error and does not modify the
// file on disk.
func TestAdvance_Idempotent(t *testing.T) {
	// idea already at clarifying; child req at clarifying → no staleness.
	// But we explicitly POST the path to advance. The endpoint should detect
	// that no advance is needed and return outcome "skipped".
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-idem.md",
			content: makeArtifact("SC Adv Idem Idea", "idea", "clarifying", "sc-adv-idem", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-idem-2.md",
			content: makeArtifact("SC Adv Idem Req", "ticket", "clarifying", "sc-adv-idem",
				"lifecycle/ideas/sc-adv-idem.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const ideaPath = "lifecycle/ideas/sc-adv-idem.md"

	// Record the initial mtime.
	statBefore, err := os.Stat(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	mtimeBefore := statBefore.ModTime()

	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": []string{ideaPath},
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Outcome == "advanced" {
		t.Error("artifact already at correct status should not produce 'advanced' outcome")
	}

	// File mtime should not have changed (no disk write occurred).
	statAfter, err := os.Stat(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	if !statAfter.ModTime().Equal(mtimeBefore) {
		t.Error("file mtime changed; unexpected disk write on idempotent advance")
	}
}

// TestAdvance_WebSocketEvent verifies that advancing an artifact via POST
// /advance results in an artifact.indexed event being broadcast on the WebSocket hub.
func TestAdvance_WebSocketEvent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-ws.md",
			content: makeArtifact("SC Adv WS Idea", "idea", "draft", "sc-adv-ws", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-ws-2.md",
			content: makeArtifact("SC Adv WS Req", "ticket", "clarifying", "sc-adv-ws",
				"lifecycle/ideas/sc-adv-ws.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// Register a hub channel before triggering the advance so no events are missed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")

	const ideaPath = "lifecycle/ideas/sc-adv-ws.md"
	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": []string{ideaPath},
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)
	if len(results) == 0 || results[0].Outcome != "advanced" {
		t.Fatalf("advance did not succeed; outcome: %+v", results)
	}

	// Wait for the artifact.indexed event on the hub channel.
	timeout := time.After(5 * time.Second)
	var gotIndexed bool
COLLECT:
	for !gotIndexed {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for artifact.indexed WebSocket event")
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "artifact.indexed" {
				continue
			}
			if path, _ := evt.Payload["path"].(string); path == ideaPath {
				gotIndexed = true
				break COLLECT
			}
		}
	}
}

// TestAdvance_ReEvaluatesAtExecution verifies that the advance endpoint
// re-evaluates staleness at execution time rather than trusting the caller's
// implied suggestion. If another client already fixed the artifact before the
// advance request arrives, the endpoint should detect no work is needed and
// return outcome "skipped" rather than attempting a stale transition.
func TestAdvance_ReEvaluatesAtExecution(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-reeval.md",
			content: makeArtifact("SC Adv Reeval Idea", "idea", "draft", "sc-adv-reeval", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-reeval-2.md",
			content: makeArtifact("SC Adv Reeval Req", "ticket", "clarifying", "sc-adv-reeval",
				"lifecycle/ideas/sc-adv-reeval.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const ideaPath = "lifecycle/ideas/sc-adv-reeval.md"

	// Simulate "another client" by advancing via the normal transition endpoint first.
	transResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+ideaPath+"/transition", map[string]any{
		"to": "clarifying",
	})
	requireStatus(t, transResp, 200)
	transResp.Body.Close()

	// Now call advance — the artifact is already at the suggested status.
	// The endpoint must re-evaluate and return "skipped".
	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": []string{ideaPath},
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Outcome == "advanced" {
		t.Errorf("expected 'skipped' outcome (already advanced by another client), got %q", results[0].Outcome)
	}
}

// ── Milestone 1: Children field shape ────────────────────────────────────────

// TestStatusCheck_ChildrenFieldShape verifies that stale entries returned by
// GET /status-check include a `children` array of objects — each with non-empty
// `path` and `status` string fields — rather than bare strings or null.
func TestStatusCheck_ChildrenFieldShape(t *testing.T) {
	// Parent at in-development, child at done.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-children.md",
			content: makeArtifact("SC Children Idea", "idea", "in-development", "sc-children", "", "Idea body."),
		},
		{
			relPath: "lifecycle/requirements/sc-children-2.md",
			content: makeArtifact("SC Children Req", "ticket", "done", "sc-children",
				"lifecycle/ideas/sc-children.md", "Req body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=sc-children", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	entry := findStaleByPath(entries, "lifecycle/ideas/sc-children.md")
	if entry == nil {
		t.Fatal("expected lifecycle/ideas/sc-children.md to appear in stale list")
	}

	// children must be a non-nil, non-empty array of objects.
	if entry.Children == nil {
		t.Fatal("children field must not be nil")
	}
	if len(entry.Children) == 0 {
		t.Fatal("expected at least one child in children array")
	}

	// Each child must have non-empty path and status.
	for i, ch := range entry.Children {
		if ch.Path == "" {
			t.Errorf("children[%d].path is empty", i)
		}
		if ch.Status == "" {
			t.Errorf("children[%d].status is empty", i)
		}
	}

	// The requirement child should report status "done".
	var found bool
	for _, ch := range entry.Children {
		if ch.Path == "lifecycle/requirements/sc-children-2.md" {
			found = true
			if ch.Status != "done" {
				t.Errorf("children path=%q status: want %q, got %q",
					ch.Path, "done", ch.Status)
			}
		}
	}
	if !found {
		t.Error("requirement path not found in children array")
	}
}

// ── Milestone 2: Advance response contract ───────────────────────────────────

// TestAdvance_OkAndAdvancedToFields verifies that POST /status-check/advance
// returns `ok: true` and a non-empty `advanced_to` on a successful advance, and
// that the artifact's frontmatter is actually updated on disk.
func TestAdvance_OkAndAdvancedToFields(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-contract.md",
			content: makeArtifact("SC Adv Contract Idea", "idea", "in-development", "sc-adv-contract", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-contract-2.md",
			content: makeArtifact("SC Adv Contract Req", "ticket", "done", "sc-adv-contract",
				"lifecycle/ideas/sc-adv-contract.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const ideaPath = "lifecycle/ideas/sc-adv-contract.md"
	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": []string{ideaPath},
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Ok {
		t.Errorf("ok: want true for successful advance, got false (reason: %q)", results[0].Reason)
	}
	if results[0].AdvancedTo == "" {
		t.Error("advanced_to must be non-empty after a successful advance")
	}
	if results[0].AdvancedTo != "done" {
		t.Errorf("advanced_to: want %q, got %q", "done", results[0].AdvancedTo)
	}

	// Verify the status was actually written to disk.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: done") {
		t.Errorf("on-disk frontmatter should contain 'status: done' after advance; got:\n%s", raw)
	}
}

// TestAdvance_ResponseContractPermissionDenied verifies that when the
// authenticated user lacks the required role, the advance result has
// `ok: false` and a non-empty `reason`, and the artifact is NOT mutated.
func TestAdvance_ResponseContractPermissionDenied(t *testing.T) {
	// dev user (backend-developer role) cannot advance idea from draft → clarifying
	// (requires product-owner or analyst).
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-adv-deny.md",
			content: makeArtifact("SC Adv Deny Idea", "idea", "draft", "sc-adv-deny", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-adv-deny-2.md",
			content: makeArtifact("SC Adv Deny Req", "ticket", "clarifying", "sc-adv-deny",
				"lifecycle/ideas/sc-adv-deny.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("dev@test.local", "dev-pass-123")

	const ideaPath = "lifecycle/ideas/sc-adv-deny.md"
	resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": []string{ideaPath},
	})
	// Overall response is still 200; the denial is per-artifact.
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	results := decodeAdvanceResults(t, data)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Ok {
		t.Error("ok must be false when the user lacks the required role")
	}
	if results[0].Reason == "" {
		t.Error("reason must be non-empty when advance is denied")
	}

	// The file must NOT have been modified.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: draft") {
		t.Error("artifact status should still be 'draft' after a denied advance")
	}
	if containsLine(string(raw), "status: clarifying") {
		t.Error("artifact status must not have been changed to 'clarifying' on permission denial")
	}
}

// ── Milestone 3: Staleness detection edge cases ───────────────────────────────

// TestStatusCheck_MultiLevelLineageStaleness verifies that in a 3-level lineage
// (idea → requirement → plan) where the idea is at in-development and both
// downstream artifacts are at done, the idea is detected as stale (its direct
// child, the requirement, is at done which is strictly ahead).
func TestStatusCheck_MultiLevelLineageStaleness(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-multilevel.md",
			content: makeArtifact("SC MultiLevel Idea", "idea", "in-development", "sc-multilevel", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-multilevel-2.md",
			content: makeArtifact("SC MultiLevel Req", "ticket", "done", "sc-multilevel",
				"lifecycle/ideas/sc-multilevel.md", "Body."),
		},
		{
			relPath: "lifecycle/backend-plans/sc-multilevel-3-be.md",
			content: makeArtifact("SC MultiLevel Plan", "plan-backend", "done", "sc-multilevel",
				"lifecycle/requirements/sc-multilevel-2.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=sc-multilevel", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	// The idea should be stale: its direct child (req at done) is strictly ahead of in-development.
	ideaEntry := findStaleByPath(entries, "lifecycle/ideas/sc-multilevel.md")
	if ideaEntry == nil {
		t.Fatal("expected lifecycle/ideas/sc-multilevel.md to be stale in multi-level lineage")
	}
	if ideaEntry.CurrentStatus != "in-development" {
		t.Errorf("current_status: want %q, got %q", "in-development", ideaEntry.CurrentStatus)
	}
	if ideaEntry.SuggestedStatus != "done" {
		t.Errorf("suggested_status: want %q, got %q", "done", ideaEntry.SuggestedStatus)
	}

	// The requirement is at done; its direct child (plan) is also at done.
	// done == done → child is NOT strictly ahead → requirement should NOT be stale.
	reqEntry := findStaleByPath(entries, "lifecycle/requirements/sc-multilevel-2.md")
	if reqEntry != nil {
		t.Errorf("requirement at 'done' with child also at 'done' should NOT be stale; got: %+v", reqEntry)
	}
}

// TestStatusCheck_MixedProgressSiblings verifies that a parent is NOT reported
// as stale when it has two children at different statuses and at least one child
// has NOT advanced past the parent. ALL non-terminal children must be ahead for
// staleness to be reported.
func TestStatusCheck_MixedProgressSiblings(t *testing.T) {
	// idea (in-development) with two direct children:
	//   req-a at done    → ahead of parent
	//   req-b at in-development → same as parent (NOT ahead)
	// Not all active children are ahead → idea should NOT be stale.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-mixed.md",
			content: makeArtifact("SC Mixed Idea", "idea", "in-development", "sc-mixed", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-mixed-2.md",
			content: makeArtifact("SC Mixed Req A", "ticket", "done", "sc-mixed",
				"lifecycle/ideas/sc-mixed.md", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-mixed-3.md",
			content: makeArtifact("SC Mixed Req B", "ticket", "in-development", "sc-mixed",
				"lifecycle/ideas/sc-mixed.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=sc-mixed", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	// Idea must NOT appear in the stale list.
	ideaEntry := findStaleByPath(entries, "lifecycle/ideas/sc-mixed.md")
	if ideaEntry != nil {
		t.Errorf("idea with mixed-progress siblings should NOT be stale; got: %+v", ideaEntry)
	}
}

// TestStatusCheck_TerminalChildExcluded_Integration verifies that a terminal
// child (rejected) is excluded from the staleness comparison. When the only
// remaining non-terminal child has advanced past the parent, the parent IS stale.
func TestStatusCheck_TerminalChildExcluded_Integration(t *testing.T) {
	// idea (in-development) with two direct children:
	//   req-a at done     → non-terminal, ahead of parent
	//   req-b at rejected → terminal, excluded from comparison
	// After excluding req-b, all remaining active children (req-a) are ahead
	// → idea IS stale; suggested_status = done.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sc-term-child.md",
			content: makeArtifact("SC Term Child Idea", "idea", "in-development", "sc-term-child", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-term-child-2.md",
			content: makeArtifact("SC Term Child Req A", "ticket", "done", "sc-term-child",
				"lifecycle/ideas/sc-term-child.md", "Body."),
		},
		{
			relPath: "lifecycle/requirements/sc-term-child-3.md",
			content: makeArtifact("SC Term Child Req B", "ticket", "rejected", "sc-term-child",
				"lifecycle/ideas/sc-term-child.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=sc-term-child", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	// The idea should be stale: rejected child is excluded, non-terminal child (done) is ahead.
	ideaEntry := findStaleByPath(entries, "lifecycle/ideas/sc-term-child.md")
	if ideaEntry == nil {
		t.Fatal("expected idea to be stale when rejected sibling is excluded and remaining child is ahead")
	}
	if ideaEntry.SuggestedStatus != "done" {
		t.Errorf("suggested_status: want %q, got %q", "done", ideaEntry.SuggestedStatus)
	}

	// The rejected requirement should itself NOT appear in the stale list (terminal).
	for _, e := range entries {
		if e.Path == "lifecycle/requirements/sc-term-child-3.md" {
			t.Errorf("rejected artifact should not appear in stale list; got: %+v", e)
		}
	}
}
