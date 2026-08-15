// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

func TestSupersede_MarksPriorADRAndPointsToNew(t *testing.T) {
	root := t.TempDir()
	priorRel, err := architecture.CreateADR(root, architecture.ADRRequest{Slug: "adopt-modular-monolith", Title: "Adopt Modular Monolith", Status: "approved"})
	if err != nil {
		t.Fatalf("CreateADR prior: %v", err)
	}
	newRel, err := architecture.CreateADR(root, architecture.ADRRequest{Slug: "readopt-cloud-native-microservices", Title: "Re-adopt Cloud-Native Microservices", Status: "approved"})
	if err != nil {
		t.Fatalf("CreateADR new: %v", err)
	}

	if err := architecture.Supersede(root, priorRel, newRel); err != nil {
		t.Fatalf("Supersede: %v", err)
	}

	content := readFile(t, filepath.Join(root, priorRel))
	if !strings.Contains(content, "status: superseded") {
		t.Errorf("expected status: superseded, got:\n%s", content)
	}
	if !strings.Contains(content, "Superseded by") || !strings.Contains(content, "adr-0002-readopt-cloud-native-microservices.md") {
		t.Errorf("expected a Superseded-by pointer to the new ADR, got:\n%s", content)
	}

	newContent := readFile(t, filepath.Join(root, newRel))
	if !strings.Contains(newContent, "status: approved") {
		t.Errorf("new ADR should remain approved, got:\n%s", newContent)
	}
}

func TestSelectionChanged_NothingPromotedYet_IsChanged(t *testing.T) {
	root := writeCatalogFixture(t)
	changed, err := architecture.SelectionChanged(root, architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/postgres-modular-monolith.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	})
	if err != nil {
		t.Fatalf("SelectionChanged: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when nothing is promoted yet")
	}
}

func TestSelectionChanged_SameSelection_IsUnchanged(t *testing.T) {
	root := writeCatalogFixture(t)
	req := architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/postgres-modular-monolith.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	}
	if _, err := architecture.Promote(root, req); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	changed, err := architecture.SelectionChanged(root, req)
	if err != nil {
		t.Fatalf("SelectionChanged: %v", err)
	}
	if changed {
		t.Error("expected changed=false for the same selection already promoted")
	}
}

func TestSelectionChanged_DifferentArchitecture_IsChanged(t *testing.T) {
	root := writeCatalogFixture(t)
	first := architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/postgres-modular-monolith.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	}
	if _, err := architecture.Promote(root, first); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	second := architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/event-sourced.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	}
	changed, err := architecture.SelectionChanged(root, second)
	if err != nil {
		t.Fatalf("SelectionChanged: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for a different architecture selection")
	}
}
