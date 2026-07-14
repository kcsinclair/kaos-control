// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"context"
	"testing"
)

// validProposalJSON returns a minimal valid propose JSON string accepted by
// parseAction. The slug "test-feature" passes the slug regex, so resolveSlug
// will not trigger a second LLM call.
func validProposalJSON() string {
	return `{"action":"propose","reply":"Here is your feature proposal.","slug":"test-feature","title":"Test Feature","labels":[],"body":"# Test Feature\n\nA concise feature proposal."}`
}

// stubCallLLM replaces CallLLM with a deterministic fake that always returns
// validProposalJSON. It restores the original via t.Cleanup.
func stubCallLLM(t *testing.T) {
	t.Helper()
	orig := CallLLM
	t.Cleanup(func() { CallLLM = orig })
	CallLLM = func(_ context.Context, _ ModelConfig, _ []LLMMessage) (string, error) {
		return validProposalJSON(), nil
	}
}

// TestGenerate_InheritsPriorityFromSource verifies that SourcePriority is
// reflected in the result frontmatter (FR-6).
func TestGenerate_InheritsPriorityFromSource(t *testing.T) {
	stubCallLLM(t)

	result, err := Generate(context.Background(), GenerateOptions{
		Input:          "Add a dark mode toggle to the settings page for better usability",
		ArtifactType:   "idea",
		SourcePriority: "high",
		ModelCfg:       ModelConfig{Model: "test-model"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	prio, _ := result.Frontmatter["priority"].(string)
	if prio != "high" {
		t.Errorf("Frontmatter priority: want %q, got %v", "high", result.Frontmatter["priority"])
	}
}

// TestGenerate_InheritsReleaseFromSource verifies that SourceRelease is
// reflected in the result frontmatter (FR-6).
func TestGenerate_InheritsReleaseFromSource(t *testing.T) {
	stubCallLLM(t)

	result, err := Generate(context.Background(), GenerateOptions{
		Input:         "Add a dark mode toggle to the settings page for better usability",
		ArtifactType:  "idea",
		SourceRelease: "KC-Release4",
		ModelCfg:      ModelConfig{Model: "test-model"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	rel, ok := result.Frontmatter["release"].(string)
	if !ok || rel != "KC-Release4" {
		t.Errorf("Frontmatter release: want %q, got %v", "KC-Release4", result.Frontmatter["release"])
	}
}

// TestGenerate_ParentlessDefaultsPriorityNormal verifies that when no
// SourcePriority is provided the frontmatter gets "normal" and no release key
// (Resolved Q3 from the spec).
func TestGenerate_ParentlessDefaultsPriorityNormal(t *testing.T) {
	stubCallLLM(t)

	result, err := Generate(context.Background(), GenerateOptions{
		Input:        "Add a dark mode toggle to the settings page for better usability",
		ArtifactType: "idea",
		ModelCfg:     ModelConfig{Model: "test-model"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	prio, _ := result.Frontmatter["priority"].(string)
	if prio != "normal" {
		t.Errorf("Frontmatter priority: want %q (default when no source), got %q", "normal", prio)
	}
	if _, hasRelease := result.Frontmatter["release"]; hasRelease {
		t.Errorf("Frontmatter release: want key absent when SourceRelease is empty, got %v", result.Frontmatter["release"])
	}
}

// TestGenerate_ParentWithNoPriorityFallsToNormal verifies that an empty
// SourcePriority (parent has no priority field) falls back to "normal" (FR-6).
func TestGenerate_ParentWithNoPriorityFallsToNormal(t *testing.T) {
	stubCallLLM(t)

	result, err := Generate(context.Background(), GenerateOptions{
		Input:          "Add a dark mode toggle to the settings page for better usability",
		ArtifactType:   "idea",
		SourcePriority: "", // parent present but has no priority
		SourcePath:     "lifecycle/ideas/some-parent.md",
		ModelCfg:       ModelConfig{Model: "test-model"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	prio, _ := result.Frontmatter["priority"].(string)
	if prio != "normal" {
		t.Errorf("Frontmatter priority: want %q (fallback when parent has no priority), got %q", "normal", prio)
	}
}

// TestGenerate_PreviewFrontmatterReflectsInheritedValues verifies that the
// GenerateResult's Frontmatter (the preview response) already reflects the
// inherited values before any artifact is persisted to disk.
func TestGenerate_PreviewFrontmatterReflectsInheritedValues(t *testing.T) {
	stubCallLLM(t)

	result, err := Generate(context.Background(), GenerateOptions{
		Input:          "Add a dark mode toggle to the settings page for better usability",
		ArtifactType:   "idea",
		SourcePriority: "high",
		SourceRelease:  "KC-Release4",
		ModelCfg:       ModelConfig{Model: "test-model"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	prio, _ := result.Frontmatter["priority"].(string)
	if prio != "high" {
		t.Errorf("preview Frontmatter priority: want %q, got %q", "high", prio)
	}
	rel, _ := result.Frontmatter["release"].(string)
	if rel != "KC-Release4" {
		t.Errorf("preview Frontmatter release: want %q, got %q", "KC-Release4", rel)
	}
}
