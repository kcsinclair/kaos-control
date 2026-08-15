// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/onboarding-architecture-selection-5-test.md
// — Milestone 4 (FR-2, FR-3, FR-7, FR-9, FR-10, OQ-2, NFR-1): the wizard's
// non-writing HTTP surface — GET /architecture/wizard (questions, prior-run
// detection, resumable state), POST /wizard/recommend, GET /wizard/stacks,
// and PUT/DELETE /wizard/state.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// wizardCatalogSeeds seeds a representative subset of the real
// lifecycle/architecture/{architectures,tech-stacks}/*.md catalog — same
// slugs, labels, and related_to edges as production — so answers to the
// shipped default question set (internal/config defaultArchitectureWizard)
// produce realistic filtering/ranking behaviour.
func wizardCatalogSeeds() []seedArtifact {
	return []seedArtifact{
		{
			relPath: "lifecycle/architecture/architectures/modular-monolith.md",
			content: wizardArchArtifact("Modular Monolith", "arch-modular-monolith",
				[]string{"architecture", "catalog", "collaborative", "low-complexity", "low-cost-start"},
				[]string{"architecture/tech-stacks/go-vue.md", "architecture/tech-stacks/python-fastapi.md"},
				"A single deployable application organised into well-bounded internal modules."),
		},
		{
			relPath: "lifecycle/architecture/architectures/mobile-native.md",
			content: wizardArchArtifact("Mobile-Native Application", "arch-mobile-native",
				[]string{"architecture", "catalog", "mobile", "offline-capable", "medium-complexity"},
				[]string{"architecture/tech-stacks/flutter.md", "architecture/tech-stacks/go-vue.md"},
				"A phone/tablet app as the primary client, talking to a cloud backend."),
		},
		{
			relPath: "lifecycle/architecture/architectures/static-site.md",
			content: wizardArchArtifact("Static Website / JAMstack", "arch-static-site",
				[]string{"architecture", "catalog", "static", "low-complexity"},
				[]string{"architecture/tech-stacks/static-html-js.md"},
				"Pre-built static files served from any web host or CDN."),
		},
		{
			relPath: "lifecycle/architecture/tech-stacks/go-vue.md",
			content: wizardStackArtifact("Go + Vue (High-Performance Lean Stack)", "stack-go-vue",
				[]string{"tech-stack", "catalog", "backend", "frontend", "go", "vue"},
				"Go backend + reactive Vue SPA."),
		},
		{
			relPath: "lifecycle/architecture/tech-stacks/python-fastapi.md",
			content: wizardStackArtifact("Python + FastAPI (Intelligence & Prototyping Stack)", "stack-python-fastapi",
				[]string{"tech-stack", "catalog", "backend", "python", "ai-ml"},
				"Async Python via FastAPI."),
		},
		{
			relPath: "lifecycle/architecture/tech-stacks/flutter.md",
			content: wizardStackArtifact("Flutter (Dart, Canvas-Rendered)", "stack-flutter",
				[]string{"tech-stack", "catalog", "desktop", "mobile", "dart"},
				"One codebase for mobile, desktop, and web."),
		},
		{
			relPath: "lifecycle/architecture/tech-stacks/static-html-js.md",
			content: wizardStackArtifact("Static HTML / CSS / JS (No-Framework Frontend)", "stack-static-html-js",
				[]string{"tech-stack", "catalog", "frontend", "static", "html", "javascript", "low-complexity"},
				"Hand-authored static HTML/CSS/vanilla-JS."),
		},
	}
}

func wizardArchArtifact(title, lineage string, labels, relatedTo []string, summary string) string {
	var sb bytes.Buffer
	sb.WriteString("---\n")
	sb.WriteString("title: " + title + "\n")
	sb.WriteString("type: architecture\n")
	sb.WriteString("status: draft\n")
	sb.WriteString("lineage: " + lineage + "\n")
	sb.WriteString("labels:\n")
	for _, l := range labels {
		sb.WriteString("    - " + l + "\n")
	}
	sb.WriteString("related_to:\n")
	for _, r := range relatedTo {
		sb.WriteString("    - " + r + "\n")
	}
	sb.WriteString("summary: " + summary + "\n")
	sb.WriteString("---\n\n# " + title + "\n\nBody.\n")
	return sb.String()
}

func wizardStackArtifact(title, lineage string, labels []string, summary string) string {
	var sb bytes.Buffer
	sb.WriteString("---\n")
	sb.WriteString("title: " + title + "\n")
	sb.WriteString("type: tech-stack\n")
	sb.WriteString("status: draft\n")
	sb.WriteString("lineage: " + lineage + "\n")
	sb.WriteString("labels:\n")
	for _, l := range labels {
		sb.WriteString("    - " + l + "\n")
	}
	sb.WriteString("summary: " + summary + "\n")
	sb.WriteString("---\n\n# " + title + "\n\nBody.\n")
	return sb.String()
}

// TestWizardGet_FreshProject_ReturnsQuestionsAndNoPriorRun covers FR-7/FR-2:
// a fresh project's GET /architecture/wizard returns the built-in default
// question set (self-repaired by config.ValidateAndRepair since no
// architecture_wizard section is configured), capped at 10, and reports no
// prior run.
func TestWizardGet_FreshProject_ReturnsQuestionsAndNoPriorRun(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())

	resp := env.doRequest("GET", "/api/p/testproject/architecture/wizard", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	questions, _ := data["questions"].([]any)
	if len(questions) == 0 || len(questions) > 10 {
		t.Fatalf("expected 1-10 questions, got %d", len(questions))
	}
	if got, _ := data["default_architecture"].(string); got != "modular-monolith" {
		t.Errorf("default_architecture = %q, want %q", got, "modular-monolith")
	}
	prior, _ := data["prior_run"].(map[string]any)
	if detected, _ := prior["detected"].(bool); detected {
		t.Errorf("expected prior_run.detected=false on a fresh project, got %v", prior)
	}
	if data["resumable_state"] != nil {
		t.Errorf("expected resumable_state=nil on a fresh project, got %v", data["resumable_state"])
	}
}

// TestWizardGet_AfterCommittedSelection_DetectsPriorRun covers FR-2/FR-3:
// once a selection has been committed, GET /architecture/wizard reports
// prior_run.detected=true with references to what was written.
func TestWizardGet_AfterCommittedSelection_DetectsPriorRun(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())

	commitResp := env.doRequest("POST", "/api/p/testproject/architecture/wizard/commit", wizardCommitReq(
		"architectures/modular-monolith.md", "tech-stacks/go-vue.md", nil, nil, nil))
	requireStatus(t, commitResp, 200)
	commitResp.Body.Close()

	resp := env.doRequest("GET", "/api/p/testproject/architecture/wizard", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	prior, _ := data["prior_run"].(map[string]any)
	if detected, _ := prior["detected"].(bool); !detected {
		t.Fatalf("expected prior_run.detected=true after a committed selection, got %v", prior)
	}
	if got, _ := prior["architecture"].(string); got != "lifecycle/architecture/modular-monolith.md" {
		t.Errorf("prior_run.architecture = %q, want lifecycle/architecture/modular-monolith.md", got)
	}
	if got, _ := prior["tech_stack"].(string); got != "lifecycle/architecture/go-vue.md" {
		t.Errorf("prior_run.tech_stack = %q, want lifecycle/architecture/go-vue.md", got)
	}
	if got, _ := prior["adr_path"].(string); got == "" {
		t.Error("expected prior_run.adr_path to be populated")
	}
	if got, _ := prior["summary_path"].(string); got != "lifecycle/architecture/architecture-summary.md" {
		t.Errorf("prior_run.summary_path = %q, want lifecycle/architecture/architecture-summary.md", got)
	}
}

// wizardAnswer builds the {"question_id":..., "value":...} shape POST
// /wizard/recommend expects.
func wizardAnswer(questionID, value string) map[string]string {
	return map[string]string{"question_id": questionID, "value": value}
}

// wizardCommitReq builds the POST /wizard/commit request body.
func wizardCommitReq(archCatalogPath, stackCatalogPath string, answers []map[string]string, breaking []map[string]string, qa []map[string]string) map[string]any {
	return map[string]any{
		"architecture_path":     archCatalogPath,
		"tech_stack_path":       stackCatalogPath,
		"answers":               answers,
		"breaking_requirements": breaking,
		"qa":                    qa,
	}
}

// TestWizardRecommend_DeterministicAndHonoursHardFilter covers FR-9/FR-12
// and the "mobile" hard filter (FR-8): identical requests byte-match, and
// answering "mobile" restricts survivors to the one catalog architecture
// carrying the "mobile" label.
func TestWizardRecommend_DeterministicAndHonoursHardFilter(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())
	req := map[string]any{"answers": []map[string]string{wizardAnswer("mobile", "yes")}}

	resp1 := env.doRequest("POST", "/api/p/testproject/architecture/wizard/recommend", req)
	requireStatus(t, resp1, 200)
	body1, err := readBodyBytes(resp1)
	if err != nil {
		t.Fatal(err)
	}

	resp2 := env.doRequest("POST", "/api/p/testproject/architecture/wizard/recommend", req)
	requireStatus(t, resp2, 200)
	body2, err := readBodyBytes(resp2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body1, body2) {
		t.Fatalf("recommend response not byte-identical across repeated identical requests:\n%s\nvs\n%s", body1, body2)
	}

	var data map[string]any
	if err := json.Unmarshal(body1, &data); err != nil {
		t.Fatal(err)
	}
	recs, _ := data["recommendations"].([]any)
	if len(recs) != 1 {
		t.Fatalf("expected exactly 1 recommendation for a mobile-hard-filtered request, got %d: %v", len(recs), recs)
	}
	rec, _ := recs[0].(map[string]any)
	item, _ := rec["item"].(map[string]any)
	if got, _ := item["slug"].(string); got != "mobile-native" {
		t.Errorf("recommended slug = %q, want mobile-native", got)
	}
	why, _ := rec["why"].([]any)
	if len(why) == 0 {
		t.Error("expected a non-empty why for the recommendation")
	}
}

// TestWizardRecommend_OverConstrained_ReturnsDroppedConstraintsFallback
// covers OQ-2: when the hard-constraint combination filters the catalog to
// zero candidates, the endpoint relaxes the weakest constraint and reports
// it in dropped_constraints rather than returning an empty result.
func TestWizardRecommend_OverConstrained_ReturnsDroppedConstraintsFallback(t *testing.T) {
	// A catalog fixture with no "mobile"-labelled architecture at all, so
	// answering "mobile: yes" (a hard constraint) has zero real survivors.
	env := newTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/architecture/architectures/modular-monolith.md",
			content: wizardArchArtifact("Modular Monolith", "arch-modular-monolith",
				[]string{"architecture", "catalog", "collaborative", "low-complexity"}, nil, "Body."),
		},
		{
			relPath: "lifecycle/architecture/architectures/static-site.md",
			content: wizardArchArtifact("Static Website / JAMstack", "arch-static-site",
				[]string{"architecture", "catalog", "static", "low-complexity"}, nil, "Body."),
		},
	})

	resp := env.doRequest("POST", "/api/p/testproject/architecture/wizard/recommend",
		map[string]any{"answers": []map[string]string{wizardAnswer("mobile", "yes")}})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	dropped, _ := data["dropped_constraints"].([]any)
	if len(dropped) != 1 || dropped[0] != "mobile" {
		t.Fatalf("dropped_constraints = %v, want [\"mobile\"]", dropped)
	}
	recs, _ := data["recommendations"].([]any)
	if len(recs) != 2 {
		t.Fatalf("expected both fixture architectures back as the fallback candidate set, got %d", len(recs))
	}
}

// TestWizardStacks_RelatedOnlyAndLanguageRanked covers FR-6/FR-10: the
// endpoint returns only the chosen architecture's related_to stacks, with
// the language-matching one ranked first.
func TestWizardStacks_RelatedOnlyAndLanguageRanked(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())

	resp := env.doRequest("GET",
		"/api/p/testproject/architecture/wizard/stacks?architecture=modular-monolith&language=python", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	stacks, _ := data["stacks"].([]any)
	if len(stacks) != 2 {
		t.Fatalf("expected exactly modular-monolith's 2 related_to stacks, got %d: %v", len(stacks), stacks)
	}
	first, _ := stacks[0].(map[string]any)
	if got, _ := first["slug"].(string); got != "python-fastapi" {
		t.Errorf("first-ranked stack = %q, want python-fastapi (language match)", got)
	}

	// A stack outside modular-monolith's related_to (flutter, only related to
	// mobile-native) must never appear.
	for _, raw := range stacks {
		s, _ := raw.(map[string]any)
		if s["slug"] == "flutter" {
			t.Error("flutter is not related_to modular-monolith but was returned")
		}
	}

	// static-site's related_to is a single stack — confirms restriction, not
	// just ranking, drives the filter.
	resp2 := env.doRequest("GET", "/api/p/testproject/architecture/wizard/stacks?architecture=static-site", nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	stacks2, _ := data2["stacks"].([]any)
	if len(stacks2) != 1 {
		t.Fatalf("expected exactly 1 stack related_to static-site, got %d: %v", len(stacks2), stacks2)
	}
	only, _ := stacks2[0].(map[string]any)
	if got, _ := only["slug"].(string); got != "static-html-js" {
		t.Errorf("static-site's only compatible stack = %q, want static-html-js", got)
	}
}

// TestWizardState_RoundTripsAndWritesNothingUnderLifecycleArchitecture
// covers OQ-3/NFR-1: PUT saves resumable state (visible back via GET
// /architecture/wizard's resumable_state), DELETE clears it, and neither
// call ever writes a file under lifecycle/architecture/.
func TestWizardState_RoundTripsAndWritesNothingUnderLifecycleArchitecture(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())
	archDir := filepath.Join(env.projectRoot, "lifecycle", "architecture")
	before := countArchitectureTreeFiles(t, archDir)

	state := map[string]any{
		"path":                "guided",
		"answers":             []map[string]string{wizardAnswer("mobile", "yes")},
		"chosen_architecture": "",
		"chosen_tech_stack":   "",
		"step":                "select",
		"updated_unix":        1700000000,
	}
	putResp := env.doRequest("PUT", "/api/p/testproject/architecture/wizard/state", state)
	requireStatus(t, putResp, 200)
	putResp.Body.Close()

	if got := countArchitectureTreeFiles(t, archDir); got != before {
		t.Fatalf("PUT /wizard/state wrote under lifecycle/architecture/: %d files before, %d after", before, got)
	}

	getResp := env.doRequest("GET", "/api/p/testproject/architecture/wizard", nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)
	resumable, _ := getData["resumable_state"].(map[string]any)
	if resumable == nil {
		t.Fatal("expected resumable_state to be populated after PUT /wizard/state")
	}
	if got, _ := resumable["path"].(string); got != "guided" {
		t.Errorf("resumable_state.path = %q, want guided", got)
	}
	if got, _ := resumable["step"].(string); got != "select" {
		t.Errorf("resumable_state.step = %q, want select", got)
	}

	delResp := env.doRequest("DELETE", "/api/p/testproject/architecture/wizard/state", nil)
	requireStatus(t, delResp, 200)
	delResp.Body.Close()

	if got := countArchitectureTreeFiles(t, archDir); got != before {
		t.Fatalf("DELETE /wizard/state wrote under lifecycle/architecture/: %d files before, %d after", before, got)
	}

	getResp2 := env.doRequest("GET", "/api/p/testproject/architecture/wizard", nil)
	requireStatus(t, getResp2, 200)
	getData2 := readJSON(t, getResp2)
	if getData2["resumable_state"] != nil {
		t.Errorf("expected resumable_state=nil after DELETE, got %v", getData2["resumable_state"])
	}
}

// countArchitectureTreeFiles recursively counts regular files under dir.
func countArchitectureTreeFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
