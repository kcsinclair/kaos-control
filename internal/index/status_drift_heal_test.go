// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"testing"
)

// TestIndexFile_HealsStatusDriftDespiteMatchingHash reproduces the "approve
// won't stick" defect: an index row whose stored body_sha256 matches the file
// on disk but whose status has drifted from the file's status. The content-hash
// skip-guard must NOT lock the stale status in — the next IndexFile of the
// (byte-unchanged) file must reconcile the status.
func TestIndexFile_HealsStatusDriftDespiteMatchingHash(t *testing.T) {
	idx, _, root := openAutoBlockIndex(t, allowSystemBlockUnblock)
	rel := "lifecycle/ideas/drift.md"
	abs := writeTestArtifact(t, root, "drift", "approved", "Just a body.", nil)

	// Correctly index the approved file → row (status=approved, hash=approvedH).
	if err := idx.IndexFile(abs); err != nil {
		t.Fatalf("initial IndexFile: %v", err)
	}

	// Simulate the desync: force the status column to draft while the stored
	// hash still matches the (unchanged) approved file on disk.
	if _, err := idx.db.Exec(`UPDATE artifacts SET status = 'draft' WHERE path = ?`, rel); err != nil {
		t.Fatalf("seed drift: %v", err)
	}
	if row, _ := idx.Get(rel); row == nil || row.Status != "draft" {
		t.Fatalf("precondition: expected drifted status=draft")
	}

	// Re-index the byte-unchanged approved file. The old hash-only guard would
	// skip (hash matches) and leave status=draft; the fix reconciles to approved.
	if err := idx.IndexFile(abs); err != nil {
		t.Fatalf("re-index: %v", err)
	}

	row, err := idx.Get(rel)
	if err != nil || row == nil {
		t.Fatalf("Get after re-index: %v", err)
	}
	if row.Status != "approved" {
		t.Errorf("status stuck at %q; expected self-heal to approved (matching the file)", row.Status)
	}
}
