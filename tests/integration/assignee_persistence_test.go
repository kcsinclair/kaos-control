// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// makeArtifactWithAssignees builds a markdown artifact string that includes an assignees block.
// Each entry in assignees is a {role, who} pair.
func makeArtifactWithAssignees(title, typ, status, lineage string, assignees []map[string]string, body string) string {
	var sb []byte
	sb = append(sb, "---\n"...)
	sb = append(sb, fmt.Sprintf("title: %s\n", title)...)
	sb = append(sb, fmt.Sprintf("type: %s\n", typ)...)
	sb = append(sb, fmt.Sprintf("status: %s\n", status)...)
	sb = append(sb, fmt.Sprintf("lineage: %s\n", lineage)...)
	if len(assignees) > 0 {
		sb = append(sb, "assignees:\n"...)
		for _, a := range assignees {
			sb = append(sb, fmt.Sprintf("    - role: %s\n", a["role"])...)
			sb = append(sb, fmt.Sprintf("      who: %s\n", a["who"])...)
		}
	}
	sb = append(sb, "---\n\n"...)
	sb = append(sb, body+"\n"...)
	return string(sb)
}

// readArtifactFMFromDisk reads a project-relative path from the test env's project root
// and parses it as an artifact, returning the Frontmatter.
func readArtifactFMFromDisk(t *testing.T, env *testEnv, relPath string) artifact.Frontmatter {
	t.Helper()
	absPath := filepath.Join(env.projectRoot, relPath)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("readArtifactFMFromDisk: %v", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("readArtifactFMFromDisk stat: %v", err)
	}
	a := artifact.Parse(raw, relPath, info.ModTime())
	return a.FM
}

// TestPutArtifact_AssigneesRoundTrip verifies that PUT with assignees correctly persists
// them in the file's YAML frontmatter and returns them on subsequent GET.
// Covers Milestone 2, scenario 1.
func TestPutArtifact_AssigneesRoundTrip(t *testing.T) {
	const relPath = "lifecycle/ideas/assign-roundtrip.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Assign Roundtrip", "idea", "draft", "assign-roundtrip", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// PUT with one assignee.
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Assign Roundtrip",
			"type":    "idea",
			"status":  "draft",
			"lineage": "assign-roundtrip",
			"assignees": []map[string]string{
				{"role": "backend-developer", "who": "agent"},
			},
		},
		"body": "Updated body.",
	})
	requireStatus(t, putResp, 200)
	putData := readJSON(t, putResp)

	// Verify the PUT API response includes the assignees.
	art, _ := putData["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	assignees, _ := fm["assignees"].([]any)
	if len(assignees) != 1 {
		t.Fatalf("PUT response: expected 1 assignee, got %d", len(assignees))
	}
	first, _ := assignees[0].(map[string]any)
	if role, _ := first["role"].(string); role != "backend-developer" {
		t.Errorf("PUT response: expected assignee role %q, got %q", "backend-developer", role)
	}
	if who, _ := first["who"].(string); who != "agent" {
		t.Errorf("PUT response: expected assignee who %q, got %q", "agent", who)
	}

	// Verify the on-disk file's frontmatter also contains the assignee.
	diskFM := readArtifactFMFromDisk(t, env, relPath)
	if len(diskFM.Assignees) != 1 {
		t.Fatalf("disk: expected 1 assignee, got %d: %v", len(diskFM.Assignees), diskFM.Assignees)
	}
	if diskFM.Assignees[0].Role != "backend-developer" {
		t.Errorf("disk: expected assignee role %q, got %q", "backend-developer", diskFM.Assignees[0].Role)
	}
	if diskFM.Assignees[0].Who != "agent" {
		t.Errorf("disk: expected assignee who %q, got %q", "agent", diskFM.Assignees[0].Who)
	}

	// Verify GET also returns the assignees.
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)

	getArt, _ := getData["artifact"].(map[string]any)
	getFM, _ := getArt["frontmatter"].(map[string]any)
	getAssignees, _ := getFM["assignees"].([]any)
	if len(getAssignees) != 1 {
		t.Fatalf("GET response: expected 1 assignee, got %d", len(getAssignees))
	}
	getFirst, _ := getAssignees[0].(map[string]any)
	if role, _ := getFirst["role"].(string); role != "backend-developer" {
		t.Errorf("GET response: expected assignee role %q, got %q", "backend-developer", role)
	}
}

// TestPutArtifact_RemoveAssignees verifies that PUT with an empty assignees list
// removes any existing assignees from the file's frontmatter.
// Covers Milestone 2, scenario 2.
func TestPutArtifact_RemoveAssignees(t *testing.T) {
	const relPath = "lifecycle/ideas/assign-remove.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifactWithAssignees(
				"Assign Remove", "idea", "draft", "assign-remove",
				[]map[string]string{{"role": "backend-developer", "who": "agent"}},
				"Body.",
			),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Confirm the seed has an assignee on disk before the PUT.
	diskFMBefore := readArtifactFMFromDisk(t, env, relPath)
	if len(diskFMBefore.Assignees) == 0 {
		t.Fatal("pre-condition: seed artifact should have 1 assignee")
	}

	// PUT with empty assignees to remove them.
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":     "Assign Remove",
			"type":      "idea",
			"status":    "draft",
			"lineage":   "assign-remove",
			"assignees": []any{},
		},
		"body": "Updated body.",
	})
	requireStatus(t, putResp, 200)
	putData := readJSON(t, putResp)

	// API response should have no assignees (omitempty means the key may be absent or empty).
	art, _ := putData["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	assignees, _ := fm["assignees"].([]any)
	if len(assignees) != 0 {
		t.Errorf("PUT response: expected 0 assignees after removal, got %d", len(assignees))
	}

	// On-disk file must no longer contain assignee entries.
	diskFM := readArtifactFMFromDisk(t, env, relPath)
	if len(diskFM.Assignees) != 0 {
		t.Errorf("disk: expected 0 assignees after removal, got %d: %v", len(diskFM.Assignees), diskFM.Assignees)
	}
}

// TestPutArtifact_MultipleAssignees verifies that PUT with two assignees persists both
// in order in both the API response and on-disk file.
// Covers Milestone 2, scenario 3.
func TestPutArtifact_MultipleAssignees(t *testing.T) {
	const relPath = "lifecycle/ideas/assign-multi.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Assign Multi", "idea", "draft", "assign-multi", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Assign Multi",
			"type":    "idea",
			"status":  "draft",
			"lineage": "assign-multi",
			"assignees": []map[string]string{
				{"role": "backend-developer", "who": "agent"},
				{"role": "qa", "who": "human"},
			},
		},
		"body": "Updated body.",
	})
	requireStatus(t, putResp, 200)
	putResp.Body.Close()

	// Verify on-disk file has both assignees in order.
	diskFM := readArtifactFMFromDisk(t, env, relPath)
	if len(diskFM.Assignees) != 2 {
		t.Fatalf("disk: expected 2 assignees, got %d: %v", len(diskFM.Assignees), diskFM.Assignees)
	}
	if diskFM.Assignees[0].Role != "backend-developer" {
		t.Errorf("disk: assignees[0].role: want %q, got %q", "backend-developer", diskFM.Assignees[0].Role)
	}
	if diskFM.Assignees[0].Who != "agent" {
		t.Errorf("disk: assignees[0].who: want %q, got %q", "agent", diskFM.Assignees[0].Who)
	}
	if diskFM.Assignees[1].Role != "qa" {
		t.Errorf("disk: assignees[1].role: want %q, got %q", "qa", diskFM.Assignees[1].Role)
	}
	if diskFM.Assignees[1].Who != "human" {
		t.Errorf("disk: assignees[1].who: want %q, got %q", "human", diskFM.Assignees[1].Who)
	}

	// Verify GET response also contains both assignees.
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)

	getArt, _ := getData["artifact"].(map[string]any)
	getFM, _ := getArt["frontmatter"].(map[string]any)
	getAssignees, _ := getFM["assignees"].([]any)
	if len(getAssignees) != 2 {
		t.Fatalf("GET response: expected 2 assignees, got %d", len(getAssignees))
	}
}

// TestPutArtifact_InvalidRole verifies that PUT with a role not in the project's
// configured roles returns 400 and names the invalid role.
// Covers Milestone 2, scenario 4.
func TestPutArtifact_InvalidRole(t *testing.T) {
	const relPath = "lifecycle/ideas/assign-invalid-role.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Assign Invalid Role", "idea", "draft", "assign-invalid-role", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Assign Invalid Role",
			"type":    "idea",
			"status":  "draft",
			"lineage": "assign-invalid-role",
			"assignees": []map[string]string{
				{"role": "nonexistent-role", "who": "agent"},
			},
		},
		"body": "Body.",
	})
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	code, _ := errData["code"].(string)
	if code != "invalid_role" {
		t.Errorf("expected error code %q, got %q", "invalid_role", code)
	}
	msg, _ := errData["message"].(string)
	if !findSubstring(msg, "nonexistent-role") {
		t.Errorf("expected error message to name the invalid role %q, got: %q", "nonexistent-role", msg)
	}
}

// TestPutArtifact_EmptyRoleOrWho verifies that PUT with an empty role or an empty who
// returns 400 for each case.
// Covers Milestone 2, scenario 5.
func TestPutArtifact_EmptyRoleOrWho(t *testing.T) {
	const relPathA = "lifecycle/ideas/assign-empty-role.md"
	const relPathB = "lifecycle/ideas/assign-empty-who.md"
	seeds := []seedArtifact{
		{
			relPath: relPathA,
			content: makeArtifact("Assign Empty Role", "idea", "draft", "assign-empty-role", "", "Body."),
		},
		{
			relPath: relPathB,
			content: makeArtifact("Assign Empty Who", "idea", "draft", "assign-empty-who", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Case 1: empty role — role="" is not in the configured roles list, expect 400.
	respA := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPathA, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Assign Empty Role",
			"type":    "idea",
			"status":  "draft",
			"lineage": "assign-empty-role",
			"assignees": []map[string]string{
				{"role": "", "who": "agent"},
			},
		},
		"body": "Body.",
	})
	requireStatus(t, respA, 400)
	respA.Body.Close()

	// Case 2: valid role but empty who — expect 400.
	respB := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPathB, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Assign Empty Who",
			"type":    "idea",
			"status":  "draft",
			"lineage": "assign-empty-who",
			"assignees": []map[string]string{
				{"role": "qa", "who": ""},
			},
		},
		"body": "Body.",
	})
	requireStatus(t, respB, 400)
	respB.Body.Close()
}
