package artifact_test

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

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
