// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// TestNextIndexForLineage_CrossFolder verifies that lineage index allocation
// is scoped to the lineage alone, never to the folder a member happens to
// live in — Milestone 5 of idea-archiving-3-be.
func TestNextIndexForLineage_CrossFolder(t *testing.T) {
	idxFlat := openTestIndex(t)
	idxNested := openTestIndex(t)

	// Same lineage, same indices (0 and 2), but one index has every member
	// flat and the other has them scattered across subfolders.
	flat0 := makeTestArtifact("lifecycle/ideas/login.md", "login", "")
	flat2 := makeTestArtifact("lifecycle/ideas/login-2.md", "login", "")
	flat2.Index = 2
	nested0 := makeTestArtifact("lifecycle/ideas/a/login.md", "login", "")
	nested2 := makeTestArtifact("lifecycle/ideas/b/login-2.md", "login", "")
	nested2.Index = 2

	for _, a := range []*artifact.Artifact{flat0, flat2} {
		if err := idxFlat.Upsert(a); err != nil {
			t.Fatalf("Upsert flat %s: %v", a.Path, err)
		}
	}
	for _, a := range []*artifact.Artifact{nested0, nested2} {
		if err := idxNested.Upsert(a); err != nil {
			t.Fatalf("Upsert nested %s: %v", a.Path, err)
		}
	}

	flatNext, err := idxFlat.NextIndexForLineage("login")
	if err != nil {
		t.Fatalf("NextIndexForLineage (flat): %v", err)
	}
	nestedNext, err := idxNested.NextIndexForLineage("login")
	if err != nil {
		t.Fatalf("NextIndexForLineage (nested): %v", err)
	}
	if flatNext != nestedNext {
		t.Errorf("next index diverged by folder placement: flat=%d nested=%d", flatNext, nestedNext)
	}
	if nestedNext != 3 {
		t.Errorf("want next index 3 regardless of folder, got %d", nestedNext)
	}
}

// TestLineageIndexCollision_AcrossFolders verifies that two artifacts sharing
// a lineage and index but living in different subfolders both persist in the
// index — exactly as two same-lineage-and-index flat files would — so a
// cross-folder collision is surfaced by the same (path-keyed) mechanism as a
// flat one, with no folder-specific special-casing.
func TestLineageIndexCollision_AcrossFolders(t *testing.T) {
	idx := openTestIndex(t)

	a := makeTestArtifact("lifecycle/ideas/a/dup-3.md", "dup", "")
	a.Index = 3
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert a: %v", err)
	}

	b := makeTestArtifact("lifecycle/ideas/b/dup-3-x.md", "dup", "")
	b.Index = 3
	if err := idx.Upsert(b); err != nil {
		t.Fatalf("Upsert b: %v", err)
	}

	rows, err := idx.ListByLineage("dup")
	if err != nil {
		t.Fatalf("ListByLineage: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want both colliding members present (surfaced like a flat collision), got %d rows", len(rows))
	}
}
