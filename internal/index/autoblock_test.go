// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/hub"
)

// transitionerFunc is a function-backed Transitioner implementation used in
// unit tests to avoid importing the workflow package (which imports index,
// creating a circular dependency).
type transitionerFunc func(from, to string, roles []string, artifactType string) bool

func (f transitionerFunc) CanTransition(from, to string, roles []string, artifactType string) bool {
	return f(from, to, roles, artifactType)
}

// allowSystemBlockUnblock allows the same transitions as the real workflow
// engine for the "system" actor: any → blocked and blocked → draft.
var allowSystemBlockUnblock = transitionerFunc(func(from, to string, roles []string, _ string) bool {
	for _, r := range roles {
		if r != "system" {
			continue
		}
		if to == "blocked" {
			return true // system can block from any status
		}
		if from == "blocked" && to == "draft" {
			return true // system can unblock
		}
	}
	return false
})

// rejectAll is a Transitioner that rejects every transition. Used to test
// that applyOpenQuestionTransition handles workflow rejection gracefully.
var rejectAll = transitionerFunc(func(_, _ string, _ []string, _ string) bool { return false })

// openAutoBlockIndex creates a fresh temp-directory SQLite index with a hub
// and workflow transitioner wired up. No stages are registered, so Open's
// startup Scan finds nothing; tests write files manually and call IndexFile.
func openAutoBlockIndex(t *testing.T, wf Transitioner) (*Index, *hub.Hub, string) {
	t.Helper()
	dir := t.TempDir()
	h := hub.New()

	// IndexFile only indexes files inside lifecycle/.
	if err := os.MkdirAll(filepath.Join(dir, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}

	idx, err := Open(dir+"/test.db", dir, nil, WithHub(h), WithWorkflow(wf))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx, h, dir
}

// writeTestArtifact serialises a minimal artifact to lifecycle/ideas/<name>.md
// inside projRoot and returns the absolute path.
func writeTestArtifact(t *testing.T, projRoot, name, status, body string, assignees []artifact.Assignee) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("---\ntitle: Test\ntype: idea\nstatus: ")
	sb.WriteString(status)
	sb.WriteString("\nlineage: test-")
	sb.WriteString(name)
	sb.WriteString("\n")
	if len(assignees) > 0 {
		sb.WriteString("assignees:\n")
		for _, a := range assignees {
			sb.WriteString("    - role: " + a.Role + "\n")
			sb.WriteString("      who: " + a.Who + "\n")
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body)
	sb.WriteString("\n")
	absPath := filepath.Join(projRoot, "lifecycle", "ideas", name+".md")
	if err := os.WriteFile(absPath, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return absPath
}

// countEventsForPath counts events table rows whose artifact_path equals relPath.
func countEventsForPath(t *testing.T, idx *Index, relPath string) int {
	t.Helper()
	events, err := idx.ListEvents(200, 0, nil)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	n := 0
	for _, e := range events {
		if e.ArtifactPath != nil && *e.ArtifactPath == relPath {
			n++
		}
	}
	return n
}

// ── Milestone 2 — Unit tests for applyOpenQuestionTransition ─────────────────

// TestAutoBlock_DraftWithOQ verifies that indexing a draft artifact whose body
// contains a non-empty "## Open Questions" section transitions the status to
// "blocked" on disk, adds a product-owner assignee, and inserts a
// status_changed event into the SQLite index.
//
// Covers test plan Milestone 2, case 1.
func TestAutoBlock_DraftWithOQ(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)
	const name = "aq-draft-oq"
	absPath := writeTestArtifact(t, projRoot, name, "draft",
		"## Open Questions\n\n- Why?\n", nil)
	relPath := "lifecycle/ideas/" + name + ".md"

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	// Verify on-disk file was rewritten to "blocked".
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading disk file: %v", err)
	}
	disk := string(raw)
	if !strings.Contains(disk, "status: blocked") {
		t.Errorf("on-disk status should be 'blocked'; file:\n%s", disk)
	}
	if !strings.Contains(disk, "role: product-owner") {
		t.Errorf("on-disk file should have product-owner assignee; file:\n%s", disk)
	}

	// Verify SQLite index row.
	row, err := idx.Get(relPath)
	if err != nil || row == nil {
		t.Fatalf("Get: %v (row=%v)", err, row)
	}
	if row.Status != "blocked" {
		t.Errorf("index status: got %q, want %q", row.Status, "blocked")
	}
	found := false
	for _, a := range row.FM.Assignees {
		if a.Role == "product-owner" && a.Who == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("index should have {role:product-owner,who:agent}; got: %v", row.FM.Assignees)
	}

	// Verify event was inserted.
	if n := countEventsForPath(t, idx, relPath); n == 0 {
		t.Error("expected at least 1 status_changed event, got 0")
	}
}

// TestAutoBlock_BlockedWithNoOQ verifies that indexing a "blocked" artifact
// with no open questions transitions the status to "draft" on disk and inserts
// a status_changed event.
//
// Covers test plan Milestone 2, case 2.
func TestAutoBlock_BlockedWithNoOQ(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)
	const name = "aq-blocked-no-oq"
	absPath := writeTestArtifact(t, projRoot, name, "blocked",
		"No open questions here.\n", nil)
	relPath := "lifecycle/ideas/" + name + ".md"

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading disk file: %v", err)
	}
	if !strings.Contains(string(raw), "status: draft") {
		t.Errorf("on-disk status should be 'draft'; file:\n%s", string(raw))
	}

	row, err := idx.Get(relPath)
	if err != nil || row == nil {
		t.Fatalf("Get: %v", err)
	}
	if row.Status != "draft" {
		t.Errorf("index status: got %q, want %q", row.Status, "draft")
	}

	if n := countEventsForPath(t, idx, relPath); n == 0 {
		t.Error("expected at least 1 status_changed event, got 0")
	}
}

// TestAutoBlock_IdempotentBlockedWithOQ verifies that indexing an already
// "blocked" artifact WITH open questions is a no-op: no disk rewrite and no
// new event.
//
// Covers test plan Milestone 2, case 3.
func TestAutoBlock_IdempotentBlockedWithOQ(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)
	const name = "aq-idem-blocked-oq"
	absPath := writeTestArtifact(t, projRoot, name, "blocked",
		"## Open Questions\n\n- Still pending.\n", nil)
	relPath := "lifecycle/ideas/" + name + ".md"

	// First call: stores SHA, no auto-transition needed (already blocked + OQ).
	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("first IndexFile: %v", err)
	}
	initialEvents := countEventsForPath(t, idx, relPath)

	// Record mtime; any disk write would produce a later mtime.
	info, _ := os.Stat(absPath)
	mtimeBefore := info.ModTime()
	time.Sleep(15 * time.Millisecond)

	// Second call: SHA guard should match → early return, no auto-transition.
	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("second IndexFile: %v", err)
	}

	if n := countEventsForPath(t, idx, relPath); n != initialEvents {
		t.Errorf("event count changed: was %d, now %d (unexpected event on idempotent call)", initialEvents, n)
	}
	info, _ = os.Stat(absPath)
	if !info.ModTime().Equal(mtimeBefore) {
		t.Error("file mtime changed: unexpected disk write on idempotent IndexFile")
	}
}

// TestAutoBlock_IdempotentDraftNoOQ verifies that indexing a draft artifact
// with no open questions is a no-op: no disk rewrite and no event.
//
// Covers test plan Milestone 2, case 4.
func TestAutoBlock_IdempotentDraftNoOQ(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)
	const name = "aq-idem-draft-no-oq"
	absPath := writeTestArtifact(t, projRoot, name, "draft",
		"Just a normal body, no questions.\n", nil)
	relPath := "lifecycle/ideas/" + name + ".md"

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("first IndexFile: %v", err)
	}
	initialEvents := countEventsForPath(t, idx, relPath)

	info, _ := os.Stat(absPath)
	mtimeBefore := info.ModTime()
	time.Sleep(15 * time.Millisecond)

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("second IndexFile: %v", err)
	}

	if n := countEventsForPath(t, idx, relPath); n != initialEvents {
		t.Errorf("event count changed: was %d, now %d (unexpected event)", initialEvents, n)
	}
	info, _ = os.Stat(absPath)
	if !info.ModTime().Equal(mtimeBefore) {
		t.Error("file mtime changed: unexpected disk write on idempotent IndexFile")
	}
}

// TestAutoBlock_WorkflowRejection verifies that when the Transitioner rejects
// the blocked transition, no disk write occurs and no event is inserted.
//
// Covers test plan Milestone 2, case 5.
func TestAutoBlock_WorkflowRejection(t *testing.T) {
	// rejectAll simulates a workflow engine that disallows all transitions.
	idx, _, projRoot := openAutoBlockIndex(t, rejectAll)
	const name = "aq-wf-reject"
	absPath := writeTestArtifact(t, projRoot, name, "draft",
		"## Open Questions\n\n- Blocked by workflow.\n", nil)
	relPath := "lifecycle/ideas/" + name + ".md"

	info, _ := os.Stat(absPath)
	mtimeBefore := info.ModTime()

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	info, _ = os.Stat(absPath)
	if !info.ModTime().Equal(mtimeBefore) {
		t.Error("file mtime changed: disk write occurred despite workflow rejection")
	}
	if n := countEventsForPath(t, idx, relPath); n != 0 {
		t.Errorf("expected 0 events when workflow rejects transition, got %d", n)
	}
}

// TestAutoBlock_AssigneeDeduplication verifies that auto-block does not add a
// second product-owner/agent assignee when one already exists in the artifact.
//
// Covers test plan Milestone 2, case 6.
func TestAutoBlock_AssigneeDeduplication(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)
	existingAssignees := []artifact.Assignee{
		{Role: "product-owner", Who: "agent"},
	}
	const name = "aq-dup-assignee"
	absPath := writeTestArtifact(t, projRoot, name, "draft",
		"## Open Questions\n\n- Should we do X?\n", existingAssignees)
	relPath := "lifecycle/ideas/" + name + ".md"

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	row, err := idx.Get(relPath)
	if err != nil || row == nil {
		t.Fatalf("Get: %v", err)
	}
	poCount := 0
	for _, a := range row.FM.Assignees {
		if a.Role == "product-owner" && a.Who == "agent" {
			poCount++
		}
	}
	if poCount != 1 {
		t.Errorf("expected exactly 1 product-owner/agent assignee, got %d (list: %v)", poCount, row.FM.Assignees)
	}
}

// TestAutoBlock_AtomicWrite verifies that after auto-block completes normally,
// no residual ".tmp" files remain in the lifecycle directory.
//
// Covers test plan Milestone 2, case 7.
func TestAutoBlock_AtomicWrite(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)
	absPath := writeTestArtifact(t, projRoot, "aq-atomic", "draft",
		"## Open Questions\n\n- How?\n", nil)

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	var tmpFiles []string
	err := filepath.WalkDir(filepath.Join(projRoot, "lifecycle"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".tmp") {
			tmpFiles = append(tmpFiles, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	if len(tmpFiles) > 0 {
		t.Errorf("residual .tmp files found after auto-block: %v", tmpFiles)
	}
}

// ── Milestone 3 — Circular-trigger prevention (SHA-256 guard) ────────────────

// TestAutoBlock_NoCircularReindex verifies that after applyOpenQuestionTransition
// rewrites a file, a subsequent IndexFile call on the same path is short-
// circuited by the SHA-256 guard: no second event is inserted and no second
// disk write occurs.
//
// Covers test plan Milestone 3.
func TestAutoBlock_NoCircularReindex(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)
	const name = "aq-no-circular"
	absPath := writeTestArtifact(t, projRoot, name, "draft",
		"## Open Questions\n\n- Circular?\n", nil)
	relPath := "lifecycle/ideas/" + name + ".md"

	// First call: auto-block fires, rewrites file, inserts 1 event.
	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("first IndexFile: %v", err)
	}
	eventCount1 := countEventsForPath(t, idx, relPath)
	if eventCount1 == 0 {
		t.Fatal("expected at least 1 event after first IndexFile (auto-block)")
	}

	// Record file stat immediately after auto-block.
	info, _ := os.Stat(absPath)
	mtimeAfterBlock := info.ModTime()

	// Brief pause so any subsequent disk write would produce a later mtime.
	time.Sleep(15 * time.Millisecond)

	// Second call: file content is now blocked+OQ; stored SHA matches → SHA
	// guard fires → early return, no Upsert, no applyOpenQuestionTransition.
	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("second IndexFile: %v", err)
	}

	eventCount2 := countEventsForPath(t, idx, relPath)
	if eventCount2 != eventCount1 {
		t.Errorf("second IndexFile inserted new events (was %d, now %d): circular reindex guard failed",
			eventCount1, eventCount2)
	}

	info, _ = os.Stat(absPath)
	if !info.ModTime().Equal(mtimeAfterBlock) {
		t.Error("file mtime changed after second IndexFile: disk was written again (circular reindex detected)")
	}
}

// ── Milestone 4 — Auto-block from 'raw' on open questions ────────────────────

// TestAutoBlock_RawWithOpenQuestions verifies that a raw artifact containing a
// non-empty "## Open Questions" section is automatically transitioned to
// "blocked" (not kept as "raw"), and that removing the section causes the
// auto-unblock reactor to transition it to "draft" (not back to "raw").
//
// This test covers the Milestone 4 acceptance criterion: the universal
// any→blocked rule (system actor, empty from-matcher) applies to "raw", and
// the system-initiated workflow check passes without a log warning.
func TestAutoBlock_RawWithOpenQuestions(t *testing.T) {
	idx, _, projRoot := openAutoBlockIndex(t, allowSystemBlockUnblock)

	const name = "aq-raw-oq"
	relPath := "lifecycle/ideas/" + name + ".md"

	// Step 1: write a raw artifact with an Open Questions section.
	absPath := writeTestArtifact(t, projRoot, name, "raw",
		"## Open Questions\n\n- What is the scope of this idea?\n", nil)

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile (step 1): %v", err)
	}

	// Assert: on-disk status is now "blocked".
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading disk file after block: %v", err)
	}
	diskStr := string(raw)
	if !strings.Contains(diskStr, "status: blocked") {
		t.Errorf("step 1: on-disk status should be 'blocked'; file:\n%s", diskStr)
	}
	if !strings.Contains(diskStr, "role: product-owner") {
		t.Errorf("step 1: on-disk file should have product-owner assignee; file:\n%s", diskStr)
	}

	// Assert: SQLite index row reflects blocked.
	row, err := idx.Get(relPath)
	if err != nil || row == nil {
		t.Fatalf("Get after step 1: %v (row=%v)", err, row)
	}
	if row.Status != "blocked" {
		t.Errorf("step 1: index status: got %q, want %q", row.Status, "blocked")
	}

	// Assert: a status_changed event was inserted.
	if n := countEventsForPath(t, idx, relPath); n == 0 {
		t.Error("step 1: expected at least 1 status_changed event after auto-block, got 0")
	}

	// Step 2: remove the Open Questions section and re-index.
	// Read the current blocked file and overwrite without the OQ section.
	rawAfterBlock, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading blocked file for step 2: %v", err)
	}
	// Replace Open Questions section with plain body text.
	noOQ := strings.ReplaceAll(string(rawAfterBlock),
		"## Open Questions\n\n- What is the scope of this idea?\n",
		"No more open questions.\n")
	if err := os.WriteFile(absPath, []byte(noOQ), 0o644); err != nil {
		t.Fatalf("writing step-2 file: %v", err)
	}

	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile (step 2): %v", err)
	}

	// Assert: auto-unblock transitions to "draft" (not back to "raw").
	raw2, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading disk file after unblock: %v", err)
	}
	diskStr2 := string(raw2)
	if !strings.Contains(diskStr2, "status: draft") {
		t.Errorf("step 2: on-disk status should be 'draft' (not 'raw'); file:\n%s", diskStr2)
	}
	if strings.Contains(diskStr2, "status: raw") {
		t.Errorf("step 2: on-disk status must not revert to 'raw'; file:\n%s", diskStr2)
	}

	row2, err := idx.Get(relPath)
	if err != nil || row2 == nil {
		t.Fatalf("Get after step 2: %v (row=%v)", err, row2)
	}
	if row2.Status != "draft" {
		t.Errorf("step 2: index status: got %q, want %q", row2.Status, "draft")
	}
}
