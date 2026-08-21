// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// writeCatalogFixture creates a minimal project root with a catalog
// architecture and tech-stack entry and returns the root path.
func writeCatalogFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "lifecycle/architecture/architectures/postgres-modular-monolith.md"),
		"---\ntitle: Postgres Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n")
	mustWrite(t, filepath.Join(root, "lifecycle/architecture/tech-stacks/go-vue.md"),
		"---\ntitle: Go + Vue\ntype: tech-stack\nstatus: approved\n---\n\nBody.\n")
	mustWrite(t, filepath.Join(root, "lifecycle/architecture/architectures/event-sourced.md"),
		"---\ntitle: Event Sourced\ntype: architecture\nstatus: approved\n---\n\nBody.\n")
	return root
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPromote_EmptyDir_CreatesRootCopiesWithParent(t *testing.T) {
	root := writeCatalogFixture(t)
	archBefore, _ := os.ReadFile(filepath.Join(root, "lifecycle/architecture/architectures/postgres-modular-monolith.md"))
	stackBefore, _ := os.ReadFile(filepath.Join(root, "lifecycle/architecture/tech-stacks/go-vue.md"))

	result, err := architecture.Promote(root, architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/postgres-modular-monolith.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	if result.PromotedArchitecture != "lifecycle/architecture/postgres-modular-monolith.md" {
		t.Errorf("PromotedArchitecture: got %q", result.PromotedArchitecture)
	}
	if result.PromotedTechStack != "lifecycle/architecture/go-vue.md" {
		t.Errorf("PromotedTechStack: got %q", result.PromotedTechStack)
	}
	if len(result.Archived) != 0 {
		t.Errorf("expected no archived files, got %v", result.Archived)
	}

	archContent := readFile(t, filepath.Join(root, result.PromotedArchitecture))
	if !strings.Contains(archContent, "parent: lifecycle/architecture/architectures/postgres-modular-monolith.md") {
		t.Errorf("promoted architecture missing parent stamp:\n%s", archContent)
	}
	stackContent := readFile(t, filepath.Join(root, result.PromotedTechStack))
	if !strings.Contains(stackContent, "parent: lifecycle/architecture/tech-stacks/go-vue.md") {
		t.Errorf("promoted tech-stack missing parent stamp:\n%s", stackContent)
	}
	// Promoted copies also get a created: stamp (catalog sources carry none).
	if !strings.Contains(archContent, "created:") || !strings.Contains(stackContent, "created:") {
		t.Errorf("promoted copies missing created: stamp:\narch:\n%s\nstack:\n%s", archContent, stackContent)
	}

	// Catalog sources are untouched.
	archAfter, _ := os.ReadFile(filepath.Join(root, "lifecycle/architecture/architectures/postgres-modular-monolith.md"))
	if string(archBefore) != string(archAfter) {
		t.Error("catalog architecture source was modified")
	}
	stackAfter, _ := os.ReadFile(filepath.Join(root, "lifecycle/architecture/tech-stacks/go-vue.md"))
	if string(stackBefore) != string(stackAfter) {
		t.Error("catalog tech-stack source was modified")
	}
}

func TestPromote_StripsCatalogLabel(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "lifecycle/architecture/architectures/modular-monolith.md"),
		"---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\nlabels:\n    - architecture\n    - catalog\n    - collaborative\n---\n\nBody.\n")
	mustWrite(t, filepath.Join(root, "lifecycle/architecture/tech-stacks/go-vue.md"),
		"---\ntitle: Go + Vue\ntype: tech-stack\nstatus: approved\nlabels:\n    - tech-stack\n    - catalog\n---\n\nBody.\n")

	result, err := architecture.Promote(root, architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/modular-monolith.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	arch := readFile(t, filepath.Join(root, result.PromotedArchitecture))
	stack := readFile(t, filepath.Join(root, result.PromotedTechStack))
	// The chosen copies are no longer catalog candidates.
	if strings.Contains(arch, "- catalog") {
		t.Errorf("promoted architecture still carries the catalog label:\n%s", arch)
	}
	if strings.Contains(stack, "- catalog") {
		t.Errorf("promoted tech-stack still carries the catalog label:\n%s", stack)
	}
	// Sibling labels are preserved.
	if !strings.Contains(arch, "- architecture") || !strings.Contains(arch, "- collaborative") {
		t.Errorf("promoted architecture lost sibling labels:\n%s", arch)
	}

	// The catalog SOURCE keeps its catalog label (only the promoted copy changes).
	src := readFile(t, filepath.Join(root, "lifecycle/architecture/architectures/modular-monolith.md"))
	if !strings.Contains(src, "- catalog") {
		t.Errorf("catalog source lost its catalog label:\n%s", src)
	}
}

func TestPromote_SameSelection_IsIdempotent(t *testing.T) {
	root := writeCatalogFixture(t)
	req := architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/postgres-modular-monolith.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	}
	if _, err := architecture.Promote(root, req); err != nil {
		t.Fatalf("first Promote: %v", err)
	}
	if _, err := architecture.Promote(root, req); err != nil {
		t.Fatalf("second Promote: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(root, "lifecycle/architecture"))
	if err != nil {
		t.Fatal(err)
	}
	var mdFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) != 2 {
		t.Errorf("expected exactly 2 root .md files after re-promoting the same selection, got %v", mdFiles)
	}

	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/archive")); !os.IsNotExist(err) {
		t.Errorf("archive/ should not be created for an idempotent re-promote, stat err=%v", err)
	}
}

func TestPromote_ChangedSelection_ArchivesPriorCopy(t *testing.T) {
	root := writeCatalogFixture(t)
	first := architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/postgres-modular-monolith.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	}
	if _, err := architecture.Promote(root, first); err != nil {
		t.Fatalf("first Promote: %v", err)
	}

	second := architecture.PromotionRequest{
		ArchitectureCatalogPath: "architectures/event-sourced.md",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	}
	result, err := architecture.Promote(root, second)
	if err != nil {
		t.Fatalf("second Promote: %v", err)
	}

	if len(result.Archived) != 1 {
		t.Fatalf("expected exactly 1 archived file, got %v", result.Archived)
	}
	if result.Archived[0] != "lifecycle/architecture/archive/postgres-modular-monolith.md" {
		t.Errorf("Archived[0]: got %q", result.Archived[0])
	}
	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/postgres-modular-monolith.md")); !os.IsNotExist(err) {
		t.Error("prior architecture root copy should have been moved, not left in place")
	}
	if _, err := os.Stat(filepath.Join(root, result.PromotedArchitecture)); err != nil {
		t.Errorf("new architecture root copy missing: %v", err)
	}
	if result.PromotedArchitecture != "lifecycle/architecture/event-sourced.md" {
		t.Errorf("PromotedArchitecture: got %q", result.PromotedArchitecture)
	}

	// Catalog untouched.
	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/architectures/postgres-modular-monolith.md")); err != nil {
		t.Errorf("catalog source should remain: %v", err)
	}
}

func TestPromote_TraversalSource_ErrorsWithPathTraversal(t *testing.T) {
	root := writeCatalogFixture(t)
	_, err := architecture.Promote(root, architecture.PromotionRequest{
		ArchitectureCatalogPath: "../../etc/x",
		TechStackCatalogPath:    "tech-stacks/go-vue.md",
	})
	if err == nil {
		t.Fatal("expected an error for a traversal source path")
	}
	if !errors.Is(err, sandbox.ErrPathTraversal) {
		t.Errorf("expected error wrapping sandbox.ErrPathTraversal, got: %v", err)
	}
}

// TestPromote_TwoArchivedGenerations_CoexistWithDisambiguator drives two
// separate replacements of the same basename (postgres-modular-monolith)
// through an intervening different selection, so the second archive attempt
// collides with the first and must pick up the -1 disambiguator rather than
// overwriting archive/postgres-modular-monolith.md.
func TestPromote_TwoArchivedGenerations_CoexistWithDisambiguator(t *testing.T) {
	root := writeCatalogFixture(t)
	promote := func(archPath string) {
		t.Helper()
		if _, err := architecture.Promote(root, architecture.PromotionRequest{
			ArchitectureCatalogPath: archPath,
			TechStackCatalogPath:    "tech-stacks/go-vue.md",
		}); err != nil {
			t.Fatalf("Promote(%s): %v", archPath, err)
		}
	}

	promote("architectures/postgres-modular-monolith.md") // root: postgres...
	promote("architectures/event-sourced.md")             // archives postgres... -> archive/postgres...md
	promote("architectures/postgres-modular-monolith.md") // archives event-sourced; root: postgres... again
	promote("architectures/event-sourced.md")             // archives postgres... again -> collides, gets -1

	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/archive/postgres-modular-monolith.md")); err != nil {
		t.Errorf("expected first archived generation to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/archive/postgres-modular-monolith-1.md")); err != nil {
		t.Errorf("expected second archived generation with -1 disambiguator to exist: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
