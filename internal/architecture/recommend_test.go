// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/config"
)

// repoRoot locates the project root from this test file's own source path so
// tests can load the real, shipped lifecycle/architecture/ catalog — the
// "real catalog fixture" the backend plan calls for.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func loadRealCatalog(t *testing.T) (arches, stacks []architecture.CatalogItem) {
	t.Helper()
	arches, stacks, err := architecture.LoadCatalog(repoRoot(t))
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(arches) == 0 || len(stacks) == 0 {
		t.Fatalf("expected a non-empty catalog, got %d architectures / %d stacks", len(arches), len(stacks))
	}
	return arches, stacks
}

func slugs(recs []architecture.Recommendation) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.Item.Slug
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func hardYesQuestion(id, label string) config.WizardQuestion {
	return config.WizardQuestion{
		ID:   id,
		Kind: "hard",
		Options: []config.WizardOption{
			{Value: "yes", Label: "Yes", Labels: []string{label}, Hard: true},
			{Value: "no", Label: "No"},
		},
	}
}

func softYesQuestion(id, label string) config.WizardQuestion {
	return config.WizardQuestion{
		ID:   id,
		Kind: "soft",
		Options: []config.WizardOption{
			{Value: "yes", Label: "Yes", Labels: []string{label}},
			{Value: "no", Label: "No"},
		},
	}
}

func TestRecommend_HardConstraint_OfflineFiltersToOfflineCapable(t *testing.T) {
	arches, _ := loadRealCatalog(t)
	wizardCfg := config.ArchitectureWizardConfig{
		DefaultArchitecture: "modular-monolith",
		Questions:           []config.WizardQuestion{hardYesQuestion("offline", "offline-capable")},
	}

	recs, dropped, err := architecture.Recommend(arches, wizardCfg, []architecture.Answer{{QuestionID: "offline", Value: "yes"}})
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if len(dropped) != 0 {
		t.Errorf("dropped = %v, want none (no over-constraint)", dropped)
	}
	got := slugs(recs)
	want := []string{"edge-hybrid", "mobile-native", "standalone-desktop"}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("offline-capable candidates = %v, want %v", got, want)
	}
	for _, r := range recs {
		if len(r.Why) == 0 {
			t.Errorf("recommendation %q has empty Why", r.Item.Slug)
		}
	}
}

func TestRecommend_HardConstraint_MobileRestrictsToMobileNative(t *testing.T) {
	arches, _ := loadRealCatalog(t)
	wizardCfg := config.ArchitectureWizardConfig{
		DefaultArchitecture: "modular-monolith",
		Questions:           []config.WizardQuestion{hardYesQuestion("mobile", "mobile")},
	}

	recs, _, err := architecture.Recommend(arches, wizardCfg, []architecture.Answer{{QuestionID: "mobile", Value: "yes"}})
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if len(recs) != 1 || recs[0].Item.Slug != "mobile-native" {
		t.Errorf("recs = %v, want exactly [mobile-native]", slugs(recs))
	}
}

func TestRecommend_OverConstrained_RelaxesWeakestAndReportsDropped(t *testing.T) {
	arches, _ := loadRealCatalog(t)
	wizardCfg := config.ArchitectureWizardConfig{
		DefaultArchitecture: "modular-monolith",
		Questions: []config.WizardQuestion{
			hardYesQuestion("offline", "offline-capable"),
			hardYesQuestion("scale", "high-scale"),
		},
	}
	answers := []architecture.Answer{
		{QuestionID: "offline", Value: "yes"},
		{QuestionID: "scale", Value: "yes"},
	}

	recs, dropped, err := architecture.Recommend(arches, wizardCfg, answers)
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("expected the closest candidates after relaxation, got none")
	}
	if len(dropped) != 1 || dropped[0] != "high-scale" {
		t.Errorf("dropped = %v, want [high-scale] (fewer catalog matches than offline-capable)", dropped)
	}
	for _, r := range recs {
		if !contains([]string{"edge-hybrid", "mobile-native", "standalone-desktop"}, r.Item.Slug) {
			t.Errorf("recommendation %q does not satisfy the surviving offline-capable constraint", r.Item.Slug)
		}
	}
}

func TestRecommend_WeakAnswers_DefaultsToModularMonolith(t *testing.T) {
	arches, _ := loadRealCatalog(t)
	wizardCfg := config.ArchitectureWizardConfig{
		DefaultArchitecture: "modular-monolith",
		Questions:           []config.WizardQuestion{softYesQuestion("collaborative", "collaborative")},
	}

	recs, _, err := architecture.Recommend(arches, wizardCfg, nil)
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if len(recs) == 0 || recs[0].Item.Slug != "modular-monolith" {
		t.Fatalf("recs = %v, want modular-monolith first", slugs(recs))
	}
	if len(recs[0].Why) != 1 || recs[0].Why[0] != "low-regret default — signals were weak" {
		t.Errorf("Why = %v, want the documented default-bias reason", recs[0].Why)
	}
}

func TestRecommend_SoftSignals_ScoreAndTopSet(t *testing.T) {
	arches, _ := loadRealCatalog(t)
	wizardCfg := config.ArchitectureWizardConfig{
		DefaultArchitecture: "modular-monolith",
		Questions: []config.WizardQuestion{
			softYesQuestion("collaborative", "collaborative"),
			softYesQuestion("cost", "low-cost-start"),
		},
	}
	answers := []architecture.Answer{
		{QuestionID: "collaborative", Value: "yes"},
		{QuestionID: "cost", Value: "yes"},
	}

	recs, _, err := architecture.Recommend(arches, wizardCfg, answers)
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if len(recs) < 2 || len(recs) > 3 {
		t.Fatalf("len(recs) = %d, want 2 or 3", len(recs))
	}
	for _, r := range recs {
		if len(r.Why) == 0 {
			t.Errorf("recommendation %q has empty Why", r.Item.Slug)
		}
	}
	for i := 1; i < len(recs); i++ {
		if recs[i-1].Score < recs[i].Score {
			t.Errorf("recs not sorted by descending score: %+v", recs)
		}
	}
}

func TestRecommend_Deterministic(t *testing.T) {
	arches, _ := loadRealCatalog(t)
	wizardCfg := config.ArchitectureWizardConfig{
		DefaultArchitecture: "modular-monolith",
		Questions: []config.WizardQuestion{
			softYesQuestion("collaborative", "collaborative"),
			hardYesQuestion("offline", "offline-capable"),
		},
	}
	answers := []architecture.Answer{
		{QuestionID: "collaborative", Value: "yes"},
		{QuestionID: "offline", Value: "no"},
	}

	recs1, dropped1, err := architecture.Recommend(arches, wizardCfg, answers)
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	recs2, dropped2, err := architecture.Recommend(arches, wizardCfg, answers)
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if !reflect.DeepEqual(recs1, recs2) {
		t.Errorf("Recommend is not deterministic:\n%+v\n%+v", recs1, recs2)
	}
	if !reflect.DeepEqual(dropped1, dropped2) {
		t.Errorf("dropped constraints not deterministic: %v vs %v", dropped1, dropped2)
	}
}

func TestRankStacks_OnlyRelatedTo_LanguageMatchedFirst(t *testing.T) {
	arches, stacks := loadRealCatalog(t)
	var modularMonolith architecture.CatalogItem
	found := false
	for _, a := range arches {
		if a.Slug == "modular-monolith" {
			modularMonolith, found = a, true
		}
	}
	if !found {
		t.Fatal("modular-monolith not found in the real catalog")
	}

	ranked := architecture.RankStacks(modularMonolith, stacks, "typescript")
	if len(ranked) == 0 {
		t.Fatal("expected related_to stacks, got none")
	}
	if ranked[0].Slug != "ts-react-nest" {
		t.Errorf("ranked[0] = %q, want the typescript-labelled stack first", ranked[0].Slug)
	}

	related := make(map[string]bool, len(modularMonolith.RelatedTo))
	for _, r := range modularMonolith.RelatedTo {
		related[r] = true
	}
	for _, s := range ranked {
		if !related[s.Path] {
			t.Errorf("ranked stack %q is not in modular-monolith's related_to", s.Slug)
		}
	}
}

func TestRankStacks_Deterministic(t *testing.T) {
	arches, stacks := loadRealCatalog(t)
	var chosen architecture.CatalogItem
	for _, a := range arches {
		if a.Slug == "modular-monolith" {
			chosen = a
		}
	}

	r1 := architecture.RankStacks(chosen, stacks, "go")
	r2 := architecture.RankStacks(chosen, stacks, "go")
	if !reflect.DeepEqual(r1, r2) {
		t.Errorf("RankStacks is not deterministic:\n%+v\n%+v", r1, r2)
	}
}
