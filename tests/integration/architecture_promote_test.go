// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architectural-artefacts-5-test.md — Milestone 3
// (FR-4–FR-7, OQ-3, NFR-3): POST /architecture/promote — first promotion,
// idempotent re-promotion, changed-choice archive-on-replace, traversal
// rejection, and the role gate.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// promoteFixtureSeeds seeds two catalog architectures ("foo", "baz") and one
// catalog tech-stack ("stack") under lifecycle/architecture/, mirroring the
// package-level fixture in internal/architecture/promote_test.go.
func promoteFixtureSeeds() []seedArtifact {
	return []seedArtifact{
		{
			relPath: "lifecycle/architecture/architectures/foo.md",
			content: makeCleanSlugArtifact("Foo Architecture", "architecture", "draft", "Body."),
		},
		{
			relPath: "lifecycle/architecture/architectures/baz.md",
			content: makeCleanSlugArtifact("Baz Architecture", "architecture", "draft", "Body."),
		},
		{
			relPath: "lifecycle/architecture/tech-stacks/stack.md",
			content: makeCleanSlugArtifact("Stack", "tech-stack", "draft", "Body."),
		},
	}
}

func promoteReq(archPath, stackPath string) map[string]string {
	return map[string]string{
		"architecture_path": archPath,
		"tech_stack_path":   stackPath,
	}
}

// TestPromote_FirstPromotion_CreatesRootCopiesWithParent covers the first
// promotion sub-case: two root copies appear, each with parent: pointing at
// its catalog source, and both are retrievable via the artifacts API (proving
// the API-write re-index path fired, NFR-2).
func TestPromote_FirstPromotion_CreatesRootCopiesWithParent(t *testing.T) {
	env := newTestEnv(t, promoteFixtureSeeds())

	resp := env.doRequest("POST", "/api/p/testproject/architecture/promote",
		promoteReq("architectures/foo.md", "tech-stacks/stack.md"))
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	wantArch := "lifecycle/architecture/foo.md"
	wantStack := "lifecycle/architecture/stack.md"
	if got, _ := data["promoted_architecture"].(string); got != wantArch {
		t.Errorf("promoted_architecture = %q, want %q", got, wantArch)
	}
	if got, _ := data["promoted_tech_stack"].(string); got != wantStack {
		t.Errorf("promoted_tech_stack = %q, want %q", got, wantStack)
	}
	if archived, _ := data["archived"].([]any); len(archived) != 0 {
		t.Errorf("expected no archived files on first promotion, got %v", archived)
	}

	// Root copies exist on disk with parent: stamped.
	archContent := string(readFileMust(t, filepath.Join(env.projectRoot, filepath.FromSlash(wantArch))))
	if !strings.Contains(archContent, "parent: lifecycle/architecture/architectures/foo.md") {
		t.Errorf("promoted architecture missing parent stamp:\n%s", archContent)
	}
	stackContent := string(readFileMust(t, filepath.Join(env.projectRoot, filepath.FromSlash(wantStack))))
	if !strings.Contains(stackContent, "parent: lifecycle/architecture/tech-stacks/stack.md") {
		t.Errorf("promoted tech-stack missing parent stamp:\n%s", stackContent)
	}

	// Both promoted copies (and their resolved parent edges) are visible via the API.
	listResp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
	requireStatus(t, listResp, 200)
	listData := readJSON(t, listResp)
	if findArtifactRow(t, listData, wantArch) == nil {
		t.Errorf("expected %q in /artifacts after promotion", wantArch)
	}
	if findArtifactRow(t, listData, wantStack) == nil {
		t.Errorf("expected %q in /artifacts after promotion", wantStack)
	}

	graphData := graphResponseForProject(t, env)
	edges := decodeGraphEdges(t, graphData)
	assertParentEdge(t, edges, wantArch, "lifecycle/architecture/architectures/foo.md")
	assertParentEdge(t, edges, wantStack, "lifecycle/architecture/tech-stacks/stack.md")
}

// TestPromote_IdempotentSameSelection_NoDuplicateNoArchive covers re-posting
// the identical selection: exactly one root copy per kind, and archive/ is
// not created.
func TestPromote_IdempotentSameSelection_NoDuplicateNoArchive(t *testing.T) {
	env := newTestEnv(t, promoteFixtureSeeds())
	req := promoteReq("architectures/foo.md", "tech-stacks/stack.md")

	resp1 := env.doRequest("POST", "/api/p/testproject/architecture/promote", req)
	requireStatus(t, resp1, 200)
	resp1.Body.Close()

	resp2 := env.doRequest("POST", "/api/p/testproject/architecture/promote", req)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	if archived, _ := data2["archived"].([]any); len(archived) != 0 {
		t.Errorf("expected no archived files on idempotent re-promote, got %v", archived)
	}

	entries, err := os.ReadDir(filepath.Join(env.projectRoot, "lifecycle", "architecture"))
	if err != nil {
		t.Fatal(err)
	}
	var mdFiles []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) != 2 {
		t.Errorf("expected exactly 2 root .md files after re-promoting the same selection, got %v", mdFiles)
	}

	if _, err := os.Stat(filepath.Join(env.projectRoot, "lifecycle", "architecture", "archive")); !os.IsNotExist(err) {
		t.Errorf("archive/ should not be created for an idempotent re-promote, stat err=%v", err)
	}
}

// TestPromote_ChangedSelection_ArchivesPriorCopy covers promoting a different
// architecture than the currently-promoted one: the prior root copy moves to
// archive/, the new copy lands at the root, and the catalog is untouched.
func TestPromote_ChangedSelection_ArchivesPriorCopy(t *testing.T) {
	env := newTestEnv(t, promoteFixtureSeeds())

	resp1 := env.doRequest("POST", "/api/p/testproject/architecture/promote",
		promoteReq("architectures/foo.md", "tech-stacks/stack.md"))
	requireStatus(t, resp1, 200)
	resp1.Body.Close()

	resp2 := env.doRequest("POST", "/api/p/testproject/architecture/promote",
		promoteReq("architectures/baz.md", "tech-stacks/stack.md"))
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)

	archived, _ := data2["archived"].([]any)
	if len(archived) != 1 || archived[0] != "lifecycle/architecture/archive/foo.md" {
		t.Fatalf("archived = %v, want [\"lifecycle/architecture/archive/foo.md\"]", archived)
	}

	if _, err := os.Stat(filepath.Join(env.projectRoot, "lifecycle", "architecture", "foo.md")); !os.IsNotExist(err) {
		t.Error("prior root copy foo.md should have been moved, not left in place")
	}
	if _, err := os.Stat(filepath.Join(env.projectRoot, "lifecycle", "architecture", "baz.md")); err != nil {
		t.Errorf("new root copy baz.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.projectRoot, "lifecycle", "architecture", "archive", "foo.md")); err != nil {
		t.Errorf("archived copy missing: %v", err)
	}

	// Catalog untouched.
	if _, err := os.Stat(filepath.Join(env.projectRoot, "lifecycle", "architecture", "architectures", "foo.md")); err != nil {
		t.Errorf("catalog source foo.md should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.projectRoot, "lifecycle", "architecture", "architectures", "baz.md")); err != nil {
		t.Errorf("catalog source baz.md should remain: %v", err)
	}
}

// TestPromote_TwoArchivedGenerations_CoexistWithDisambiguator drives foo/baz
// back and forth through the same stack so the second archive attempt for
// "foo" collides with the first and must pick up the -1 disambiguator rather
// than overwriting archive/foo.md (NFR-3 — never overwrite archived history).
func TestPromote_TwoArchivedGenerations_CoexistWithDisambiguator(t *testing.T) {
	env := newTestEnv(t, promoteFixtureSeeds())

	promote := func(archSlug string) {
		t.Helper()
		resp := env.doRequest("POST", "/api/p/testproject/architecture/promote",
			promoteReq("architectures/"+archSlug+".md", "tech-stacks/stack.md"))
		requireStatus(t, resp, 200)
		resp.Body.Close()
	}

	promote("foo") // root: foo
	promote("baz") // archives foo -> archive/foo.md; root: baz
	promote("foo") // archives baz -> archive/baz.md; root: foo again
	promote("baz") // archives foo again -> collides, gets -1; root: baz again

	archDir := filepath.Join(env.projectRoot, "lifecycle", "architecture", "archive")
	if _, err := os.Stat(filepath.Join(archDir, "foo.md")); err != nil {
		t.Errorf("expected first archived generation to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(archDir, "foo-1.md")); err != nil {
		t.Errorf("expected second archived generation with -1 disambiguator to exist: %v", err)
	}
}

// TestPromote_TraversalPath_Returns400 covers the traversal guard.
func TestPromote_TraversalPath_Returns400(t *testing.T) {
	env := newTestEnv(t, promoteFixtureSeeds())

	resp := env.doRequest("POST", "/api/p/testproject/architecture/promote",
		promoteReq("../../etc/passwd", "tech-stacks/stack.md"))
	requireStatus(t, resp, 400)
}

// architectureRoleGateCfgYAML is like defaultCfgYAML but demotes qa@test.local
// to a role outside RolesArtifactEditors (reviewer only), so it can be used
// to exercise the 403 role-gate path on the architecture endpoints.
const architectureRoleGateCfgYAML = `git:
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
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [reviewer]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// TestPromote_ReadOnlyUser_Returns403 covers the role gate: a user holding
// only a role outside RolesArtifactEditors (reviewer) must be denied.
func TestPromote_ReadOnlyUser_Returns403(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, promoteFixtureSeeds(), architectureRoleGateCfgYAML)
	env.login("qa@test.local", "qa-pass-123") // qa@test.local holds only "reviewer" under this config

	resp := env.doRequest("POST", "/api/p/testproject/architecture/promote",
		promoteReq("architectures/foo.md", "tech-stacks/stack.md"))
	requireStatus(t, resp, 403)
}

func readFileMust(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// assertParentEdge fails the test unless a "parent" kind edge from src to dst
// is present in the graph edges slice.
func assertParentEdge(t *testing.T, edges []any, src, dst string) {
	t.Helper()
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		s, _ := edge["source"].(string)
		d, _ := edge["target"].(string)
		k := graphEdgeKind(edge)
		if s == src && d == dst && k == "parent" {
			return
		}
	}
	t.Errorf("expected a parent-kind edge from %q to %q, not found in graph response", src, dst)
}
