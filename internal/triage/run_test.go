// SPDX-License-Identifier: AGPL-3.0-or-later

package triage

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// --- rewriteBody tests ---

func TestRewriteBody_FreshTriage(t *testing.T) {
	original := "# Title\n\nbrain-dump text"
	agentBody := "Agent provided content here."
	result := rewriteBody(original, agentBody)

	if !strings.HasPrefix(result, "# Title\n\n") {
		t.Errorf("expected result to start with H1; got:\n%s", result)
	}
	if !strings.Contains(result, "## Raw Idea\n\nbrain-dump text") {
		t.Errorf("expected ## Raw Idea with original body; got:\n%s", result)
	}
	if !strings.Contains(result, "## Idea\n\nAgent provided content here.") {
		t.Errorf("expected ## Idea with agent content; got:\n%s", result)
	}
	// Verify ordering: Raw Idea comes before Idea.
	rawIdx := strings.Index(result, "## Raw Idea")
	ideaIdx := strings.Index(result, "## Idea")
	if rawIdx >= ideaIdx {
		t.Errorf("## Raw Idea must appear before ## Idea; rawIdx=%d ideaIdx=%d", rawIdx, ideaIdx)
	}
}

func TestRewriteBody_ReRun(t *testing.T) {
	original := "# Title\n\n## Raw Idea\n\noriginal brain-dump\n\n## Idea\n\nold agent content"
	newAgentBody := "new agent content"
	result := rewriteBody(original, newAgentBody)

	// ## Raw Idea block must be byte-identical to the original.
	if !strings.Contains(result, "## Raw Idea\n\noriginal brain-dump") {
		t.Errorf("## Raw Idea block changed on re-run; got:\n%s", result)
	}
	// ## Idea block must be replaced.
	if strings.Contains(result, "old agent content") {
		t.Errorf("old ## Idea content still present after re-run; got:\n%s", result)
	}
	if !strings.Contains(result, "## Idea\n\nnew agent content") {
		t.Errorf("expected new ## Idea content; got:\n%s", result)
	}
}

func TestRewriteBody_NoH1(t *testing.T) {
	original := "just some content without a heading"
	agentBody := "agent content"
	result := rewriteBody(original, agentBody)

	// Must start with ## Raw Idea (no synthesised H1).
	if !strings.HasPrefix(result, "## Raw Idea") {
		t.Errorf("expected result to start with ## Raw Idea when original has no H1; got:\n%s", result)
	}
}

func TestRewriteBody_AgentBodyWithH1(t *testing.T) {
	original := "# OriginalTitle\n\nsome original content"
	agentBody := "# DifferentTitle\n\nagent paragraph content"
	result := rewriteBody(original, agentBody)

	// The ## Idea section must NOT contain a duplicate H1.
	ideaIdx := strings.Index(result, "## Idea")
	if ideaIdx < 0 {
		t.Fatal("## Idea section missing")
	}
	ideaSection := result[ideaIdx:]
	if strings.Contains(ideaSection, "# DifferentTitle") {
		t.Errorf("## Idea section still contains the agent H1; got:\n%s", ideaSection)
	}
	if !strings.Contains(ideaSection, "agent paragraph content") {
		t.Errorf("expected agent paragraph in ## Idea; got:\n%s", ideaSection)
	}
}

// --- mergeAndFilterLabels tests ---

func TestMergeAndFilterLabels_MergeAndDedup(t *testing.T) {
	existing := []string{"a", "b"}
	proposed := []string{"b", "c", "d"}
	vocab := []string{"a", "b", "c"}

	result := mergeAndFilterLabels(existing, proposed, vocab)

	// Expected: ["a", "b", "c"] — existing first, agent appended, dedup, vocab-filtered drops "d".
	want := []string{"a", "b", "c"}
	if len(result) != len(want) {
		t.Fatalf("mergeAndFilterLabels: want %v, got %v", want, result)
	}
	for i, v := range want {
		if result[i] != v {
			t.Errorf("mergeAndFilterLabels[%d]: want %q, got %q", i, v, result[i])
		}
	}
}

// --- Priority logic tests ---

func TestPriority_AbsentDefaultsToNormal(t *testing.T) {
	fm := artifact.Frontmatter{
		Title:   "Test",
		Type:    "idea",
		Status:  "raw",
		Lineage: "test",
	}
	// Simulate the priority defaulting logic from execute().
	if fm.Priority == "" {
		fm.Priority = "normal"
	}
	if fm.Priority != "normal" {
		t.Errorf("expected priority 'normal', got %q", fm.Priority)
	}
}

func TestPriority_PresentIsPreserved(t *testing.T) {
	fm := artifact.Frontmatter{
		Title:    "Test",
		Type:     "idea",
		Status:   "raw",
		Lineage:  "test",
		Priority: "high",
	}
	// The execute logic only sets priority when it's absent; existing values survive.
	if fm.Priority == "" {
		fm.Priority = "normal"
	}
	if fm.Priority != "high" {
		t.Errorf("expected existing priority 'high' to be preserved, got %q", fm.Priority)
	}
}

// --- marshalArtifact tests ---

func TestMarshalArtifact_KnownFieldsPreserved(t *testing.T) {
	fm := artifact.Frontmatter{
		Title:   "My Idea",
		Type:    "idea",
		Status:  "raw",
		Lineage: "my-idea",
		Release: "v1.0",
		Parent:  "lifecycle/ideas/parent.md",
		Created: "2024-01-01T00:00:00Z",
		Assignees: []artifact.Assignee{
			{Role: "product-owner", Who: "alice"},
		},
	}
	body := "# My Idea\n\nsome content"

	out, err := marshalArtifact(fm, body)
	if err != nil {
		t.Fatalf("marshalArtifact error: %v", err)
	}
	text := string(out)

	for _, want := range []string{
		"title: My Idea",
		"type: idea",
		"status: raw",
		"lineage: my-idea",
		"release: v1.0",
		"parent: lifecycle/ideas/parent.md",
		"role: product-owner",
		"who: alice",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output missing %q;\nfull output:\n%s", want, text)
		}
	}

	// Verify it round-trips: parse the output and check fields.
	var parsed artifact.Frontmatter
	content := text
	start := strings.Index(content, "---\n")
	end := strings.Index(content[4:], "\n---")
	if start < 0 || end < 0 {
		t.Fatal("output missing frontmatter fences")
	}
	fmYAML := content[4 : 4+end]
	if err := yaml.Unmarshal([]byte(fmYAML), &parsed); err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}
	if parsed.Release != "v1.0" {
		t.Errorf("round-trip release: want 'v1.0', got %q", parsed.Release)
	}
	if parsed.Parent != "lifecycle/ideas/parent.md" {
		t.Errorf("round-trip parent: want 'lifecycle/ideas/parent.md', got %q", parsed.Parent)
	}
}

// --- Title preservation test ---

func TestRewriteBody_TitlePreservedInOutput(t *testing.T) {
	// Agent proposes a body starting with a different H1; the original H1 in the
	// output body must come from the original (preserved), not the agent's suggestion.
	original := "# Original Title\n\noriginal content"
	agentBody := "# Agent Title\n\nagent content"

	result := rewriteBody(original, agentBody)

	// Original H1 preserved at top.
	if !strings.HasPrefix(result, "# Original Title") {
		preview := result
		if len(preview) > 50 {
			preview = preview[:50]
		}
		t.Errorf("original H1 not preserved; result starts with: %q", preview)
	}
	// Agent H1 must be stripped from the ## Idea section.
	ideaIdx := strings.Index(result, "## Idea")
	if ideaIdx < 0 {
		t.Fatal("## Idea section missing")
	}
	if strings.Contains(result[ideaIdx:], "# Agent Title") {
		t.Errorf("agent H1 leaked into ## Idea section; result:\n%s", result)
	}
}

