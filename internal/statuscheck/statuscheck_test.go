// SPDX-License-Identifier: AGPL-3.0-or-later

package statuscheck

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
)

func row(path, lineage, status, parent string) *index.ArtifactRow {
	return &index.ArtifactRow{
		Path:    path,
		Lineage: lineage,
		Status:  status,
		FM:      artifact.Frontmatter{Lineage: lineage, Parent: parent},
	}
}

func TestCheck_SingleArtifact(t *testing.T) {
	results := Check([]*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
	})
	if len(results) != 0 {
		t.Fatalf("expected no results for single artifact, got %d", len(results))
	}
}

func TestCheck_NoChildren(t *testing.T) {
	// Parent with no children in the slice — not stale.
	results := Check([]*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "in-development", ""),
	})
	// Neither artifact has a parent pointing to the other in the slice.
	if len(results) != 0 {
		t.Fatalf("expected no results when no parent links, got %d", len(results))
	}
}

func TestCheck_NotStale_ChildBehind(t *testing.T) {
	// Child has same status as parent → not stale.
	results := Check([]*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "draft", "lifecycle/ideas/foo.md"),
	})
	if len(results) != 0 {
		t.Fatalf("expected no results when child is not ahead, got %d", len(results))
	}
}

func TestCheck_Stale_AllChildrenAhead(t *testing.T) {
	// Parent is "draft", child is "in-development" → parent is stale.
	artifacts := []*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "in-development", "lifecycle/ideas/foo.md"),
	}
	results := Check(artifacts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Path != "lifecycle/ideas/foo.md" {
		t.Errorf("unexpected stale path: %s", r.Path)
	}
	if r.CurrentStatus != "draft" {
		t.Errorf("unexpected current_status: %s", r.CurrentStatus)
	}
	if r.SuggestedStatus != "in-development" {
		t.Errorf("unexpected suggested_status: %s", r.SuggestedStatus)
	}
}

func TestCheck_SuggestedIsMinChildStatus(t *testing.T) {
	// Parent is "draft"; children are "in-qa" and "in-development".
	// Suggested should be the minimum = "in-development".
	artifacts := []*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "in-development", "lifecycle/ideas/foo.md"),
		row("lifecycle/backend-plans/foo-3-be.md", "foo", "in-qa", "lifecycle/ideas/foo.md"),
	}
	results := Check(artifacts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SuggestedStatus != "in-development" {
		t.Errorf("expected suggested_status=in-development, got %s", results[0].SuggestedStatus)
	}
}

func TestCheck_TerminalChildExcluded(t *testing.T) {
	// Parent is "draft"; one child is "in-development", one is "rejected" (terminal).
	// Only the non-terminal child counts; all non-terminal children are ahead → stale.
	artifacts := []*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "in-development", "lifecycle/ideas/foo.md"),
		row("lifecycle/requirements/foo-3.md", "foo", "rejected", "lifecycle/ideas/foo.md"),
	}
	results := Check(artifacts)
	if len(results) != 1 {
		t.Fatalf("expected 1 stale result, got %d", len(results))
	}
	if results[0].SuggestedStatus != "in-development" {
		t.Errorf("unexpected suggested_status: %s", results[0].SuggestedStatus)
	}
}

func TestCheck_AllChildrenTerminal(t *testing.T) {
	// Parent is "draft"; all children are terminal → not stale.
	artifacts := []*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "rejected", "lifecycle/ideas/foo.md"),
		row("lifecycle/requirements/foo-3.md", "foo", "abandoned", "lifecycle/ideas/foo.md"),
	}
	results := Check(artifacts)
	if len(results) != 0 {
		t.Fatalf("expected no results when all children are terminal, got %d", len(results))
	}
}

func TestCheck_ParentTerminal_NotStale(t *testing.T) {
	// Parent is "blocked" (terminal) and child is "in-development" → parent not stale.
	artifacts := []*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "blocked", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "in-development", "lifecycle/ideas/foo.md"),
	}
	results := Check(artifacts)
	if len(results) != 0 {
		t.Fatalf("expected no results for terminal parent, got %d", len(results))
	}
}

func TestCheck_MultipleStaleArtifacts(t *testing.T) {
	// Chain: grandparent → parent → child, all out of sync.
	// grandparent(draft) → parent(in-development) → child(approved)
	// grandparent is stale (child: in-development), parent is stale (child: approved).
	artifacts := []*index.ArtifactRow{
		row("lifecycle/ideas/foo.md", "foo", "draft", ""),
		row("lifecycle/requirements/foo-2.md", "foo", "in-development", "lifecycle/ideas/foo.md"),
		row("lifecycle/backend-plans/foo-3-be.md", "foo", "approved", "lifecycle/requirements/foo-2.md"),
	}
	results := Check(artifacts)
	if len(results) != 2 {
		t.Fatalf("expected 2 stale results, got %d", len(results))
	}
}

func TestCheck_EmptySlice(t *testing.T) {
	results := Check(nil)
	if len(results) != 0 {
		t.Fatalf("expected no results for nil slice, got %d", len(results))
	}
}
