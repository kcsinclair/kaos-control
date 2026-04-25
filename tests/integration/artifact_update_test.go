//go:build integration

package integration

import (
	"testing"
)

// TestPutArtifactValidPriority verifies that PUT with a valid priority value
// (e.g. "normal") succeeds.
func TestPutArtifactValidPriority(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/put-valid-prio.md",
			content: makeArtifact("PUT Valid Priority", "idea", "draft", "put-valid-prio", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/put-valid-prio.md"

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+path, map[string]any{
		"frontmatter": map[string]any{
			"title":   "PUT Valid Priority",
			"type":    "idea",
			"status":  "draft",
			"lineage": "put-valid-prio",
			"priority": "normal",
		},
		"body": "Updated body.",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	fm, _ := artifact["frontmatter"].(map[string]any)
	if priority, _ := fm["priority"].(string); priority != "normal" {
		t.Errorf("PUT priority: want %q, got %q", "normal", priority)
	}
}

// TestPutArtifactInvalidPriority verifies that PUT with an unrecognised
// priority value returns 400 and the error message lists allowed values.
func TestPutArtifactInvalidPriority(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/put-invalid-prio.md",
			content: makeArtifact("PUT Invalid Priority", "idea", "draft", "put-invalid-prio", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/put-invalid-prio.md"

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+path, map[string]any{
		"frontmatter": map[string]any{
			"title":   "PUT Invalid Priority",
			"type":    "idea",
			"status":  "draft",
			"lineage": "put-invalid-prio",
			"priority": "urgent",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "bad_request" {
		t.Errorf("expected error code 'bad_request', got %q", code)
	}
	// The error message should mention allowed values.
	msg, _ := errData["message"].(string)
	for _, want := range []string{"high", "medium", "normal", "low"} {
		if !containsSubstring(msg, want) {
			t.Errorf("error message %q should mention allowed value %q", msg, want)
		}
	}
}

// TestPutArtifactEmptyPriority verifies that PUT with priority="" succeeds
// (unset is valid).
func TestPutArtifactEmptyPriority(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/put-empty-prio.md",
			content: makeArtifactWithPriority("PUT Empty Priority", "idea", "draft", "put-empty-prio", "high", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/put-empty-prio.md"

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+path, map[string]any{
		"frontmatter": map[string]any{
			"title":    "PUT Empty Priority",
			"type":     "idea",
			"status":   "draft",
			"lineage":  "put-empty-prio",
			"priority": "",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()
}

// TestPutArtifactNoPriorityField verifies that PUT without a priority field at
// all succeeds (the field is optional).
func TestPutArtifactNoPriorityField(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/put-no-prio.md",
			content: makeArtifact("PUT No Priority Field", "idea", "draft", "put-no-prio", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/put-no-prio.md"

	// Send frontmatter without a "priority" key at all.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+path, map[string]any{
		"frontmatter": map[string]any{
			"title":   "PUT No Priority Field",
			"type":    "idea",
			"status":  "draft",
			"lineage": "put-no-prio",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()
}

// containsSubstring returns true if s contains substr (plain substring, not regex).
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
