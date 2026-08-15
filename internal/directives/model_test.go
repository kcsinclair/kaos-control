// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

func TestBuildModel_PromotedStack(t *testing.T) {
	root := t.TempDir()
	raw, err := catalogfs.FS.ReadFile("tech-stacks/go-vue.md")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "lifecycle/architecture/go-vue.md")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := BuildModel(root)
	if err != nil {
		t.Fatalf("BuildModel: %v", err)
	}
	if !m.ArchitecturePointer {
		t.Error("expected ArchitecturePointer=true")
	}
	if m.StackTitle == "" {
		t.Error("expected non-empty StackTitle")
	}
	if len(m.RepoLayout) == 0 {
		t.Error("expected non-empty RepoLayout")
	}
}

func TestBuildModel_NoPromotedStack(t *testing.T) {
	root := t.TempDir()
	_, err := BuildModel(root)
	if !errors.Is(err, architecture.ErrNoPromotedStack) {
		t.Fatalf("expected ErrNoPromotedStack, got %v", err)
	}
}
