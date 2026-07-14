// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact_test

import (
	"errors"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

func TestParseOpenQuestions_DashMarkers(t *testing.T) {
	body := "## Open Questions\n\n- Why is X?\n- What about Y?\n"
	qs, ok := artifact.ParseOpenQuestions(body)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(qs) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(qs))
	}
	if qs[0].Text != "Why is X?" || qs[0].Index != 0 {
		t.Errorf("q0 = %+v", qs[0])
	}
	if qs[1].Text != "What about Y?" || qs[1].Index != 1 {
		t.Errorf("q1 = %+v", qs[1])
	}
}

func TestParseOpenQuestions_NumberedMarkers(t *testing.T) {
	body := "## Open Questions\n\n1. First question?\n2. Second question?\n"
	qs, ok := artifact.ParseOpenQuestions(body)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(qs) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(qs))
	}
	if qs[0].Text != "First question?" {
		t.Errorf("q0.Text = %q", qs[0].Text)
	}
	if qs[1].Text != "Second question?" {
		t.Errorf("q1.Text = %q", qs[1].Text)
	}
}

func TestParseOpenQuestions_MultiLineQuestionWithSubItemsAndProse(t *testing.T) {
	body := "## Open Questions\n\n" +
		"- First line of question\n  continues here\n\n  more prose\n  - a sub-item, not a new question\n" +
		"- Second question?\n"
	qs, ok := artifact.ParseOpenQuestions(body)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(qs) != 2 {
		t.Fatalf("expected 2 questions, got %d: %+v", len(qs), qs)
	}
	if qs[1].Text != "Second question?" {
		t.Errorf("q1.Text = %q, want %q", qs[1].Text, "Second question?")
	}
	wantSubstr := "continues here"
	if !contains(qs[0].Text, wantSubstr) {
		t.Errorf("q0.Text = %q, want it to contain %q", qs[0].Text, wantSubstr)
	}
	if !contains(qs[0].Text, "a sub-item, not a new question") {
		t.Errorf("q0.Text = %q, want it to contain the sub-item text", qs[0].Text)
	}
}

func TestParseOpenQuestions_PreExistingAnswer(t *testing.T) {
	body := "## Open Questions\n\n" +
		"- **Q1**: Should we do X?\n\n> Yes, do X.\n\n" +
		"- **Q2**: Should we do Y?\n"
	qs, ok := artifact.ParseOpenQuestions(body)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(qs) != 2 {
		t.Fatalf("expected 2 questions, got %d: %+v", len(qs), qs)
	}
	if qs[0].Answer != "Yes, do X." {
		t.Errorf("q0.Answer = %q, want %q", qs[0].Answer, "Yes, do X.")
	}
	if qs[0].Text != "**Q1**: Should we do X?" {
		t.Errorf("q0.Text = %q", qs[0].Text)
	}
	if qs[1].Answer != "" {
		t.Errorf("q1.Answer = %q, want empty", qs[1].Answer)
	}
}

func TestParseOpenQuestions_MultiLineAnswer(t *testing.T) {
	body := "## Open Questions\n\n" +
		"- Q1?\n\n> Line one.\n> Line two.\n"
	qs, ok := artifact.ParseOpenQuestions(body)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(qs) != 1 {
		t.Fatalf("expected 1 question, got %d", len(qs))
	}
	want := "Line one.\nLine two."
	if qs[0].Answer != want {
		t.Errorf("Answer = %q, want %q", qs[0].Answer, want)
	}
}

func TestParseOpenQuestions_AbsentSection(t *testing.T) {
	qs, ok := artifact.ParseOpenQuestions("No heading here.\n")
	if ok || qs != nil {
		t.Errorf("expected (nil, false), got (%v, %v)", qs, ok)
	}
}

func TestParseOpenQuestions_EmptySection(t *testing.T) {
	qs, ok := artifact.ParseOpenQuestions("## Open Questions\n\n   \n\n## Next Section\ncontent\n")
	if ok || qs != nil {
		t.Errorf("expected (nil, false), got (%v, %v)", qs, ok)
	}
}

func TestParseOpenQuestions_MalformedSectionNoListItems(t *testing.T) {
	qs, ok := artifact.ParseOpenQuestions("## Open Questions\n\nJust a paragraph, no list items.\n")
	if ok || qs != nil {
		t.Errorf("expected (nil, false), got (%v, %v)", qs, ok)
	}
}

func TestParseOpenQuestions_HeadingRenamedToResolved(t *testing.T) {
	// After completion the heading becomes "## Resolved Questions"; the parser
	// should no longer find an "## Open Questions" section.
	qs, ok := artifact.ParseOpenQuestions("## Resolved Questions\n\n- Q1?\n\n> A1.\n")
	if ok || qs != nil {
		t.Errorf("expected (nil, false) for renamed heading, got (%v, %v)", qs, ok)
	}
}

func TestApplyAnswers_PartialInsertsAnswerKeepsHeading(t *testing.T) {
	body := "## Open Questions\n\n- Q1?\n\n- Q2?\n"
	out, err := artifact.ApplyAnswers(body, map[int]string{0: "A1."}, "blockquote", false)
	if err != nil {
		t.Fatalf("ApplyAnswers: %v", err)
	}
	if !contains(out, "## Open Questions") {
		t.Errorf("expected heading to remain 'Open Questions', got:\n%s", out)
	}
	if contains(out, "## Resolved Questions") {
		t.Errorf("heading must not be renamed on partial answers:\n%s", out)
	}
	qs, ok := artifact.ParseOpenQuestions(out)
	if !ok || len(qs) != 2 {
		t.Fatalf("expected 2 questions parseable from output, got ok=%v qs=%v", ok, qs)
	}
	if qs[0].Answer != "A1." {
		t.Errorf("q0.Answer = %q, want %q", qs[0].Answer, "A1.")
	}
	if qs[1].Answer != "" {
		t.Errorf("q1.Answer = %q, want empty", qs[1].Answer)
	}
}

func TestApplyAnswers_CompleteAllAnsweredRenamesHeading(t *testing.T) {
	body := "## Open Questions\n\n- Q1?\n\n- Q2?\n"
	out, err := artifact.ApplyAnswers(body, map[int]string{0: "A1.", 1: "A2."}, "blockquote", true)
	if err != nil {
		t.Fatalf("ApplyAnswers: %v", err)
	}
	if !contains(out, "## Resolved Questions") {
		t.Errorf("expected heading renamed to 'Resolved Questions', got:\n%s", out)
	}
	if contains(out, "## Open Questions") {
		t.Errorf("old heading must not remain:\n%s", out)
	}
}

func TestApplyAnswers_CompleteWithEmptyAnswerErrors(t *testing.T) {
	body := "## Open Questions\n\n- Q1?\n\n- Q2?\n"
	_, err := artifact.ApplyAnswers(body, map[int]string{0: "A1."}, "blockquote", true)
	if !errors.Is(err, artifact.ErrIncompleteAnswers) {
		t.Fatalf("expected ErrIncompleteAnswers, got %v", err)
	}
	if !contains(body, "## Open Questions") {
		t.Fatal("sanity: original body unchanged")
	}
}

func TestApplyAnswers_Idempotent(t *testing.T) {
	body := "## Open Questions\n\n- Q1?\n\n- Q2?\n"
	answers := map[int]string{0: "A1.", 1: "A2."}

	out1, err := artifact.ApplyAnswers(body, answers, "blockquote", false)
	if err != nil {
		t.Fatalf("first ApplyAnswers: %v", err)
	}
	out2, err := artifact.ApplyAnswers(out1, answers, "blockquote", false)
	if err != nil {
		t.Fatalf("second ApplyAnswers: %v", err)
	}
	if out1 != out2 {
		t.Errorf("ApplyAnswers is not idempotent:\n--- out1 ---\n%s\n--- out2 ---\n%s", out1, out2)
	}
}

func TestApplyAnswers_PreservesOtherSectionsAndFrontmatterVerbatim(t *testing.T) {
	full := "---\ntitle: T\ntype: idea\nstatus: blocked\nlineage: t\n---\n\n" +
		"## Summary\n\nSome unrelated content.\n\n" +
		"## Open Questions\n\n- Q1?\n\n" +
		"## Footer\n\nFooter content.\n"
	// Only the body (post-frontmatter) is passed to ApplyAnswers, matching how
	// the HTTP layer separates frontmatter from body.
	bodyStart := "## Summary\n\nSome unrelated content.\n\n" +
		"## Open Questions\n\n- Q1?\n\n" +
		"## Footer\n\nFooter content.\n"
	out, err := artifact.ApplyAnswers(bodyStart, map[int]string{0: "A1."}, "blockquote", false)
	if err != nil {
		t.Fatalf("ApplyAnswers: %v", err)
	}
	if !contains(out, "## Summary\n\nSome unrelated content.") {
		t.Errorf("Summary section not preserved verbatim:\n%s", out)
	}
	if !contains(out, "## Footer\n\nFooter content.") {
		t.Errorf("Footer section not preserved verbatim:\n%s", out)
	}
	_ = full
}

func TestApplyAnswers_NoOpenQuestionsSectionReturnsBodyUnchanged(t *testing.T) {
	body := "## Summary\n\nNo questions here.\n"
	out, err := artifact.ApplyAnswers(body, map[int]string{0: "A1."}, "blockquote", false)
	if err != nil {
		t.Fatalf("ApplyAnswers: %v", err)
	}
	if out != body {
		t.Errorf("expected unchanged body, got:\n%s", out)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
