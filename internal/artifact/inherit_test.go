// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact_test

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// TestApplyInheritedFields covers the unit behaviour of the shared helper
// (FR-2, FR-3, FR-4, and the "no fabricated default" rule).
func TestApplyInheritedFields(t *testing.T) {
	t.Run("empty child priority inherits from parent", func(t *testing.T) {
		child := artifact.Frontmatter{}
		parent := artifact.Frontmatter{Priority: "high"}
		artifact.ApplyInheritedFields(&child, parent)
		if child.Priority != "high" {
			t.Errorf("Priority: want %q, got %q", "high", child.Priority)
		}
	})

	t.Run("empty child release inherits from parent", func(t *testing.T) {
		child := artifact.Frontmatter{}
		parent := artifact.Frontmatter{Release: "KC-Release4"}
		artifact.ApplyInheritedFields(&child, parent)
		if child.Release != "KC-Release4" {
			t.Errorf("Release: want %q, got %q", "KC-Release4", child.Release)
		}
	})

	t.Run("non-empty child priority wins over parent (FR-4)", func(t *testing.T) {
		child := artifact.Frontmatter{Priority: "low"}
		parent := artifact.Frontmatter{Priority: "high"}
		artifact.ApplyInheritedFields(&child, parent)
		if child.Priority != "low" {
			t.Errorf("Priority: want %q (child wins), got %q", "low", child.Priority)
		}
	})

	t.Run("non-empty child release wins over parent (FR-4)", func(t *testing.T) {
		child := artifact.Frontmatter{Release: "child-release"}
		parent := artifact.Frontmatter{Release: "parent-release"}
		artifact.ApplyInheritedFields(&child, parent)
		if child.Release != "child-release" {
			t.Errorf("Release: want %q (child wins), got %q", "child-release", child.Release)
		}
	})

	t.Run("empty child and empty parent leave fields empty (no fabricated default)", func(t *testing.T) {
		child := artifact.Frontmatter{}
		parent := artifact.Frontmatter{}
		artifact.ApplyInheritedFields(&child, parent)
		if child.Priority != "" {
			t.Errorf("Priority: want empty (no default injected), got %q", child.Priority)
		}
		if child.Release != "" {
			t.Errorf("Release: want empty (no default injected), got %q", child.Release)
		}
	})

	t.Run("title, type, lineage, labels, assignees are not inherited", func(t *testing.T) {
		child := artifact.Frontmatter{
			Title:   "original title",
			Type:    "idea",
			Lineage: "my-lineage",
			Labels:  []string{"a", "b"},
			Assignees: []artifact.Assignee{
				{Role: "analyst", Who: "agent"},
			},
		}
		parent := artifact.Frontmatter{
			Priority:  "high",
			Release:   "v1",
			Title:     "parent title",
			Labels:    []string{"x"},
			Assignees: []artifact.Assignee{{Role: "qa", Who: "agent"}},
		}
		artifact.ApplyInheritedFields(&child, parent)

		if child.Title != "original title" {
			t.Errorf("Title must not be inherited; want %q, got %q", "original title", child.Title)
		}
		if child.Type != "idea" {
			t.Errorf("Type must not be inherited; want %q, got %q", "idea", child.Type)
		}
		if child.Lineage != "my-lineage" {
			t.Errorf("Lineage must not be inherited; want %q, got %q", "my-lineage", child.Lineage)
		}
		if len(child.Labels) != 2 || child.Labels[0] != "a" || child.Labels[1] != "b" {
			t.Errorf("Labels must not be inherited; want [a b], got %v", child.Labels)
		}
		if len(child.Assignees) != 1 || child.Assignees[0].Role != "analyst" {
			t.Errorf("Assignees must not be inherited; want [{analyst agent}], got %v", child.Assignees)
		}
		// Priority and release ARE inherited when child fields are empty.
		if child.Priority != "high" {
			t.Errorf("Priority: want %q, got %q", "high", child.Priority)
		}
		if child.Release != "v1" {
			t.Errorf("Release: want %q, got %q", "v1", child.Release)
		}
	})
}
