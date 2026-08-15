// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/onboarding-architecture-selection-5-test.md
// — Milestone 5 (FR-13–FR-16, NFR-1, NFR-2, OQ-5): the wizard's single write
// path, POST /architecture/wizard/commit — first commit, abandon-before-commit,
// idempotent re-commit, changed-selection re-commit (archive + supersede),
// and the product-owner role gate.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWizardCommit_FirstRun_PromotesWritesSummaryAndADR0001 covers
// FR-13/FR-14/FR-15/NFR-2: the first commit promotes both catalog sources to
// the lifecycle/architecture/ root, writes architecture-summary.md and
// adr-0001-*.md (with the Q&A trail), and every one of those lands in
// GET /artifacts (proving the synchronous re-index fired).
func TestWizardCommit_FirstRun_PromotesWritesSummaryAndADR0001(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())

	answers := []map[string]string{wizardAnswer("collaborative", "yes")}
	qa := []map[string]string{{"question": "Will multiple people edit shared data?", "answer": "Yes"}}
	breaking := []map[string]string{{"label": "collaborative", "requirement": "Multiple editors", "mapping": "Modular Monolith supports concurrent access"}}

	resp := env.doRequest("POST", "/api/p/testproject/architecture/wizard/commit",
		wizardCommitReq("architectures/modular-monolith.md", "tech-stacks/go-vue.md", answers, breaking, qa))
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	wantArch := "lifecycle/architecture/modular-monolith.md"
	wantStack := "lifecycle/architecture/go-vue.md"
	if got, _ := data["promoted_architecture"].(string); got != wantArch {
		t.Errorf("promoted_architecture = %q, want %q", got, wantArch)
	}
	if got, _ := data["promoted_tech_stack"].(string); got != wantStack {
		t.Errorf("promoted_tech_stack = %q, want %q", got, wantStack)
	}
	if archived, _ := data["archived"].([]any); len(archived) != 0 {
		t.Errorf("expected no archived files on first commit, got %v", archived)
	}
	adrPath, _ := data["adr_path"].(string)
	if !strings.HasPrefix(adrPath, "lifecycle/architecture/decisions/adr-0001-") {
		t.Errorf("adr_path = %q, want an adr-0001-* path", adrPath)
	}
	if got, _ := data["summary_path"].(string); got != "lifecycle/architecture/architecture-summary.md" {
		t.Errorf("summary_path = %q, want lifecycle/architecture/architecture-summary.md", got)
	}
	if got, _ := data["superseded_adr_path"].(string); got != "" {
		t.Errorf("superseded_adr_path = %q, want empty on first commit", got)
	}

	adrContent := string(readFileMust(t, filepath.Join(env.projectRoot, filepath.FromSlash(adrPath))))
	if !strings.Contains(adrContent, "Adopt Modular Monolith with Go + Vue (High-Performance Lean Stack)") {
		t.Errorf("adr title missing/wrong:\n%s", adrContent)
	}
	if !strings.Contains(adrContent, "Will multiple people edit shared data?") {
		t.Errorf("adr missing Q&A trail:\n%s", adrContent)
	}

	summaryContent := string(readFileMust(t, filepath.Join(env.projectRoot, "lifecycle", "architecture", "architecture-summary.md")))
	if !strings.Contains(summaryContent, "Multiple editors") {
		t.Errorf("summary missing breaking-requirement row:\n%s", summaryContent)
	}

	listResp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
	requireStatus(t, listResp, 200)
	listData := readJSON(t, listResp)
	for _, p := range []string{wantArch, wantStack, adrPath, "lifecycle/architecture/architecture-summary.md"} {
		if findArtifactRow(t, listData, p) == nil {
			t.Errorf("expected %q in /artifacts after commit (re-index)", p)
		}
	}
}

// TestWizardCommit_AbandonedBeforeCommit_LeavesNoFiles covers NFR-1: reading
// the wizard, fetching recommendations, and saving in-progress state never
// write anything under lifecycle/architecture/ beyond the pre-existing
// catalog — only an actual commit call does.
func TestWizardCommit_AbandonedBeforeCommit_LeavesNoFiles(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())
	archDir := filepath.Join(env.projectRoot, "lifecycle", "architecture")

	env.doRequest("GET", "/api/p/testproject/architecture/wizard", nil).Body.Close()
	rec := env.doRequest("POST", "/api/p/testproject/architecture/wizard/recommend",
		map[string]any{"answers": []map[string]string{wizardAnswer("mobile", "yes")}})
	requireStatus(t, rec, 200)
	rec.Body.Close()
	st := env.doRequest("PUT", "/api/p/testproject/architecture/wizard/state", map[string]any{
		"path": "guided", "step": "select", "updated_unix": 1700000000,
	})
	requireStatus(t, st, 200)
	st.Body.Close()

	entries, err := os.ReadDir(archDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("expected no root-level files under lifecycle/architecture/ before commit, found %q", e.Name())
		}
		if e.IsDir() && e.Name() != "architectures" && e.Name() != "tech-stacks" {
			t.Errorf("expected only the pre-seeded catalog dirs, found unexpected dir %q", e.Name())
		}
	}
}

// TestWizardCommit_SameSelectionRecommit_Idempotent covers NFR-2: recommitting
// the identical architecture/stack selection produces exactly one of each
// artefact — no duplicate promoted copies, ADRs, or summaries.
func TestWizardCommit_SameSelectionRecommit_Idempotent(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())
	req := wizardCommitReq("architectures/modular-monolith.md", "tech-stacks/go-vue.md", nil, nil, nil)

	resp1 := env.doRequest("POST", "/api/p/testproject/architecture/wizard/commit", req)
	requireStatus(t, resp1, 200)
	resp1.Body.Close()

	resp2 := env.doRequest("POST", "/api/p/testproject/architecture/wizard/commit", req)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	if archived, _ := data2["archived"].([]any); len(archived) != 0 {
		t.Errorf("expected no archived files on idempotent re-commit, got %v", archived)
	}
	if got, _ := data2["superseded_adr_path"].(string); got != "" {
		t.Errorf("superseded_adr_path = %q, want empty on a same-selection re-commit", got)
	}

	archDir := filepath.Join(env.projectRoot, "lifecycle", "architecture")
	rootMD := countRootMarkdownFiles(t, archDir)
	if rootMD != 3 { // promoted architecture + promoted stack + architecture-summary.md
		t.Errorf("expected exactly 3 root .md files (architecture + stack + summary), got %d", rootMD)
	}

	decisionsDir := filepath.Join(archDir, "decisions")
	adrEntries, err := os.ReadDir(decisionsDir)
	if err != nil {
		t.Fatal(err)
	}
	adrCount := 0
	for _, e := range adrEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			adrCount++
		}
	}
	if adrCount != 1 {
		t.Errorf("expected exactly 1 ADR file after idempotent re-commit, got %d", adrCount)
	}
}

// TestWizardCommit_ChangedSelectionRecommit_ArchivesAndSupersedes covers
// FR-16: committing a different architecture/stack archives the prior
// promoted copies, writes a new superseding ADR, marks the prior ADR
// superseded, and refreshes the summary.
func TestWizardCommit_ChangedSelectionRecommit_ArchivesAndSupersedes(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())

	resp1 := env.doRequest("POST", "/api/p/testproject/architecture/wizard/commit",
		wizardCommitReq("architectures/modular-monolith.md", "tech-stacks/go-vue.md", nil, nil, nil))
	requireStatus(t, resp1, 200)
	data1 := readJSON(t, resp1)
	firstADR, _ := data1["adr_path"].(string)

	resp2 := env.doRequest("POST", "/api/p/testproject/architecture/wizard/commit",
		wizardCommitReq("architectures/static-site.md", "tech-stacks/static-html-js.md", nil, nil, nil))
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)

	archived, _ := data2["archived"].([]any)
	if len(archived) != 2 {
		t.Fatalf("expected both prior promoted copies archived, got %v", archived)
	}
	wantNewArch := "lifecycle/architecture/static-site.md"
	wantNewStack := "lifecycle/architecture/static-html-js.md"
	if got, _ := data2["promoted_architecture"].(string); got != wantNewArch {
		t.Errorf("promoted_architecture = %q, want %q", got, wantNewArch)
	}
	if got, _ := data2["promoted_tech_stack"].(string); got != wantNewStack {
		t.Errorf("promoted_tech_stack = %q, want %q", got, wantNewStack)
	}

	newADR, _ := data2["adr_path"].(string)
	if newADR == "" || newADR == firstADR {
		t.Fatalf("expected a new, distinct ADR on changed-selection re-commit; first=%q new=%q", firstADR, newADR)
	}
	if got, _ := data2["superseded_adr_path"].(string); got != firstADR {
		t.Errorf("superseded_adr_path = %q, want the first commit's adr_path %q", got, firstADR)
	}

	priorADRContent := string(readFileMust(t, filepath.Join(env.projectRoot, filepath.FromSlash(firstADR))))
	if !strings.Contains(priorADRContent, "status: superseded") {
		t.Errorf("prior ADR not marked superseded:\n%s", priorADRContent)
	}
	if !strings.Contains(priorADRContent, "Superseded by:") {
		t.Errorf("prior ADR missing 'Superseded by' pointer:\n%s", priorADRContent)
	}

	summaryContent := string(readFileMust(t, filepath.Join(env.projectRoot, "lifecycle", "architecture", "architecture-summary.md")))
	if !strings.Contains(summaryContent, "static-site.md") {
		t.Errorf("summary was not refreshed to the new selection:\n%s", summaryContent)
	}
}

// TestWizardCommit_NonProductOwner_Returns403 covers OQ-5: the commit
// endpoint is gated to product-owner; a qa-only user is denied.
func TestWizardCommit_NonProductOwner_Returns403(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())
	env.login("qa@test.local", "qa-pass-123") // holds only "qa" under defaultCfgYAML

	resp := env.doRequest("POST", "/api/p/testproject/architecture/wizard/commit",
		wizardCommitReq("architectures/modular-monolith.md", "tech-stacks/go-vue.md", nil, nil, nil))
	requireStatus(t, resp, 403)
}

// countRootMarkdownFiles counts .md files directly under dir (not recursing
// into subdirectories like archive/ or decisions/).
func countRootMarkdownFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			n++
		}
	}
	return n
}
