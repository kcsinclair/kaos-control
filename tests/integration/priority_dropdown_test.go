// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPriorityDropdownCreateNormal verifies that an artifact created with
// priority: normal is returned as "normal" by GET /artifacts/:path.
func TestPriorityDropdownCreateNormal(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	path := createArtifactViaAPI(t, env, "ideas", "pd-create-normal", map[string]any{
		"title":    "Priority Dropdown Create Normal",
		"type":     "idea",
		"status":   "draft",
		"lineage":  "pd-create-normal",
		"priority": "normal",
	}, "Body.")

	fm := artifactFrontmatterJSON(t, env, path)
	if got, _ := fm["priority"].(string); got != "normal" {
		t.Errorf("priority after create: want %q, got %q", "normal", got)
	}
}

// TestPriorityDropdownUpdateToHigh verifies that updating priority to "high"
// via PUT persists and is returned by GET.
func TestPriorityDropdownUpdateToHigh(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/pd-to-high.md",
			content: makeArtifactWithPriority("Priority Dropdown To High", "idea", "draft", "pd-to-high", "normal", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/pd-to-high.md"

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+path, map[string]any{
		"frontmatter": map[string]any{
			"title":    "Priority Dropdown To High",
			"type":     "idea",
			"status":   "draft",
			"lineage":  "pd-to-high",
			"priority": "high",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	fm := artifactFrontmatterJSON(t, env, path)
	if got, _ := fm["priority"].(string); got != "high" {
		t.Errorf("priority after PUT high: want %q, got %q", "high", got)
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "priority: high") {
		t.Errorf("disk file does not contain 'priority: high':\n%s", raw)
	}
}

// TestPriorityDropdownUnsetViaEmpty verifies that sending priority="" (the
// "— none —" selection) via PUT causes the priority key to be absent or empty
// in the returned frontmatter and on disk.
func TestPriorityDropdownUnsetViaEmpty(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/pd-unset.md",
			content: makeArtifactWithPriority("Priority Dropdown Unset", "idea", "draft", "pd-unset", "high", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/pd-unset.md"

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+path, map[string]any{
		"frontmatter": map[string]any{
			"title":    "Priority Dropdown Unset",
			"type":     "idea",
			"status":   "draft",
			"lineage":  "pd-unset",
			"priority": "",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	fm := artifactFrontmatterJSON(t, env, path)
	// After clearing priority, the field should either be absent or empty.
	if got, _ := fm["priority"].(string); got != "" {
		t.Errorf("priority after unset: want empty, got %q", got)
	}

	// On disk the priority key must be absent.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "priority:") {
		t.Errorf("disk file still contains 'priority:' after unset:\n%s", raw)
	}
}

// TestPriorityDropdownUnknownValueAccepted verifies that an unknown priority
// value (e.g. "critical") can be written and read back without error — the
// backend accepts any string value for priority.
func TestPriorityDropdownUnknownValueAccepted(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/pd-unknown.md",
			content: makeArtifact("Priority Dropdown Unknown", "idea", "draft", "pd-unknown", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/pd-unknown.md"
	const unknownPriority = "critical"

	// Test via PUT.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+path, map[string]any{
		"frontmatter": map[string]any{
			"title":    "Priority Dropdown Unknown",
			"type":     "idea",
			"status":   "draft",
			"lineage":  "pd-unknown",
			"priority": unknownPriority,
		},
		"body": "Body.",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	fm := artifactFrontmatterJSON(t, env, path)
	if got, _ := fm["priority"].(string); got != unknownPriority {
		t.Errorf("GET priority after PUT unknown: want %q, got %q", unknownPriority, got)
	}

	// Test via PATCH.
	const unknownPriority2 = "extreme"
	resp2 := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": unknownPriority2,
	})
	requireStatus(t, resp2, 200)
	resp2.Body.Close()

	fm2 := artifactFrontmatterJSON(t, env, path)
	if got, _ := fm2["priority"].(string); got != unknownPriority2 {
		t.Errorf("GET priority after PATCH unknown: want %q, got %q", unknownPriority2, got)
	}
}
