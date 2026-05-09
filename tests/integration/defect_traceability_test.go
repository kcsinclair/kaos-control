// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for defect-to-test traceability.
//
// The QA agent is expected to set a related_to field in defect artifacts that
// points back to the test artifact it was run against.  These tests verify:
//   1. Defects created during a QA run against a test artifact carry related_to.
//   2. The related_to field survives an API round-trip (GET → PUT → GET).
//   3. Defects created during a QA run against a non-test artifact do NOT get
//      an auto-injected related_to from the backend.
//
// Test plan: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md §Milestone 5

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// defectTraceabilityCfgYAML is a QA agent config that permits writing to the
// defects stage (needed so the agent's git commit includes defect files).
const defectTraceabilityCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: qa
    role: [qa]
    driver: claude-code-cli
    active_status: in-qa
    allowed_write_paths:
      - lifecycle/defects
      - lifecycle/tests
    git_identity:
      name: QA Agent
      email: qa@test.local
    prompt_templates:
      qa: "Test QA prompt for {target_path} (related test: {related_test})"

  - name: analyst
    role: [analyst]
    driver: claude-code-cli
    active_status: clarifying
    allowed_write_paths:
      - lifecycle/defects
      - lifecycle/requirements
    git_identity:
      name: Analyst Agent
      email: analyst@test.local
    prompt_templates:
      analyst: "Test analyst prompt for {target_path}"
`

// setupFakeClaudeWritingDefect installs a stub claude binary that writes a
// defect markdown file to relativeDefectPath (relative to project root) and
// exits 0.  The defect includes a related_to entry pointing to relatedToPath.
func setupFakeClaudeWritingDefect(t *testing.T, relativeDefectPath, relatedToPath string) {
	t.Helper()
	fakeDir := t.TempDir()

	// Build the script using a heredoc so YAML content is not shell-interpolated.
	var sb bytes.Buffer
	fmt.Fprintf(&sb, "#!/bin/sh\n")
	fmt.Fprintf(&sb, "mkdir -p \"$(dirname '%s')\"\n", relativeDefectPath)
	fmt.Fprintf(&sb, "cat > '%s' << 'DEFECT_EOF'\n", relativeDefectPath)
	sb.WriteString("---\n")
	sb.WriteString("title: Test Defect\n")
	sb.WriteString("type: defect\n")
	sb.WriteString("status: draft\n")
	sb.WriteString("lineage: test-defect-auto\n")
	sb.WriteString("related_to:\n")
	fmt.Fprintf(&sb, "  - %s\n", relatedToPath)
	sb.WriteString("---\n\n")
	sb.WriteString("Defect found during QA run.\n")
	sb.WriteString("DEFECT_EOF\n")
	sb.WriteString("exit 0\n")

	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, sb.Bytes(), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// setupFakeClaudeWritingDefectNoRelated installs a stub claude binary that
// writes a defect WITHOUT a related_to field and exits 0.
func setupFakeClaudeWritingDefectNoRelated(t *testing.T, relativeDefectPath string) {
	t.Helper()
	fakeDir := t.TempDir()

	var sb bytes.Buffer
	fmt.Fprintf(&sb, "#!/bin/sh\n")
	fmt.Fprintf(&sb, "mkdir -p \"$(dirname '%s')\"\n", relativeDefectPath)
	fmt.Fprintf(&sb, "cat > '%s' << 'DEFECT_EOF'\n", relativeDefectPath)
	sb.WriteString("---\n")
	sb.WriteString("title: Defect No Related\n")
	sb.WriteString("type: defect\n")
	sb.WriteString("status: draft\n")
	sb.WriteString("lineage: test-defect-no-related\n")
	sb.WriteString("---\n\n")
	sb.WriteString("Defect with no related_to.\n")
	sb.WriteString("DEFECT_EOF\n")
	sb.WriteString("exit 0\n")

	fakeScript := filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(fakeScript, sb.Bytes(), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// TC 5.1: A defect created by a QA agent run against a test artifact includes
// a related_to field pointing to that test artifact.
func TestDefectTraceability_DefectIncludesRelatedTo(t *testing.T) {
	const testPath = "lifecycle/tests/traceable-test.md"
	const defectPath = "lifecycle/defects/test-defect-auto.md"

	setupFakeClaudeWritingDefect(t, defectPath, testPath)

	env := newAgentTestEnvWithCfg(t, defectTraceabilityCfgYAML, []seedArtifact{{
		relPath: testPath,
		content: makeArtifact("Traceable Test", "test", "approved", "traceable-test", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	runID := startAgentRun(t, env, "qa", testPath)
	waitForRunCompletion(t, env, runID)

	// Verify the defect file was written with related_to.
	absDefectPath := filepath.Join(env.projectRoot, defectPath)
	content, err := os.ReadFile(absDefectPath)
	if err != nil {
		t.Fatalf("defect file not found at %s: %v", absDefectPath, err)
	}

	// related_to must reference the test artifact path.
	if !strings.Contains(string(content), testPath) {
		t.Errorf("defect frontmatter must contain related_to pointing to %q; got:\n%s", testPath, content)
	}

	// Verify via the index that related_to is parsed correctly.
	row, err := env.proj.Idx.Get(defectPath)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("defect artifact not indexed")
	}
	found := false
	for _, rel := range row.FM.Related {
		if rel == testPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("indexed defect must have related_to=%q; got: %v", testPath, row.FM.Related)
	}
}

// TC 5.2: The related_to field in a defect artifact is preserved through an
// API round-trip: GET frontmatter → PUT same frontmatter → GET again.
func TestDefectTraceability_RelatedToPreservedThroughRoundTrip(t *testing.T) {
	const testPath = "lifecycle/tests/roundtrip-test.md"
	const defectPath = "lifecycle/defects/roundtrip-defect.md"

	setupFakeClaudeWritingDefect(t, defectPath, testPath)

	env := newAgentTestEnvWithCfg(t, defectTraceabilityCfgYAML, []seedArtifact{{
		relPath: testPath,
		content: makeArtifact("Round Trip Test", "test", "approved", "roundtrip-test", "", "Test body."),
	}})
	env.login("qa@test.local", "qa-pass-123")

	// Run the QA agent so the defect is written and indexed.
	runID := startAgentRun(t, env, "qa", testPath)
	waitForRunCompletion(t, env, runID)

	// GET the defect from the API.
	env.login("admin@test.local", "admin-pass-123") // admin can read any artifact
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+defectPath, nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)

	art, _ := getData["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	body, _ := getData["body"].(string)
	fileSHA, _ := getData["file_sha"].(string)

	// Verify related_to is present in the GET response.
	rawRelated, _ := fm["related_to"].([]any)
	foundInGet := false
	for _, v := range rawRelated {
		if s, _ := v.(string); s == testPath {
			foundInGet = true
			break
		}
	}
	if !foundInGet {
		t.Fatalf("GET response must contain related_to=%q; got frontmatter: %v", testPath, fm)
	}

	// PUT the artifact back with the same frontmatter and body.
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+defectPath, map[string]any{
		"frontmatter":  fm,
		"body":         body,
		"expected_sha": fileSHA,
	})
	if putResp.StatusCode != 200 {
		b, _ := io.ReadAll(putResp.Body)
		putResp.Body.Close()
		t.Fatalf("PUT expected 200, got %d: %s", putResp.StatusCode, b)
	}
	putResp.Body.Close()

	// GET again and verify related_to is still present.
	getResp2 := env.doRequest("GET", "/api/p/testproject/artifacts/"+defectPath, nil)
	requireStatus(t, getResp2, 200)
	getData2 := readJSON(t, getResp2)

	art2, _ := getData2["artifact"].(map[string]any)
	fm2, _ := art2["frontmatter"].(map[string]any)
	rawRelated2, _ := fm2["related_to"].([]any)

	foundAfterPut := false
	for _, v := range rawRelated2 {
		if s, _ := v.(string); s == testPath {
			foundAfterPut = true
			break
		}
	}
	if !foundAfterPut {
		t.Errorf("related_to must be preserved after PUT round-trip; got frontmatter: %v", fm2)
	}
}

// TC 5.3: A defect created by a QA agent run against a NON-test artifact does
// not have related_to auto-injected by the backend.  Only whatever the agent
// itself writes is present.
func TestDefectTraceability_NonTestArtifactNoAutoInjection(t *testing.T) {
	const reqPath = "lifecycle/requirements/non-test-req-2.md"
	const defectPath = "lifecycle/defects/no-related-defect.md"

	// Fake claude writes a defect WITHOUT related_to.
	setupFakeClaudeWritingDefectNoRelated(t, defectPath)

	env := newAgentTestEnvWithCfg(t, defectTraceabilityCfgYAML, []seedArtifact{{
		relPath: reqPath,
		content: makeArtifact("Non-Test Req", "requirement", "clarifying", "non-test-req", "", "Req body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	// Use the analyst agent (active_status=clarifying, writes to lifecycle/defects).
	runID := startAgentRun(t, env, "analyst", reqPath)
	waitForRunCompletion(t, env, runID)

	// Defect file must exist.
	absDefectPath := filepath.Join(env.projectRoot, defectPath)
	content, err := os.ReadFile(absDefectPath)
	if err != nil {
		t.Fatalf("defect file not found at %s: %v", absDefectPath, err)
	}

	// related_to must NOT have been auto-injected by the backend.
	if strings.Contains(string(content), "related_to:") {
		t.Errorf("backend must NOT auto-inject related_to for non-test artifact defects; got:\n%s", content)
	}

	// Verify via index as well.
	row, err := env.proj.Idx.Get(defectPath)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("defect artifact not indexed")
	}
	if len(row.FM.Related) > 0 {
		t.Errorf("indexed defect must have empty related_to for non-test runs; got: %v", row.FM.Related)
	}
}
