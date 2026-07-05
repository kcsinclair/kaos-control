// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
)

// TestOpenQuestionsConfigDefaultFormat verifies that the default answer format
// is "blockquote" when no open_questions configuration is present.
func TestOpenQuestionsConfigDefaultFormat(t *testing.T) {
	seeds := []SeedArtifact{
		{
			RelPath: "lifecycle/ideas/test-config-default.md",
			Content: MakeArtifact("Test Config Default", "idea", "draft", "test-config-default", "", "Initial body."),
		},
	}
	env := NewTestEnv(t, seeds)

	resp := env.DoRequest("GET", "/api/p/testproject/config/open-questions", nil)
	RequireStatus(t, resp, 200)
	data := ReadJSON(t, resp)

	format, ok := data["answer_format"].(string)
	if !ok {
		t.Fatalf("expected answer_format field in response")
	}
	if format != "blockquote" {
		t.Errorf("expected default format 'blockquote', got %q", format)
	}
}