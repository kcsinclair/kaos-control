// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// TestParseErrorsForMalformedArtifact verifies that a malformed artifact
// (missing required fields) produces parse errors visible via the API.
// Test plan §8: "Parse errors" scenario.
func TestParseErrorsForMalformedArtifact(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/good.md",
			content: makeArtifact("Good Artifact", "idea", "draft", "good", "", "Valid."),
		},
		{
			// Missing title, type, status, lineage — all required.
			relPath: "lifecycle/ideas/bad.md",
			content: "---\n---\n\nNo frontmatter fields at all.\n",
		},
	}

	env := newTestEnv(t, seeds)

	resp, err := http.Get(env.baseURL + "/api/p/testproject/parse-errors")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	errors, ok := data["errors"].([]any)
	if !ok {
		t.Fatal("expected errors array")
	}

	// The malformed artifact should have parse errors.
	if len(errors) == 0 {
		t.Error("expected at least one parse error for the malformed artifact")
	}

	// Look for errors mentioning the bad artifact.
	var foundBad bool
	for _, e := range errors {
		errMap, _ := e.(map[string]any)
		path, _ := errMap["path"].(string)
		if path == "lifecycle/ideas/bad.md" {
			foundBad = true
			break
		}
	}
	if !foundBad {
		t.Error("expected parse error for lifecycle/ideas/bad.md")
	}
}
