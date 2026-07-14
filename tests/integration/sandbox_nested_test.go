// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// Milestone 4 — nested-path safety and traversal rejection for sandbox.Resolve.
//
// internal/sandbox has no existing unit test file; per the write-scope
// restriction for this suite (tests/**, see idea-archiving-5-test.md) these
// cases exercise the exported sandbox.Resolve function directly rather than
// adding internal/sandbox/sandbox_test.go.

func TestSandboxResolve_NestedPathStaysInRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lifecycle", "ideas", "archive"), 0o755); err != nil {
		t.Fatal(err)
	}

	resolved, err := sandbox.Resolve(root, "lifecycle/ideas/archive/foo.md")
	if err != nil {
		t.Fatalf("Resolve nested path: %v", err)
	}
	evalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(evalRoot, "lifecycle", "ideas", "archive", "foo.md")
	if resolved != want {
		t.Errorf("resolved: want %q, got %q", want, resolved)
	}
}

func TestSandboxResolve_DeeplyNestedNonExistentPathStaysInRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}

	// The 2026/q3 subdirectories don't exist yet — Resolve must still permit
	// creating a new artifact under them (walks up to the nearest existing
	// ancestor to verify containment).
	resolved, err := sandbox.Resolve(root, "lifecycle/ideas/2026/q3/release-x.md")
	if err != nil {
		t.Fatalf("Resolve deeply nested non-existent path: %v", err)
	}
	evalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(evalRoot, "lifecycle", "ideas", "2026", "q3", "release-x.md")
	if resolved != want {
		t.Errorf("resolved: want %q, got %q", want, resolved)
	}
}

func TestSandboxResolve_TraversalRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"../outside.md",
		"lifecycle/ideas/../../../outside.md",
		"lifecycle/ideas/../../../etc/passwd",
	}
	for _, c := range cases {
		_, err := sandbox.Resolve(root, c)
		if err == nil {
			t.Errorf("Resolve(%q): want traversal error, got nil", c)
		}
	}
}

func TestSandboxResolve_AbsolutePathRejected(t *testing.T) {
	root := t.TempDir()
	_, err := sandbox.Resolve(root, "/etc/passwd")
	if err == nil {
		t.Error("Resolve with absolute path: want error, got nil")
	}
}

func TestSandboxResolve_ExistingNestedSymlinkEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "lifecycle", "ideas", "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks not supported in this environment: %v", err)
	}

	_, err := sandbox.Resolve(root, "lifecycle/ideas/escape/secret.md")
	if err == nil {
		t.Error("Resolve through a symlink escaping the root: want error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "traversal") {
		t.Errorf("expected a traversal error, got: %v", err)
	}
}
