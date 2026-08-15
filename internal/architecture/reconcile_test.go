// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

func TestEnsureArchitectureScaffold_EmptyProject_SeedsFullCatalog(t *testing.T) {
	root := t.TempDir()

	results, err := architecture.EnsureArchitectureScaffold(root)
	if err != nil {
		t.Fatalf("EnsureArchitectureScaffold: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for a freshly-scaffolded project, got none")
	}

	for _, sub := range []string{"architectures", "tech-stacks", "decisions", "standards"} {
		if !isDirT(t, filepath.Join(root, "lifecycle/architecture", sub)) {
			t.Errorf("expected directory lifecycle/architecture/%s to exist", sub)
		}
	}
	for _, empty := range []string{"decisions", "standards"} {
		if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture", empty, ".gitkeep")); err != nil {
			t.Errorf("expected .gitkeep in %s: %v", empty, err)
		}
	}

	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/README.md")); err != nil {
		t.Errorf("expected README.md to be seeded: %v", err)
	}

	wantFiles := catalogFileCount(t)
	gotFiles := 0
	filepath.WalkDir(filepath.Join(root, "lifecycle/architecture"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(p) == ".md" {
			gotFiles++
		}
		return nil
	})
	if gotFiles != wantFiles {
		t.Errorf("seeded %d catalog .md files, want %d", gotFiles, wantFiles)
	}
}

func TestEnsureArchitectureScaffold_PopulatedTree_Untouched(t *testing.T) {
	root := t.TempDir()
	customArch := filepath.Join(root, "lifecycle/architecture/architectures/local-web.md")
	mustWriteT(t, customArch, "---\ntitle: Custom\ntype: architecture\nstatus: draft\n---\n\ncustom content\n")

	if _, err := architecture.EnsureArchitectureScaffold(root); err != nil {
		t.Fatalf("EnsureArchitectureScaffold: %v", err)
	}

	got, err := os.ReadFile(customArch)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "---\ntitle: Custom\ntype: architecture\nstatus: draft\n---\n\ncustom content\n" {
		t.Errorf("existing catalog file was overwritten: %s", got)
	}
}

func TestEnsureArchitectureScaffold_PartialTree_Completed(t *testing.T) {
	root := t.TempDir()
	// Only the architectures/ subdirectory exists, with no seeded files.
	if err := os.MkdirAll(filepath.Join(root, "lifecycle/architecture/architectures"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := architecture.EnsureArchitectureScaffold(root); err != nil {
		t.Fatalf("EnsureArchitectureScaffold: %v", err)
	}

	for _, sub := range []string{"tech-stacks", "decisions", "standards"} {
		if !isDirT(t, filepath.Join(root, "lifecycle/architecture", sub)) {
			t.Errorf("expected partial tree to be completed with %s", sub)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "lifecycle/architecture/architectures/local-web.md")); err != nil {
		t.Errorf("expected architectures/ to be seeded: %v", err)
	}
}

func TestEnsureArchitectureScaffold_ReRun_IsNoOp(t *testing.T) {
	root := t.TempDir()
	if _, err := architecture.EnsureArchitectureScaffold(root); err != nil {
		t.Fatalf("first run: %v", err)
	}

	second, err := architecture.EnsureArchitectureScaffold(root)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("expected re-run to be a no-op, got %d results: %v", len(second), second)
	}
}

func isDirT(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func catalogFileCount(t *testing.T) int {
	t.Helper()
	count := 0
	if err := fs.WalkDir(catalogfs.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return count
}

func mustWriteT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
