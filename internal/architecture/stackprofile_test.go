// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

func mustReadCatalog(t *testing.T, relPath string) []byte {
	t.Helper()
	raw, err := catalogfs.FS.ReadFile(relPath)
	if err != nil {
		t.Fatalf("reading embedded catalog file %q: %v", relPath, err)
	}
	return raw
}

func TestParseStackProfile_GoVue(t *testing.T) {
	raw := mustReadCatalog(t, "tech-stacks/go-vue.md")

	profile, err := architecture.ParseStackProfile(raw)
	if err != nil {
		t.Fatalf("ParseStackProfile: %v", err)
	}

	if profile.Run != "go run ./cmd/<app>" {
		t.Errorf("Run: got %q", profile.Run)
	}
	if len(profile.Roles) != 3 {
		t.Fatalf("expected 3 roles, got %d: %v", len(profile.Roles), profile.Roles)
	}
	be, ok := profile.Roles["backend-developer"]
	if !ok {
		t.Fatal("missing backend-developer role")
	}
	if len(be.WritePaths) != 2 || be.WritePaths[0] != "internal" || be.WritePaths[1] != "cmd" {
		t.Errorf("backend-developer.WritePaths: got %v", be.WritePaths)
	}
	if be.Build != "go build ./..." {
		t.Errorf("backend-developer.Build: got %q", be.Build)
	}
	if !be.IsRequired() {
		t.Error("backend-developer should be required by default")
	}
}

func TestParseStackProfile_StaticHTMLJS(t *testing.T) {
	raw := mustReadCatalog(t, "tech-stacks/static-html-js.md")

	profile, err := architecture.ParseStackProfile(raw)
	if err != nil {
		t.Fatalf("ParseStackProfile: %v", err)
	}

	be, ok := profile.Roles["backend-developer"]
	if !ok {
		t.Fatal("missing backend-developer role")
	}
	if be.IsRequired() {
		t.Error("backend-developer should not be required for static-html-js")
	}

	fe, ok := profile.Roles["frontend-developer"]
	if !ok {
		t.Fatal("missing frontend-developer role")
	}
	if fe.Build != "" {
		t.Errorf("frontend-developer.Build: expected empty, got %q", fe.Build)
	}
}

func TestParseStackProfile_NoBlock(t *testing.T) {
	_, err := architecture.ParseStackProfile([]byte("---\ntitle: X\ntype: tech-stack\nstatus: draft\n---\n\nNo profile here.\n"))
	if !errors.Is(err, architecture.ErrNoStackProfile) {
		t.Fatalf("expected ErrNoStackProfile, got %v", err)
	}
}

func TestLoadPromotedStackProfile(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "lifecycle/architecture/go-vue.md")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, mustReadCatalog(t, "tech-stacks/go-vue.md"), 0o644); err != nil {
		t.Fatal(err)
	}

	profile, title, err := architecture.LoadPromotedStackProfile(root)
	if err != nil {
		t.Fatalf("LoadPromotedStackProfile: %v", err)
	}
	if title != "Go + Vue (High-Performance Lean Stack)" {
		t.Errorf("title: got %q", title)
	}
	if profile.Run != "go run ./cmd/<app>" {
		t.Errorf("Run: got %q", profile.Run)
	}
}

func TestLoadPromotedStackProfile_NonePromoted(t *testing.T) {
	root := t.TempDir()
	_, _, err := architecture.LoadPromotedStackProfile(root)
	if !errors.Is(err, architecture.ErrNoPromotedStack) {
		t.Fatalf("expected ErrNoPromotedStack, got %v", err)
	}
}
