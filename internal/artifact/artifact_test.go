// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact_test

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// TestParse_RawStatus verifies that a markdown file with status: raw is parsed
// without any "unknown status" error and that FM.Status is set to "raw".
func TestParse_RawStatus(t *testing.T) {
	raw := []byte("---\ntitle: Quick capture\ntype: idea\nstatus: raw\nlineage: capture-test\n---\n\nBrain dump.\n")
	a := artifact.Parse(raw, "lifecycle/ideas/capture-test.md", time.Now())

	if a.FM.Status != "raw" {
		t.Errorf("FM.Status: want %q, got %q", "raw", a.FM.Status)
	}
	for _, e := range a.ParseErrs {
		if strings.Contains(strings.ToLower(e), "unknown status") {
			t.Errorf("unexpected unknown-status parse error: %s", e)
		}
	}
	if len(a.ParseErrs) > 0 {
		t.Errorf("unexpected parse errors: %v", a.ParseErrs)
	}
}

// TestKnownStatuses_Raw verifies that KnownStatuses["raw"] evaluates to true.
func TestKnownStatuses_Raw(t *testing.T) {
	if !artifact.KnownStatuses["raw"] {
		t.Error("KnownStatuses[\"raw\"] should be true")
	}
}

// TestParse_CreatedFieldPresent verifies that a YAML frontmatter block containing
// a well-formed `created` RFC3339 value is decoded into FM.Created.
func TestParse_CreatedFieldPresent(t *testing.T) {
	const created = "2026-04-27T10:00:00+10:00"
	raw := []byte("---\ntitle: Test\ntype: idea\nstatus: draft\nlineage: test\ncreated: \"" + created + "\"\n---\n\nBody text.\n")
	a := artifact.Parse(raw, "lifecycle/ideas/test.md", time.Now())

	if a.FM.Created == "" {
		t.Fatal("expected FM.Created to be populated, got empty string")
	}
	if a.FM.Created != created {
		t.Errorf("FM.Created: want %q, got %q", created, a.FM.Created)
	}
	// No unexpected parse errors (required fields are all present).
	for _, e := range a.ParseErrs {
		if strings.Contains(strings.ToLower(e), "created") {
			t.Errorf("unexpected parse error mentioning created: %s", e)
		}
	}
}

// TestParse_CreatedFieldAbsent verifies that when the `created` field is absent
// FM.Created is the empty string and no parse error is emitted for it.
func TestParse_CreatedFieldAbsent(t *testing.T) {
	raw := []byte("---\ntitle: Test\ntype: idea\nstatus: draft\nlineage: test\n---\n\nBody text.\n")
	a := artifact.Parse(raw, "lifecycle/ideas/test.md", time.Now())

	if a.FM.Created != "" {
		t.Errorf("expected FM.Created to be empty, got %q", a.FM.Created)
	}
	// The created field is optional; absence must not produce a parse error.
	for _, e := range a.ParseErrs {
		if strings.Contains(strings.ToLower(e), "created") {
			t.Errorf("unexpected parse error mentioning created: %s", e)
		}
	}
}

// ── HasOpenQuestions unit tests ───────────────────────────────────────────────

// TestHasOpenQuestions_HeadingWithBulletList verifies that a "## Open
// Questions" heading followed by a bullet list is detected as non-empty.
func TestHasOpenQuestions_HeadingWithBulletList(t *testing.T) {
	body := "## Open Questions\n\n- Q1\n- Q2\n"
	if !artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions to return true for heading with bullet list")
	}
}

// TestHasOpenQuestions_HeadingWithParagraph verifies that a "## Open
// Questions" heading followed by a prose paragraph is detected as non-empty.
func TestHasOpenQuestions_HeadingWithParagraph(t *testing.T) {
	body := "## Open Questions\n\nSome question here.\n"
	if !artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions to return true for heading with paragraph text")
	}
}

// TestHasOpenQuestions_HeadingWithOnlyWhitespace verifies that a "## Open
// Questions" heading whose section body contains only blank/whitespace lines
// returns false (section is considered empty).
func TestHasOpenQuestions_HeadingWithOnlyWhitespace(t *testing.T) {
	body := "## Open Questions\n\n   \n\n"
	if artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions to return false for heading with only whitespace")
	}
}

// TestHasOpenQuestions_NoHeading verifies that a body with no "## Open
// Questions" heading returns false.
func TestHasOpenQuestions_NoHeading(t *testing.T) {
	body := "This is just a regular body.\n\nNo special headings here.\n"
	if artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions to return false when heading is absent")
	}
}

// TestHasOpenQuestions_HeadingAtWrongLevel verifies that "### Open Questions"
// (H3, not H2) does not match — the function requires exactly "## ".
func TestHasOpenQuestions_HeadingAtWrongLevel(t *testing.T) {
	body := "### Open Questions\n\n- Q1\n"
	if artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions to return false for H3 heading (must be H2)")
	}
}

// TestHasOpenQuestions_HeadingMidDocument verifies that a "## Open Questions"
// section appearing after other content in the document is still detected.
func TestHasOpenQuestions_HeadingMidDocument(t *testing.T) {
	body := "# Title\n\nSome intro text.\n\n## Background\n\nContext here.\n\n## Open Questions\n\n- Is this working?\n"
	if !artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions to return true when heading appears mid-document with content")
	}
}

// TestHasOpenQuestions_HeadingFollowedImmediatelyByNextHeading verifies that
// "## Open Questions" immediately followed by another "## " heading (with no
// content lines in between) returns false.
func TestHasOpenQuestions_HeadingFollowedImmediatelyByNextHeading(t *testing.T) {
	body := "## Open Questions\n## Next Section\n"
	if artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions to return false when heading is immediately followed by next H2")
	}
}

// TestParse_CreatedFieldRoundTrip verifies that marshalling the parsed Frontmatter
// back to YAML and re-parsing it preserves the `created` value exactly.
func TestParse_CreatedFieldRoundTrip(t *testing.T) {
	const created = "2026-04-27T10:00:00+10:00"
	raw := []byte("---\ntitle: RT\ntype: idea\nstatus: draft\nlineage: rt\ncreated: \"" + created + "\"\n---\n\nRound-trip body.\n")

	a := artifact.Parse(raw, "lifecycle/ideas/rt.md", time.Now())
	if a.FM.Created != created {
		t.Fatalf("initial parse: want %q, got %q", created, a.FM.Created)
	}

	// Marshal the Frontmatter struct back to YAML (simulating buildMarkdown).
	fmBytes, err := yaml.Marshal(a.FM)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	rebuilt := "---\n" + string(fmBytes) + "---\n\nRound-trip body.\n"

	a2 := artifact.Parse([]byte(rebuilt), "lifecycle/ideas/rt.md", time.Now())
	if a2.FM.Created != created {
		t.Errorf("round-trip: want %q, got %q", created, a2.FM.Created)
	}
}
