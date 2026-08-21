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

// ── RICE frontmatter fields ───────────────────────────────────────────────────

// TestParse_RiceFieldsAbsent verifies a file with no RICE fields parses
// without ParseErrs and decodes all four RICE pointers to nil, not 0.
func TestParse_RiceFieldsAbsent(t *testing.T) {
	raw := []byte("---\ntitle: Test\ntype: idea\nstatus: draft\nlineage: test\n---\n\nBody text.\n")
	a := artifact.Parse(raw, "lifecycle/ideas/test.md", time.Now())

	if len(a.ParseErrs) > 0 {
		t.Errorf("unexpected parse errors: %v", a.ParseErrs)
	}
	if a.FM.RiceReach != nil || a.FM.RiceImpact != nil || a.FM.RiceConfidence != nil || a.FM.RiceEffort != nil {
		t.Errorf("expected all RICE fields nil, got reach=%v impact=%v confidence=%v effort=%v",
			a.FM.RiceReach, a.FM.RiceImpact, a.FM.RiceConfidence, a.FM.RiceEffort)
	}
}

// TestParse_RiceFieldsPartialSubset verifies a file with a subset of the four
// RICE fields parses without ParseErrs; set fields decode to non-nil, absent
// fields decode to nil.
func TestParse_RiceFieldsPartialSubset(t *testing.T) {
	raw := []byte("---\ntitle: Test\ntype: idea\nstatus: draft\nlineage: test\nrice_reach: 100\nrice_effort: 2\n---\n\nBody text.\n")
	a := artifact.Parse(raw, "lifecycle/ideas/test.md", time.Now())

	if len(a.ParseErrs) > 0 {
		t.Errorf("unexpected parse errors: %v", a.ParseErrs)
	}
	if a.FM.RiceReach == nil || *a.FM.RiceReach != 100 {
		t.Errorf("RiceReach: want 100, got %v", a.FM.RiceReach)
	}
	if a.FM.RiceEffort == nil || *a.FM.RiceEffort != 2 {
		t.Errorf("RiceEffort: want 2, got %v", a.FM.RiceEffort)
	}
	if a.FM.RiceImpact != nil {
		t.Errorf("RiceImpact: want nil, got %v", a.FM.RiceImpact)
	}
	if a.FM.RiceConfidence != nil {
		t.Errorf("RiceConfidence: want nil, got %v", a.FM.RiceConfidence)
	}
}

// TestParse_RiceFieldZeroIsNotNil verifies a RICE field present with value 0
// decodes to a non-nil pointer to 0, distinct from an absent field.
func TestParse_RiceFieldZeroIsNotNil(t *testing.T) {
	raw := []byte("---\ntitle: Test\ntype: idea\nstatus: draft\nlineage: test\nrice_reach: 0\nrice_impact: 1\nrice_confidence: 50\nrice_effort: 1\n---\n\nBody text.\n")
	a := artifact.Parse(raw, "lifecycle/ideas/test.md", time.Now())

	if a.FM.RiceReach == nil {
		t.Fatal("RiceReach: want non-nil pointer to 0, got nil")
	}
	if *a.FM.RiceReach != 0 {
		t.Errorf("RiceReach: want 0, got %v", *a.FM.RiceReach)
	}
}

// TestFrontmatter_RiceYAMLRoundTrip verifies that yaml.Marshal (the mechanism
// buildMarkdown relies on) omits nil RICE fields and emits set fields with
// their numeric value.
func TestRemoveFrontmatterListItem(t *testing.T) {
	const doc = "---\n" +
		"title: X\n" +
		"type: architecture\n" +
		"labels:\n" +
		"    - architecture\n" +
		"    - catalog\n" +
		"    - collaborative\n" +
		"related_to:\n" +
		"    - catalog\n" + // same value under a different key — must survive
		"---\n\nBody with a - catalog dash that must survive.\n"

	out, ok := artifact.RemoveFrontmatterListItem([]byte(doc), "labels", "catalog")
	if !ok {
		t.Fatal("expected the catalog label to be removed")
	}
	s := string(out)
	// The labels block lost catalog but kept its siblings.
	if strings.Contains(s, "labels:\n    - architecture\n    - collaborative\n") == false {
		t.Errorf("labels block not as expected:\n%s", s)
	}
	// The catalog entry under related_to and the body dash are untouched.
	if !strings.Contains(s, "related_to:\n    - catalog\n") {
		t.Errorf("related_to catalog entry was wrongly removed:\n%s", s)
	}
	if !strings.Contains(s, "Body with a - catalog dash that must survive.") {
		t.Errorf("body was modified:\n%s", s)
	}

	// Absent item / absent key → (raw, false).
	if _, ok := artifact.RemoveFrontmatterListItem([]byte(doc), "labels", "nope"); ok {
		t.Error("expected false for an item that is not present")
	}
	if _, ok := artifact.RemoveFrontmatterListItem([]byte(doc), "missing", "catalog"); ok {
		t.Error("expected false for a key that is not present")
	}
}

func TestFrontmatter_RiceYAMLRoundTrip(t *testing.T) {
	fm := artifact.Frontmatter{
		Title: "Test", Type: "idea", Status: "draft", Lineage: "test",
		RiceReach:  f64(50),
		RiceEffort: f64(0), // a real zero, must still be emitted
		// RiceImpact and RiceConfidence left nil.
	}
	out, err := yaml.Marshal(fm)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "rice_reach: 50") {
		t.Errorf("expected rice_reach: 50 in output, got:\n%s", s)
	}
	if !strings.Contains(s, "rice_effort: 0") {
		t.Errorf("expected rice_effort: 0 in output (real zero must round-trip), got:\n%s", s)
	}
	if strings.Contains(s, "rice_impact") {
		t.Errorf("expected rice_impact absent (nil), got:\n%s", s)
	}
	if strings.Contains(s, "rice_confidence") {
		t.Errorf("expected rice_confidence absent (nil), got:\n%s", s)
	}

	// Round-trip: unmarshal back and confirm the nil/set distinction survives.
	var fm2 artifact.Frontmatter
	if err := yaml.Unmarshal(out, &fm2); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if fm2.RiceReach == nil || *fm2.RiceReach != 50 {
		t.Errorf("round-tripped RiceReach: want 50, got %v", fm2.RiceReach)
	}
	if fm2.RiceEffort == nil || *fm2.RiceEffort != 0 {
		t.Errorf("round-tripped RiceEffort: want 0, got %v", fm2.RiceEffort)
	}
	if fm2.RiceImpact != nil {
		t.Errorf("round-tripped RiceImpact: want nil, got %v", fm2.RiceImpact)
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
// TestHasOpenQuestions_HeadingLowercaseAndWhitespace verifies the heading match
// tolerates natural casing and surrounding whitespace, so an artifact authored
// with "## Open questions" is still detected (and thus auto-blocked / listed).
func TestHasOpenQuestions_HeadingLowercaseAndWhitespace(t *testing.T) {
	for _, heading := range []string{"## Open questions", "##  OPEN QUESTIONS ", "## open questions"} {
		body := heading + "\n\n- Q1\n- Q2\n"
		if !artifact.HasOpenQuestions(body) {
			t.Errorf("expected HasOpenQuestions=true for heading %q", heading)
		}
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

// TestHasOpenQuestions_SentinelIsNotBlocking verifies that a section whose only
// content is a "no questions" sentinel (None, N/A, etc., with or without a list
// marker or punctuation) is treated as empty and does NOT block.
func TestHasOpenQuestions_SentinelIsNotBlocking(t *testing.T) {
	sentinelBodies := []string{
		"## Open Questions\n\nNone\n",
		"## Open Questions\n\n- None\n",
		"## Open Questions\n\nN/A\n",
		"## Open Questions\n\n_None._\n",
		"## Open Questions\n\nNo open questions\n",
		"## Open Questions\n\n- No questions\n",
		"## Open Questions\n\nTBD\n",
	}
	for _, body := range sentinelBodies {
		if artifact.HasOpenQuestions(body) {
			t.Errorf("expected HasOpenQuestions=false (non-blocking) for sentinel body %q", body)
		}
	}
}

// TestHasOpenQuestions_RealQuestionAlongsideSentinelBlocks verifies that a
// genuine question still blocks even when a sentinel line is also present.
func TestHasOpenQuestions_RealQuestionAlongsideSentinelBlocks(t *testing.T) {
	body := "## Open Questions\n\n- None\n- What auth model should we use?\n"
	if !artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions=true when a real question is present alongside a sentinel")
	}
}

// TestHasOpenQuestions_QuestionContainingSentinelWordBlocks verifies that a real
// question that merely contains a sentinel word (e.g. "none") is not mistaken
// for a sentinel.
func TestHasOpenQuestions_QuestionContainingSentinelWordBlocks(t *testing.T) {
	body := "## Open Questions\n\n- Should none of the users have admin rights?\n"
	if !artifact.HasOpenQuestions(body) {
		t.Error("expected HasOpenQuestions=true for a real question containing a sentinel word")
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

// TestParse_ArchitectureZoneCleanSlugNoLineage verifies that a clean-slug file
// under lifecycle/architecture/ with no lineage: field parses with zero
// ParseErrs, Index == 0, and Lineage left empty (not backfilled to slug).
func TestParse_ArchitectureZoneCleanSlugNoLineage(t *testing.T) {
	raw := []byte("---\ntitle: Postgres Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n")
	a := artifact.Parse(raw, "lifecycle/architecture/postgres-modular-monolith.md", time.Now())

	if len(a.ParseErrs) != 0 {
		t.Errorf("expected zero ParseErrs, got %v", a.ParseErrs)
	}
	if a.Index != 0 {
		t.Errorf("Index: want 0, got %d", a.Index)
	}
	if a.FM.Lineage != "" {
		t.Errorf("Lineage: want empty, got %q", a.FM.Lineage)
	}
}

// TestParse_ArchitectureZonePromotedCopyParentEdge verifies that a promoted
// copy whose parent: points at a catalog entry parses with zero ParseErrs and
// emits a parent edge to that target.
func TestParse_ArchitectureZonePromotedCopyParentEdge(t *testing.T) {
	raw := []byte("---\ntitle: Postgres Modular Monolith\ntype: architecture\nstatus: approved\nparent: lifecycle/architecture/architectures/postgres-modular-monolith.md\n---\n\nBody.\n")
	a := artifact.Parse(raw, "lifecycle/architecture/postgres-modular-monolith.md", time.Now())

	if len(a.ParseErrs) != 0 {
		t.Errorf("expected zero ParseErrs, got %v", a.ParseErrs)
	}
	found := false
	for _, l := range a.Links {
		if l.Kind == artifact.EdgeKindParent && l.To == "lifecycle/architecture/architectures/postgres-modular-monolith.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a parent edge to the catalog source, got links: %+v", a.Links)
	}
}

// TestParse_ADRTypeValid verifies that type: adr validates (no "unknown type"
// ParseErr) and that type: doc still validates.
func TestParse_ADRTypeValid(t *testing.T) {
	raw := []byte("---\ntitle: Adopt X\ntype: adr\nstatus: draft\n---\n\nBody.\n")
	a := artifact.Parse(raw, "lifecycle/architecture/decisions/adr-0001-adopt-x.md", time.Now())
	for _, e := range a.ParseErrs {
		if strings.Contains(e, "unknown type") {
			t.Errorf("unexpected unknown-type parse error: %s", e)
		}
	}

	raw2 := []byte("---\ntitle: A doc\ntype: doc\nstatus: draft\nlineage: a-doc\n---\n\nBody.\n")
	a2 := artifact.Parse(raw2, "lifecycle/docs/a-doc.md", time.Now())
	for _, e := range a2.ParseErrs {
		if strings.Contains(e, "unknown type") {
			t.Errorf("unexpected unknown-type parse error: %s", e)
		}
	}
}

// TestParse_OutsideArchitectureZoneStillRequiresLineage is a regression guard:
// a file outside lifecycle/architecture/ with no lineage: still records the
// "missing required field: lineage" ParseErr — the relaxation is path-scoped.
func TestParse_OutsideArchitectureZoneStillRequiresLineage(t *testing.T) {
	raw := []byte("---\ntitle: Some idea\ntype: idea\nstatus: draft\n---\n\nBody.\n")
	a := artifact.Parse(raw, "lifecycle/ideas/some-idea.md", time.Now())

	found := false
	for _, e := range a.ParseErrs {
		if e == "missing required field: lineage" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing-lineage ParseErr for a non-architecture path, got %v", a.ParseErrs)
	}
	if a.FM.Lineage != "some-idea" {
		t.Errorf("Lineage: want backfilled slug %q, got %q", "some-idea", a.FM.Lineage)
	}
}

// TestIsArchitecturePath verifies the path-prefix helper directly, including
// Windows-style separators normalised via filepath.ToSlash.
func TestIsArchitecturePath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"lifecycle/architecture/postgres-modular-monolith.md", true},
		{"lifecycle/architecture/decisions/adr-0001-adopt-x.md", true},
		{"lifecycle/ideas/login.md", false},
		{"lifecycle/architecture-templates/foo.md", false},
	}
	for _, c := range cases {
		if got := artifact.IsArchitecturePath(c.path); got != c.want {
			t.Errorf("IsArchitecturePath(%q): want %v, got %v", c.path, c.want, got)
		}
	}
}

// TestParse_TypedRelationshipFieldsAbsent verifies that omitting evolves_into,
// alternative_to, and composed_with produces zero new ParseErrs and no
// corresponding links (regression-safe degrade, FR-4/NFR-5).
func TestParse_TypedRelationshipFieldsAbsent(t *testing.T) {
	raw := []byte("---\ntitle: Postgres Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n")
	a := artifact.Parse(raw, "lifecycle/architecture/architectures/postgres-modular-monolith.md", time.Now())

	if len(a.ParseErrs) != 0 {
		t.Errorf("expected zero ParseErrs, got %v", a.ParseErrs)
	}
	for _, l := range a.Links {
		switch l.Kind {
		case artifact.EdgeKindEvolvesInto, artifact.EdgeKindAlternativeTo, artifact.EdgeKindComposedWith:
			t.Errorf("unexpected typed relationship link with no source field: %+v", l)
		}
	}
}

// TestParse_TypedRelationshipFieldEvolvesInto verifies that evolves_into
// yields a link classified with Kind == "evolves_into" whose target resolves
// the same way related_to targets do.
func TestParse_TypedRelationshipFieldEvolvesInto(t *testing.T) {
	raw := []byte("---\ntitle: Postgres Modular Monolith\ntype: architecture\nstatus: approved\nevolves_into:\n  - architecture/architectures/foo.md\n---\n\nBody.\n")
	a := artifact.Parse(raw, "lifecycle/architecture/architectures/postgres-modular-monolith.md", time.Now())

	if len(a.ParseErrs) != 0 {
		t.Errorf("expected zero ParseErrs, got %v", a.ParseErrs)
	}
	found := false
	for _, l := range a.Links {
		if l.Kind == artifact.EdgeKindEvolvesInto {
			found = true
			if l.To != "lifecycle/architecture/architectures/foo.md" {
				t.Errorf("evolves_into target: want %q, got %q", "lifecycle/architecture/architectures/foo.md", l.To)
			}
		}
	}
	if !found {
		t.Errorf("expected an evolves_into link, got links: %+v", a.Links)
	}
}

// TestParse_TypedRelationshipFieldMalformedDoesNotPanic verifies that a
// scalar value in place of a list fails only that artifact's parse via the
// existing frontmatter-unmarshal error path and does not panic.
func TestParse_TypedRelationshipFieldMalformedDoesNotPanic(t *testing.T) {
	raw := []byte("---\ntitle: Postgres Modular Monolith\ntype: architecture\nstatus: approved\nevolves_into: not-a-list\n---\n\nBody.\n")
	a := artifact.Parse(raw, "lifecycle/architecture/architectures/postgres-modular-monolith.md", time.Now())

	found := false
	for _, e := range a.ParseErrs {
		if strings.Contains(e, "frontmatter decode error") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a frontmatter decode error, got %v", a.ParseErrs)
	}
}
