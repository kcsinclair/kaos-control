// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// ----- buildWhere unit tests -----

// TestFilterQ_Only verifies that a Filter with only Q set produces a WHERE
// clause containing the five-column LIKE OR group with the correct pattern.
func TestFilterQ_Only(t *testing.T) {
	clause, args := buildWhere(Filter{Q: "hello"})

	if !strings.Contains(clause, "WHERE") {
		t.Fatalf("expected WHERE clause, got: %q", clause)
	}
	for _, col := range []string{"title", "slug", "lineage", "type", "status"} {
		if !strings.Contains(clause, col+" LIKE ?") {
			t.Errorf("expected LIKE condition on %q, clause: %q", col, clause)
		}
	}
	// Five args for five LIKE placeholders.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}
	for i, a := range args {
		if a != "%hello%" {
			t.Errorf("args[%d] = %q, want %q", i, a, "%hello%")
		}
	}
}

// TestFilterQ_WithStatus verifies that Q and Status are combined with AND.
func TestFilterQ_WithStatus(t *testing.T) {
	clause, args := buildWhere(Filter{Q: "hello", Status: "draft"})

	if !strings.Contains(clause, "status = ?") {
		t.Errorf("expected status = ? condition, clause: %q", clause)
	}
	// Should have AND joining the two conditions.
	if !strings.Contains(clause, " AND ") {
		t.Errorf("expected AND in clause, got: %q", clause)
	}
	// 1 arg for status + 5 args for Q.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(args))
	}
}

// TestFilterQ_SpecialChars verifies that % and _ in the query are escaped.
func TestFilterQ_SpecialChars(t *testing.T) {
	_, args := buildWhere(Filter{Q: "100%"})
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}
	for i, a := range args {
		if a != `%100\%%` {
			t.Errorf("args[%d] = %q, want %q", i, a, `%100\%%`)
		}
	}

	_, args2 := buildWhere(Filter{Q: "some_thing"})
	for i, a := range args2 {
		if a != `%some\_thing%` {
			t.Errorf("args2[%d] = %q, want %q", i, a, `%some\_thing%`)
		}
	}
}

// TestFilterQ_EmptyProducesNoCondition verifies that an empty Q adds nothing.
func TestFilterQ_EmptyProducesNoCondition(t *testing.T) {
	clause, args := buildWhere(Filter{Q: ""})
	if clause != "" {
		t.Errorf("expected empty clause, got: %q", clause)
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

// ----- integration test using a real SQLite index -----

// makeFilterTestArtifact builds a minimal Artifact for filter tests.
func makeFilterTestArtifact(path, slug, title, status string) *artifact.Artifact {
	return &artifact.Artifact{
		Path:  path,
		Slug:  slug,
		Stage: "ideas",
		Index: 0,
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   title,
			Type:    "idea",
			Status:  status,
			Lineage: slug,
		},
	}
}

// TestFilterQ_CaseInsensitive inserts two artifacts with "Kanban" in their
// titles (mixed case) and verifies that a lowercase Q matches both.
func TestFilterQ_CaseInsensitive(t *testing.T) {
	idx := openTestIndex(t)

	a1 := makeFilterTestArtifact("lifecycle/ideas/kanban-view.md", "kanban-view", "Kanban View", "draft")
	a2 := makeFilterTestArtifact("lifecycle/ideas/kanban-board.md", "kanban-board", "kanban-board", "draft")
	a3 := makeFilterTestArtifact("lifecycle/ideas/other.md", "other", "Something Else", "draft")

	for _, a := range []*artifact.Artifact{a1, a2, a3} {
		if err := idx.Upsert(a); err != nil {
			t.Fatalf("Upsert(%s): %v", a.Path, err)
		}
	}

	rows, total, err := idx.List(Filter{Q: "kanban"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(rows) != 2 {
		t.Errorf("len(rows) = %d, want 2", len(rows))
	}
	for _, r := range rows {
		if !strings.Contains(strings.ToLower(r.Title), "kanban") &&
			!strings.Contains(strings.ToLower(r.Slug), "kanban") {
			t.Errorf("unexpected row: title=%q slug=%q", r.Title, r.Slug)
		}
	}
}
