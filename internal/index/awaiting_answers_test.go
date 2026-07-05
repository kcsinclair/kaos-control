// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"testing"
)

// ── Milestone 4 — has_open_questions column + awaiting_answers filter ───────

// TestUpsert_HasOpenQuestionsColumnPopulated verifies that Upsert derives
// has_open_questions from the artifact body via artifact.HasOpenQuestions,
// and that the awaiting_answers filter uses it (blocked AND has_open_questions=1).
func TestUpsert_HasOpenQuestionsColumnPopulated(t *testing.T) {
	idx, _, dir := openAutoBlockIndex(t, rejectAll)

	blockedWithQuestions := writeTestArtifact(t, dir, "blocked-with-q", "blocked",
		"## Open Questions\n\n- Why?\n", nil)
	blockedNoQuestions := writeTestArtifact(t, dir, "blocked-no-q", "blocked",
		"No questions here.", nil)
	draftWithQuestions := writeTestArtifact(t, dir, "draft-with-q", "draft",
		"## Open Questions\n\n- Why?\n", nil)

	for _, p := range []string{blockedWithQuestions, blockedNoQuestions, draftWithQuestions} {
		if err := idx.IndexFile(p); err != nil {
			t.Fatalf("IndexFile(%s): %v", p, err)
		}
	}

	count, err := idx.Count(Filter{AwaitingAnswers: true})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 artifact awaiting answers, got %d", count)
	}

	items, total, err := idx.List(Filter{AwaitingAnswers: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 item, got total=%d len(items)=%d", total, len(items))
	}
	if items[0].Status != "blocked" {
		t.Errorf("expected the blocked-with-questions artifact, got status=%q", items[0].Status)
	}
}

// TestBuildWhere_AwaitingAnswers verifies the SQL fragment produced for
// AwaitingAnswers is a status+has_open_questions conjunction.
func TestBuildWhere_AwaitingAnswers(t *testing.T) {
	clause, args := buildWhere(Filter{AwaitingAnswers: true})
	if clause != " WHERE status = 'blocked' AND has_open_questions = 1" {
		t.Errorf("unexpected clause: %q", clause)
	}
	if len(args) != 0 {
		t.Errorf("expected no bound args, got %v", args)
	}
}
