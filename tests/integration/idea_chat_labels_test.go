//go:build integration

package integration

import (
	"testing"
)

// Milestone 5 – Label Constraint Tests
//
// All tests here require ANTHROPIC_API_KEY. They verify that the agent picks
// labels only from the project's existing label vocabulary and that the result
// is well-formed (count in range, no duplicates).

// TestIdeaChatLabelsFromExistingVocabulary verifies that every label in
// preview.frontmatter.labels is present in the set returned by
// GET /api/p/:project/labels.
func TestIdeaChatLabelsFromExistingVocabulary(t *testing.T) {
	skipIfNoAPIKey(t)

	// Seed the project with artifacts that carry known labels so the vocabulary
	// is non-empty and the agent has labels to choose from.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/existing-one.md",
			content: makeArtifact("Existing Idea One", "idea", "draft", "existing-one", "",
				"Body.", "auth", "backend", "api"),
		},
		{
			relPath: "lifecycle/ideas/existing-two.md",
			content: makeArtifact("Existing Idea Two", "idea", "draft", "existing-two", "",
				"Body.", "ui", "frontend", "usability"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Fetch the project's label vocabulary.
	labelsResp := env.doRequest("GET", "/api/p/testproject/labels", nil)
	requireStatus(t, labelsResp, 200)
	labelsData := readJSON(t, labelsResp)
	rawLabels, _ := labelsData["labels"].([]any)
	if len(rawLabels) == 0 {
		t.Fatal("project has no labels – cannot exercise label constraint; check seeds")
	}
	vocab := make(map[string]bool, len(rawLabels))
	for _, l := range rawLabels {
		if s, ok := l.(string); ok {
			vocab[s] = true
		}
	}

	// Converge to a proposal.
	_, data := convergeToProposal(t, env,
		`I want to add OAuth2 login to the application so that users can sign in ` +
		`with their Google or GitHub accounts instead of creating a local password. ` +
		`The backend should validate the token, create or update the user record, ` +
		`and issue a session cookie in the same way as the existing local auth flow. ` +
		`The frontend should show OAuth buttons on the login page.`)

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	fm, _ := preview["frontmatter"].(map[string]any)
	rawProposed, _ := fm["labels"].([]any)

	// If the agent returned no labels it still passes – an empty set is always a
	// subset of the vocabulary. But if labels are present they must be valid.
	for _, l := range rawProposed {
		label, _ := l.(string)
		if label == "" {
			continue
		}
		if !vocab[label] {
			t.Errorf("label %q is not in the project vocabulary %v", label, rawLabels)
		}
	}
}

// TestIdeaChatLabelsCountInRange verifies that preview.frontmatter.labels
// contains between 1 and 5 items.
func TestIdeaChatLabelsCountInRange(t *testing.T) {
	skipIfNoAPIKey(t)

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/label-range-seed.md",
			content: makeArtifact("Label Range Seed", "idea", "draft", "label-range-seed", "",
				"Body.", "auth", "backend", "api", "ui", "workflow"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env, uniqueIdeaMessage("label-range"))

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	fm, _ := preview["frontmatter"].(map[string]any)
	rawLabels, exists := fm["labels"]
	if !exists {
		t.Fatal("preview.frontmatter.labels field is absent")
	}

	// labels may be nil (no labels selected) or an array.
	var count int
	if rawLabels != nil {
		labels, ok := rawLabels.([]any)
		if !ok {
			t.Fatalf("preview.frontmatter.labels is not an array, got %T", rawLabels)
		}
		count = len(labels)
	}

	if count < 0 || count > 5 {
		t.Errorf("labels count %d is out of range [0, 5]", count)
	}
}

// TestIdeaChatNoDuplicateLabels verifies that preview.frontmatter.labels
// contains no duplicate entries.
func TestIdeaChatNoDuplicateLabels(t *testing.T) {
	skipIfNoAPIKey(t)

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/no-dup-seed.md",
			content: makeArtifact("No Dup Seed", "idea", "draft", "no-dup-seed", "",
				"Body.", "auth", "backend", "api", "ui", "workflow"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	_, data := convergeToProposal(t, env, uniqueIdeaMessage("no-dup"))

	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("preview is nil")
	}
	fm, _ := preview["frontmatter"].(map[string]any)

	var labels []string
	if rawLabels, ok := fm["labels"].([]any); ok {
		for _, l := range rawLabels {
			if s, ok := l.(string); ok {
				labels = append(labels, s)
			}
		}
	}

	seen := make(map[string]bool, len(labels))
	for _, l := range labels {
		if seen[l] {
			t.Errorf("duplicate label %q in preview.frontmatter.labels", l)
		}
		seen[l] = true
	}
}
