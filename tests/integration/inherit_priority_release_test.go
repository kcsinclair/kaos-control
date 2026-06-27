// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Package integration: tests for inherit-priority-and-release feature.
//
// Test plan: lifecycle/test-plans/inherit-priority-and-release-5-test.md
// Milestones covered: 2 (POST /artifacts), 3 (generate handler), 4 (rejection),
// 5 (override isolation / no-migration), 6 (cross-path consistency).

package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeArtifactFull builds a markdown artifact with optional priority and release
// fields. Pass empty strings to omit those fields.
func makeArtifactFull(title, typ, status, lineage, parent, priority, release, body string) string {
	var sb bytes.Buffer
	sb.WriteString("---\n")
	sb.WriteString("title: " + title + "\n")
	sb.WriteString("type: " + typ + "\n")
	sb.WriteString("status: " + status + "\n")
	sb.WriteString("lineage: " + lineage + "\n")
	if parent != "" {
		sb.WriteString("parent: " + parent + "\n")
	}
	if priority != "" {
		sb.WriteString("priority: " + priority + "\n")
	}
	if release != "" {
		sb.WriteString("release: " + release + "\n")
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body + "\n")
	return sb.String()
}

// ── Milestone 2: Manual creation path (POST /artifacts) ──────────────────────

// TestInherit_Create_InheritsPriorityFromParent verifies that creating an
// artifact with a parent that has priority: high and omitting priority in the
// request results in the child inheriting priority: high (FR-2).
func TestInherit_Create_InheritsPriorityFromParent(t *testing.T) {
	parentContent := makeArtifactFull("Parent Idea", "idea", "draft", "inh-prio-parent",
		"", "high", "", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/inh-prio-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "inh-prio-child",
		"frontmatter": map[string]any{
			"title":   "Inherit Priority Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "inh-prio-child",
			"parent":  "lifecycle/ideas/inh-prio-parent.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatalf("reading child artifact: %v", err)
	}
	if !strings.Contains(string(raw), "priority: high") {
		t.Errorf("child artifact missing inherited 'priority: high':\n%s", string(raw))
	}
}

// TestInherit_Create_InheritsReleaseFromParent verifies that creating an
// artifact with a parent that has release: KC-Release4 and omitting release in
// the request results in the child inheriting release: KC-Release4 (FR-3).
func TestInherit_Create_InheritsReleaseFromParent(t *testing.T) {
	parentContent := makeArtifactFull("Parent Idea", "idea", "draft", "inh-rel-parent",
		"", "", "KC-Release4", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/inh-rel-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "inh-rel-child",
		"frontmatter": map[string]any{
			"title":   "Inherit Release Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "inh-rel-child",
			"parent":  "lifecycle/ideas/inh-rel-parent.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatalf("reading child artifact: %v", err)
	}
	if !strings.Contains(string(raw), "release: KC-Release4") {
		t.Errorf("child artifact missing inherited 'release: KC-Release4':\n%s", string(raw))
	}
}

// TestInherit_Create_ExplicitValuesWin verifies that explicitly supplied
// priority and release are preserved even when the parent has different values
// (FR-4 / NFR-1).
func TestInherit_Create_ExplicitValuesWin(t *testing.T) {
	parentContent := makeArtifactFull("Parent Idea", "idea", "draft", "inh-explicit-parent",
		"", "high", "parent-release", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/inh-explicit-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "inh-explicit-child",
		"frontmatter": map[string]any{
			"title":    "Explicit Values Child",
			"type":     "requirement",
			"status":   "draft",
			"lineage":  "inh-explicit-child",
			"parent":   "lifecycle/ideas/inh-explicit-parent.md",
			"priority": "low",
			"release":  "child-release",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatalf("reading child artifact: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "priority: low") {
		t.Errorf("explicit priority 'low' not preserved; parent had 'high':\n%s", content)
	}
	if !strings.Contains(content, "release: child-release") {
		t.Errorf("explicit release 'child-release' not preserved; parent had 'parent-release':\n%s", content)
	}
}

// TestInherit_Create_ParentWithNoFields verifies that when the parent has no
// priority or release, the child receives neither (no fabricated defaults).
func TestInherit_Create_ParentWithNoFields(t *testing.T) {
	parentContent := makeArtifact("Parent No Fields", "idea", "draft", "inh-nofields-parent",
		"", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/inh-nofields-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "inh-nofields-child",
		"frontmatter": map[string]any{
			"title":   "No Fields Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "inh-nofields-child",
			"parent":  "lifecycle/ideas/inh-nofields-parent.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatalf("reading child artifact: %v", err)
	}
	content := string(raw)
	if strings.Contains(content, "priority:") {
		t.Errorf("child should have no priority field when parent has none:\n%s", content)
	}
	if strings.Contains(content, "release:") {
		t.Errorf("child should have no release field when parent has none:\n%s", content)
	}
}

// TestInherit_Create_DanglingParent verifies that a parent pointing to a
// non-existent file produces a 201 Created response without inheritance, and
// does NOT cause the request to fail (FR-5 / NFR-4).
func TestInherit_Create_DanglingParent(t *testing.T) {
	env := newTestEnv(t, nil)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "inh-dangling-child",
		"frontmatter": map[string]any{
			"title":   "Dangling Parent Child",
			"type":    "idea",
			"status":  "draft",
			"lineage": "inh-dangling-child",
			"parent":  "lifecycle/ideas/does-not-exist.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatalf("reading created artifact: %v", err)
	}
	content := string(raw)
	// No priority or release should be injected from a non-existent parent.
	if strings.Contains(content, "priority:") {
		t.Errorf("artifact should have no priority field when parent is dangling:\n%s", content)
	}
	if strings.Contains(content, "release:") {
		t.Errorf("artifact should have no release field when parent is dangling:\n%s", content)
	}
}

// TestInherit_Create_NFR2_InheritedEqualsExplicit verifies that creating an
// artifact via inheritance produces the same frontmatter values as creating one
// with the values supplied explicitly (NFR-2).
func TestInherit_Create_NFR2_InheritedEqualsExplicit(t *testing.T) {
	parentContent := makeArtifactFull("NFR2 Parent", "idea", "draft", "nfr2-parent",
		"", "high", "v-nfr2", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/nfr2-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	// Child A: created with parent → inherits priority and release.
	respA := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "nfr2-inherited",
		"frontmatter": map[string]any{
			"title":   "NFR2 Inherited",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "nfr2-inherited",
			"parent":  "lifecycle/ideas/nfr2-parent.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, respA, 201)
	dataA := readJSON(t, respA)
	pathA, _ := dataA["path"].(string)

	// Child B: created with explicit priority and release, no parent.
	respB := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "nfr2-explicit",
		"frontmatter": map[string]any{
			"title":    "NFR2 Explicit",
			"type":     "requirement",
			"status":   "draft",
			"lineage":  "nfr2-explicit",
			"priority": "high",
			"release":  "v-nfr2",
		},
		"body": "Child body.",
	})
	requireStatus(t, respB, 201)
	dataB := readJSON(t, respB)
	pathB, _ := dataB["path"].(string)

	// Both artifacts must have the same priority and release values on disk.
	rawA, err := os.ReadFile(filepath.Join(env.projectRoot, pathA))
	if err != nil {
		t.Fatalf("reading artifact A: %v", err)
	}
	rawB, err := os.ReadFile(filepath.Join(env.projectRoot, pathB))
	if err != nil {
		t.Fatalf("reading artifact B: %v", err)
	}

	hasA := strings.Contains(string(rawA), "priority: high") && strings.Contains(string(rawA), "release: v-nfr2")
	hasB := strings.Contains(string(rawB), "priority: high") && strings.Contains(string(rawB), "release: v-nfr2")
	if !hasA {
		t.Errorf("artifact A (inherited) missing expected priority/release:\n%s", string(rawA))
	}
	if !hasB {
		t.Errorf("artifact B (explicit) missing expected priority/release:\n%s", string(rawB))
	}
}

// TestInherit_Create_OnlyDirectParentInherited verifies that only the
// direct parent's priority and release are inherited — no recursive lineage
// walk is performed (NFR-3).
func TestInherit_Create_OnlyDirectParentInherited(t *testing.T) {
	// Grandparent has priority: critical, release: grandparent-release.
	// Parent has priority: high (its own value); grandparent's critical is NOT propagated.
	grandparentContent := makeArtifactFull("Grandparent Idea", "idea", "draft", "nfr3-lineage",
		"", "critical", "grandparent-release", "Grandparent body.")
	parentContent := makeArtifactFull("Parent Req", "requirement", "draft", "nfr3-lineage",
		"lifecycle/ideas/nfr3-gp.md", "high", "", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/nfr3-gp.md", content: grandparentContent},
		{relPath: "lifecycle/requirements/nfr3-parent-2.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	// Child specifies the direct parent (nfr3-parent-2.md) and no priority/release.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "backend-plans",
		"slug":  "nfr3-lineage",
		"frontmatter": map[string]any{
			"title":   "Child Plan",
			"type":    "plan-backend",
			"status":  "draft",
			"lineage": "nfr3-lineage",
			"parent":  "lifecycle/requirements/nfr3-parent-2.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatalf("reading child artifact: %v", err)
	}
	content := string(raw)

	// Must inherit priority: high from the direct parent (not critical from grandparent).
	if !strings.Contains(content, "priority: high") {
		t.Errorf("child should inherit direct parent priority 'high', got:\n%s", content)
	}
	if strings.Contains(content, "priority: critical") {
		t.Errorf("child must not inherit grandparent priority 'critical':\n%s", content)
	}
	// Direct parent has no release, grandparent has grandparent-release — child must have none.
	if strings.Contains(content, "release:") {
		t.Errorf("child should have no release (direct parent has none); grandparent release must not be inherited:\n%s", content)
	}
}

// ── Milestone 3: Agent / LLM generation path ──────────────────────────────────

// TestInherit_Generate_InheritsPriority verifies that passing source_path to
// the generate endpoint propagates the source's priority into the preview
// frontmatter (FR-6). Requires live LLM.
func TestInherit_Generate_InheritsPriority(t *testing.T) {
	skipIfNoAPIKey(t)

	parentContent := makeArtifactFull("Gen Source Idea", "idea", "draft", "gen-prio-source",
		"", "high", "", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/gen-prio-source.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{
		"input":       "Add a dark mode toggle so users can reduce eye strain when working in low light environments",
		"source_path": "lifecycle/ideas/gen-prio-source.md",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("generate response missing 'frontmatter'")
	}
	if prio, _ := fm["priority"].(string); prio != "high" {
		t.Errorf("generate frontmatter priority: want %q, got %q", "high", prio)
	}
}

// TestInherit_Generate_InheritsRelease verifies that the source's release
// appears in the generate preview frontmatter (FR-6). Requires live LLM.
func TestInherit_Generate_InheritsRelease(t *testing.T) {
	skipIfNoAPIKey(t)

	parentContent := makeArtifactFull("Gen Release Source", "idea", "draft", "gen-rel-source",
		"", "", "KC-Release4", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/gen-rel-source.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{
		"input":       "Add a dark mode toggle so users can reduce eye strain when working in low light environments",
		"source_path": "lifecycle/ideas/gen-rel-source.md",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("generate response missing 'frontmatter'")
	}
	if rel, _ := fm["release"].(string); rel != "KC-Release4" {
		t.Errorf("generate frontmatter release: want %q, got %q", "KC-Release4", rel)
	}
}

// TestInherit_Generate_ParentlessDefaultPriority verifies that a generate call
// with no source_path results in priority: normal and no release key.
// Requires live LLM.
func TestInherit_Generate_ParentlessDefaultPriority(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnv(t, nil)

	resp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{
		"input": "Add a dark mode toggle so users can reduce eye strain when working in low light environments",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("generate response missing 'frontmatter'")
	}
	if prio, _ := fm["priority"].(string); prio != "normal" {
		t.Errorf("parentless generate priority: want %q, got %q", "normal", prio)
	}
	if _, hasRelease := fm["release"]; hasRelease {
		t.Errorf("parentless generate: release key should be absent, got %v", fm["release"])
	}
}

// TestInherit_Generate_EmptySourcePriorityFallsToNormal verifies that when
// the source artifact has no priority field, generate falls back to "normal"
// (FR-6). Requires live LLM.
func TestInherit_Generate_EmptySourcePriorityFallsToNormal(t *testing.T) {
	skipIfNoAPIKey(t)

	// Parent has no priority or release.
	parentContent := makeArtifact("Gen No Priority Source", "idea", "draft", "gen-noprio-source",
		"", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/gen-noprio-source.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{
		"input":       "Add a dark mode toggle so users can reduce eye strain when working in low light environments",
		"source_path": "lifecycle/ideas/gen-noprio-source.md",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("generate response missing 'frontmatter'")
	}
	if prio, _ := fm["priority"].(string); prio != "normal" {
		t.Errorf("generate with no-priority source: want %q, got %q", "normal", prio)
	}
}

// ── Milestone 4: Workflow rejection path ─────────────────────────────────────

// TestInherit_Rejection_InheritsPriorityAndRelease verifies that transitioning
// an artifact to "rejected" with a comment creates a rejection artifact that
// inherits the source's priority and release (FR-7).
func TestInherit_Rejection_InheritsPriorityAndRelease(t *testing.T) {
	sourceContent := makeArtifactFull("Rejection Source Idea", "idea", "draft", "rej-inh-source",
		"", "high", "rej-release-v1", "Source body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/rej-inh-source.md", content: sourceContent},
	}
	env := newTestEnv(t, seeds)

	const sourcePath = "lifecycle/ideas/rej-inh-source.md"
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+sourcePath+"/transition",
		map[string]any{
			"to":      "rejected",
			"comment": "Does not meet acceptance criteria.",
		})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	rejPath, _ := data["rejection_artifact"].(string)
	if rejPath == "" {
		t.Fatal("transition response missing 'rejection_artifact' path")
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, rejPath))
	if err != nil {
		t.Fatalf("reading rejection artifact at %s: %v", rejPath, err)
	}
	content := string(raw)

	if !strings.Contains(content, "priority: high") {
		t.Errorf("rejection artifact missing inherited 'priority: high':\n%s", content)
	}
	if !strings.Contains(content, "release: rej-release-v1") {
		t.Errorf("rejection artifact missing inherited 'release: rej-release-v1':\n%s", content)
	}
	// The rejection artifact should also have the standard inherited fields.
	if !strings.Contains(content, "lineage: rej-inh-source") {
		t.Errorf("rejection artifact missing 'lineage: rej-inh-source':\n%s", content)
	}
	if !strings.Contains(content, "parent: "+sourcePath) {
		t.Errorf("rejection artifact missing 'parent: %s':\n%s", sourcePath, content)
	}
}

// TestInherit_Rejection_SourceWithNoFields verifies that when the rejected
// source has no priority or release, the rejection artifact gets neither.
func TestInherit_Rejection_SourceWithNoFields(t *testing.T) {
	sourceContent := makeArtifact("Rejection No Fields Source", "idea", "draft",
		"rej-nofields-source", "", "Source body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/rej-nofields-source.md", content: sourceContent},
	}
	env := newTestEnv(t, seeds)

	const sourcePath = "lifecycle/ideas/rej-nofields-source.md"
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+sourcePath+"/transition",
		map[string]any{
			"to":      "rejected",
			"comment": "Does not meet acceptance criteria.",
		})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	rejPath, _ := data["rejection_artifact"].(string)
	if rejPath == "" {
		t.Fatal("transition response missing 'rejection_artifact' path")
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, rejPath))
	if err != nil {
		t.Fatalf("reading rejection artifact: %v", err)
	}
	content := string(raw)

	if strings.Contains(content, "priority:") {
		t.Errorf("rejection artifact should have no priority when source has none:\n%s", content)
	}
	if strings.Contains(content, "release:") {
		t.Errorf("rejection artifact should have no release when source has none:\n%s", content)
	}
}

// ── Milestone 5: Override isolation, validation, and no-migration ─────────────

// TestInherit_Override_PatchPriorityIsolatedToChild verifies that after
// inheritance, PATCHing priority on the child changes only the child file; the
// parent file is byte-unchanged on disk (FR-9).
func TestInherit_Override_PatchPriorityIsolatedToChild(t *testing.T) {
	parentContent := makeArtifactFull("Priority Isolation Parent", "idea", "draft",
		"prio-isol-parent", "", "high", "", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/prio-isol-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	// Create child via inheritance.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "prio-isol-child",
		"frontmatter": map[string]any{
			"title":   "Priority Isolation Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "prio-isol-child",
			"parent":  "lifecycle/ideas/prio-isol-parent.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)
	childPath, _ := data["path"].(string)

	// Verify child inherited priority.
	childRaw, err := os.ReadFile(filepath.Join(env.projectRoot, childPath))
	if err != nil {
		t.Fatalf("reading child: %v", err)
	}
	if !strings.Contains(string(childRaw), "priority: high") {
		t.Fatal("child did not inherit priority: high before patch")
	}

	// Record parent file before PATCH.
	parentBefore, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/prio-isol-parent.md"))
	if err != nil {
		t.Fatalf("reading parent before patch: %v", err)
	}

	// PATCH child priority to "low".
	patchResp := env.doRequest("PATCH",
		"/api/p/testproject/artifacts/"+childPath+"/priority",
		map[string]any{"priority": "low"})
	requireStatus(t, patchResp, 200)
	patchResp.Body.Close()

	// Child must now have priority: low.
	childAfter, err := os.ReadFile(filepath.Join(env.projectRoot, childPath))
	if err != nil {
		t.Fatalf("reading child after patch: %v", err)
	}
	if !strings.Contains(string(childAfter), "priority: low") {
		t.Errorf("child priority not updated to 'low' after patch:\n%s", string(childAfter))
	}

	// Parent must be byte-unchanged.
	parentAfter, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/prio-isol-parent.md"))
	if err != nil {
		t.Fatalf("reading parent after patch: %v", err)
	}
	if !bytes.Equal(parentBefore, parentAfter) {
		t.Errorf("parent file changed after child priority patch:\nbefore: %q\nafter:  %q",
			string(parentBefore), string(parentAfter))
	}
}

// TestInherit_Override_PatchReleaseIsolatedToChild verifies that PATCHing
// release on the child changes only the child; the parent file is unchanged
// on disk (FR-9).
func TestInherit_Override_PatchReleaseIsolatedToChild(t *testing.T) {
	parentContent := makeArtifactFull("Release Isolation Parent", "idea", "draft",
		"rel-isol-parent", "", "", "parent-release", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/rel-isol-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	// Create child via inheritance.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "rel-isol-child",
		"frontmatter": map[string]any{
			"title":   "Release Isolation Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "rel-isol-child",
			"parent":  "lifecycle/ideas/rel-isol-parent.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)
	childPath, _ := data["path"].(string)

	// Verify child inherited release.
	childRaw, err := os.ReadFile(filepath.Join(env.projectRoot, childPath))
	if err != nil {
		t.Fatalf("reading child: %v", err)
	}
	if !strings.Contains(string(childRaw), "release: parent-release") {
		t.Fatal("child did not inherit 'release: parent-release' before patch")
	}

	// Record parent file before PATCH.
	parentBefore, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/rel-isol-parent.md"))
	if err != nil {
		t.Fatalf("reading parent before patch: %v", err)
	}

	// Create the new release in the project so PATCH can validate it.
	createRelease(t, env, map[string]any{"name": "child-updated-release", "status": "planned"})

	// PATCH child release.
	patchResp := patchRelease(env, childPath, strPtr("child-updated-release"))
	requireStatus(t, patchResp, 200)
	patchResp.Body.Close()

	// Child must have updated release.
	childAfter, err := os.ReadFile(filepath.Join(env.projectRoot, childPath))
	if err != nil {
		t.Fatalf("reading child after patch: %v", err)
	}
	if !strings.Contains(string(childAfter), "release: child-updated-release") {
		t.Errorf("child release not updated after patch:\n%s", string(childAfter))
	}

	// Parent must be byte-unchanged.
	parentAfter, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/rel-isol-parent.md"))
	if err != nil {
		t.Fatalf("reading parent after patch: %v", err)
	}
	if !bytes.Equal(parentBefore, parentAfter) {
		t.Errorf("parent file changed after child release patch:\nbefore: %q\nafter:  %q",
			string(parentBefore), string(parentAfter))
	}
}

// TestInherit_Create_InheritedReleaseSkipsValidation verifies that an
// inherited release that is not in the project release list is accepted at
// creation time without a validation error (FR-11).
func TestInherit_Create_InheritedReleaseSkipsValidation(t *testing.T) {
	// The release name "not-in-release-list" is intentionally not registered
	// in the project; the create endpoint must NOT validate it.
	parentContent := makeArtifactFull("Validation Skip Parent", "idea", "draft",
		"inh-skipval-parent", "", "", "not-in-release-list", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/inh-skipval-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "inh-skipval-child",
		"frontmatter": map[string]any{
			"title":   "Skip Validation Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "inh-skipval-child",
			"parent":  "lifecycle/ideas/inh-skipval-parent.md",
		},
		"body": "Child body.",
	})
	// Must succeed despite the release name not being in the project list.
	requireStatus(t, resp, 201)

	data := readJSON(t, resp)
	path, _ := data["path"].(string)
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatalf("reading child artifact: %v", err)
	}
	if !strings.Contains(string(raw), "release: not-in-release-list") {
		t.Errorf("child should carry inherited release even if not in list:\n%s", string(raw))
	}
}

// TestInherit_Override_PatchReleaseStillValidatesUnknownRelease verifies that
// PATCH /release with a release name not in the project list returns 422 with
// error code invalid_release (existing validation preserved after inheritance).
func TestInherit_Override_PatchReleaseStillValidatesUnknownRelease(t *testing.T) {
	parentContent := makeArtifactFull("Patch Validation Parent", "idea", "draft",
		"patchval-parent", "", "", "v1", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/patchval-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	// Create a child that inherits the release from its parent.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "patchval-child",
		"frontmatter": map[string]any{
			"title":   "Patch Validation Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "patchval-child",
			"parent":  "lifecycle/ideas/patchval-parent.md",
		},
		"body": "Child body.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)
	childPath, _ := data["path"].(string)

	// Now try to PATCH with a release name that does not exist in the list.
	patchResp := patchRelease(env, childPath, strPtr("nonexistent-release-xyz"))
	requireStatus(t, patchResp, 422)
	patchData := readJSON(t, patchResp)

	errObj, _ := patchData["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "invalid_release" {
		t.Errorf("expected error code 'invalid_release', got %q", code)
	}
}

// TestInherit_NoMigration verifies that starting the server against a tree of
// pre-existing artifacts with priority and release fields does not modify any
// file on disk (no migration / no backfill side-effects).
func TestInherit_NoMigration(t *testing.T) {
	// Seed artifacts in various states; some with priority/release, some without.
	ideaContent := makeArtifactFull("No Migration Idea", "idea", "draft",
		"nomig-idea", "", "high", "v1.0", "Idea body.")
	reqContent := makeArtifactFull("No Migration Req", "requirement", "draft",
		"nomig-req", "lifecycle/ideas/nomig-idea.md", "low", "v1.0", "Req body.")
	bareContent := makeArtifact("No Migration Bare", "idea", "approved", "nomig-bare",
		"", "Bare body.")

	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/nomig-idea.md", content: ideaContent},
		{relPath: "lifecycle/requirements/nomig-req-2.md", content: reqContent},
		{relPath: "lifecycle/ideas/nomig-bare.md", content: bareContent},
	}

	env := newTestEnv(t, seeds)

	// Let the indexer stabilise (the watcher debounce is 150 ms).
	time.Sleep(300 * time.Millisecond)

	// All seeded files must be byte-identical to the original content.
	for _, s := range seeds {
		absPath := filepath.Join(env.projectRoot, s.relPath)
		got, err := os.ReadFile(absPath)
		if err != nil {
			t.Fatalf("reading %s after server start: %v", s.relPath, err)
		}
		if string(got) != s.content {
			t.Errorf("file %s was modified on disk by the server:\nwant: %q\n got: %q",
				s.relPath, s.content, string(got))
		}
	}
}

// ── Milestone 6: Single-enforcement-point and cross-path consistency ──────────

// TestInherit_CrossPath_ManualAndRejectionConsistency verifies that the manual
// (POST /artifacts) and rejection (transition to rejected) paths both yield the
// same inherited priority and release for identical parent values (NFR-5 / FR-8).
func TestInherit_CrossPath_ManualAndRejectionConsistency(t *testing.T) {
	// Single parent fixture with known priority and release.
	parentContent := makeArtifactFull("Cross Path Parent", "idea", "draft",
		"cross-parent", "", "high", "cross-release-v1", "Parent body.")
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/cross-parent.md", content: parentContent},
	}
	env := newTestEnv(t, seeds)

	const (
		wantPriority = "high"
		wantRelease  = "cross-release-v1"
	)

	// Path 1 — Manual creation via POST /artifacts.
	createResp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "cross-manual-child",
		"frontmatter": map[string]any{
			"title":   "Cross Path Manual Child",
			"type":    "requirement",
			"status":  "draft",
			"lineage": "cross-manual-child",
			"parent":  "lifecycle/ideas/cross-parent.md",
		},
		"body": "Manual child body.",
	})
	requireStatus(t, createResp, 201)
	createData := readJSON(t, createResp)
	manualPath, _ := createData["path"].(string)

	manualRaw, err := os.ReadFile(filepath.Join(env.projectRoot, manualPath))
	if err != nil {
		t.Fatalf("reading manual child: %v", err)
	}
	manualContent := string(manualRaw)
	if !strings.Contains(manualContent, "priority: "+wantPriority) {
		t.Errorf("[manual] priority: want %q, file:\n%s", wantPriority, manualContent)
	}
	if !strings.Contains(manualContent, "release: "+wantRelease) {
		t.Errorf("[manual] release: want %q, file:\n%s", wantRelease, manualContent)
	}

	// Path 2 — Rejection artifact via transition → rejected.
	transResp := env.doRequest("POST",
		"/api/p/testproject/artifacts/lifecycle/ideas/cross-parent.md/transition",
		map[string]any{
			"to":      "rejected",
			"comment": "Cross-path consistency check.",
		})
	requireStatus(t, transResp, 200)
	transData := readJSON(t, transResp)

	rejPath, _ := transData["rejection_artifact"].(string)
	if rejPath == "" {
		t.Fatal("transition response missing 'rejection_artifact' path")
	}

	rejRaw, err := os.ReadFile(filepath.Join(env.projectRoot, rejPath))
	if err != nil {
		t.Fatalf("reading rejection artifact: %v", err)
	}
	rejContent := string(rejRaw)
	if !strings.Contains(rejContent, "priority: "+wantPriority) {
		t.Errorf("[rejection] priority: want %q, file:\n%s", wantPriority, rejContent)
	}
	if !strings.Contains(rejContent, "release: "+wantRelease) {
		t.Errorf("[rejection] release: want %q, file:\n%s", wantRelease, rejContent)
	}

	// Path 3 — Generate (LLM) path: only exercised when an API key is available.
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Log("[generate] skipping LLM path (ANTHROPIC_API_KEY not set)")
		return
	}

	genResp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{
		"input":       "Add a dark mode toggle so users can reduce eye strain when working in low light environments",
		"source_path": "lifecycle/ideas/cross-parent.md",
	})
	requireStatus(t, genResp, 200)
	genData := readJSON(t, genResp)

	fm, _ := genData["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("[generate] response missing 'frontmatter'")
	}
	if prio, _ := fm["priority"].(string); prio != wantPriority {
		t.Errorf("[generate] priority: want %q, got %q", wantPriority, prio)
	}
	if rel, _ := fm["release"].(string); rel != wantRelease {
		t.Errorf("[generate] release: want %q, got %q", wantRelease, rel)
	}
}
