// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architecture-overview-view-5-test.md —
// Milestone T2 (verifies M-B2, M-B3): GET /api/p/{project}/architecture/overview
// through a live server — populated + degraded response shape, auth-but-not-
// editor parity, the read-only guarantee, and on-disk freshness with no
// manual rebuild. Also folds in the NFR-2/NFR-3/NFR-6 assertions from
// Milestone T6 that are integration-testable (NFR-1 is a diff/CI check;
// NFR-4 and the remainder of NFR-5 belong to the frontend Vitest suites in
// Milestone T3).
//
// internal/architecture/overview_test.go and
// internal/http/architecture_overview_test.go already cover LoadOverview and
// the handler in isolation. These tests exercise the same contract end to
// end through newTestEnv's running server and real project fixtures, per the
// plan.
//
// Unlike /architecture-map (which reads through the SQLite index),
// LoadOverview enumerates lifecycle/architecture/** directly off disk on
// every call (see internal/architecture/overview.go: loadRoleDir,
// currentlyPromoted both os.ReadDir the project root). So, unlike
// architecture_zones_test.go's finding for index-backed listing, pre-boot
// seeds under lifecycle/architecture/ DO appear in the overview immediately
// — no watcher wait is needed to observe freshness at this endpoint.

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	ovArchCandidatePath  = "lifecycle/architecture/architectures/ov-candidate.md"
	ovStackCandidatePath = "lifecycle/architecture/tech-stacks/ov-candidate-stack.md"
	ovArchToPromotePath  = "lifecycle/architecture/architectures/ov-chosen.md"
	ovStackToPromotePath = "lifecycle/architecture/tech-stacks/ov-chosen-stack.md"
	ovSummaryPath        = "lifecycle/architecture/architecture-summary.md"
	ovStandardPath       = "lifecycle/architecture/standards/ov-standard.md"
	ovArchivePath        = "lifecycle/architecture/archive/ov-archived.md"
)

func ovOverviewItems(t *testing.T, data map[string]any, key string) []any {
	t.Helper()
	items, ok := data[key].([]any)
	if !ok {
		t.Fatalf("expected %q to be a JSON array, got %T: %v", key, data[key], data[key])
	}
	return items
}

func ovFindByPath(items []any, path string) map[string]any {
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item["path"] == path {
			return item
		}
	}
	return nil
}

// TestArchitectureOverview_PopulatedModel_ClassifiesAndOrders seeds and
// promotes a full architecture zone (catalog candidates, a promoted
// architecture + stack, a summary, a standard, two ADRs, and an archived
// file) and asserts every item comes back with the correct catalog_role
// (FR-9), the promoted root is never classified as catalog, and ADRs are
// strictly newest-first (FR-7). It also pins NFR-3 (prompt response) with a
// generous bound.
func TestArchitectureOverview_PopulatedModel_ClassifiesAndOrders(t *testing.T) {
	env := newTestEnv(t, []seedArtifact{
		{relPath: ovArchCandidatePath, content: makeCleanSlugArtifact("OV Candidate", "architecture", "draft", "Body.")},
		{relPath: ovStackCandidatePath, content: makeCleanSlugArtifact("OV Candidate Stack", "tech-stack", "draft", "Body.")},
		{relPath: ovArchToPromotePath, content: makeCleanSlugArtifact("OV Chosen", "architecture", "draft", "Body.")},
		{relPath: ovStackToPromotePath, content: makeCleanSlugArtifact("OV Chosen Stack", "tech-stack", "draft", "Body.")},
		{relPath: ovSummaryPath, content: makeCleanSlugArtifact("Architecture Summary", "doc", "draft", "Summary body.")},
		{relPath: ovStandardPath, content: makeCleanSlugArtifact("OV Standard", "doc", "draft", "Standard body.")},
		{relPath: ovArchivePath, content: makeCleanSlugArtifact("OV Archived", "architecture", "approved", "Superseded body.")},
	})

	// Promote through the real endpoint so this test also pins the B3
	// regression concern: promotion continues to work and is immediately
	// visible through the overview (no separate re-index step needed here
	// beyond what handlePromoteArchitecture already does).
	promoteResp := env.doRequest("POST", "/api/p/testproject/architecture/promote", map[string]string{
		"architecture_path": "architectures/ov-chosen.md",
		"tech_stack_path":   "tech-stacks/ov-chosen-stack.md",
	})
	requireStatus(t, promoteResp, http.StatusOK)
	promoteResp.Body.Close()

	// Create two ADRs through the real endpoint (ascending creation order);
	// the overview must return them newest-number-first.
	adr1 := createADR(t, env, "ov-first", "OV First Decision", "", "Because reasons.")
	adr2 := createADR(t, env, "ov-second", "OV Second Decision", "", "Because more reasons.")
	adr1Path, _ := adr1["path"].(string)
	adr2Path, _ := adr2["path"].(string)

	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/architecture/overview", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("NFR-3: overview took %s for a small curated fixture, expected well under 2s", elapsed)
	}

	wantPromotedArch := "lifecycle/architecture/ov-chosen.md"
	wantPromotedStack := "lifecycle/architecture/ov-chosen-stack.md"

	if data["has_chosen_architecture"] != true {
		t.Errorf("has_chosen_architecture = %v, want true", data["has_chosen_architecture"])
	}
	chosenArch, _ := data["chosen_architecture"].(map[string]any)
	if chosenArch == nil || chosenArch["path"] != wantPromotedArch {
		t.Fatalf("chosen_architecture = %v, want path %q", data["chosen_architecture"], wantPromotedArch)
	}
	if chosenArch["catalog_role"] != "chosen-architecture" {
		t.Errorf("chosen_architecture.catalog_role = %v, want chosen-architecture", chosenArch["catalog_role"])
	}
	chosenStack, _ := data["chosen_stack"].(map[string]any)
	if chosenStack == nil || chosenStack["path"] != wantPromotedStack {
		t.Fatalf("chosen_stack = %v, want path %q", data["chosen_stack"], wantPromotedStack)
	}
	if chosenStack["catalog_role"] != "chosen-stack" {
		t.Errorf("chosen_stack.catalog_role = %v, want chosen-stack", chosenStack["catalog_role"])
	}

	summary, _ := data["summary"].(map[string]any)
	if summary == nil || summary["path"] != ovSummaryPath || summary["catalog_role"] != "summary" {
		t.Errorf("summary = %v, want path %q with catalog_role summary", data["summary"], ovSummaryPath)
	}

	standards := ovOverviewItems(t, data, "standards")
	standard := ovFindByPath(standards, ovStandardPath)
	if standard == nil || standard["catalog_role"] != "standard" {
		t.Errorf("expected %q in standards with catalog_role standard, got: %v", ovStandardPath, standards)
	}

	archive := ovOverviewItems(t, data, "archive")
	archived := ovFindByPath(archive, ovArchivePath)
	if archived == nil || archived["catalog_role"] != "archive" {
		t.Errorf("expected %q in archive with catalog_role archive, got: %v", ovArchivePath, archive)
	}

	// Catalog: both the never-promoted candidate pair AND the promoted
	// pair's untouched catalog source files (promotion never mutates or
	// removes the catalog source) must classify as catalog, never chosen-*.
	catalog := ovOverviewItems(t, data, "catalog")
	for _, path := range []string{ovArchCandidatePath, ovStackCandidatePath, ovArchToPromotePath, ovStackToPromotePath} {
		item := ovFindByPath(catalog, path)
		if item == nil {
			t.Errorf("expected %q in catalog, got: %v", path, catalog)
			continue
		}
		if item["catalog_role"] != "catalog" {
			t.Errorf("%s.catalog_role = %v, want catalog", path, item["catalog_role"])
		}
	}
	if item := ovFindByPath(catalog, wantPromotedArch); item != nil {
		t.Errorf("promoted root %q must never classify as catalog, got: %v", wantPromotedArch, item)
	}
	if item := ovFindByPath(catalog, wantPromotedStack); item != nil {
		t.Errorf("promoted root %q must never classify as catalog, got: %v", wantPromotedStack, item)
	}

	// ADRs newest-first (FR-7): adr2 (0002) before adr1 (0001).
	adrs := ovOverviewItems(t, data, "adrs")
	if len(adrs) != 2 {
		t.Fatalf("expected 2 adrs, got %d: %v", len(adrs), adrs)
	}
	first, _ := adrs[0].(map[string]any)
	second, _ := adrs[1].(map[string]any)
	if first["path"] != adr2Path {
		t.Errorf("adrs[0].path = %v, want newest %q", first["path"], adr2Path)
	}
	if second["path"] != adr1Path {
		t.Errorf("adrs[1].path = %v, want oldest %q", second["path"], adr1Path)
	}
	for _, adr := range adrs {
		item, _ := adr.(map[string]any)
		if item["catalog_role"] != "adr" {
			t.Errorf("adr %v catalog_role = %v, want adr", item["path"], item["catalog_role"])
		}
	}
}

// TestArchitectureOverview_DegradedEmptyModel_Never500 verifies FR-10/NFR-5:
// with nothing promoted and no summary/standards/ADRs, the endpoint still
// returns 200 with has_chosen_architecture=false, a null summary, and empty
// (non-nil — present as `[]`, not `null`) slices for standards/adrs/archive/
// catalog.
func TestArchitectureOverview_DegradedEmptyModel_Never500(t *testing.T) {
	env := newTestEnv(t, nil)

	resp := env.doRequest("GET", "/api/p/testproject/architecture/overview", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	if data["has_chosen_architecture"] != false {
		t.Errorf("has_chosen_architecture = %v, want false", data["has_chosen_architecture"])
	}
	if data["chosen_architecture"] != nil {
		t.Errorf("chosen_architecture = %v, want null", data["chosen_architecture"])
	}
	if data["chosen_stack"] != nil {
		t.Errorf("chosen_stack = %v, want null", data["chosen_stack"])
	}
	if data["summary"] != nil {
		t.Errorf("summary = %v, want null", data["summary"])
	}
	for _, key := range []string{"standards", "adrs", "archive", "catalog"} {
		items := ovOverviewItems(t, data, key) // fails the test if it decoded as null, not []
		if len(items) != 0 {
			t.Errorf("%s = %v, want empty", key, items)
		}
	}
}

// TestArchitectureOverview_RequiresAuth mirrors
// TestArchitectureMap_RequiresAuth: an unauthenticated request is rejected
// with 401, not served.
func TestArchitectureOverview_RequiresAuth(t *testing.T) {
	env := newTestEnv(t, nil)
	env.logout()

	resp := env.doRequest("GET", "/api/p/testproject/architecture/overview", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request to /architecture/overview, got %d", resp.StatusCode)
	}
}

// reviewerOnlyCfgYAML mirrors defaultCfgYAML but gives qa@test.local only
// the "reviewer" role — not a member of RolesArtifactEditors — so it can
// authenticate as a genuinely non-editor user while reusing the fixed
// qa@test.local/qa-pass-123 auth-store credentials.
const reviewerOnlyCfgYAML = `git:
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
    roles: [product-owner, analyst]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [reviewer]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// TestArchitectureOverview_NoEditorRoleRequired verifies NFR-2/the plan's
// role-parity claim: unlike the mutating architecture endpoints
// (POST .../promote, POST .../adrs, both gated on RolesArtifactEditors), GET
// .../architecture/overview has no role gate at all — an authenticated user
// holding only a non-editor role (reviewer) still gets the model, while the
// same user is forbidden from the mutating endpoints.
func TestArchitectureOverview_NoEditorRoleRequired(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, reviewerOnlyCfgYAML)
	env.login("qa@test.local", "qa-pass-123") // now holds only "reviewer" per reviewerOnlyCfgYAML

	overviewResp := env.doRequest("GET", "/api/p/testproject/architecture/overview", nil)
	requireStatus(t, overviewResp, http.StatusOK)
	overviewResp.Body.Close()

	promoteResp := env.doRequest("POST", "/api/p/testproject/architecture/promote", map[string]string{
		"architecture_path": "architectures/does-not-exist.md",
		"tech_stack_path":   "tech-stacks/does-not-exist.md",
	})
	defer promoteResp.Body.Close()
	if promoteResp.StatusCode != http.StatusForbidden {
		t.Errorf("reviewer-only POST /architecture/promote = %d, want 403 (role gate should reject before path validation)", promoteResp.StatusCode)
	}

	adrResp := env.doRequest("POST", "/api/p/testproject/architecture/adrs", map[string]string{
		"slug": "reviewer-attempt", "title": "Reviewer Attempt", "body": "Body.",
	})
	defer adrResp.Body.Close()
	if adrResp.StatusCode != http.StatusForbidden {
		t.Errorf("reviewer-only POST /architecture/adrs = %d, want 403", adrResp.StatusCode)
	}
}

// TestArchitectureOverview_NoWriteSideEffect verifies NFR-2: calling the
// overview endpoint issues no artifact-content write and leaves the index
// unchanged — the seeded architecture file's bytes and mtime are untouched,
// and the artifact count reported by /artifacts is identical before and
// after.
func TestArchitectureOverview_NoWriteSideEffect(t *testing.T) {
	env := newTestEnv(t, []seedArtifact{
		{relPath: ovArchToPromotePath, content: makeCleanSlugArtifact("OV Chosen", "architecture", "draft", "Body.")},
	})

	absPath := filepath.Join(env.projectRoot, filepath.FromSlash(ovArchToPromotePath))
	before, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	statBefore, err := os.Stat(absPath)
	if err != nil {
		t.Fatal(err)
	}

	countResp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
	requireStatus(t, countResp, http.StatusOK)
	countBefore := readJSON(t, countResp)
	itemsBefore, _ := countBefore["items"].([]any)

	for i := 0; i < 3; i++ {
		resp := env.doRequest("GET", "/api/p/testproject/architecture/overview", nil)
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	}

	after, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("architecture file bytes changed after GET /architecture/overview:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	statAfter, err := os.Stat(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if statBefore.ModTime() != statAfter.ModTime() {
		t.Errorf("architecture file mtime changed after GET /architecture/overview: %v -> %v", statBefore.ModTime(), statAfter.ModTime())
	}

	countResp2 := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
	requireStatus(t, countResp2, http.StatusOK)
	countAfter := readJSON(t, countResp2)
	itemsAfter, _ := countAfter["items"].([]any)
	if len(itemsBefore) != len(itemsAfter) {
		t.Errorf("/artifacts item count changed after GET /architecture/overview: %d -> %d", len(itemsBefore), len(itemsAfter))
	}
}

// TestArchitectureOverview_DiskChangeReflectedWithoutRebuild verifies FR-12
// at the model layer (per backend plan Milestone B3): adding a standard on
// disk after boot is visible on the very next request — no watcher wait, no
// manual rebuild — because LoadOverview enumerates lifecycle/architecture/**
// directly off disk on every call (see the package doc comment above).
func TestArchitectureOverview_DiskChangeReflectedWithoutRebuild(t *testing.T) {
	env := newTestEnv(t, nil)

	before := readJSON(t, env.doRequest("GET", "/api/p/testproject/architecture/overview", nil))
	if items := ovOverviewItems(t, before, "standards"); len(items) != 0 {
		t.Fatalf("expected no standards before the disk write, got: %v", items)
	}

	standardPath := "lifecycle/architecture/standards/new-standard.md"
	absPath := filepath.Join(env.projectRoot, filepath.FromSlash(standardPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(makeCleanSlugArtifact("New Standard", "doc", "draft", "Body.")), 0o644); err != nil {
		t.Fatal(err)
	}

	after := readJSON(t, env.doRequest("GET", "/api/p/testproject/architecture/overview", nil))
	items := ovOverviewItems(t, after, "standards")
	if ovFindByPath(items, standardPath) == nil {
		t.Fatalf("expected %q to appear in standards on the next request with no rebuild, got: %v", standardPath, items)
	}

	if err := os.Remove(absPath); err != nil {
		t.Fatal(err)
	}
	afterRemoval := readJSON(t, env.doRequest("GET", "/api/p/testproject/architecture/overview", nil))
	itemsAfterRemoval := ovOverviewItems(t, afterRemoval, "standards")
	if ovFindByPath(itemsAfterRemoval, standardPath) != nil {
		t.Errorf("expected %q to disappear from standards after removal, got: %v", standardPath, itemsAfterRemoval)
	}
}
