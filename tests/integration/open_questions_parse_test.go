// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"strings"
	"testing"
)

// TestOpenQuestionsParse_PrepopulatedAnswersPreserved verifies that
// GET .../open-questions parses each top-level list item under
// "## Open Questions" as a question, and preserves any answer already
// written as a trailing blockquote.
func TestOpenQuestionsParse_PrepopulatedAnswersPreserved(t *testing.T) {
	const relPath = "lifecycle/ideas/parse-prepopulated.md"
	body := "Intro paragraph.\n\n" +
		"## Open Questions\n\n" +
		"- Why is the sky blue?\n\n" +
		"> Rayleigh scattering.\n\n" +
		"- What is the airspeed velocity of an unladen swallow?\n"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeArtifact("Parse Prepopulated", "idea", "draft", "parse-prepopulated", "", body)},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath+"/open-questions", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if heading, _ := data["heading"].(string); heading != "## Open Questions" {
		t.Errorf("heading = %q, want '## Open Questions'", heading)
	}
	if format, _ := data["format"].(string); format != "blockquote" {
		t.Errorf("format = %q, want blockquote", format)
	}

	questions, _ := data["questions"].([]any)
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d: %v", len(questions), questions)
	}

	q0, _ := questions[0].(map[string]any)
	if text, _ := q0["text"].(string); text != "Why is the sky blue?" {
		t.Errorf("questions[0].text = %q", text)
	}
	if answer, _ := q0["answer"].(string); answer != "Rayleigh scattering." {
		t.Errorf("questions[0].answer = %q, want %q", answer, "Rayleigh scattering.")
	}

	q1, _ := questions[1].(map[string]any)
	if text, _ := q1["text"].(string); text != "What is the airspeed velocity of an unladen swallow?" {
		t.Errorf("questions[1].text = %q", text)
	}
	if answer, _ := q1["answer"].(string); answer != "" {
		t.Errorf("questions[1].answer = %q, want empty (no answer written yet)", answer)
	}
}

// TestOpenQuestionsParse_MalformedOrEmptySectionReturnsEmptyList verifies
// that a "## Open Questions" heading with no top-level list items (malformed
// or empty), and an artifact with no such heading at all, both return an
// empty (non-null) questions array rather than an error (NFR6).
func TestOpenQuestionsParse_MalformedOrEmptySectionReturnsEmptyList(t *testing.T) {
	const malformedPath = "lifecycle/ideas/parse-malformed.md"
	const absentPath = "lifecycle/ideas/parse-absent.md"
	seeds := []seedArtifact{
		{relPath: malformedPath, content: makeArtifact("Parse Malformed", "idea", "draft", "parse-malformed", "",
			"## Open Questions\n\nJust some prose, no list items here.\n")},
		{relPath: absentPath, content: makeArtifact("Parse Absent", "idea", "draft", "parse-absent", "",
			"No open questions section in this body at all.")},
	}
	env := newTestEnv(t, seeds)

	for _, relPath := range []string{malformedPath, absentPath} {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath+"/open-questions", nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)

		questions, ok := data["questions"].([]any)
		if !ok {
			t.Fatalf("%s: expected 'questions' to be an array, got %v", relPath, data["questions"])
		}
		if len(questions) != 0 {
			t.Errorf("%s: expected 0 questions, got %d: %v", relPath, len(questions), questions)
		}
	}
}

// TestOpenQuestionsParse_PreviewPartialIsIdempotentAndPreservesRest verifies
// that the preview endpoint with complete=false inserts the given answer as
// a blockquote, leaves unrelated sections of the document byte-for-byte
// unchanged, and produces byte-identical output when applied twice with the
// same inputs.
func TestOpenQuestionsParse_PreviewPartialIsIdempotentAndPreservesRest(t *testing.T) {
	const relPath = "lifecycle/ideas/parse-preview-partial.md"
	body := "# Heading\n\nSome unrelated prose that must survive untouched.\n\n" +
		"## Open Questions\n\n- Q1?\n\n- Q2?\n\n" +
		"## Another Section\n\nMore unrelated prose.\n"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeArtifact("Parse Preview Partial", "idea", "draft", "parse-preview-partial", "", body)},
	}
	env := newTestEnv(t, seeds)

	previewReq := map[string]any{
		"answers":  map[string]string{"0": "Answer to Q1."},
		"complete": false,
	}

	first := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", previewReq)
	requireStatus(t, first, 200)
	firstData := readJSON(t, first)
	firstBody, _ := firstData["body"].(string)

	if !strings.Contains(firstBody, "Some unrelated prose that must survive untouched.") {
		t.Errorf("preview body dropped unrelated prose before the section: %q", firstBody)
	}
	if !strings.Contains(firstBody, "## Another Section\n\nMore unrelated prose.") {
		t.Errorf("preview body dropped the section following Open Questions: %q", firstBody)
	}
	if !strings.Contains(firstBody, "## Open Questions") {
		t.Errorf("preview body should keep the heading unchanged for a partial (complete=false) preview, got: %q", firstBody)
	}
	if strings.Contains(firstBody, "## Resolved Questions") {
		t.Errorf("partial preview must not rename the heading, got: %q", firstBody)
	}
	if !strings.Contains(firstBody, "> Answer to Q1.") {
		t.Errorf("expected answer to be written as a blockquote, got: %q", firstBody)
	}

	// Re-run the identical preview request; the endpoint is compute-only, so
	// applying the same inputs to the same on-disk body must be idempotent.
	second := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", previewReq)
	requireStatus(t, second, 200)
	secondData := readJSON(t, second)
	secondBody, _ := secondData["body"].(string)

	if firstBody != secondBody {
		t.Errorf("preview is not idempotent:\nfirst:  %q\nsecond: %q", firstBody, secondBody)
	}
}

// TestOpenQuestionsParse_PreviewCompleteRenamesHeadingAndErrorsOnIncomplete
// verifies that the preview endpoint with complete=true renames the heading
// to "## Resolved Questions" once every question has an answer, and returns
// 422 without modifying anything when any question would still be left
// unanswered.
func TestOpenQuestionsParse_PreviewCompleteRenamesHeadingAndErrorsOnIncomplete(t *testing.T) {
	const relPath = "lifecycle/ideas/parse-preview-complete.md"
	body := "## Open Questions\n\n- Q1?\n\n- Q2?\n"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeArtifact("Parse Preview Complete", "idea", "draft", "parse-preview-complete", "", body)},
	}
	env := newTestEnv(t, seeds)

	// Only answering one of two questions must fail with 422 and leave the
	// artifact's on-disk body untouched.
	incompleteResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
		"answers":  map[string]string{"0": "Answer to Q1."},
		"complete": true,
	})
	requireStatus(t, incompleteResp, 422)

	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath+"/open-questions", nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)
	if heading, _ := getData["heading"].(string); heading != "## Open Questions" {
		t.Errorf("heading changed after a failed complete preview: %q", heading)
	}

	// Answering both questions succeeds and renames the heading.
	completeResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
		"answers": map[string]string{
			"0": "Answer to Q1.",
			"1": "Answer to Q2.",
		},
		"complete": true,
	})
	requireStatus(t, completeResp, 200)
	completeData := readJSON(t, completeResp)
	newBody, _ := completeData["body"].(string)

	if !strings.Contains(newBody, "## Resolved Questions") {
		t.Errorf("expected heading renamed to '## Resolved Questions', got: %q", newBody)
	}
	if strings.Contains(newBody, "## Open Questions") {
		t.Errorf("expected old heading to be gone after complete resolve, got: %q", newBody)
	}
	if !strings.Contains(newBody, "> Answer to Q1.") || !strings.Contains(newBody, "> Answer to Q2.") {
		t.Errorf("expected both answers written as blockquotes, got: %q", newBody)
	}
}
